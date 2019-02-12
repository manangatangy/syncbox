package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	// "bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
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
	TimeStamp string
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
	backupStatus := BackupStatus{}
	serverTime := time.Now()
	backupStatus.ServerTime = serverTime.Format(REPORT_TIME_FORMAT)

	syncthingStatus, err1 := GetSyncthingStatus()
	if err1 == nil {
		backupStatus.BackedUpFiles = syncthingStatus.LocalFiles
		backupStatus.BackedUpBytes = syncthingStatus.LocalBytes
	}
	acerStatus, err2 := GetAcerStatus()
	if err2 == nil {
		backupStatus.AcerFiles = acerStatus.FileCount
		backupStatus.AcerBytes = acerStatus.ByteCount
		backupStatus.AcerTimeStamp = acerStatus.TimeStamp
		if err1 == nil {
			backupStatus.MissingFiles = backupStatus.AcerFiles - backupStatus.BackedUpFiles
			backupStatus.MissingBytes = backupStatus.AcerBytes - backupStatus.BackedUpBytes
			// The AcerTimeString as read from the file, is parsed and then reformatted nicely.
			acerTime, err3 := parseAcerTimeStamp(backupStatus.AcerTimeStamp)
			if err3 == nil {
				backupStatus.AcerTimeStamp = acerTime.Format(REPORT_TIME_FORMAT)
				diff := serverTime.Sub(*acerTime)
				backupStatus.AcerAge = diff.String()
			}
		}
	}
	return &backupStatus, nil
}

// Read the file contents at AcerStatusPath and create a corresponding AcerStatus
// If an error occurs, it is logged here
func GetAcerStatus() (*AcerStatus, error) {
	file, err := os.Open(configuration.AcerFilePath)
	if err != nil {
		log.Printf("ERROR: opening for read %s: %s\n", configuration.AcerFilePath, err)
		return nil, err
	}
	defer file.Close()
	acerStatus := AcerStatus{
		FileCount: -1, ByteCount: -1,
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(acerStatus.TimeStamp) == 0 {
			fmt.Println("==> found timestamp: " + line)
			acerStatus.TimeStamp = line
		} else if acerStatus.FileCount == -1 {
			fmt.Println("==> found filecount: " + line)
			i64, err := strconv.ParseInt(line, 10, 32)
			if err != nil {
				log.Printf("ERROR: parsing acerStatus.FileCount from %s: %s\n", line, err)
				return nil, err
			}
			acerStatus.FileCount = int32(i64)
		} else if acerStatus.ByteCount == -1 {
			fmt.Println("==> found bytecount: " + line)
			i64, err := strconv.ParseInt(line, 10, 64)
			if err != nil {
				log.Printf("ERROR: parsing acerStatus.ByteCount from %s: %s\n", line, err)
				return nil, err
			}
			acerStatus.ByteCount = i64
		} else {
			break
		}
	}
	// TODO check all fields were present in the input file
	fmt.Printf("==> %s, %d, %d\n", acerStatus.TimeStamp, acerStatus.FileCount, acerStatus.ByteCount)
	return &acerStatus, nil
}

// The date/time string read from the acer status file is like "2019 3 21 10:05 pm"
// Parse this into a golang time struct, for use in comparison
func parseAcerTimeStamp(acerDateTime string) (*time.Time, error) {
	timeString := acerDateTime + " " + configuration.AcerTimeZone
	fmt.Printf("==> parsing acerTimeStamp %s\n", timeString)
	// TODO this will change once I determine the correct input format
	time, err := time.Parse(ACER_TIME_FORMAT, timeString)
	if err != nil {
		log.Printf("ERROR: parsing acerTimeStamp %s: %s\n", timeString, err)
		return nil, err
	}
	return &time, nil
}

// Read the response from SyncApiEndpoint/SyncFolderId and return a corresponding SyncthingStatus
// If an error occurs, it is logged here
func GetSyncthingStatus() (*SyncthingStatus, error) {
	endpoint := configuration.SyncApiEndpoint + "?folder=" + configuration.SyncFolderId
	client := &http.Client{}
	request, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Printf("ERROR: http.NewRequest: %s\n", err)
		return nil, err
	}
	request.Header.Set("X-API-Key", configuration.SyncApiKey)
	response, err := client.Do(request)
	if err != nil {
		log.Printf("ERROR: http.Client.Do-request: %s\n", err)
		return nil, err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("ERROR: ioutil.ReadAll: %s\n", err)
		return nil, err
	}
	syncthingStatus := SyncthingStatus{}
	err = json.Unmarshal(data, &syncthingStatus)
	if err != nil {
		log.Printf("ERROR: json.Unmarshal: %s\n", err)
		syncthingStatus.LocalFiles = -1
		syncthingStatus.LocalBytes = -1
		return nil, err
	}
	fmt.Printf("syncthingStatus==> %v\n", syncthingStatus)
	return &syncthingStatus, nil
}
