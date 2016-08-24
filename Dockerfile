FROM alpine:edge

ENV VERSION=1.6.1 \
    DOCKERCLOUD_PEER_COUNT=1 \
    WEAVE_DOCKER_ARGS="-e LOGSPOUT=ignore" \
    WEAVEMESH_NETWORK=dockercloud
RUN apk add --update ethtool conntrack-tools curl iptables iproute2 util-linux bind-tools
ADD https://github.com/weaveworks/weave/releases/download/v$VERSION/weave /weave
ADD dockercloud-network-daemon /
RUN chmod +x dockercloud-network-daemon weave
ADD run.sh /

ENTRYPOINT ["/run.sh"]
