# KELP
<p align="center">
  <img height="150px" src="./logo.png"  alt="KELP" title="KELP">
</p>

## What is it
A simple replacement for homebrew for installing binary packages on MacOS & Linux written in Go.

## How Does it Work?

1. It downloads a Github releases package matching your operating system and architecture to `~/.kelp/cache`
2. It extracts any binary files to `~/.kelp/bin`
3. It unquarantines the binary (mac)

Example:

```
kelp update --install crumb
🌐 Getting releases for crhuber/crumb:latest...
Latest release v0.0.24. Kelp configured release v0.0.21.
===> Installing crhuber/crumb:v0.0.24...
🌐 Getting releases by tag v0.0.24...
🍏 Finding assets to download...
===> Downloading https://api.github.com/repos/crhuber/crumb/releases/assets/372978000...
Downloading 100%
📂 Extracting /Users/Craig/.kelp/cache/crumb_0.0.24_darwin_arm64.tar.gz
🧐 Checking for binary files in extract...
💾 Copying crumb to kelp bin...
✅ Installed crumb !
🛃 Unquarantining /Users/Craig/.kelp/bin/crumb..
```

## Why?

I built Kelp to scratch my own itch:

* No waiting for a formula to become available on homebrew
* Keep all your computers up to date with a single installation manifest
* No homebrew auto update
* No bloat, no magic

Couldn't this just be a bash script? Probably.

## How To Install

Go to the [releases](https://github.com/crhuber/kelp/releases) page. Download the latest release

Add kelp binary path to your PATH

```
export PATH=~/.kelp/bin/:$PATH
```

## Quick Setup


1. Initialize Kelp

```
kelp init
```

2. Add a new package

```
kelp add junegunn/fzf
```

To use a specific version use the `-r` flag. Where `-r` is the github release version

```
kelp add junegunn/fzf -r 1.0.0
```

3. Install

```
kelp install fzf
```
or

```
kelp add junegunn/fzf --install
```

### Updating a Package

Update

```
kelp update fzf
kelp install fzf
```

Update and install

```
kelp update fzf --install
```


Update to a specific version

```
kelp set fzf -r v0.72.0
kelp install fzf
```

## Configuration

```
GLOBAL OPTIONS:
   --config string, -c string  path to kelp config file (default: "/Users/Craig/.kelp/kelp.json") [$KELP_CONFIG]
   --verbose                   verbose output [$KELP_VERBOSE]
```


## Troubleshooting

### What if the package I want is not on github releases?

Just kelp add the link to the binary, ie:

```
kelp add hashicorp/terraform -r https://releases.hashicorp.com/terraform/0.11.13/terraform_0.11.13_darwin_amd64.zip
```

Also supported are packages like `helm` that provide external non-github download links in the release description.

### Increase logging level

Use `--verbose` or set `KELP_VERBOSE=1` to get more information, expecially during the installation process.

### Why wasnt my package installed ?

Kelp looks for binaries made for MacOS or Linux. If it cannot file a suitable binary for your machine architecture and operating system it will skip downloading it or wont extract it.

Use inspect to open the cache and bin directories for your package

```
kelp inspect
```

To see what binaries exist use:

```
kelp doctor
```

If your binary has a different filename than the name of the Github project, kelp doctor may not find it. To give it a hint you can add the name of the binary to the kelp config

```
kelp set jira-cli -b "jira"`
```

To see whats in your config use:

```
kelp ls
```

### Does it work for Linux?

Yes!

### What if I'm rate limited by Github Api?

Set a github token environment variable

```
export GITHUB_TOKEN="XYZ"
```

## Contributing

If you find bugs, please open an issue first. If you have feature requests, I may not honor it because this project is being built mostly to suit my personal workflow and preferences.
