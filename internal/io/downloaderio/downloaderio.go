package downloaderio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gnames/gndict/internal/ent"
	"github.com/gnames/gndict/pkg/config"
	"github.com/gnames/gnsys"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

var (
	names      = "names.txt"
	genera     = "genera.txt"
	weakAuthor = "uninomials-weak.csv"

	// wikispeciesID is the ID of Wikispecies data source in GNverifier.
	wikispeciesID = 2

	// ionID is the ID of ION data source in GNverifier.
	ionID = 168
)

type downloaderio struct {
	cfg config.Config
	db  *pgx.Conn
}

func New(cfg config.Config) ent.Downloader {
	db, err := pgx.Connect(context.Background(), getURL(cfg))
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create PostgreSQL connection")
	}
	exist, _, _ := gnsys.DirExists(cfg.CacheDir)
	if !exist {
		log.Info().Msgf("Dir %s does not exist, creating.", cfg.CacheDir)
		err := gnsys.MakeDir(cfg.CacheDir)
		if err != nil {
			log.Fatal().Err(err).Msgf("Cannot make dir %s", cfg.CacheDir)
		}
	}
	return &downloaderio{cfg: cfg, db: db}
}

func (d *downloaderio) Close() error {
	return d.db.Close(context.Background())
}

func (d *downloaderio) Download() error {
	var err error
	if !d.cfg.ForceDownload && d.downloadHappened() {
		log.Info().Msg("Download of names is already done, skipping...")
	} else {
		log.Info().Msg("Starting creation of the names dump.")
		err = d.getNames()
		if err != nil {
			return err
		}

		log.Info().Msg("Starting creation of genera dump.")
		err = d.getGenera()
		if err != nil {
			return err
		}
	}

	// weak uninomials are downloaded separately, so changes in their format
	// do not require a long download of all names.
	weak := filepath.Join(d.cfg.CacheDir, weakAuthor)
	weakExist, _ := gnsys.FileExists(weak)
	if !d.cfg.ForceDownload && weakExist {
		log.Info().Msg("Download of weak uninomials is already done, skipping...")
		return nil
	}
	log.Info().Msg("Starting creation of weak uninomials dump.")
	err = d.getWeakUninomials()
	if err != nil {
		return err
	}
	return nil
}

// getWeakUninomials saves one-word names that are weakly supported and are
// also used as authors' names. They are often artifacts: an author in
// parentheses parsed as a subgenus (`Anisocycla (Monterosato)`), or a place
// or a person in a data source (`Missouri`). A name is weakly supported if
// it comes from only one data source, or never starts a name-string (it is
// known only as a subgenus).
//
// Every line of the file is `name,wikispecies_only`, where wikispecies_only
// is true if the name comes only from Wikispecies. Such names are mostly
// artifacts of Wikispecies dump, where places, people and section headers
// can be entered as names.
func (d *downloaderio) getWeakUninomials() error {
	// the file is written under a temporary name, so a failed download does
	// not leave a file that would be taken for a finished one.
	path := filepath.Join(d.cfg.CacheDir, weakAuthor)
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}()
	q := `
WITH uni AS (
  SELECT c.name,
    count(DISTINCT nsi.data_source_id) AS sources,
    count(DISTINCT nsi.data_source_id) FILTER (
      WHERE ns.name = c.name OR ns.name LIKE c.name || ' %'
    ) AS head_sources,
    bool_and(nsi.data_source_id = $1) AS wikispecies_only
  FROM canonicals c
    JOIN name_strings ns ON ns.canonical_id = c.id
    JOIN name_string_indices nsi ON nsi.name_string_id = ns.id
    JOIN data_sources ds ON ds.id = nsi.data_source_id
  WHERE (ds.is_curated = true
      OR nsi.data_source_id = 11
      OR nsi.data_source_id = 12
      OR nsi.data_source_id = 206)
    AND position(' ' in c.name) = 0
  GROUP BY c.name
)
SELECT u.name, u.wikispecies_only
  FROM uni u
  WHERE (u.sources = 1 OR u.head_sources = 0)
    AND EXISTS (
      SELECT 1 FROM words w
        -- type 4 is an author word
        WHERE w.normalized = u.name AND w.type_id = 4
    )
  ORDER BY u.name`
	rows, err := d.db.Query(context.Background(), q, wikispeciesID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var name string
	var wsOnly bool
	var count, wsCount int
	for rows.Next() {
		err = rows.Scan(&name, &wsOnly)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(f, "%s,%t\n", name, wsOnly)
		if err != nil {
			return err
		}
		count++
		if wsOnly {
			wsCount++
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		return err
	}
	log.Info().Msgf(
		"Downloaded %d weak uninomials, %d of them only from Wikispecies.",
		count, wsCount,
	)
	return nil
}

// getNames saves canonical forms of names from reliable sources and from
// ION. ION names come from many people and contain unique names, as well
// as OCR and human errors. Such errors repeat in texts, so they are useful
// for finding names. ION is not used for other downloads, because it would
// make weakly supported names look supported.
func (d *downloaderio) getNames() error {
	names := make(map[string]struct{})
	path := filepath.Join(d.cfg.CacheDir, "names.txt")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	q := `
	SELECT DISTINCT c.name
	    FROM canonicals c
	        JOIN name_strings ns
	            ON ns.canonical_id = c.id
	        JOIN name_string_indices nsi
	            ON nsi.name_string_id = ns.id
	        JOIN data_sources ds
	            ON ds.id = nsi.data_source_id
	    WHERE ds.is_curated = true
			OR nsi.data_source_id = 11
			OR nsi.data_source_id = 12
			OR nsi.data_source_id = 206
			OR nsi.data_source_id = $1
`
	rows, err := d.db.Query(context.Background(), q, ionID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var name string
	for rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			return err
		}
		names[name] = struct{}{}
	}
	if err = rows.Err(); err != nil {
		return err
	}

	namesAry := make([]string, len(names))
	var i int
	for k := range names {
		namesAry[i] = k
		i++
	}

	sort.Strings(namesAry)

	for _, name := range namesAry {
		_, err = f.WriteString(name + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *downloaderio) getGenera() error {
	path := filepath.Join(d.cfg.CacheDir, "genera.txt")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// IRMNG data source ID is 181
	q := `
SELECT DISTINCT c.name
    FROM name_string_indices nsi
        JOIN name_strings ns on ns.id = nsi.name_string_id
        JOIN canonicals c on c.id = ns.canonical_id
    WHERE data_source_id = 181 AND lower(rank) = 'genus'`
	rows, err := d.db.Query(context.Background(), q)
	if err != nil {
		return err
	}
	defer rows.Close()
	var name string
	var count int
	for rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			return err
		}
		_, err = f.WriteString(name + "\n")
		if err != nil {
			return err
		}
		count++
	}
	if err = rows.Err(); err != nil {
		return err
	}

	// without IRMNG genera all genera would come only from binomials
	if count == 0 {
		_ = os.Remove(path)
		return errors.New("no IRMNG genera found in the database")
	}
	log.Info().Msgf("Downloaded %d IRMNG genera.", count)
	return nil
}

func getURL(cfg config.Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
		cfg.PgUser, cfg.PgPass, cfg.PgHost, cfg.PgDb)
}

func (d *downloaderio) downloadHappened() bool {
	names := filepath.Join(d.cfg.CacheDir, "names.txt")
	genera := filepath.Join(d.cfg.CacheDir, "genera.txt")
	namesExist, _ := gnsys.FileExists(names)
	generaExist, _ := gnsys.FileExists(genera)
	return namesExist && generaExist
}
