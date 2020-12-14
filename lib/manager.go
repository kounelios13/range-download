package lib

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
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


	fragments := make([]Fragment ,0) // Keep data received from range requests
	body := make([]byte ,0)
	var globalError error
	// First we need to determine the filesize
	response , err := http.Head(url) // We perform a Head request to get header information

	if response.StatusCode != http.StatusOK{
		return nil ,fmt.Errorf("received code %d",response.StatusCode)
	}
	if err != nil{
		return nil , err
	}

	maxConnections := m.limit // Number of maximum concurrent go routines
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

	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(maxConnections)
	for i:=0;i<maxConnections;i++{
		min := bufferSize * i
		if i != 0{
			min++
		}
		max := bufferSize * (i+1)
		if i==maxConnections-1{
			max+=diff // Check to see if we have any leftover data to retrieve for the last request
		}

		go func(lowerBound,upperBound int, index int,mutex *sync.Mutex, waitgroup *sync.WaitGroup) {
			mutex.Lock()
			defer mutex.Unlock()
			defer waitgroup.Done()
			req , _ := http.NewRequest("GET" , url, nil)
			req.Header.Add("Range" ,fmt.Sprintf("bytes=%d-%d",lowerBound,upperBound))
			res , e := http.DefaultClient.Do(req)
			if e != nil{
				globalError = e
			}
			data , e :=ioutil.ReadAll(res.Body)
			res.Body.Close()
			if e != nil{
				globalError = e
				return
			}
			fragment := Fragment{Data: data, Index: index} // We store the information we got and their respective index
			fragments = append(fragments, fragment)
		}(min,max,i,&mu,&wg)

	}

	wg.Wait()
	// We retrieved the data . However we need to make sure we have them in the correct order in order to reconstruct them
	sort.Slice(fragments , func(i, j int) bool {
		return fragments[i].Index < fragments[j].Index
	})

	// Start reconstruction
	for _ ,fr:= range fragments{
		body = append(body, fr.Data...)
	}
	return body, globalError
}