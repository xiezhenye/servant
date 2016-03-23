package server
import (
	"strings"
	"strconv"
	"fmt"
	"crypto/sha1"
	"encoding/hex"
	"time"
	"net"
)

/*
 Authorization: user ts sha1(user + key + ts + method + uri)

 */
func (self *Session) auth() (err error) {
	defer func() {
		// deley 1s on fail to prevent attact
		if err != nil {
			time.Sleep(1 * time.Second)
		}
	}()
	if !self.config.Auth.Enabled {
		return nil
	}
	authStr := self.req.Header.Get("Authorization")
	reqUser, reqHash, ts, err := parseAuthHeader(authStr)
	if err != nil {
		return err
	}
	user, ok := self.config.Users[reqUser]
	if !ok {
		return fmt.Errorf("user %s not found", reqUser)
	}
	if err = checkHosts(self.req.RemoteAddr, user.Hosts); err != nil {
		return err
	}
	if user.Key != "" {
		nowTs := time.Now().Unix()
		maxDelta := self.config.Auth.MaxTimeDelta
		if nowTs - ts > int64(maxDelta) || ts - nowTs > int64(maxDelta) {
			return fmt.Errorf("timestamp delta too large")
		}
		strToHash := reqUser + user.Key + strconv.Itoa(ts) + self.req.Method + self.req.RequestURI
		sha1Sum := sha1.Sum([]byte(strToHash))
		realHash := hex.EncodeToString(sha1Sum[:])
		if reqHash != realHash {
			return fmt.Errorf("auth failed")
		}
	}
	return nil
}

func parseAuthHeader(authStr string) (user, hash string, ts int64, err error){
	segs := strings.SplitN(authStr, " ", 3)
	user = segs[0]
	if len(segs) < 3 {
		hash = ""
		ts = 0
		return
	}
	tsStr := segs[1]
	hash = segs[2]
	ts, err = strconv.ParseInt(tsStr, 10, 64)
	return
}

func checkHosts(remoteAddr string, hosts []string) error {
	if len(hosts) <= 0 {
		return nil
	}
	ok := false
	for _, host := range (hosts) {
		_, allowedNet, err := net.ParseCIDR(host)
		if err != nil {
			continue
		}
		if allowedNet.Contains(net.ParseIP(remoteAddr)) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("remote host is denied")
	}
	return nil
}
