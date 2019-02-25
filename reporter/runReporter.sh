#!/bin/bash

NAME=reporter
USER=pi
WDIR=$(eval echo "~$USER")/syncbox/reporter
PIDFILE=$WDIR/$NAME.pid
#This is the command to be run, give the full pathname
DAEMON=$WDIR/${NAME}
LOGFILE=$WDIR/${NAME}.log

case "$1" in
  start)
    echo -n "Starting: "$NAME
    start-stop-daemon \
        --start \
        --pidfile $PIDFILE \
        --user $USER \
        --make-pidfile \
        --startas $DAEMON \
        --background \
        --chuid $USER \
        --chdir $WDIR \
        --verbose \
        -- -config syncbox-config.json -logfile $LOGFILE 
    ;;
  stop)
    echo -n "Stopping: "$NAME
    start-stop-daemon \
        --stop \
        --pidfile $PIDFILE \
        --user $USER \
        --remove-pidfile \
        --retry=TERM/30/KILL/5 \
        --verbose
    ;;
  status)
    # Ref: http://refspecs.linuxfoundation.org/LSB_3.1.0/LSB-Core-generic/LSB-Core-generic/iniscrptact.html
    start-stop-daemon \
        --status \
        --pidfile $PIDFILE \
        --user $USER 
    status=$?
    case "$status" in
      0)
        echo "$NAME is running OK"
        ;;
      1)
        echo "$NAME is dead and /var/run pid file exists"
        ;;
      2)
        echo "$NAME is dead and /var/lock lock file exists"
        ;;
      3)
        echo "$NAME is not running"
        ;;
      *)
        echo "$NAME status is unknown"
        ;;
    esac
    exit $status
    ;;
  *)
    echo "Usage: "$1" {start|stop|status}"
    exit 1
esac

exit 0
