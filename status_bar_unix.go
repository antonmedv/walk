//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os/user"
	"strconv"
	"syscall"
)

func (e Env) Owner() (string, error) {
	fileInfo, err := e.CurrentFile.Info()
	if err != nil {
		return "", err
	}

	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return "", errors.New("unsupported platform")
	}

	uidStr := strconv.FormatUint(uint64(stat.Uid), 10)
	gidStr := strconv.FormatUint(uint64(stat.Gid), 10)

	username := uidStr
	if userInfo, err := user.LookupId(uidStr); err == nil {
		username = userInfo.Username
	}

	groupname := gidStr
	if groupInfo, err := user.LookupGroupId(gidStr); err == nil {
		groupname = groupInfo.Name
	}

	return fmt.Sprintf("%s %s", username, groupname), nil
}
