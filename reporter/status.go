package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	// "time"
	// "bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type AcerStatus struct {
	TimeStamp string
	FileCount int32
	ByteCount int64
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
			acerStatus.TimeStamp = line
		} else if acerStatus.FileCount == -1 {
			i64, err := strconv.ParseInt(line, 10, 32)
			if err != nil {
				log.Printf("ERROR: parsing acerStatus.FileCount from %s: %s\n", line, err)
				return nil, err
			}
			acerStatus.FileCount = int32(i64)
		} else if acerStatus.ByteCount == -1 {
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
	log.Printf("%s, %d, %d\n", acerStatus.TimeStamp, acerStatus.FileCount, acerStatus.ByteCount)
	return &acerStatus, nil
}

type SyncthingStatus struct {
	errors            int
	globalBytes       int64
	globalDeleted     int
	globalDirectories int
	globalFiles       int
	globalSymlinks    int
	globalTotalItems  int
	ignorePatterns    bool
	inSyncBytes       int64
	inSyncFiles       int
	invalid           string
	localBytes        int64
	localDeleted      int
	localDirectories  int
	localFiles        int
	localSymlinks     int
	localTotalItems   int
	needBytes         int64
	needDeletes       int
	needDirectories   int
	needFiles         int
	needSymlinks      int
	needTotalItems    int
	pullErrors        int
	sequence          int
	state             string
	stateChanged      string
	// "2019-02-12T10:47:15.179455+11:00",
	version int
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
		return nil, err
	}
	fmt.Println("syncthingStatus==> " + string(data))
	return &syncthingStatus, nil
}
