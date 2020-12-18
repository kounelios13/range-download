package lib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"sync"
)

// normalizeMaxConnections - Return the closest number of max connections for a given data size
// that we can use.
func normalizeMaxConnections(dataSize int64, maxConnections int64, minSizeForConnection int64) int64 {

	// Warning. We are dealing with bytes here

	// UC1 : Data size is equal to the number of connections
	// E.g: dataSize 100bytes  maxConnections 100
	// What do we do. We may return the number of data size / 2 so the user can iterate through ranges
	// However we need to take into account the actual data size. If it is too small it is not worth the actual overhead

	// We need to set some constraints
	// Constraint 1: Requests with data size less than a specific size will occupy only one connection
	// Constraint 2: Max connections can never be equal to the actual data size . If we allow that to happen we will get requests failing due to them
	// trying to request a range like this : bytes=1-1 or bytes=100-00 which is incorrect
	// Constraint 3: The number of max connection to return cannot be more that what the user requested . Only exception is when user enters zero or a negative value
	// Constraint 4: Max connections cannot exceed the number of request size .

	// Constraint 5: In order to perform range request we need to have a condition such as
	// connections = datasize div 2 in order to be able to generate valid range headers such as bytes=1-2 , bytes=3,4

	max := maxConnections

	// Constraint 1

	isDataSizeSmall := minSizeForConnection >= 2 && minSizeForConnection > dataSize
	if isDataSizeSmall || max < 1 {
		return 1
	}

	// Constraint 2
	if dataSize == max {
		// UC1
		max--
		return max
	}

	//max = int(dataSize) / 2
	if dataSize < maxConnections {
		// Do not return
		// If the size of the data is too small the max will be 0 . We cannot return 0
		// The last if statement will fix that for us
		max = dataSize - 1
	}

	if max == 0 {
		// Datasize is to small. Return 1 to avoid a range request
		return 1
	}
	return max
}

type DownloadManager struct {
	limit int64
}

func NewManager(limit int64) *DownloadManager {
	return &DownloadManager{
		limit: limit,
	}
}

func (m *DownloadManager) DownloadBodyOld(url string) ([]byte, error) {

	fragments := make([]Fragment, 0) // Keep data received from range requests
	maxConnections := m.limit        // Number of maximum concurrent go routines
	body := make([]byte, 0)
	var globalError error
	response, err := http.Head(url) // We perform a Head request to get header information

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received code %d", response.StatusCode)
	}
	if err != nil {
		return nil, err
	}

	// First we need to determine the filesize

	bodySize := response.ContentLength
	maxConnections = normalizeMaxConnections(bodySize, m.limit, 0)
	bufferSize := (bodySize) / (maxConnections)
	diff := (bodySize) % maxConnections
	if bufferSize == 0 {
		// Data size is to small to break into ranged requests . Perform a simple request instead
		maxConnections = 1    // force a single request
		bufferSize = bodySize // by setting buffersize to body size we force the code to perform a single range request
		// and get all the bytes at once
	}
	rangeHeader := response.Header.Get("Accept-Ranges")
	supportsPartialContent := rangeHeader != "" && rangeHeader != "none"

	if !supportsPartialContent {
		maxConnections = 1
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(int(maxConnections))
	log.Println("Maximum connections:", maxConnections)
	for i := int64(0); i < maxConnections; i++ {
		min := bufferSize * i
		if i != 0 {
			min++
		}
		max := bufferSize * (i + 1)
		if i == maxConnections-1 {
			max += diff // Check to see if we have any leftover data to retrieve for the last request
		}

		go func(lowerBound, upperBound, index int64, mutex *sync.Mutex, waitgroup *sync.WaitGroup) {
			mutex.Lock()
			defer mutex.Unlock()
			defer waitgroup.Done()
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", lowerBound, upperBound))
			res, e := http.DefaultClient.Do(req)
			if e != nil {
				globalError = e
			}
			data, e := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if e != nil {
				globalError = e
				return
			}
			fragment := Fragment{Data: data, Index: index} // We store the information we got and their respective index
			fragments = append(fragments, fragment)
		}(min, max, i, &mu, &wg)

	}

	wg.Wait()
	// We retrieved the data . However we need to make sure we have them in the correct order in order to reconstruct them
	sort.Slice(fragments, func(i, j int) bool {
		return fragments[i].Index < fragments[j].Index
	})

	// Start reconstruction
	for _, fr := range fragments {
		body = append(body, fr.Data...)
	}
	return body, globalError
}

