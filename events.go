package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Event struct {
	Summary       string
	StartTime     time.Time
	EndTime       time.Time
	IsAllDayEvent bool
}

func getTime(dateTime *calendar.EventDateTime, tz *time.Location, isEnd bool) (time.Time, error) {
	if dateTime.DateTime != "" {
		eventTime, err := time.Parse(time.RFC3339, dateTime.DateTime)
		if err != nil {
			return time.Time{}, err
		}
		return eventTime.In(tz), nil

	}
	eventTime, err := time.ParseInLocation(time.DateOnly, dateTime.Date, tz)
	if err != nil {
		return time.Time{}, err
	}
	if isEnd {
		return eventTime.Add(-1 * time.Second), nil
	}
	return eventTime, nil
}

func NewEvent(e *calendar.Event, tz *time.Location) (Event, error) {
	start, err := getTime(e.Start, tz, false)
	if err != nil {
		return Event{}, err
	}
	end, err := getTime(e.End, tz, true)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Summary:       e.Summary,
		StartTime:     start,
		EndTime:       end,
		IsAllDayEvent: e.Start.Date != "",
	}, nil
}

func isSameDay(a time.Time, b time.Time, tz *time.Location) bool {
	a1 := midnight(a, tz)
	b1 := midnight(b, tz)
	return a1.Equal(b1)
}

func (e Event) StartsOnDate(date time.Time, tz *time.Location) bool {
	return isSameDay(e.StartTime, date, tz)
}

func (e Event) EndsOnDate(date time.Time, tz *time.Location) bool {
	return isSameDay(e.EndTime, date, tz)
}

func (e Event) StartTimeShort(tz *time.Location) string {
	fmt := "3:04pm "
	if e.StartTime.Minute() == 0 {
		fmt = "3pm "
	}
	return e.StartTime.In(tz).Format(fmt)
}

func FetchEvents(today time.Time, calendarId string, tz *time.Location) (map[time.Time][]Event, time.Time, time.Time) {
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

	start := startOfDayOfWeek(today, tz)
	end := start.AddDate(0, 0, numWeeks*7)
	lastday := end.AddDate(0, 0, -1)

	dateMap := make(map[time.Time][]Event)

	events, err := srv.Events.List(calendarId).ShowDeleted(false).
		SingleEvents(true).TimeMin(start.Format(time.RFC3339)).TimeMax(end.Format(time.RFC3339)).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		for _, item := range events.Items {
			e, _ := NewEvent(item, tz)
			roundedDate := midnight(e.StartTime, tz)
			roundedEndDate := midnight(e.EndTime, tz)
			dateMap[roundedDate] = append(dateMap[roundedDate], e)
			for roundedDate = roundedDate.Add(24 * time.Hour); roundedDate.Compare(roundedEndDate) <= 0; roundedDate = roundedDate.Add(24 * time.Hour) {
				dateMap[roundedDate] = append(dateMap[roundedDate], e)
			}
		}
	}

	return dateMap, start, lastday
}

func startOfDayOfWeek(date time.Time, location *time.Location) time.Time {
	daysSinceSunday := int(date.Weekday())
	return midnight(date.AddDate(0, 0, -daysSinceSunday), location)
}

func midnight(t time.Time, tz *time.Location) time.Time {
	t = t.In(tz)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, tz)
}

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
