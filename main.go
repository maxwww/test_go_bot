package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var Keyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Головні новини"),
		tgbotapi.NewKeyboardButton("Новини Сумщини"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Погода"),
		tgbotapi.NewKeyboardButton("Налаштування"),
	),
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	token := os.Getenv("TOKEN")
	pgUser := os.Getenv("PG_USER")
	pgBasename := os.Getenv("PG_BASENAME")
	pgPassword := os.Getenv("PG_PASSWORD")
	pgHost := os.Getenv("PG_HOST")

	connStr := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable", pgHost, pgBasename, pgUser, pgPassword)
	db, err := sql.Open("postgres", connStr)
	defer func() {
		err := db.Close()
		if err != nil {
			log.Print(err)
		}
	}()
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users
	(
		id  int UNIQUE PRIMARY KEY,
		is_bot BOOLEAN NOT NULL,
		first_name VARCHAR(250) NOT NULL,
		last_name VARCHAR(250),
		user_name VARCHAR(250),
		language_code VARCHAR(250),
		requests int NOT NULL
	)
`)
	if err != nil {
		log.Fatal(err)
	}
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
		log.Printf("[%s] %v - start", update.Message.From.UserName, update.Message.Text)

		go func(update tgbotapi.Update) {
			var exists bool
			row := db.QueryRow("SELECT EXISTS (SELECT id FROM users WHERE id = $1)", update.Message.From.ID)
			err := row.Scan(&exists)
			if err != nil {
				if err == sql.ErrNoRows {
					_, err = db.Exec(`
INSERT INTO users (id, is_bot, first_name, last_name, user_name, language_code, requests )
VALUES ($1, $2, $3, $4, $5, $6, $7)`, update.Message.From.ID, update.Message.From.IsBot, update.Message.From.FirstName, update.Message.From.LastName, update.Message.From.UserName, update.Message.From.LanguageCode, 1)
					if err != nil {
						log.Print(err)
					}
				} else {
					log.Print(err)
				}
			} else {
				_, err = db.Exec(`
