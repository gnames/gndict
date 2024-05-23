package ent

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/gnames/gndict/internal/ent/data"
	"github.com/gnames/gndict/pkg/config"
	"github.com/gnames/gnsys"
)

type Output struct {
	cfg   config.Config
	sys   Sys
	dat   *data.Data
	genSp map[string][]string
}

func NewOutput(cfg config.Config, sys Sys, dat *data.Data) (*Output, error) {
	res := &Output{
		cfg: cfg,
		sys: sys,
		dat: dat,
	}

	dictDir := filepath.Join(cfg.CacheDir, "dict")
	err := gnsys.MakeDir(dictDir)
	if err != nil {
		err = fmt.Errorf("-> gnsys.MakeDir: %w", err)
		return nil, err
	}
	err = gnsys.CleanDir(dictDir)
	if err != nil {
		err = fmt.Errorf("-> gnsys.CleanDir: %w", err)
		return nil, err
	}
	for _, v := range []string{"common", "in", "in-ambig", "not-in"} {
		path := filepath.Join(dictDir, v)
		err = gnsys.MakeDir(path)
		if err != nil {
			err = fmt.Errorf("-> gnsys.MakeDir: %w", err)
			return nil, err
		}
	}

	return res, nil
}

func (o *Output) Create() error {
	err := o.uninomials()
	if err != nil {
		err = fmt.Errorf("-> o.uninomials: %w", err)
		return err
	}
	err = o.genera()
	if err != nil {
		err = fmt.Errorf("-> o.genera: %w", err)
		return err
	}
	err = o.species()
	if err != nil {
		err = fmt.Errorf("-> o.species: %w", err)
		return err
	}

	err = o.fromData()
	if err != nil {
		err = fmt.Errorf("-> o.fromData: %w", err)
		return err
	}

	return nil
}

func (o *Output) uninomials() error {
	var white, grey []string
	lines, err := o.sys.ReadFile("uninomials.csv")
	if err != nil {
		return err
	}

	for _, v := range lines {
		name, _, _ := strings.Cut(v, ",")
		if o.uninomialProblems(name) {
			continue
		}
		if o.isGreyWord(name) {
			grey = append(grey, v)
		} else {
			white = append(white, v)
		}
	}
	for _, v := range [][]string{white, grey} {
		slices.Sort(v)
	}
	return o.saveUniOrSp(white, grey, "uninomials.csv")

}

func (o *Output) saveUniOrSp(white, grey []string, file string) error {
	var err error
	err = o.saveStrings("in/"+file, white)
	if err != nil {
		err = fmt.Errorf("-> os.saveStrings: %w", err)
		return err
	}

	err = o.saveStrings("in-ambig/"+file, grey)
	if err != nil {
		err = fmt.Errorf("-> os.saveStrings: %w", err)
		return err
	}
	return nil
}

func (o *Output) genera() error {
	var white, grey, greySp []string
	lines, err := o.sys.ReadFile("genera.csv")
	if err != nil {
		err = fmt.Errorf("-> sys.ReadFile: %w", err)
		return err
	}

	for _, v := range lines {
		name, _, _ := strings.Cut(v, ",")
		if o.uninomialProblems(name) {
			continue
		}
		if o.isGreyWord(name) {
			grey = append(grey, v)
		} else {
			white = append(white, v)
		}
	}
	greySp, err = o.greySpecies(grey)
	if err != nil {
		err = fmt.Errorf("-> o.greySpecies: %w", err)
		return err
	}
	for _, v := range [][]string{white, grey, greySp} {
		sort.Strings(v)
	}
	return o.saveGen(white, grey, greySp)
}

func (o *Output) greySpecies(grey []string) ([]string, error) {
	gSp := make(map[string]map[string]struct{})
	for _, v := range grey {
		name, _, _ := strings.Cut(v, ",")
		gSp[name] = make(map[string]struct{})
	}
	names, err := o.sys.Canonicals()
	if err != nil {
		err = fmt.Errorf("-> sys.Canonicals: %w", err)
		return nil, err
	}

	for _, v := range names {
		words := strings.Split(v, " ")
		if len(words) < 2 {
			continue
		}
		wrd := words[0]
		if _, ok := gSp[wrd]; !ok {
			continue
		}
		for _, name := range nameCombos(words) {
			gSp[wrd][name] = struct{}{}
		}
	}
	var res []string
	for _, v := range gSp {
		for k := range v {
			res = append(res, k)
		}
	}
	sort.Strings(res)
	return res, nil
}

