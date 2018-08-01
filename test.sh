#!/bin/bash

# A report is to be generated/emailed x days after the last one
# (even if that was manually requested) or after the startup.
# 
# 
reportFileName="xxx"
lastReportDate=$(ls -l --time-style=full-iso "$reportFileName" | awk '{ print $6, $7, $8 }')
echo "lastReportDate=$lastReportDate"
nextReportDate=$(date -d "$lastReportDate +1 day")
echo "nextReportDate=$nextReportDate"

nextReportSecs=$(date -d "$nextReportDate" "+%s")
echo "nextReportSecs=$nextReportSecs"

nowSecs=$(date "+%s")
secsUntilReportDue=$(( $nextReportSecs - $nowSecs ))
echo "secsUntilReportDue=$secsUntilReportDue"

#date -d '@1533131552' "+%F %T %Z"


lastReportDate=2018-08-02 00:16:56.036975022 +1000
pi@syncbox:~/syncbox $ nextReportDate=$(date -d "$lastReportDate +1 day")
pi@syncbox:~/syncbox $ echo "nextReportDate=$nextReportDate"
nextReportDate=Fri Aug  3 00:16:56 AEST 2018
pi@syncbox:~/syncbox $ nextReportSecs=$(date -d "$nextReportDate" "+%s")
pi@syncbox:~/syncbox $ echo "nextReportSecs=$nextReportSecs"
nextReportSecs=1533219416
pi@syncbox:~/syncbox $ nowSecs=$(date "+%s")
pi@syncbox:~/syncbox $ secsUntilReportDue=$(( $nextReportSecs - $nowSecs ))
pi@syncbox:~/syncbox $ echo "secsUntilReportDue=$secsUntilReportDue"
secsUntilReportDue=86034
