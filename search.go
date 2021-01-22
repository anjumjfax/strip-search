/*
Peanuts Search, search engine
Copyright (C) 2017, 2018, 2019, 2020 Anjum Ahmed

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/acme/autocert"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"strconv"
	"strings"
	"regexp"
)

var dba *sql.DB

type strip struct {
	Date string  `json:"date"`
	Rel  float64 `json:"rel"`
}

type results struct {
	Error  string  `json:"errors"`
	Strips []strip `json:"strips"`
}

type htmlResults struct {
	Strips []strip
	Pages []page
}

type page struct {
	No string
	Off int
	Q string
	Ord string
}

func search(query string, order int) results {
	results := results{"", make([]strip, 0, 256)}

	if len(query) < 2 {
		results.Error = "Query must be 3 or more characters."
		return results
	}

	var rows *sql.Rows
	var err error

	reg, regerr := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(regerr)
	}

	query = reg.ReplaceAllString(query, " ")

	if order == 0 {
	rows, err = dba.Query("select date, rank from txscripts where body match ? order by rank", query)
	if err != nil {
		log.Println(err)
	}
	}
	if order > 0 {
	rows, err = dba.Query("select date, rank from txscripts where body match ? order by date desc", query)
	if err != nil {
		log.Println(err)
	}
	}
	if order < 0 {
	rows, err = dba.Query("select date, rank from txscripts where body match ? order by date asc", query)
	if err != nil {
		log.Println(err)
	}
	}

	for rows.Next() {
		var date string
		var rank float64
		_ = rows.Scan(&date, &rank)

		results.Strips = append(results.Strips, strip{date, rank})
	}

	return results
}

func oneSearch(query string) string {
	if len(query) < 2 {
		return ""
	}
	var date string
	err := dba.QueryRow("select date from txscripts where body match ? order by rank limit 1", query).Scan(&date)
	if err != nil {
		log.Println(err)
	}
	return date
}

func getOneSearchQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q["q"] == nil {
		http.NotFound(w, r)
		return
	}

	query := q["q"][0]
	filename := oneSearch(query)

	if filename == "" {
		filename = "favicon.png"
	} else {
		filename = "i/"+strings.Replace(filename, "-", "", 3)
	}

	fp, err := ioutil.ReadFile(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(fp)
}

func postSearchQuery(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	start := time.Now()
	searchResults := search(string(b), 0)
	end := time.Now()
	fmt.Println(end.Sub(start))
	jsonResponse, err := json.Marshal(searchResults)
	if err != nil {
		log.Printf("Malformed JSON for response.")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func pageNos(length int, offset int, q string, ord string) []page {
	pages := make([]page, 0)
	if offset - 96 >= 0 {
		pages = append(pages, page {"-96", offset -96, q, ord})
	}
	if offset - 48 >= 0 {
		pages = append(pages, page {"-48", offset -48, q, ord})
	}
	if offset - 24 >= 0 {
		pages = append(pages, page {"-24", offset-24, q, ord})
	}
	if offset + 24 < length {
		pages = append(pages, page {"+24", offset + 24, q, ord})
	}
	if offset + 48 < length {
		pages = append(pages, page {"+48", offset + 48, q, ord})
	}
	if offset + 96 < length {
		pages = append(pages, page {"+96", offset + 96, q, ord})
	}
	return pages
}

func getSearchQueryHTML(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q["q"] == nil {
		tmp := htmlResults{}
		t, _ := template.ParseFiles("results.html")
		t.Execute(w, tmp)
		return
	}
	b := q["q"][0]
	offset := 0
	if q["offset"] != nil {
		offset_conv, err := strconv.Atoi(q["offset"][0])
		if err == nil {
			offset = offset_conv
		}
	}
	order := 0
	if q["order"] != nil {
		order_q, err := strconv.Atoi(q["order"][0])
		if err == nil {
			order = order_q
		}
	}
	start := time.Now()
	searchResults := search(b, order)
	end := time.Now()
	fmt.Println(end.Sub(start))
	no := len(searchResults.Strips)
	if offset < 0 {
		offset = 0;
	}
	if no < offset {
		offset = no - 24
	}
	upper := offset + 24
	if no < upper {
		upper = no
	}
	res := htmlResults{ searchResults.Strips[offset:upper], pageNos(no, offset, b, strconv.Itoa(order)) }
	for n, i := range res.Strips {
		res.Strips[n].Date = strings.Replace(i.Date, "-", "", 3)
	}
	t, _ := template.ParseFiles("results.html")
	t.Execute(w, res)
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("index.html")
	q := r.URL.Query()
	w.Header().Set("Content-Type", "text/html")
	if len(q["q"]) > 0 {
		t.Execute(w, q["q"][0])
	} else {
		t.Execute(w, nil)
	}
}

func getImage(w http.ResponseWriter, r *http.Request) {
	img := r.URL.Path[len("/i/"):]
	fp, err := ioutil.ReadFile("i/" + img)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(fp)
}

func getImageBig(w http.ResponseWriter, r *http.Request) {
	img := r.URL.Path[len("/i/"):]
	fp, err := ioutil.ReadFile("I/" + img)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Write(fp)
}

func getAsset(w http.ResponseWriter, r *http.Request) {
	mime := "image/jpeg"
	filename := r.URL.Path[len("/a/"):]
	switch filename {
	case "about.html":
		mime = "text/html"
	case "js.js":
		mime = "application/javascript"
	case "favicon.png":
		mime = "image/png"
	case "bg.png":
		mime = "image/png"
	}
	fp, err := ioutil.ReadFile(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Write(fp)
}

func getGoogle(w http.ResponseWriter, r *http.Request) {
	fp, err := ioutil.ReadFile("google99ab2ca2e675d9dd.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(fp)
}

func secure() bool {
	_, certErr := os.Stat("certs")
	secure := true
	if certErr != nil {
		secure = false
	}
	return secure
}

func routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/q", postSearchQuery)
	mux.HandleFunc("/html", getSearchQueryHTML)
	mux.HandleFunc("/r", getOneSearchQuery)
	mux.HandleFunc("/i/", getImage)
	mux.HandleFunc("/I/", getImageBig)
	mux.HandleFunc("/a/", getAsset)
	mux.HandleFunc("/", getIndex)
	mux.HandleFunc("/google99ab2ca2e675d9dd.html", getGoogle)
	return mux
}

func main() {
	dba, _ = sql.Open("sqlite3", "txscripts.db")
	defer dba.Close()

	mux := routes()

	if secure() {
		certManager := autocert.Manager{
			Prompt: autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(
				"peanuts-search.com"),
			Cache: autocert.DirCache("certs"),
		}
		server := &http.Server{
			Addr:    ":https",
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}
		go http.ListenAndServe(":http", certManager.HTTPHandler(nil))
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		fmt.Println("Warning: Not being served over HTTPS")
		server := &http.Server{
			Addr:    ":8080",
			Handler: mux,
		}
		log.Fatal(server.ListenAndServe())
	}
}
