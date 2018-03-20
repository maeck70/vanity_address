 package main

import "os"
import "fmt"
import "time"
import "bufio"
import "math/rand"
import "strings"
import "log"
import "github.com/maeck70/giota"
import "database/sql"
import	_ "github.com/lib/pq"

const tryteAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ9"
const blockSize = 100
const minWordSize = 5


type addressDetail struct {
	address giota.Address
	index int
	security int
	wordsFound string
	numWords int
	score int
}

type seedBlock struct {
	seed giota.Trytes
	addressDetailSet [blockSize]addressDetail
}


var wordSet []string
var db int


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
						ad.wordsFound = fmt.Sprintf("%s %s", ad.wordsFound, wordSet[i])
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
		sb.addressDetailSet[i].index = i
		sb.addressDetailSet[i].security = security
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


func collect(sb *seedBlock, db *sql.DB) {

	sb.seed = generateSeed()

	getAddressBlock(sb)

	fmt.Printf("Seed: %s\n", sb.seed)
	for i := 0; i < blockSize; i++ {
		if sb.addressDetailSet[i].score > 0 {
			writeAddress(db, &sb.seed, &sb.addressDetailSet[i])
			fmt.Printf("[%d] %s", i, highlightWords(string(sb.addressDetailSet[i].address), sb.addressDetailSet[i].wordsFound))
			fmt.Printf(" %d %s\n", sb.addressDetailSet[i].score, sb.addressDetailSet[i].wordsFound)
		}
	}
	fmt.Println()	

}



func writeSeed(db *sql.DB, seed *giota.Trytes) int {
	var seedId int

	err := db.QueryRow(`SELECT id FROM public."Address_seed" WHERE seed = $1`, seed).Scan(&seedId)

	switch {
	case err == sql.ErrNoRows:
		err := db.QueryRow(`INSERT INTO public."Address_seed"(seed) 
					VALUES($1) RETURNING id`, seed).Scan(&seedId)
		if err != nil {
			log.Fatal(err)
		}
	case err != nil:
		log.Fatal(err)
	}

	return seedId
}



func writeWords(db *sql.DB, addressId int, ad *addressDetail) {
	var wordId int

	words := strings.Split(ad.wordsFound, " ")

	for i := range(words) {
		word := words[i]
		err := db.QueryRow(`SELECT id FROM public."Address_word" WHERE word = $1`, word).Scan(&wordId)

		switch {
		case err == sql.ErrNoRows:
			err := db.QueryRow(`INSERT INTO public."Address_word"(word, score_multiplier) 
						VALUES($1, $2) RETURNING id`, word, 1).Scan(&wordId)
			if err != nil {
				log.Fatal(err)
			}
		case err != nil:
			log.Fatal(err)
		}

		index := strings.Index(string(ad.address), word)
		err = db.QueryRow(`INSERT INTO public."Address_addressword"(address_id, word_id, position) 
					VALUES($1, $2, $3) RETURNING id`, addressId, wordId, index).Scan(&wordId)
		if err != nil {
			log.Fatal(err)
		}
	}
}



func writeAddress(db *sql.DB, seed *giota.Trytes, ad *addressDetail) {
	var seedId int
	var addressId int

	seedId = writeSeed(db, seed)
	if seedId != 0 {
		err := db.QueryRow(`INSERT INTO public."Address_address"(seed_id, address, index, security, score)
					VALUES($1, $2, $3, $4, $5) RETURNING id`, seedId, ad.address, ad.index, ad.security, ad.score).Scan(&addressId)
		if err != nil {
			log.Fatal(err)
		}
		writeWords(db, addressId, ad)
	}
}



func main(){

	loadWords()

	connStr := "user=django password=wedersmeer dbname=iotaVanityAddress"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	for i := 500; i > 0; i-- {
		sb := seedBlock{}

		collect(&sb, db)
	}
}

