package settings

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"reporter/config"
	"strconv"
	"time"
)

type SettingsPageVariables struct {
	LocalServer       bool
	SuccessMessage    string
	Settings          []Setting
	AutoEmailSettings []AutoEmailSetting
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

	// The validator's job is to read appropriate field value(s) from the request Form, place
	// them into the setting for reply to the page, and also (if they validate OK) write them
	// to the settings.
	Validator func(f url.Values, c *config.Configuration, s *Setting) error
}

const (
	EMAIL_TIME_FORMAT = "2006-01-02 15:04:00"
	// time, err := time.Parse(ACER_TIME_FORMAT, acerDateTime)
)

// Create a Settings list from the specified Configuration values
// with the Description set and the Errored empty.
func getSettings(c config.Configuration) []Setting {
	var settings []Setting
	settings = append(settings, Setting{
		Id: "DialTimeout", Name: "Connection Timeout", Type: "number",
		Value: strconv.Itoa(c.DialTimeout), Description: "Retry count for the initial connection",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id) // [s.Id] Unconditionally return to the page
			var err error
			if c.DialTimeout, err = strconv.Atoi(s.Value); err == nil {
				if c.DialTimeout < 1 || c.DialTimeout > 10000 {
					err = errors.New("out of range (1,10000)")
				}
			}
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
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id) // [s.Id] Unconditionally return to the page
			c.AcerFilePath = s.Value
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "SyncFolderId", Name: "Syncthing Folder Id", Type: "text",
		Value: c.SyncFolderId, Description: "Identifies folder being monitored (from Syncthing-GUI)",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id) // [s.Id] Unconditionally return to the page
			c.SyncFolderId = s.Value
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "SyncApiKey", Name: "Syncthing API Key", Type: "text",
		Value: c.SyncApiKey, Description: "Authorises API access (from Syncthing-GUI)",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id) // [s.Id] Unconditionally return to the page
			c.SyncApiKey = s.Value
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
	return settings
}

type AutoEmailSetting struct {
	Id          string
	Name        string
	Checked     string // Is either "checked" or ""
	Count       string
	Period      string
	NextEmail   string
	Description string
	Errored     string

	// The validator's job is to read appropriate field value(s) from the request Form, place
	// them into the setting for reply to the page, and also (if they validate OK) write them
	// to the config.
	Validator func(f url.Values, s *AutoEmailSetting, c *config.Configuration) error
}

// The Validator takes a new value (as the entered string), validates it and stores
// it as the correct type in a field in the config.  It also should place the value
// back in the Setting for response to the page. This should happen uncondictionally
// since the page should always show the value as entered by the user.  Validators
// have the choice of when to place the value in the page, since for a checkbox, the
// value is encoded as the 'checked' attribute, not the value attribute.
// If the validation fails, then an error is returned.
// This field is only used when Readonly = false

// Returns "checked" or ""
func formatChecked(checked bool) string {
	if checked {
		return "checked"
	} else {
		return ""
	}
}

// Returns true (for "checked") or false (for anything else)
func parseChecked(checked string) bool {
	if checked == "checked" {
		return true
	} else {
		return false
	}
}

// html creates elements for
// s.Checked ==> ReporterLogAutoEmail_Checked,
// s.Count ==> ReporterLogAutoEmail_Count,
// s.Period ==> ReporterLogAutoEmail_Period,
// s.NextEmail ==> ReporterLogAutoEmail_NextEmail  (readonly)
func getAutoEmailSettings(c config.Configuration) []AutoEmailSetting {
	var settings []AutoEmailSetting
	settings = append(settings, AutoEmailSetting{
		Id: "ReporterLogAutoEmail", Name: "Reporter Auto Email",
		Checked:     formatChecked(c.ReporterLogAutoEmailEnable),
		Count:       strconv.Itoa(c.ReporterLogAutoEmailCount),
		Period:      c.ReporterLogAutoEmailPeriod,
		NextEmail:   c.ReporterLogAutoEmailNext,
		Description: "Check to enable auto emailing of Reporter logs",
		Validator: func(f url.Values, s *AutoEmailSetting, c *config.Configuration) error {
			// Validation is only performed (and the nextEmail value updated) if the checkbox is
			// changed from clear (during which the Count, Period can be entered) to set (after which
			// Count and Period are readonly).
			s.Checked = f.Get("ReporterLogAutoEmail_Checked")
			if !c.ReporterLogAutoEmailEnable && (s.Checked != "") {
				s.Count = f.Get("ReporterLogAutoEmail_Count")
				s.Period = f.Get("ReporterLogAutoEmail_Period")
				// Update the Period, count, next fields
				c.ReporterLogAutoEmailCount, _ = strconv.Atoi(s.Count)
				c.ReporterLogAutoEmailPeriod = s.Period
				// Calculate next as now plus the specified period
				next := time.Now()
				switch s.Period {
				case "hours":
					next = next.Add(time.Hour * time.Duration(c.ReporterLogAutoEmailCount))
				case "days":
					next = next.AddDate(0, 0, c.ReporterLogAutoEmailCount)
				case "weeks":
					next = next.AddDate(0, 0, 7*c.ReporterLogAutoEmailCount)
				}
				s.NextEmail = next.Format(EMAIL_TIME_FORMAT)
				c.ReporterLogAutoEmailNext = s.NextEmail
			} else {
				s.Count = strconv.Itoa(c.ReporterLogAutoEmailCount)
				s.Period = c.ReporterLogAutoEmailPeriod

			}
			c.ReporterLogAutoEmailEnable = (s.Checked != "")
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
	settingsPageVars.AutoEmailSettings = getAutoEmailSettings(config.Get())
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
					if err := setting.Validator(r.Form, &c, setting); err != nil {
						setting.Description = err.Error()
						setting.Errored = "errored"
						success = false
					}
				}
				// fmt.Printf("setting after validating ==> %v\n", setting)
			}
			for a := 0; a < len(settingsPageVars.AutoEmailSettings); a++ {
				auto := &settingsPageVars.AutoEmailSettings[a]
				if err := auto.Validator(r.Form, auto, &c); err != nil {
					auto.Description = err.Error()
					auto.Errored = "errored"
					success = false
				}
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
