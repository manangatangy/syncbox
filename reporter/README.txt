This directory contains the reporter for the syncbox system.
It is a subproject of the syncbox project.

The reporter subproject consists of;
- standalone executable (source: reporter.go), which
  - logs some info from the local syncthing instance
  - serves (http://localhost:8090) some status, report, and config html pages
  - periodically emails some reports
- daemonising script runReporter.sh, which
  - starts, stops, status' the reporter exe.  This script should be run at startup
    from /etc/rc.local
- various templates (*.html)
- a local installation of purecss.io, by default this is expected to be installed at
  ./static (in the working directory of reporter) but it is configurable


