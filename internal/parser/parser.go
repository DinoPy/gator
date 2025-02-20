package parser

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"
)

type RSSFeed struct {
	Channel struct {
		Title		string		`xml:"title"`
		Link		string		`xml:"link"`
		Descripiton	string		`xml:"descripiton"`
		Item		[]RSSItem	`xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title		string		`xml:"title"`
	Link		string		`xml:"link"`
	Description	string		`xml:"description"`
	PubDate		string		`xml:"pubDate"`
}

func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		feedURL,
		nil,
	)
	if err != nil {
		return &RSSFeed{ }, fmt.Errorf("Failed to create FetchFeed request. Error: %w\n", err)
	}

	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("Failed to make the FeechFeed request. Error: %w\n", err)
	}

	defer res.Body.Close()
	byteData, err := io.ReadAll(res.Body)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("Failed to read body data. Error %w\n", err)
	}

	var rssFeed RSSFeed
	err = xml.Unmarshal(byteData, &rssFeed)
	if err != nil {
		return &RSSFeed{}, fmt.Errorf("Failed to unmarshal the FetchFeed XML response. Error: %w\n", err)
	}

	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Descripiton = html.UnescapeString(rssFeed.Channel.Descripiton)
	for i := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[i].Title = html.UnescapeString(rssFeed.Channel.Item[i].Title)
		rssFeed.Channel.Item[i].Description = html.UnescapeString(rssFeed.Channel.Item[i].Description)
	}

	return &rssFeed, nil
}

func ParseDate (dateString string) (time.Time, error) {
	layouts := []string {
		time.RFC3339,         // "2006-01-02T15:04:05Z07:00"
		"2006-01-02 15:04:05", // "2006-01-02 15:04:05"
		"2006-01-02",          // "2006-01-02"
		"01/02/2006",          // "01/02/2006"
		"02 Jan 2006",        // "02 Jan 2006"
		time.RFC1123Z,        // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,         // "Mon, 02 Jan 2006 15:04:05 MST"
		"Jan 02 2006",        // "Jan 02 2006"
		"02 Jan 06 15:04 MST", // "02 Jan 06 15:04 MST" (RFC822)
		"01/02/06 15:04:05",  // "01/02/06 15:04:05"
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, dateString)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("Coult not parse the time string: %s", dateString)
}
