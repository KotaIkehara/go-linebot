package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// ハンドラの登録
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/callback", lineHandler)

	fmt.Println("http://localhost:8080 で起動中...")
	// HTTPサーバを起動
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Hello World"
	fmt.Fprintf(w, msg)
}

func lineHandler(w http.ResponseWriter, r *http.Request) {
	// BOTを初期化
	bot, err := linebot.New(
		os.Getenv("LINE_SECRET"),
		os.Getenv("LINE_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// リクエストからBOTのイベントを取得
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	for _, event := range events {
		// イベントがメッセージの受信だった場合
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			// メッセージがテキスト形式の場合
			case *linebot.TextMessage:
				replyMessage := message.Text
				_, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do()
				if err != nil {
					log.Print(err)
				}
			//メッセージが位置情報の場合
			case *linebot.LocationMessage:
				sendRestoInfo(bot, event)
			}
			// 他にもスタンプや画像、位置情報など色々受信可能
		}
	}
}

func sendRestoInfo(bot *linebot.Client, e *linebot.Event) {
	// 位置情報メッセージの取得
	msg := e.Message.(*linebot.LocationMessage)

	//strconv.FormatFloatで値を文字列に変換する
	lat := strconv.FormatFloat(msg.Latitude, 'f', 2, 64)
	lng := strconv.FormatFloat(msg.Longitude, 'f', 2, 64)

	replyMsg := getRestoInfo(lat, lng)

	res := linebot.NewTemplateMessage(
		"レストラン一覧",
		linebot.NewCarouselTemplate(replyMsg...).WithImageOptions("rectangle", "cover"),
	)

	_, err := bot.ReplyMessage(e.ReplyToken, res).Do()
	if err != nil {
		log.Print(err)
	}
}

//レスポンスの構造体定義
type response struct {
	Results results `json:"results"`
}
type results struct {
	Shop []shop `json:"shop"`
}
type shop struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Photo   photo  `json:"photo"`
	URLS    urls   `json:"urls"`
}

type photo struct {
	Mobile mobile `json:"mobile"`
}

type mobile struct {
	L string `json:"l"`
}

type urls struct {
	PC string `json:"pc"`
}

func getRestoInfo(lat string, lng string) []*linebot.CarouselColumn {
	url := fmt.Sprintf(
		"https://webservice.recruit.co.jp/hotpepper/gourmet/v1/?format=json&key=%s&lat=%s&lng=%s",
		os.Getenv("HOTPEPPER_API_KEY"), lat, lng)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var data response
	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatal(err)
	}

	var ccs []*linebot.CarouselColumn
	for _, shop := range data.Results.Shop {
		addr := shop.Address
		if 60 < utf8.RuneCountInString(addr) {
			addr = string([]rune(addr)[:60])
		}

		cc := linebot.NewCarouselColumn(
			shop.Photo.Mobile.L,
			shop.Name,
			addr,
			linebot.NewURIAction("ホットペッパーで開く", shop.URLS.PC),
		).WithImageOptions("#FFFFFF")
		ccs = append(ccs, cc)
	}
	return ccs
}
