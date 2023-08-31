# Hangover - A WissKI Data Viewer

A __WissKI Data Viewer__ and __WissKI Exporter__ to display or export data found in a WissKI based on a pathbuilder and a triplestore export.

*Work in progress*

*Documentation may be oudated and incomplete*

## Installation

### from source

1. Install [Go](https://go.dev/), Version 1.21 or newer
2. Install [Yarn](https://yarnpkg.com/) (to build some frontend stuff)
3. Clone this repository somewhere.
4. Fetch dependencies:

```bash
make deps
```

5. Use the `Makefile` to build dependencies into the `dist` directory:

```bash
make all
```

5. Run the exectuables, either by placing them in your `$PATH` or telling your interpreter where they are directly.

As an alternative to steps 4 and 5, you may also run executables directly:

```bash
go run ./cmd/hangover arguments...
```

Replace `hangover` with the name of the executable you want to run.

### from a binary

We publish binaries for Mac, Linux and Windows for every release.
These can be found on the releases page on GitHub. 

## Usage

### hangover - A WissKI Viewer

The `hangover` executable implements a WissKI Viewer.
It is invoked with two parameters, the pathbuilder to a pathbuilder xml `xml` and triplestore `nquads` export.
It then starts up a server at `localhost:3000` by default.

For example:

```bash
hangover schreibkalender.xml schreibkalender.nq
```

It supports a various set of other options, which can be found using  `hangover -help`.
The most important ones are:

- `-html`, `-images`: Automatically display html and image content found within the WissKI export. By default, these are only displayed as text.
- `-public`: Set the _public URL_ this dump originates from, for example `https://wisski.example.com/`. This automatically finds all references to it within the data dump with references to the local viewer.
- `-cache`: By default all indexes of the dataset required by the viewer are constructed in main memory. This can take several gigabytes. Instead, you can specify a temporary directory to read and write temporary indexes from.
- `-export`: Index the entire dataset, then dump the export in binary into a file. Afterwards `hangover` can be invoked using only such a file (as opposed to a pathbuilder and triplestore export), skipping the indexing step. The file format may change between different builds of drincw and should be treated as a blackbox.

#### n2j - A WissKI Viewer

n2j stands for `NQuads 2 JSON` and can convert a WissKI export into json (or more general, relational) format.
Like `hangover`, it takes both a pathbuilder and export as an argument.
By default, it produces a single `.json` file on standard output.

Further options supports a various set of other options, which can be found using  `n2j -help`.

## Development

During development standard go tools are used.
Commands can be found in `./cmd/`.
Packages are documented and tested where applicable. 

Some files are generated, in particular the legal notices and frontend assets.
This requires some external tools written in go.
The frontend assets furthermore require node packages to be installed using [yarn](https://yarnpkg.com/).

A Makefile exists to simply the setup on a fresh system.
To install all (go) dependencies required for a build, run `make deps`.
To regenerate all assets, run `make generate`.
To build the `dist` directory, run `make all`.

go executables remain buildable without installing external dependencies.

## License

Licensed under the terms of [AGPL 3.0](https://github.com/FAU-CDI/hangover/blob/main/LICENSE) for everyone.
Aditionally licensed under the terms of the standard GPL license, version 3, for internal usage at FAU-CDI only. 
