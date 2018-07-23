#!/bin/bash

# Should be executed as user pi
# Runs the monitor script as a background process in
# pi home, writing all output to ~/monitor.log
# eg place this in /etc/rc.local:
#    sudo -u pi ~/runMonitor.sh &
# The & is needed because rc.local exits and we want

cd ~
rm runMonitor.log monitor.log
echo "runMonitor: id=$(id)"  >>runMonitor.log
echo "runMonitor: pwd=$(pwd)"  >>runMonitor.log
echo "runMonitor: starting /.monitor.sh"  >>runMonitor.log

##./monitor.sh 2>&1 >monitor.log &
## https://stackoverflow.com/a/11255498
## &>logfile
## ref: http://man7.org/linux/man-pages/man8/start-stop-daemon.8.html


(
while : ; do
    echo "test monitor stil running..." >>monitor.log
    sleep 10
done
) &

echo "runMonitor: monitor pid $!" >>runMonitor.log
wait
exit $?

