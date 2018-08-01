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

function killProcess {
    # Single arg which is the pid of job to be killed.
    # Undefined, empty, or noJob arg means nothing will be killed.
    #echo "monitor: killing ${1}"
    if [[ ${1:-noJob} != noJob ]] ; then
        kill ${1} >/dev/null 2>&1
    fi
}

# ----------- led routines ----------------
# ref: http://wiringpi.com/the-gpio-utility/
gpio -g mode 10 output

# Prevent uneccesary kill/fork if no change in led.
ledArg=""

function led {
    if [[ "$ledArg" == "$1" ]] ; then
        return
    fi

    ledArg="$1"
    # Single arg which is speed of the blinking.
    killProcess ${ledPid:-noJob}
    case $1 in
        fast)  pause=0.125
            ;;
        slow)  pause=1.0
            ;;
        normal) pause=0.5
            ;;
        on) pause=0
            gpio -g write 10 1
            return
            ;;
        off) pause=0
            gpio -g write 10 0
            return
            ;;
        *)
            return
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
    # https://unix.stackexchange.com/questions/3886/difference-between-nohup-disown-and
    disown
    ledPid=$!
    #echo "monitor: new ledPid is $ledPid"
}

# ----------- lcd ----------------
function displayLcd {
    # Takes two display strings
    ./display.py "$1" "$2" >/dev/null 2>&1
}

# ----------- button support ----------------
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
        exit 1
    ) &
    timerPid=$!

    wait -n $buttonPid $timerPid
    # status code 0 (true) indicates that gpio process finished.
    # status code 1 (false) indicates that sleep process finished.
    waitStatus=$?

    killProcess ${buttonPid}
    killProcess ${timerPid}
    return $waitStatus
}

# ----------- availability check routines ----------------
function testWlan {
    # An alternative is iwconfig 2>&1 | grep wlan0 | grep ESSID
    nmcli | grep "wlan0: connected to" >/dev/null 2>&1
}

function testGateway {
    gatewayIp=$(ip r | grep default | cut -d ' ' -f 3)
    ping -q -w 1 -c 1 $gatewayIp >/dev/null 2>&1
}

function testSyncPing {
    curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/ping 2>/dev/null | \
        json_pp 2>/dev/null | grep pong >/dev/null
}

function syncAvailability {
    # Checks for availability and if all good, then writes
    # ip-address to stdout and sets return status 0.
    # Otherwise writes an error message to stdout:
    # "no wifi", "no gateway", or "no ping from syncthing",
    # displays an an error message on the LCD, set return status 1.

    # 1. Check there is a wifi connection
    if ! testWlan ; then
        displayLcd "no wifi, maybe" "config needed ?"
        echo "no wifi"
        return 1
    fi

    # 2. Fetch gateway ip address and check connectivity to gateway
    if ! testGateway ; then
        displayLcd "no net gateway" "connectivity"
        echo "no gateway"
        return 1
    fi

    myIp="$(getIp)"

    # 3. Ping the syncthing api
    if ! testSyncPing ; then
        displayLcd "${myIp}" "syncthing dead?"
        echo "no ping from syncthing"
        return 1
    fi

    echo "${myIp}"
    return 0
}

# ----------- status routines ----------------
function getIp {
    # Writes to stdout "192.168.0.99"
    myIp=$(ifconfig wlan0 | grep 'inet ' | awk '{ print $2 }')
    echo "$myIp"
}

function getUptime {
    # Writes to stdout "uptime 234 days"
    # Fetch the pid, sedding out the 2nd grep (which is on grep command itself)
    syncPid=$(ps -ef | grep syncthing | sed -n '1p' | awk '{ print $2 }')

    # Fetch uptime from syncthing api
    uptimeSecs=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | \
        json_pp 2>/dev/null | \
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
    # Writes to stdout "sync load 23%"
    # Check cpu load. 
    syncCpuTxt=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | \
        json_pp 2>/dev/null | \
        grep cpuPercent | tr -d ':\",' | awk '{ print $2 }' | cut -c -5)
    echo "sync load ${syncCpuTxt}%"
}

