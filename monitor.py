import docker
import thread
import json
import subprocess

def join_weave(container_id):
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
            print "%s:adding to weave with IP %s" % (container_id, cidr)
            cmd = "/weave attach %s %s" % (cidr, container_id)
            p = subprocess.Popen(args=cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, close_fds=True)
            if p.wait():
                print "%s:%s" % (container_id, p.stderr.read()),
            else:
                print "%s:%s" % (container_id, p.stdout.read()),
        else:
            print "%s:cannot find the IP address to add to weave" % container_id
    except Exception as e:
        print "%s:%s" % (container_id, e)


if __name__ == "__main__":

    docker_client = docker.Client()
    output = docker_client.events()
    for line in output:
        try:
            event = json.loads(line)
            status = event.get("status", "")
            if status == "start":
                container_id = event.get("id", "")
                print "%s:%s" % (container_id, status)
                thread.start_new_thread(join_weave, (container_id,))
        except Exception as e:
            print e
