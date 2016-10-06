FROM alpine:edge

ARG DOCKER_VERSION=1.11.2-cs5
ENV VERSION=1.6.2 \
    DOCKERCLOUD_PEER_COUNT=1 \
    WEAVE_DOCKER_ARGS="-e LOGSPOUT=ignore" \
    WEAVEMESH_NETWORK=dockercloud
RUN apk add --update ethtool conntrack-tools curl iptables iproute2 util-linux bind-tools tar

# Download docker statically linked binary
ADD https://files.cloud.docker.com/packages/docker/docker-$DOCKER_VERSION.tgz /tmp/
RUN tar zxf /tmp/docker-$DOCKER_VERSION.tgz -C /usr/local/bin/ --strip-components 1

# Download weave
ADD https://github.com/weaveworks/weave/releases/download/v$VERSION/weave /weave

# Add network daemon binary and scripts
ADD dockercloud-network-daemon /
RUN chmod +x dockercloud-network-daemon weave
ADD *.sh /

ENTRYPOINT ["/run.sh"]
