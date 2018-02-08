package util

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func Request(method string, uri string) ([]byte, error) {
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	byteArray, err := ioutil.ReadAll(resp.Body)
	return byteArray, err
}

func URLParse(url string) (string, string) {
	path := ""
	port := ""
	tmp := strings.Split(url, ":")
	if len(tmp) > 1 {
		path, port = tmp[0], tmp[1]
	} else {
		path, port = tmp[0], ""
	}
	return path, port
}

func GetNowTime() (time.Time, string) {
	// 時刻更新
	now := time.Now()
	nowUTC := now.UTC()
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	nowJST := nowUTC.In(jst)
	strJST := nowJST.Format("15:04:05 2006-01-02")
	return nowJST, strJST
}
