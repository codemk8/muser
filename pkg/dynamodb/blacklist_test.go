package dynamo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBlackListMap(t *testing.T) {
	blacklist := NewBlackListMap()
	assert.Equal(t, blacklist["normal_user"], false, "a normal user name")
	assert.Equal(t, blacklist["umac-128-etm"], true, "a random pick in the black list")
	assert.Equal(t, blacklist["google"], true, "company name in the end")
	assert.Equal(t, blacklist["geforce"], true, "the last one?")
}
