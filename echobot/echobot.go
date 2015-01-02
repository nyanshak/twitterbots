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
)

var (
	api *anaconda.TwitterApi
	myOauth       *oauth.Client
	myCredentials *oauth.Credentials

	consumerKey = os.Getenv("ECHO_CONSUMER_KEY")
	consumerSecret = os.Getenv("ECHO_CONSUMER_SECRET")
	accessKey = os.Getenv("ECHO_ACCESS_KEY")
	accessSecret = os.Getenv("ECHO_ACCESS_SECRET")

	screenName     = os.Getenv("ECHO_NAME")
	listen         = flag.String("listen", ":8080", "Spec to listen on")
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

	go echoer()
}

func echoer() {

	req, err := http.NewRequest("GET", "https://userstream.twitter.com/1.1/user.json?with=user", nil)
	if err != nil {
		log.Fatalln("Failed to create status stream: %s", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", myOauth.AuthorizationHeader(myCredentials, "GET", req.URL, nil))

	conn := tl.NewConnection(0 * time.Second)

	resp, err := conn.Client.Do(req)

	if err != nil {
		log.Fatalln("Error getting status stream: %s", err)
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		log.Fatalln("Error getting status stream (%d): %s", resp.StatusCode, body)
	}

	conn.Setup(resp.Body)

	for {
		if tweet, err := conn.Next(); err == nil {
			// avoid having bot infinitely retweet itself
			if !strings.EqualFold(tweet.User.ScreenName, screenName) {
				api.Retweet(tweet.Id, false)
			}
		} else {
			log.Fatalln("decoding tweet failed: %s", err)
		}
	}
}

func main() {
	err := http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
}
