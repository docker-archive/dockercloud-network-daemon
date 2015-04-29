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


logger = logging.getLogger("weave-daemon")
docker_client = docker.Client(version="auto")
TUTUM_NODE_FQDN = os.getenv("TUTUM_NODE_FQDN")
WEAVE_CMD = "/weave --local"

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
                cmd = "%s attach %s %s" % (WEAVE_CMD, cidr, container_id)
                p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, close_fds=True)
                if p.wait():
                    logger.error("%s: %s" % (container_id, p.stderr.read() or p.stdout.read()))
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
            if event.get("status") == "start" and not event.get("from").startswith("weaveworks/weave"):
                attach_container(event.get("id"))
        except Exception as e:
            logger.exception(e)


def discover_peers():
    global peer_cache
    tries = 0
    while True:
        try:
            nodes = tutum.Node.list(state="Deployed")
            for node in nodes:
                if node.external_fqdn == TUTUM_NODE_FQDN or node.public_ip in peer_cache:
                    continue
                connect_to_peer(node)
                peer_cache.append(node.public_ip)
            break
        except Exception as e:
            tries += 1
            if tries > 3:
                raise Exception("Unable to discover peers: %s" % str(e))
        time.sleep(1)


def connect_to_peer(node):
    tries = 0
    while True:
        logger.info("%s: connecting to newly discovered peer: %s" %
                    (node.external_fqdn, node.public_ip))
        cmd = "%s connect %s" % (WEAVE_CMD, node.public_ip)
        p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, close_fds=True)
        if p.wait():
            logger.error("%s: %s" % (node.external_fqdn, p.stderr.read() or p.stdout.read()))
            tries += 1
            if tries > 3:
                raise Exception("Unable to 'weave connect' to new peer: %s" % p.stderr.read() or p.stdout.read())
        else:
            break
        time.sleep(1)


def event_handler(event):
    try:
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

    if os.getenv("TUTUM_AUTH"):
        logger.info("Detected Tutum API access - starting peer discovery thread")
        events = tutum.TutumEvents()
        events.on_message(event_handler)
        thread.start_new_thread(events.run_forever, ())
        discover_peers()
    container_attach_thread()