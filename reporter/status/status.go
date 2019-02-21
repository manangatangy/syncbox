package status

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reporter/config"
	"strings"
	"time"
)

const (
	ACER_TIME_FORMAT   = "03:04 PM, Mon 02/01/2006 MST" // As found on AcerDataFile
	REPORT_TIME_FORMAT = "2006-01-02 15:04:00"          // As written to reports
)

// Represents the dfference between an AcerStatus and a SyncthingStatus
type BackupStatus struct {
	ServerTime    string // Also the timestamp for this record
	MissingFiles  int32  // BackedUp/Acer diff
	MissingBytes  int64  // BackedUp/Acer diff
	BackedUpFiles int32  // Syncthing.localFiles
	BackedUpBytes int64  // Syncthing.localBytes
	AcerFiles     int32
	AcerBytes     int64
	AcerTimeStamp string
	AcerAge       string // How long after ServerTime is the AcerTimeStamp expressed as ddd:hh:mm
}

type AcerStatus struct {
	Title      string
	DateString string
	TimeString string
	// TimeStamp  string
	FileCount int32
	ByteCount int64
}

// Struct fields must be exported/visible otherwise unmarshall cannot see them.
// Ref: https://stackoverflow.com/a/28228444/1402287
type SyncthingStatus struct {
	Errors            int
	GlobalBytes       int64
	GlobalDeleted     int
	GlobalDirectories int
	GlobalFiles       int
	GlobalSymlinks    int
	GlobalTotalItems  int
	IgnorePatterns    bool
	InSyncBytes       int64
	InSyncFiles       int
	Invalid           string
	LocalBytes        int64
	LocalDeleted      int
	LocalDirectories  int
	LocalFiles        int32
	LocalSymlinks     int
	LocalTotalItems   int
	NeedBytes         int64
	NeedDeletes       int
	NeedDirectories   int
	NeedFiles         int
	NeedSymlinks      int
	NeedTotalItems    int
	PullErrors        int
	Sequence          int
	State             string
	StateChanged      string
	// "2019-02-12T10:47:15.179455+11:00",
	Version int
}

// Return a new current BackupStatus, using the current contents of the AcerFile, and a fresh
// call to the Syncthng API. The freshness of the AcerFile will be indicated by the AcerAge.
// If an error occurs, it is logged here, and a partially populated BackupStatus is returned.
// Missing counts will be -1 to indicate an error with either or both the get calls.
func GetBackupStatus() (*BackupStatus, error) {
	var errText string
	backupStatus := BackupStatus{}
	serverTime := time.Now()
	backupStatus.ServerTime = serverTime.Format(REPORT_TIME_FORMAT)

	syncthingStatus, err1 := GetSyncthingStatus()
	if err1 != nil {
		errText = err1.Error()
	} else {
		// fmt.Printf("syncthingStatus ==> %v\n", syncthingStatus)
		backupStatus.BackedUpFiles = syncthingStatus.LocalFiles
		backupStatus.BackedUpBytes = syncthingStatus.LocalBytes
	}
	acerStatus, err2 := GetAcerStatus()
	if err2 != nil {
		errText = errText + " + " + err2.Error()
	} else {
		// fmt.Printf("acerStatus ==> %v\n", acerStatus)
		backupStatus.AcerFiles = acerStatus.FileCount
		backupStatus.AcerBytes = acerStatus.ByteCount
		backupStatus.AcerTimeStamp = acerStatus.TimeString + ", " + acerStatus.DateString + " " + config.Get().AcerTimeZone
		if err1 == nil {
			backupStatus.MissingFiles = backupStatus.AcerFiles - backupStatus.BackedUpFiles
			backupStatus.MissingBytes = backupStatus.AcerBytes - backupStatus.BackedUpBytes
			// The AcerTimeString as read from the file, is parsed and then reformatted nicely.
			acerTime, err3 := parseAcerTimeStamp(backupStatus.AcerTimeStamp)
			if err3 != nil {
				errText = errText + " + " + err3.Error()
			} else {
				backupStatus.AcerTimeStamp = acerTime.Format(REPORT_TIME_FORMAT)
				diff := serverTime.Sub(*acerTime)
				backupStatus.AcerAge = diff.String()
			}
		}
	}
	var err error
	if len(errText) > 0 {
		err = errors.New(errText)
	}
	return &backupStatus, err
}

// Read the file contents at AcerStatusPath and create a corresponding AcerStatus
// If an error occurs, it is logged here
func GetAcerStatus() (*AcerStatus, error) {
	acerFilePath := config.Get().AcerFilePath
	file, err := os.Open(acerFilePath)
	if err != nil {
		log.Printf("ERROR: opening for read %s: %s\n", acerFilePath, err)
		return nil, err
	}
	defer file.Close()
	acerStatus := AcerStatus{
		FileCount: -1, ByteCount: -1,
	}
	var builder strings.Builder
	// Remainder of the input file.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(acerStatus.Title) == 0 {
			acerStatus.Title = line
		} else if len(acerStatus.DateString) == 0 {
			acerStatus.DateString = line
		} else if len(acerStatus.TimeString) == 0 {
			acerStatus.TimeString = line
		} else {
			builder.WriteString(line)
		}
	}
	stringData := builder.String()
	syncthingStatus := SyncthingStatus{}
	err = json.Unmarshal([]byte(stringData), &syncthingStatus)
	if err != nil {
		log.Printf("ERROR: AcerFile json.Unmarshal: %s\n", err)
		log.Printf("ERROR: AcerFile api response: " + stringData)
		syncthingStatus.LocalFiles = -1
		syncthingStatus.LocalBytes = -1
		return nil, err
	}
	acerStatus.FileCount = syncthingStatus.LocalFiles
	acerStatus.ByteCount = syncthingStatus.LocalBytes
	// TODO check all fields were present in the input file
	return &acerStatus, nil
}

// The date/time string read from the acer status file is like
// "03:04 PM, Mon 02/01/2006 MST"
// Parse this into a golang time struct, for use in comparison
func parseAcerTimeStamp(acerDateTime string) (*time.Time, error) {
	time, err := time.Parse(ACER_TIME_FORMAT, acerDateTime)
	if err != nil {
		log.Printf("ERROR: parsing acerTimeStamp %s: %s\n", acerDateTime, err)
		return nil, err
	}
	return &time, nil
}

// Read the response from SyncApiEndpoint/SyncFolderId and return a corresponding SyncthingStatus
// If an error occurs, it is logged here
func GetSyncthingStatus() (*SyncthingStatus, error) {
	endpoint := config.Get().SyncApiEndpoint + "?folder=" + config.Get().SyncFolderId
	client := &http.Client{}
	request, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Printf("ERROR: SyncthingStatus http.NewRequest: %s\n", err)
		return nil, err
	}
	request.Header.Set("X-API-Key", config.Get().SyncApiKey)
	response, err := client.Do(request)
	if err != nil {
		log.Printf("ERROR: SyncthingStatus http.Client.Do-request: %s\n", err)
		return nil, err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("ERROR: SyncthingStatus ioutil.ReadAll: %s\n", err)
		return nil, err
	}
	syncthingStatus := SyncthingStatus{}
	err = json.Unmarshal(data, &syncthingStatus)
	if err != nil {
		log.Printf("ERROR: SyncthingStatus json.Unmarshal: %s\n", err)
		log.Printf("ERROR: syncthing api response: " + string(data))
		syncthingStatus.LocalFiles = -1
		syncthingStatus.LocalBytes = -1
		return nil, err
	}
	return &syncthingStatus, nil
}
