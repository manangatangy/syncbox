package main

// support logging /home/pi/syncbox/reporter/reporter -logfile /home/pi/syncbox/reporter/reporter.log

// Refs: https://golang.org/pkg/
import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	STATIC_DIR         = "/static/"                     // prefix for urls withing templated html
	ACER_TIME_FORMAT   = "03:04 PM, Mon 02/01/2006 MST" // As found on AcerDataFile
	REPORT_TIME_FORMAT = "2006-01-02 15:04:00"          // As written to reports
)

func main() {
	// Only -config=cfgpath and -logfile=logpath are supported.
	flag.StringVar(&configPath, "config", "config.json", "path to configuration file")
	logFilePath := flag.String("logfile", "", "path to logfile")
	flag.Parse()
	if *logFilePath != "" {
		f, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		CheckDie(err)
		defer f.Close()
		log.SetOutput(f)
	}

	log.Println("STARTING ...")
	ConfigurationLoad()

	// saveConfiguration()
	router := mux.NewRouter().StrictSlash(true)

	// Ref: https://gowebexamples.com/static-files/
	log.Println("serving static assets from: " + configuration.AssetsRoot)

	staticHandler := http.FileServer(http.Dir(configuration.AssetsRoot))
	router.PathPrefix(STATIC_DIR).Handler(http.StripPrefix(STATIC_DIR, staticHandler))
	// Test:  curl -s http://localhost:8090/static/test.txt

	router.HandleFunc("/", HomePage)
	router.HandleFunc("/history", HistoryPage)

	log.Printf("listening at: %s:%s\n", getOutboundIP(), configuration.Port)
	log.Fatal("FATAL: ", http.ListenAndServe(":"+configuration.Port, router))
}

func CheckDie(e error) {
	if e != nil {
		log.Fatal("FATAL: ", e)
	}
}

/*
 - port for serving html [8090]
 - path to root of served documents (may be absolute or relative to wd) [./]
 - path to static documents (may be absolute or relative to wd) [./static]
 - syncthing check period in hours [24]
 - report email target [me@gmail.com]
 - report email period in hours [24]
*/
type Configuration struct {
	DialTimeout     int
	Port            string
	AcerFilePath    string
	AcerTimeZone    string
	SyncApiEndpoint string
	SyncApiKey      string
	SyncFolderId    string

	DocRoot      string
	AssetsRoot   string
	CheckHours   int
	EmailHours   int
	EmailTargets []string
	HistoryFile  string
}

var configPath string
var configuration Configuration

func ConfigurationLoad() {
	// Load the config (from configPath) making it globally available.
	log.Println("config path: ", configPath)
	content, err := ioutil.ReadFile(configPath)
	CheckDie(err)
	// Ref: https://blog.golang.org/json-and-go
	configuration = Configuration{}
	err = json.Unmarshal(content, &configuration)
	CheckDie(err)
	log.Println("configuration loaded")
}

func ConfigurationSave() {
	content, err := json.MarshalIndent(configuration, "", "  ")
	CheckDie(err)
	err = ioutil.WriteFile(configPath, content, 0666)
	CheckDie(err)
	log.Println("configuration saved")
}

type HomePageVariables struct {
	LocalServer bool
	Error       string
	Status      BackupStatus
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	// now := time.Now()
	// emailerResult := "SendReport OK"
	// if err := SendReport(); err == nil {
	// 	log.Println(emailerResult)
	// } else {
	// 	emailerResult = err.Error()
	// }
	// Ref: https://gowebexamples.com/templates/

	homePageVars := HomePageVariables{
		LocalServer: true,
	}
	backupStatus, err := GetBackupStatus()
	if err != nil {
		homePageVars.Error = err.Error()
	} else {
		homePageVars.Status = *backupStatus
	}
	t, err := template.ParseFiles("home.html")
	if err != nil {
		log.Print("ERROR: HomePage template parsing error: ", err)
	}
	err = t.Execute(w, homePageVars)
	if err != nil {
		log.Print("ERROR: HomePage template executing error: ", err)
	}
}

