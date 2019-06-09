/*
Peanuts Search, search engine
Copyright (C) 2017, 2018, 2019 Anjum Ahmed

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
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"crypto/tls"
	"golang.org/x/crypto/acme/autocert"
	"strings"
	"strconv"
	"time"
	"os"
)

var db = strips{}
var idx = idxs{}

type idxs struct {
	Index []struct {
		Key string `"key":"key"`
		Dates []strip `"dates":"dates"`
	}
}


type strips struct {
	Strip []struct {
		Date  string `"strip":"date"`
		Tags  string `"tags": "tags"`
		Lines string `"strip":"lines"`
	}
}

type strip struct {
	Date string  `json:"date"`
	Rel  float64 `json:"rel"`
	//Num int `json:"num"`
}

type results struct {
	Error  string  `json:"errors"`
	Strips []strip `json:"strips"`
}

func Ex(slice []strip, element strip) []strip {
	n := len(slice)
	if n == cap(slice) {
		newSlice := make([]strip, len(slice), 2*len(slice)+1)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0 : n+1]
	slice[n] = element
	return slice
}

func rank(terms []string, s string) (score float64) {
	//words := strings.Split(s, " ")
	count := 0.0
	for i, term := range terms {
		if i > 0 {
			count = float64(strings.Count(s, term)) * count
		} else  {
			count = float64(strings.Count(s, term))
		}
			//if count > 0 {
			//		count = count + 100
			//}
		//for i := 0; i < len(words); i++ {
		//	if (strings.Contains(words[i], term)) {
		//		count = count + 1
		//	}
		//}
	}
	return count
}

func seek(date string) (index int) {
	for i, strip := range db.Strip {
		if strip.Date == date {
			return i
		}
	}
	return -1
}

func cutup(terms []string) {
	var from string
	var to string
	for _, v := range terms {
		if strings.Contains(v, "from:") {
			from = strings.Split(v, ":")[1]
			fmt.Println(seek(from))
		}
		if strings.Contains(v, "to:") {
			to = strings.Split(v, ":")[1]
			fmt.Println(seek(to))
		}
	}
}

func search(query string) results {
	results := results{"", make([]strip, 0, 256)}

	if len(query) < 2 {
		results.Error = "Query must be 3 or more characters."
		return results
	}

	query = strings.Trim(query, " ")
	query = strings.ToLower(query)
	terms := strings.Split(query, " ")
	count := 0.0

	cutup(terms)
	for _, v := range db.Strip {
		count = rank(terms, v.Lines)
		if count > 0 {
			results.Strips = Ex(results.Strips,
				strip{v.Date, count})
		}
	}

	if len(results.Strips) == 0 {
		results.Error = "No strips found."
	}

	return results
}

func termInIndex(t string) *[]strip {
	for _, v := range idx.Index {
		if strings.Contains(v.Key, t) {
			return &v.Dates
		}
	}
	return nil
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

func setupJson(){
	fp, err := ioutil.ReadFile("transcript.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(fp), &db)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	setupJson()
	_, certErr := os.Stat("certs")

	secure := true
	if certErr != nil {
		secure = false
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/q", postSearchQuery)
	mux.HandleFunc("/html", postSearchQueryHTML)
	mux.HandleFunc("/i/", getImage)
	mux.HandleFunc("/I/", getImageBig)
	mux.HandleFunc("/a/", getAsset)
	mux.HandleFunc("/", getIndex)

	if secure {
		certManager := autocert.Manager{
			Prompt: autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(
				"peanuts-search.com"),
			Cache: autocert.DirCache("certs"),
		}
		server := &http.Server {
			Addr: ":https",
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}
		go http.ListenAndServe(":http", certManager.HTTPHandler(nil))
		log.Fatal(server.ListenAndServeTLS("",""))
	} else {
		fmt.Println("Warning: Not being served over HTTPS")
		server := &http.Server {
			Addr: ":80",
			Handler: mux,
		}
		server.ListenAndServe()
	}
}
