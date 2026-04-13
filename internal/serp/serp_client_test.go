package serp

import "testing"

func TestPickApplyLink(t *testing.T) {
	google := "https://www.google.com/search?ibp=htl;jobs&q=test"
	zip := "https://www.ziprecruiter.com/job/123?utm_campaign=google_jobs_apply"

	opts := []struct {
		Title string `json:"title"`
		Link  string `json:"link"`
	}{
		{Title: "Google", Link: google},
		{Title: "Zip", Link: zip},
	}
	if got := pickApplyLink("https://share.example", opts); got != zip {
		t.Fatalf("expected zip link when first option is google shell, got %q", got)
	}

	opts2 := []struct {
		Title string `json:"title"`
		Link  string `json:"link"`
	}{{Link: zip}}
	if got := pickApplyLink(google, opts2); got != zip {
		t.Fatalf("expected apply option, got %q", got)
	}

	if got := pickApplyLink(google, nil); got != google {
		t.Fatalf("expected share link fallback, got %q", got)
	}
}

func TestIsGoogleJobsShellURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.google.com/search?ibp=htl;jobs&q=x", true},
		{"https://www.google.com/search?q=foo&htivrt=jobs", true},
		{"https://www.ziprecruiter.com/c/Co/Job", false},
		{"https://www.linkedin.com/jobs/view/123", false},
	}
	for _, c := range cases {
		if got := isGoogleJobsShellURL(c.url); got != c.want {
			t.Errorf("isGoogleJobsShellURL(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}
