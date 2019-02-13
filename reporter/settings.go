package main

// support logging /home/pi/syncbox/reporter/reporter -logfile /home/pi/syncbox/reporter/reporter.log

// Refs: https://golang.org/pkg/
import (
	// "bufio"
	// "encoding/json"
	// "errors"
	"fmt"
	// "github.com/gorilla/mux"
	"html/template"
	// "io/ioutil"
	"log"
	// "net"
	"net/http"
	"net/url"
	"strconv"
	// "os"
	// "os/exec"
	// "regexp"
	// "strings"
	// "time"
)

type SettingsPageVariables struct {
	LocalServer bool
	Error       string
	Settings    []Setting
}

type Setting struct {
	Id          string
	Name        string
	Type        string
	Value       string
	Readonly    bool
	Description string
}

// Create a Settings list from the current Configuration values.
func readSettingsFromConfig() []Setting {
	var settings []Setting
	settings = append(settings, Setting{
		Id: "DialTimeout", Name: "Connection Timeout", Type: "number",
		Value: strconv.Itoa(configuration.DialTimeout), Description: "Retry count for the initial connection",
	})
	settings = append(settings, Setting{
		Id: "Port", Name: "Server Port", Type: "text", Readonly: true,
		Value: configuration.Port, Description: "Reporter server listening port",
	})
	settings = append(settings, Setting{
		Id: "AcerFilePath", Name: "Acer File Path", Type: "text",
		Value: configuration.AcerFilePath, Description: "Location of file containing AcerStatus",
	})
	settings = append(settings, Setting{
		Id: "SyncFolderId", Name: "Syncthing Folder Id", Type: "text",
		Value: configuration.SyncFolderId, Description: "Identifies folder being monitored (from Syncthing-GUI)",
	})
	settings = append(settings, Setting{
		Id: "SyncApiKey", Name: "Syncthing API Key", Type: "text",
		Value: configuration.SyncApiKey, Description: "Authorises API access (from Syncthing-GUI)",
	})
	// settings = append(settings, Setting{
	// 	Id: "SyncApiKey", Name: "Syncthing", Type: "text",
	// 	Value: configuration.Port, Description: "",
	// })
	return settings
}

// Create a Settings list with values taken from the Form (or the Configuration by default)
// Use the readSettingsFromConfig function because it sets the other Setting fields.
func getSettingsFromForm(Form url.Values) []Setting {
	settings := readSettingsFromConfig()
	for key, vals := range Form {
		var val string
		if len(vals) != 0 {
			val = vals[0]
		}
		// fmt.Printf("===> FORM returned %s ===> %s ==> %s\n", key, vals, val)
		for s := 0; s < len(settings); s++ {
			// setting := settings[s]
			if settings[s].Id == key {
				fmt.Printf("===> USING form value %s for key %s\n", val, key)
				settings[s].Value = val
				break
			}
		}
	}
	return settings
}

// func findSetting(settings *[]Setting, id string) (*Setting, error) {
// 	for s := 0; s < len(*settings); s++ {
// 		setting := (*settings)[s]
// 		if setting.Id == id {
// 			return &setting, nil
// 		}
// 	}
// 	return nil, errors.New("no Setting for " + id)
// }

func SettingsPage(w http.ResponseWriter, r *http.Request) {
	settingsPageVars := SettingsPageVariables{
		LocalServer: true,
	}
	fmt.Printf("SettingsPage method ===> %v\n", r.Method)
	if r.Method != http.MethodPost {
		// This is the getting of the field values.
		settingsPageVars.Settings = readSettingsFromConfig()
	} else {
		// This is the setting of the field values.
		r.ParseForm()
		if r.Form.Get("submit") == "yes" {
			settingsPageVars.Settings = getSettingsFromForm(r.Form)
			// for k, v := range r.Form {
			// 	fmt.Printf("FORM ===> %s = %s\n", k, v)
			// }
			settingsPageVars.Error = "some error !!! (not really)"
		} else if r.Form.Get("reset") == "yes" {
			settingsPageVars.Settings = readSettingsFromConfig()
		}
	}

	t, err := template.ParseFiles("settings.html")
	if err != nil {
		log.Print("ERROR: SettingsPage template parsing error: ", err)
	}
	err = t.Execute(w, settingsPageVars)
	if err != nil {
		log.Print("ERROR: SettingsPage template executing error: ", err)
	}
}
