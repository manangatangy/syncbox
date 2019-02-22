package main

import (
	// "bufio"
	"bytes"
	// "encoding/json"
	"flag"
	// "fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	// "fmt"
	// "os/exec"
	// "regexp"
	"reporter/config"
	"reporter/logging"
	"reporter/mail"
	"reporter/settings"
	"reporter/status"
	// "strings"
	"strconv"
	"time"
)

const (
	STATIC_DIR = "/static/" // prefix for urls withing templated html
)

func main() {
	// Only -config=cfgpath and -logfile=logpath are supported.
	logFilePath := flag.String("logfile", "", "path to log file")
	configPath := flag.String("config", "config.json", "path to configuration file")
	flag.Parse()
	if *logFilePath != "" {
		f, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		CheckDie(err)
		defer f.Close()
		log.SetOutput(f)
	}

	log.Println("STARTING ...")
	config.Path(*configPath)
	config.MailerControl = map[int]chan config.ControlMsg{
		config.KEY_HISTORY:  make(chan config.ControlMsg),
		config.KEY_REPORTER: make(chan config.ControlMsg),
		config.KEY_SIMMON:   make(chan config.ControlMsg),
		config.KEY_STATUS:   make(chan config.ControlMsg),
	}

	// Start the mailers
	go mail.PeriodicMailer(config.MailerControl[config.KEY_HISTORY], config.KEY_HISTORY, makeGen(config.KEY_HISTORY))
	go mail.PeriodicMailer(config.MailerControl[config.KEY_REPORTER], config.KEY_REPORTER, makeGen(config.KEY_REPORTER))
	go mail.PeriodicMailer(config.MailerControl[config.KEY_SIMMON], config.KEY_SIMMON, makeGen(config.KEY_SIMMON))
	go mail.WatcherMailer(config.MailerControl[config.KEY_STATUS], config.KEY_STATUS, makeGen(config.KEY_STATUS))

	router := mux.NewRouter().StrictSlash(true)

	// Ref: https://gowebexamples.com/static-files/
	log.Println("serving static assets from: " + config.Get().AssetsRoot)

	staticHandler := http.FileServer(http.Dir(config.Get().AssetsRoot))
	router.PathPrefix(STATIC_DIR).Handler(http.StripPrefix(STATIC_DIR, staticHandler))
	// Test:  curl -s http://localhost:8090/static/test.txt

	router.HandleFunc("/", HomePage)
	router.HandleFunc("/history", status.HistoryPage)
	router.HandleFunc("/settings", settings.SettingsPage)
	router.HandleFunc("/logging", logging.LoggingPage)

	port := config.Get().Port
	log.Printf("listening at: %s:%s\n", getOutboundIP(), port)
	log.Fatal("FATAL: ", http.ListenAndServe(":"+port, router))
}

func CheckDie(e error) {
	if e != nil {
		log.Fatal("FATAL: ", e)
	}
}

type HomePageVariables struct {
	LocalServer bool
	Error       string
	Status      status.BackupStatus
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	// now := time.Now()
	// emailerResult := "SendReport OK"
	// if err := SendReport(); err == nil {
	// 	log.Println(emailerResult)
	// } else {
	// 	emailerResult = err.Error()
	// }
	// Ref: https://gowebexamples.com/templates/

	homePageVars := HomePageVariables{
		LocalServer: true,
	}
	backupStatus, err := status.GetBackupStatus()
	if err != nil {
		homePageVars.Error = err.Error()
	} else {
		homePageVars.Status = *backupStatus
	}
	t, err := template.ParseFiles("home.html")
	if err != nil {
		log.Print("ERROR: HomePage template parsing error: ", err)
	}
	if err = t.Execute(w, homePageVars); err != nil {
		log.Print("ERROR: HomePage template executing error: ", err)
	}
}

