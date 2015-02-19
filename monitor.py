import argparse
import os
import docker
import thread
import json
import subprocess
import logging
import sys
from docker.errors import APIError
import requests
import requests.exceptions
import time

logger = logging.getLogger("weave-daemon")
docker_client = docker.Client(version="1.14")
TUTUM_HOST = os.getenv("TUTUM_HOST", "https://dashboard.tutum.co")
POLLING_INTERVAL = max(os.getenv("POLLING_INTERVAL", 30), 5)
TUTUM_AUTH = os.getenv("TUTUM_AUTH")


def join_weave(container_id):
    try:
        inspect = docker_client.inspect_container(container_id)
        cidr = None
        if inspect:
            env_vars = inspect.get("Config", {}).get("Env", [])
            for env_var in env_vars:
                if env_var.startswith("TUTUM_IP_ADDRESS="):
                    cidr = env_var[len("TUTUM_IP_ADDRESS="):]
                    break
        if cidr:
            tries = 0
            while tries < 3:
                logger.info("%s: adding to weave with IP %s" % (container_id, cidr))
                cmd = "/weave attach %s %s" % (cidr, container_id)
                p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, close_fds=True)
                if p.wait():
                    logger.error("%s: %s" % (container_id, p.stderr.read()))
                    tries += 1
                    time.sleep(1)
                else:
                    break
        else:
            logger.warning("%s: cannot find the IP address to add to weave" % container_id)
    except APIError:
        logger.exception("%s: exception when inspecting the container")


def container_attach_thread():
    # Attach existing containers
    containers = docker_client.containers(quiet=True)
    for container in containers:
        if container:
            thread.start_new_thread(join_weave, (container.get('Id'),))

    # Listen for events and attach new containers
    output = docker_client.events()
    for line in output:
        try:
            event = json.loads(line)
            logger.debug("Processing event: %s", event)
            status = event.get("status", "")
            if status == "start":
                thread.start_new_thread(join_weave, (event.get("id"),))
        except Exception as e:
            logger.exception(e)


def discover_peers_thread():
    peer_cache = []

    while True:
        try:
            r = requests.get("%s/api/v1/node/?state=Deployed" % TUTUM_HOST,
                             headers={"Authorization": TUTUM_AUTH})
            r.raise_for_status()
            nodes = r.json()["objects"]
            for node in nodes:
                if node["uuid"] not in peer_cache:
                    logger.info("%s: connecting to newly discovered peer: %s" %
                                (node["external_fqdn"], node["public_ip"]))
                    cmd = "/weave connect %s" % node["public_ip"]
                    p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, close_fds=True)
                    if p.wait():
                        logger.error("%s: %s" % (node["external_fqdn"], p.stderr.read()))
                    peer_cache.append(node["uuid"])
        except:
            logger.exception("Exception on peer discovery thread")
        time.sleep(POLLING_INTERVAL)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--debug', action="store_true")
    args = parser.parse_args()
    logging.basicConfig(stream=sys.stdout, format='%(asctime)s | %(levelname)s | %(message)s')
    logging.getLogger("weave-daemon").setLevel(logging.DEBUG if args.debug else logging.INFO)

    if TUTUM_AUTH:
        logger.info("Detected Tutum API access - starting peer discovery thread")
        thread.start_new_thread(discover_peers_thread, ())
    container_attach_thread()