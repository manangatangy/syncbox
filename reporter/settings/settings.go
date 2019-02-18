package settings

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reporter/config"
	"strconv"
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
	Checked     bool
	Description string
	// The Validator takes a new value (as the entered string), validates it and stores
	// it as the correct type in a field in the config.  It also should place the value
	// back in the Setting for response to the page. This should happen uncondictionally
	// since the page should always show the value as entered by the user.  Validators
	// have the choice of when to place the value in the page, since for a checkbox, the
	// value is encoded as the 'checked' attribute, not the value attribute.
	// If the validation fails, then an error is returned.
	// This field is only used when Readonly = false
	Validator func(newValue string, c *config.Configuration, s *Setting) error
}

// Create a Settings list from the specified Configuration values
// with the Description set and the Errored empty.
func getSettings(c config.Configuration) []Setting {
	var settings []Setting
	settings = append(settings, Setting{
		Id: "DialTimeout", Name: "Connection Timeout", Type: "number",
		Value: strconv.Itoa(c.DialTimeout), Description: "Retry count for the initial connection",
		Validator: func(newValue string, c *config.Configuration, s *Setting) error {
			var err error
			if c.DialTimeout, err = strconv.Atoi(newValue); err == nil {
				if c.DialTimeout < 1 || c.DialTimeout > 10000 {
					err = errors.New("out of range (1,10000)")
				}
			}
			s.Value = newValue // Unconditionally return to the page
			return err
		},
	})
	settings = append(settings, Setting{
		Id: "Port", Name: "Server Port", Type: "text",
		Value: c.Port, Description: "Reporter server listening port",
		Readonly: true,
	})
	settings = append(settings, Setting{
		Id: "AcerFilePath", Name: "Acer File Path", Type: "text",
		Value: c.AcerFilePath, Description: "Location of file containing AcerStatus",
		Validator: func(newValue string, c *config.Configuration, s *Setting) error {
			c.AcerFilePath = newValue
			s.Value = newValue
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "SyncFolderId", Name: "Syncthing Folder Id", Type: "text",
		Value: c.SyncFolderId, Description: "Identifies folder being monitored (from Syncthing-GUI)",
		Validator: func(newValue string, c *config.Configuration, s *Setting) error {
			c.SyncFolderId = newValue
			s.Value = newValue
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "SyncApiKey", Name: "Syncthing API Key", Type: "text",
		Value: c.SyncApiKey, Description: "Authorises API access (from Syncthing-GUI)",
		Validator: func(newValue string, c *config.Configuration, s *Setting) error {
			c.SyncApiKey = newValue
			s.Value = newValue
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "SimmonLogFilePath", Name: "Simmon Log Path", Type: "text",
		Value: c.SimmonLogFilePath, Description: "Path to logfile for Simmon",
		Readonly: true,
	})
	settings = append(settings, Setting{
		Id: "ReporterLogFilePath", Name: "Reporter Log Path", Type: "text",
		Value: c.ReporterLogFilePath, Description: "Path to logfile for Reporter",
		Readonly: true,
	})
	settings = append(settings, Setting{
		Id: "SimmonLogAutoEmail", Name: "Simmon Auto Email", Type: "checkbox",
		Value: "enabled", Description: "Check to enable auto emailing of Simmon logs",
		Checked: c.SimmonLogAutoEmail,
		Validator: func(newValue string, c *config.Configuration, s *Setting) error {
			// When checked, the value may be "enabled" or "" (if unchecked)
			c.SimmonLogAutoEmail = (newValue != "")
			s.Checked = c.SimmonLogAutoEmail // Return to the page/checkbox
			// s.Value = "enabled"
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "ReporterLogAutoEmail", Name: "Reporter Auto Email", Type: "checkbox",
		Value: "enabled", Description: "Check to enable auto emailing of Reporter logs",
		Checked: c.ReporterLogAutoEmail,
		Validator: func(newValue string, c *config.Configuration, s *Setting) error {
			c.ReporterLogAutoEmail = (newValue != "")
			s.Checked = c.ReporterLogAutoEmail // Return to the page/checkbox
			// s.Value = "enabled"
			return nil
		},
	})

	return settings
}

func SettingsPage(w http.ResponseWriter, r *http.Request) {
	settingsPageVars := SettingsPageVariables{
		LocalServer: true,
	}
	// Fetch the current config values into the page. For GET (initial page load)
	// and for (POST, "reset") this will be the values 'returned' back to the page.
	settingsPageVars.Settings = getSettings(config.Get())
	fmt.Printf("SettingsPage method ===> %v\n", r.Method)
	if r.Method == http.MethodPost {
		r.ParseForm()
		if r.Form.Get("submit") == "yes" {
			c := config.Get() // Use a temp local configuration
			success := true
			for s := 0; s < len(settingsPageVars.Settings); s++ {
				setting := &settingsPageVars.Settings[s]
				// Only check writeable settings.
				// If there is a new value for the setting in the form, then store it in
				// the setting (where it can be sent back to the page) and validate it,
				// updating the local config with the new value. Indicate an error by
				// placing error details in the Description field and setting Errored flag.
				if !setting.Readonly {
					// could also use "if val, ok := m[key]; ok" to test for contains
					fmt.Printf("===> USING form value '%s' for key %s\n", setting.Value, setting.Id)
					if err := setting.Validator(r.Form.Get(setting.Id), &c, setting); err != nil {
						setting.Description = err.Error()
						setting.Errored = "errored"
						success = false
					}
				}
				// fmt.Printf("setting after validating ==> %v\n", setting)
			}
			// fmt.Printf("config after validating ==> %v\n", config)
			if success {
				// If all the settings are valid, then update the configuration.
				if err := config.Set(c); err == nil {
					settingsPageVars.SuccessMessage = "Settings updated successfully"
				} else {
					settingsPageVars.SuccessMessage = "Error saving config: " + err.Error()
				}
			}
		}
	}

	t, err := template.ParseFiles("settings/settings.html")
	if err != nil {
		log.Print("ERROR: SettingsPage template parsing error: ", err)
	}
	err = t.Execute(w, settingsPageVars)
	if err != nil {
		log.Print("ERROR: SettingsPage template executing error: ", err)
	}
}
