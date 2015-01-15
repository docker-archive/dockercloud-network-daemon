#!/bin/bash

if [ "${WEAVE_LAUNCH}" = "**None**" ]; then
    echo "WEAVE_LAUNCH is **None**. Do not run weave launch"
else
    if docker ps | awk '{print $2}' | grep -q -F 'zettio/weave'; then
        echo "weave router has been launched already"
    else
        echo "runing weave launch ${WEAVE_LAUNCH}"
        /weave launch ${WEAVE_LAUNCH}
    fi
fi

echo "start monitoring docker event"

exec python monitor.py
