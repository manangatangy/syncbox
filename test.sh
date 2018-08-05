#!/bin/bash

# On starting ...
reportSleeper &
reportPid="$!"

# on Ad-hoc
killProcess ${reportPid:-noJob}
reportSleeper "ad-hoc" &
reportPid="$!"







reportTarget="david.x.weiss@gmail.com"
reportFile="report.txt"
reportInterval="+10 seconds"

function reportSleeper {
    if [[ "${1}" == "ad-hoc" ]] ; then
        sendReport "ad-hoc"
        lastReportDate=$(date)
    else
        # Use the timestamp of the reportFile to determine when the
        # next report is due (use now time if there is no reportFile).
        if test -f "$reportFile" ; then
            lastReportDate=$(ls -l --time-style=full-iso "$reportFile" | awk '{ print $6, $7, $8 }')
            echo "$BASHPID: reportProcess, last report on: $lastReportDate"
        else
            lastReportDate=$(date)
            echo "$BASHPID: reportProcess, $reportFile not found"
        fi
    fi

    while : ; do
        # Use lastReportDate to determine how long to sleep for
        echo "$BASHPID: reportProcess, interval $reportInterval"
        nextReportDate=$(date -d "$lastReportDate $reportInterval")
        nextReportDateSecs=$(date -d "$nextReportDate" "+%s")
        echo "$BASHPID: reportProcess, next report on: $nextReportDate"
        nowSecs=$(date "+%s")
        secs="$(( $nextReportDateSecs - $nowSecs ))"
        echo "$BASHPID: reportProcess, sleeping for $secs secs"
        sleep "$secs"
        echo "$BASHPID: reportProcess, sending report..."
        sendReport "scheduled"
        lastReportDate=$(date)
    done

}

function sendReport {
    # Called with subject sub-field.
    echo "$BASHPID: emailing $1 report"
    reportSubject="Syncbox $1 report"
    sleep 5    
    ##./report.sh "$reportFile" | mail -s "$reportSubject" "$reportTarget" 
}
