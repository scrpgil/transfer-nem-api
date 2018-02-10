# transfer-nem-api

NIS APIへリクエストを中継するサーバーです。  
GAE/Goで書かれています。  

[NISによってAPI呼び出しの結果が古い場合について](https://goo.gl/DSJWT6)


## 使い方

### デモ

nemofolioで使っています。

[nemfolio](https://nemfolio.net)


### インストール

````
git clone https://github.com/scrpgil/transfer-nem-api.git 
cd transfer-nem-api
go run *.go
````


### GAEへのデプロイ

以下のコマンドを実行してください。

````
gcloud app deploy
````

gcloudの設定は別途やっておく必要があります。公式ドキュメントがおすすめです。

[Google App Engine Go Standard Environment ドキュメント](https://cloud.google.com/appengine/docs/standard/go/?hl=ja)

### 設定

````
const (
	MODE      = 2                          // 1:シングルモード, 2:マルチモード
	FIRST_URL = "http://go.nem.ninja:7890" //最初にpeerlistを取得しにいくURLです
	HOUR      = 8                          // 更新間隔
	PORT_MODE = false                      // ポートによって処理を分けるか？
	THRESHOLD = 10                         // MAXのブロック高さとのしきい値
)
````

定数で処理を分けています。

- MODE

	シングルモードだと、FIRST_URLに対してのみ通信を行います。  
	マルチモードだと「/node/peer-list/all」で取得したノードに対してランダムで通信します。  

- FIRST_URL

	シングルモードやマルチモードの「/node/peer-list/all」を取得するNISのURLです。  

- HOUR

	マルチモードでActiveなNISを更新する間隔です。デフォルトの8時間は適当です。  

- PORT_MODE

	中継方法をポートで分けるか選択できます。  
	trueだと7890で受信したリクエストをNISへ中継します。  
	falseだと、「/」以外の通信をNISへ中継します。  

- THRESHOLD

	マルチモード時に最大チェーンの高さとどれくらいの差異を許容するかの変数です。デフォルトは10にしています。  


### マルチモード時の起動について

「go run *.go」をしてから起動するまで大体30秒くらいです。各ノードの最大チェーンを取得する処理は20リクエストを並行に走らせています。  


### 各ノードの最大チェーン取得結果について

以下リンクより各ノードの最大チェーン取得結果を返却します。内容は「/node/peer-list/all」の取得結果に更新日時、最大チェーン、各ノードの最大チェーンのメンバを付け加えたものです。

[https://transfer.nemfolio.net/]

````json
{
  "update": "更新日時（例：2018-02-10T10:47:32.088503257+09:00）",
  "update_str": "更新日時別フォーマット（例：10:47:32 2018-02-10）",
  "max_height": 最大チェーン数（例：1497099）,
  "inactive":[非アクティブノード],
  "active":[
      {
     	「/node/peer-list/all」の内容と同じ。
      },
      "height": 最大チェーン数,
      "active": 状態
    },
  ],
  "low_height":[ // max_height-THRESHOLDの値よりheightが低いノード
      {
     	「/node/peer-list/all」の内容と同じ。
      },
      "height": 最大チェーン数,
      "active": 状態
    },
  ],
````


## transfer-nem-apiについて

### 目的

以下の目的で作成しています。

- SSL通信を中継する

- NISの負荷分散がしたい 

SSL通信を中継するのは、nemfolioがPWAだからです。PWAはSSL通信前提の作りです。  
そのため非SSL通信のNISとの通信はMixedContent扱いなので、動作しなくなります。  

NISへの負荷分散がしたかったのはビビリなので、一つのNISに対してリクエスト送りすぎて迷惑だと言われたらどうしようと感じたからです。  

### 運用について

現在、nemfolio.netにて動作確認中です。


