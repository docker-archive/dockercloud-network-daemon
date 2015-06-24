#!/bin/bash
set -e

EXTERNAL_DOCKER=no
MOUNTED_DOCKER_FOLDER=no
if [ -S /var/run/docker.sock ]; then
	echo "=> Detected unix socket at /var/run/docker.sock"
	docker version > /dev/null 2>&1 || (echo "   Failed to connect to docker daemon at /var/run/docker.sock" && exit 1)
	EXTERNAL_DOCKER=yes
else
	if [ "$(ls -A /var/lib/docker)" ]; then
		echo "=> Detected pre-existing /var/lib/docker folder"
		MOUNTED_DOCKER_FOLDER=yes
	fi
	echo "=> Starting docker"
	wrapdocker > /dev/null 2>&1 &
	sleep 2
	echo "=> Checking docker daemon"
	docker version > /dev/null 2>&1 || (echo "   Failed to start docker (did you use --privileged when running this container?)" && exit 1)
fi

IP1=10.7.0.1/16
IP2=10.7.0.2/16

echo "=> Launching hello world containers"
docker run -d --name web-a -e HOSTNAME="web-a" -e TUTUM_IP_ADDRESS=$IP1 tutum/hello-world
docker run -d --name web-b -e HOSTNAME="web-b" -e TUTUM_IP_ADDRESS=$IP2 tutum/hello-world

echo "=> Launching weave-daemon"
docker run -d \
      --net host \
      --privileged \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /usr/bin/docker:/usr/local/bin/docker:ro \
      -v /proc:/hostproc \
      -e PROCFS=/hostproc \
      -e WEAVE_LAUNCH="" \
      -e WEAVE_PASSWORD="pass" \
      --name weave-daemon \
      tutum/weave-daemon:staging

echo "=> Waiting for logs"
sleep 60
touch weave-logs
docker logs weave-daemon >> weave-logs

cat weave-logs | grep "adding to weave with IP 10.7.0.2/16"
cat weave-logs | grep "adding to weave with IP 10.7.0.1/16"
