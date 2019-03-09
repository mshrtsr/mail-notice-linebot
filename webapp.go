package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"

	"./mailmanager"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
)

// LineUser ...
type LineUser struct {
	LineID            string
	LineName          string
	RegisteredAddress []string
}

// MailObject ..
type MailObject struct {
	TargetLineID string
	MailSubject  []string
}

func main() {
	if len(os.Getenv("DOTENV_LOADED")) < 1 {
		DotEnvLoad()
	}

	//mailCheck()
	//sendVerificationMail("Test User", os.Getenv("IMAP_AUTH_USER"), time.Now().String())

	port := os.Getenv("PORT")
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// DotEnvLoad load .env file
func DotEnvLoad() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("DotEnv:", err)
	}
}

func mailCheck() {
	mboxName := "inbox"
	//messages := mailmanager.PopMailByUID(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1), mboxName, os.Getenv("IMAP_SERVER_NAME"), os.Getenv("IMAP_AUTH_USER"), os.Getenv("IMAP_AUTH_PASSWORD"))
	messages := mailmanager.PopMail(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1), mboxName, os.Getenv("IMAP_SERVER_NAME"), os.Getenv("IMAP_AUTH_USER"), os.Getenv("IMAP_AUTH_PASSWORD"))
	log.Println("fetched messages: ", len(messages))
	for _, msg := range messages {
		log.Println(msg.Envelope.Date.String() + ":" + msg.Envelope.Subject)
	}
	if len(messages) > 0 {
		lineUser := []LineUser{}

		mailObjects := ConvertMessagesToMailObjectByLineUser(messages, lineUser)
		log.Println(mailObjects)

		//sendPushNotification(targetUserId, Mail)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

	//lineChannelID := os.Getenv("LINE_CHANNEL_ID")
	lineChannelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	lineAccessToken := os.Getenv("LINE_ACCESS_TOKEN")

	bot, err := linebot.New(lineChannelSecret, lineAccessToken)

	events, err := bot.ParseRequest(r)
	if err != nil {
		// Do something when something bad happened.
		log.Print("ParseRequest: ", err)
		w.WriteHeader(400)
		return
	}

	for _, event := range events {

		// var userID string
		// var groupID string
		// var RoomID string
		var targetID string

		log.Print("EventSource Type: ", event.Source.Type)
		switch event.Source.Type {
		case linebot.EventSourceTypeUser:
			//userID = event.Source.UserID
			targetID = event.Source.UserID
		case linebot.EventSourceTypeGroup:
			//groupID = event.Source.GroupID
			targetID = event.Source.GroupID
		case linebot.EventSourceTypeRoom:
			//RoomID = event.Source.RoomID
			targetID = event.Source.RoomID
		}
		log.Print("TargetID: ", targetID)

		eventSourceType := event.Source.Type
		replyToken := event.ReplyToken

		log.Print("Event Type: ", event.Type)
		switch event.Type {
		case linebot.EventTypeMessage:
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				switch {
				case strings.Contains(message.Text, "メールお知らせ"):
					fallthrough
				case strings.Contains(message.Text, "メールおしらせ"):
					sendConfirmSetupForwarding(bot, replyToken)
				case strings.Contains(message.Text, "お知らせ解除"):
					sendConfirmRevokeForwarding(bot, replyToken)
				}

			}
			if eventSourceType == linebot.EventSourceTypeUser {
				sendRandomReply(bot, replyToken)
			}

		case linebot.EventTypeFollow:
			// Send Introduction to user
			sendIntroduction(bot, replyToken)
		case linebot.EventTypeUnfollow:
			// TODO: Delete User from database
		case linebot.EventTypeJoin:
			// Send Introduction to the group
			sendIntroduction(bot, replyToken)
		case linebot.EventTypeLeave:
			// TODO: Delete group from database
		case linebot.EventTypeMemberJoined:
			// Send message to Joined User
			// Default send nothing
		case linebot.EventTypeMemberLeft:
			// Send message to Left User
			// Default send nothing
		case linebot.EventTypePostback:
			// Do Nothing
		case linebot.EventTypeBeacon:
			// Do Nothing
		default:
			// Do Nothing
		}
	}
}

func sendConfirmSetupForwarding(bot *linebot.Client, replyToken string) {
	// Send Current registered addres and confirm resetting
	var messages []linebot.SendingMessage

	addresses := []string{"a@a.a", "b@b.b", "c@c.c"}

	// Current e-mail addresses
	var textContents = "こんにちは！メールお知らせくんです。\n"
	if len(addresses) > 0 {
		textContents = textContents + "現在お知らせ設定されているメールアドレスは\n" + strings.Join(addresses, "\n") + "\nです"
	} else {
		textContents = textContents + "現在お知らせ設定されているメールアドレスはありません"
	}
	messages = append(messages, linebot.NewTextMessage(textContents))

	// Confirm template message
	var altText string
	if len(addresses) > 0 {
		altText = "メールお知らせを再設定しますか？"
	} else {
		altText = "メールお知らせを設定しますか？"
	}
	leftBtn := linebot.NewPostbackAction("はい", "setup=true", "", "はい")
	rightBtn := linebot.NewPostbackAction("いいえ", "setup=false", "", "いいえ")
	template := linebot.NewConfirmTemplate(altText, leftBtn, rightBtn)
	messages = append(messages, linebot.NewTemplateMessage(altText, template))

	// Send messages
	if _, err := bot.ReplyMessage(replyToken, messages...).Do(); err != nil {
		log.Print(err)
	}
}

