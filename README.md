# KIN Distributor
This project lets you send KIN to a given wallet address over Stellar TestNet.
It will create a KIN asset, related to the issuer seed in `config.json` file.
Use the `send()` function which accepts a URL address as a parameter. This will issue 1000 KIN to the specified address.
You can create issuer seed on [Stellar Laboratory](https://www.stellar.org/laboratory/#account-creator?network=test).


## How to Install
* Install[golang](https://golang.org/doc/install#install)
* Clone the repo
* Install [glide](https://github.com/Masterminds/glide) v0.13.1 for dependency management.
* Execute the following bash code for downloading glide and dependencies.  
```bash
glide install
```


## How to Run
Run the command below
```bash
go build && ./StellarKinDistributor
```