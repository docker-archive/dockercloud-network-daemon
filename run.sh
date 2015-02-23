#!/bin/bash
set -e

echo "=> Using docker binary:"
docker version

WEAVE_IMAGES=$(docker images | grep zettio/weave | wc -l)
if [ ${WEAVE_IMAGES} -eq "0" ]; then
    echo "=> Setting up weave images"
    /weave setup
fi

if [ "${WEAVE_LAUNCH}" = "**None**" ]; then
    echo "WEAVE_LAUNCH is **None**. Not running 'weave launch'"
else
    echo "=> Stopping any existing weave routers"
    OLD_ROUTER_ID=$(docker ps --no-trunc | grep "zettio/weave" | awk '{print $1}')
    if [ ! -z "${OLD_ROUTER_ID}" ]; then
        docker stop ${OLD_ROUTER_ID}
        sleep 5
        docker rm ${OLD_ROUTER_ID} || true
    fi
    echo "=> Running: weave launch \"${WEAVE_LAUNCH}\""
    if [ -z "${WEAVE_PASSWORD}" ]; then
        echo "!! WARNING: No \$WEAVE_PASSWORD set!"
        /weave launch -password ${WEAVE_PASSWORD} ${WEAVE_LAUNCH} || true
    else
        /weave launch ${WEAVE_LAUNCH} || true
    fi
    sleep 2
fi

echo "=> Current weave router status"
/weave status

echo "=> Starting peer discovery daemon"
exec python -u /app/monitor.py $@
