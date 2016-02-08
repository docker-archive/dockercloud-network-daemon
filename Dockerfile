FROM alpine:edge

ENV VERSION 1.4.1
ENV DOCKERCLOUD_PEER_COUNT 1

RUN ["apk", "add", "--update", "ethtool", "conntrack-tools", "curl", "iptables", "iproute2", "util-linux", "bind-tools"]
ADD https://github.com/weaveworks/weave/releases/download/v$VERSION/weave /weave
ADD dockercloud-network-daemon /
RUN chmod +x dockercloud-network-daemon weave
ENV WEAVE_DOCKER_ARGS="-e LOGSPOUT=ignore"
ADD run.sh /
ENTRYPOINT ["/run.sh"]
