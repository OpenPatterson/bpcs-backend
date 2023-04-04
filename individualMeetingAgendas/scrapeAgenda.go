package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
)

func connectPlanetScale() (*sql.DB, error) {
	dotEnvErr := godotenv.Load(".env")
	if dotEnvErr != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := sql.Open("mysql", os.Getenv("DSN"))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func insertPlanetScale(meetingID int, agenda string, db *sql.DB) {
	stmt, err := db.Prepare("INSERT INTO agendas (meetingID, agendaMD) VALUES (?, ?)")
	if err != nil {
		panic(err.Error())
	}
	_, err = stmt.Exec(meetingID, agenda)
	if err != nil {
		panic(err.Error())
	}
}

// Sometimes there is no agenda, see: https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1121
// If there is no agenda make empty document? or document with "No Agenda" in it? Next.JS will put all downloads at top of page

//Test following pages
// https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1373
// https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1121
// https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1401

func scrapeAgenda(e *colly.HTMLElement) string {
	agenda := ""
	result := ""
	e.ForEach("td", func(_ int, e *colly.HTMLElement) {
		if e.Text != "" {
			// fmt.Println("Children", e.DOM.Children().Length(), "Num Class?", e.DOM.HasClass("Num"), "Title Class", e.DOM.HasClass("Title"), "Colspan?", e.Attr("colspan"), "Text:", e.Text)

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
				agenda += result + " \\n "
				result = ""
			}
		}
	})
	return agenda
}

func findAgendas(meetingID string, db *sql.DB) {
	url := "https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=" + meetingID
	eventID := "#ContentPlaceholder1_lblOutline"
	c := colly.NewCollector(
		// Visit only domains: pattersonca.iqm2.com
		colly.AllowedDomains("pattersonca.iqm2.com"),
	)
	c.OnHTML(eventID, func(e *colly.HTMLElement) {
		// log.Println("Meeting Found...")
		agenda := scrapeAgenda(e)
		meetingIDString, err := strconv.Atoi(meetingID)
		if err != nil {
			panic(err)
		}
		insertPlanetScale(meetingIDString, agenda, db)
	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})
	c.Visit(url)
}

func main() {
	db, err := connectPlanetScale()
	if err != nil {
		panic(err)
	}
	findAgendas("1401", db)
	defer db.Close()
}
