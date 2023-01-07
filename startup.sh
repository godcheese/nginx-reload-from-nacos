#!/bin/bash

# startup.sh
#
# author godcheese [godcheese@outlook.com]
# date 2023-01-05

nohup ./nginx-reload-from-nacos > ./run.log 2>&1 &
echo 'startup success.'