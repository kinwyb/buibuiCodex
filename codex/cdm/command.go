package cdm

import "strings"

type Command struct {
	Msg       string
	NewThread bool //新线程
	Steer     bool //附加消息
}

// CommandParse 命令解析
func CommandParse(msg string) *Command {
	if msg != "" {
		if after, ok := strings.CutPrefix(msg, "/new "); ok {
			return &Command{
				Msg:       after,
				NewThread: true,
			}
		}
		if after, ok := strings.CutPrefix(msg, "/steer "); ok {
			return &Command{
				Msg:   after,
				Steer: true,
			}
		}
	}
	return &Command{
		Msg: msg,
	}
}
