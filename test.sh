#!/bin/bash

function Onstarting {
reportSleeper &
reportPid="$!"
}

function Adhoc {
killProcess ${reportPid:-noJob}
reportSleeper "ad-hoc" &
reportPid="$!"
}

function StopReport {
killProcess ${reportPid:-noJob}
}




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





function gpioButton {
    # Waits for a button click, or a timeout.
    # Takes a single optional arg which is the number of seconds to wait.
    # Specifying 0, (or nothing) means wait forever for the button press.
    # Returns 0 if the button was pressed, or 1 if timed out waiting.
    # For debugging use: ps -elfT | grep -e gpio -e sleep
    waitTimeSecs=${1:-0}
    if (( $waitTimeSecs == 0)) ; then
        waitTimeSecs=10
    fi

    # pgio27 is the button input
    gpio -g mode 27 in
    # tie the input up
    gpio -g mode 27 up

    # wait indefinitely (in background) for button press (falling edge)
    gpio -g wfi 27 falling &
    buttonPid=$!

    (
        sleep $waitTimeSecs
        killProcess ${buttonPid}
    ) &
    timerPid=$!

    wait $buttonPid
    # status code 0 (true) indicates that gpio process finished.
    # other status codes (false) indicates that sleep process
    # killed the button process.
    waitStatus=$?

    killProcess ${timerPid}
    # # Maybe gpio needs more delay between calls?
    # sleep 1
    return $waitStatus
}
function killProcess {
    # Single arg which is the pid of job to be killed.
    # Undefined, empty, or noJob arg means nothing will be killed.
    #echo "monitor: killing ${1}"
    if [[ ${1:-noJob} != noJob ]] ; then
        kill ${1}
    fi
}

