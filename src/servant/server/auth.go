package server
import (
	"strings"
	"strconv"
	"fmt"
	"crypto/sha1"
	"encoding/hex"
	"time"
)

/*
 Authorization: user ts sha1(user + key + ts + method + uri)

 */
func (self *Session) auth() (err error) {
	defer func() {
		if err != nil {
			time.Sleep(1 * time.Second)
		}
	}()
	if !self.config.Auth.Enabled {
		return nil
	}
	authStr := self.req.Header.Get("Authorization")
	segs := strings.SplitN(authStr, " ", 3)
	if len(segs) < 3 {
		return fmt.Errorf("bad Authorization header")
	}
	reqUser := segs[0]
	tsStr := segs[1]
	reqHash := segs[2]

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return err
	}
	nowTs := time.Now().Unix()
	maxDelta := self.config.Auth.MaxTimeDelta
	if nowTs - ts > int64(maxDelta) || ts - nowTs > int64(maxDelta) {
		return fmt.Errorf("timestamp delta too large")
	}
	user, ok := self.config.Users[reqUser]
	if !ok {
		return fmt.Errorf("user %s not found", reqUser)
	}
	strToHash := reqUser + user.Key + tsStr + self.req.Method + self.req.RequestURI
	fmt.Println(strToHash)
	sha1Sum := sha1.Sum([]byte(strToHash))
	realHash := hex.EncodeToString(sha1Sum[:])
	if reqHash != realHash {
		return fmt.Errorf("auth failed")
	}
	return nil
}
