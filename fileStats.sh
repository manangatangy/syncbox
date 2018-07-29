#!/bin/bash

# Prints list of each directory under the specified path, and for each
# dir, a list of child files (using default ls ordering).  An optional
# comma-separated list of dirs to be not scanned can be specified.
# eg $ visitDirs /media/syncdisk 
usage="$0 path [-exclude excludePathList]"

defaultExcludePathList=".fseventsd|.stfolder|.stversions|.Spotlight-V100"
excludePathList="$defaultExcludePathList"

if [[ $2 == "-exclude" ]] ; then
    excludePathList=${3}
fi

function visitDirs {
  # 
  # arg1 is the name of the dir being visited
  thisDir=$1
  thisBaseDir=$(basename "$thisDir")
  # the parameter is already assumed to be checked against exclude list
  #ls -l "$thisDir"

  # print the directory name, trailed with / to make clear it's dir
  echo "${thisDir}/"

  # list the children of directory, exclude dirs, exclude total
  ls -og --file-type "$thisDir" | \
    grep -v ^d.*/ | \
    grep -v ^total.* | \
    cut -c 13-

  # blank line for separator
  echo 

  # now visit all child dirs except those in exclude list
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

echo "===== file stats at $(date) ====="
visitDirs $1
