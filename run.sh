#!/bin/sh
set -e

echo "=> Using weave version: $VERSION"

echo "=> Using docker binary:"
docker version

if [ "${WEAVE_LAUNCH}" = "**None**" ]; then
    echo "WEAVE_LAUNCH is **None**. Not running 'weave launch'"
else
    ROUTER_PRESENT=`docker ps -a | grep -c "weave:${VERSION}" || true`
    if [ "${ROUTER_PRESENT}" = "0" ]; then
        echo "=> No weave router version ${VERSION} found"
        WEAVE_IMAGES=`docker images | grep -c "weaveworks/weave:${VERSION}" || true`
        if [ "${WEAVE_IMAGES}" = "0" ]; then
            echo "=> Setting up weave images"
            /weave --local setup
        fi

        echo "=> Resetting weave on the node"
        /weave --local reset
    else
        echo "=> Weave router version ${VERSION} found"
    fi

    PRIVATE_SUBNETS=$(ip addr show | grep "eth[0-9]" | grep -oE "(10.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}/[0-9]+|172.(16|17|18|19|2[0-9]|30|31).[0-9]{1,3}.[0-9]{1,3}/[0-9]+|192.168.[0-9]{1,3}.[
0-9]{1,3}/[0-9]+)" | tr '\n' ',' | head -c -1)
    if [ ! -z "${DOCKERCLOUD_PRIVATE_CIDR}" ]; then
        # TODO: check which private subnets detected can be trusted
        # TRUSTED_SUBNETS="${PRIVATE_SUBNETS},${DOCKERCLOUD_PRIVATE_CIDR}"
        TRUSTED_SUBNETS="${DOCKERCLOUD_PRIVATE_CIDR}"
    fi
    echo "=> Marking the following private subnets as trusted (unencrypted): ${TRUSTED_SUBNETS:-none}"

    if [ ! -z "${WEAVE_PASSWORD}" ]; then
        echo "=> Running: weave launch -password XXXXXX ${WEAVE_LAUNCH}"
        echo "=> Peer count: ${DOCKERCLOUD_PEER_COUNT}"
        WEAVE_EXTRA_ARGS="--password=${WEAVE_PASSWORD}"
    else
        echo "!! WARNING: No \$WEAVE_PASSWORD set!"
        echo "=> Running: weave launch ${WEAVE_LAUNCH}"
        echo "=> Peer count: ${DOCKERCLOUD_PEER_COUNT}"
    fi
    /weave --local launch-router --connlimit=0 --ipalloc-range=10.128.0.0/10 --trusted-subnets=${TRUSTED_SUBNETS} --no-dns --no-discovery --init-peer-count=${DOCKERCLOUD_PEER_COUNT} ${WEAVE_EXTRA_ARGS} ${WEAVE_LAUNCH} || true
    sleep 2
fi

echo "=> Current weave router status"
/weave --local status

echo "=> Running weave expose"
/weave --local expose 10.7.255.254/16
docker ps | grep -q "weave:${VERSION}"

docker logs -f weave &

echo "=> Starting peer discovery daemon"
exec /network-daemon $@
