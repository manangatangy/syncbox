package main

// support logging /home/pi/syncbox/reporter/reporter -logfile /home/pi/syncbox/reporter/reporter.log

// Refs: https://golang.org/pkg/
import (
	// "bufio"
	// "encoding/json"
	"errors"
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
	LocalServer    bool
	SuccessMessage string
	Settings       []Setting
}

type Setting struct {
	Id          string
	Name        string
	Type        string
	Value       string
	Readonly    bool
	Errored     string
	Description string
	// The Validator takes a new value (as the entered string), validates it and stores it
	// as the correct type in a field in the config.  If the validation fails, then the
	// storage is not performed and an error is returned.
	Validator func(newValue string, config Configuration) error
}

// Create a Settings list from the current Configuration values
// with the Description set and the Errored empty.
func getCurrentSettings() []Setting {
	var settings []Setting
	settings = append(settings, Setting{
		Id: "DialTimeout", Name: "Connection Timeout", Type: "number",
		Value: strconv.Itoa(configuration.DialTimeout), Description: "Retry count for the initial connection",
		Validator: func(newValue string, config Configuration) error {
			var err error
			config.DialTimeout, err = strconv.Atoi(newValue)
			if config.DialTimeout < 100 || config.DialTimeout > 10000 {
				err = errors.New("out of range (100,10000)")
			}
			return err
		},
	})
	settings = append(settings, Setting{
		Id: "Port", Name: "Server Port", Type: "text",
		Value: configuration.Port, Description: "Reporter server listening port",
		Readonly: true,
	})
	settings = append(settings, Setting{
		Id: "AcerFilePath", Name: "Acer File Path", Type: "text",
		Value: configuration.AcerFilePath, Description: "Location of file containing AcerStatus",
		Validator: func(newValue string, config Configuration) error {
			config.AcerFilePath = newValue
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "SyncFolderId", Name: "Syncthing Folder Id", Type: "text",
		Value: configuration.SyncFolderId, Description: "Identifies folder being monitored (from Syncthing-GUI)",
		Validator: func(newValue string, config Configuration) error {
			config.SyncFolderId = newValue
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "SyncApiKey", Name: "Syncthing API Key", Type: "text",
		Value: configuration.SyncApiKey, Description: "Authorises API access (from Syncthing-GUI)",
		Validator: func(newValue string, config Configuration) error {
			config.SyncApiKey = newValue
			return nil
		},
	})
	return settings
}

// Create a Settings list with values taken from the Form (or the Configuration by default)
// Use the getCurrentSettings function because it sets the other Setting fields.
func getSettingsFromForm(Form url.Values) []Setting {
	settings := getCurrentSettings()
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
		// http.MethodGet: Place the current config values into the page.
		settingsPageVars.Settings = getCurrentSettings()
	} else {
		// http.MethodPost:
		r.ParseForm()
		if r.Form.Get("submit") == "yes" {
			settingsPageVars.Settings = getSettingsFromForm(r.Form)
			// Validate/set the new values from the settings into a temp config.
			config := ConfigurationLoad()
			success := true
			for _, setting := range settingsPageVars.Settings {
				// Validate the new value (val).  An error is indicated by placing
				// error details in the Description field and setting Errored
				if !setting.Readonly {
					if err := setting.Validator(setting.Value, config); err != nil {
						setting.Description = err.Error()
						setting.Errored = "errored"
						success = false
					}
				}
				fmt.Printf("setting after validating ==> %v\n", setting)
			}
			fmt.Printf("config after validating ==> %v\n", config)
			if success {
				// If all the settings are valid, then save the configuration, and
				// replace the current global configuration value.
				// TODO
				settingsPageVars.SuccessMessage = "Settings updated"
			}
		} else if r.Form.Get("reset") == "yes" {
			settingsPageVars.Settings = getCurrentSettings()
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
