package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	token := os.Getenv("TOKEN")

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		log.Printf("[%s] %s - start", update.Message.From.UserName, update.Message.Text)

		go func(update tgbotapi.Update) {
			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi, i'm a ukr news bot.")
				_, err := bot.Send(msg)
				if err != nil {
					log.Fatal(err)
				}
			case "/all":
				if response, err := http.Get("https://www.ukr.net/ajax/news.json"); err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
					_, err := bot.Send(msg)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					defer func() {
						err := response.Body.Close()
						if err != nil {
							log.Fatal(err)
						}
					}()
					contents, _ := ioutil.ReadAll(response.Body)
					rr := &RequestResults{}
					if err = json.Unmarshal([]byte(contents), rr); err != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
						_, err := bot.Send(msg)
						if err != nil {
							log.Fatal(err)
						}
					}
					message := ""
					for _, v := range rr.Results {
						message = fmt.Sprintf("%s*%s*\n", message, v.Title)

						for _, v := range v.Items {
							message = fmt.Sprintf("%s- %s. [More](%s).\n", message, v.Title, v.URL)
						}
						message = message + "\n"
					}
					message = strings.TrimSuffix(message, "\n")
					message = message + "\nRetry /all or /sumy"
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
					msg.ParseMode = "markdown"
					msg.DisableWebPagePreview = true
					_, err := bot.Send(msg)
					if err != nil {
						log.Fatal(err)
					}
					log.Printf("[%s] %s - sent", update.Message.From.UserName, update.Message.Text)
				}
			case "/sumy":
				if response, err := http.Get("https://www.ukr.net/ajax/regions.json?snr=18"); err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
					_, err := bot.Send(msg)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					defer func() {
						err := response.Body.Close()
						if err != nil {
							log.Fatal(err)
						}
					}()
					contents, _ := ioutil.ReadAll(response.Body)
					rr := &RegionResult{}
					if err = json.Unmarshal([]byte(contents), rr); err != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
						_, err := bot.Send(msg)
						if err != nil {
							log.Fatal(err)
						}
					}
					message := "*Sumy news*\n"
					for _, v := range rr.Items {
						message = fmt.Sprintf("%s- %s. [More](%s).\n", message, v.Title, v.URL)
					}
					message = message + "\nRetry /all or /sumy"
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
					msg.ParseMode = "markdown"
					msg.DisableWebPagePreview = true
					_, err := bot.Send(msg)
					if err != nil {
						log.Fatal(err)
					}
					log.Printf("[%s] %s - sent", update.Message.From.UserName, update.Message.Text)
				}
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
				_, err := bot.Send(msg)
				if err != nil {
					log.Fatal(err)
				}
			}
		}(update)
		log.Printf("[%s] %s - end", update.Message.From.UserName, update.Message.Text)
	}
}

type Item struct {
	Title, URL string
}

type Result struct {
	Title string
	Items []Item
}

type RequestResults struct {
	Results []Result
}

type RegionResult struct {
	Items []Item
}

func (rr *RequestResults) UnmarshalJSON(bs []byte) error {
	responseMap := make(map[string]interface{})
	if err := json.Unmarshal(bs, &responseMap); err != nil {
		return err
	}

	for _, v := range responseMap["news"].([]interface{}) {
		aNew := v.(map[string]interface{})
		switch aNew["id"].(float64) {
		case 13, 7, 22, 4:
			items := []Item{}
			for _, v := range aNew["items"].([]interface{}) {
				itemMap := v.(map[string]interface{})
				items = append(items, Item{itemMap["title"].(string), itemMap["url"].(string)})

			}
			rr.Results = append(rr.Results, Result{aNew["title"].(string), items})
		}
	}

	return nil
}

func (rr *RegionResult) UnmarshalJSON(bs []byte) error {
	responseMap := make(map[string]interface{})
	if err := json.Unmarshal(bs, &responseMap); err != nil {
		return err
	}

	for _, v := range responseMap["region"].([]interface{}) {
		itemMap := v.(map[string]interface{})
		rr.Items = append(rr.Items, Item{itemMap["title"].(string), itemMap["url"].(string)})

	}

	return nil
}
