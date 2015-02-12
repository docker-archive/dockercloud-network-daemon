import docker
import thread
import json
import subprocess
import logging
import sys

logger = logging.getLogger("weave-daemon")

def join_weave(docker_client, container_id):
    try:
        inspect = docker_client.inspect_container(container_id)

        cidr = ""
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
                logger.info("%s:%s" % (container_id, p.stderr.read()))
        else:
            logger.info("%s: cannot find the IP address to add to weave" % container_id)
    except Exception as e:
        logger.exception("%s:%s" % (container_id, e))


def init_attach(docker_client):
    containers = docker_client.containers(quiet=True)
    for container in containers:
        container_id = container.get('Id', '')
        if container:
            thread.start_new_thread(join_weave, (docker_client, container_id))


if __name__ == "__main__":
    logging.basicConfig(stream=sys.stdout)
    logging.getLogger("weave-daemon").setLevel(logging.INFO)

    docker_client = docker.Client()

    logger.info("attaching existing running containers to weave network")
    init_attach(docker_client)

    logger.info("monitoring docker event")
    output = docker_client.events()
    for line in output:
        try:
            event = json.loads(line)
            status = event.get("status", "")
            if status == "start":
                container_id = event.get("id", "")
                image = event.get("from","")
                logger.info("%s: (from %s) %s" % (container_id, image, status))
                thread.start_new_thread(join_weave, (docker_client, container_id))
        except Exception as e:
            logger.exception(e)
