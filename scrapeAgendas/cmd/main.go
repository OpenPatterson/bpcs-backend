package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly/v2"
)

var STARTING_DATE = "'2023-01-01 00:00:00'"

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

func getAllMeetingIDs(db *sql.DB) []int {
	var meetingIDS []int
	rows, err := db.Query("SELECT DISTINCT meetingID FROM meetings WHERE meetingTime >=" + STARTING_DATE + ";")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var meetingID int
		err = rows.Scan(&meetingID)
		if err != nil {
			panic(err)
		}
		meetingIDS = append(meetingIDS, meetingID)
	}
	return meetingIDS
}

func insertPlanetScale(meetingID int, agenda string, db *sql.DB) {
	stmt, err := db.Prepare("INSERT INTO agendas (meetingID, agendaMD) VALUES (?, ?)")
	if err != nil {
		panic(err.Error())
	}
	defer stmt.Close()
	_, err = stmt.Exec(meetingID, agenda)
	if err != nil {
		panic(err.Error())
	}
}

func scrapeAgenda(e *colly.HTMLElement) string {
	agenda := ""
	result := ""
	e.ForEach("td", func(_ int, e *colly.HTMLElement) {
		if e.Text != "" {
			if e.DOM.HasClass("Num") {
				result += e.Text
			} else {
				link := e.ChildAttr("a", "href")
				indent, err := strconv.Atoi(e.Attr("colspan"))
				if err != nil {
					panic(err)
				}
				indent = 10 - indent
				if indent == 0 {
					result = "### " + result + e.Text
				}
				if indent == 1 {
					result = "- " + result + e.Text
				}
				if indent == 2 {
					result = "    - " + result + e.Text
				}
				if link != "" {
					spltLink := strings.Split(link, "&Type")[0]
					result += " [Link](https://pattersonca.iqm2.com/Citizens/" + spltLink + ")"
				}
				agenda += result + " \n "
				result = ""
			}
		}
	})
	return agenda
}

func findAgendas(meetingID int, db *sql.DB) {
	meetingIDString := strconv.Itoa(meetingID)
	url := "https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=" + meetingIDString
	eventID := "#ContentPlaceholder1_lblOutline"
	c := colly.NewCollector(
		colly.AllowedDomains("pattersonca.iqm2.com"),
	)
	c.OnHTML(eventID, func(e *colly.HTMLElement) {
		agenda := scrapeAgenda(e)
		insertPlanetScale(meetingID, agenda, db)
	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})
	c.Visit(url)
}

func Handler() {
	db, err := connectPlanetScale()
	if err != nil {
		panic(err)
	}

	allMeetingIDS := getAllMeetingIDs(db)

	for _, meetingID := range allMeetingIDS {
		findAgendas(meetingID, db)
	}

	defer db.Close()
}

func main() {
	lambda.Start(Handler)
}
