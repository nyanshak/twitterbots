package main 

import (
	"flag"
	"github.com/ChimeraCoder/anaconda"
	"github.com/garyburd/go-oauth/oauth"
	tl "github.com/nyanshak/twitterlib"
	"log"
	"net/http"
	"time"
	"os"
	"strings"
	"io/ioutil"
	"strconv"
)

var (
	api *anaconda.TwitterApi
	myOauth       *oauth.Client
	myCredentials *oauth.Credentials

	consumerKey		= os.Getenv("TWEET_FIT_CONSUMER_KEY")
	consumerSecret	= os.Getenv("TWEET_FIT_CONSUMER_SECRET")
	accessKey		= os.Getenv("TWEET_FIT_ACCESS_KEY")
	accessSecret	= os.Getenv("TWEET_FIT_ACCESS_SECRET")

	screenName		= os.Getenv("TWEET_FIT_NAME")
	listen			= flag.String("listen", ":8080", "Spec to listen on")
)

func init() {
	flag.Parse()

	if consumerKey == "" || consumerSecret == "" || accessKey == "" || accessSecret == "" {
		log.Fatalln("Credentials invalid: at least one is empty")
	}

	if screenName == "" {
		log.Fatalln("bot username left blank")
	}

	anaconda.SetConsumerKey(consumerKey)
	anaconda.SetConsumerSecret(consumerSecret)
	api = anaconda.NewTwitterApi(accessKey, accessSecret)

	myOauth = &oauth.Client{
		Credentials: oauth.Credentials{
			Token: consumerKey,
			Secret: consumerSecret,
		},
	}

	myCredentials = &oauth.Credentials{
		Token: accessKey,
		Secret: accessSecret,
	}

	go measureStats()
}

func measureStats() {

	req, err := http.NewRequest("POST", "https://stream.twitter.com/1.1/statuses/filter.json?track=fitstats_en_us", nil)
	if err != nil {
		log.Fatalf("Failed to create stream: %s\n", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", myOauth.AuthorizationHeader(myCredentials, "GET", req.URL, nil))

	conn := tl.NewConnection(0 * time.Second)

	resp, err := conn.Client.Do(req)

	if err != nil {
		log.Fatalf("Error getting stream: %s\n", err)
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		log.Fatalf("Error getting stream (%d): %s\n", resp.StatusCode, body)
	}

	conn.Setup(resp.Body)

	for {
		if tweet, err := conn.Next(); err == nil {
			// avoid having bot infinitely retweet itself
			if !strings.EqualFold(tweet.User.ScreenName, screenName) {
				log.Println(tweet)
			}
		} else {
			log.Fatalf("decoding tweet failed: %s\n", err)
		}
	}
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}

func main() {
	err := http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatalf("failed to listen: %s\n", err)
	}
}
