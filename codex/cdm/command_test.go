package cdm

import "testing"

func TestCommandParseSteerAcceptsOptionalSpace(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want string
	}{
		{name: "有空格", msg: "/steer 补充要求", want: "补充要求"},
		{name: "无空格", msg: "/steer补充要求", want: "补充要求"},
		{name: "多余空白", msg: "/steer \t 补充要求 ", want: "补充要求"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := CommandParse(tc.msg)
			if !cmd.Steer || cmd.Msg != tc.want {
				t.Fatalf("steer 解析不符: %+v", cmd)
			}
		})
	}
}
