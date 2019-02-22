package mail

// ref: https://stackoverflow.com/questions/2591755/how-to-send-html-email-using-linux-command-line

// Ref: https://unix.stackexchange.com/questions/15405/how-do-i-send-html-email-using-linux-mail-command is a
// good rundown on the various problems and options with unix mail clients.
// Ref: https://24ways.org/2009/rock-solid-html-emails might be worth a read
// Ref: https://github.com/go-gomail/gomail and https://godoc.org/gopkg.in/gomail.v2 seems to be the go hah
// Ref: https://gist.github.com/chrisgillis/10888032  has useful info

// Watch the specified AcerFilePath (with a poll interval) for changes.
// When the file changes, then retrieve a current BackupStatus from
// ???, create a HistoryRecord from the two, and send it using the Emailer
// and also add to the HistoryArchive.

// Refs: https://golang.org/pkg/
import (
	"bytes"
	"gopkg.in/gomail.v2"
	// "bufio"
	// "encoding/json"
	// "github.com/gorilla/mux"
	// "html/template"
	// "io/ioutil"
	"log"
	// "net"
	// "net/http"
	"os"
	// "os/exec"
	// "regexp"
	"errors"
	"fmt"
	"reporter/config"
	"reporter/settings"
	"reporter/status"
	"time"
	// "github.com/go-fsnotify/fsnotify"
)

const (
	KEY_HISTORY  = 1
	KEY_REPORTER = 2
	KEY_SIMMON   = 3
	KEY_STATUS   = 4
)

var KeyName = map[int]string{
	KEY_HISTORY:  "HISTORY",
	KEY_REPORTER: "REPORTER",
	KEY_SIMMON:   "SIMMON",
	KEY_STATUS:   "STATUS",
}

type EmailGen func(body *bytes.Buffer) (subject string, err error)

type ControlMsg int

const (
	CONTROL_CONFIG_CHANGE   = 1
	CONTROL_EMAIL_IMMEDIATE = 2
	CONTROL_TIMER_EXPIRED   = 3
)

var MsgName = map[ControlMsg]string{
	CONTROL_CONFIG_CHANGE:   "CONTROL_CONFIG_CHANGE",
	CONTROL_EMAIL_IMMEDIATE: "CONTROL_EMAIL_IMMEDIATE",
	CONTROL_TIMER_EXPIRED:   "CONTROL_TIMER_EXPIRED",
}

// Waits until the next scheduled email, then sends it, and schedules the next one.
// The channel is used to alert mailer that the config has changed and to re-load it.
// This function will update the config in order to re-schedule the email.
// Ref: https://stackoverflow.com/questions/17797754/ticker-stop-behaviour-in-golang
func PeriodicMailer(control <-chan ControlMsg, key int, gen EmailGen) {
	log.Printf("mailer(%s): starting\n", KeyName[key])
	for {
		aec := getEmailConfig(key)
		if waitDuration, err := getWaitDuration(KeyName[key], aec); err != nil {
			// The current aec.AutoEmailNext can't be used; advance to the next time and save it.
			_, aec.AutoEmailNext = settings.CalculateNextTime(time.Now(), aec.AutoEmailCount, aec.AutoEmailPeriod)
			log.Printf("mailer(%s): Calculated new nextTime after %d %s ==> %s\n",
				KeyName[key], aec.AutoEmailCount, aec.AutoEmailPeriod, aec.AutoEmailNext)
			setEmailConfig(key, aec)
		} else {
			var msg ControlMsg
			if waitDuration == 0 {
				// Zero duration means periodic email is disabled, so just wait for control msg.
				log.Printf("mailer(%s): waiting indefinitely...\n", KeyName[key])
				msg = waitIndefinite(control)
			} else {
				td := status.ShortenTimeDiff(waitDuration.String())
				log.Printf("mailer(%s): waiting for timeout %s ...\n", KeyName[key], td)
				msg = waitTimed(control, waitDuration)
			}
			log.Printf("mailer(%s): wait completed, msg: %s\n", KeyName[key], MsgName[msg])
			if msg != CONTROL_CONFIG_CHANGE {
				log.Printf("mailer(%s): mailing...\n", KeyName[key])
				mail(key, gen) // Time reached; email the report
			}
		}
	}
}

// HMMMM Watches a file and if changed, then sends an email.

// Simply waits for a control message, either that the acerfile has changed or to reload config.
func WatcherMailer(control <-chan ControlMsg, key int, gen EmailGen) {
	log.Printf("Wmailer(%s): starting\n", KeyName[key])
	filePath := config.Get().AcerFilePath
	currentModTime, err := getModTime(filePath)
	// fmt.Printf("TEMP ===> mailer(%s): currentModTime %v\n", KeyName[key], currentModTime)
	if err != nil {
		// Ignore error; modTime will be empty but still comparable to polled value.
		log.Printf("ERROR: mailer(%s): getModTime(%s) error: %v\n", KeyName[key], filePath, err)
	}
	for {
		c := config.Get()
		var msg ControlMsg
		if !c.EnableAcerFileWatch {
			// fmt.Printf("TEMP ===> mailer(%s): waiting indefinitely...\n", KeyName[key])
			msg = waitIndefinite(control)
		} else {
			secs := c.AcerFileWatchPeriod
			if secs < 10 {
				secs = 10 // Minimum polling period
			}
			waitDuration := time.Duration(secs) * time.Second
			msg = waitTimed(control, waitDuration)

			// log.Printf("TEMP ===> mailer(%s): wait completed, msg: %s\n", KeyName[key], MsgName[msg])

			if msg == CONTROL_CONFIG_CHANGE {
				log.Printf("mailer(%s): config change occurred\n", KeyName[key])
			} else {
				// Check for a file change
				modTime, err := getModTime(c.AcerFilePath)
				if err != nil {
					log.Printf("ERROR: mailer(%s): os.Stat(%s) error: %v\n", KeyName[key], c.AcerFilePath, err)
				} else {
					if !modTime.Equal(currentModTime) {
						mail(key, gen)
						currentModTime = modTime
					}
				}
			}
		}
	}
}

