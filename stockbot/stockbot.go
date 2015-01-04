package main 


import (
	"flag"
	"github.com/ChimeraCoder/anaconda"
	"github.com/garyburd/go-oauth/oauth"
	tl "github.com/nyanshak/twitterlib"
	"github.com/nyanshak/go-markit/markit"
	"log"
	"net/http"
	"time"
	"os"
	"strings"
	"io/ioutil"
	"regexp"
	"strconv"
)

var (
	api *anaconda.TwitterApi
	myOauth       *oauth.Client
	myCredentials *oauth.Credentials

	consumerKey		= os.Getenv("STOCK_CONSUMER_KEY")
	consumerSecret	= os.Getenv("STOCK_CONSUMER_SECRET")
	accessKey		= os.Getenv("STOCK_ACCESS_KEY")
	accessSecret	= os.Getenv("STOCK_ACCESS_SECRET")

	screenName		= os.Getenv("STOCK_NAME")
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

	go quoter()
}

func quoter() {

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
				quotes := getQuotesFromTweet(*tweet)

				for _, v := range quotes {
					api.PostTweet(v, nil)
				}
			}
		} else {
			log.Fatalln("decoding tweet failed: %s", err)
		}
	}
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}

func getStringFromQuote(q markit.Quote) string {
	last := floatToString(q.LastPrice)
	changePercent := floatToString(q.ChangePercent)
	high := floatToString(q.High)
	low := floatToString(q.Low)
	openPrice := floatToString(q.Open)

	return q.Symbol + ": " + last + "(" + changePercent + "%) // High: " +
			high + " // Low: " + low + " // Opened at " +
			openPrice + " // #stockQuoted at " + q.Timestamp
}

func getQuotesFromTweet(tweet anaconda.Tweet) []string {
	r, _ := regexp.Compile("q:[0-9A-Za-z_]+(,[0-9A-Za-z_]+)*")

	requestedQuotes := strings.Split(strings.Replace(r.FindString(tweet.Text), "q:", "", 1), ",")

	quotes := []string{}
	for _, v := range requestedQuotes {
		quote, err := markit.GetQuote(v)

		// no company exists with that symbol, so try to lookup companies by name instead
		if err != nil {
			if strings.HasPrefix(err.Error(), "No symbol matches found") {
				companies, err := markit.Lookup(v)

				if err != nil {
					continue
				}

				for _, company := range companies {
					q, err := markit.GetQuote(company.Symbol)

					if err != nil {
						continue
					}
					quotes = append(quotes, getStringFromQuote(*q))
				}

			}
		} else {
			quotes = append(quotes, getStringFromQuote(*quote))
		}

	}

	return quotes
}

func main() {
	err := http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
}
