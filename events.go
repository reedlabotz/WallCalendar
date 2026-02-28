package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Event struct {
	ID            string
	Summary       string
	StartTime     time.Time
	EndTime       time.Time
	IsAllDayEvent bool
	Slot          int
	RowHeight     int
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
	summary := e.Summary
	// Strip variation selectors (U+FE00-U+FE0F) which render as boxes in Unifont
	summary = strings.Map(func(r rune) rune {
		if r >= 0xFE00 && r <= 0xFE0F {
			return -1
		}
		return r
	}, summary)

	return Event{
		ID:            e.Id,
		Summary:       summary,
		StartTime:     start,
		EndTime:       end,
		IsAllDayEvent: e.Start.Date != "",
		Slot:          -1,
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

func FetchEvents(today time.Time, calendarId string, tz *time.Location, numWeeks int, tokenJSON string) (map[time.Time][]*Event, time.Time, time.Time) {
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
	config.RedirectURL = "http://localhost:8080"

	tok := &oauth2.Token{}
	if err := json.Unmarshal([]byte(tokenJSON), tok); err != nil {
		log.Fatalf("Unable to parse token from config: %v", err)
	}
	client := config.Client(ctx, tok)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	start := startOfDayOfWeek(today, tz)
	end := start.AddDate(0, 0, numWeeks*7)
	lastday := end.AddDate(0, 0, -1)

	dateMap := make(map[time.Time][]*Event)
	var allEvents []*Event

	events, err := srv.Events.List(calendarId).ShowDeleted(false).
		SingleEvents(true).TimeMin(start.Format(time.RFC3339)).TimeMax(end.Format(time.RFC3339)).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	if len(events.Items) != 0 {
		for _, item := range events.Items {
			e, _ := NewEvent(item, tz)
			ePtr := &e
			allEvents = append(allEvents, ePtr)

			roundedDate := midnight(e.StartTime, tz)
			roundedEndDate := midnight(e.EndTime, tz)
			dateMap[roundedDate] = append(dateMap[roundedDate], ePtr)
			for roundedDate = roundedDate.Add(24 * time.Hour); roundedDate.Compare(roundedEndDate) <= 0; roundedDate = roundedDate.Add(24 * time.Hour) {
				dateMap[roundedDate] = append(dateMap[roundedDate], ePtr)
			}
		}
	}

	// Assign slots
	// We need to iterate day by day.
	// For each day, we look at the events on that day.
	// If an event already has a slot assigned (from a previous day), we respect it.
	// If not, we find the first available slot that is free for the duration of the event (or at least for the days we are rendering).

	// To do this efficiently, we can keep track of slot usage per day.
	// map[time.Time]map[int]bool -> day -> slotIndex -> occupied

	slotUsage := make(map[time.Time]map[int]bool)

	// Initialize slotUsage for all days in range
	curr := start
	for !curr.After(lastday) {
		slotUsage[curr] = make(map[int]bool)
		curr = curr.AddDate(0, 0, 1)
	}

	// Iterate through all events. Since they are sorted by start time (from API), we process earlier events first.
	// However, the API sort might not be strictly what we want for multi-day vs single day.
	// Usually, longer events starting on the same day should be processed first or standard layout algorithms apply.
	// For simplicity, we process in the order returned.

	for _, e := range allEvents {
		// Find the days this event covers within our view
		var days []time.Time
		s := midnight(e.StartTime, tz)
		if s.Before(start) {
			s = start
		}
		f := midnight(e.EndTime, tz)
		if f.After(lastday) {
			f = lastday
		}

		// Collect days
		iter := s
		for !iter.After(f) {
			days = append(days, iter)
			iter = iter.AddDate(0, 0, 1)
		}

		if len(days) == 0 {
			continue
		}

		// Find a slot that is free for ALL 'days'
		slot := 0
		for {
			isFree := true
			for _, day := range days {
				if used, ok := slotUsage[day]; ok {
					if used[slot] {
						isFree = false
						break
					}
				}
			}
			if isFree {
				break
			}
			slot++
		}

		// Assign slot
		e.Slot = slot

		// Mark as used
		for _, day := range days {
			if _, ok := slotUsage[day]; ok {
				slotUsage[day][slot] = true
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

// getTokenFromWeb requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	codeCh := make(chan string)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, "Authentication successful! You can close this window.")
			codeCh <- code
		}
	})

	server := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser if it doesn't open automatically: \n%v\n", authURL)

	// Try to open the browser
	_ = exec.Command("open", authURL).Start()

	authCode := <-codeCh
	server.Shutdown(context.Background())

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}
