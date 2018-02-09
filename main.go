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
	MODE      = 2                          // 1:シングルモード, 2:マルチモード
	FIRST_URL = "http://go.nem.ninja:7890" //最初にpeerlistを取得しにいくURLです
	HOUR      = 8                          // 更新間隔
	PORT_MODE = false                      // ポートによって処理を分けるか？
	THRESHOLD = 10                         // MAXのブロック高さとのしきい値
)

type PeerList struct {
	Update    time.Time `json:"update"`
	UpdateStr string    `json:"update_str"`
	MaxHeight int64     `json:"max_height"`
	Inactive  []*Node   `json:"inactive"`
	Active    []*Node   `json:"active"`
	Busy      []*Node   `json:"busy"`
	LowHeight []*Node   `json:"low_height"`
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

var peerList *PeerList
var maxHeight int64

func init() {
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
	if MODE == 2 {
		if peerList == nil {
			http.Error(w, "During startup.", 404)
			return
		}
	}
	host := r.Host
	_, port := util.URLParse(host)
	switch port {
	case "8080":
		PathCheck(w, r)
	case "7890":
		// ポート：7890ならNISへリクエストを中継
		TransferApi(w, r)
	default:
		if PORT_MODE == true {
			// それ以外はエラー
			http.Error(w, "URL Not found.", 404)
		} else {
			// ポートによって処理を分けないなら中継処理へ。GAE用
			PathCheck(w, r)
		}
	}
}

// パスによって処理を分ける
func PathCheck(w http.ResponseWriter, r *http.Request) {
	// ポート：8080なら取得したAPI情報を返却
	path := r.URL.Path
	switch path {
	case "/":
		if MODE == 2 {
			GetPeerList(w, r)
		} else {
			http.Error(w, "Not found.", 404)
		}
	default:
		// デフォルトならNISへリクエストを中継。GAEで7890ポートの解放ができなかったため
		TransferApi(w, r)
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
	uri := FIRST_URL + "/node/peer-list/all"
	byteArray, _ := util.Request("GET", uri)
	tmpPeerList := &PeerList{}
	_ = json.Unmarshal(byteArray, tmpPeerList)
	maxHeight = int64(0)

	// WaitGroupの値を取る
	wg := &sync.WaitGroup{}
	c := make(chan int, 20)
	for idx, a := range tmpPeerList.Active {
		wg.Add(1)
		go func(num int, a *Node, p *PeerList) {
			c <- 1
			process(num, a, p)
			defer func() {
				<-c
				wg.Done()
			}()
		}(idx, a, tmpPeerList)
	}
	wg.Wait()
	// 高さチェック
	newActive := []*Node{}
	lowHeight := []*Node{}
	for _, a := range tmpPeerList.Active {
		if a.Height < maxHeight-THRESHOLD {
			a.Active = false
			e := a.Endpoint
			uri := e.Protocol + "://" + e.Host + ":" + strconv.Itoa(e.Port) + "/chain/height"
			fmt.Println("LowHeight:", uri)
			lowHeight = append(lowHeight, a)
			continue
		} else {
			newActive = append(newActive, a)
		}
	}
	tmpPeerList.Active = newActive
	tmpPeerList.LowHeight = lowHeight
	tmpPeerList.MaxHeight = maxHeight

	now, timeStr := util.GetNowTime()
	tmpPeerList.Update = now
	tmpPeerList.UpdateStr = timeStr
	peerList = tmpPeerList
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

func process(idx int, a *Node, p *PeerList) {
	e := a.Endpoint
	uri := e.Protocol + "://" + e.Host + ":" + strconv.Itoa(e.Port) + "/chain/height"
	fmt.Println("uri:", uri)
	byteArray, err := util.Request("GET", uri)
	if err != nil {
		fmt.Println("応答なし:", uri)
		p.Active[idx].Active = false
	} else {
		h := &Height{0}
		p.Active[idx].Height = h.Height
		if err := json.Unmarshal(byteArray, &h); err != nil {
			fmt.Println("応答なし:", uri)
			p.Active[idx].Active = false
		} else {
			fmt.Println("height:", h.Height)
			p.Active[idx].Active = true
			p.Active[idx].Height = h.Height
			if h.Height >= maxHeight {
				maxHeight = h.Height
			}
		}
	}
}
