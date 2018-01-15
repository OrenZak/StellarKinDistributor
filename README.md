# KIN Distributor
This project sets up a webserver that sends KIN assets to addresses over the Stellar TestNet.
It will create a KIN asset, associated with the seed in `config.json` file.
The webserver then exposes a GET endpoint (on port 9000) that takes an address as its parameter and sends KIN assets to it.

```bash
curl 'localhost:9000/send?addr=<Stellar address to send kin too>'
```

Note that the receiving address will have to establish a trustline to the KIN asset issuer before it could receive any KINs.

You can create issuer seed on [Stellar Laboratory](https://www.stellar.org/laboratory/#account-creator?network=test).


## How to Install
* Install [golang](https://golang.org/doc/install#install)
* Clone the repo to `GOPATH/src`
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
Once the account is funded by testnet's friendbot, the webserver waits for incoming requests on port 9000.