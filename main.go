// Copyright 2015 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Sample helloworld is a basic App Engine flexible app.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/xiaca/transfer-nem-api/util"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	MODE      = 1                          // 1:シングルモード, 2:マルチモード
	FIRST_URL = "http://go.nem.ninja:7890" //最初にpeerlistを取得しにいくURLです
	HOUR      = 24                         // 更新間隔
)

type PeerList struct {
	Update    time.Time `json:"update"`
	UpdateStr string    `json:"update_str"`
	Inactive  []*Node   `json:"inactive"`
	Active    []*Node   `json:"active"`
	Busy      []*Node   `json:"busy"`
}

type Node struct {
	MetaData *MetaData `json:"metadata"`
	Endpoint *Endpoint `json:"endpoint"`
	Identity *Identity `json:"identity"`
	Height   int64     `json:"height"`
	Active   bool      `json:"active"`
}

type MetaData struct {
	Features    int         `json:"features"`
	Application interface{} `json:"features"`
	NetworkId   int         `json:"networkId"`
	Vesrion     string      `json:"vesrion"`
	Platform    string      `json:"platform"`
}

type Endpoint struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type Height struct {
	Height int64 `json:"height"`
}

type Identity struct {
	Name      string `json:"name"`
	PublicKey string `json:"public-key"`
}

var peerList PeerList
var maxHeight int64

func init() {
	uri := FIRST_URL + "/node/peer-list/all"
	byteArray, _ := util.Request("GET", uri)
	_ = json.Unmarshal(byteArray, &peerList)
	if MODE == 2 {
		go func() {
			t := time.NewTicker(HOUR * time.Hour) // 指定時間置きに実行
			GetMultiNode()
			for {
				select {
				case <-t.C:
					GetMultiNode()
				}
			}
			t.Stop() // タイマを止める。
		}()
	}
}

func main() {
	http.HandleFunc("/", handle)
	http.HandleFunc("/_ah/health", healthCheckHandler)
	log.Print("Listening on port 8080 and 7890")
	go func() {
		log.Fatal(http.ListenAndServe(":7890", nil))
	}()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	_, port := util.URLParse(host)
	switch port {
	case "8080":
		// ポート：8080なら取得したAPI情報を返却
		path := r.URL.Path
		switch path {
		case "/":
			GetPeerList(w, r)
		default:
			// それ以外はエラー
			http.Error(w, "URL is not found.", 404)
		}
	case "7890":
		// ポート：7890ならNISへリクエストを中継
		TransferApi(w, r)
	default:
		// それ以外はエラー
		http.Error(w, "Not found.", 404)
	}
}

// NEM APIを中継する処理
func TransferApi(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	path := r.URL.Path
	query := r.URL.RawQuery
	var tmp interface{}
	uri := getUri()
	uri = uri + path + "?" + query
	byteArray, err := util.Request(method, uri)
	if err != nil {
		fmt.Println("err:", err)
	}
	fmt.Println(uri)
	if byteArray == nil {
		fmt.Println("byteArray nil")
	}
	if err := json.Unmarshal(byteArray, &tmp); err != nil {
		fmt.Println("err:", err)
	}
	res, _ := json.Marshal(tmp)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

// NEM APIを中継する処理
func GetPeerList(w http.ResponseWriter, r *http.Request) {
	res, _ := json.Marshal(peerList)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

// ヘルスチェック用の処理
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

// ヘルスチェック用の処理
func GetMultiNode() {
	maxHeight = int64(0)

	// WaitGroupの値を取る
	wg := &sync.WaitGroup{}
	c := make(chan int, 20)
	for idx, a := range peerList.Active {
		wg.Add(1)
		go func(num int, a *Node) {
			c <- 1
			process(num, a)
			defer func() {
				<-c
				wg.Done()
			}()
		}(idx, a)
	}
	wg.Wait()
	// 高さチェック
	for _, a := range peerList.Active {
		if a.Height < maxHeight-10 {
			a.Active = false
			e := a.Endpoint
			uri := e.Protocol + "://" + e.Host + ":" + strconv.Itoa(e.Port) + "/chain/height"
			fmt.Println("低すぎ:", uri)
			continue
		}
	}
	now, timeStr := util.GetNowTime()
	peerList.Update = now
	peerList.UpdateStr = timeStr
}

// プライベート関数
func getUri() string {
	if MODE == 1 {
		// シングルモード
		return FIRST_URL
	} else {
		// マルチモード
		my_rand := rand.New(rand.NewSource(1))
		my_rand.Seed(time.Now().UnixNano())
		n := len(peerList.Active)
		j := my_rand.Intn(n)
		e := peerList.Active[j].Endpoint
		uri := e.Protocol + "://" + e.Host + ":" + strconv.Itoa(e.Port)
		return uri
	}
}

func process(idx int, a *Node) {
	e := a.Endpoint
	uri := e.Protocol + "://" + e.Host + ":" + strconv.Itoa(e.Port) + "/chain/height"
	fmt.Println("uri:", uri)
	byteArray, err := util.Request("GET", uri)
	if err != nil {
		fmt.Println("応答なし:", uri)
		peerList.Active[idx].Active = false
	} else {
		h := &Height{0}
		peerList.Active[idx].Height = h.Height
		if err := json.Unmarshal(byteArray, &h); err != nil {
			fmt.Println("応答なし:", uri)
			peerList.Active[idx].Active = false
		} else {
			fmt.Println("height:", h.Height)
			peerList.Active[idx].Active = true
			peerList.Active[idx].Height = h.Height
			if h.Height >= maxHeight {
				maxHeight = h.Height
			}
		}
	}
}
