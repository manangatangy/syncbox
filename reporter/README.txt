This directory contains the reporter for the syncbox system.
It is a subproject of the syncbox project.

The reporter subproject consists of;
- standalone executable (source: reporter.go), which
  - write to stdout log, error info (this may be redirected by runReporter.sh)
  - has a single command line arg specifying the config file path
  - periodically fetches some info from the local syncthing instance and records it
  - serves (http://localhost:8090) some status, report, and config html pages
  - periodically emails some reports
- daemonising script runReporter.sh, which
  - starts, stops, status' the reporter exe.  This script should be run at startup
    from /etc/rc.local; as
    ~pi/syncbox/runSimmon.sh start
    ~pi/syncbox/reporter/runReporter.sh start
- various templates (*.html)
- config file that specifies;
 - port for serving html [8090]
 - path to root of served documents (may be absolute or relative to wd) [./]
 - path to static assets (may be absolute or relative to wd) [./assets]
 - syncthing check period [1 day]
 - report email target [me@gmail.com]
 - report email period [1 day]
- a local installation of purecss.io, by default this is expected to be installed at
  ./static/pure-release-1.0.0/ (in the working directory of reporter) but it is configurable

Project structure
reporter/
    home.html
    local.css
    config.json         (default runtime config file)
    reporter.go         (entry point)
    reporter.log        (default runtime log file)
    config/
        config.go
    status/
        history.html
        status.go
        history.go
    settings/
        settings.html
        settings.go
    mail/
        mailer.go
        watcher.go
    logging/
        logging.html


https://gowebexamples.com/templates/
