#!/bin/bash

# Collect some status info and mail it to me.
# The info is 1) stats from syncthing, and 2) file info from the 
# syncdisk.  Regading the file info, the current fileStats are saved
# in a local-file, and it's the diff between that and the previous
# file stats, that is actually mailed

statsFileName="mailStatus.txt"

existingFileStatsFileName="fileStats.txt"
newFileStatsFileName="fileStats.new"

syncDir="/media/syncdisk"

./fileStats.sh "$syncDir" >"$newFileStatsFileName"
if test -f "$existingFileStatsFileName" ; then
  diff --side-by-side "$existingFileStatsFileName" "$newFileStatsFileName" \
    > "$statsFileName" 
else
  cp "$newFileStatsFileName" "$statsFileName"
fi
# New stats become existing for subsequent diff/run
cp "$newFileStatsFileName" "$existingFileStatsFileName"
rm "$newFileStatsFileName"

./syncStats.sh >> "$statsFileName"

mail -s "Syncbox status: $subject" \
  david.x.weiss@gmail.com <"$statsFileName" 

# http://www.raspberry-projects.com/pi/software_utilities/email/ssmtp-to-send-emails
# this ref is about setting up without requiring a password
# https://blog.dantup.com/2016/04/setting-up-raspberry-pi-raspbian-jessie-to-send-email/
