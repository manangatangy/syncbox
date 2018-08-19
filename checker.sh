#!/bin/bash

logfile="no-logfile"
if [[ $1 == "-logfile" ]] ; then
    logfile=${2:-logfile}
    exec &>> $logfile
fi

function log {
    echo "$(date --iso-8601=minutes) $1"
}

# ----------- self checking ----------------
# After a day or so, the monitor.sh script main thread gets stuck
# in a system command; usually ps.  At this point the trace stops 
# and the cpu for the main process hits over 70% up to 90ish.  
# The led keeps flashing, and the report keeps generating, but the 
# LCD display loop is blocked.  The process table is not full and 
# there is only one or two zombies, plenty of disk memory etc.  
# My best guess is that making new processes (and killing them) at 
# the rate of one every few seconds, eventually upsets linux. 
# But I have no idea why.
# For now, I'll just detect this state and restart when it occurs.

function restart {
    log "checker($BASHPID): stopping monitor"
    ./runMonitor.sh stop
    log "checker($BASHPID): starting monitor"
    ./runMonitor.sh start
}

function overloadChecker {
    # This is a debug facility that uses a similar method to 
    # check for a manual "overload" and restart the monitor.
    # Run the overload.sh script to trigger this.
    while : ; do
        count=$(ps -ef | grep overload | wc -l)
        log "checker($BASHPID): $count"
        if (( $count >= 2 )) ; then     
            restart  
        fi
        checkWait=30
        log "checker($BASHPID): next overload check in $checkWait secs"
        sleep $checkWait
    done
}

overloadChecker &

while : ; do
    load=$(top -b -n2 | grep monitor | \
      awk '{sum+=$9} END { if (sum >= 60) print "PS-NBG" ; else print "PS-OK" }')
    log "checker($BASHPID): $load"
    if [[ "$load" == "PS-NBG" ]] ; then
        restart  
    fi
    # Once every 30 minutes is enough
    checkWait=1800
    log "checker($BASHPID): next monitor check in $checkWait secs"
    sleep $checkWait
done


exit 0