// Get preferred outbound ip of this machine
// Ref: https://stackoverflow.com/a/37382208/1402287
func getOutboundIP() net.IP {
	// Try for up to DialTimeout seconds before quitting
	attempts := 0
	for {
		conn, err := net.Dial("udp", "8.8.8.8:80")
		if err == nil {
			defer conn.Close()
			localAddr := conn.LocalAddr().(*net.UDPAddr)
			return localAddr.IP
		}
		attempts = attempts + 1
		if attempts >= configuration.DialTimeout {
			log.Fatal("FATAL: ", err)
		}
		log.Print("ERROR: failed to net.Dial, trying again in 1 second; ", err)
		time.Sleep(1 * time.Second)
	}
}

func execAndProcessError(handleError bool,
	w http.ResponseWriter, c string, arg ...string) (string, error) {
	// Ref: https://stackoverflow.com/a/32721097/1402287
	// And: https://stackoverflow.com/a/16252034/1402287
	s := []string{c}
	s = append(s, arg...)
	fmt.Println("exec:")
	fmt.Println(s)

	out, err := exec.Command("bash", s...).Output()
	if handleError && err != nil {
		fmt.Printf("error: %s\n", err)
		fmt.Printf("output: %s\n", out)
		json.NewEncoder(w).Encode(GetErrorResponse{
			ResultCode: "Fail",
			Message:    err.Error(),
		})
	} else {
		fmt.Printf("success: %s\n", out)
	}
	return string(out), err
}

func GetCurrent(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling GetCurrent request...")
	out, err := execAndProcessError(true, w, scripts["current"])
	if err == nil {
		json.NewEncoder(w).Encode(GetCurrentResponse{
			ResultCode: "OK",
			Current:    strings.TrimSpace(out),
		})
	}
}

func GetLocations(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling GetLocations request...")
	out, err := execAndProcessError(true, w, scripts["locations"])
	if err == nil {
		var locations []string
		// Ref: https://stackoverflow.com/a/33162587/1402287
		// And: https://stackoverflow.com/a/33162487/1402287
		scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(out)))
		for scanner.Scan() {
			// Parse locations from output.
			s := strings.TrimSpace(scanner.Text())
			//fmt.Printf("scanned: %s\n", s)
			locations = append(locations, s)
		}
		json.NewEncoder(w).Encode(GetLocationsResponse{
			ResultCode: "OK",
			Locations:  locations,
		})
	}
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling GetStatus request...")
	out, err := execAndProcessError(false, w, scripts["status"])
	// This script forces exit 0
	if err == nil {
		var squidActive bool = false
		var vpnActive bool = false
		var vpnLocation string = "N.A."
		scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(out)))
		for scanner.Scan() {
			// Parse active and current from output.
			s := strings.TrimSpace(scanner.Text())
			if strings.Contains(s, "squid3 is running") {
				squidActive = true
			}
			match, _ := regexp.MatchString("VPN ['a-zA-Z0-9_]* is running", s)
			if match {
				vpnLocation = strings.Split(s, "'")[1]
				vpnActive = true
			}
		}
		json.NewEncoder(w).Encode(GetStatusResponse{
			ResultCode:  "OK",
			SquidActive: squidActive,
			VpnActive:   vpnActive,
			VpnLocation: vpnLocation,
		})
	}
}

func GetPing(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling GetPing request...")
	vars := mux.Vars(r)
	target := vars["target"]

	out, err := execAndProcessError(false, w, scripts["ping"], target)
	if err == nil {
		json.NewEncoder(w).Encode(GetPingResponse{
			ResultCode: "OK",
			Target:     target,
		})
	} else {
		json.NewEncoder(w).Encode(GetErrorResponse{
			ResultCode: "Fail",
			Message:    strings.TrimSpace(out),
		})
	}
}

func PostStart(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling PostStart request...")
	out, err := execAndProcessError(false, w, scripts["start"])
	// This script forces exit 0
	if err == nil {
		var squidOk bool = false
		var vpnOk bool = false
		scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(out)))
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if strings.Contains(s, "Starting virtual private network daemon") {
				vpnOk = true
			}
			if strings.Contains(s, "Starting Squid HTTP Proxy") {
				squidOk = true
			}
		}
		var resultCode string = "OK"
		if !vpnOk || !squidOk {
			resultCode = "Fail"
			if !vpnOk {
				resultCode = resultCode + ", VPN-not-started"
			}
			if !squidOk {
				resultCode = resultCode + ", Squid-not-started"
			}
		}
		json.NewEncoder(w).Encode(PostStartResponse{
			ResultCode: resultCode,
		})
	}
}

