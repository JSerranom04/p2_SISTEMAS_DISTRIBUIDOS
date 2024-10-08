#!/bin/sh

endpointsFile="data/endpoints/endpoints1.txt"
contentRFile="data/content/contentRFile.txt"
sharedRWFilePrefix="data/content/sharedRWFile"
totalWriters=6
totalEndpoints=12

while read -r line
do
    for i in $(seq 1 $totalEndpoints)
    do
        test "$(grep -c '^'"$line"'[ ]*$' $sharedRWFilePrefix$i.txt)" -ne $totalWriters \
&& echo "❌ WRONG LINE COUNT, ERROR ❌" >&2 && exit 1
    done
done < "$contentRFile"

echo "✅ CORRECT!! ✅"