func getModTime(filePath string) (time.Time, error) {
	fi, err := os.Stat(filePath)
	if err == nil {
		// fmt.Printf("TEMP ===> getModTime %s %v\n", filePath, fi.ModTime())
		return fi.ModTime(), nil
	} else {
		// fmt.Printf("TEMP ===> getModTime %s %v\n", filePath, err.Error())
		return time.Time{}, err
	}
}

func mail(key int, gen EmailGen) {
	var body bytes.Buffer
	subject, _ := gen(&body)
	m := gomail.NewMessage()
	c := config.Get()
	m.SetHeader("From", c.EmailFrom)
	m.SetHeader("To", c.EmailTo)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body.String())
	// m.Attach("/home/Alex/lolcat.jpg")
	d := gomail.NewDialer(c.EmailHost, 465, c.EmailUserName, c.EmailPassword)
	if err := d.DialAndSend(m); err != nil {
		log.Printf("ERROR: mailer(%s): dialer.DialAndSend error: %v\n", KeyName[key], err)
	} else {
		log.Printf("mailer(%s): emailed OK\n", KeyName[key])
	}
}

func getEmailConfig(key int) (emailConfig config.AutoEmailConfig) {
	log.Printf("mailer(%s): Reading email config\n", KeyName[key])
	c := config.Get()
	switch key {
	case KEY_HISTORY:
		return c.HistoryLogAutoEmail
	case KEY_REPORTER:
		return c.ReporterLogAutoEmail
	case KEY_SIMMON:
		return c.SimmonLogAutoEmail
	default:
		log.Fatal("FATAL: getEmailConfig bad key:" + string(key))
	}
	return config.AutoEmailConfig{}
}

func setEmailConfig(key int, emailConfig config.AutoEmailConfig) {
	c := config.Get()
	switch key {
	case KEY_HISTORY:
		c.HistoryLogAutoEmail = emailConfig
	case KEY_REPORTER:
		c.ReporterLogAutoEmail = emailConfig
	case KEY_SIMMON:
		c.SimmonLogAutoEmail = emailConfig
	default:
		log.Fatal("FATAL: setEmailConfig bad key:" + string(key))
	}
	config.Set(c)
	log.Printf("mailer(%s): Saved email config\n", KeyName[key])
}

func xxxKeyName(key int) (keyName string) {
	switch key {
	case KEY_HISTORY:
		keyName = "HISTORY"
	case KEY_REPORTER:
		keyName = "REPORTER"
	case KEY_SIMMON:
		keyName = "SIMMON"
	}
	return
}

// Use the config to determine the wait duration, or an error.
// No error and zero duration means mailing is disabled.
// Also logs the error here.
func getWaitDuration(key string, aec config.AutoEmailConfig) (time.Duration, error) {
	if !aec.AutoEmailEnable {
		log.Printf("mailer(%s): mailer disabled\n", key)
		return 0, nil
	}
	loc, _ := time.LoadLocation("Local")
	if nextTime, err := time.ParseInLocation(config.TIME_FORMAT, aec.AutoEmailNext, loc); err != nil {
		log.Printf("ERROR: mailer(%s) parsing '%s' error: %s\n", key, aec.AutoEmailNext, err.Error())
		return 0, err
	} else {
		now := time.Now()
		if !nextTime.After(now) {
			// Consider (nextTime == now) to be too late also.
			log.Printf("mailer(%s): Config next email time %s has expired\n", key, aec.AutoEmailNext)
			return 0, errors.New("next email time has expired")
		} else {
			waitDuration := nextTime.Sub(now)
			minWaitDuration := time.Duration(10) * time.Second
			if waitDuration < minWaitDuration {
				log.Printf("mailer(%s): Not enough time remaining; skipping\n", key)
				return 0, errors.New("not enough time remaining")
			} else {
				return waitDuration, nil
			}
		}
	}
}

// Wait for just a command on the control channel.
// If a false message was received, this means to reload config.
func waitIndefinite(control <-chan ControlMsg) (msg ControlMsg) {
	select {
	case msg = <-control:
		return msg
	}
}

// Wait for the specified duration, and on the control channel.
// Ref: https://github.com/golang/go/issues/27169
func waitTimed(control <-chan ControlMsg, waitDuration time.Duration) (msg ControlMsg) {
	timer := time.NewTimer(waitDuration)
	defer timer.Stop()
	select {
	case msg = <-control:
		return msg
	case <-timer.C:
		return CONTROL_TIMER_EXPIRED
	}
}

// https://tour.golang.org/concurrency/3
// https://github.com/andlabs/wakeup     MainWIndow app FTW!
// https://mmcgrana.github.io/2012/09/go-by-example-timers-and-tickers.html
// https://gobyexample.com/channel-synchronization    Channel sync
