package main

import "testing"

func TestParseCLIOptions_SteerNext(t *testing.T) {
	opts, _, err := parseCLIOptions([]string{"--headless", "--steer-file", "s.txt", "--next"})
	if err != nil || opts.SteerFile != "s.txt" || !opts.Next {
		t.Fatalf("got %+v, %v", opts, err)
	}
	for _, argv := range [][]string{
		{"--next"},
		{"--steer-file", "s.txt"},
		{"--headless", "--steer-file"},
		{"--headless", "--prompt", "x", "--next"},
		{"--headless", "--prompt-file", "b.md", "--steer-file", "s.txt"},
	} {
		if _, _, err := parseCLIOptions(argv); err == nil {
			t.Errorf("%v: want error", argv)
		}
	}
}
