package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMediaURL(t *testing.T) {
	cases := map[string]string{
		"https://img.wangmoyu.com/mw600/a.png":        "https://img.wangmoyu.com/mw600/a.png.webp",
		"//img.wangmoyu.com/mw1024/a.gif":             "https://img.wangmoyu.com/large/a.mp4",
		"https://img.toto.im/mw600/a.jpg?x=1&amp;y=2": "https://img.toto.im/mw600/a.jpg.webp?x=1&y=2",
		"https://img.wangmoyu.com/large/a.mp4":        "https://img.wangmoyu.com/large/a.mp4",
		"javascript:alert(1)":                         "",
	}
	for input, want := range cases {
		if got := mediaURL(input); got != want {
			t.Errorf("%s: got %s, want %s", input, got, want)
		}
	}
}
func TestGetRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403); w.Write([]byte("blocked")) }))
	defer server.Close()
	if _, err := get(server.URL); err == nil {
		t.Fatal("HTTP failure accepted")
	}
}
func TestMediaExtraction(t *testing.T) {
	matches := sources.FindAllStringSubmatch(`<img src="https://img.example/mw600/a.gif"><video controls src='https://img.example/large/b.mp4'></video>`, -1)
	if len(matches) != 2 {
		t.Fatalf("got %d media", len(matches))
	}
}
