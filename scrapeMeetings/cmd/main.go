package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly/v2"
)

type meeting struct {
	meetingID       int
	meetingLink     string
	meetingTime     time.Time
	meetingType     string
	meetingStatus   string
	boardType       string
	agendaURL       string
	agendaPacketURL string
	summaryURL      string
	minutesURL      string
}

func connectPlanetScale() (*sql.DB, error) {
	// dotEnvErr := godotenv.Load(".env")
	// if dotEnvErr != nil {
	// 	log.Fatal("Error loading .env file")
	// }

	db, err := sql.Open("mysql", os.Getenv("DSN"))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func convertStringToTime(dateStr string) time.Time {
	layout := "Monday, January 2, 2006 3:04 PM"
	location, err := time.LoadLocation("America/Los_Angeles")
	dateStr = strings.ReplaceAll(dateStr, "\x0d", "")
	if err != nil {
		fmt.Println("Error loading location:", err)
	}
	dateTime, err := time.ParseInLocation(layout, dateStr, location)
	if err != nil {
		fmt.Println("Error parsing date:", err)
	}
	return dateTime
}

func insertMeeting(db *sql.DB, meeting meeting) error {
	stmt, err := db.Prepare("INSERT INTO meetings (meetingID, meetingLink, meetingTime, meetingType, meetingStatus, boardType, agendaUrl, agendaPacketURL, summaryURL, minutesUrl) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(meeting.meetingID, meeting.meetingLink, meeting.meetingTime, meeting.meetingType, meeting.meetingStatus, meeting.boardType, meeting.agendaURL, meeting.agendaPacketURL, meeting.summaryURL, meeting.minutesURL)
	if err != nil {
		return err
	}
	return nil
}

func extractSubstring(str string, pre string, post string) string {
	if pre == "" {
		return strings.Split(str, post)[0]
	}
	return strings.Split(strings.Split(str, pre)[1], post)[0]
}

func scrapeMeetingInfo(e *colly.HTMLElement) (int, string, time.Time, string, string, string) {
	url := e.ChildAttr(".RowLink a", "href")
	meetingID := strings.Split(url, "ID=")[1]
	meetingIDConverted, convertErr := strconv.Atoi(meetingID)
	if convertErr != nil {
		panic(convertErr)
	}

	meetingTitle := e.ChildAttr(".RowLink a", "title")
	regex, regexErr := regexp.Compile("\n\n")
	if regexErr != nil {
		panic(regexErr)
	}
	rawTime := extractSubstring(meetingTitle, "", "Board:")

	meetingTitle = regex.ReplaceAllString(meetingTitle, "\n")
	meetingLink := "https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=" + meetingID
	meetingTime := convertStringToTime(rawTime)
	boardType := extractSubstring(meetingTitle, "Board:", "Type:")
	meetingType := extractSubstring(meetingTitle, "Type:", "Status:")
	meetingStatus := extractSubstring(meetingTitle, "Status:", "Council Chambers")

	return meetingIDConverted, meetingLink, meetingTime, boardType, meetingType, meetingStatus
}

func scrapeMeetingLinks(e *colly.HTMLElement) (string, string, string, string) {
	linkClass := ".RowRight.MeetingLinks div"
	var agendaURL string
	var agendaPacketURL string
	var summaryURL string
	var minutesURL string

	e.ForEach(linkClass, func(_ int, f *colly.HTMLElement) {
		link := f.ChildAttr("a", "href")
		linkText := f.ChildText("a")

		if linkText == "Agenda" {
			agendaURL = link
		}
		if linkText == "Agenda Packet" {
			agendaPacketURL = link
		}
		if linkText == "Summary" {
			summaryURL = link
		}
		if linkText == "Minutes" {
			minutesURL = link
		}
	})

	return agendaURL, agendaPacketURL, summaryURL, minutesURL
}

func scrapeMeeting(e *colly.HTMLElement) meeting {
	tempMeeting := &meeting{}

	tempMeeting.meetingID, tempMeeting.meetingLink, tempMeeting.meetingTime, tempMeeting.boardType, tempMeeting.meetingType, tempMeeting.meetingStatus = scrapeMeetingInfo(e)
	tempMeeting.agendaURL, tempMeeting.agendaPacketURL, tempMeeting.summaryURL, tempMeeting.minutesURL = scrapeMeetingLinks(e)

	return *tempMeeting
}

func findMeetings(db *sql.DB) {
	eventClass := ".Row.MeetingRow"
	// Instantiate default collector
	c := colly.NewCollector(
		// Visit only domains: pattersonca.iqm2.com
		colly.AllowedDomains("pattersonca.iqm2.com"),
	)

	c.OnHTML(eventClass, func(e *colly.HTMLElement) {
		// log.Println("Meeting Found...")
		meeting := scrapeMeeting(e)

		// log.Println("Meeting ID:", meeting.meetingID)
		// log.Println("Meeting Time:", meeting.meetingTime)
		// log.Println("Meeting Agenda URL:", meeting.agendaURL)
		// log.Println("Meeting Agenda Packet URL:", meeting.agendaPacketURL)
		// log.Println("Meeting Minutes URL:", meeting.minutesURL)
		// log.Println("Meeting Summary URL:", meeting.summaryURL)

		// log.Println("Attempting to insert meeting into database...")
		insertErr := insertMeeting(db, meeting)
		if insertErr != nil {
			panic(insertErr)
		}
	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	currentTime := time.Now()
	currentYear := strconv.Itoa(currentTime.Year())

	url := "https://pattersonca.iqm2.com/Citizens/calendar.aspx?From=1/1/" + currentYear + "&To=12/31/" + currentYear

	// Start scraping on https://pattersonca.iqm2.com/Citizens/calendar.aspx?From=1/1/[currentYear]&To=12/31/[currentYear]
	c.Visit(url)
}

func Handler() {
	db, err := connectPlanetScale()
	if err != nil {
		log.Fatal(err)
	}
	db.Ping()

	log.Println("Successfully connected to PlanetScale!")

	log.Println("Searching for Meetings...")
	findMeetings(db)
	log.Println("Completed Meeting Search and Insertion!")

}

func main() {
	lambda.Start(Handler)
}
