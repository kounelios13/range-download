package lib

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestManager_DownloadBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(writer,request ,"hey" ,time.Now() ,bytes.NewReader([]byte(`hello world!!!!`)))
	}))

	defer ts.Close()


	m := NewManager(10000)
	data , err := m.DownloadBody(ts.URL)
	if err != nil{
		t.Errorf("%s",err)
	}

	if string(data) != "hello world!!!!"{
		t.Errorf("Expected hello world!!!! . received : [%s]",data)
	}

}

func TestNewManager(t *testing.T) {
	type testCase struct {
		limit int
		data []byte
	}

	cases := []testCase{
		//{limit: 100 ,data:[]byte(`hello bob`)},
		{limit: 2 , data:[]byte(strings.Repeat("Hello alice",5))},
		{limit: 4,data: []byte(strings.Repeat("Hello you again",100))},
		{limit: 8 , data: []byte("h")},
		{limit: 1000 , data: []byte(strings.Repeat("Hello. This will be a long string",10000))},
		{limit: 10000 , data: []byte(strings.Repeat("Hello. This will be a long string",10000))},
	}

	for _ , item := range cases{
		manager := NewManager(item.limit)
		dataToTest := item.data
		ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.ServeContent(writer,request ,"Hello" ,time.Now() ,bytes.NewReader(dataToTest))
		}))
		resData , _ := manager.DownloadBody(ts.URL)
		ts.Close()
		comparison := bytes.Compare(resData, dataToTest)
		if comparison != 0 {
			t.Errorf("Different data received")
		}
	}
	//ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
	//	http.ServeContent(writer,request ,"hey" ,time.Now() ,bytes.NewReader([]byte(`hello world!!!!`)))
	//}))
	//
	//defer ts.Close()
}
