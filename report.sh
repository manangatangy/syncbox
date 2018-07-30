#!/bin/bash

# usage: "$0 [-email target-email-address]"
# Generate a report and optionally send to the specified email.
# or just print the report to stdout.

# ----------- file list ----------------
excludePathList=".fseventsd|.stfolder|.stversions|.Spotlight-V100"

function visitDirs {
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

function listFiles {
    # Starting at the specified path, print the path and then list each
    # file in the path.  Then visit child subdirs (except those which 
    # match the comma-separated list of dirs to be not scanned).
    echo "===== file stats at $(date) ====="
    visitDirs $1
}

# ----------- syncthing report ----------------
function syncthingReport {

    key="GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"

    echo '------------- system status -------------'
    curl -H "X-API-Key: $key"   \
        'http://127.0.0.1:8384/rest/system/status' 2>/dev/null |   \
        jq '. | {cpuPercent: .cpuPercent, startTime: .startTime }'

    curl -H "X-API-Key: $key" \
        'http://127.0.0.1:8384/rest/system/config' 2>/dev/null | \
        jq --compact-output \
        '.folders[] | {id: .id, label: .label, path: .path}' | \
        tr ':' ',' | tr -d '{"}' | while read inputLine
    do
        folderId=$(echo "$inputLine" | awk -F , '{print $2}')
        folderLabel=$(echo "$inputLine" | awk -F , '{print $4}')
        folderPath=$(echo "$inputLine" | awk -F , '{print $6}')

        if [[ "$folderId" != "default" ]] ; then
            echo '------------- system config -------------'
            echo "folderId=$folderId"
            echo "folderLabel=$folderLabel"
            echo "folderPath=$folderPath"

            echo "------------- db status \"${folderLabel}\" folder -------------"
            curl -H "X-API-Key: $key" \
                http://127.0.0.1:8384/rest/db/status?folder=$folderId 2>/dev/null | \
                jq '.'

            echo '------------- stats folder -------------'
            curl -H "X-API-Key: $key"   \
                'http://127.0.0.1:8384/rest/stats/folder' 2>/dev/null |   \
                jq ". | {\"${folderId}\": .\"${folderId}\"}"
        fi
    done

    echo '------------- system error -------------'
    curl -H "X-API-Key: $key"   \
        'http://127.0.0.1:8384/rest/system/error' 2>/dev/null |   \
        jq '.'
    echo '-------------  -------------'
}

# ----------- report generation ----------------
function generateReport {

    # Print a report covering the file-list difference from the
    # last generation, and a series of other syncthing reports.

    fileListOld="report-files.txt"
    fileListNew="report-files.new"

    syncDir="/media/syncdisk"

    listFiles "$syncDir" >"$fileListNew"
    if test -f "$fileListOld" ; then
        diff --side-by-side "$fileListOld" "$fileListNew"
    else
        cat "$fileListNew"
    fi

    # New stats become existing for subsequent diff/run
    cp "$fileListNew" "$fileListOld"
    rm "$fileListNew"

    syncthingReport
}

# ----------- entry ----------------
if [[ $1 == "-email" ]] ; then
    reportTarget="$2"
    reportSubject="Syncbox report"
    reportFile="mailStatus.txt"
    
    generateReport | mail -s "$reportSubject" "$reportTarget" 
else
    generateReport
fi


# http://www.raspberry-projects.com/pi/software_utilities/email/ssmtp-to-send-emails
# this ref is about setting up without requiring a password
# https://blog.dantup.com/2016/04/setting-up-raspberry-pi-raspbian-jessie-to-send-email/

