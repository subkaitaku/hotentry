package hatebu

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type HotEntry struct {
	Items []*Item `xml:"item"`
}

type Item struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	Date          string `xml:"date"`
	BookmarkCount int    `xml:"bookmarkcount"`
}

type Content struct {
	Title string
	URL   string
}

type blockDomain string
type blockDomains []blockDomain

type blockWord string
type blockWords []blockWord

func httpGet(url string) string {
	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	defer response.Body.Close()
	return string(body)
}

func maxWidth(entries []*Item, max int) int {
	width := 0

	for _, e := range entries {
		count := utf8.RuneCountInString(e.Title)
		if count > width {
			width = count
		}

		if width > max {
			return max
		}
	}

	return width
}

func (ds blockDomains) Match(url string) bool {
	for _, d := range ds {
		if strings.Contains(url, string(d)) {
			return true
		}
	}
	return false
}

func (ws blockWords) Match(title string) bool {
	for _, w := range ws {
		if strings.Contains(title, string(w)) {
			return true
		}
	}
	return false
}

func replaceOverflowText(text string, width int) string {
	if runewidth.StringWidth(text) > width {
		return runewidth.Truncate(text, width-3, "...")
	} else {
		return text
	}
}

func PrintHatebu(w http.ResponseWriter, r *http.Request) {
	data := httpGet("http://b.hatena.ne.jp/hotentry/it.rss")

	hotentry := HotEntry{}

	err := xml.Unmarshal([]byte(data), &hotentry)

	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	titleWidth := maxWidth(hotentry.Items, 200)
	urlWidth := maxWidth(hotentry.Items, 200)
	urlFmt := fmt.Sprintf("%%-%ds", urlWidth)

	contents := []Content{}
	for _, bookmark := range hotentry.Items {
		var bds blockDomains
		bds = []blockDomain{
			"anond.hatelabo.jp",
			"togetter.com",
			"gizmodo.jp",
			"blog.livedoor.jp",
			"twitter.com",
			"x.com",
			"2ch",
			"5ch",
		}
		if bds.Match(bookmark.Link) {
			continue
		}

		var bws blockWords
		bws = []blockWord{
			"ハッとした",
			"常識",
			"残念",
			"必見",
			"政治",
			"ヤバい",
			"初心者",
			"驚愕",
			"遺憾",
			"駆け出し",
			"マルチ",
		}
		if bws.Match(bookmark.Title) {
			continue
		}

		title := bookmark.Title
		link := bookmark.Link
		contents = append(contents, Content{runewidth.FillRight(replaceOverflowText(title, titleWidth), titleWidth), fmt.Sprintf(urlFmt, link)})
	}

	htmlTemplate := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Hatebu Hotentry</title>
	</head>
	<body>
		<h1>Hatebu Hotentry</h1>
		<ul>
			{{range .}}
				<li><a href="{{.URL}}" target="_blank">{{.Title}}</a></li>
			{{end}}
		</ul>
	</body>
	</html>
`

	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, contents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}