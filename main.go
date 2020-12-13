package main

import (
	"concurent-downloader/lib"
	"io/ioutil"
	"log"
)


func main() {
	nvUrl := "https://media.wired.com/photos/5a593a7ff11e325008172bc2/16:9/w_2400,h_1350,c_limit/pulsar-831502910.jpg"
	manager := lib.NewManager(200)
	data , e:= manager.DownloadBody(nvUrl)
	if  e!= nil{
		log.Fatalln(e)
	}
	ioutil.WriteFile("foo.jpg" , data,0777)
}
