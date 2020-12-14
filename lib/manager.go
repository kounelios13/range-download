package lib

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type DownloadManager struct{
	limit int
}


func NewManager(limit int) *DownloadManager {
	return &DownloadManager{
		limit: limit,
	}
}

func (m *DownloadManager) DownloadBody(url string ) ([]byte ,error){
	// First we need to determine the filesize
	body := make([]byte ,0)
	response , err := http.Head(url) // We perform a Head request to get header information

	if response.StatusCode != http.StatusOK{
		return nil ,fmt.Errorf("received code %d",response.StatusCode)
	}
	if err != nil{
		return nil , err
	}

	maxConnections := m.limit // Number of maximum concurrent co routines
	// need to ensure that if we use maxConnections we won't wxceed original file size (in case it is to small)
	bodySize , e := strconv.Atoi(response.Header.Get("Content-Length"))
	if e != nil{
		log.Fatalln(e)
	}
	bufferSize :=(bodySize) / (maxConnections)
	diff := bodySize % maxConnections
	if bufferSize == 0{
		// Data size is to small to break into ranged requests . Perform a simple request instead
		maxConnections = 1 // force a single request
		bufferSize = bodySize // by setting buffersize to body size we force the code to perform a single range request
		// and get all the bytes at once
	}
	read := 0
	for i:=0;i<maxConnections;i++{
		min := bufferSize * i
		if i != 0{
			min++
		}
		max := bufferSize * (i+1)
		if i==maxConnections-1{
			max+=diff // Check to see if we have any leftover data to retrieve for the last request
		}
		req , _ := http.NewRequest("GET" , url, nil)
		req.Header.Add("Range" ,fmt.Sprintf("bytes=%d-%d",min,max))
		res , e := http.DefaultClient.Do(req)
		if e != nil{
			return body , e
		}
		log.Printf("Index:%d . Range:bytes=%d-%d",i,min,max)
		data , e :=ioutil.ReadAll(res.Body)
		res.Body.Close()
		if e != nil{
			return body,e
		}
		log.Println("Data for  request: ",len(data))
		read = read + len(data)
		body = append(body, data...)
	}
	log.Println("File size:",bodySize , "Downloaded size:",len(body)," Actual read:",read)
	return body, nil
}