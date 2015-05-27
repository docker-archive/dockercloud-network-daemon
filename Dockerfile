FROM alpine

ENV VERSION 0.10.0
ENV WEAVE_DOCKER_ARGS="--restart=always"

RUN ["apk", "add", "--update", "ethtool", "conntrack-tools", "curl", "iptables", "iproute2", "util-linux"]
ADD weave-daemon /
RUN chmod +x weave-daemon
ADD weave /
ADD run.sh /
ENTRYPOINT ["/run.sh"]