// Get preferred outbound ip of this machine
// Ref: https://stackoverflow.com/a/37382208/1402287
func getOutboundIP() net.IP {
	// Try for up to DialTimeout seconds before quitting
	attempts := 0
	for {
		conn, err := net.Dial("udp", "8.8.8.8:80")
		if err == nil {
			defer conn.Close()
			localAddr := conn.LocalAddr().(*net.UDPAddr)
			return localAddr.IP
		}
		attempts = attempts + 1
		if attempts >= config.Get().DialTimeout {
			log.Fatal("FATAL: ", err)
		}
		log.Print("ERROR: failed to net.Dial, trying again in 1 second; ", err)
		time.Sleep(1 * time.Second)
	}
}

// ------------------------------------------

func makeGen(key int) mail.EmailGen {
	keyName := config.KeyName[key]

	switch key {
	case config.KEY_HISTORY:
		return func(body *bytes.Buffer) (subject string, err error) {
			if config.Get().HistoryFileAutoAppend {
				// Optionally create a new BackupStatus and append to
				// the History, which is then emailed in the report.
				backupStatus, err := status.GetBackupStatus()
				if err == nil {
					status.SaveStatusToHistory(*backupStatus)
				}
			}
			subject = keyName + " report"
			historyPageVariables := status.HistoryPageVariables{
				LocalServer: false,
			}
			reportTime := time.Now().Format(config.TIME_FORMAT)
			body.Write([]byte("ReportTime <b>" + reportTime + "</b>\n"))
			if err = status.HistoryFetch(body, &historyPageVariables); err != nil {
				// Email the error message instead of the report
				body.WriteString(err.Error())
				subject = subject + ": FAILED"
			} else {
				// Use the most recent history record in the subject
				if historyPageVariables.Error != "" {
					subject = subject + ": FAILED - " + historyPageVariables.Error
				} else {
					if len := len(historyPageVariables.History); len > 0 {
						backupStatus := historyPageVariables.History[len-1]
						c := strconv.Itoa(int(backupStatus.MissingFiles))
						d := status.ShortenTimeDiff(backupStatus.AcerAge)
						subject = subject + ": MISSING(" + c + ") AGE(" + d + ")"
					}
				}
			}
			return subject, err
		}
	case config.KEY_STATUS:
		return func(body *bytes.Buffer) (subject string, err error) {
			subject = keyName + " report"
			body.Write([]byte("ReportTime <b>" + time.Now().Format(config.TIME_FORMAT) + "</b>\n"))
			backupStatus, err := status.GetBackupStatus()
			if err != nil {
				subject = subject + ": FAILED - " + err.Error()
			} else {
				status.SaveStatusToHistory(*backupStatus)
				c := strconv.Itoa(int(backupStatus.MissingFiles))
				d := status.ShortenTimeDiff(backupStatus.AcerAge)
				subject = subject + ": MISSING(" + c + ") AGE(" + d + ")"
			}
			return subject, err
		}
	default:
		// case mail.KEY_REPORTER:
		// case mail.KEY_SIMMON:
		return func(body *bytes.Buffer) (subject string, err error) {
			subject = keyName + " report"
			loggingPageVars := logging.LoggingPageVariables{
				LocalServer: false,
			}
			reportTime := time.Now().Format(config.TIME_FORMAT_START_OF_HOUR)
			fields := logging.ValidatedFormFields{
				StartDate: reportTime,
				MaxLines:  10000,
			}
			if key == config.KEY_SIMMON {
				fields.LogFilePath = config.Get().SimmonLogFilePath
			} else {
				fields.LogFilePath = config.Get().ReporterLogFilePath
			}
			body.Write([]byte("ReportTime <b>" + reportTime + "</b>\n"))
			if err = logging.LoggingFetch(body, loggingPageVars, fields); err != nil {
				// Email the error message instead of the report
				body.WriteString(err.Error())
				subject = subject + ": FAILED"
			}
			return subject, err
		}
	}
	return nil
}
