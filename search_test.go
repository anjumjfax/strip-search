package main

import(
	"testing"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func TestSearch(t *testing.T){
	dba, _ = sql.Open("sqlite3", "txscripts.db")
	defer dba.Close()

	var res results
	res = search("blanket", 0)

	if len(res.Strips) == 0 {
		t.Errorf("search for blanket yielded nothing")
	}
}
