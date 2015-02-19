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
    echo "=> Running: weave launch \"${WEAVE_LAUNCH}\""
    /weave launch ${WEAVE_LAUNCH} || true
    sleep 2
fi

echo "=> Current weave router status"
/weave status

echo "=> Starting peer discovery daemon"
exec python -u /app/monitor.py $@
