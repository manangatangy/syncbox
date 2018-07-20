#!/bin/bash

trap "exit" INT TERM ERR
trap "kill 0" EXIT

function kill_process {
    echo "killing ${1}"
    if [[ ${1-noJob} != noJob ]] ; then
        kill ${1} 
    fi
}

# ----------- led support ----------------
gpio -g mode 10 output

function led_on {
    kill_process ${ledPid-noJob}
    gpio -g write 10 1
    ledPid=noJob
}

function led_off {
    kill_process ${ledPid-noJob}
    gpio -g write 10 0
    ledPid=noJob
}

function led_blink {
    kill_process ${ledPid-noJob}
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
    disown
    ledPid=$!
    echo "new ledPid is $ledPid"
}

# ----------- status checks ----------------
function status {
    # Check there is a wifi connection
    # An alternative is iwconfig 2>&1 | grep wlan0 | grep ESSID
    wlanOk=$(nmcli | grep "wlan0: connected to" >/dev/null \
        && echo ok || echo error)
    if [[ $wlanOk != "ok" ]] ; then
        ./display.py "no wifi, please" "reboot & config"
        return 1
    fi

    # Fetch ip address and check connectivity to gateway
    gatewayIp=$(ip r | grep default | cut -d ' ' -f 3)
    pingGatewayOk=$(ping -q -w 1 -c 1 $gatewayIp >/dev/null \
        && echo ok || echo error)
    if [[ $pingGatewayOk != "ok" ]] ; then
        ./display.py "no net gateway" "connectivity"
        return 1
    fi

    # Ping the syncthing api
    pingApiOk=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/ping 2>/dev/null | json_pp | \
        grep pong >/dev/null \
        && echo ok || echo error)
    if [[ $pingApiOk != "ok" ]] ; then
        ./display.py "no api-ping" "syncthing dead?"
        return 1
    fi

    # Fetch uptime from syncthing api
    uptimeSecs=$(curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  \
        http://127.0.0.1:8384/rest/system/status 2>/dev/null | json_pp | \
        grep uptime | tr -d ':,' | awk '{ print $2 }')

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

    myIp=$(ifconfig wlan0 | grep 'inet ' | awk '{ print $2 }')

    # Check cpu load
    userCpu=$(top -b -n 1 -p 0 | grep Cpu | awk '{ print $2 }')
    sysCpu=$(top -b -n 1 -p 0 | grep Cpu | awk '{ print $4 }')
    totalCpu=$(echo $userCpu $sysCpu | awk '{ print $1 + $2 }')

    # normal display "ip:192.168.0.17"
    #                "123 days, 2.91%"
    ./display.py "ip:${myIp}" "${uptime}, ${totalCpu}%"
    return 0
}

if status ; then
    led_blink slow
else
    led_blink fast
fi

echo "myIp=$myIp"
echo "pingGatewayOk=$pingGatewayOk"
echo "totalCpu=$totalCpu"
echo "pingApiOk=$pingApiOk"
echo "uptimeSecs=$uptimeSecs"

sleep 25
exit

# ----------- led support ----------------
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


# ----------- led support ----------------

# blink while waiting 5 seconds for the button press
#./blink &
led_blink fast

#BLINK=$!
echo "monitor: waiting 5 secs for button press"
./display.py "press button to" "configure wifi"
./button.sh 5
BUTTON=$?
if [ "$BUTTON" -eq "0" ] ; then
        led_blink
        ./display.py "connect to wifi" "Syncbox:wolfgang"
        echo wifi-connect --portal-ssid Syncbox --portal-passphrase wolfgang
        sleep 10
        led_off
fi

led_off






st_cpu=`curl -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  http://127.0.0.1:8384/rest/system/status |json_pp | grep cpuPercent | tr -d ':\",' | awk '{ print $2 }' | cut -c -6`
echo "syncthing cpu is $st_cpu"

./display.py "cpu $st_cpu%" ""
# curl -X GET -H "X-API-Key: GSwL53QQ96gZWJU5DpDTnqzJTzi2bn4K"  http://127.0.0.1:8384/rest/stats/folder | json_pp

