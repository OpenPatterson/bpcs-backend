package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

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

// Iterate through td elements
// If HasClass("Num") then it's a number, store it
// If HasClass("Title") then it's a title, store it and check for any links below it <a class="Link"
// If there is e.Attr("colspan") then it's a continuation of the previous item
// To get the text of the item, use e.Text

// Sometimes there is no agenda, see: https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1121
// If there is no agenda make empty document? or document with "No Agenda" in it? Next.JS will put all downloads at top of page

//Test following pages
// https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1373
// https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1121
// https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=1401

func scrapeAgenda(e *colly.HTMLElement) int {
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
				if link != "" {
					result += e.Text
					fmt.Println("Complete line at indent:", indent, result, "With Link:", link)
					result = ""
				} else {
					result += e.Text
					fmt.Println("Complete line at indent:", indent, result)
					result = ""
				}

			}
		}
	})
	// fmt.Println(e)
	return 0
}

func findAgendas(meetingID string) {
	url := "https://pattersonca.iqm2.com/Citizens/Detail_Meeting.aspx?ID=" + meetingID
	eventID := "#ContentPlaceholder1_lblOutline"
	c := colly.NewCollector(
		// Visit only domains: pattersonca.iqm2.com
		colly.AllowedDomains("pattersonca.iqm2.com"),
	)
	c.OnHTML(eventID, func(e *colly.HTMLElement) {
		// log.Println("Meeting Found...")
		agenda := scrapeAgenda(e)
		fmt.Print("ignore", agenda)
	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})
	c.Visit(url)
}

func main() {
	findAgendas("1373")
}
