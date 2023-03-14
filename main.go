package main

import (
	"strings"
	"os"
	"flag"
	"sort"
	"fmt"
	"log"
	"github.com/go-nlp/bm25"
	"github.com/go-nlp/tfidf"

	"github.com/gohugoio/hugo/hugofs"
	"github.com/gohugoio/hugo/hugolib"
	"github.com/gohugoio/hugo/deps"
	"github.com/gohugoio/hugo/resources/resource"
)

type doc []int

func (d doc) IDs() []int { return []int(d) }


func makeCorpus(a []string) (map[string]int, []string) {
	retVal := make(map[string]int)
	invRetVal := make([]string, 0)
	var id int
	for _, s := range a {
		for _, f := range strings.Fields(s) {
			f = strings.ToLower(f)
			if _, ok := retVal[f]; !ok {
				retVal[f] = id
				invRetVal = append(invRetVal, f)
				id++
			}
		}
	}
	return retVal, invRetVal
}

func makeDocuments(a []string, c map[string]int) []tfidf.Document {
	retVal := make([]tfidf.Document, 0, len(a))
	for _, s := range a {
		var ts []int
		for _, f := range strings.Fields(s) {
			f = strings.ToLower(f)
			id := c[f]
			ts = append(ts, id)
		}
		retVal = append(retVal, doc(ts))
	}
	return retVal
}

func indexHugo() []string{
	inv := []string{}
	osFs := hugofs.Os
	cwd, _ := os.Getwd()
	cfg, _, err := hugolib.LoadConfig(
		hugolib.ConfigSourceDescriptor{
			Fs:         osFs,
			Filename:   "config.toml",
			Path:       cwd,
			WorkingDir: cwd,
		})
	if err != nil {
		wd, _ := os.Getwd()
		log.Fatal("Could not load Hugo config.toml (cwd=", wd, "): ", err)
	}
	fmt.Println(cfg)
	fs := hugofs.NewFrom(osFs, cfg)
	fmt.Println(fs)
	sites, err := hugolib.NewHugoSites(deps.DepsCfg{Fs: fs, Cfg: cfg})
	if err != nil {
		log.Fatal("Could not load Hugo site(s): ", err)
	}
	fmt.Println(sites)
	err = sites.Build(hugolib.BuildCfg{SkipRender: true})
	if err != nil {
		log.Fatal("Could not run render: ", err)
	}
	for _, p := range sites.Pages() {
		if p.Draft() || resource.IsFuture(p) || resource.IsExpired(p) {
			continue
		}
		//title := p.Description()
		//path := p.Permalink()
		content := p.Plain()
		//fmt.Printf("%v, %v\n\n",title, path)
		//fmt.Println(content)
		inv = append(inv, content)
	}
	return inv
}

func main() {
	//
	// Flag
	term := flag.String("term", "test", "term to search")
	flag.Parse()

	// Parse doc
	inventaire := indexHugo()

	corpus, _ := makeCorpus(inventaire)
	docs := makeDocuments(inventaire, corpus)
	tf := tfidf.New()

	for _, doc := range docs {
		tf.Add(doc)
	}
	tf.CalculateIDF()
	// now we search
	fmt.Println(inventaire)

	// "ishmael" is a query
	corpus_term := doc{corpus[*term]}

	corpusScores := bm25.BM25(tf, corpus_term, docs, 1.5, 0.75)

	sort.Sort(sort.Reverse(corpusScores))

	fmt.Printf("Top 3 Relevant Docs to \"%s\":\n", *term)
	for _, d := range corpusScores[:3] {
		fmt.Printf("\tID   : %d\n\tScore: %1.3f\n\tDoc  : %10.30q\n", d.ID, d.Score, inventaire[d.ID])
	}
}
