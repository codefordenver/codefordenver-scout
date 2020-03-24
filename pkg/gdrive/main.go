package gdrive

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/codefordenver/codefordenver-scout/models"
	"github.com/codefordenver/codefordenver-scout/pkg/shared"
	"github.com/jinzhu/gorm"
	"github.com/rickar/cal"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

var calendars map[int]*cal.Calendar

func Monday(date time.Time) time.Time {
	weekdayInt := int(date.Weekday())
	if 8-weekdayInt >= 7 {
		return date.AddDate(0, 0, 1-weekdayInt)
	} else {
		return date.AddDate(0, 0, 8-weekdayInt)
	}
}

// Get the time corresponding to the first day of the current month
func StartOfMonth(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}

// Get which monday of the month a date is(or -1 if it is not a monday)
func MondayOfMonth(date time.Time) int {
	if date.Weekday() == time.Monday {
		return (date.Day()-Monday(StartOfMonth(date)).Day())/7 + 1
	} else {
		return -1
	}
}

// Check if date is a meeting day
func isMeetingDay(date time.Time) bool {
	mondayInt := MondayOfMonth(date)
	return mondayInt == 1 || mondayInt == 2 || mondayInt == 4
}

var client *drive.Service
var db *gorm.DB

// Create a drive API client and calendar object for meeting tracking
func New(dbConnection *gorm.DB) error {
	db = dbConnection

	credsEnv := os.Getenv("GDRIVE_CREDS")
	creds, err := base64.StdEncoding.DecodeString(credsEnv)
	if err != nil {
		fmt.Println("error reading Drive client secret file,", err)
		return err
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(creds, drive.DriveReadonlyScope, drive.DriveFileScope)
	if err != nil {
		fmt.Println("error parsing client secret file to Drive config,", err)
		return err
	}
	HTTPClient, err := getClient(config)
	if err != nil {
		return err
	}

	ctx := context.Background()
	client, err = drive.NewService(ctx, option.WithHTTPClient(HTTPClient), option.WithScopes(drive.DriveMetadataReadonlyScope))
	if err != nil {
		fmt.Println("error retrieving Drive client,", err)
		return err
	}

	calendars = make(map[int]*cal.Calendar, 0)

	var brigades []models.Brigade
	if err = db.Find(&brigades).Error; err != nil {
		fmt.Println("error fetching all brigades,", err)
		return err
	}

	for _, brigade := range brigades {
		calendars[brigade.ID] = &cal.Calendar{
			WorkdayFunc: isMeetingDay,
			Observed:    cal.ObservedExact,
		}
		cal.AddUsHolidays(calendars[brigade.ID])
	}

	return nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tokenEnv := os.Getenv("GDRIVE_ACCESS_TOKEN")
		if tokenEnv == "" {
			tok, err = getTokenFromWeb(config)
			if err != nil {
				return nil, err
			}
		} else {
			dToken, err := base64.StdEncoding.DecodeString(tokenEnv)
			if err != nil {
				fmt.Println("error reading client secret file,", err)
				return nil, err
			}

			tok = &oauth2.Token{}
			r := bytes.NewReader(dToken)
			err = json.NewDecoder(r).Decode(tok)
		}
	}

	if err := saveToken(tokFile, tok); err != nil {
		return nil, err
	}
	return config.Client(context.Background(), tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		fmt.Println("error reading authorization code,", err)
		return nil, err
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		fmt.Println("error retrieving token from web,", err)
		return nil, err
	}
	return tok, nil
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
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("error caching OAuth token,", err)
		return err
	}
	defer f.Close()
	if err = json.NewEncoder(f).Encode(token); err != nil {
		fmt.Println("error encoding token to JSON,", err)
		return err
	}
	return nil
}

func FetchAgenda(data shared.CommandData) shared.FunctionResponse {
	location, err := time.LoadLocation(data.Brigade.TimezoneString)
	if err != nil {
		fmt.Println(err)
	}
	date := time.Now().In(location)

	nextMeetingDate := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, location)

	c := calendars[data.Brigade.ID]

	if c.IsWorkday(date) {
		nextMeetingDate = nextMeetingDate.AddDate(0, 0, date.Day()-1)
	} else if c.WorkdaysRemain(date) == 0 {
		nextMonth := date.AddDate(0, 1, 0)
		nextMeetingDate = nextMeetingDate.AddDate(0, 1, c.WorkdayN(nextMonth.Year(), nextMonth.Month(), 1)-1)
	} else {
		nextMeetingDate = nextMeetingDate.AddDate(0, 0, c.WorkdayN(date.Year(), date.Month(), c.Workdays(date.Year(), date.Month())-c.WorkdaysRemain(date)+1)-1)
	}

	r, err := client.Files.List().Q(fmt.Sprintf("name contains 'Meeting Agenda - %s'", nextMeetingDate.Format("2006/01/02"))).OrderBy("modifiedTime desc").PageSize(1).
		Fields("files(name, webViewLink)").Do()
	if err != nil {
		fmt.Println("error fetching files,", err)
		return shared.FunctionResponse{
			ChannelID: data.ChannelID,
			Error:     "Failed to fetch files from Google Drive. Try again later.",
		}
	}
	var agenda *drive.File
	if len(r.Files) == 0 {
		r, err = client.Files.List().Q(fmt.Sprintf("'%s' in parents", data.Brigade.AgendaFolderID)).OrderBy("modifiedTime desc").PageSize(1).Fields("files(id, parents)").Do()
		if err != nil {
			fmt.Println("error fetching files,", err)
			return shared.FunctionResponse{
				ChannelID: data.ChannelID,
				Error:     "Failed to fetch files from Google Drive. Try again later.",
			}
		}
		if len(r.Files) == 0 {
			fmt.Println("empty or invalid agenda folder")
			return shared.FunctionResponse{
				ChannelID: data.ChannelID,
				Error:     "Found an empty or invalid agenda folder in Google Drive. Check your configuration.",
			}
		}
		newAgenda := drive.File{Name: fmt.Sprintf("Meeting Agenda %s", nextMeetingDate.Format("2006/01/02"))}
		agenda, err = client.Files.Copy(r.Files[0].Id, &newAgenda).Fields("name, webViewLink").Do()
		if err != nil {
			fmt.Println("error copying file,", err)
			return shared.FunctionResponse{
				ChannelID: data.ChannelID,
				Error:     "Failed to create new agenda. Try again later.",
			}
		}
	} else {
		agenda = r.Files[0]
	}
	return shared.FunctionResponse{
		ChannelID: data.ChannelID,
		Success:   fmt.Sprintf("%s - %s", agenda.Name, agenda.WebViewLink),
	}
}
