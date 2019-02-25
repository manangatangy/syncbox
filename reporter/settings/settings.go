package settings

import (
	"errors"
	// "fmt"
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
	Checked     string // Is either "checked" or ""
	Description string

	// The validator's job is to read appropriate field value(s) from the request Form, place
	// them into the setting for reply to the page, and also (if they validate OK) write them
	// to the settings.
	Validator func(f url.Values, c *config.Configuration, s *Setting) error
}

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
		Id: "EmailFrom", Name: "Email Account", Type: "text",
		Value: c.EmailFrom, Description: "Email account address used to send reports",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id)
			end := 0
			for j, r := range s.Value {
				if r == '@' {
					end = j
					break
				}
			}
			// Ref: a very useful substring routine
			// https://stackoverflow.com/a/38537764/1402287
			if end == 0 {
				return errors.New("not a valid email address")
			} else {
				c.EmailFrom = s.Value
				c.EmailUserName = s.Value[0:end]
				return nil
			}
		},
	})

	settings = append(settings, Setting{
		Id: "EmailPassword", Name: "Email Password", Type: "text",
		Value: c.EmailPassword, Description: "Email account password used to send reports",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id)
			c.EmailPassword = s.Value
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "EmailTo", Name: "Email Target", Type: "text",
		Value: c.EmailTo, Description: "Target email address for all reports",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id)
			c.EmailTo = s.Value
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "AcerFileWatchPeriod", Name: "Acer File Period", Type: "number",
		Value:       strconv.Itoa(c.AcerFileWatchPeriod),
		Description: "File polling period in seconds",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Value = f.Get(s.Id) // [s.Id] Unconditionally return to the page
			var err error
			if c.AcerFileWatchPeriod, err = strconv.Atoi(s.Value); err == nil {
				if c.AcerFileWatchPeriod < 10 {
					err = errors.New("out of range (10,)")
				}
			}
			return err
		},
	})
	settings = append(settings, Setting{
		// For checkboxes, the Value is always "checked" and the Checked field is set
		// from the form and written to the html input.
		Id: "EnableAcerFileWatch", Name: "Watch Acer File", Type: "checkbox",
		Value:       "checked",
		Checked:     formatChecked(c.EnableAcerFileWatch),
		Description: "Create and email new history record, on acer file change",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Checked = f.Get(s.Id)
			c.EnableAcerFileWatch = (s.Checked != "")
			config.ReloadConfig(config.KEY_STATUS)
			return nil
		},
	})
	settings = append(settings, Setting{
		// For checkboxes, the Value is always "checked" and the Checked field is set
		// from the form and written to the html input.
		Id: "HistoryFileAutoAppend", Name: "History Auto Append", Type: "checkbox",
		Value:       "checked",
		Checked:     formatChecked(c.HistoryFileAutoAppend),
		Description: "At email report time, add fresh backupStatus to history",
		Validator: func(f url.Values, c *config.Configuration, s *Setting) error {
			s.Checked = f.Get(s.Id)
			c.HistoryFileAutoAppend = (s.Checked != "")
			return nil
		},
	})
	settings = append(settings, Setting{
		Id: "HistoryFile", Name: "History File Path", Type: "text",
		Value: c.HistoryFile, Description: "Path to History file of backup status's",
		Readonly: true,
	})
	settings = append(settings, Setting{
		Id: "ReporterLogFilePath", Name: "Reporter Log Path", Type: "text",
		Value: c.ReporterLogFilePath, Description: "Path to logfile for Reporter",
		Readonly: true,
	})
	settings = append(settings, Setting{
		Id: "SimmonLogFilePath", Name: "Simmon Log Path", Type: "text",
		Value: c.SimmonLogFilePath, Description: "Path to logfile for Simmon",
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
	Key         int

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

type GetAutoConfig func(c *config.Configuration) (aec *config.AutoEmailConfig)

func makeAutoEmailSetting(shortName string, id string, c config.Configuration, get GetAutoConfig, key int) AutoEmailSetting {
	var aec *config.AutoEmailConfig = get(&c)
	return AutoEmailSetting{
		Id: id, Name: shortName + " Auto Email",
		Checked:     formatChecked(aec.AutoEmailEnable),
		Count:       strconv.Itoa(aec.AutoEmailCount),
		Period:      aec.AutoEmailPeriod,
		NextEmail:   aec.AutoEmailNext,
		Description: "Check to enable auto emailing of " + shortName + " logs",
		Key:         key,
		Validator: func(f url.Values, s *AutoEmailSetting, c *config.Configuration) error {
			var aec *config.AutoEmailConfig = get(c)
			// Validation is only performed (and the nextEmail value updated) if the checkbox is
			// changed from clear (during which the Count, Period can be entered) to set (after which
			// Count and Period are readonly).
			s.Checked = f.Get(shortName + "LogAutoEmail_Checked")
			// checkbox has changed from set -> clear
			reload := (aec.AutoEmailEnable && (s.Checked == ""))
			if !aec.AutoEmailEnable && (s.Checked != "") {
				// checkbox has changed from clear -> set
				reload = true
				s.Count = f.Get(shortName + "LogAutoEmail_Count")
				s.Period = f.Get(shortName + "LogAutoEmail_Period")
				// Update the Period, count, next fields
				aec.AutoEmailCount, _ = strconv.Atoi(s.Count)
				aec.AutoEmailPeriod = s.Period
				// Calculate next as now plus the specified period
				_, s.NextEmail = CalculateNextTime(time.Now(), aec.AutoEmailCount, aec.AutoEmailPeriod)
				aec.AutoEmailNext = s.NextEmail
			} else {
				s.Count = strconv.Itoa(aec.AutoEmailCount)
				s.Period = aec.AutoEmailPeriod
			}
			aec.AutoEmailEnable = (s.Checked != "")
			if reload {
				config.ReloadConfig(key)
			}
			return nil
		},
	}
}

func CalculateNextTime(from time.Time, count int, period string) (time.Time, string) {
	// Calculate next as now plus the specified period
	switch period {
	case "secs":
		from = from.Add(time.Second * time.Duration(count))
	case "mins":
		from = from.Add(time.Minute * time.Duration(count))
	case "hours":
		from = from.Add(time.Hour * time.Duration(count))
	case "days":
		from = from.AddDate(0, 0, count)
	case "weeks":
		from = from.AddDate(0, 0, count*7)
	}
	return from, from.Format(config.TIME_FORMAT)
}

// html creates elements for
// s.Checked ==> ReporterLogAutoEmail_Checked,
// s.Count ==> ReporterLogAutoEmail_Count,
// s.Period ==> ReporterLogAutoEmail_Period,
// s.NextEmail ==> ReporterLogAutoEmail_NextEmail  (readonly)
func getAutoEmailSettings(c config.Configuration) []AutoEmailSetting {
	var settings []AutoEmailSetting
	settings = append(settings,
		makeAutoEmailSetting("History", "HistoryLogAutoEmail", c,
			func(c *config.Configuration) (aec *config.AutoEmailConfig) {
				return &c.HistoryLogAutoEmail
			}, config.KEY_HISTORY))
	settings = append(settings,
		makeAutoEmailSetting("Reporter", "ReporterLogAutoEmail", c,
			func(c *config.Configuration) (aec *config.AutoEmailConfig) {
				return &c.ReporterLogAutoEmail
			}, config.KEY_REPORTER))
	settings = append(settings,
		makeAutoEmailSetting("Simmon", "SimmonLogAutoEmail", c,
			func(c *config.Configuration) (aec *config.AutoEmailConfig) {
				return &c.SimmonLogAutoEmail
			}, config.KEY_SIMMON))

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
	// fmt.Printf("SettingsPage method ===> %v\n", r.Method)
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
					// fmt.Printf("===> USING form value '%s' for key %s\n", setting.Value, setting.Id)
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
