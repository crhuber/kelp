<p align="center">
  <img height="150px" src="./logo.png"  alt="KELP" title="KELP">
</p>

# KELP
A simple replacement for homebrew for installing binary packages on MacOS written in Go.

## Why?

I built Kelp to scratch my own itch:

* No waiting for a formula to become available on homebrew
* Keep all your computers up to date with a single installation manifest
* Install multiple packages at one time.

## How To Install

```
curl -L https://github.com/crhuber/kelp/releases/download/v1.5.0/kelp_1.5.0_darwin_amd64.tar.gz -o ./kelp.tar.gz
tar -zxvf kelp.tar.gz
chmod +x ./kelp
```

## How Does it Work?

It downloads all github releases packages defined in the config file `~/.kelp/kelp.json` to `~/.kelp/bin`.

## How Do I Use It?


1. Initialize Kelp

    `./kelp init`

2. Add kelp binary path to your PATH

    `export PATH=~/.kelp/bin/:$PATH >> ~/.bash_profile`

3. Boostrap install kelp

    This allows kelp to be managed by itself

    ```
    ./kelp install kelp
    rm ./kelp           # removes the bootstrap version
    ```

4. Add a new package

    `kelp add ogham/exa -r latest`

4. Install

    `kelp install exa`


### How do I install multiple packages at one time?

1. Edit  `~/.kelp/kelp.json` and add all your favorite packages there. For example mine looks [this](https://github.com/crhuber/dotfiles/blob/master/kelp/kelp.json)

2. Run kelp

    `kelp install --all`

### What if the package I want is not on github releases?

Easy. Just add the http(s) link to the binary like so to your .kelp.json file

`
kelp add hashicorp/terraform https://releases.hashicorp.com/terraform/0.11.13/terraform_0.11.13_darwin_amd64.zip
`


## Troubleshooting

Use inspect the cache and bin directories for your package

`kelp inspect`

### Why wasnt my package installed ?

Kelp looks for binaries made for MacOS, if it finds a binary for linux or windows it will skip downloading it.

To see what binaries exist use:

`kelp doctor`

To see whats in your config use:

`kelp ls`
### Does it work for Linux?

Not yet

