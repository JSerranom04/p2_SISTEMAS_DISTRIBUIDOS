#!/bin/sh

oldIFS="$IFS"
IFS=":"

contentRFile="data/content/contentRFile.txt"
sharedRWFilePrefix="data/content/sharedRWFile"
endpointsFile="data/endpoints/endpoints1.txt"

readerArgs="$endpointsFile $sharedRWFilePrefix"
writerArgs="$readerArgs $contentRFile"

readerPath="cmd/lector/lector.go"
writerPath="cmd/escritor/escritor.go"

args="$writerArgs"
scriptPath="cmd/escritor/escritor.go"
id=1

while read -r ip _
do
    # SSH into remote machines & run lector/escritor
    cwd="$(pwd)"
    ssh -n "$ip" "cd $cwd && go run $scriptPath $id $args" &
    if [ "$scriptPath" = "$writerPath" ]; then
        scriptPath="$readerPath"
        args="$readerArgs"
    else
        scriptPath="$writerPath"
        args="$writerArgs"
    fi
    id="$((id + 1))"
done < "$endpointsFile"

IFS="$oldIFS"