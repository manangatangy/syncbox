#!/bin/bash
trap "onTerm" INT TERM ERR
trap "onExit" EXIT

function onExit {
    exitArgument=$?
    echo "onExit($exitArgument)"
    # this causes a trap to onTerm, passing the exit arg
    # via exitArgument channel
    kill 0 
}

function onTerm {
    termArgument=$?
    echo "onTerm($termArgument) exitArgument=$exitArgument"
    if [[ "$termArgument" != "0" ]] ; then
        # upon ctrl-C, a non-zero result value (130) occurs.
        # use this as the parameter to onExit.  Note that this
        # invocation of onTerm is the second last, and there
        # must be another final invocation, resulting from
        # onExit calling kill.
        exitArgument=$termArgument
        echo "onTerm setting exitArgument=$exitArgument"
    else
        # This is the final invocation of onTerm, and the
        # call from here to exit will return to the shell
        # Put your final clean up here.
        # ...
        echo "onTerm exiting with $exitArgument"
    fi
    # this call to exit doesn't result in trap to onExit; it
    # goes out to the caller, passing its arg to be available
    # to the caller as $?
    exit $exitArgument
}

sleep 5
# the arg to exit is available in onExit as $?
exit 0

