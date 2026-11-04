package ent

type Downloader interface {
	// Download connects to gnames database and downloads canonical forms of
	// names from sources that are considered to be 'reliable' and from ION.
	// Then it downloads generic names from the IRMNG project.
	// These files will be first preprocessed, than the data will be converted
	// to the output for gnfinder.
	Download() error
	// Close cleans up database connections.
	Close() error
}

type Sys interface {
	Names() ([]string, error)
	Canonicals() ([]string, error)
	Genera() ([]string, error)
	ReadFile(path string) ([]string, error)
}
