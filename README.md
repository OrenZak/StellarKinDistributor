## KIN Distributor
This project lets you run a small localhost server that communicates with stellar network.
It will create a KIN asset on Stellar Test Network, related to the issuer seed in `config.json` file.
You will have a rest-api `sendkin` where you need to pass address as url parameter<br/>
and the issuer will send 1K KIN to the specific address.<br/>
You can create issuer seed on [Stellar Laboratory](https://www.stellar.org/laboratory/#account-creator?network=test).


### How to install
* Clone the repo
* Run the below commands from project root dir, to install glide and have all the dependencies   

<br/>
<b>params:</b><br/>
glide_version := v0.13.1<br/>
glide_arch := linux-amd64  (change that to your architecture depends on the OS)

```bash
        curl -sSLo glide.tar.gz https://github.com/Masterminds/glide/releases/download/$(glide_version)/glide-$(glide_version)-$(glide_arch).tar.gz
        tar -xf ./glide.tar.gz
        mv ./$(glide_arch)/glide ./glide
        rm -rf ./$(glide_arch) ./glide.tar.gz
        glide install
```


### How to run
Run the command below
```bash
    go build && ./StellarKinDistributor
```