func sendConfirmRevokeForwarding(bot *linebot.Client, replyToken string) {
	// Send Current registered addres and confirm resetting
	var messages []linebot.SendingMessage

	addresses := []string{"a@a.a", "b@b.b", "c@c.c"}

	// Current e-mail addresses
	var textContents = "こんにちは！メールお知らせくんです。\n"
	if len(addresses) > 0 {
		textContents = textContents + "現在お知らせ設定されているメールアドレスは\n" + strings.Join(addresses, "\n") + "\nです"
	} else {
		textContents = textContents + "現在お知らせ設定されているメールアドレスはありません"
	}
	messages = append(messages, linebot.NewTextMessage(textContents))

	if len(addresses) <= 0 {
		return
	}

	// Confirm template message
	altText := "メールお知らせを解除しますか？"
	leftBtn := linebot.NewPostbackAction("はい", "revoke=true", "", "はい")
	rightBtn := linebot.NewPostbackAction("いいえ", "revoke=false", "", "いいえ")
	template := linebot.NewConfirmTemplate(altText, leftBtn, rightBtn)
	messages = append(messages, linebot.NewTemplateMessage(altText, template))

	// Send messages
	if _, err := bot.ReplyMessage(replyToken, messages...).Do(); err != nil {
		log.Print(err)
	}
}

func sendRandomReply(bot *linebot.Client, replyToken string) {
	contentPatterns := []string{
		"ごめんなさい！よく分かりませんでした！",
		"「メールお知らせくん」と呼んでいただければメールお知らせ設定が確認できます",
		"「お知らせ解除」と言っていただければメールお知らせを解除できます",
		"新しいメールはたぶんありません！",
	}
	// Randomize reply
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(contentPatterns))
	message := linebot.NewTextMessage(contentPatterns[i])
	// Send messages
	if _, err := bot.ReplyMessage(replyToken, message).Do(); err != nil {
		log.Print(err)
	}
}

func sendIntroduction(bot *linebot.Client, replyToken string) {
	// Send Greeting and introduction
	var messages []linebot.SendingMessage

	// Greeting
	var textContents = "登録ありがとうございます！メールお知らせくんです。\n"
	textContents += "登録されたメールアドレスにメールが届くとお知らせします。\n"
	messages = append(messages, linebot.NewTextMessage(textContents))

	// Confirm template message
	altText := "メールお知らせを設定しますか？"
	leftBtn := linebot.NewPostbackAction("はい", "setup=true", "", "はい")
	rightBtn := linebot.NewPostbackAction("いいえ", "setup=false", "", "いいえ")
	template := linebot.NewConfirmTemplate(altText, leftBtn, rightBtn)
	messages = append(messages, linebot.NewTemplateMessage(altText, template))

	// Send messages
	if _, err := bot.ReplyMessage(replyToken, messages...).Do(); err != nil {
		log.Print(err)
	}
}

func sendVerificationMail(userName, userAddress, verificationKey string) {
	from := mail.Address{Name: os.Getenv("SENDER_USERNAME"), Address: os.Getenv("SENDER_ADDRESS")}
	to := mail.Address{Name: userName, Address: userAddress}
	subject := "LINEBOT: メールお知らせくん登録確認"
	body := "この度はメールお知らせくんのご利用ありがとうございます。\n LINEの戻って以下の確認コードを送信してください。\n 確認コード：" + verificationKey
	smptServerName := os.Getenv("SMTP_SERVER_NAME")
	smtpAuthUser := os.Getenv("SMTP_AUTH_USER")
	smtpAuthPassword := os.Getenv("SMTP_AUTH_PASSWORD")
	mailmanager.SendMail(from, to, subject, body, smptServerName, smtpAuthUser, smtpAuthPassword)
}

func sendPushNotification(targetID, textContents string) {
	//lineChannelID := os.Getenv("LINE_CHANNEL_ID")
	lineChannelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	lineAccessToken := os.Getenv("LINE_ACCESS_TOKEN")

	bot, err := linebot.New(lineChannelSecret, lineAccessToken)
	if err != nil {
		log.Print(err)
	}

	if _, err := bot.PushMessage(targetID, linebot.NewTextMessage(textContents)).Do(); err != nil {
		log.Print(err)
	}
}

// ConvertMessagesToMailObjectByLineUser ..
func ConvertMessagesToMailObjectByLineUser(messages []imap.Message, lineUsers []LineUser) []MailObject {
	var mailObjects []MailObject
	for _, msg := range messages {
		for _, lineUser := range lineUsers {
			log.Println(lineUser, msg)

		}
	}

	return mailObjects
}
