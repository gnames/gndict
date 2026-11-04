package data

import (
	_ "embed"
	"strings"
)

//go:embed static/common-eu-words.txt
var common string

//go:embed static/species-black.txt
var spBlack string

//go:embed static/uninomials-black.txt
var uniBlack string

// genGrey are real genera that are also common words in some languages,
// names of places, people, journals, or capitalized species epithets. In
// texts they are mostly these words, so they are ambiguous genera and
// uninomials. Unlike common words, they do not make species epithets
// ambiguous.
//
//go:embed static/genera-grey.txt
var genGrey string

type Data struct {
	Common, SpBlack, UniBlack, GenGrey map[string]struct{}
	GenSp                              map[string][]string
}

func New() *Data {
	return &Data{
		Common:   toMap(common),
		SpBlack:  toMap(spBlack),
		UniBlack: toMap(uniBlack),
		GenGrey:  toMap(genGrey),
	}
}

func toMap(s string) map[string]struct{} {
	res := make(map[string]struct{})
	lines := strings.Split(s, "\n")
	for _, v := range lines {
		v = strings.TrimSpace(v)
		res[v] = struct{}{}
	}
	return res
}
