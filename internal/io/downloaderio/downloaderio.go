package downloaderio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gnames/gndict/internal/ent"
	"github.com/gnames/gndict/internal/ent/data"
	"github.com/gnames/gndict/pkg/config"
	"github.com/gnames/gnsys"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

var (
	names      = "names.txt"
	genera     = "genera.txt"
	weakAuthor = "uninomials-weak.txt"
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

func (d *downloaderio) Download(dat *data.Data) error {
	if !d.cfg.ForceDownload && d.downloadHappened() {
		log.Info().Msg("Download is already done, skipping...")
		return nil
	}

	log.Info().Msg("Starting creation of the names dump.")
	err := d.getNames(dat)
	if err != nil {
		return err
	}

	log.Info().Msg("Starting creation of genera dump.")
	err = d.getGenera()
	if err != nil {
		return err
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
func (d *downloaderio) getWeakUninomials() error {
	path := filepath.Join(d.cfg.CacheDir, weakAuthor)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	q := `
WITH uni AS (
  SELECT c.name,
    count(DISTINCT nsi.data_source_id) AS sources,
    count(DISTINCT nsi.data_source_id) FILTER (
      WHERE ns.name = c.name OR ns.name LIKE c.name || ' %'
    ) AS head_sources
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
SELECT u.name
  FROM uni u
  WHERE (u.sources = 1 OR u.head_sources = 0)
    AND EXISTS (
      SELECT 1 FROM words w
        -- type 4 is an author word
        WHERE w.normalized = u.name AND w.type_id = 4
    )
  ORDER BY u.name`
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
	log.Info().Msgf("Downloaded %d weak uninomials.", count)
	return nil
}

func (d *downloaderio) getNames(dat *data.Data) error {
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
`
	rows, err := d.db.Query(context.Background(), q)
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

	for k := range dat.ION {
		names[k] = struct{}{}
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
	weak := filepath.Join(d.cfg.CacheDir, weakAuthor)
	namesExist, _ := gnsys.FileExists(names)
	generaExist, _ := gnsys.FileExists(genera)
	weakExist, _ := gnsys.FileExists(weak)
	return namesExist && generaExist && weakExist
}
