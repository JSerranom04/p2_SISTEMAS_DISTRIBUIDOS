#!/bin/sh

for i in $(seq 13 16)
do
    ssh -n "192.168.3.$i" "rm -r /tmp/go-build*"
done
