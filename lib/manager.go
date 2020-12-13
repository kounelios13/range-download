package lib

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type Manager struct{
	limit int
}


func NewManager(limit int) *Manager{
	return &Manager{
		limit: limit,
	}
}

func (m *Manager) DownloadBody(url string ) ([]byte ,error){
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
	bodySize , _ := strconv.Atoi(response.Header.Get("Content-Length"))
	bufferSize :=(bodySize) / (maxConnections)
	diff := bodySize % maxConnections
	read := 0
	for i:=0;i<maxConnections;i++{
		min := bufferSize * i
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