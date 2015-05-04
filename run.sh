#!/bin/sh
set -e

echo "=> Using weave version: $VERSION"

echo "=> Using docker binary:"
docker version

WEAVE_IMAGES=$(docker images | grep weaveworks/weave | wc -l)
if [ ${WEAVE_IMAGES} -eq "0" ]; then
    echo "=> Setting up weave images"
    /weave --local setup
fi

if [ "${WEAVE_LAUNCH}" = "**None**" ]; then
    echo "WEAVE_LAUNCH is **None**. Not running 'weave launch'"
else
    echo "=> Resetting weave on the node"
    /weave --local reset
    if [ ! -z "${WEAVE_PASSWORD}" ]; then
        echo "=> Running: weave launch -password XXXXXX ${WEAVE_LAUNCH}"
        /weave --local launch -password ${WEAVE_PASSWORD} ${WEAVE_LAUNCH} || true
    else
        echo "!! WARNING: No \$WEAVE_PASSWORD set!"
        echo "=> Running: weave launch ${WEAVE_LAUNCH}"
        /weave --local launch ${WEAVE_LAUNCH} || true
    fi
    sleep 2
fi

echo "=> Current weave router status"
/weave --local status

docker logs -f weave &

echo "=> Starting peer discovery daemon"
exec go run /go/src/github.com/tutumcloud/weave-daemon/main.go $@
