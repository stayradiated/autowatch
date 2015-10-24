package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stayradiated/slacker"
	"log"
	"net/http"
	"time"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	urls := viper.GetStringMapString("urls")
	username := viper.GetString("auth.username")
	password := viper.GetString("auth.password")

	for title, url := range urls {
		c := Checker{
			Title:    title,
			URL:      url,
			Username: username,
			Password: password,
		}

		go func() {
			for {
				if err := c.Check(); err != nil {
					log.Println(err)
				}
				time.Sleep(time.Second * 5)
			}
		}()
	}

	select {}
}

type Checker struct {
	Title    string
	URL      string
	Info     *Info
	Username string
	Password string
}

type Info struct {
	Version string
	Branch  string
}

func (c *Checker) Check() error {
	log.Println("Checking", c.Title)

	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		return err
	}

	// set auth
	if c.Username != "" || c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body := &Info{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	if c.Info == nil {
		fmt.Println("Initial version", body.Version)
		c.Info = body
		return nil
	}

	before := c.Info
	after := body

	oldVersion := before.Version
	newVersion := after.Version

	log.Println(oldVersion, newVersion)

	if oldVersion != newVersion {
		c.Info = body
		if err := c.Notify(); err != nil {
			panic(err)
		}
		time.Sleep(time.Second * 60)
	}

	return nil
}

func (c *Checker) Notify() error {
	message := fmt.Sprintf("%s is now running version %s from branch '%s'.", c.Title, c.Info.Version, c.Info.Branch)

	bot := &slacker.Slacker{
		URL:      viper.GetString("slack.url"),
		Icon:     viper.GetString("slack.icon"),
		Username: viper.GetString("slack.username"),
	}

	return bot.Send(message)
}
