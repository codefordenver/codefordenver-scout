package gdrive

import (
	"encoding/json"
	"fmt"
	"github.com/codefordenver/scout/global"
	"github.com/rickar/cal"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

var c *cal.Calendar

func Monday(date time.Time) time.Time {
	weekdayInt := int(date.Weekday())
	if 8-weekdayInt >= 7 {
		return date.AddDate(0, 0, 1-weekdayInt)
	} else {
		return date.AddDate(0, 0, 8-weekdayInt)
	}
}

// Get the time corresponding to the first day of the current month
func StartOfMonth() time.Time {
	location, err := time.LoadLocation(global.LocationString)
	if err != nil {
		fmt.Println(err)
	}
	date := time.Now().In(location)
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, location)
}

// Get which monday of the month a date is(or -1 if it is not a monday)
func MondayOfMonth(date time.Time) int {
	if date.Weekday() == time.Monday {
		return (date.Day()-Monday(StartOfMonth()).Day())/7 + 1
	} else {
		return -1
	}
}

// Check if date is a meeting day
func isMeetingDay(date time.Time) bool {
	mondayInt := MondayOfMonth(date)
	return mondayInt == 1 || mondayInt == 2 || mondayInt == 4
}

// Create a drive API client and calendar object for meeting tracking
func Create() (*drive.Service, error) {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client), option.WithScopes(drive.DriveMetadataReadonlyScope))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
		return nil, err
	}

	c = cal.NewCalendar()

	c.WorkdayFunc = isMeetingDay

	cal.AddUsHolidays(c)

	return srv, nil
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
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
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
	if err = json.NewEncoder(f).Encode(token); err != nil {
		fmt.Println("Failed to encode token to JSON")
	}
}

func FetchAgenda(s *drive.Service) string {
	location, err := time.LoadLocation(global.LocationString)
	if err != nil {
		fmt.Println(err)
	}
	date := time.Date(2019, time.June, 1, 0, 0, 0, 0, location)

	nextMeetingDate := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, location)

	if c.WorkdaysRemain(date) == 0 {
		nextMonth := date.AddDate(0, 1, 0)
		nextMeetingDate = nextMeetingDate.AddDate(0, 1, c.WorkdayN(nextMonth.Year(), nextMonth.Month(), 1)-1)
	} else {
		nextMeetingDate = nextMeetingDate.AddDate(0, 0, c.WorkdayN(date.Year(), date.Month(), c.Workdays(date.Year(), date.Month())-c.WorkdaysRemain(date)+1)-1)
	}
	r, err := s.Files.List().Q(fmt.Sprintf("name contains 'Meeting Agenda - %s'", nextMeetingDate.Format("2006/01/02"))).OrderBy("modifiedTime desc").PageSize(1).
		Fields("files(name, webViewLink)").Do()
	if err != nil {
		fmt.Println(err)
		return "Error fetching files from Google Drive"
	}
	var agenda *drive.File
	if len(r.Files) == 0 {
		r, err = s.Files.List().Q(fmt.Sprintf("'%s' in parents", global.AgendaFolderID)).OrderBy("modifiedTime desc").PageSize(1).Fields("files(id, parents)").Do()
		if err != nil {
			fmt.Println(err)
			return "Error fetching files from Google Drive"
		}
		newAgenda := drive.File{Name: fmt.Sprintf("Meeting Agenda %s", nextMeetingDate.Format("2006/01/02"))}
		agenda, err = s.Files.Copy(r.Files[0].Id, &newAgenda).Fields("name, webViewLink").Do()
		if err != nil {
			fmt.Println(err)
			return "Error creating new agenda"
		}
	} else {
		agenda = r.Files[0]
	}
	return fmt.Sprintf("%s - %s", agenda.Name, agenda.WebViewLink)
}
