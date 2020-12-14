package main

import (
	"concurent-downloader/lib"
	"io/ioutil"
	"log"
)

type MSeeker struct{

}

func (m *MSeeker) Seek(offset int64, whence int) (int64, error){
	return 0,nil
}
//Seek(offset int64, whence int) (int64, error)
func main() {
	//ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
	//
	//	http.ServeContent(writer,request ,"hey" ,time.Now() ,bytes.NewReader([]byte(`hello world!!!!`)))
	//}))
	//defer ts.Close()
	fUrl:="https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1-recent.json.zip"
	//imgUrl := "https://media.wired.com/photos/5a593a7ff11e325008172bc2/16:9/w_2400,h_1350,c_limit/pulsar-831502910.jpg"
	maxConnections := 4
	manager := lib.NewManager(maxConnections)
	data , e:= manager.DownloadBody(fUrl)
	if  e!= nil{
		log.Fatalln(e)
	}
	//fmt.Printf("Data %s",data)
	ioutil.WriteFile("foosome.zip" , data,0777)
	//r , _ := http.Get(ts.URL)
	//data , _ := ioutil.ReadAll(r.Body)
	//defer r.Body.Close()
	//fmt.Printf("%s \n",data)
}
