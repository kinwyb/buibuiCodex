package cdm

import "strings"

type command struct {
	msg       string
	newThread bool //新线程
}

// commandParse 命令解析
func commandParse(msg string) *command {
	if msg != "" {
		if after, ok := strings.CutPrefix(msg, "/new "); ok {
			return &command{
				msg:       after,
				newThread: true,
			}
		}
	}
	return &command{
		msg: msg,
	}
}
