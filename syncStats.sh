#!/bin/bash

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
    echo '------------- system config -------------'
    folderId=$(echo "$inputLine" | awk -F , '{print $2}')
    folderLabel=$(echo "$inputLine" | awk -F , '{print $4}')
    folderPath=$(echo "$inputLine" | awk -F , '{print $6}')

    echo "folderId=$folderId"
    echo "folderLabel=$folderLabel"
    echo "folderPath=$folderPath"

    echo "------------- db status \"${folderLabel}\" folder -------------"
    curl -H "X-API-Key: $key" \
      http://127.0.0.1:8384/rest/db/status?folder=$folderId 2>/dev/null \
      | jq '.'

    echo '------------- stats folder -------------'
    curl -H "X-API-Key: $key"   \
      'http://127.0.0.1:8384/rest/stats/folder' 2>/dev/null |   \
      jq ". | {\"${folderId}\": .\"${folderId}\"}"

done

echo '------------- system error -------------'
curl -H "X-API-Key: $key"   \
  'http://127.0.0.1:8384/rest/system/error' 2>/dev/null |   \
  jq '.'

echo '-------------  -------------'
