package schema

import (
	"fmt"
	"math/rand"
	"time"
)

// A schema for user profile
// It stores: username, salt, an email and an avatar string

type Profile struct {
	Email    string `json:"email,omitempty"`
	Verified bool   `json:"verified"`
	Avatar   string `json:"avatar,omitempty"`
}

// Secret group fields hidden from normal access
type Secret struct {
	// the password, salted of course
	Salt string `json:"salt,omitempty"`
	// used for email verifications
	VerifyCode string `json:"verify_code,omitempty"`
	CodeExpiry int64  `json:"expiry,omitempty"`
}

// User is the user schame in database
type User struct {
	UserName string  `json:"user_name,omitempty"`
	Created  int64   `json:"created,omitempty"`
	Profile  Profile `json:"profile,omitempty"`
	Secret   Secret  `json:"secret,omitempty"`
}

// A helper function to generate a 6-digit verification code with an expiry unit timestamp
func GenVerifyCodeAndExpiry(expireInMin int) (string, int64) {
	rand.Seed(time.Now().UnixNano())
	randNum := rand.Intn(999999)
	return fmt.Sprintf("%06d", randNum), time.Now().Local().Add(time.Minute * time.Duration(expireInMin)).Unix()
}

func NewUser(username string, salt string) *User {
	return &User{
		UserName: username,
		Created:  time.Now().Unix(),
		Secret: Secret{
			Salt: salt,
		},
		Profile: Profile{
			Verified: false,
		},
	}
}
