#!/bin/bash

set -xe

function start_pushprox_client()
{
    # start push proxy client
    proxy_url=$PROXY_URL
    fqdn=$PROXY_FQDN
    proxy_log_dir=${PROXY_LOG_DIR:="./"}

    proxy_log_file="${proxy_log_dir%%/}/pushprox-client.log"

    if [ -z "${proxy_url}" ]; then
        echo "proxy_url not set"
        exit -1
    fi
    if [ -z "${fqdn}" ]; then
        echo "fqdn not set"
        exit -1
    fi

    ./pushprox-client --proxy-url=${proxy_url} --fqdn=${fqdn} > ${proxy_log_file} 2>&1 &
}

if [ "$PUSH_PROXY_ENABLED" == "1" ]; then
    start_pushprox_client
fi

# start geth
./geth $@