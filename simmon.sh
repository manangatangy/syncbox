#!/bin/bash

# Simplified monitor

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

# ----------- housekeeping routines ----------------
echoedText=""

function log {
    # Echo to stdout, which is prob redirected
    # Only echo if the arg is different from the last.
    if [[ "$echoedText" != "$1" ]] ; then
        echo "$(date --iso-8601=minutes) $1"
        echoedText="$1"
    fi
}

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

# ----------- pauseAndDisplayIsButtonDown ----------------
# Loop up to [sec] times, each time around; displaying the two lines,
# appending [secs-remaining] to the second line, and toggling the led.
# On each loop check and return true if the button's down.
# Return false once the loop finishes.
function pauseAndDisplayIsButtonDown {
    secs="$1"
    text1="$2"
    text2="$3"
    while : ; do
        if (( $secs <= 0 )) ; then
            return 1
        fi
        displayLcd "$text1" "$text2[$secs]"
        toggleLed
        sleep 1
        if isButtonDown ; then
            return 0
        fi
        secs="$(( $secs - 1 ))"
    done
}

# ----------- flashQuickly ----------------
# Display the two lines of text and flash the led quickly for specified
# number of time.  No checking of the button.
function flashQuickly {
    # each count is 0.1 of a second
    count="$1"
    text1="$2"
    text2="$3"
    displayLcd "$text1" "$text2"
    while : ; do
        if (( count <= 0 )) ; then
            return
        fi
        toggleLed
        sleep 0.1
        count="$(( $count - 1 ))"
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

function confirmIfShutDownRequested {
    # This will typically be called directly after isButtonDown has
    # returned true, so if we immediately check again for button down,
    # it will probably quickly return true.  Therefore, pause for a few
    # seconds and clear the lcd so the user knows his first press has been
    # acknowledged, and they will hopefully release the button.  Then
    # display and ask for confirmation.
    flashQuickly 30 "pausing...  "  " "
    if pauseAndDisplayIsButtonDown 5 "press again to" "shut down    " ; then
        echo "log: user requested shutdown"
        flashQuickly 30 "shutting down...  "  " "
        #### TODO exec sudo shutdown now
        exit 0
    fi
}

# ----------- main ----------------
function main {

    # We pause for about 10 secs to allow some time for the services to start
    flashQuickly 50 "starting syncbox..."  " "
    log "STARTING SYNCBOX"

    if ! testWlan ; then
        log "no connection at startup; running wifi-connect"
        displayLcd "wifi-connect at:" "Syncbox:wolfgang"
        sudo wifi-connect --portal-ssid Syncbox \
            --portal-passphrase wolfgang
        log "connected to $(nmcli -t -f NAME con show)"
    else
        # Although we are already connected, allow chance to re-config
        if pauseAndDisplayIsButtonDown 5 "re-configure" "wifi ?       " ; then
            log "user requested at startup; running wifi-connect"
            displayLcd "wifi-connect at:" "Syncbox:wolfgang"
            sudo wifi-connect --portal-ssid Syncbox \
                --portal-passphrase wolfgang
            log "connected to $(nmcli -t -f NAME con show)"
        fi
    fi

    while : ; do
        syncAvailability="nbg"
        # 1. Check there is a wifi connection
        if ! testWlan ; then
            line1="no wifi"
            line2="connection   "
        else
            # 2. Fetch gateway ip address and check connectivity to gateway
            if ! testGateway ; then
                line1="no gateway"
                line2="connectivity "
            else
                # 3. Ping the syncthing api
                if ! testSyncPing ; then
                    line1="no ping from"
                    line2="syncthing    "
                else
                    # All tests are ok
                    syncAvailability="ok"
                fi
            fi
        fi
        if [[ "$syncAvailability" == "nbg" ]] ; then
            # Display the error for 5 secs, checking for button-press-shutdown-request
            log "error: $line1 $line2"
            if pauseAndDisplayIsButtonDown 5 "$line1" "$line2" ; then
                confirmIfShutDownRequested
            fi
        else
            myIp="$(getIp)"
            log "connected, ip=$myIp"
            if pauseAndDisplayIsButtonDown 5 "$(date '+%R %F')" "$myIp " ; then
                confirmIfShutDownRequested
            fi
            if pauseAndDisplayIsButtonDown 5 "$(getUptime)" "$myIp " ; then
                confirmIfShutDownRequested
            fi
            if pauseAndDisplayIsButtonDown 5 "$(getSyncCpu)" "$myIp " ; then
                confirmIfShutDownRequested
            fi
        fi
    done
}

if [[ $1 == "-logfile" ]] ; then
    logfile=${2:-logfile}
    exec &>> $logfile
fi

main
