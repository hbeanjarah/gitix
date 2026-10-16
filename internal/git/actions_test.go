package git

import "testing"

func TestVersion(t *testing.T) {
	cases := map[string][2]int{
		"git version 2.55.0":                 {2, 55},
		"git version 2.45.1.windows.1":       {2, 45},
		"git version 2.39.5 (Apple Git-154)": {2, 39},
	}
	for out, want := range cases {
		major, minor, err := New(fakeRunner{out: out}).Version()
		if err != nil || major != want[0] || minor != want[1] {
			t.Errorf("%q: got %d.%d, %v", out, major, minor, err)
		}
	}
}

func TestConfigListSplitsOnSpacesAndCommas(t *testing.T) {
	names, ok, err := New(fakeRunner{out: "main, develop staging"}).ConfigList("gitix.keep")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(names) != 3 || names[0] != "main" || names[1] != "develop" || names[2] != "staging" {
		t.Errorf("got %q", names)
	}
}

func TestConfigListUnset(t *testing.T) {
	_, ok, err := New(fakeRunner{err: &Error{ExitCode: 1}}).ConfigList("gitix.keep")
	if err != nil || ok {
		t.Errorf("want ok=false err=nil, got ok=%v err=%v", ok, err)
	}
}
