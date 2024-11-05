package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"time"

	"wallcalendar/waveshare"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	numWeeks = 4
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func startOfDayOfWeek(date time.Time, location *time.Location) time.Time {
	daysSinceSunday := int(date.Weekday())
	return midnight(date.AddDate(0, 0, -daysSinceSunday), location)
}

func midnight(t time.Time, location *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, location)
}

func getEventTime(start *calendar.EventDateTime) (time.Time, error) {
	if start.Date != "" {
		return time.Parse(time.DateOnly, start.Date)
	}
	if start.DateTime != "" {
		return time.Parse(time.RFC3339, start.DateTime)
	}
	return time.UnixMicro(0), errors.New("no date found on event")
}

func main() {
	useCachedImage := flag.Bool("use_cached_image", false, "If cached image should be used")
	flag.Parse()

	if !*useCachedImage {
		ctx := context.Background()
		b, err := os.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		// If modifying these scopes, delete your previously saved token.json.
		config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}
		client := getClient(config)

		srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("Unable to retrieve Calendar client: %v", err)
		}

		newYork, _ := time.LoadLocation("America/New_York")

		today := midnight(time.Now().In(newYork), newYork)
		start := startOfDayOfWeek(today, newYork)
		end := start.AddDate(0, 0, numWeeks*7+1)

		dateMap := make(map[time.Time][]*calendar.Event)

		events, err := srv.Events.List("family01175849838019336469@group.calendar.google.com").ShowDeleted(false).
			SingleEvents(true).TimeMin(start.Format(time.RFC3339)).TimeMax(end.Format(time.RFC3339)).OrderBy("startTime").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
		}

		if len(events.Items) == 0 {
			fmt.Println("No upcoming events found.")
		} else {
			for _, item := range events.Items {
				date, _ := getEventTime(item.Start)
				roundedDate := midnight(date, newYork)
				dateMap[roundedDate] = append(dateMap[roundedDate], item)
			}
		}

		c := canvas.New(1304, 984)
		context := canvas.NewContext(c)
		context.MoveTo(0, 0)
		context.LineTo(1304, 0)
		context.LineTo(1304, 984)
		context.LineTo(0, 984)
		context.SetFillColor(canvas.Whitesmoke)
		context.Fill()

		fontQuattrocento := canvas.NewFontFamily("quattrocento")
		if err := fontQuattrocento.LoadFontFile("Quattrocento-Regular.ttf", canvas.FontRegular); err != nil {
			panic(err)
		}
		if err := fontQuattrocento.LoadFontFile("Quattrocento-Regular.ttf", canvas.FontBold); err != nil {
			panic(err)
		}

		calendar := &Calendar{
			HeaderFace:    fontQuattrocento.Face(60.0, canvas.Black, canvas.FontBold),
			EventFace:     fontQuattrocento.Face(45.0, canvas.Black, canvas.FontRegular),
			EventTimeFace: fontQuattrocento.Face(45.0, canvas.Black, canvas.FontItalic),
			Timezone:      newYork,
		}

		daysOfWeek := []string{
			"SUN",
			"MON",
			"TUE",
			"WED",
			"THU",
			"FRI",
			"SAT",
		}
		for i, day := range daysOfWeek {
			calendar.RenderHeader(context, float64(i*186+18), 925, day)
		}

		for i := 0; i < numWeeks; i++ {
			for j := 0; j < 7; j++ {
				date := start.AddDate(0, 0, 7*i+j)

				boxLeft := float64(j*186 + 18)
				boxTop := float64(900 - 225*i)
				calendar.Render(context, boxLeft, boxTop, date, dateMap[date], date == today)
			}

		}

		if err := renderers.Write("output.png", c, canvas.DPMM(1)); err != nil {
			panic(err)
		}
	}

	waveshare.Initialize()
	//waveshare.Clear()

	f, err := os.Open("output.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}

	waveshare.Display(image)
	waveshare.Sleep()
}
