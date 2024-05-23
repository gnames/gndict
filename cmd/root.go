/*
Copyright © 2023 Dmitry Mozzherin <dmozzherin@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnames/gndict/internal/io/downloaderio"
	"github.com/gnames/gndict/internal/io/sysio"
	gndict "github.com/gnames/gndict/pkg"
	"github.com/gnames/gndict/pkg/config"
	"github.com/gnames/gnsys"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed gndict.yaml
var configYAML string

var opts []config.Option

type cfgData struct {
	PgHost   string
	PgUser   string
	PgPass   string
	PgDb     string
	CacheDir string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gndict",
	Short: "gndict generates dictionaries for GNfinder",
	Long: `This is a service app, GNfinder uses dictionaries generated from
the GNverfier data. We use gndict to generate these dictionaries.`,
	Run: func(cmd *cobra.Command, args []string) {
		versionFlag(cmd)
		if redownloadFlag(cmd) {
			opts = append(opts, config.OptForceDownload(true))
		}
		cfg := config.New(opts...)

		dl := downloaderio.New(cfg)
		defer dl.Close()

		sys := sysio.New(cfg)

		dict := gndict.New(cfg, dl, sys)

		_ = dict
		err := dict.Download()
		if err != nil {
			err = fmt.Errorf("-> dict.Download: %w", err)
			log.Fatal().Err(err).Msg("Cannot download names")
		}

		err = dict.Preprocess()
		if err != nil {
			err = fmt.Errorf("-> dict.Preprocess: %w", err)
			log.Fatal().Err(err).Msg("Cannot Preprocess")
		}

		if err == nil {
			err = dict.Output()
			if err != nil {
				err = fmt.Errorf("-> dict.Output: %w", err)
				log.Fatal().Err(err).Msg("Cannot build output")
			}
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().BoolP("version", "V", false, "Show version")
	rootCmd.Flags().BoolP("redownload", "r", false, "Force reload from db")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	configFile := "gndict"
	cfgPath, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot find home directory")
	}
	cfgPath = filepath.Join(cfgPath, ".config")

	// Search config in home directory with name ".gnames" (without extension).
	viper.AddConfigPath(cfgPath)
	viper.SetConfigName(configFile)

	configPath := filepath.Join(cfgPath, fmt.Sprintf("%s.yaml", configFile))
	touchConfigFile(configPath)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msgf("Using config file: %s.", viper.ConfigFileUsed())
	}
	getOpts()
}

func getOpts() []config.Option {
	cfg := &cfgData{}
	err := viper.Unmarshal(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot deserialize config data")
	}

	if cfg.CacheDir != "" {
		opts = append(opts, config.OptCacheDir(cfg.CacheDir))
	}
	if cfg.PgHost != "" {
		opts = append(opts, config.OptPgHost(cfg.PgHost))
	}
	if cfg.PgUser != "" {
		opts = append(opts, config.OptPgUser(cfg.PgUser))
	}
	if cfg.PgPass != "" {
		opts = append(opts, config.OptPgPass(cfg.PgPass))
	}
	if cfg.PgDb != "" {
		opts = append(opts, config.OptPgDb(cfg.PgDb))
	}
	return opts
}

func versionFlag(cmd *cobra.Command) {
	b, _ := cmd.Flags().GetBool("version")
	if !b {
		return
	}
	fmt.Printf("\nVersion: %s\n\nBuild: %s\n\n", gndict.Version, gndict.Build)
	os.Exit(0)
}

func redownloadFlag(cmd *cobra.Command) bool {
	b, _ := cmd.Flags().GetBool("redownload")
	return b
}

// touchConfigFile checks if config file exists, and if not, it gets created.
func touchConfigFile(configPath string) {
	if ok, err := gnsys.FileExists(configPath); ok && err == nil {
		return
	}

	log.Info().Msgf("Creating config file '%s'", configPath)
	createConfig(configPath)
}

// createConfig creates config file.
func createConfig(path string) {
	err := gnsys.MakeDir(filepath.Dir(path))
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot create dir %s", path)
	}

	err = os.WriteFile(path, []byte(configYAML), 0644)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot write to file %s", path)
	}
}
