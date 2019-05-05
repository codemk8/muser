package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	dynamo "github.com/codemk8/muser/pkg/dynamodb"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var ip = flag.String("addr", "127.0.0.1:8000", "Serving host and port")
var table = flag.String("table", "dev.muser.codemk8", "Table name")
var region = flag.String("region", "us-west-2", "AWS Region the table is in")

// HashPassword encrypts password into bcrypt hash, the cost should be at least 12
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash compares a bcrypt hashed password with its possible
// plaintext equivalent. Returns true on match
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "foo")
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "foo")
}

func main() {
	flag.Parse()

	client, err := dynamo.NewClient(*table, *region)
	if err != nil {
		panic("Failed init dynano")
	}
	fmt.Printf("User exist :%v", client.UserExist("non-user"))

	r := mux.NewRouter()
	r.HandleFunc("/user/register", registerHandler).Methods("POST")
	r.HandleFunc("/user/auth", registerHandler).Methods("GET")
	srv := &http.Server{
		Handler: r,
		Addr:    *ip,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
