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

# ----------- status checks ----------------
function status {
    # Perform various healths on wifi, network, syncthing, and return 0 if 
    # everything is good. If not ok, the display errors to the LCD and return false.

    # Check there is a wifi connection
    # An alternative is iwconfig 2&>1 | grep wlan0 | grep ESSID
    displayLcd "fetching system" "status..."

    wlanOk=$(nmcli | grep "wlan0: connected to" >/dev/null \
        && echo ok || echo error)
    if [[ $wlanOk != "ok" ]] ; then
        displayLcd "no wifi, please" "reboot & config"
        return 1
    fi

    # Fetch ip address and check connectivity to gateway
    gatewayIp=$(ip r | grep default | cut -d ' ' -f 3)
    pingGatewayOk=$(ping -q -w 1 -c 1 $gatewayIp >/dev/null \
        && echo ok || echo error)
    if [[ $pingGatewayOk != "ok" ]] ; then
        displayLcd "no net gateway" "connectivity"
        return 1
    fi

    myIp=$(ifconfig wlan0 | grep 'inet ' | awk '{ print $2 }')

    # Ping the syncthing api
    pingApiOk=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/ping 2>/dev/null | json_pp | \
        grep pong >/dev/null \
        && echo ok || echo error)
    if [[ $pingApiOk != "ok" ]] ; then
        displayLcd "${myIp}" "syncthing dead?"
        return 1
    fi

    # Fetch uptime from syncthing api
    uptimeSecs=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | json_pp | \
        grep uptime | tr -d ':,' | awk '{ print $2 }')

    # ref for exprs: http://wiki.bash-hackers.org/start
    if (( $uptimeSecs < 60)) ; then
        # less than 1 min
        uptime="$uptimeSecs secs"
    elif (( $uptimeSecs < 3600)) ; then
        # less than 1 hour
        uptime="$(($uptimeSecs / 60)) mins"
    elif (( $uptimeSecs < 86400)) ; then
        # less than 1 day
        uptime="$(($uptimeSecs / 3600)) hours"
    else
        uptime="$(($uptimeSecs / 86400)) days"
    fi

    # Check cpu load. 
    syncCpu=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | json_pp | \
        grep cpuPercent | tr -d ':\",' | awk '{ print $2 }' | cut -c -5)

    # Fetch the pid, sedding out the 2nd grep (which is on grep command itself)
    syncPid=$(ps -ef | grep syncthing | sed -n '1p' | awk '{ print $2 }')

    # normal display "ip:192.168.0.17"
    #                "123 days, 2.91%"
    displayLcd "${myIp}" "${uptime}, ${syncCpu}%"
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

# Waiting 5 seconds for the button press, blinking led to draw user attention.
led_blink fast
displayLcd "press button to" "configure wifi"
if button 5 ; then
    echo "monitor: running wifi-connect"
    led_blink
    displayLcd "wifi-connect at:" "Syncbox:wolfgang"
    echo "wifi-connect --portal-ssid Syncbox --portal-passphrase wolfgang"
    sleep 10		; # simulate wifi connect
    led_off
fi

while : ; do
    # Update display/led with sys status. This is done asyn because it takes 
    # quite a while and we don't want the button to be unresponsive for that time.
    
    if status ; then
        led_blink slow
    else
        led_blink fast
    fi
    
    echo "monitor: sleeping for a bit..."
    sleep 30

done










echo "myIp=$myIp"
echo "pingGatewayOk=$pingGatewayOk"
echo "syncCpu=$syncCpu"
echo "syncPid=$syncPid"
echo "pingApiOk=$pingApiOk"
echo "uptimeSecs=$uptimeSecs"

sleep 25
exit


# ----------- led support ----------------
led_blink fast
sleep 5
led_on
sleep 3
led_blink slow
sleep 5
led_off
led_on
led_off
exit







st_cpu=`curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  http://127.0.0.1:8384/rest/system/status |json_pp | grep cpuPercent | tr -d ':\",' | awk '{ print $2 }' | cut -c -6`
echo "syncthing cpu is $st_cpu"

./display.py "cpu $st_cpu%" ""
# curl -X GET -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  http://127.0.0.1:8384/rest/stats/folder | json_pp

