package logging

import (
	// "errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	// "reporter/config"
	"strconv"
)

type LoggingPageVariables struct {
	LocalServer bool
	Message     string
	StartDate   string // Also returned in the Form
	Log         []string
}

func LoggingPage(w http.ResponseWriter, r *http.Request) {
	loggingPageVars := LoggingPageVariables{
		LocalServer: true,
	}
	fmt.Printf("LoggingPage method ===> %v\n", r.Method)
	if r.Method != http.MethodPost {
		// TODO - determine start date, as say yesterday using format "2019/02/11 11:11:42"
		loggingPageVars.StartDate = "default-start-date"
	} else {
		r.ParseForm()
		if r.Form.Get("retrieve") == "yes" {
			startDate := r.Form.Get("StartDateId")
			// TODO
			// Validate this start date and fetch the following records
			loggingPageVars.StartDate = startDate
			// TODO
			// Fetch records from the specified log file
			for i := 0; i < 300; i++ {
				loggingPageVars.Log = append(loggingPageVars.Log, "line "+strconv.Itoa(i))
			}
			// set loggingPageVars.Message to any error, or perhaps the number of records retrieved
			loggingPageVars.Message = "300 records retrieved"
		}
	}

	t, err := template.ParseFiles("logging/logging.html")
	if err != nil {
		log.Print("ERROR: LoggingPage template parsing error: ", err)
	}
	err = t.Execute(w, loggingPageVars)
	if err != nil {
		log.Print("ERROR: LoggingPage template executing error: ", err)
	}

}
