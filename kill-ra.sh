#!/bin/sh

oldIFS="$IFS"
IFS=":"

endpointsFile="data/endpoints/endpoints1.txt"

while read -r ip port
do
    # SSH into remote machines & kill lector/escritor
    ssh -n "$ip" "killall -9 escritor; killall -9 lector"
done < "$endpointsFile"

rm data/content/sharedRWFile*
rm logs/logs*

IFS="$oldIFS"
