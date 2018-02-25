package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

// Data Model
// Index is position of this Block in whole chain
// Timestamp is creation time of this Block
// Data is (Data field)
// PrevHash is SHA256 value of previsou Block
type Block struct {
	Index     int
	Timestamp string
	Data      int
	Hash      string
	PrevHash  string
}

var Blockchain []Block

type Message struct {
	Data int
}

func main() {
	err := godotenv.Load() // reading
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		t := time.Now()
		genesisBlock := Block{0, t.String(), 0, "", ""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()
	log.Fatal(run())
}

// Hashing and Generating New Blocks
// SHA256 algorithm is used for maintain the order and position of Block in whole chain

func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + block.PrevHash + string(block.Data) // attached the field data
	hash := sha256.New()
	hash.Write([]byte(record))
	hashed := hash.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(old Block, Data int) (Block, error) {
	t := time.Now()
	// Build new Block
	var newBlock Block
	newBlock.Index = old.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.PrevHash = old.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}

// Block Validation
func isBlockValid(new, old Block) bool {
	if old.Index+1 != new.Index {
		return false
	}
	if old.Hash != new.PrevHash {
		return false
	}
	if calculateHash(new) != new.Hash {
		return false
	}
	return true
}

// Distributed system has consensus problem
// Simple strategy : Chose the longest chain
func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

// Web Server
func run() error {
	mux := makeMuxRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8333"
	}
	log.Println("Listening on ", os.Getenv("PORT"))
	s := &http.Server{
		Addr:           ":" + port,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := s.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

// no now API design,
func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlockchain).Methods("POST")
	return muxRouter
}

// Simple strategy : directly return JSON
func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	io.WriteString(w, string(bytes))
}

func handleWriteBlockchain(w http.ResponseWriter, r *http.Request) {
	var m Message
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()
	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.Data)
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}
	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, playload interface{}) {
	response, err := json.MarshalIndent(playload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}
