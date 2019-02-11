package main

import (
	"bufio"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
)

type HistoryPageVariables struct {
	Error          string
	LocalFonts     bool
	PureCssBaseURL string
	Records        []HistoryRecord
}

// Serve the history records as a page for local/connected access
func HistoryPage(w http.ResponseWriter, r *http.Request) {
	historyPageVariables := HistoryPageVariables{
		LocalFonts:     true,
		PureCssBaseURL: "static/pure-release-1.0.0/",
	}
	HistoryFetch(w, historyPageVariables)

	// temp
	record := HistoryRecord{
		ServerTime:       "2019-03-22 20:00:10",
		OutOfSyncFiles:   33,
		OutOfSyncByes:    44,
		BackedUpFiles:    11111,
		BackedUpByes:     88888,
		WorkstationFiles: 88888,
		WorkstationByes:  11111,
		WorkstationTime:  "2019-03-22 20:00:10",
		TimeDifference:   "6 hours",
	}
	HistorySaveRecord(record)
}

// Fetches all history records from the history file,
// and writes the expanded html string to the parm.
// Errors are logged here.
// PureCssBaseURL "https://unpkg.com/purecss@1.0.0/build/"
// PureCssBaseURL "static/pure-release-1.0.0/"
// Nesting: https://stackoverflow.com/questions/11467731/is-it-possible-to-have-nested-templates-in-go-using-the-standard-library-googl
func HistoryFetch(w io.Writer, historyPageVariables HistoryPageVariables) error {
	records, err1 := HistoryReadAllRecords()
	historyPageVariables.Records = records
	historyPageVariables.Error = err1
	t, err2 := template.ParseFiles("history.html")
	if err2 != nil {
		log.Print("ERROR: template parsing error: ", err2)
	}
	err3 := t.Execute(w, historyPageVariables)
	if err3 != nil {
		log.Print("ERROR: template executing error: ", err3)
	}
	return nil
}

type HistoryRecord struct {
	ServerTime       string
	OutOfSyncFiles   int32
	OutOfSyncByes    int64
	BackedUpFiles    int32
	BackedUpByes     int64
	WorkstationFiles int32
	WorkstationByes  int64
	WorkstationTime  string
	TimeDifference   string // How long after ServerTime is the WorkstationTime expressed as ddd:hh:mm
}

func HistoryReadAllRecords() ([]HistoryRecord, string) {
	file, err := os.Open(configuration.HistoryFile)
	if err != nil {
		log.Printf("ERROR: opening for read %s: %s\n", configuration.HistoryFile, err)
		return nil, err.Error()
	}
	defer file.Close()
	var records []HistoryRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		historyRecord := HistoryRecord{}
		err := json.Unmarshal([]byte(line), &historyRecord)
		if err != nil {
			log.Printf("ERROR: parsing history record %s: %s\n", line, err)
			return nil, err.Error()
		} else {
			records = append(records, historyRecord)
		}
	}
	return records, ""
}

func HistorySaveRecord(record HistoryRecord) error {
	line, err := json.Marshal(record)
	if err != nil {
		log.Printf("ERROR: marshall during saveHistoryRecord: %s\n", err)
		return err
	}
	line = append(line, '\n')
	file, err := os.OpenFile(configuration.HistoryFile,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("ERROR: opening during saveHistoryRecord %s: %s\n", configuration.HistoryFile, err)
		return err
	}
	defer file.Close()
	_, err = file.Write(line)
	if err != nil {
		log.Printf("ERROR: write during saveHistoryRecord: %s\n", err)
		return err
	}
	log.Println("history record saved, TimeDifference: " + record.TimeDifference)
	return nil
}
