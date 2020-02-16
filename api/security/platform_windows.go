// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package security

import (
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"os/user"
	"syscall"

	"github.com/frankhang/util/logutil"
	acl "github.com/hectane/go-acl"
	"golang.org/x/sys/windows"
)

var (
	wellKnownSidStrings = map[string]string{
		"Administrators": "S-1-5-32-544",
		"System":         "S-1-5-18",
		"Users":          "S-1-5-32-545",
	}
	wellKnownSids = make(map[string]*windows.SID)
)

func init() {
	for key, val := range wellKnownSidStrings {
		sid, err := windows.StringToSid(val)
		if err == nil {
			wellKnownSids[key] = sid
		}
	}
}

// lookupUsernameAndDomain obtains the username and domain for usid.
func lookupUsernameAndDomain(usid *syscall.SID) (username, domain string, e error) {
	username, domain, t, e := usid.LookupAccount("")
	if e != nil {
		return "", "", e
	}
	if t != syscall.SidTypeUser {
		return "", "", fmt.Errorf("user: should be user account type, not %d", t)
	}
	return username, domain, nil
}

// writes auth token(s) to a file with the same permissions as datadog.yaml
func saveAuthToken(token, tokenPath string) error {
	// get the current user
	var sidString string
	currUser, err := user.Current()
	if err != nil {
		logutil.BgLogger().Warnf("Unable to get current user", zap.Error(err))
		logutil.BgLogger().Info("Attempting to get current user information directly")
		tok, e := syscall.OpenCurrentProcessToken()
		if e != nil {
			logutil.BgLogger().Warn("Couldn't get process token", zap.Error(e))
			return e
		}
		defer tok.Close()
		user, e := tok.GetTokenUser()
		if e != nil {
			logutil.BgLogger().Warnf("Couldn't get  token user", zap.Error(e))
			return e
		}
		sidString, e = user.User.Sid.String()
		if e != nil {
			logutil.BgLogger().Warnf("Couldn't get  user sid string", zap.Error(e))
			return e
		}

		logutil.BgLogger().Infof("Got sidstring from token user")

		// now just do some debugging, see what we weren't able to get.
		pg, e := tok.GetTokenPrimaryGroup()
		if e != nil {
			logutil.BgLogger().Warnf("Would have failed getting token PG", zap.Error(e))
		}
		_, e = pg.PrimaryGroup.String()
		if e != nil {
			logutil.BgLogger().Warn("Would have failed getting  PG  string", zap.Error(e))
		}
		dir, e := tok.GetUserProfileDirectory()
		if e != nil {
			logutil.BgLogger().Warn("Would have failed getting  primary directory", zap.Error(e))
		} else {
			logutil.BgLogger().Info(fmt.Sprintf("Profile directory is %v", dir))
		}
		username, domain, e := lookupUsernameAndDomain(user.User.Sid)
		if e != nil {
			logutil.BgLogger().Warn("Would have failed getting username and domain", zap.Error(e))
		} else {
			logutil.BgLogger().Info(fmt.Sprintf("Username/domain is %v %v", username, domain))
		}

	} else {
		logutil.BgLogger().Info("Getting sidstring from current user")
		sidString = currUser.Uid
	}
	currUserSid, err := windows.StringToSid(sidString)
	if err != nil {
		logutil.BgLogger().Warnf("Unable to get current user sid", zap.Error(err))
		return err
	}
	err = ioutil.WriteFile(tokenPath, []byte(token), 0755)
	if err == nil {
		err = acl.Apply(
			tokenPath,
			true,  // replace the file permissions
			false, // don't inherit
			acl.GrantSid(windows.GENERIC_ALL, wellKnownSids["Administrators"]),
			acl.GrantSid(windows.GENERIC_ALL, wellKnownSids["System"]),
			acl.GrantSid(windows.GENERIC_ALL, currUserSid))
		logutil.BgLogger().Info("Wrote auth token acl", zap.Error(err))
	}
	return err
}
