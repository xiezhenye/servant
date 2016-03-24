package overlord

import (
	"net/http"
	"crypto/sha1"
	"encoding/hex"
	"time"
	"strconv"
	"fmt"
	"io"
)

type Client struct {
	Host  string
	User  string
	Key   string
}

func (self *Client) AuthHeader(method, uri string) string {
	// Authorization: user ts sha1(user + key + ts + method + uri)
	tsStr := strconv.Itoa(time.Now().Unix())
	strToHash := self.User + self.Key + tsStr + method + uri
	sha1Sum := sha1.Sum([]byte(strToHash))
	hexHash := hex.EncodeToString(sha1Sum[:])
	return fmt.Sprintf("%s %s %s", self.User, tsStr, hexHash)
}

func (self *Client) HttpClient(method, uri string, body io.Reader) {
	ret, err:= http.NewRequest(method, uri, body)
	if err != nil {
		return
	}
	ret.Header.Set("Authorization", self.AuthHeader(method, uri))
}
