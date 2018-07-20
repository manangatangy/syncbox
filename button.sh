#!/bin/bash

# Start two jobs; one with a timer and one with the gpio (listening for button)
# pgio27 is the button input
gpio -g mode 27 in
# tie the input up
gpio -g mode 27 up   
# wait indefinitely (in background) for button press (falling edge)
( gpio -g wfi 27 falling ; exit 2 ) &
gpioPid=$!
echo "gpioPid=$gpioPid"

( sleep 5 ; exit 3 ) &
timerPid=$!
echo "timerPid=$timerPid"

wait -n $gpioPid $timerPid
waitStatus=$?
echo "waitStatus=$waitStatus"

exit 0
echo uhuhuhuhu

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
    echo "new ledPid is $ledPid"








usage="$0 max_wait_seconds (0 means forever)"
# returns; 1 on error
#	0 on button press
#	2 on timeout

if [ "$#" -ne "1" ] ; then 
    echo "error, usage: $usage"
    exit 1
fi

if [ "$1" = "0" ] ; then
	wait_forever="yes"
else
	wait_forever="no"
fi

# pgio27 is the button input
gpio -g mode 27 in
# tie the input up
gpio -g mode 27 up   
# wait indefinitely (in background) for button press (falling edge)
gpio -g wfi 27 falling &
gpioPid=$!

TIMER=$1
while [ true ]; do 
	sleep 1
	# check if the gpio process is still runnning
	ps -p $PID 2>&1 >/dev/null
	PS=$?
	if [ "$PS" -ne "0" ] ; then
		echo "button: press occurred, returning 0"
		exit 0		;# press occurred
	fi
	if [ "$wait_forever" == "no" ] ; then
		TIMER=`expr $TIMER - 1`	
		if [ $TIMER -le 0 ] ; then
			echo "button: timer expired, returning 2"
			kill $PID 2>&1 >/dev/null
			exit 2		;# time expired, no press
		fi
	fi
done