function getDiskUsed {
    # Takes usage "getDiskUsed match text"
    # Writes to stdout "text 23%"
    used=$(df --output=source,target,pcent | grep $1 \
            | awk '{ print $3 }')
    echo "$2 $used"
}

# ----------- status routines ----------------
function displayStatus {
    # Takes two args, line1 and the cycle mode (0..4),  and
    # displays to lcd some info depending on the mode.
    # The first line displayed is arg1, second depends on the mode:
    # 0 --> "11:15 2018-07-24"  (date time)
    # 1 --> "uptime 234 days "
    # 2 --> "sync load 23%   "
    # 3 --> "diskuse root 99%"
    # 4 --> "diskuse sync 99%"
    line1=$1
    mode=$2
    case $mode in
        0)  displayLcd "${line1}" "$(date '+%R %F')"
            ;;
        1)  displayLcd "${line1}" "$(getUptime)"
            ;;
        2)  displayLcd "${line1}" "$(getSyncCpu)"
            ;;
        3)  displayLcd "${line1}" "$(getDiskUsed root 'diskuse root')"
            ;;
        4)  displayLcd "${line1}" "$(getDiskUsed syncdisk 'diskuse sync')"
            ;;
    esac
}

echoedText=""

function log {
    # Only echo if the arg is different from the last.
    if [[ "$echoedText" != "$1" ]] ; then
        echo "$(date --iso-8601=minutes) $1"
        echoedText="$1"
    fi
}

# ----------- nmcli routines ----------------
function deleteConnections {
    nmcli --fields NAME con show | awk '{print $1}' | while read name
    do
        if [[ "$name" != "NAME" ]] ; then
            log "deleting connection $name"
            sudo nmcli con del id "$name" || true
        fi
    done 
}

# ----------- option selection ----------------
function pauseForOptionSelect {
    # Pause for a short while, waiting for the button press.
    # if none, then return.
    # Otherwise, step through a list of options, pausing at each
    # for a short time for the user to click the button.
    if ! gpioButton 6 ; then
        return
    fi

    led fast
    displayLcd "press to select" " an option..."

    sleep 3
    if testGateway ; then
        displayLcd "1. email status" "   report ?"
        if gpioButton 3 ; then
            log "user request email report"
            displayLcd "emailing ..." ""
            ./report.sh -email || true
            return
        fi
    fi 
    displayLcd "2. shutdown" "   syncbox ?"
    if gpioButton 3 ; then
        log "user request shutdown"
        displayLcd "shutting down..." ""
        sudo shutdown now
        exit 0
    fi
    displayLcd "3. reboot" "   syncbox ?"
    if gpioButton 3 ; then
        log "user request reboot"
        displayLcd "rebooting ..." ""
        sudo reboot
        exit 0
    fi
    displayLcd "4. configure" "   wifi ?"
    if gpioButton 3 ; then
        log "running wifi-connect"
        displayLcd "wifi-connect at:" "Syncbox:wolfgang"
        sudo wifi-connect --portal-ssid Syncbox \
            --portal-passphrase wolfgang
        log "connected to $(nmcli -t -f NAME con show)"
        return
    fi
    displayLcd "5. reset" "   wifi ?"
    if gpioButton 3 ; then
        log "user request reset wifi connections"
        displayLcd "resetting ..." ""
        deleteConnections
        return
    fi
}

# ----------- main ----------------
function main {
    log "startup"
    led on

    while : ; do
        if result="$(syncAvailability)" ; then
            # result ==> myIp
            log "ok:${result}"
            for mode in 0 1 2 3 4 ; do
                led slow
                displayStatus "${result}" $mode
                pauseForOptionSelect
            done
        else
            # result ==> error text
            log "error:${result}"
            led fast
            pauseForOptionSelect
        fi
    done
}

# ----------- entry ----------------
if [[ $1 == "-logfile" ]] ; then
    logfile=${2:-logfile}
    exec &>> $logfile
fi

main

#todo: scheduled report email


