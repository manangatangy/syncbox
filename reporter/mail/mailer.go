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

type EmailGen func(body *bytes.Buffer) (subject string, err error)

// Waits until the next scheduled email, then sends it, and schedules the next one.
// The channel is used to alert mailer that the config has changed and to re-load it.
// This function will update the config in order to re-schedule the email.
// Ref: https://stackoverflow.com/questions/17797754/ticker-stop-behaviour-in-golang
func PeriodicMailer(control <-chan config.ControlMsg, key int, gen EmailGen) {
	tag := fmt.Sprintf("PeriodicMailer(%s)", config.KeyName[key])
	log.Printf("%s: starting\n", tag)
	for {
		aec := getEmailConfig(key)
		if waitDuration, err := getWaitDuration(tag, aec); err != nil {
			// The current aec.AutoEmailNext can't be used; advance to the next time and save it.
			_, aec.AutoEmailNext = settings.CalculateNextTime(time.Now(), aec.AutoEmailCount, aec.AutoEmailPeriod)
			log.Printf("%s: Calculated new nextTime after %d %s ==> %s\n",
				tag, aec.AutoEmailCount, aec.AutoEmailPeriod, aec.AutoEmailNext)
			setEmailConfig(key, aec)
		} else {
			var msg config.ControlMsg
			if waitDuration == 0 {
				// Zero duration means periodic email is disabled, so just wait for control msg.
				log.Printf("%s: waiting indefinitely...\n", tag)
				msg = waitIndefinite(control)
			} else {
				td := status.ShortenTimeDiff(waitDuration.String())
				log.Printf("%s: waiting for timeout %s ...\n", tag, td)
				msg = waitTimed(control, waitDuration)
			}
			log.Printf("%s: wait completed, msg: %s\n", tag, config.MsgName[msg])
			if msg != config.CONTROL_CONFIG_CHANGE {
				log.Printf("%s: mailing...\n", tag)
				mail(tag, gen) // Time reached; email the report
			}
		}
	}
}

// Simply waits for a control message, either that the acerfile has changed or to reload config.
func WatcherMailer(control <-chan config.ControlMsg, key int, gen EmailGen) {
	tag := fmt.Sprintf("WatcherMailer(%s)", config.KeyName[key])
	log.Printf("%s: starting\n", tag)
	filePath := config.Get().AcerFilePath
	currentModTime, _ := getModTime(tag, filePath)
	// Ignore error; modTime will be empty but still comparable to polled value.
	for {
		c := config.Get()
		var msg config.ControlMsg
		if !c.EnableAcerFileWatch {
			log.Printf("%s: waiting indefinitely...\n", tag)
			msg = waitIndefinite(control)
		} else {
			secs := c.AcerFileWatchPeriod
			if secs < 10 {
				secs = 10 // Minimum polling period
			}
			msg = waitTimed(control, time.Duration(secs) * time.Second)
		}
		if msg == config.CONTROL_CONFIG_CHANGE {
			log.Printf("%s: config change occurred\n", tag)
		} else {
			// Timeout: check for a file change
			modTime, err := getModTime(tag, c.AcerFilePath)
			if err == nil && !modTime.Equal(currentModTime) {
				mail(tag, gen)
				currentModTime = modTime
			}
		}
	}
}

func getModTime(tag, filePath string) (time.Time, error) {
	fi, err := os.Stat(filePath)
	if err == nil {
		return fi.ModTime(), nil
	} else {
		log.Printf("ERROR: %s: getModTime(%s) error: %v\n", tag, filePath, err)
		return time.Time{}, err
	}
}

func mail(tag string, gen EmailGen) {
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
		log.Printf("ERROR: %s: dialer.DialAndSend error: %v\n", tag, err)
	} else {
		log.Printf("%s: emailed OK\n", tag)
	}
}

func getEmailConfig(key int) (emailConfig config.AutoEmailConfig) {
	log.Printf("getEmailConfig(%s): Reading email config\n", config.KeyName[key])
	c := config.Get()
	switch key {
	case config.KEY_HISTORY:
		return c.HistoryLogAutoEmail
	case config.KEY_REPORTER:
		return c.ReporterLogAutoEmail
	case config.KEY_SIMMON:
		return c.SimmonLogAutoEmail
	default:
		log.Fatal("FATAL: getEmailConfig bad key:" + string(key))
	}
	return config.AutoEmailConfig{}
}

func setEmailConfig(key int, emailConfig config.AutoEmailConfig) {
	c := config.Get()
	switch key {
	case config.KEY_HISTORY:
		c.HistoryLogAutoEmail = emailConfig
	case config.KEY_REPORTER:
		c.ReporterLogAutoEmail = emailConfig
	case config.KEY_SIMMON:
		c.SimmonLogAutoEmail = emailConfig
	default:
		log.Fatal("FATAL: setEmailConfig bad key:" + string(key))
	}
	config.Set(c)
	log.Printf("setEmailConfig(%s): Saved email config\n", config.KeyName[key])
}

// Use the config to determine the wait duration, or an error.
// No error and zero duration means mailing is disabled.
// Also logs the error here.
func getWaitDuration(tag string, aec config.AutoEmailConfig) (time.Duration, error) {
	if !aec.AutoEmailEnable {
		log.Printf("%s: mailer disabled\n", tag)
		return 0, nil
	}
	loc, _ := time.LoadLocation("Local")
	if nextTime, err := time.ParseInLocation(config.TIME_FORMAT, aec.AutoEmailNext, loc); err != nil {
		log.Printf("ERROR: %s parsing '%s' error: %s\n", tag, aec.AutoEmailNext, err.Error())
		return 0, err
	} else {
		now := time.Now()
		if !nextTime.After(now) {
			// Consider (nextTime == now) to be too late also.
			log.Printf("%s: Config next email time %s has expired\n", tag, aec.AutoEmailNext)
			return 0, errors.New("next email time has expired")
		} else {
			waitDuration := nextTime.Sub(now)
			minWaitDuration := time.Duration(10) * time.Second
			if waitDuration < minWaitDuration {
				log.Printf("%s: Not enough time remaining; skipping\n", tag)
				return 0, errors.New("not enough time remaining")
			} else {
				return waitDuration, nil
			}
		}
	}
}

// Wait for just a command on the control channel.
// If a false message was received, this means to reload config.
func waitIndefinite(control <-chan config.ControlMsg) (msg config.ControlMsg) {
	select {
	case msg = <-control:
		return msg
	}
}

// Wait for the specified duration, and on the control channel.
// Ref: https://github.com/golang/go/issues/27169
func waitTimed(control <-chan config.ControlMsg, waitDuration time.Duration) (msg config.ControlMsg) {
	timer := time.NewTimer(waitDuration)
	defer timer.Stop()
	select {
	case msg = <-control:
		return msg
	case <-timer.C:
		return config.CONTROL_TIMER_EXPIRED
	}
}

// https://tour.golang.org/concurrency/3
// https://github.com/andlabs/wakeup     MainWIndow app FTW!
// https://mmcgrana.github.io/2012/09/go-by-example-timers-and-tickers.html
// https://gobyexample.com/channel-synchronization    Channel sync
