import argparse
import docker
import thread
import json
import subprocess
import logging
import sys
from docker.errors import APIError

logger = logging.getLogger("weave-daemon")
docker_client = docker.Client(version="1.14")


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
            logger.info("%s: adding to weave with IP %s" % (container_id, cidr))
            cmd = "/weave attach %s %s" % (cidr, container_id)
            p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, close_fds=True)
            if not p.wait():
                logger.error("%s:%s" % (container_id, p.stderr.read()))
        else:
            logger.warning("%s: cannot find the IP address to add to weave" % container_id)
    except APIError:
        logger.exception("%s: exception when inspecting the container")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--debug', action="store_true")
    args = parser.parse_args()
    logging.basicConfig(stream=sys.stdout)
    logging.getLogger("weave-daemon").setLevel(logging.DEBUG if args.debug else logging.INFO)

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