func nameCombos(words []string) []string {
	name := strings.Join(words, " ")
	if len(words) < 3 {
		return []string{name}
	}
	return []string{
		name,
		words[0] + " " + words[1],
		words[0] + " " + words[2],
	}
}

func (o *Output) saveGen(white, grey, greySp []string) error {
	var err error

	err = o.saveStrings("in/genera.csv", white)
	if err != nil {
		err = fmt.Errorf("-> os.saveStrings: %w", err)
		return err
	}

	err = o.saveStrings("in-ambig/genera.csv", grey)
	if err != nil {
		err = fmt.Errorf("-> os.saveStrings: %w", err)
		return err
	}

	err = o.saveStrings("in-ambig/genera_species.csv", greySp)
	if err != nil {
		err = fmt.Errorf("-> os.saveStrings: %w", err)
		return err
	}

	return nil
}

func (o *Output) species() error {
	var white, grey []string
	lines, err := o.sys.ReadFile("species.csv")
	if err != nil {
		return err
	}

	for _, v := range lines {
		name, _, _ := strings.Cut(v, ",")
		if o.speciesProblems(name) {
			continue
		}

		if o.isGreyWord(name) {
			grey = append(grey, v)
		} else {
			white = append(white, v)
		}

	}

	for _, v := range [][]string{white, grey} {
		sort.Strings(v)
	}
	return o.saveUniOrSp(white, grey, "species.csv")
}

func (o *Output) uninomialProblems(word string) bool {
	var res bool
	word = strings.ToLower(word)
	if _, ok := o.dat.UniBlack[word]; ok {
		res = true
	}

	if strings.Contains(word, ".") {
		res = true
	}
	return res
}

func (o *Output) speciesProblems(sp string) bool {
	spLow := strings.ToLower(sp)
	if _, ok := o.dat.SpBlack[spLow]; ok {
		return true
	}

	if len(sp) < 2 || strings.Contains(sp, ".") {
		return true
	}

	for _, v := range sp {
		if unicode.IsDigit(v) {
			return true
		}
	}
	return false
}

func (o *Output) isGreyWord(word string) bool {
	var res bool
	if len(word) < 4 {
		res = true
	}
	word = strings.ToLower(word)
	if _, ok := o.dat.Common[word]; ok {
		res = true
	}

	return res
}

func (o *Output) fromData() error {
	com := make([]string, len(o.dat.Common))
	var i int
	for k := range o.dat.Common {
		com[i] = k
		i++
	}

	blkSp := make([]string, len(o.dat.SpBlack))
	i = 0
	for k := range o.dat.SpBlack {
		blkSp[i] = k
		i++
	}

	blkUni := make([]string, len(o.dat.UniBlack))
	i = 0
	for k := range o.dat.UniBlack {
		blkUni[i] = k
		i++
	}

	for _, v := range [][]string{com, blkUni, blkSp} {
		sort.Strings(v)
	}

	return o.saveFromData(com, blkSp, blkUni)
}

func (o *Output) saveFromData(com, blkSp, blkUni []string) error {
	var err error
	err = o.saveStrings("common/eu.csv", com)
	if err != nil {
		err = fmt.Errorf("-> os.SaveStrings: %w", err)
		return err
	}

	err = o.saveStrings("not-in/species.csv", blkSp)
	if err != nil {
		err = fmt.Errorf("-> os.SaveStrings: %w", err)
		return err
	}

	err = o.saveStrings("not-in/uninomials.csv", blkUni)
	if err != nil {
		err = fmt.Errorf("-> os.SaveStrings: %w", err)
		return err
	}
	return nil
}

func (o *Output) saveStrings(path string, data []string) error {
	var f *os.File
	var err error
	f, err = os.Create(
		filepath.Join(o.cfg.CacheDir, "dict", path),
	)
	if err != nil {
		err = fmt.Errorf("-> os.Create: %w", err)
		return err
	}
	defer f.Close()

	s := strings.Join(data, "\n")
	s = strings.TrimSpace(s)
	_, err = f.WriteString(s)
	if err != nil {
		err = fmt.Errorf("-> WriteString: %w", err)
		return err
	}
	return nil
}
