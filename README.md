# ghrelease

ghrelease is a simple CLI tool that downloads the latest release assets from Github for MacOS and Linux architectures, specifically "amd64" and "arm64". 

The tool automatically identifies your OS and architecture, and downloads the release. If the release is compressed or in an archive format, it will automatically extract and unpack it, no matter how it's compressed, and keep ONLY the binary.

You can also choose to skip the extraction and keep the archive.

## Installation

Download the latest binary from the [releases](https://github.com/kavishgr/ghrelease/releases) section and place it in your `$PATH`. 

### Dependencies

A GitHub personal access token is required.

<details>
<summary>Set the token</summary>
<br>

**Method 1: Store in shell config file (Recommended)**

Add to ~/.bashrc, ~/.zshrc, or ~/.profile:

```bash
echo "export GITHUB_TOKEN='ghp_xxxxxxxxxxxxx'" >> ~/.zshrc
source ~/.zshrc
```

**Method 2: Read from file**

Save token to a file:

```bash
echo 'ghp_xxxxxxxxxxxxx' > ~/.github-token
chmod 600 ~/.github-token
```

Add to shell config:

```bash
echo "export GITHUB_TOKEN=\$(cat ~/.github-token)" >> ~/.zshrc
source ~/.zshrc
```

**Method 3: Set for current session only**

Use a space before the command to avoid shell history (works in bash/zsh):

```bash
 export GITHUB_TOKEN='ghp_xxxxxxxxxxxxx'
```

**Verify it's set:**
```bash
echo $GITHUB_TOKEN
```
</details>

**Note:** Avoid typing `export GITHUB_TOKEN=...` directly in your terminal as it will be saved to shell history.

## Usage

```sh
ghrelease -h
```

All the supported flags:

```sh
-list    list all the releases found
            Will print the latest release for your OS and Architecture.

-con     <int> set the concurrency level (default: 2)

-download will download and extract the binary inside `/tmp/ghrelease`
            Example: cat urls_from_list_results.txt | ghrelease -download 

-skipextraction skip the extraction process
	 Example: echo "helix-editor/helix" | ghrelease -list | getghrel -download -skipextraction

-tempdir <string> specify a temporary directory to download/extract the binaries
            Default is `/tmp/ghrelease`
            Example: cat urls_from_list_results.txt | ghrelease -download -tempdir /tmp/test

-version display version
```

### List Found Releases

To list the found releases, create a text file with a **complete URL** or **owner/repo** per line, and run:

```sh
# List of URLs
# e.g "sharkdp/bat" or https://github.com/sharkdp/bat

cat urls.txt | ghrelease -list -con 3 | tee releases.txt

# Single one
echo "sharkdp/bat" | ghrelease -list | sort
```

#### Demo

![-list](examples/list-demo.gif)


This will display a list of URLs representing the latest release assets found for each repository.

In rare cases, you may come across additional files like checksums and SBOMs that are specific to your operating system and architecture. I have taken care to exclude them in the regular expression. However, if any such files exist, you can simply filter them out before using the `-download` flag to ensure a clean download. But don't worry, even if you don't filter the output, the tool will automatically keep only the binaries and remove any unnecessary files. Filtering them out can help save bandwidth.

In the case of `N/A`(not available), it means that the repository doesn't have any release assets available. For Linux releases, there might be separate versions for both GNU and Musl. You can choose to filter them out based on your preferences.

> **Note**: In the example above, the first line shows that the 'neovim' package is unavailable. But neovim does have a latest release. The reason it's not listed is because my regex always checks for assets containing both the OS and architecture, while neovim's assets only specify the OS. There are ways to resolve this issue, but it involves dealing with regex, which can be a bit complex. Nevertheless, you can be confident that every release will be discoverable, except for this particular case.

>> **update**: On macOS, Neovim is now being listed because the release includes the OS and arch:

```
➜  ~ echo "neovim/neovim" | ghrelease -list
https://github.com/neovim/neovim/releases/download/v0.10.0/nvim-macos-arm64.tar.gz
```
Duplicates are unlikely, but if they do occur, you can easily filter them out using tools like `sort` and `uniq`. That should do the trick.

In case a repository lacks a latest release tag, the tool will search for the most recent release tag instead. In rare cases this can be an unstable/nightly release.

### GitHub Token

A GitHub personal access token is required to use this tool. The token is validated on startup.

**Required Scopes**: `public_repo` (for public repos) or `repo` (for private repos)

**Common Errors**:
- `401 Unauthorized`: Invalid or expired token - generate a new one
- `403 Forbidden`: Valid token but lacks required permissions - check token scopes
- Network errors: Check your internet connection or GitHub API status

Generate a token at: https://github.com/settings/tokens

### Download Found Assets

To download the found assets and keep the binaries in a temporary folder (which is `/tmp/ghrelease` by default), simply use the `-download` flag:

```sh
# List of URLS found with -list
cat releases.txt | ghrelease -download
cat releases.txt | ghrelease -download -con 3

# Single one
echo "https://github.com/sharkdp/bat" | ghrelease -list | getghrel -download
```

Before using `-download`, remove any lines starting with 'N/A' from the list of found assets, like shown below.

#### Demo

![-download](examples/download-demo.gif)


In the example above, you can observe that the `ClementTsang/bottom` package had two releases due to different versions of GNU. However, the tool only retained one version. You can filter out these additional releases. I included them here for the purpose of this example.

To download to a different location, use the `-tempdir` flag :

```sh
# List of URLS
cat releases.txt | ghrelease -download -tempdir '/tmp/tempbin'

# Single one
echo "https://github.com/sharkdp/bat" | ghrelease -list | getghrel -download -tempdir '/tmp/tempbin'
```

### Skip Extraction

To keep the archive or compressed release, simply use the `-skipextraction` option:

```sh
echo "helix-editor/helix" | ghrelease -list | getghrel -skipextraction -download
```

It is useful for releases that require dependencies bundled together in separate files or folders, rather than just a single binary.

#### Demo

![-skipextraction](examples/skipextraction-demo.gif)

## TODO

- **Optional**: update the regex or add some sort of backup/rescue regex to include releases that contain only the operating system and not the architecture. Most releases do include both the OS and architecture, I'm mentioning it here because of the neovim(solved now) issue discussed earlier.

## FAQ

**Why did I create this tool instead of using a package manager?** 

