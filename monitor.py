import argparse
import os
import thread
import json
import subprocess
import logging
import sys
import time

import docker
import tutum
from docker.errors import APIError
import requests
import requests.exceptions


logger = logging.getLogger("weave-daemon")
docker_client = docker.Client(version="auto")
TUTUM_HOST = os.getenv("TUTUM_HOST", "https://dashboard.tutum.co")
POLLING_INTERVAL = max(os.getenv("POLLING_INTERVAL", 30), 5)
TUTUM_AUTH = os.getenv("TUTUM_AUTH")
TUTUM_NODE_FQDN = os.getenv("TUTUM_NODE_FQDN")

peer_cache = []


def attach_container(container_id):
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
            attach_container(container.get('Id'))

    # Listen for events and attach new containers
    output = docker_client.events()
    for line in output:
        try:
            event = json.loads(line)
            logger.debug("Processing event: %s", event)
            if event.get("status") == "start" and not event.get("from").startswith("zettio/weave"):
                attach_container(event.get("id"))
        except Exception as e:
            logger.exception(e)


def discover_peers():
    global peer_cache
    tries = 0
    while True:
        try:
            r = requests.get("%s/api/v1/node/?state=Deployed&limit=100" % TUTUM_HOST,
                             headers={"Authorization": TUTUM_AUTH})
            r.raise_for_status()
            nodes = r.json()["objects"]
            for node in nodes:
                if node["external_fqdn"] == TUTUM_NODE_FQDN or node["public_ip"] in peer_cache:
                    continue
                connect_to_peer(node["public_ip"], node["external_fqdn"])
                peer_cache.append(node["public_ip"])
            break
        except Exception as e:
            tries += 1
            if tries > 3:
                raise Exception("Unable to discover peers: %s" % str(e))
        time.sleep(1)


def connect_to_peer(public_ip, external_fqdn="unknown"):
    tries = 0
    while True:
        logger.info("%s: connecting to newly discovered peer: %s" %
                    (external_fqdn, public_ip))
        cmd = "/weave connect %s" % public_ip
        p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, close_fds=True)
        if p.wait():
            logger.error("%s: %s" % (external_fqdn, p.stderr.read()))
            tries += 1
            if tries > 3:
                raise Exception("Unable to 'weave connect' to new peer: %s" % p.stderr.read())
        else:
            break
        time.sleep(1)


def on_tutum_message(msg):
    try:
        event = json.loads(msg)
        if event.get("type", "") == "node" and event.get("state", "") == "Deployed":
            discover_peers()
    except Exception as e:
        logger.exception("Failed to process tutum event message: %s" % str(e))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--debug', action="store_true")
    args = parser.parse_args()
    logging.basicConfig(stream=sys.stdout, format='%(asctime)s | %(levelname)s | %(message)s')
    logging.getLogger("weave-daemon").setLevel(logging.DEBUG if args.debug else logging.INFO)

    if TUTUM_AUTH:
        logger.info("Detected Tutum API access - starting peer discovery thread")
        events = tutum.TutumEvents()
        events.on_message(on_tutum_message)
        thread.start_new_thread(events.run_forever, ())
    container_attach_thread()