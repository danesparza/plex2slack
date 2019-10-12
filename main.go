package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danesparza/plex2slack/data"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

//	Set up our flags
var (
	port           = flag.Int("port", 3300, "The port to listen on")
	allowedOrigins = flag.String("allowedOrigins", "*", "A comma-separated list of valid CORS origins")
	webhookURL     = flag.String("webhook", "https://hooks.slack.com/services/YOURWEBHOOKINFO", "A valid Slack webhook URL")
)

func parseEnvironment() {
	//	Check for the listen port
	if envPort := os.Getenv("PLEX2SLACK_PORT"); envPort != "" {
		*port, _ = strconv.Atoi(envPort)
	}

	//	Check for allowed origins
	if envOrigins := os.Getenv("PLEX2SLACK_ALLOWED_ORIGINS"); envOrigins != "" {
		*allowedOrigins = envOrigins
	}

	//	Check for allowed origins
	if envWebhook := os.Getenv("PLEX2SLACK_ALLOWED_ORIGINS"); envWebhook != "" {
		*webhookURL = envWebhook
	}
}

func main() {
	//	Parse environment variables:
	parseEnvironment()

	//	Parse the command line for flags:
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/plex", func(w http.ResponseWriter, req *http.Request) {

		readForm, err := req.MultipartReader()
		if err != nil {
			log.Printf("There was a problem reading the data: %v", err)
			return
		}

		for {
			part, errPart := readForm.NextPart()
			if errPart == io.EOF {
				break
			}
			if part.FormName() == "thumb" {
				partBytes, _ := ioutil.ReadAll(part)
				err := ioutil.WriteFile("thumb.jpg", partBytes, 0644)
				if err != nil {
					fmt.Printf("Error saving thumbnail: %v\n", err)
				}
			} else if part.FormName() == "payload" {
				partBytes, _ := ioutil.ReadAll(part)
				msg := data.PlexMessage{}
				if err := json.Unmarshal(partBytes, &msg); err != nil {
					panic(err)
				}

				//	First, see what kind of message it is:
				if msg.Event == "library.new" {

					//	Format our notification:
					slackMsg := data.SlackRequestBody{}

					//	Movie
					if msg.Metadata.Type == "movie" {
						slackMsg = data.SlackRequestBody{
							Blocks: []data.SlackBlock{
								data.SlackBlock{
									Type: "section",
									Text: &data.SlackText{
										Text: fmt.Sprintf("*%v*", msg.Metadata.Title),
										Type: "mrkdwn",
									},
								},
								data.SlackBlock{
									Type: "context",
									Elements: []data.SlackElement{
										data.SlackElement{
											Type: "mrkdwn",
											Text: "added to Movies",
										},
									},
								},
								data.SlackBlock{
									Type: "divider",
								},
							},
						}
					}

					//	TV show
					if msg.Metadata.Type == "episode" {
						slackMsg = data.SlackRequestBody{
							Blocks: []data.SlackBlock{
								data.SlackBlock{
									Type: "section",
									Text: &data.SlackText{
										Text: fmt.Sprintf("New episode of *%v %v*: _%v_", msg.Metadata.GrandparentTitle, msg.Metadata.ParentTitle, msg.Metadata.Title),
										Type: "mrkdwn",
									},
								},
								data.SlackBlock{
									Type: "context",
									Elements: []data.SlackElement{
										data.SlackElement{
											Type: "mrkdwn",
											Text: "added to TV shows",
										},
									},
								},
								data.SlackBlock{
									Type: "divider",
								},
							},
						}
					}

					//	Send the message to slack
					SendSlackNotification(*webhookURL, slackMsg)
				}

				//	Used for debugging:
				log.Printf("Event: %v / Type: %v / Grandparent title: %v / Parent title: %v / Title: %v \n", msg.Event, msg.Metadata.Type, msg.Metadata.GrandparentTitle, msg.Metadata.ParentTitle, msg.Metadata.Title)
			}
		}
	})

	//	CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(*allowedOrigins, ","),
		AllowCredentials: true,
	})
	handler := c.Handler(r)

	//	Indicate what port we're starting the service on
	portString := strconv.Itoa(*port)
	log.Printf("Allowed origins: %v\n", *allowedOrigins)
	log.Printf("Webhook URL: %v\n", *webhookURL)
	log.Printf("Starting server on : %v\n", portString)
	http.ListenAndServe(":"+portString, handler)
}

// SendSlackNotification will post to an 'Incoming Webook' url setup in Slack Apps. It accepts
// some text and the slack channel is saved within Slack.
func SendSlackNotification(webhookURL string, msg data.SlackRequestBody) error {

	slackBody, _ := json.Marshal(msg)

	log.Printf("Sending message to Slack:\n %v\n", string(slackBody))

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return errors.New("Non-ok response returned from Slack")
	}
	return nil
}
