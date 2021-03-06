package status

import (
	"bufio"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"reporter/config"
)

type HistoryPageVariables struct {
	Error       string
	LocalServer bool
	History     []BackupStatus
}

// Serve the history records as a page for local/connected access
func HistoryPage(w http.ResponseWriter, r *http.Request) {
	historyPageVariables := HistoryPageVariables{
		LocalServer: true,
	}
	HistoryFetch(w, &historyPageVariables)
}

// Fetches all history records from the history file,
// and writes the expanded html string to the parm.
// Errors are logged here.
// Nesting: https://stackoverflow.com/questions/11467731/is-it-possible-to-have-nested-templates-in-go-using-the-standard-library-googl
func HistoryFetch(w io.Writer, historyPageVariables *HistoryPageVariables) error {
	history, err1 := ReadStatusHistory()
	historyPageVariables.History = history
	historyPageVariables.Error = err1
	t, err2 := template.ParseFiles("status/history.html")
	if err2 != nil {
		log.Print("ERROR: HistoryFetch template parsing error: ", err2)
	}
	err3 := t.Execute(w, historyPageVariables)
	if err3 != nil {
		log.Print("ERROR: HistoryFetch template executing error: ", err3)
	}
	return nil
}

func ReadStatusHistory() ([]BackupStatus, string) {
	historyPath := config.Get().HistoryFile
	file, err := os.Open(historyPath)
	if err != nil {
		log.Printf("ERROR: opening for read %s: %s\n", historyPath, err)
		return nil, err.Error()
	}
	defer file.Close()
	var history []BackupStatus
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		status := BackupStatus{}
		err := json.Unmarshal([]byte(line), &status)
		if err != nil {
			log.Printf("ERROR: parsing history record %s: %s\n", line, err)
			return nil, err.Error()
		} else {
			history = append(history, status)
		}
	}
	return history, ""
}

func SaveStatusToHistory(record BackupStatus) error {
	// TODO protect with mutex
	line, err := json.Marshal(record)
	if err != nil {
		log.Printf("ERROR: marshall during SaveStatusToHistory: %s\n", err)
		return err
	}
	line = append(line, '\n')
	historyPath := config.Get().HistoryFile
	file, err := os.OpenFile(historyPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("ERROR: opening during saveHiSaveStatusToHistorystoryRecord %s: %s\n", historyPath, err)
		return err
	}
	defer file.Close()
	_, err = file.Write(line)
	if err != nil {
		log.Printf("ERROR: write during SaveStatusToHistory: %s\n", err)
		return err
	}
	return nil
}
