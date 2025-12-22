package options

import (
	"flag"
	// "fmt"
	"github.com/mitchellh/colorstring"
	"os"
	"strings"
)

type options struct {
	List           bool
	Download       bool
	SkipExtraction bool
	Concurrency    int
	GHToken        string
	TempDir        string
	Version        bool
}

func ParseFlags() options {

	flag.Usage = func() {
		h := []string{
			"",
			"Download releases from github and retain only the binaries",
			"",
			"[light_cyan]Usage:[reset]",
			"",
			"  echo 'https://github.com/sharkdp/bat' | getghrel -list | sort",
			"  cat urls.txt | getghrel -list | sort | tee releases.txt",
			"  cat releases.txt | getghrel -download",
			"  cat releases.txt | getghrel -download -tempdir '/tmp/bin'",
			"  cat urls.txt | getghrel -list | grep -vi '^n/a'",
			" ",
			"[light_cyan]The url format for -list[reset]: \n",
			"  A github url -> 'https://github.com/owner/repo'",
			"  For e.g 'https://github.com/sharkdp/bat'\n",
			"  Or only owner and repository -> 'owner/repo'",
			"  For e.g -> 'sharkdp/bat'\n",
			"For more examples, browse to: https://github.com/kavishgr/getghrel",
			"",
			"Options:",
			"  [light_cyan]-list[reset]",
			"",
			"\tWill list all the release/releases found for your OS and Architecture.\n",
			"\tExample: cat urls.txt | getghrel -list | sort",
			"\tExample: echo 'https://github.com/sharkdp/bat' | getghrel -list | sort",
			"\tExample: echo 'sharkdp/bat' | getghrel -list | sort",
			"",
			"  [light_cyan]-con[reset]",
			"",
			"\t Set the concurrency level (default: 2)\n",
			"\t Example: cat urls.txt | getghrel -list -con 3 | tee releases.txt",
			"\t Example: cat releases.txt | getghrel -download -con 3",
			"",
			"  [light_cyan]-ghtoken[reset]",
			"",
			"\t Specify your GITHUB TOKEN",
			"\t Default is the GIHUB_TOKEN environment variable.\n",
			"\t Example: cat urls.txt | getghrel -list -ghtoken 'YOUR TOKEN'",
			"",
			"  [light_cyan]-download[reset]",
			"",
			"\t Download the releases",
			"\t Default directory in which the release will be downloaded is '/tmp/getghrel'",
			"\t If the release is compressed or in an archive format, the tool will automatically",
			"\t extract and unpack it no matter how it's compressed or archived",
			"\t and keep only the binary.\n",
			"\t Example: cat releases.txt | getghrel -download",
			"\t Example: cat releases.txt | getghrel -download -tempdir '/tmp/test'",
			"",
			"  [light_cyan]-skipextraction[reset]",
			"",
			"\t Skip the extraction/unpack process\n",
			"\t Example: echo \"neovim/neovim\" | getghrel -list | getghrel -download -skipextraction",
			"",
			"  [light_cyan]-tempdir[reset] ",
			"",
			"\t Specify a temporary directory to download/extract the binaries\n",
			"\t Example: cat releases.txt | getghrel -download -tempdir '/tmp/test'",
			"",
			"  [light_cyan]-version[reset]",
			"\t Print version\n",
			"",
		}
		help := strings.Join(h, "\n")

		// fmt.Fprint(os.Stderr, strings.Join(h, "\n"))
		colorstring.Println(help)
	}

	opts := options{}
	flag.BoolVar(&opts.Download, "download", false, "")
	flag.BoolVar(&opts.List, "list", false, "")
	flag.BoolVar(&opts.SkipExtraction, "skipextraction", false, "")
	flag.IntVar(&opts.Concurrency, "con", 2, "")
	default_ghtoken := os.Getenv("GITHUB_TOKEN")
	flag.StringVar(&opts.GHToken, "ghtoken", default_ghtoken, "")
	flag.StringVar(&opts.TempDir, "tempdir", "/tmp/ghrelease", "")
	flag.BoolVar(&opts.Version, "version", false, "")

	flag.Parse()

	return opts
}
