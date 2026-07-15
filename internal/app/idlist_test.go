package app

import (
	"reflect"
	"testing"
)

func TestParseIDList(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    []string
		wantErr bool
	}{
		{"json array", `["id1","id2"]`, []string{"id1", "id2"}, false},
		{"json single", `["184fd1d7742ac942"]`, []string{"184fd1d7742ac942"}, false},
		{"json with spaces", ` [ "id1" , "id2" ] `, []string{"id1", "id2"}, false},
		{"json inner padding", `[" id1 ", "id2"]`, []string{"id1", "id2"}, false},
		{"json numbers stringified", `[1, 22]`, []string{"1", "22"}, false},
		{"csv", "id1,id2", []string{"id1", "id2"}, false},
		{"csv with spaces", " id1 , id2 ", []string{"id1", "id2"}, false},
		{"csv single", "abc", []string{"abc"}, false},
		{"csv trailing comma", "id1,id2,", []string{"id1", "id2"}, false},
		{"csv empty items dropped", "id1,,id2", []string{"id1", "id2"}, false},
		{"empty string", "", nil, true},
		{"whitespace only", "   ", nil, true},
		{"only commas", ",,,", nil, true},
		{"empty json array", `[]`, nil, true},
		{"json array of empties", `["", " "]`, nil, true},
		{"malformed json not csv-fallback", `["id1",`, nil, true},
		{"json non-string item", `[{"id":"x"}]`, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIDList(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseIDList(%q) = %v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseIDList(%q) unexpected error: %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseIDList(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
