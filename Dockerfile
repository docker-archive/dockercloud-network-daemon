FROM alpine:edge

ENV VERSION 1.0.3

RUN ["apk", "add", "--update", "ethtool", "conntrack-tools", "curl", "iptables", "iproute2", "util-linux", "bind-tools"]
ADD weave-daemon /
RUN chmod +x weave-daemon
ENV WEAVE_DOCKER_ARGS="-e LOGSPOUT=ignore"
ADD weave /
ADD run.sh /
ENTRYPOINT ["/run.sh"]
