#!/bin/sh

echo '=> Restart the "restart failed" containers'
for exited_container in $(docker ps -f status=exited -q --no-trunc); do
    restart_policy=$(docker inspect --format="{{ .HostConfig.RestartPolicy.Name }}" ${exited_container})
    if [ "${restart_policy}" = "always" ]; then
        err_msg=$(docker inspect --format="{{ .State.Error }}" ${exited_container})
        if [ "${err_msg}" = "failed to add endpoint: plugin not found" ]; then
            echo "Restarting ${exited_container}"
            docker start ${exited_container}
        fi
    fi
done