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
	"strconv"
	"time"
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

func search(query string) results {
	results := results{"", make([]strip, 0, 256)}

	if len(query) < 2 {
		results.Error = "Query must be 3 or more characters."
		return results
	}

	rows, _ := dba.Query("select date, rank from txscripts where body match ?", query)

	for rows.Next() {
		var date string
		var rank float64
		_ = rows.Scan(&date, &rank)

		results.Strips = append(results.Strips, strip{date, rank})
	}

	return results
}

func postSearchQuery(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	start := time.Now()
	searchResults := search(string(b))
	end := time.Now()
	fmt.Println(end.Sub(start))
	jsonResponse, err := json.Marshal(searchResults)
	if err != nil {
		log.Printf("Malformed JSON for response.")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func postSearchQueryHTML(w http.ResponseWriter, r *http.Request) {
	whole, _ := ioutil.ReadAll(r.Body)
	b := make([]byte, 0, 0)
	if len(whole) > 2 {
		b = whole[2:]
	}
	start := time.Now()
	searchResults := search(string(b))
	end := time.Now()
	fmt.Println(end.Sub(start))

	strips := searchResults.Strips
	w.Write([]byte("have not done a non javascript version yet!!!\n"))
	w.Write([]byte("number of results:\n"))
	w.Write([]byte(strconv.Itoa(len(strips))))
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Println("getting index")
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
	fp, err := ioutil.ReadFile("bg.png")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/png")
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
	mux.HandleFunc("/html", postSearchQueryHTML)
	mux.HandleFunc("/i/", getImage)
	mux.HandleFunc("/I/", getImageBig)
	mux.HandleFunc("/a/", getAsset)
	mux.HandleFunc("/", getIndex)
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
