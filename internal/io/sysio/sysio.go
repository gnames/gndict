package sysio

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/gnames/gndict/internal/ent"
	"github.com/gnames/gndict/pkg/config"
)

type sysio struct {
	cfg config.Config
}

func New(cfg config.Config) ent.Sys {
	return sysio{cfg: cfg}
}

func (s sysio) Names() ([]string, error) {
	return s.ReadFile("names.txt")
}

func (s sysio) Canonicals() ([]string, error) {
	return s.ReadFile("canonicals.csv")
}

func (s sysio) Genera() ([]string, error) {
	return s.ReadFile("genera.txt")
}

func (s sysio) ReadFile(fname string) ([]string, error) {
	var res []string
	path := filepath.Join(s.cfg.CacheDir, fname)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		res = append(res, txt)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}
