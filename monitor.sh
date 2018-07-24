#!/bin/bash

# ----------- trap related routines ----------------
trap "onTerm" INT TERM ERR
trap "onExit" EXIT

function onExit {
    exitArgument=$?
    echo "monitor: onExit($exitArgument)"
    # this causes a trap to onTerm, passing the exit arg
    # via exitArgument channel
    kill 0 
}

function onTerm {
    termArgument=$?
    echo "monitor: onTerm($termArgument) exitArgument=$exitArgument"
    if [[ "$termArgument" != "0" ]] ; then
        # upon ctrl-C, a non-zero result value (130) occurs.
        # use this as the parameter to onExit.  Note that this
        # invocation of onTerm is the second last, and there
        # must be another final invocation, resulting from
        # onExit calling kill.
        exitArgument=$termArgument
        echo "monitor: onTerm setting exitArgument=$exitArgument"
    else
        # This is the final invocation of onTerm, and the
        # call from here to exit will return to the shell
        # Put your final clean up here.
        # ...
        echo "monitor: onTerm exiting with $exitArgument"
    fi
    # this call to exit doesn't result in trap to onExit; it
    # goes out to the caller, passing its arg to be available
    # to the caller as $?
    exit $exitArgument
}

function kill_process {
    # Single arg which is the pid of job to be killed.
    # Undefined, empty, or noJob arg means nothing will be killed.
    echo "monitor: killing ${1}"
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
    echo "monitor: new ledPid is $ledPid"
}

# ----------- lcd ----------------
function displayLcd {
    # Takes two display strings
    ~pi/display.py "$1" "$2"
}

# ----------- status routines ----------------
function status {
    # Perform various healths on wifi, network, syncthing, and 
    # return 0 if everything is good. If not ok, then display 
    # errors to the LCD and return false.
    # If no error, display line 1 
    #    is "192.168.0.17    ", line 2 depends on the arg:
    # 0 --> "2018 03 30,20:10"  (date time)
    # 1 --> "uptime 234 days "
    # 2 --> "sync load 23%   "
    # 3 --> "diskuse root 99%"
    # 4 --> "diskuse sync 99%"
    # The connectivity check is only performed for arg=0

    displayLcd "fetching system" "status..."

    getConnectivity ; connectivityOk=$?
    if (( $connectivityOk != 0 )) ; then
        case $? in
            1)
                displayLcd "no wifi, please" "reboot & config"
                ;;
            2)
                displayLcd "no net gateway" "connectivity"
                ;;
            3)
                displayLcd "${myIp}" "syncthing dead?"
                ;;
            *)
                displayLcd "connectivity" "error $connectivityOk"
                ;;
        esac
    else
    fi

    myIp=$(ifconfig wlan0 | grep 'inet ' | awk '{ print $2 }')

    upTime=$(getUptime)
    syncCpu=$(getSyncCpu)


    # normal display "192.168.0.17    "
    #                "123 days, 2.91% "
    displayLcd "${myIp}" "${upTime}, ${syncCpu}"
    return 0
}

function getConnectivity {
    # Returns 0, if connectivity is ok, otherwise
    # there is a connectivity error, then return error code:
    # 1 -> no wifi
    # 2 -> no gateway
    # 3 -> no syncthing ping

    # Check there is a wifi connection
    # An alternative is iwconfig 2&>1 | grep wlan0 | grep ESSID
    wlanOk=$(nmcli | grep "wlan0: connected to" >/dev/null \
        && echo ok || echo error)
    if [[ $wlanOk != "ok" ]] ; then
        return 1
    fi

    # Fetch gateway ip address and check connectivity to gateway
    gatewayIp=$(ip r | grep default | cut -d ' ' -f 3)
    pingGatewayOk=$(ping -q -w 1 -c 1 $gatewayIp >/dev/null \
        && echo ok || echo error)
    if [[ $pingGatewayOk != "ok" ]] ; then
        return 2
    fi

    # Ping the syncthing api
    pingApiOk=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/ping 2>/dev/null | json_pp | \
        grep pong >/dev/null \
        && echo ok || echo error)
    if [[ $pingApiOk != "ok" ]] ; then
        return 3
    fi
    return 0
}

function getIp {
    # Returns 0, writes to stdout "192.168.0.99"
    myIp=$(ifconfig wlan0 | grep 'inet ' | awk '{ print $2 }')
    echo "$myIp"
    return 0
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
    return 0
}

function getSyncCpu {
    # Returns 0, writes to stdout "sync load 23%"
    # Check cpu load. 
    syncCpuTxt=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | json_pp | \
        grep cpuPercent | tr -d ':\",' | awk '{ print $2 }' | cut -c -5)
    echo "${syncCpuTxt}%"
    return 0
}

# ----------- button support ----------------
function button {
    # Waits for a button click, or a timeout.
    # Takes a single optional arg which is the number of seconds to wait.
    # Specifying 0, (or nothing) means wait forever for the button press.
    # Returns 0 if the button was pressed, or 1 if timed out waiting.
    waitTimeSecs=${1:-0}
    echo "monitor: waiting $waitTimeSecs secs for button press"

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
    echo "monitor: buttonPid=$buttonPid"

    (
        if (( $waitTimeSecs > 0)) ; then
            sleep $waitTimeSecs
        else
            sleep
        fi
        exit $buttonTimedOut 
    ) &
    timerPid=$!
    echo "monitor: timerPid=$timerPid"

    wait -n $gpioPid $timerPid
    waitStatus=$?
    echo "monitor: button $( if [[ $waitStatus == $buttonPressed ]] ; \
        then echo 'PRESSED' ; else echo 'NOT pressed' ; fi )"

    kill_process ${buttonPid}
    kill_process ${timerPid}
    return $waitStatus
}


# ----------- main ----------------

if [[ $1 == "-logfile" ]] ; then
    logfile=${2:-logfile}
    exec &> $logfile
fi

# Waiting 10 seconds for the button press, blinking led to draw user attention.
led_blink fast
displayLcd "press button to" "configure wifi"
if button 10 ; then
    echo "monitor: running wifi-connect"
    led_blink
    displayLcd "wifi-connect at:" "Syncbox:wolfgang"
    sudo wifi-connect --portal-ssid Syncbox --portal-passphrase wolfgang
    led_off
fi

while : ; do
    if status ; then
        led_blink slow
    else
        led_blink fast
    fi
    sleep 30
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

