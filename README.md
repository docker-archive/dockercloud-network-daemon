tutum/weave-daemon
==================

System container used by [Tutum](http://www.tutum.co/) to provide a secure overlay network between nodes using [Weave](http://weave.works/net/). System containers are launched, configured and managed automatically on every node.

Performs two main tasks: attach containers to the weave network when they are started, and connect to newly discovered peers (via Tutum's API).


## Usage

    docker run -d \
      --net host \
      --privileged \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /usr/bin/docker:/usr/local/bin/docker:r \
      -v /proc:/hostproc \
      -e PROCFS=/hostproc \
      -e WEAVE_LAUNCH="" \
      -e WEAVE_PASSWORD="pass" \
      tutum/weave-daemon


## Arguments

Key | Description
----|------------
WEAVE_LAUNCH | Argument for `weave launch` command, possible values: `""`, when you launch the first weave router; `"<ip/hostname>"`, when you want weave to join other's network; `"**None**"`, to not run `weave launch`
WEAVE_PASSWORD | Shared password used to secure the weave network
