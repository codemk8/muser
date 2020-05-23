package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"

	"github.com/go-resty/resty/v2"
)

func SendVerifyEmail(emailEndpoint string, username string, email string, verifyCode string) error {
	if emailEndpoint == "" {
		glog.Warning("email svc not configured. skipping sending verifying email")
	}
	requestJSON := VerifyRequest{
		UserName:   username,
		To:         email,
		VerifyCode: verifyCode,
	}
	requestBody, err := json.Marshal(requestJSON)
	if err != nil {
		glog.Warningf("error marshalling json %v", err)
		return fmt.Errorf("%v", err)
	}
	client := resty.New()
	request := client.R().SetHeader("Content-Type", "application/json").SetBody(requestBody)
	resp, err := request.Post(emailEndpoint)
	if err != nil {
		glog.Warningf("error post to email service %v", err)
		return err
	}
	if resp.StatusCode() == http.StatusOK {
		return nil
	}
	if len(resp.Body()) != 0 {
		glog.Warningf("Error sending email: %s", string(resp.Body()))
		return errors.New("remote API error response: " + string(resp.Body()))
	}
	glog.Warningf("Error sending email, error code %d", strconv.Itoa(resp.StatusCode()))
	return errors.New("remote API error code " + strconv.Itoa(resp.StatusCode()))
}
