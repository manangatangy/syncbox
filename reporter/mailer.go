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
package main

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
	// "strings"
	"fmt"
	"time"
)

func say(s string) {
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Println(s)
	}
}

// https://tour.golang.org/concurrency/3
// https://github.com/andlabs/wakeup     MainWIndow app FTW!
// https://mmcgrana.github.io/2012/09/go-by-example-timers-and-tickers.html
// https://gobyexample.com/channel-synchronization    Channel sync

func testMain() {
	go say("world")
	say("hello")
}

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
	historyPageVariables := HistoryPageVariables{
		LocalServer: false,
	}
	if err := HistoryFetch(&body, historyPageVariables); err != nil {
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
