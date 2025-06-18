# pocket-obsidian <!-- omit from toc -->
Convert pocket CSV's into obsidian clippings markdown files

[Mozilla Pocket is EOL'd](https://support.mozilla.org/en-US/kb/future-of-pocket). Which is not ideal. 
You can download your clippings by following these instructions: 
- https://support.mozilla.org/en-US/kb/exporting-your-pocket-list
- This will give you a CSV file containing your saves. 

I liked this article about moving from Mozilla Pocket to [Obsidian](https://obsidian.md/)
- https://obsidian.rocks/the-best-free-pocket-alternative-obsidian/

So I decided to give it a whirl. Follow the instructions it's really straighforward. 
However, there is no direct way to convert this CSV into Obsidian, so I wrote one. 
Feedback gladly received. 

- [Download a release](#download-a-release)
- [To build from source](#to-build-from-source)
- [Usage](#usage)
  

# Download a release

You can get releases from the releases page [Releases](https://github.com/fergalsomers/pocket-obsidian/releases)

- Download the one most appropriate for your system
- if you are on a Mac you will need to use Security Manager to let it 

# To build from source

Alternatively, just build from source. 

Pre-requiisite have Golang installed (e.g. https://go.dev/doc/install or  `brew install go`)

```
git clone https://github.com/fergalsomers/pocket-obsidian.git
make
```

# Usage

```
./pocket-obsidian [CSV file]
```

This will create an `archive` directory containing the generated markdown files `.md`. If any of the URL's don't work anymore they will be written to a `failed.csv` file - so you can check their errors. 

Some handy options:
-  Change the location of the archive directory ` -o [archive dir]`
-  The location of the failure csv file `-f [failure CSV file] `
-  Also use `-r` to automatically mark all imported clippings as `read`. 

For help:

```
./pocket-obsidian --help
Usage of pocket-obsidian [input-csv-file]
  -f, --fail-csv string     Default tags to write failed entries to (default "/Users/fergalsomers/build/git/pocket-obsidian/failed.csv")
  -o, --output-dir string   Directory to write output files to defaults to ./archive (default "/Users/fergalsomers/build/git/pocket-obsidian/archive")
  -r, --read                Mark articles as read in Pocket
  -t, --tags stringArray    Default tags to add to all csv entries, defaults to clippings (per obsidian webclipper plugin) (default [clippings,pocket])
pocket-obsidian: Convert Mozilla Pocket exported CSV to Obsidian Markdown. See https://github.com/fergalsomers/pocket-obsidian
``` 

