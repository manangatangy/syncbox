package logging

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	// "io"
	"log"
	"net/http"
	"os"
	"reporter/config"
	"strconv"
	"strings"
	"unicode"
)

type Setting struct {
	Id          string
	Name        string
	Type        string
	Value       string
	Readonly    bool
	Errored     string
	Description string
	// The Validator takes a new value (as the entered string). If the validation
	// fails, then an error is returned. This field is only used when Readonly = false
	Validator func(newValue string, fields *ValidatedFormFields) error
}

type LoggingPageVariables struct {
	LocalServer bool
	Settings    []Setting
	Message     string // Any error/success message
	LogLines    []string
}

type ValidatedFormFields struct {
	LogFilePath string // Set via radiobutton (in the Form)
	StartDate   string // Empty means from the start (in the Form)
	MaxLines    int    // zero means no maximum (in the Form)
}

// Use the specified values to create settings used for the form display
func getSettings(logType string, startDate string, maxLines string) []Setting {
	var settings []Setting
	settings = append(settings, Setting{
		Id: "LogType", Name: "Log Type", Type: "text",
		Value: logType, Description: "Either SIMMON or REPORTER",
		Validator: func(newValue string, fields *ValidatedFormFields) error {
			switch newValue {
			case "SIMMON":
				fields.LogFilePath = config.Get().SimmonLogFilePath
			case "REPORTER":
				fields.LogFilePath = config.Get().ReporterLogFilePath
			default:
				return errors.New("Only SIMMON or REPORTER allowed")
			}
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "StartDate", Name: "Start Date-Time", Type: "text",
		Value: startDate, Description: "Show records from this date/time on",
		Validator: func(newValue string, fields *ValidatedFormFields) error {
			// TODO test against valid
			fields.StartDate = newValue
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "MaxLines", Name: "Max Line Count", Type: "number",
		Value: maxLines, Description: "Maximum number of lines to show (0 means show all lines)",
		Validator: func(newValue string, fields *ValidatedFormFields) error {
			maxLines, err := strconv.Atoi(newValue)
			if err != nil {
				if maxLines < 0 || maxLines > 1000 {
					err = errors.New("out of range (0,1000)")
				}
			}
			fields.MaxLines = maxLines
			return err
		},
	})
	// settings = append(settings, makeSetting(
	// 	"Port", "Server Port", "text",
	// 	c.Port, "Listening port",
	// 	nil,		// Readonly
	// ))
	return settings
}

func LoggingPage(w http.ResponseWriter, r *http.Request) {
	loggingPageVars := LoggingPageVariables{
		LocalServer: true,
	}
	fmt.Printf("LoggingPage method ===> %v\n", r.Method)
	if r.Method != http.MethodPost {
		// TODO - determine start date, as say yesterday using format "2019/02/11 11:11:42"
		// startDate is a string like
		// 2019-02-14 19:42:14+11:00	written to simmon.log
		// 2019/02/11 21:34:46			written to reporter.log
		// 2006-01-02 15:04:00			written to history.json
		// Due to the difficulty in changing the reporter.log datestring format, we have to support both.

		// Set defaults for form fields
		loggingPageVars.Settings = getSettings("REPORTER", "2018/02/11 21:34:46", "100")
	} else {
		r.ParseForm()
		if r.Form.Get("retrieve") == "yes" {
			loggingPageVars.Settings = getSettings(r.Form.Get("LogType"), r.Form.Get("StartDate"), r.Form.Get("MaxLines"))
			success := true
			fields := ValidatedFormFields{}
			for s := 0; s < len(loggingPageVars.Settings); s++ {
				setting := &loggingPageVars.Settings[s]
				if !setting.Readonly {
					if err := setting.Validator(setting.Value, &fields); err != nil {
						setting.Description = err.Error()
						setting.Errored = "errored"
						success = false
					}
				}
				fmt.Printf("setting after validating ==> %v\n", setting)
			}
			// fmt.Printf("config after validating ==> %v\n", config)
			if success {
				// If all the settings are valid, then fetch the records from the specified log file.
				lines, err := readLog(fields.LogFilePath, fields.StartDate, fields.MaxLines)
				if err == nil {
					loggingPageVars.LogLines = *lines
					loggingPageVars.Message = strconv.Itoa(len(*lines)) + " log lines retrieved"
				} else {
					loggingPageVars.Message = err.Error()
				}
			}
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

// startDate is a string like
// 2019-02-14 19:42:14+11:00	written to simmon.log
// 2019/02/11 21:34:46			written to reporter.log
// 2006-01-02 15:04:00			written to history.json
// Due to the difficulty in changing the reporter.log datestring format, we have to support both.
// Step through the file from the start, looking for the first line that is prefixed with the
// same (or later) string and read that and all subsequent lines, up to a specified max lines
// (or all lines, if maxLines is 0).
func readLog(logFilePath, startDate string, maxLines int) (*[]string, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Printf("ERROR: opening logFile for read %s: %s\n", logFilePath, err)
		return nil, err
	}
	defer file.Close()
	var lines []string
	count := 0
	scanner := bufio.NewScanner(file)
	for adding, finished := false, false; !finished && scanner.Scan(); {
		line := scanner.Text()
		if !adding && matches(startDate, line) {
			adding = true
		}
		if adding {
			lines = append(lines, line)
			if count = count + 1; maxLines > 0 && count >= maxLines {
				finished = true
			}
		}
	}
	fmt.Printf("===> readLog added %d lines\n", count)
	return &lines, nil
}

// Return true if the line should passes the matching test, and is the
// first of the selected /extracted lines.  The matching test is only
// passed if ;
// 1. the prefix is ampty string, or
// 2. the prefix is an exact prefix of the line, or
// 3. both strings are time-stamps and the prefix-time occurs before/equal
// to line-time.  This test is processed according to;
// 3.1. The prefix must contain some digits after all non-digit chars are
// converted to spaces.
// 3.2. extract from the line a string of up to the same length as prefix.
// 3.3. take the prefix and for all the non-numeric characters, set the
// character to space and in the same position in the extract, also
// to space.
// 3.4. perform an string comparison

func matches(prefix, line string) bool {
	s := len(prefix)
	if s == 0 { // Empty prefix means start from the first line
		return true
	}
	if s > len(line) {
		s = len(line) // Shorten the length of comparison
	}
	prefixT := prefix[:s]
	lineT := line[:s]
	if strings.Compare(string(prefixT), string(lineT)) == 0 {
		return true
	}
	p := []rune(prefixT)
	l := []rune(lineT)
	fmt.Printf("==> subs '%s' length %d or %d\n", prefixT, s, len(prefixT))
	fmt.Printf("==> subs '%s' length %d or %d\n", lineT, s, len(lineT))
	fmt.Printf("==> rune '%v' length %d\n", p, len(p))
	fmt.Printf("==> rune '%v' length %d\n", l, len(l))
	var assembleP []rune
	var assembleL []rune
	for i := 0; i < s; i++ {
		fmt.Printf("==> char at %d is %v and %v\n", i, p[i], l[i])
		if unicode.IsDigit(p[i]) {
			if unicode.IsDigit(l[i]) {
				assembleP = append(assembleP, p[i])
				assembleL = append(assembleL, l[i])
			} else {
				return false
			}
		} else {
			assembleP = append(assembleP, rune(' '))
			assembleL = append(assembleL, rune(' '))
		}
	}
	strP := string(assembleP)
	strL := string(assembleL)
	result := strings.Compare(strP, strL) <= 0
	fmt.Printf("==> matches '%s' and '%s' ==> %v\n", strP, strL, result)
	return result
}