func (m *DownloadManager) DownloadBody(url string) ([]byte, error) {

	log.Println("Will use channels")
	fragments := make([]Fragment, 0) // Keep data received from range requests
	maxConnections := m.limit        // Number of maximum concurrent go routines
	body := make([]byte, 0)
	var globalError error
	response, err := http.Head(url) // We perform a Head request to get header information

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received code %d", response.StatusCode)
	}
	if err != nil {
		return nil, err
	}

	// First we need to determine the filesize

	bodySize := response.ContentLength
	maxConnections = normalizeMaxConnections(bodySize, m.limit, 0)

	//maxConnections = 109
	bufferSize := (bodySize) / (maxConnections)
	rangeHeader := response.Header.Get("Accept-Ranges")
	supportsPartialContent := rangeHeader != "" && rangeHeader != "none"
	if !supportsPartialContent {
		maxConnections = 1
	}


	diff := (bodySize) % maxConnections
	if bufferSize == 0 {
		// Data size is to small to break into ranged requests . Perform a simple request instead
		maxConnections = 1    // force a single request
		bufferSize = bodySize // by setting buffersize to body size we force the code to perform a single range request
		// and get all the bytes at once
	}


	var wg sync.WaitGroup
	wg.Add(int(maxConnections))
	ch := make(chan Fragment, maxConnections)
	for i := int64(0); i < maxConnections; i++ {
		if globalError != nil{
			return nil, globalError
		}
		min := bufferSize * i
		if i != 0 {
			min++
		}
		max := bufferSize * (i + 1)
		if i == maxConnections-1 {
			max += diff // Check to see if we have any leftover data to retrieve for the last request
		}

		go func(lowerBound, upperBound, index int64, waitgroup *sync.WaitGroup) {

			defer waitgroup.Done()
			req, f := http.NewRequest("GET", url, nil)
			if req == nil{
				log.Fatalln(f)
			}
			req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", lowerBound, upperBound))

			res, e := http.DefaultClient.Do(req)
			if res == nil {

				globalError = errors.New("empty response body")
			}
			if e != nil {
				globalError = e
				return
			}

			if res.Body == nil {

				globalError = errors.New("empty response body")
				return
			}
			data, e := ioutil.ReadAll(res.Body)
			_  = res.Body.Close()
			if e != nil {
				globalError = e
				return
			}
			fragment := Fragment{Data: data, Index: index} // We store the information we got and their respective index
			ch <- fragment
		}(min, max, i, &wg)

	}

	wg.Wait()
	var status = make(map[int64]bool)


	var read int64 = 0
	// We retrieved the data . However we need to make sure we have them in the correct order in order to reconstruct them
	for i:=int64(0);i<maxConnections;i++{
		fr := <- ch

		if status[fr.Index] == true{
			log.Fatalln("Value exists :",fr.Index)
		}

		read += int64(len(fr.Data))
		status[fr.Index] = true
//		fmt.Println("Received value index:",fr.Index)
		fragments = append(fragments, fr)
	}
	sort.Slice(fragments, func(i, j int) bool {
		return fragments[i].Index < fragments[j].Index
	})

	log.Println("Total fragments:",len(fragments))
	// Start reconstruction

	for _, fr := range fragments {
		body = append(body, fr.Data...)
	}

	log.Printf("Read:[%d] . Original :[%d]\n",read , bodySize)
	return body, globalError
}
