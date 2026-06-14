package sysapproach

// Chapter is one chapter of Computer Networks: A Systems Approach.
type Chapter struct {
	Rank   int    `json:"rank"   csv:"rank"   tsv:"rank"`
	Number int    `json:"number" csv:"number" tsv:"number"`
	Title  string `json:"title"  csv:"title"  tsv:"title"`
	URL    string `json:"url"    csv:"url"    tsv:"url"`
}
