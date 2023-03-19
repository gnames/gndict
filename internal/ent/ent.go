package ent

import "github.com/gnames/gndict/internal/ent/data"

type Downloader interface {
	// Download connects to gnames database and downloads canonical forms of
	// names from sources that are considered to be 'reliable'.
	// Then it downloads generic names from the IRMNG project.
	// These files will be first preprocessed, than the data will be converted
	// to the output for gnfinder.
	Download(dat *data.Data) error
	// Close cleans up database connections.
	Close() error
}

type Sys interface {
	Names() ([]string, error)
	Canonicals() ([]string, error)
	Genera() ([]string, error)
	ReadFile(path string) ([]string, error)
}
