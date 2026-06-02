package detector

import "testing"

func TestExtractIP(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"myip.ipip.net", "当前 IP：220.163.92.113  来自于：中国 云南 昆明  电信", "220.163.92.113"},
		{"ddns.oray", "Current IP Address: 220.163.92.113", "220.163.92.113"},
		{"vv.video.qq", `QZOutputJson={"s":"o","t":1780403016,"ip":"220.163.92.113","pos":"---"};`, "220.163.92.113"},
		{"pconline", `{"ip":"220.163.92.113","pro":"云南省","proCode":"530000"}`, "220.163.92.113"},
		{"g3.letv", `{ "ipint": "3701693553", "host": "220.163.92.113", "remote": "220.163.92.113", "geo": "CN.25.353.1"}`, "220.163.92.113"},
		{"no ip", `{"err":"not found"}`, ""},
		{"out of range octet", "version 999.1.1.1 build", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractIP(c.body); got != c.want {
				t.Errorf("extractIP(%q) = %q, want %q", c.body, got, c.want)
			}
		})
	}
}
