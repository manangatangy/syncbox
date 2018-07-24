#!/bin/bash

# ----------- trap related routines ----------------
trap "onTerm" INT TERM ERR
trap "onExit" EXIT

function onExit {
    exitArgument=$?
    log "onExit exitArgument=$exitArgument"
    # this causes a trap to onTerm, passing the exit arg
    # via exitArgument channel
    kill 0 
}

function onTerm {
    termArgument=$?
    if [[ "$termArgument" != "0" ]] ; then
        # upon ctrl-C, a non-zero result value (130) occurs.
        # use this as the parameter to onExit.  Note that this
        # invocation of onTerm is the second last, and there
        # must be another final invocation, resulting from
        # onExit calling kill.
        exitArgument=$termArgument
        log "onTerm setting exitArgument=$exitArgument"
    else
        # This is the final invocation of onTerm, and the
        # call from here to exit will return to the shell
        # Put your final clean up here.
        # ...
        log "onTerm exiting with $exitArgument"
    fi
    # this call to exit doesn't result in trap to onExit; it
    # goes out to the caller, passing its arg to be available
    # to the caller as $?
    exit $exitArgument
}

function kill_process {
    # Single arg which is the pid of job to be killed.
    # Undefined, empty, or noJob arg means nothing will be killed.
    #echo "monitor: killing ${1}"
    if [[ ${1:-noJob} != noJob ]] ; then
        kill ${1} 2&>1 >/dev/null
    fi
}

# ----------- led routines ----------------
# ref: http://wiringpi.com/the-gpio-utility/
gpio -g mode 10 output

function led_on {
    kill_process ${ledPid:-noJob}
    gpio -g write 10 1
    ledPid=noJob
}

function led_off {
    kill_process ${ledPid:-noJob}
    gpio -g write 10 0
    ledPid=noJob
}

function led_blink {
    # Single arg which is speed of the blinking.
    kill_process ${ledPid:-noJob}
    case $1 in
        fast)  pause=0.125
        ;;
        slow)  pause=0.5
        ;;
        *) pause=0.25
        ;;
    esac
    (
    while : ; do
        gpio -g write 10 1
        sleep $pause
        gpio -g write 10 0
        sleep $pause
    done
    ) &
    # inhibit shell printing '[1] terminated ...'
    # ref: https://www.maketecheasier.com/run-bash-commands-background-linux/
    disown
    ledPid=$!
    #echo "monitor: new ledPid is $ledPid"
}

# ----------- lcd ----------------
function displayLcd {
    # Takes two display strings
    ~pi/display.py "$1" "$2" &>/dev/null
}

# ----------- status routines ----------------
function statusDisplay {
    # Takes two args, line1 and the cycle mode (0..4),  and
    # displays to lcd some info (mode dependent).
    # The first line displayed is arg1, second depends on the mode:
    # 0 --> "11:15 2018-07-24"  (date time)
    # 1 --> "uptime 234 days "
    # 2 --> "sync load 23%   "
    # 3 --> "diskuse root 99%"
    # 4 --> "diskuse sync 99%"
    line1=$1
    mode=$2
    case $mode in 
        0)
            displayLcd "${line1}" "$(date '+%R %F')"
            ;;
        1)
            displayLcd "${line1}" "$(getUptime)"
            ;;
        2)
            displayLcd "${line1}" "$(getSyncCpu)"
            ;;
        3)
            displayLcd "${line1}" "$(getDiskUsed root 'diskuse root')"
            ;;
        4)
            displayLcd "${line1}" "$(getDiskUsed syncdisk 'diskuse sync')"
            ;;
    esac
}

#    displayLcd "fetching system" "status..."
    # normal display "192.168.0.17    "
    #                "123 days, 2.91% "
#    displayLcd "${myIp}" "${upTime}, ${syncCpu}"

function checkConnectivity {
    # Writes to stdout an integer connectivity status:
    # 0 means no connectivity errors
    # 1 means no wifi
    # 2 means no gateway
    # 3 means no syncthing ping

    # Check there is a wifi connection
    # An alternative is iwconfig 2&>1 | grep wlan0 | grep ESSID
    wlanOk=$(nmcli | grep "wlan0: connected to" >/dev/null \
        && echo ok || echo error)
    if [[ $wlanOk != "ok" ]] ; then
        echo 1
    fi

    # Fetch gateway ip address and check connectivity to gateway
    gatewayIp=$(ip r | grep default | cut -d ' ' -f 3)
    pingGatewayOk=$(ping -q -w 1 -c 1 $gatewayIp >/dev/null \
        && echo ok || echo error)
    if [[ $pingGatewayOk != "ok" ]] ; then
        echo 2
    fi

    # Ping the syncthing api
    pingApiOk=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/ping 2>/dev/null | json_pp | \
        grep pong >/dev/null \
        && echo ok || echo error)
    if [[ $pingApiOk != "ok" ]] ; then
        echo 3
    fi
    echo 0
}

