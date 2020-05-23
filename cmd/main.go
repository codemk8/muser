package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"time"

	"github.com/golang/glog"

	dynamo "github.com/codemk8/muser/pkg/dynamodb"
	"github.com/codemk8/muser/pkg/schema"
	"github.com/codemk8/muser/pkg/verify"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var ip = flag.String("addr", "127.0.0.1:8000", "Serving host and port")
var table = flag.String("table", "dev.muser.codemk8", "Table name")
var region = flag.String("region", "us-west-2", "AWS Region the table is in")
var apiRoot = flag.String("api_root", "/v1", "api root path")
var emailEndpoint = flag.String("emailep", "", "Email service for verification")
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

// UserJSON defines the new user format
type UserJSON struct {
	UserName string `json:"user_name,omitempty"`
	Password string `json:"password,omitempty"`
}

// Veirfy is request for (email) verification
type VerifyJSON struct {
	UserName   string `json:"user_name,omitempty"`
	VerifyCode string `json:"verify_code,omitempty"`
}

// validation.Field(&a.Email, validation.Required, is.Email),
func (a UserJSON) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.UserName, validation.Required, validation.Length(5, 32)),
		validation.Field(&a.Password, validation.Required, validation.Length(7, 32)),
	)
}

// UpdateUserJSON defines the update json format
type UpdateUserJSON struct {
	UserName    string `json:"user_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	Password    string `json:"password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

func (update UpdateUserJSON) Validate() error {
	if update.Password != "" {
		return validation.ValidateStruct(&update,
			validation.Field(&update.NewPassword, validation.Required, validation.Length(7, 32)))
	}
	if update.Email != "" {
		return validation.ValidateStruct(&update,
			validation.Field(&update.Email, validation.Required, is.Email))
	}
	return nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	user := UserJSON{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		glog.Warningf("Failed to decode json: %v.", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = user.Validate()
	if err != nil {
		b, _ := json.Marshal(err)
		glog.Warningf("Bad request: %f", err)
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}

	if client.BadUserName(user.UserName) {
		glog.Warningf("Username %s is in blacklist", user.UserName)
		http.Error(w, "username is not available", http.StatusBadRequest)
		return
	}

	if client.UserExist(user.UserName) {
		glog.Warningf("User already exist")
		http.Error(w, "the username already exist", http.StatusBadRequest)
		return
	}

	hash, err := HashPassword(user.Password)
	if err != nil {
		glog.Warningf("Error hashing password")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	dbUser := schema.NewUser(user.UserName, hash)
	err = client.AddNewUser(dbUser)
	if err != nil {
		glog.Warningf("Error adding new user: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	return
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	username, password, authOK := r.BasicAuth()
	if authOK == false {
		glog.Warning("Failed to parse basic auth from header")
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	user, err := client.GetUser(username, true)
	if err != nil {
		glog.Warningf("Failed to get user from db: %v.", err)
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	match := CheckPasswordHash(password, user.Secret.Salt)
	if !match {
		glog.Warning("Password does not match hash.")
		http.Error(w, "Wrong user name or password", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	update := UpdateUserJSON{}
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		glog.Warningf("Failed to decode json: %v.", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if update.UserName == "" {
		http.Error(w, "bad request: no username specified", http.StatusBadRequest)
		return
	}

	dbUser, err := client.GetUser(update.UserName, true)
	if err != nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	if dbUser == nil {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	err = update.Validate()
	if err != nil {
		b, _ := json.Marshal(err)
		glog.Warningf("bad request: %f", err)
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}

	if update.Password != "" {
		match := CheckPasswordHash(update.Password, dbUser.Secret.Salt)
		if !match {
			http.Error(w, "Invalid user name or password", http.StatusUnauthorized)
			return
		}
		newHash, err := HashPassword(update.NewPassword)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		dbUser.Secret.Salt = newHash

	} else {
		if update.Email != "" {
			// TODO check duplicated email, reject if same email found

			dbUser.Profile.Email = update.Email
			// generate code here
			code, expiry := schema.GenVerifyCodeAndExpiry(60)
			dbUser.Secret.VerifyCode = code
			dbUser.Secret.CodeExpiry = expiry
			dbUser.Profile.Verified = false
			// send notification to email service to send the verify code
			verify.SendVerifyEmail(*emailEndpoint, update.UserName, update.Email, code)
		}
		if update.Avatar != "" {
			// TODO check format
			dbUser.Profile.Avatar = update.Avatar
		}
	}
	err = client.AddNewUser(dbUser)
	if err != nil {
		glog.Warningf("Error adding new user: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	return
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	user := UserJSON{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		glog.Warningf("Failed to decode json: %v.", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if user.UserName == "" {
		glog.Warningf("No user name in get handler")
		http.Error(w, "bad request, needs usename", http.StatusBadRequest)
		return
	}
	dbUser, err := client.GetUser(user.UserName, false)
	if err != nil {
		glog.Warningf("Failed to get user from db: %v.", err)
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	userJSON, err := json.Marshal(dbUser)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Write(userJSON)

}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	verify := VerifyJSON{}
	err := json.NewDecoder(r.Body).Decode(&verify)
	if err != nil {
		glog.Warningf("Failed to decode json: %v.", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if verify.UserName == "" || verify.VerifyCode == "" {
		glog.Warningf("No user name or verify code in verify handler")
		http.Error(w, "bad request, needs usename or verifying code", http.StatusBadRequest)
		return
	}
	dbUser, err := client.GetUser(verify.UserName, true)
	if err != nil {
		http.Error(w, "not authorized", http.StatusUnauthorized)
		return
	}
	if dbUser == nil {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}
	now := time.Now().Local().Unix()
	if now >= dbUser.Secret.CodeExpiry {
		http.Error(w, "verification code expired", http.StatusBadRequest)
		return
	}
	if verify.VerifyCode != dbUser.Secret.VerifyCode {
		http.Error(w, "incorrect verification code", http.StatusBadRequest)
		return
	}
	dbUser.Profile.Verified = true
	err = client.AddNewUser(dbUser)
	if err != nil {
		glog.Warningf("Error updating user: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	return
}

func main() {
	flag.Parse()
	var err error
	glog.Infof("Creating AWS client...\n")
	client, err = dynamo.NewClient(*table, *region)
	if err != nil {
		panic("Failed init dynamoDB, check credentials or table name.")
	}
	glog.Infof("Creating AWS client done!\n")

	r := mux.NewRouter()
	r.HandleFunc(*apiRoot+"/user/register", registerHandler).Methods("POST")
	r.HandleFunc(*apiRoot+"/user/auth", authHandler).Methods("GET")
	r.HandleFunc(*apiRoot+"/user/update", updateHandler).Methods("POST")
	r.HandleFunc(*apiRoot+"/user", getHandler).Methods("POST")
	r.HandleFunc(*apiRoot+"/user/verify", verifyHandler).Methods("POST")
	srv := &http.Server{
		Handler: r,
		Addr:    *ip,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	glog.Infof("Running server on %s%s/user/register|auth|update, table name %s on region %s.\n", *ip, *apiRoot, *table, *region)
	glog.Fatal(srv.ListenAndServe())
}
