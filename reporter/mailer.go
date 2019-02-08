// ref: https://stackoverflow.com/questions/2591755/how-to-send-html-email-using-linux-command-line

// Ref: https://unix.stackexchange.com/questions/15405/how-do-i-send-html-email-using-linux-mail-command is a
// good rundown on the various problems and options with unix mail clients.
// Ref: https://24ways.org/2009/rock-solid-html-emails might be worth a read
// Ref: https://github.com/go-gomail/gomail and https://godoc.org/gopkg.in/gomail.v2 seems to be the go hah
// Ref: https://gist.github.com/chrisgillis/10888032  has useful info

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
// 

func testMain() {
	go say("world")
	say("hello")
}

func SendReport() error {
	var body bytes.Buffer
	body.Write([]byte("Hello <b>Bob</b> and <i>Cora</i>!\n"))
	historyPageVariables := HistoryPageVariables{
		LocalFonts: false,
		// Use prefix for url access from within the target's mail reader
		PureCssBaseURL: "https://unpkg.com/purecss@1.0.0/build/",
	}
	if err := FetchHistory(&body, historyPageVariables); err != nil {
		// Email the error message instead of the report
		body.WriteString(err.Error())
	}
	m := gomail.NewMessage()
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
