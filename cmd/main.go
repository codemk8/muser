package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
var apiRoot = flag.String("api_root", "/v1", "api root path")
var client *dynamo.DynamoClient

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

type UserJson struct {
	UserName string `json:"user_name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

type UpdateUserJson struct {
	UserName    string `json:"user_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Password    string `json:"password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	var user UserJson
	err = json.Unmarshal(body, &user)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if user.UserName == "" || user.Password == "" || user.Email == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if client.UserExist(user.UserName) {
		http.Error(w, "User already exist", http.StatusBadRequest)
		return
	}
	hash, err := HashPassword(user.Password)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	dbUser := &dynamo.User{UserName: user.UserName,
		Salt:    hash,
		Created: time.Now().Unix(),
	}
	err = client.AddNewUser(dbUser)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	return
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	username, password, authOK := r.BasicAuth()
	if authOK == false {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	user, err := client.GetUser(username)
	if err != nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	match := CheckPasswordHash(password, user.Salt)
	if !match {
		http.Error(w, "Wrong user name or password", http.StatusUnauthorized)
		return
	}
	fmt.Printf("User %s authorized.\n", username)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	var user UpdateUserJson
	err = json.Unmarshal(body, &user)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if user.UserName == "" || user.Password == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if !client.UserExist(user.UserName) {
		http.Error(w, "User does exist", http.StatusBadRequest)
		return
	}
	dbUser, err := client.GetUser(user.UserName)
	if err != nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	match := CheckPasswordHash(user.Password, dbUser.Salt)
	if !match {
		http.Error(w, "Invalid user name or password", http.StatusUnauthorized)
		return
	}

	if user.NewPassword != "" {
		newHash, err := HashPassword(user.NewPassword)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		err = client.UpdateUserPass(&dynamo.User{UserName: user.UserName,
			Salt: newHash})
		if err != nil {
			http.Error(w, "Internal error ", http.StatusInternalServerError)
			return
		}
		fmt.Printf("User %s password updated.\n", user.UserName)
	}

	if user.Email != "" && user.Email != dbUser.Email {
		err = client.UpdateUserEmail(&dynamo.User{UserName: user.UserName,
			Email: user.Email})
		if err != nil {
			http.Error(w, "Internal error ", http.StatusInternalServerError)
			return
		}
		fmt.Printf("User %s email updated.\n", user.UserName)
	}

	return
}

func main() {
	flag.Parse()
	var err error
	fmt.Printf("Creating AWS client...\n")
	client, err = dynamo.NewClient(*table, *region)
	if err != nil {
		panic("Failed init dynamoDB, check credentials or table name.")
	}
	fmt.Printf("Creating AWS client done!\n")

	r := mux.NewRouter()
	r.HandleFunc(*apiRoot+"/user/register", registerHandler).Methods("POST")
	r.HandleFunc(*apiRoot+"/user/auth", authHandler).Methods("GET")
	r.HandleFunc(*apiRoot+"/user/update", updateHandler).Methods("POST")
	srv := &http.Server{
		Handler: r,
		Addr:    *ip,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Printf("Running server on %s%s/user/register|auth|update, table name %s on region %s.\n", *ip, *apiRoot, *table, *region)
	log.Fatal(srv.ListenAndServe())
}
