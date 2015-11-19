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
        /weave --local reset || true
    else
        echo "=> Weave router version ${VERSION} found"
    fi

    if [ ! -z "${WEAVE_PASSWORD}" ]; then
        echo "=> Running: weave launch -password XXXXXX ${WEAVE_LAUNCH}"
        echo "=> Peer count: ${TUTUM_PEER_COUNT}"
        /weave --local launch-router --ipalloc-range=10.128.0.0/10 --no-dns --no-discovery --init-peer-count ${TUTUM_PEER_COUNT} --password ${WEAVE_PASSWORD} ${WEAVE_LAUNCH} || true
    else
        echo "!! WARNING: No \$WEAVE_PASSWORD set!"
        echo "=> Running: weave launch ${WEAVE_LAUNCH}"
        echo "=> Peer count: ${TUTUM_PEER_COUNT}"
        /weave --local launch-router --ipalloc-range=10.128.0.0/10 --no-dns --no-discovery --init-peer-count ${TUTUM_PEER_COUNT} ${WEAVE_LAUNCH} || true
    fi
    sleep 2
fi

echo "=> Current weave router status"
/weave --local status

echo "=> Running weave expose"
/weave --local expose 10.7.255.254/16
docker ps | grep -q "weave:${VERSION}"

docker logs -f weave &

echo "=> Starting peer discovery daemon"
exec /weave-daemon $@