function displayConnectivity {
    # Perform various healths on wifi, network, syncthing and
    # if all good, print "192.168.0.17" (ip address) to stdout
    # and return status of 0.  Otherwise display the error msg,
    # echo it to stdout and return status of 1.

    case "$(checkConnectivity)" in
        0)
            echo "$(getIp)"
            return 0
            ;;
        1)
            displayLcd "no wifi, please" "reboot & config"
            errorText="no wifi"
            ;;
        2)
            displayLcd "no net gateway" "connectivity"
            errorText="no gateway"
            ;;
        3)
            displayLcd "$(getIp)" "syncthing dead?"
            errorText="no ping from syncthing"
            ;;
    esac
    echo "${errorText}"
    return 1
}

function getIp {
    # Returns 0, writes to stdout "192.168.0.99"
    myIp=$(ifconfig wlan0 | grep 'inet ' | awk '{ print $2 }')
    echo "$myIp"
}

function getUptime {
    # Returns 0, writes to stdout "uptime 234 days"
    # Fetch the pid, sedding out the 2nd grep (which is on grep command itself)
    syncPid=$(ps -ef | grep syncthing | sed -n '1p' | awk '{ print $2 }')

    # Fetch uptime from syncthing api
    uptimeSecs=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | json_pp | \
        grep uptime | tr -d ':,' | awk '{ print $2 }')

    # ref for exprs: http://wiki.bash-hackers.org/start
    if (( $uptimeSecs < 60)) ; then
        # less than 1 min
        uptimeTxt="$uptimeSecs secs"
    elif (( $uptimeSecs < 3600)) ; then
        # less than 1 hour
        uptimeTxt="$(($uptimeSecs / 60)) mins"
    elif (( $uptimeSecs < 86400)) ; then
        # less than 1 day
        uptimeTxt="$(($uptimeSecs / 3600)) hours"
    else
        uptimeTxt="$(($uptimeSecs / 86400)) days"
    fi
    echo "uptime $uptimeTxt"
}

function getSyncCpu {
    # Returns 0, writes to stdout "sync load 23%"
    # Check cpu load. 
    syncCpuTxt=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | json_pp | \
        grep cpuPercent | tr -d ':\",' | awk '{ print $2 }' | cut -c -5)
    echo "sync load ${syncCpuTxt}%"
}

function getDiskUsed {
    # Takes usage "getDiskUsed match text"
    # Returns 0, writes to stdout "text 23%"
    used=$(df --output=source,target,pcent | grep $1 \
            | awk '{ print $3 }')
    echo "$2 $used"
}

# ----------- button support ----------------
function button {
    # Waits for a button click, or a timeout.
    # Takes a single optional arg which is the number of seconds to wait.
    # Specifying 0, (or nothing) means wait forever for the button press.
    # Returns 0 if the button was pressed, or 1 if timed out waiting.
    waitTimeSecs=${1:-0}
    log "waiting $waitTimeSecs secs for button press"

    buttonPressed=0
    buttonTimedOut=1

    (
        # pgio27 is the button input
        gpio -g mode 27 in
        # tie the input up
        gpio -g mode 27 up   
        # wait indefinitely (in background) for button press (falling edge)
        gpio -g wfi 27 falling
        exit $buttonPressed 
    ) &
    buttonPid=$!

    (
        if (( $waitTimeSecs > 0)) ; then
            sleep $waitTimeSecs
        else
            sleep
        fi
        exit $buttonTimedOut 
    ) &
    timerPid=$!

    wait -n $gpioPid $timerPid
    waitStatus=$?
    log "button $( if [[ $waitStatus == $buttonPressed ]] ; \
        then echo 'PRESSED' ; else echo 'NOT pressed' ; fi )"

    kill_process ${buttonPid}
    kill_process ${timerPid}
    return $waitStatus
}

function log {
    echo "$(date --iso-8601=minutes) $1"
}

# ----------- main ----------------
if [[ $1 == "-logfile" ]] ; then
    logfile=${2:-logfile}
    exec &> $logfile
fi

log "startup"

# Waiting 10 seconds for the button press, blinking led to draw user attention.
led_blink fast
displayLcd "press button to" "configure wifi"
if button 10 ; then
    log "running wifi-connect"
    ##nmcli device wifi | while read line ; do log "$line" done
    led_blink
    displayLcd "wifi-connect at:" "Syncbox:wolfgang"
    sudo wifi-connect --portal-ssid Syncbox --portal-passphrase wolfgang
    log "connected to $(nmcli -t -f NAME con show)"
    led_off
fi

while : ; do
    if connectivityResult="$(displayConnectivity)" ; then
        # connectivityResult ==> myIp
        led_blink slow
        for mode in 0 1 2 3 4 ; do
            statusDisplay "${connectivityResult}" $mode
            sleep 6
            # 30 seconds for a complete cycle
        done
    else
        # connectivityResult ==> error text
        log "error:${connectivityResult}"
        led_blink fast
        sleep 5
    fi
done
exit 0

# "192.168.0.17    "
# "2018 03 30,20:10"
# "uptime 234 days "
# "sync load 23%   "
# "diskuse root 99%"
# "diskuse sync 99%"


# "diskuse 99% 99% "
# ""

# mail
# http://www.raspberry-projects.com/pi/software_utilities/email/ssmtp-to-send-emails
# this ref is about setting up without requiring a password
# https://blog.dantup.com/2016/04/setting-up-raspberry-pi-raspbian-jessie-to-send-email/

