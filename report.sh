#!/bin/bash

# usage: "$0 [-email]"
# Generate a report and optionally email it.
# Else just print the report to stdout.

# ----------- file list ----------------
excludePathList=".fseventsd|.stfolder|.stversions|.Spotlight-V100"

# --NOT USED--
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

# --NOT USED--
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

            # echo '------------- stats folder -------------'
            # curl -H "X-API-Key: $key"   \
            #     'http://127.0.0.1:8384/rest/stats/folder' 2>/dev/null |   \
            #     jq ". | {\"${folderId}\": .\"${folderId}\"}"
        fi
    done

    echo '------------- system error -------------'
    curl -H "X-API-Key: $key"   \
        'http://127.0.0.1:8384/rest/system/error' 2>/dev/null |   \
        jq '.'
}

# ----------- report generation ----------------
function generateReport {
    # $1 - name of file holding previous listFile

    # Print a report covering the file-list difference from the
    # last generation, and a series of other syncthing reports.

    # listFile="${1}"
    # listFileNew="${listFile}.tmp"

    # syncDir="/media/syncdisk"
    # listFiles "$syncDir" >"$listFileNew"
    # if test -f "$listFile" ; then
        # diff --side-by-side "$listFile" "$listFileNew"
    # else
        # cat "$listFileNew"
    # fi

    # New stats become existing for subsequent diff/run
    # cp "$listFileNew" "$listFile"
    # rm "$listFileNew" 2>/dev/null

    # echo '------------- ls -l  -------------'
    # ls -l

    echo '------------- AcerPC-Sync-Status-Report -------------'
    acerReport="/media/syncdisk/backups/Documents/AcerPC-Sync-Status-Report.txt"
    if test -f "$acerReport" ; then
        cat "$acerReport"
    else
        echo "NOT FOUND: $acerReport"
    fi

    syncthingReport

    echo '------------- df -H -------------'
    df -H

    # Concat the last bit of the log file
    # echo '------------- tail -500 monitor.log -------------'
    # tail -500 monitor.log
}

# http://www.raspberry-projects.com/pi/software_utilities/email/ssmtp-to-send-emails
# this ref is about setting up without requiring a password
# https://blog.dantup.com/2016/04/setting-up-raspberry-pi-raspbian-jessie-to-send-email/

generateReport "${1}"