UPDATE users
SET requests = requests + 1
WHERE id = $1;`, update.Message.From.ID)
				if err != nil {
					log.Print(err)
				}
			}

			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi, i'm a ukr news bot.")
				_, err := bot.Send(msg)
				if err != nil {
					log.Print(err)
				}
			case "Головні новини":
				if response, err := http.Get("https://www.ukr.net/ajax/news.json"); err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
					_, err := bot.Send(msg)
					if err != nil {
						log.Print(err)
					}
				} else {
					defer func() {
						err := response.Body.Close()
						if err != nil {
							log.Print(err)
						}
					}()
					contents, _ := ioutil.ReadAll(response.Body)
					rr := &RequestResults{}
					if err = json.Unmarshal([]byte(contents), rr); err != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
						_, err := bot.Send(msg)
						if err != nil {
							log.Print(err)
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
					message = strings.TrimSuffix(message, "\n")
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
					msg.ParseMode = "markdown"
					msg.DisableWebPagePreview = true
					msg.ReplyMarkup = Keyboard
					_, err := bot.Send(msg)
					if err != nil {
						log.Print(err)
					}
					log.Printf("[%s] %s - sent", update.Message.From.UserName, update.Message.Text)
				}
			case "Новини Сумщини":
				if response, err := http.Get("https://www.ukr.net/ajax/regions.json?snr=18"); err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
					_, err := bot.Send(msg)
					if err != nil {
						log.Print(err)
					}
				} else {
					defer func() {
						err := response.Body.Close()
						if err != nil {
							log.Print(err)
						}
					}()
					contents, _ := ioutil.ReadAll(response.Body)
					rr := &RegionResult{}
					if err = json.Unmarshal([]byte(contents), rr); err != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
						_, err := bot.Send(msg)
						if err != nil {
							log.Print(err)
						}
					}
					message := "*Новини Сумщини*\n"
					for _, v := range rr.Items {
						message = fmt.Sprintf("%s- %s. [More](%s).\n", message, v.Title, v.URL)
					}
					message = strings.TrimSuffix(message, "\n")
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
					msg.ParseMode = "markdown"
					msg.DisableWebPagePreview = true
					msg.ReplyMarkup = Keyboard
					_, err := bot.Send(msg)
					if err != nil {
						log.Print(err)
					}
					log.Printf("[%s] %s - sent", update.Message.From.UserName, update.Message.Text)
				}
			case "Погода":
				api_key := os.Getenv("WEATHER_TOKEN")
				if response, err := http.Get("http://api.openweathermap.org/data/2.5/forecast?q=sumy&units=metric&lang=ua&appid=" + api_key); err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
					_, err := bot.Send(msg)
					if err != nil {
						log.Print(err)
					}
				} else {
					defer func() {
						err := response.Body.Close()
						if err != nil {
							log.Print(err)
						}
					}()
					contents, _ := ioutil.ReadAll(response.Body)
					WeatherResult := &WeatherForecastResult{}
					if err = json.Unmarshal([]byte(contents), WeatherResult); err != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Something going wrong, try to change your question")
						_, err := bot.Send(msg)
						if err != nil {
							log.Print(err)
						}
					}
					message := "*Погода в Сумах*\n"
					day := ""
					for _, v := range WeatherResult.List {
						tm := time.Unix(int64(v.Dt), int64(WeatherResult.City.Timezone)*1e9)
						rDay := tm.Format("02 January 2006")
						if day != rDay {
							day = rDay
							message = fmt.Sprintf("%s*%s*\n", message, day)
						}
						message = fmt.Sprintf("%s%s *%+d*℃ %s %s %s\n", message, tm.Format("15:04"), int64(v.Main.Temp), Icons[v.Weather[0].Icon], v.Weather[0].Description, GetWindIcon(&v.Wind))

					}
					message = strings.TrimSuffix(message, "\n")
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
					msg.ParseMode = "markdown"
					msg.DisableWebPagePreview = true
					msg.ReplyMarkup = Keyboard
					_, err := bot.Send(msg)
					if err != nil {
						log.Print(err)
					}
					log.Printf("[%s] %s - sent", update.Message.From.UserName, update.Message.Text)
				}
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, Icons["01d"])
				msg.ReplyMarkup = Keyboard
				_, err := bot.Send(msg)
				if err != nil {
					log.Print(err)
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

type Wind struct {
	Speed float64 `json:"speed"`
	Deg   float64 `json:"deg"`
}

type WeatherForecastResult struct {
	City struct {
		Name     string  `json:"name"`
		Timezone float64 `json:"timezone"`
	} `json:"city"`
	List []struct {
		Dt   float64 `json:"Dt"`
		Main struct {
			Temp       float64 `json:"temp"`
			Temp_min   float64 `json:"temp_min"`
			Temp_max   float64 `json:"temp_max"`
			Pressure   float64 `json:"pressure"`
			Sea_level  float64 `json:"sea_level"`
			Grnd_level float64 `json:"grnd_level"`
			Humidity   float64 `json:"humidity"`
			Temp_kf    float64 `json:"temp_kf"`
		} `json:"main"`
		Weather []struct {
			Id          float64 `json:"id"`
			Main        string  `json:"main"`
			Description string  `json:"description"`
			Icon        string  `json:"icon"`
		} `json:"weather"`
		Wind Wind `json:"wind"`
	} `json:"list"`
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

var Icons = map[string]string{
	"01d": "☀️",
	"01n": "🌙",
	"02d": "🌤 ️",
	"02n": "🌙",
	"03d": "🌥 ",
	"03n": "☁️",
	"04d": "☁️",
	"04n": "☁️",
	"09d": "🌧 ️",
	"09n": "🌧 ️",
	"10d": "🌦 ",
	"10n": "🌧 ️",
	"11d": "🌩 ",
	"11n": "🌩 ",
	"13d": "❄️",
	"13n": "❄️",
	"50d": "🌫 ",
	"50n": "🌫 ",
}

func GetWindIcon(w *Wind) string {
	direction := "⬇"
	switch {
	case w.Deg >= 22.5 && w.Deg < 67.5:
		direction = "↙"
	case w.Deg >= 67.5 && w.Deg < 112.5:
		direction = "️⬅️"
	case w.Deg >= 112.5 && w.Deg < 157.5:
		direction = "️↖"
	case w.Deg >= 157.5 && w.Deg < 202.5:
		direction = "⬆"
	case w.Deg >= 202.5 && w.Deg < 247.5:
		direction = "↗"
	case w.Deg >= 247.5 && w.Deg < 292.5:
		direction = "➡️"
	case w.Deg >= 292.5 && w.Deg < 337.5:
		direction = "↘️"
	}

	return fmt.Sprintf("%s %.fм/с", direction, w.Speed)
}
