# RICART-AGRAWALA ALGORITHM FOR THE READERS AND WRITERS PROBLEM

## To generate shiViz log file

*GoVector --log_type shiviz --log_dir logs --outfile shiviz-log.log*
It will generate the *shiviz-log.log* file, which you can then plug
into *<https://bestchai.bitbucket.io/shiviz/>* to view the sequence
diagram for the communications between processes.
IMPORTANT: The `GoVector` binary is under $HOME/go/bin, so add it
($HOME/go/bin) to your PATH env var if it's not already added.

## To run the program with N writers and N readers
*./run-ra.sh*
IMPORTANT: the default endpoints file is data/endpoints/endpoints1.txt;
if desired to run the program with different endpoints, CHANGE the
"endpointsFile" variable in *./run-ra.sh* with the path of the new endpoints file.
NOTE: the variable "contentRFile" holds the path to a shared READ-ONLY file
that writers will read from to write on the shared file. Also, the variable
"sharedRWFilePrefix" holds the prefix path for the mutex file (each process
will have one exact copy of it).

## To KILL THE READER/WRITER PROCESSES (DON'T FORGET TO DO IT OR THEY WILL KEEP HOLDING THE PORT)

*./kill-ra.sh*
Again, if a different endpoints file has been used, replace the "endpointsFile"
variable in this script with the path of that new file.
