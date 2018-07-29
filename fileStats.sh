#!/bin/bash

# Prints list of each directory under the specified path, and for each
# dir, a list of child files, ordered by last-mode time.  An optional
# comma-separated list of dirs to be not scanned can be specified.
# eg $ visitDirs /media/syncdisk 
usage="$0 path [-exclude excludePathList]"

defaultExcludePathList=".fseventsd|.stfolder|.stversions|.Spotlight-V100"
excludePathList="$defaultExcludePathList"

if [[ $2 == "-exclude" ]] ; then
    excludePathList=${3}
fi

#stat --format="%Y,%s,%y" monitor.sh


function visitDirs {
  # arg1 is the name of the dir being visited
  thisDir=$1
  thisBaseDir=$(basename "$thisDir")
  # the parameter is assumed to be checked against exclude list
  echo "$thisDir"

  #echo "thisDir=$thisDir"
  #echo "thisBaseDir=$thisBaseDir"

  find "$thisDir" -maxdepth 1 -type d -print | while read childDir ; do
    childBaseDir=$(basename "$childDir")
    #echo "childDir=$childDir"
    #echo "childBaseDir=$childBaseDir"

    if [[ ! "$childBaseDir" == "$thisBaseDir" ]] ; then
      if [[ ! "$childBaseDir" == +($excludePathList) ]] ; then
        # this dir is not in the exclude list
        #echo "====> VISITING with $childDir"
        visitDirs "$childDir"
        #echo "<==== VISITING from $childDir"
      #else
        #echo "===== EXCLUDING $childDir"
      fi
    fi
  done
}

visitDirs $1


