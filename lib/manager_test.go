package lib

import (
	"bytes"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	kb int64 = 1024
	mb       = kb * 1024
	gb       = mb * 1024
)

func generateDummyData(size int64) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

func TestManager_DownloadBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data := "hello world!!!!"
		http.ServeContent(writer, request, "hey", time.Now(), bytes.NewReader([]byte(data)))
	}))

	defer ts.Close()

	m := NewManager(10000)
	data, err := m.DownloadBody(ts.URL)
	if err != nil {
		t.Errorf("%s", err)
	}

	got := string(data)
	want := "hello world!!!!"
	if got != want {
		t.Errorf("Expected hello world!!!! . received : [%s]", data)
	}
}

func TestNewManager(t *testing.T) {
	type testCase struct {
		limit int64
		name  string
		data  []byte
	}

	bigCaseData := generateDummyData(10 ^ 12)
	bigCaseData2 := generateDummyData(10 ^ 18)
	cases := []testCase{
		//{limit: 100 ,data:[]byte(`hello bob`)},
		{name: "limit 2 repeat 5", limit: 2, data: []byte(strings.Repeat("Hello alice", 5))},
		{name: "limit 4 repeat 100", limit: 4, data: []byte(strings.Repeat("Hello you again", 100))},
		{name: "limit 8 no repeat", limit: 8, data: []byte("h")},
		{name: "limit 1000 repeat 10^4", limit: 1000, data: []byte(strings.Repeat("Hello. This will be a long string", 10000))},
		{name: "limit 10^4 repeat 10^4", limit: 10000, data: []byte(strings.Repeat("Hello. This will be a long string", 10000))},
		{name: "limit 10^7 repeat 1", limit: 10000000, data: []byte(strings.Repeat("Hello. This will be a long string", 1))},
		{name: "limit 10^12", limit: 10 ^ 12, data: bigCaseData},
		{name: "limit 10^18", limit: 10 ^ 18, data: bigCaseData2},
		{name: "Size 3MB limit 500", limit: 500, data: generateDummyData(mb * 3)},
		{name: "Size 5 bytes max connections 4", limit: 4, data: generateDummyData(5)},
	}

	for _, item := range cases {
		manager := NewManager(item.limit)
		dataToTest := item.data
		ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.ServeContent(writer, request, "Hello", time.Now(), bytes.NewReader(dataToTest))
		}))
		resData, _ := manager.DownloadBody(ts.URL)
		ts.Close()
		//comparison := bytes.Compare(resData, dataToTest)
		comparison := string(resData) == string(dataToTest)
		if !comparison {
			t.Errorf("Different data received. Test name [%s] [%s] [%s]", item.name, (resData), (dataToTest))
		}
	}
}

func benchmarkDownload(limit int64, url string, b *testing.B) {
	m := NewManager(limit)
	for n := 0; n < b.N; n++ {
		m.DownloadBody(url)
	}
}

func BenchmarkDownloadManager_DownloadBody10(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		dataToTest := strings.Repeat("this is a very long long long string", 10000)
		http.ServeContent(writer, request, "Hello", time.Now(), bytes.NewReader([]byte(dataToTest)))
	}))
	defer ts.Close()
	benchmarkDownload(10, ts.URL, b)
}

func BenchmarkDownloadManager_DownloadBody100(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		dataToTest := strings.Repeat("this is a very long long long string", 10000)
		http.ServeContent(writer, request, "Hello", time.Now(), bytes.NewReader([]byte(dataToTest)))
	}))
	defer ts.Close()
	benchmarkDownload(100, ts.URL, b)
}

func BenchmarkDownloadManager_DownloadBody1000(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		dataToTest := strings.Repeat("this is a very long long long string", 10000)
		http.ServeContent(writer, request, "Hello", time.Now(), bytes.NewReader([]byte(dataToTest)))
	}))
	defer ts.Close()
	benchmarkDownload(1000, ts.URL, b)
}

func BenchmarkDownloadManager_DownloadBody10000(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		dataToTest := strings.Repeat("this is a very long long long string", 10000)
		http.ServeContent(writer, request, "Hello", time.Now(), bytes.NewReader([]byte(dataToTest)))
	}))
	defer ts.Close()
	benchmarkDownload(10000, ts.URL, b)
}

func Test_normalizeMaxConnections(t *testing.T) {
	type args struct {
		dataSize       int64
		maxConnections int64
		minSize        int64
	}

	type testCase struct {
		name string
		args args
		want int64
	}

	uc1a := testCase{
		name: "Size 100 bytes Connections:200",
		args: args{dataSize: 100, maxConnections: 100, minSize: 0},
		want: 99,
	}
	uc1b := testCase{
		name: "Size 25 bytes Connections:25",
		args: args{dataSize: 25, maxConnections: 25, minSize: 0},
		want: 24,
	}

	uc2 := testCase{
		name: "Size 100 bytes Connections:10,Min size:1MB",
		args: args{dataSize: 100, maxConnections: 10, minSize: mb},
		want: 1,
	}

	uc2b := testCase{
		name: "Size 2Gb connections 4294967296",
		args: args{dataSize: 2 * gb, maxConnections: 4294967296, minSize: 0},
		want: 2*gb - 1,
	}

	uc2c := testCase{
		name: "Size 500 bytes connections 700",
		args: args{dataSize: 500, maxConnections: 700, minSize: 0},
		want: 499,
	}
	tests := []testCase{
		uc1a,
		uc1b,
		uc2,
		uc2b,
		uc2c,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeMaxConnections(tt.args.dataSize, tt.args.maxConnections, tt.args.minSize); got != tt.want {
				t.Errorf("normalizeMaxConnections() .size [%d] .Received conections :[%d] got connections: [%d], want connections: [%v] min-size [%d]", tt.args.dataSize, tt.args.maxConnections, got, tt.want, tt.args.minSize)
			}
		})
	}
}

func TestDownloadManager_ChangeClient(t *testing.T) {


	type fields struct {
		limit  int64
		client *http.Client
	}
	type args struct {
		c *http.Client
	}


	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Passing a client should not return an error",
			fields: fields{client: nil,limit: 0},
			args: args{
				c: http.DefaultClient,
			},
			wantErr: false,
		},
		{
			name: "Passing a nil client should  return an error",
			fields: fields{client: nil,limit: 0},
			args: args{
				c: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DownloadManager{
				limit:  tt.fields.limit,
				client: tt.fields.client,
			}
			if err := m.ChangeClient(tt.args.c); (err != nil) != tt.wantErr {
				t.Errorf("ChangeClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}