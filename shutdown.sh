#!/bin/bash

# shutdown.sh
#
# author godcheese [godcheese@outlook.com]
# date 2023-01-05

function get_running_pid() {
  echo "$(ps -ef | grep "$1" | grep -v grep | grep -v sh | awk '{print $2}' ORS=" ")"
}
pids=`get_running_pid "nginx-reload-from-nacos"`
if [ ! -z $pids ]; then
    kill $pids
    echo 'shutdown success.'
else
  echo 'not running.'
fi



