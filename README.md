tutum/weave-daemon
==================

```
    docker run -d \
      --net=host \
      --privileged \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /usr/bin/docker:/usr/local/bin/docker:r \
      -v /proc:/hostproc \
      -e PROCFS=/hostproc \
      -e WEAVE_LAUNCH="" \
      -e WEAVE_PASSWORD="pass" \
      tutum/weave-daemon
```

**Arguments**

```
    WEAVE_LAUNCH    argument for "weave launch" command, possible values:
                    "", when you launch the first weave router
                    "<ip/hostname>", when you want weave join other's network
                    "**None**", do not run "weave launch"
    WEAVE_PASSORD   password for weave network, empty by default
```
