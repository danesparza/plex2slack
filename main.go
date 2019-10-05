package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

//	Set up our flags
var (
	port           = flag.Int("port", 3300, "The port to listen on")
	allowedOrigins = flag.String("allowedOrigins", "*", "A comma-separated list of valid CORS origins")
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
}

func main() {
	//	Parse environment variables:
	parseEnvironment()

	//	Parse the command line for flags:
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/plex", func(w http.ResponseWriter, req *http.Request) {

		readForm, err := req.MultipartReader()
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
				buf := new(bytes.Buffer)
				buf.ReadFrom(part)
				log.Println("payload is: ", buf.String())
			}
		}

		// Dump the request.
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("\n*** BEGIN ***")
		fmt.Println(string(requestDump))
		fmt.Println("*** END ***")

		fmt.Fprintf(w, "hello\n")
	})

	//	CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(*allowedOrigins, ","),
		AllowCredentials: true,
	})
	handler := c.Handler(r)

	//	Indicate what port we're starting the service on
	portString := strconv.Itoa(*port)
	fmt.Println("Allowed origins: ", *allowedOrigins)
	fmt.Println("Starting server on :", portString)
	http.ListenAndServe(":"+portString, handler)
}
