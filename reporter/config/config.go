package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"sync"
)

const (
	TIME_FORMAT = "2006-01-02 15:04:05"
)

type AutoEmailConfig struct {
	AutoEmailEnable bool
	AutoEmailCount  int
	AutoEmailPeriod string
	AutoEmailNext   string
}

// The following config data is stored in a file at ConfigPath
// Read/write access should be done using Path/Get/Set to make it thread safe.
// Path() must be called before Get() or Set()
type Configuration struct {
	DialTimeout     int    // Retry count for the initial connection.
	Port            string // port for serving html [8090]
	AcerFilePath    string // Location of file containing AcerStatus, read on demand or file-change.
	AcerTimeZone    string // Applied to AcerStatus date/time strings.
	SyncApiEndpoint string // Where syncthing status is obtained from.
	SyncApiKey      string // Form the syncthing-gui advanced page.
	SyncFolderId    string // As above.

	DocRoot    string // path to root of served documents (may be absolute or relative to wd) [./]
	AssetsRoot string // path to static documents (may be absolute or relative to wd) [./static]

	HistoryFile         string // Where the BackupStatus records are appended to.
	HistoryLogAutoEmail AutoEmailConfig

	ReporterLogFilePath  string
	ReporterLogAutoEmail AutoEmailConfig

	SimmonLogFilePath  string
	SimmonLogAutoEmail AutoEmailConfig

	EmailFrom     string
	EmailTo       string
	EmailUserName string
	EmailPassword string
	EmailHost     string
}

var configPath string
var configuration Configuration // Current config, not used directly outside of this package
var cached bool
var mutex = &sync.Mutex{}

func Path(path string) {
	mutex.Lock()
	defer mutex.Unlock()
	configPath = path
	cached = false // force reload upon next call to Get
}

// Returns a copy of the current configuration, loading from the
// previously specified path if not already available in the current cache.
func Get() Configuration {
	mutex.Lock()
	defer mutex.Unlock()
	// Structs containing only primitives are copied by value
	// Ref: https://stackoverflow.com/a/51638160/1402287
	if !cached {
		configuration = load()
		cached = true
	}
	config := configuration
	return config
}

// Assigns new values for the configuration, overwriting the current value
// and also writing out to the config path.
func Set(config Configuration) error {
	mutex.Lock()
	defer mutex.Unlock()
	configuration = config
	cached = true
	log.Println("config: set")
	return save(configuration)
}

// Loads a new config and returns (a copy)
func load() Configuration {
	log.Println("config: loading from path: ", configPath)
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal("FATAL: ", err)
	}
	// Ref: https://blog.golang.org/json-and-go
	config := Configuration{}
	if err := json.Unmarshal(content, &config); err != nil {
		log.Fatal("FATAL: ", err)
	}
	log.Println("config: loaded")
	return config
}

// Write the current configuration to the configPath.
// Json errors are fatal but file errors are returned.
func save(config Configuration) error {
	if !cached {
		err := errors.New("config not yet loaded")
		log.Println("Error: saving config: ", err.Error())
		return err
	}
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatal("FATAL: ", err)
	}
	err = ioutil.WriteFile(configPath, content, 0666)
	if err != nil {
		log.Println("Error: saving config: ", err.Error())
		return err
	}
	log.Println("config: saved to path: ", configPath)
	return nil
}
