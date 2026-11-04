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

type Data struct {
	Common, SpBlack, UniBlack map[string]struct{}
	GenSp                     map[string][]string
}

func New() *Data {
	return &Data{
		Common:   toMap(common),
		SpBlack:  toMap(spBlack),
		UniBlack: toMap(uniBlack),
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
