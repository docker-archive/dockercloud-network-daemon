FROM alpine

ENV VERSION 1.0.1

RUN ["apk", "add", "--update", "ethtool", "conntrack-tools", "curl", "iptables", "iproute2", "util-linux", "arping"]
ADD weave-daemon /
RUN chmod +x weave-daemon
ENV WEAVE_DOCKER_ARGS="-e LOGSPOUT=ignore"
ADD weave /
ADD run.sh /
ENTRYPOINT ["/run.sh"]
