package gndict

import (
	"fmt"

	"github.com/gnames/gndict/internal/ent"
	"github.com/gnames/gndict/internal/ent/data"
	"github.com/gnames/gndict/pkg/config"
	"github.com/rs/zerolog/log"
)

type gndict struct {
	cfg config.Config
	sys ent.Sys
	dat *data.Data
	ent.Downloader
}

func New(
	cfg config.Config,
	dl ent.Downloader,
	sys ent.Sys,
) DictGen {
	dat := data.New()
	return &gndict{cfg: cfg, Downloader: dl, sys: sys, dat: dat}
}

func (d *gndict) Download() error {
	return d.Downloader.Download(d.dat)
}

func (d *gndict) Preprocess() error {
	log.Info().Msg("Start Preprocessing")
	ppr, err := ent.NewPreproc(d.cfg, d.sys, d.dat)
	if err != nil {
		err = fmt.Errorf("-> ent.NewPreproc: %w", err)
		return err
	}
	return ppr.Preprocess()
}

func (d *gndict) Output() error {
	log.Info().Msg("Creating Output")
	o, err := ent.NewOutput(d.cfg, d.sys, d.dat)
	if err != nil {
		err = fmt.Errorf("-> ent.NewOutput: %w", err)
		return err
	}
	return o.Create()
}
