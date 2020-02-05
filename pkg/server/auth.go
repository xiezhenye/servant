package server

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

/*
 Authorization: user ts sha1(user + key + ts + method + uri)

*/
func (self *Session) auth() (username string, err error) {
	defer func() {
		// deley 1s on fail to prevent attack
		if err != nil {
			time.Sleep(1 * time.Second)
		}
	}()
	if !self.config.Auth.Enabled {
		return "", nil
	}
	authStr := self.req.Header.Get("Authorization")
	reqUser, reqHash, ts, err := parseAuthHeader(authStr)
	if err != nil {
		return "", err
	}
	user, ok := self.config.Users[reqUser]
	if !ok {
		return "", fmt.Errorf("user %s not found", reqUser)
	}
	remoteHost := strings.Split(self.req.RemoteAddr, ":")[0]
	if !checkHosts(remoteHost, user.Hosts) {
		return reqUser, fmt.Errorf("remote host %s is denied", self.req.RemoteAddr)
	}
	if user.Key != "" {
		nowTs := time.Now().Unix()
		maxDelta := self.config.Auth.MaxTimeDelta
		if nowTs-ts > int64(maxDelta) || ts-nowTs > int64(maxDelta) {
			return reqUser, fmt.Errorf("timestamp delta too large")
		}
		strToHash := reqUser + user.Key + strconv.FormatInt(ts, 10) + self.req.Method + self.req.RequestURI
		sha1Sum := sha1.Sum([]byte(strToHash))
		realHash := hex.EncodeToString(sha1Sum[:])
		if reqHash != realHash {
			return reqUser, fmt.Errorf("auth failed")
		}
	}
	return reqUser, nil
}

func parseAuthHeader(authStr string) (user, hash string, ts int64, err error) {
	segs := strings.SplitN(authStr, " ", 3)
	user = segs[0]
	if len(segs) < 3 {
		hash = ""
		ts = 0
		err = fmt.Errorf("bad auth header")
		return
	}
	tsStr := segs[1]
	hash = segs[2]
	ts, err = strconv.ParseInt(tsStr, 10, 64)
	return
}

func checkPermission(group string, allows []string) bool {
	for _, allow := range allows {
		if group == allow {
			return true
		}
	}
	return false
}

func (self *Session) checkPermission() bool {
	if self.username == "" {
		return true
	}
	return checkPermission(self.group, self.UserConfig().Allows[self.resource])
}

func checkHosts(remoteAddr string, hosts []string) bool {
	if len(hosts) <= 0 {
		return true
	}
	ok := false
	for _, host := range hosts {
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
		return false
	}
	return true
}
