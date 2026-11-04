package ent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnames/gndict/internal/ent/data"
	"github.com/gnames/gndict/pkg/config"
	"github.com/gnames/gnfinder/pkg/io/dict/hashdict"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSys returns preprocessed files from memory.
type fakeSys map[string][]string

func (fs fakeSys) Names() ([]string, error)      { return nil, nil }
func (fs fakeSys) Genera() ([]string, error)     { return nil, nil }
func (fs fakeSys) Canonicals() ([]string, error) { return fs["canonicals.csv"], nil }
func (fs fakeSys) ReadFile(path string) ([]string, error) {
	return fs[path], nil
}

func newFakeSys() fakeSys {
	return fakeSys{
		"uninomials.csv": {
			"Plantaginaceae,12", "Minimi,1", "Monterosato,1", "Missouri,1",
		},
		"uninomials-weak.csv": {"Missouri,true", "Monterosato,false"},
		"genera.csv":          {"Plantago,40", "America,3", `"Odd,Name",1`},
		"species.csv":         {"plantaginea,7", "amabilis,4", "all,2"},
		"canonicals.csv":      {"America amabilis", "Plantago major"},
	}
}

func createOutput(t *testing.T) string {
	dir := t.TempDir()
	cfg := config.New(config.OptCacheDir(dir))
	o, err := NewOutput(cfg, newFakeSys(), data.New())
	require.NoError(t, err)
	require.NoError(t, o.Create())
	return filepath.Join(dir, "dict")
}

func loadHashDict(t *testing.T, dictDir string) *hashdict.Dict {
	bs, err := os.ReadFile(filepath.Join(dictDir, hashdict.Path))
	require.NoError(t, err)
	d, err := hashdict.Load(bs)
	require.NoError(t, err)
	return d
}

func TestHashDict(t *testing.T) {
	assert := assert.New(t)
	d := loadHashDict(t, createOutput(t))

	tests := []struct {
		word  string
		flags hashdict.Flags
	}{
		{"Plantago", hashdict.InGenus},
		{"Plantaginaceae", hashdict.InUninomial},
		// weakly supported author's name is an ambiguous uninomial
		{"Monterosato", hashdict.InAmbigUninomial},
		// weakly supported author's name only from Wikispecies is never
		// a uninomial, gnfinder checks it lowercased
		{"missouri", hashdict.NotInUninomial},
		{"plantaginea", hashdict.InSpecies},
		// America is a common word, so it is an ambiguous genus
		{"America", hashdict.InAmbigGenus},
		{"America amabilis", hashdict.InAmbigGenusSp},
		{"america", hashdict.CommonWord},
		// only the first CSV column is a word
		{"Odd,Name", hashdict.InGenus},
	}
	for _, v := range tests {
		assert.True(d.Lookup(v.word).Has(v.flags), v.word)
	}
	assert.False(d.Lookup("Plantago").Has(hashdict.CommonWord))
	assert.False(d.Lookup("Monterosato").Has(hashdict.InUninomial))
	assert.Equal(hashdict.Flags(0), d.Lookup("Missouri"))
	assert.Equal(hashdict.Flags(0), d.Lookup("Notawordatall"))
	assert.Equal(hashdict.Flags(0), d.Lookup(""))
}

// TestHashDictMatchesCSV makes sure the binary dictionary has the same words
// and flags as the CSV files created in the same run.
func TestHashDictMatchesCSV(t *testing.T) {
	dictDir := createOutput(t)
	d := loadHashDict(t, dictDir)

	sources := []struct {
		csv   string
		flags hashdict.Flags
	}{
		{"in/genera.csv", hashdict.InGenus},
		{"in/species.csv", hashdict.InSpecies},
		{"in/uninomials.csv", hashdict.InUninomial},
		{"in-ambig/genera.csv", hashdict.InAmbigGenus},
		{"in-ambig/species.csv", hashdict.InAmbigSpecies},
		{"in-ambig/uninomials.csv", hashdict.InAmbigUninomial},
		{"in-ambig/genera_species.csv", hashdict.InAmbigGenusSp},
		{"not-in/uninomials.csv", hashdict.NotInUninomial},
		{"not-in/species.csv", hashdict.NotInSpecies},
		{"common/eu.csv", hashdict.CommonWord},
	}
	b := hashdict.NewBuilder()
	for _, v := range sources {
		f, err := os.Open(filepath.Join(dictDir, v.csv))
		require.NoError(t, err)
		require.NoError(t, b.Add(f, v.flags), v.csv)
		_ = f.Close()
	}
	fromCSV, err := b.Build()
	require.NoError(t, err)

	var buf1, buf2 bytes.Buffer
	require.NoError(t, d.Save(&buf1))
	require.NoError(t, fromCSV.Save(&buf2))
	assert.Equal(t, d.Len(), fromCSV.Len())
	assert.Equal(t, buf2.Bytes(), buf1.Bytes(),
		"binary and CSV dictionaries differ")
}
