# GNdict

This app generates dictionaries for GNfinder binary.

It is used instead of older Rust-based app to generate dictionaries for
gnfinder.

<!-- vim-markdown-toc GFM -->

* [Dictionaries](#dictionaries)
* [Installation](#installation)
* [Configuration](#configuration)
* [Usage](#usage)
* [Updating GNfinder](#updating-gnfinder)
* [Compatibility with GNfinder](#compatibility-with-gnfinder)
* [Testing](#testing)

<!-- vim-markdown-toc -->

## Dictionaries

GNfinder uses dictionaries of known genera, species epithets and other
uninomials, their ambiguous variants (names that are also common words),
ambiguous genus-species combinations, common words, and words that are
never parts of names.

gndict creates them from names in the GNverifier database and the genera
of IRMNG. Names come from curated sources, GBIF, EOL, Ruhoff 1980 and ION.
ION contains many unique names, as well as OCR and human errors. Such
errors repeat in texts, so finding them is useful. All dictionaries are combined in one binary file, `dict.bin`,
which GNfinder embeds into its executable. Every word is stored there as a
64-bit hash with flags that tell which dictionaries contain it. The file
format is described in the `pkg/io/dict/hashdict` package of GNfinder.

gndict also writes every dictionary as a CSV file. GNfinder does not use
them, they are useful for reading and comparing dictionaries between
updates.

## Installation

Install Go v1.27 or higher and [just].

```bash
git clone git@github.com:/gnames/gndict
cd gndict
just install
```

## Configuration

gndict needs access to the PostgreSQL database of GNverifier. On the first
run it creates the configuration file `$HOME/.config/gndict.yaml`. Set the
database access there:

```yaml
PgHost: 0.0.0.0
PgUser: postgres
PgPass: postgres
PgDb: gnames
# CacheDir: ~/.cache/gndict
```

The values above are the defaults.

## Usage

```bash
gndict
```

gndict works in three steps:

1. Download. Names are downloaded from the database to the cache directory
   (`$HOME/.cache/gndict` by default). The download is reused on later
   runs. To download names again use:

    ```bash
    gndict -r
    ```

2. Preprocessing. Names are split into words, and counts of genera,
   species epithets and uninomials are saved to the cache directory.

3. Output. Dictionaries are created in `$HOME/.cache/gndict/dict`:

    | Path                          | Content                            |
    |-------------------------------|------------------------------------|
    | `hashdict/dict.bin`           | all dictionaries for GNfinder      |
    | `in/genera.csv`               | genera                             |
    | `in/species.csv`              | species epithets                   |
    | `in/uninomials.csv`           | other uninomials                   |
    | `in-ambig/genera.csv`         | ambiguous genera                   |
    | `in-ambig/species.csv`        | ambiguous species epithets         |
    | `in-ambig/uninomials.csv`     | ambiguous uninomials               |
    | `in-ambig/genera_species.csv` | names with ambiguous genera        |
    | `not-in/species.csv`          | words that are never epithets      |
    | `not-in/uninomials.csv`       | words that are never uninomials    |
    | `common/eu.csv`               | common words of European languages |

    A word is ambiguous if it is a common word, or is shorter than 4
    letters.

    Uninomials that are also authors' names and are weakly supported (come
    from one data source, or are known only as subgenera) are often
    artifacts. They are listed in `uninomials-weak.csv` in the cache
    directory. If such a name comes only from Wikispecies, it is nearly
    always a place, a person or a section header, so it goes to
    `not-in/uninomials.csv`. Other weak names are ambiguous uninomials.

To see the version:

```bash
gndict -V
```

## Updating GNfinder

Copy the binary dictionary to GNfinder and rebuild it:

```bash
cp ~/.cache/gndict/dict/hashdict/dict.bin \
  path_to/gnfinder/pkg/io/dict/data/hashdict/dict.bin
cd path_to/gnfinder
just test
just install
```

## Compatibility with GNfinder

gndict writes `dict.bin` with the `hashdict` package of GNfinder, so it has
to be built with a GNfinder version that has the same file format. GNfinder
checks the format version and a few known words when it loads `dict.bin`.
If they do not match, it stops with a "Cannot load dictionary" error caused
by an "incompatible dictionary file". In this case update the GNfinder
version in `go.mod`, rebuild gndict and generate the file again.

## Testing

```bash
just test
```

[just]: https://github.com/casey/just
