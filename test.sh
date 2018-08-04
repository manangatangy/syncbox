#!/bin/bash

# A report is to be generated/emailed x days after the last one
# (even if that was requested ad-hoc) or after the startup.

function killProcess {
    # Single arg which is the pid of job to be killed.
    # Undefined, empty, or noJob arg means nothing will be killed.
    #echo "monitor: killing ${1}"
    if [[ ${1:-noJob} != noJob ]] ; then
        kill ${1}
    fi
}

function killReport {
    # Retrieve the pid from the file
    while ! test -f "$reportPidFile" ; do
        sleep 1
    done
    reportPid=$(cat "$reportPidFile")
    echo "retrieved/killed reportPid= $reportPid"
    killProcess ${reportPid:-noJob}
}

reportTarget="david.x.weiss@gmail.com"
reportPeriod="+30 seconds"
reportFile="report.txt"
reportPid="noJob"
reportPidFile="report.pid"
nextReportDateSecs="the date in secs when the next report is due"

function scheduleInitReport {
    # Use the timestamp of the reportFile to determine when the
    # next report is due (use nowtime if there is no reportFile).
    # Start a background process that sleeps until the report is
    # due, and then generates the report.
    if test -f "$reportFile" ; then
        lastReportDate=$(ls -l --time-style=full-iso "$reportFileName" | awk '{ print $6, $7, $8 }')
        echo "previous report was made $lastReportDate"
    else
        lastReportDate=$(date)
        echo "no previous report found"
    fi
    # Ensure no other report is pending
    killReport
    scheduleNextReport "$lastReportDate"
}

function scheduleNextReport {
    # Schedules the next report for reportPeriod after startDate.
    startDate="$1"
    nextReportDate=$(date -d "$startDate $reportPeriod")
    echo "next report scheduled for $nextReportDate"

    nextReportDateSecs=$(date -d "$nextReportDate" "+%s")
    nowSecs=$(date "+%s")
    secs="$(( $nextReportDateSecs - $nowSecs ))"
    echo "next report due in $secs"

    rm "$reportPidFile"
    (
        sleep "$secs" &
        reportPid="$!"
        # Write the pid to a file, so the sibling may kill.
        echo "$reportPid" >"$reportPidFile"
        echo "started $reportPid for $secs secs"
        if wait $reportPid ; then
            echo "$reportPid finished OK"
            doReport "scheduled" "$secs"        
        else
            echo "$reportPid finished NBG"
        fi
    ) &
    disown
}

function doReport {
    # Emails report and schedules next one. 
    # Can be called ad-hoc or as a scheduled report.
    echo "emailing $1 report"
    reportSubject="Syncbox $1 report"
sleep 5    
    ##./report.sh "$reportFile" | mail -s "$reportSubject" "$reportTarget" 
    # Could use date of reportFile, but now is easier.
    scheduleNextReport "$(date)"
}

function adHocReport {
    # Ensure no other report is pending
    killReport
    doReport "ad-hoc"
}

# reportFileName="xxx"
# lastReportDate=$(ls -l --time-style=full-iso "$reportFileName" | awk '{ print $6, $7, $8 }')
# echo "lastReportDate=$lastReportDate"
# nextReportDate=$(date -d "$lastReportDate +1 day")
# echo "nextReportDate=$nextReportDate"

# nextReportSecs=$(date -d "$nextReportDate" "+%s")
# echo "nextReportSecs=$nextReportSecs"

# nowSecs=$(date "+%s")
# secsUntilReportDue=$(( $nextReportSecs - $nowSecs ))
# echo "secsUntilReportDue=$secsUntilReportDue"

# #date -d '@1533131552' "+%F %T %Z"


# lastReportDate=2018-08-02 00:16:56.036975022 +1000
# pi@syncbox:~/syncbox $ nextReportDate=$(date -d "$lastReportDate +1 day")
# pi@syncbox:~/syncbox $ echo "nextReportDate=$nextReportDate"
# nextReportDate=Fri Aug  3 00:16:56 AEST 2018
# pi@syncbox:~/syncbox $ nextReportSecs=$(date -d "$nextReportDate" "+%s")
# pi@syncbox:~/syncbox $ echo "nextReportSecs=$nextReportSecs"
# nextReportSecs=1533219416
# pi@syncbox:~/syncbox $ nowSecs=$(date "+%s")
# pi@syncbox:~/syncbox $ secsUntilReportDue=$(( $nextReportSecs - $nowSecs ))
# pi@syncbox:~/syncbox $ echo "secsUntilReportDue=$secsUntilReportDue"
# secsUntilReportDue=86034
