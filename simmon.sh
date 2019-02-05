#!/bin/bash

# Simplified monitor


# ----------- led/button routines ----------------
# ref: http://wiringpi.com/the-gpio-utility/
# Initialise the led on pin 10 and the button on pin 27 (pgio)
gpio -g mode 10 output
gpio -g mode 27 in
# tie the input up
gpio -g mode 27 up

ledState="0"

function toggleLed {
    if [[ "$ledState" == "0" ]] ; then
        ledState="1"
    else
        ledState="0"
    fi
    gpio -g write 10 "$ledState"
}

function isButtonDown {
    # gpio => "0" (if button down) or "1" (if button up)
    test "$(gpio -g read 27)" -eq "0"
}

# ----------- lcd ----------------
function displayLcd {
    # Takes two display strings
    ./display.py "$1" "$2" >/dev/null 2>&1
}

# ----------- flashSleepIsButtonDown ----------------
# Loop up to [sec] times, each time around; displaying the two lines,
# appending [secs-remaining] to the second line, and toggling the led.
# On each loop check and return true if the button's down.
# Return false once the loop finishes.
function flashSleepIsButtonDown {
    secs="$1"
    text1="$2"
    text2="$3"
    while : ; do
        if (( $secs <= 0 )) ; then
            return 1
        fi
        displayLcd "$text1" "$text2 [$secs]"
        toggleLed
        sleep 1
        if isButtonDown ; then
            return 0
        fi
        secs="$(( $secs - 1 ))"
    done
}

# ----------- availability check routines (return true/false) ----------------
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

# ----------- status routines (write to stdout) ----------------
function getIp {
    # Writes to stdout "192.168.0.99"
    myIp=$(ifconfig wlan0 | grep 'inet ' | awk '{ print $2 }')
    echo "$myIp"
}

function getUptime {
    # Writes to stdout "uptime 234 days"
    # Fetch the pid, sedding out the 2nd grep (which is on grep command itself)
    ##syncPid=$(ps -ef | grep syncthing | sed -n '1p' | awk '{ print $2 }')
    # There was a failure (with cpu of around 70%) that traced to here
    # with the shell starting ps, grep, sed, and awk and then no more
    # traceouts.  This pipeline had got stuck - don't know how.
    # Will try again with this command commented out - syncPid not used anyway.

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

# ----------- main ----------------

# flashSleepButtonCheck 5 'starting syncbox...' ''
# if not-connected
#     display 'wifi-connect at ...'
#     exec wifi-connect
# else
#     flashSleepButtonCheck 're-configure', 'wifi ?'
#     if button-pressed
#         display 'wifi-connect at ...'
#         exec wifi-connect
#     fi
# fi
#
# loop-forever
#     check syncAvailability, if bad then
#         flashSleepButtonCheck 5 error-message-line-1 error-message-line-2
#         checkIfShutDownRequested
#     else
#         flashSleepButtonCheck 3 info-1-line-1 info-1-line-2
#         checkIfShutDownRequested
#         flashSleepButtonCheck 3 info-2-line-1 info-2-line-2
#         checkIfShutDownRequested
#         flashSleepButtonCheck 3 info-3-line-1 info-3-line-2
#         checkIfShutDownRequested
#     fi
# end-forever
#
# function checkIfShutDownRequested
#     if button-pressed
#         flashSleepButtonCheck 5 'press again' 'to shut down?'
#         if button-pressed
#             display 'shutting down...'
#             exec sudo shutdown now
#         fi
#     fi
# end-function


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

