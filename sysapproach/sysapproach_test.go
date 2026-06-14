package sysapproach_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/sysapproach-cli/sysapproach"
)

const fakeHTML = `<html><body>
<a href="foundation.html">Chapter 1:  Foundation</a>
<a href="direct.html">Chapter 2:  Direct Links</a>
<a href="foundation.html">Chapter 1:  Foundation</a>
</body></html>`

func newTestClient(ts *httptest.Server) *sysapproach.Client {
	cfg := sysapproach.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return sysapproach.NewClient(cfg)
}

func TestChapters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fakeHTML)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	chapters, err := c.Chapters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 2 {
		t.Fatalf("want 2 (deduped), got %d", len(chapters))
	}
	if chapters[0].Number != 1 {
		t.Errorf("Number[0] = %d", chapters[0].Number)
	}
	if chapters[0].Title != "Foundation" {
		t.Errorf("Title[0] = %q", chapters[0].Title)
	}
	if chapters[0].Rank != 1 {
		t.Errorf("Rank = %d", chapters[0].Rank)
	}
}