func PostStop(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling PostStop request...")
	out, err := execAndProcessError(false, w, scripts["stop"])
	// This script forces exit 0
	if err == nil {
		var squidOk bool = false
		var vpnOk bool = false
		scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(out)))
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if strings.Contains(s, "Stopping virtual private network daemon") {
				vpnOk = true
			}
			if strings.Contains(s, "Stopping Squid HTTP Proxy") {
				squidOk = true
			}
		}
		var resultCode string = "OK"
		if !vpnOk || !squidOk {
			resultCode = "Fail"
			if !vpnOk {
				resultCode = resultCode + ", VPN-not-stopped"
			}
			if !squidOk {
				resultCode = resultCode + ", Squid-not-stopped"
			}
		}
		json.NewEncoder(w).Encode(PostStopResponse{
			ResultCode: resultCode,
		})
	}
}

func PostSwitch(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling PostSwitch request...")
	vars := mux.Vars(r)
	newLocation := vars["newLocation"]
	oldLocation := ""

	fmt.Println("Fetching current location...")
	out, err := execAndProcessError(false, w, scripts["current"])
	if err == nil {
		oldLocation = strings.TrimSpace(out)
	}

	fmt.Println("Now Handling PostSwitch request...")
	out, err = execAndProcessError(false, w, scripts["switch"], newLocation)
	// This script will always exit 0
	if err == nil {
		var resultCode string = "Fail"
		scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(out)))
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if strings.Contains(s, "vpn config switched") {
				resultCode = "OK"
			}
		}
		json.NewEncoder(w).Encode(PostSwitchResponse{
			ResultCode:  resultCode,
			OldLocation: oldLocation,
			NewLocation: newLocation,
		})
	}
}

// Refs: https://www.thepolyglotdeveloper.com/2016/07/create-a-simple-restful-api-with-golang/
// https://www.thepolyglotdeveloper.com/2017/07/consume-restful-api-endpoints-golang-application/
// https://golang.org/pkg/net/http/

type GetErrorResponse struct {
	ResultCode string `json:"resultCode"`
	Message    string `json:"message"`
}

type GetCurrentResponse struct {
	ResultCode string `json:"resultCode"`
	Current    string `json:"current"`
}

type GetLocationsResponse struct {
	ResultCode string   `json:"resultCode"`
	Locations  []string `json:"locations"`
}

type GetStatusResponse struct {
	ResultCode  string `json:"resultCode"`
	SquidActive bool   `json:"squidActive"`
	VpnActive   bool   `json:"vpnActive"`
	VpnLocation string `json:"vpnLocation"`
}

type GetPingResponse struct {
	ResultCode string `json:"resultCode"`
	Target     string `json:"target"`
}

type PostStartResponse struct {
	ResultCode string `json:"resultCode"`
}

type PostStopResponse struct {
	ResultCode string `json:"resultCode"`
}

type PostSwitchResponse struct {
	ResultCode  string `json:"resultCode"`
	OldLocation string `json:"oldLocation"`
	NewLocation string `json:"newLocation"`
}

var realScripts = map[string]string{
	"current":   "./vpn_current.sh",
	"locations": "./vpn_locations.sh",
	"ping":      "./vpn_ping.sh",
	"start":     "./vpn_start.sh",
	"status":    "./vpn_status.sh",
	"stop":      "./vpn_stop.sh",
	"switch":    "./vpn_switch.sh",
}

var passingTestScripts = map[string]string{
	"current":   "./pass_vpn_current.sh",
	"locations": "./pass_vpn_locations.sh",
	"ping":      "./pass_vpn_ping.sh",
	"start":     "./pass_vpn_start.sh",
	"status":    "./pass_vpn_status.sh",
	"stop":      "./pass_vpn_stop.sh",
	"switch":    "./pass_vpn_switch.sh",
}

var failingTestScripts = map[string]string{
	"current":   "./fail_vpn_current.sh",
	"locations": "./fail_vpn_locations.sh",
	"ping":      "./fail_vpn_ping.sh",
	"start":     "./fail_vpn_start.sh",
	"status":    "./fail_vpn_status.sh",
	"stop":      "./fail_vpn_stop.sh",
	"switch":    "./fail_vpn_switch.sh",
}

var scripts map[string]string
