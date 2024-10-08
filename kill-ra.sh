#!/bin/sh

oldIFS="${IFS}"
IFS=":"

endpointsFile="data/endpoints/endpoints0.txt"

while read -r ip _ #ignorar el puerto
do
    # SSH into remote machines & kill lector/escritor
    # killall -2: send SIGINT (will be captured)
    # killall -15: send SIGTERM (will be captured)
    # killall -9: send SIGKILL (non-capturable)
    ssh -n "${ip}" "killall -9 escritor; killall -9 lector"
done < "${endpointsFile}"

rm data/content/sharedRWFile*
rm logs/logs*

IFS="${oldIFS}"
