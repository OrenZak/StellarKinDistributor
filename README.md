# KIN Distributor
This project lets you run a small local server that communicates with Stellar TestNet.
It will create a KIN asset, related to the issuer seed in `config.json` file.
Use the `send()` function which accepts a URL address as a parameter. This will issue 1000 KIN to the specified address.
You can create issuer seed on [Stellar Laboratory](https://www.stellar.org/laboratory/#account-creator?network=test).


## How to Install
* [Install `golang`](https://golang.org/doc/install#install)
* Clone the repo
* This project uses [glide](https://github.com/Masterminds/glide) v0.13.1 for dependency management. Execute the following bash code for downloading glide and dependencies.  

```bash
export GLIDE_VERSION = "v0.13.1"
# set architecture depending on your machine
# set architectures in https://github.com/Masterminds/glide/releases
export GLIDE_ARCH = "linux-amd64"

curl -sSLo glide.tar.gz https://github.com/Masterminds/glide/releases/download/$(GLIDE_VERSION)/glide-$(GLIDE_VERSION)-$(GLIDE_ARCH).tar.gz
tar -xf ./glide.tar.gz
mv ./$(GLIDE_ARCH)/glide ./glide
rm -rf ./$(GLIDE_ARCH) ./glide.tar.gz
glide install
```


## How to Run
Run the command below
```bash
go build && ./StellarKinDistributor
```