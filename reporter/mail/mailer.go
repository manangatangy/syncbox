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
	// "os"
	// "os/exec"
	// "regexp"
	"errors"
	// "fmt"
	"reporter/config"
	"reporter/settings"
	"reporter/status"
	"time"
)

const (
	KEY_HISTORY  = 1
	KEY_REPORTER = 2
	KEY_SIMMON   = 3
)

/*
	f := func(body *bytes.Buffer) (subject string, err error) {

		return "", nil
	}


*/

type EmailGen func(body *bytes.Buffer) (subject string, err error)

// Waits until the next scheduled email, then sends it, and schedules the next one.
// The channel is used to alert mailer that the config has changed and to re-load it.
// This function will update the config in order to re-schedule the email.
// make(chan struct{})
// bytes.Buffer
// Ref: https://stackoverflow.com/questions/17797754/ticker-stop-behaviour-in-golang
func Mailer(readConfig <-chan struct{}, key int, gen EmailGen) {
	for {
		aec := getEmailConfig(key)
		if waitDuration, err := getWaitDuration(keyName(key), aec); err != nil {
			// The current aec.AutoEmailNext can't be used; advance to the next time and save it.
			_, aec.AutoEmailNext = settings.CalculateNextTime(time.Now(), aec.AutoEmailCount, aec.AutoEmailPeriod)
			log.Printf("mailer(%s): Calculated new nextTime after %d %s ==> %s\n",
				keyName(key), aec.AutoEmailCount, aec.AutoEmailPeriod, aec.AutoEmailNext)
			setEmailConfig(key, aec)
		} else {
			log.Printf("mailer(%s): waiting...\n", keyName(key))
			if wait(readConfig, waitDuration) {
				// Time reached; email the report
				log.Printf("mailer(%s): emailing\n", keyName(key))
				var body bytes.Buffer
				subject, _ := gen(&body)

				m := gomail.NewMessage()
				// TODO alter subkect to be summary of missing files, and since

				/*
					EmailFrom     string	"syncboxmichele@gmail.com"
					EmailTo       string	"david.x.weiss@gmail.com"
					EmailUserName string	"syncboxmichele"
					EmailPassword string	"kAk&dee14"
					EmailHost     string    "smtp.gmail.com"
				*/
				c := config.Get()
				m.SetHeader("From", c.EmailFrom)
				m.SetHeader("To", c.EmailTo)
				// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
				m.SetHeader("Subject", subject)
				m.SetBody("text/html", body.String())
				// m.Attach("/home/Alex/lolcat.jpg")
				d := gomail.NewDialer(c.EmailHost, 465, c.EmailUserName, c.EmailPassword)
				if err := d.DialAndSend(m); err != nil {
					log.Print("ERROR: dialer.DialAndSend error: ", err)
					panic(err)
				}
			}
			// On the next iteration re-read the config, due to a config change
		}
	}
}

func getEmailConfig(key int) (emailConfig config.AutoEmailConfig) {
	log.Printf("mailer(%s): Reading email config\n", keyName(key))
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
	log.Printf("mailer(%s): Saved email config\n", keyName(key))
}

func keyName(key int) (keyName string) {
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
// Also logs the error here.
func getWaitDuration(key string, aec config.AutoEmailConfig) (time.Duration, error) {
	if nextTime, err := time.Parse(config.TIME_FORMAT, aec.AutoEmailNext); err != nil {
		log.Printf("ERROR: mailer(%s) parsing '%s' error: %s\n", key, aec.AutoEmailNext, err.Error())
		return 0, err
	} else {
		now := time.Now()
		if nextTime.Before(now) {
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

// Ref: https://github.com/golang/go/issues/27169
func wait(readConfig <-chan struct{}, waitDuration time.Duration) (timedOut bool) {
	timer := time.NewTimer(waitDuration)
	defer timer.Stop()
	select {
	case <-readConfig:
		return false
	case <-timer.C:
		return true
	}
}

// https://tour.golang.org/concurrency/3
// https://github.com/andlabs/wakeup     MainWIndow app FTW!
// https://mmcgrana.github.io/2012/09/go-by-example-timers-and-tickers.html
// https://gobyexample.com/channel-synchronization    Channel sync

// Mailer returns a chan int to which any value should be sent,
// to trigger then emailing, based on the current history.
// func Mailer(updateInterval time.Duration) chan<- int {
// 	commands := make(chan int)
// 	urlStatus := make(map[string]string)
// 	ticker := time.NewTicker(updateInterval)
// 	go func() {
// 		for {
// 			select {
// 			case c := <-commands:
// 				SendReport()
// 			}
// 		}
// 	}()
// 	return commands
// }

func SendReport() error {

	var body bytes.Buffer
	body.Write([]byte("Hello <b>Bob</b> and <i>Cora</i>!\n"))
	historyPageVariables := status.HistoryPageVariables{
		LocalServer: false,
	}
	if err := status.HistoryFetch(&body, historyPageVariables); err != nil {
		// Email the error message instead of the report
		body.WriteString(err.Error())
	}
	m := gomail.NewMessage()
	// todo get  to/from from configuration
	// todo akter subkect to be summary of missing files, and since
	m.SetHeader("From", "syncboxmichele@gmail.com")
	m.SetHeader("To", "david.x.weiss@gmail.com")
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Hello!")
	m.SetBody("text/html", body.String())
	// m.Attach("/home/Alex/lolcat.jpg")
	d := gomail.NewDialer("smtp.gmail.com", 465, "syncboxmichele", "kAk&dee14")
	if err := d.DialAndSend(m); err != nil {
		log.Print("ERROR: dialer.DialAndSend error: ", err)
		panic(err)
	}
	return nil
}
