docker/dockercloud-network-daemon
==================

System container used by [Dockercloud](https://cloud.docker.com/) to provide a secure overlay network between nodes using [Weave](http://weave.works/net/). System containers are launched, configured and managed automatically on every node.

Performs two main tasks: attach containers to the weave network when they are started, and connect to newly discovered peers (via Tutum's API).


## Usage

    docker run -d \
      --net host \
      --privileged \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /usr/bin/docker:/usr/local/bin/docker \
      -v /proc:/hostproc \
      -e PROCFS=/hostproc \
      -e WEAVE_LAUNCH="" \
      -e WEAVE_PASSWORD="pass" \
      dockercloud/network-daemon

## Launch containers

    docker run -d --net dockercloud --ip 10.7.x.x dockercloud/hello-world

## Arguments

Key | Description
----|------------
WEAVE_LAUNCH | Argument for `weave launch` command, possible values: `""`, when you launch the first weave router; `"<ip/hostname>"`, when you want weave to join other's network; `"**None**"`, to not run `weave launch`
WEAVE_PASSWORD | Shared password used to secure the weave network
