package ent

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gnames/gndict/internal/ent/data"
	"github.com/gnames/gndict/pkg/config"
	"github.com/gnames/gnfmt"
)

type Preproc struct {
	cfg                         config.Config
	sys                         Sys
	dat                         *data.Data
	names                       []string
	genMap, canonical           map[string]struct{}
	uninomials, genera, species map[string]int
}

func NewPreproc(cfg config.Config, sys Sys, dat *data.Data) (*Preproc, error) {
	names, err := sys.Names()
	if err != nil {
		return nil, err
	}

	genera, err := sys.Genera()
	if err != nil {
		return nil, err
	}
	genMap := make(map[string]struct{})
	for i := range genera {
		genMap[genera[i]] = struct{}{}
	}
	res := Preproc{
		cfg:        cfg,
		sys:        sys,
		dat:        dat,
		names:      names,
		genMap:     genMap,
		uninomials: make(map[string]int),
		genera:     make(map[string]int),
		species:    make(map[string]int),
		canonical:  make(map[string]struct{}),
	}
	return &res, nil
}

func (p *Preproc) Preprocess() error {
	var err error
	for _, v := range p.names {
		if strings.ContainsRune(v, '×') {
			continue
		}
		p.words(v)
	}
	p.cleanupUni()

	err = p.makeCSV(p.uninomials, "uninomials.csv")
	if err != nil {
		err := fmt.Errorf("-> p.makeCSV: %w", err)
		return err
	}
	err = p.makeCSV(p.genera, "genera.csv")
	if err != nil {
		err := fmt.Errorf("-> p.makeCSV: %w", err)
		return err
	}
	err = p.makeCSV(p.species, "species.csv")
	if err != nil {
		err := fmt.Errorf("-> p.makeCSV: %w", err)
		return err
	}

	err = p.makeCanonicals()
	if err != nil {
		err := fmt.Errorf("-> p.makeCanonicals: %w", err)
		return err
	}

	return nil
}

func (p *Preproc) makeCanonicals() error {
	path := filepath.Join(p.cfg.CacheDir, "canonicals.csv")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for k := range p.canonical {
		_, err = f.WriteString(k + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Preproc) makeCSV(dat map[string]int, file string) error {
	path := filepath.Join(p.cfg.CacheDir, file)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for k, v := range dat {
		row := gnfmt.ToCSV([]string{k, strconv.Itoa(v)}, ',')
		_, err = f.WriteString(row + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Preproc) words(s string) {
	words := strings.Split(s, " ")

	if len(words) == 1 {
		p.wordsUni(words[0])
	}

	if len(words) > 1 {
		p.wordsSp(words)
	}
}

func (p *Preproc) wordsUni(s string) {
	if _, ok := p.genMap[s]; ok {
		p.genera[s] += 1
		return
	}
	p.uninomials[s] += 1
}

func (p *Preproc) wordsSp(words []string) {
	var idxBlkSp int
	for i, v := range words[1:] {
		if _, ok := p.dat.SpBlack[v]; ok {
			if idxBlkSp == 0 {
				idxBlkSp = i + 1
			}
			continue
		}
		p.species[v] += 1
	}
	// if bad word follows uninomial, do not save
	if idxBlkSp > 0 && idxBlkSp < 2 {
		return
	}

	// if bad word happens after a reasonable species word, save genera
	// and save canonical
	p.genera[words[0]] += 1
	if idxBlkSp == 0 {
		p.canonical[strings.Join(words, " ")] = struct{}{}
		return
	}
	p.canonical[strings.Join(words[0:idxBlkSp], " ")] = struct{}{}
}

func (p *Preproc) cleanupUni() {
	// at this point we found some genera that is not in IRMNG, move it from
	// uninomials to genera.
	var genera []string
	for k, v := range p.uninomials {
		if _, ok := p.genera[k]; ok {
			p.genera[k] += v
			genera = append(genera, k)
		}
	}
	for _, v := range genera {
		delete(p.uninomials, v)
	}
}
