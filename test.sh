#!/bin/bash
set -e
set -m
CheckNetworkSetup ()
{
    LOOP_LIMIT=120
    for (( i=0 ; ; i++ )); do
        if [ ${i} -eq ${LOOP_LIMIT} ]; then
            echo "Time out. Current docker network on the node:"
            docker network ls
            exit 1
        fi
        echo "=> Waiting for docker network to be created, trying ${i}/${LOOP_LIMIT} ..."
        sleep 1
        docker network ls | grep -iF "dockercloud" > /dev/null 2>&1 && break
    done
}

EXTERNAL_DOCKER=no
MOUNTED_DOCKER_FOLDER=no
if [ -S /var/run/docker.sock ]; then
    echo "=> Detected unix socket at /var/run/docker.sock"
    docker version || (echo "   Failed to connect to docker daemon at /var/run/docker.sock" && exit 1)
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

DOCKER_BINARY=${DOCKER_BINARY:-"/usr/bin/docker"}

echo "=> Building the image"
docker build -t this .

echo "=> Launching network-daemon"
docker rm -f network-daemon >/dev/null 2>&1 || true
docker run -d \
      --net host \
      --privileged \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v ${DOCKER_BINARY}:/usr/bin/docker \
      -v /proc:/hostproc \
      -e PROCFS=/hostproc \
      -e WEAVE_LAUNCH="" \
      -e WEAVE_PASSWORD="pass" \
      -e NO_PEER_DISCOVERY="true" \
      --name network-daemon \
      this
docker logs -f network-daemon &

IP1=10.7.0.2
IP2=10.7.0.3
IP3=10.7.0.4

CheckNetworkSetup

echo "=> Launching hello world containers"
docker rm -f C1 C2 >/dev/null 2>&1 || true
docker run -d --name C1 --net dockercloud --ip $IP1 dockercloud/hello-world
docker run -d --name C2 --net dockercloud --ip $IP2 dockercloud/hello-world

echo "=> Pinging hello world containers"
docker rm -f pingC >/dev/null 2>&1 || true
docker run -d --name=pingC --net dockercloud --ip $IP3 dockercloud/hello-world
docker exec -t pingC ping $IP1 -c 15
docker exec -t pingC ping $IP2 -c 15

kill %
echo "=> Pass!"