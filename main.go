package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

/* ============================
   Models
============================ */

type Link struct {
	ID         string            `json:"id"`
	URL        string            `json:"url"`
	Title      string            `json:"title"`
	Comment    string            `json:"comment"`
	Tags       []string          `json:"tags"`
	Labels     map[string]string `json:"labels"`
	OpenCount  int               `json:"open_count"`
	LastOpened time.Time         `json:"last_opened"`
}

type Store struct {
	Links []Link `json:"links"`
}

type Weights struct {
	Weights struct {
		Tag        float64 `json:"tag"`
		Label      float64 `json:"label"`
		Title      float64 `json:"title"`
		Comment    float64 `json:"comment"`
		Popularity float64 `json:"popularity"`
		Recency    float64 `json:"recency"`
	} `json:"weights"`
}

/* ============================
   Paths / Load
============================ */

func baseDir() string {
	dir, err := os.Getwd()

	if err != nil {
		panic("Unknow directory")
	}

	return filepath.Join(dir, ".linker")
}

func loadJSON(path string, v any) {
	b, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(b, v)
	}
}

/* ============================
   Scoring
============================ */

func scoreLink(l Link, w Weights, terms []string) float64 {
	var score float64

	title := strings.ToLower(l.Title)
	comment := strings.ToLower(l.Comment)

	for _, term := range terms {
		t := strings.ToLower(term)

		// tag match
		if contains(l.Tags, t) {
			score += w.Weights.Tag
		}

		// label match
		for k, v := range l.Labels {
			if t == k || t == v || t == k+"="+v {
				score += w.Weights.Label
			}
		}

		// title match
		if strings.Contains(title, t) {
			score += w.Weights.Title
		}

		// comment match
		if strings.Contains(comment, t) {
			score += w.Weights.Comment
		}
	}

	// popularity
	score += float64(l.OpenCount) * w.Weights.Popularity

	// recency
	if !l.LastOpened.IsZero() {
		days := time.Since(l.LastOpened).Hours() / 24
		score += w.Weights.Recency / (days + 1)
	}

	return score
}

func handleAdd() {
	if len(os.Args) < 3 {
		fmt.Println("usage: linker add <url> [flags]")
		return
	}

	url := os.Args[2]

	var (
		title   string
		comment string
		tags    []string
		labels  []string
	)

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--title":
			i++
			title = os.Args[i]
		case "--comment":
			i++
			comment = os.Args[i]
		case "--tag":
			i++
			tags = append(tags, os.Args[i])
		case "--label":
			i++
			labels = append(labels, os.Args[i])
		}
	}

	var store Store
	path := filepath.Join(baseDir(), "links.json")
	loadJSON(path, &store)

	link := Link{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		URL:       url,
		Title:     title,
		Comment:   comment,
		Tags:      tags,
		Labels:    parseLabels(labels),
		OpenCount: 0,
	}

	store.Links = append(store.Links, link)

	if err := os.MkdirAll(baseDir(), 0755); err != nil {
		panic(err)
	}

	if err := saveJSON(path, &store); err != nil {
		panic(err)
	}

	fmt.Println("added:", url)
}

/* ============================
   Helpers
============================ */

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if strings.ToLower(x) == s {
			return true
		}
	}
	return false
}

func saveJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func parseLabels(vals []string) map[string]string {
	m := make(map[string]string)
	for _, v := range vals {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

/* ============================
   Main
============================ */

func main() {
	var store Store
	var weights Weights

	weights = Weights{
		Weights: struct {
			Tag        float64 `json:"tag"`
			Label      float64 `json:"label"`
			Title      float64 `json:"title"`
			Comment    float64 `json:"comment"`
			Popularity float64 `json:"popularity"`
			Recency    float64 `json:"recency"`
		}{
			Tag:        2.0,
			Label:      1.5,
			Title:      3.0,
			Comment:    1.0,
			Popularity: 0.05,
			Recency:    0.02,
		},
	}

	loadJSON(filepath.Join(baseDir(), "links.json"), &store)
	// loadJSON(filepath.Join(baseDir(), "weights.json"), &weights)

	terms := os.Args[1:]

	if len(os.Args) > 1 && os.Args[1] == "add" {
		handleAdd()
		return
	}

	type scored struct {
		Link
		Score float64
	}

	var results []scored
	for _, l := range store.Links {
		score := scoreLink(l, weights, terms)
		if len(terms) == 0 || score > 0 {
			results = append(results, scored{l, score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) == 0 {
		fmt.Println("no matches")
		return
	}

	// fzf
	fzf := exec.Command("fzf", "--with-nth=2..")
	in, _ := fzf.StdinPipe()
	out, _ := fzf.StdoutPipe()
	_ = fzf.Start()

	go func() {
		for _, r := range results {
			fmt.Fprintf(in,
				"%.2f\t%s\t[%s]\n",
				r.Score,
				r.URL,
				strings.Join(r.Tags, ","),
			)
		}
		in.Close()
	}()

	buf := make([]byte, 2048)
	n, _ := out.Read(buf)
	_ = fzf.Wait()

	if n == 0 {
		return
	}

	url := strings.Split(string(buf[:n]), "\t")[1]

	openCmd := "xdg-open"
	if _, err := exec.LookPath("open"); err == nil {
		openCmd = "open"
	}
	_ = exec.Command(openCmd, url).Start()
}
