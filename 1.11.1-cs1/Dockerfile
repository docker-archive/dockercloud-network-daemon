FROM alpine:edge

ENV VERSION=1.5.2 \
    DOCKERCLOUD_PEER_COUNT=1 \
    WEAVE_DOCKER_ARGS="-e LOGSPOUT=ignore" \
    WEAVEMESH_NETWORK=dockercloud
RUN apk add --update ethtool conntrack-tools curl iptables iproute2 util-linux bind-tools
ADD https://github.com/weaveworks/weave/releases/download/v$VERSION/weave /weave
ADD https://s3.amazonaws.com/files.cloud.docker.com/packages/docker/docker-1.11.1-cs1.tgz /docker.tgz
ADD dockercloud-network-daemon /
RUN tar zxvf /docker.tgz && \
    cp /docker/* /usr/bin/ && \
    rm -rf /docker && \
    chmod +x dockercloud-network-daemon weave /usr/bin/docker
ADD run.sh /

ENTRYPOINT ["/run.sh"]
