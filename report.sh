#!/bin/bash

# usage: "$0 [-email]"
# Generate a report and optionally email it.
# Else just print the report to stdout.

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
    # Print a report covering a series of syncthing reports.

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
}

# http://www.raspberry-projects.com/pi/software_utilities/email/ssmtp-to-send-emails
# this ref is about setting up without requiring a password
# https://blog.dantup.com/2016/04/setting-up-raspberry-pi-raspbian-jessie-to-send-email/

generateReport

