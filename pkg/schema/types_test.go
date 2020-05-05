package schema

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenVerifyCodeAndExpiry(t *testing.T) {
	code, codeExpiry := GenVerifyCodeAndExpiry(1)
	assert.Equal(t, len(code), 6, "hardcoded to 6 digit")
	now := time.Now().Unix()
	fmt.Printf("now is: %d future is %d", now, codeExpiry)
	assert.Equal(t, codeExpiry-now > 58, true, "should be ok with a computer")
	assert.Equal(t, codeExpiry-now < 62, true, "should be ok with a computer")
}
