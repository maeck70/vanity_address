 package main

import "os"
import "fmt"
import "time"
import "bufio"
import "math/rand"
import "strings"
import "github.com/maeck70/giota"
/***
import "database/sql"
import	_ "github.com/mattn/go-sqlite3"
***/

const tryteAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ9"
const blockSize = 100
const minWordSize = 5


type addressDetail struct {
	address giota.Address
	wordsFound string
	numWords int
	score int
}

type seedBlock struct {
	seed giota.Trytes
	addressDetailSet [blockSize]addressDetail
}


var wordSet []string

func loadWords() {

	c := 0

	for i := 0; i < len(tryteAlphabet); i++ {
		filename := fmt.Sprintf("wordlists/%s Words.txt", string(tryteAlphabet[i]))

		r, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Error while opening file %s\n", filename)
		}
		defer r.Close()

		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			word := strings.ToUpper(scanner.Text())
			wordSet = append(wordSet, word)
			c++
		}
	}

	fmt.Printf("Loaded %d words.\n\n", c)
}


func findWords(ad *addressDetail) {

	for i := range(wordSet) {
		if strings.Contains(string(ad.address), wordSet[i]) {
			p := strings.Index(string(ad.address), wordSet[i])
			if len(wordSet[i]) >= minWordSize {
				if len(ad.wordsFound) == 0 {
						ad.wordsFound = fmt.Sprintf("%s", wordSet[i])
					} else {
						ad.wordsFound = fmt.Sprintf("%s, %s", ad.wordsFound, wordSet[i])
					}
				ad.score += (len(wordSet[i])*5) + (81 - p)
				if p == 0 {
					ad.score += 50
				}
				ad.numWords++
			}
		}
	}

}


func generateSeed() giota.Trytes {

	var seed string 

	rand.Seed(time.Now().UnixNano())

	for i := 81; i > 0; i-- {
		l := rand.Intn(len(tryteAlphabet))
		seed += string(tryteAlphabet[l])
	}

	return giota.Trytes(seed)
}


func getAddressBlock(sb *seedBlock) {

	security := 2

	for i := 0; i < blockSize; i++ {
		addr, err := giota.NewAddress(sb.seed, i, security)
		if err != nil {
			fmt.Printf("Error: %s", err)
		}
		sb.addressDetailSet[i].address = addr
		findWords(&sb.addressDetailSet[i])
	}
}


func highlightWords(address string, wordsFound string) string {
	words := strings.Split(wordsFound, " ")

	for i := 0; i < len(words); i++ {
		index := strings.Index(address, words[i])
		if index != -1 {
			address = fmt.Sprintf("%s\033[1;31m%s\033[0m%s", address[0:index], words[i], address[index+len(words[i]):])			
		}
	}

	return address
}


func collect(sb *seedBlock) {

	sb.seed = generateSeed()

	getAddressBlock(sb)

	fmt.Printf("Seed: %s\n", sb.seed)
	for i := 0; i < blockSize; i++ {
		if sb.addressDetailSet[i].score > 0 {
			fmt.Printf("[%d] %s", i, highlightWords(string(sb.addressDetailSet[i].address), sb.addressDetailSet[i].wordsFound))
			fmt.Printf(" %d %s\n", sb.addressDetailSet[i].score, sb.addressDetailSet[i].wordsFound)
		}
	}
	fmt.Println()	

}


/***
func createTables(db *){
	sqlStmt := "create table seed (id integer not null primary key, seed text);"
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	sqlStmt = "create table address (id integer not null primary key, seed integer, address text, words integer, words_found text, score integer);"
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
}


func writeAddress(db *, seed *giota.Trytes, ad *addressDetail) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("insert into seed (id, name) values(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("こんにちわ世界%03d", i))
		if err != nil {
			log.Fatal(err)
		}
	}

	tx.Commit()
}
***/


func main(){

	loadWords()

	/***
	db, err := sql.Open("sqlite3", "./vanityDB.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTables(*db)
	***/

	for i := 500; i > 0; i-- {
		sb := seedBlock{}

		collect(&sb)
	}
}

