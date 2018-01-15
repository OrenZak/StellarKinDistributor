package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/stellar/go/build"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/clients/horizon"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"io"
	"encoding/json"
)

const (
	horizonURL = "https://horizon-testnet.stellar.org"
)

type Message struct {
	msg string `json:"msg,omitempty"`
}

type Config struct {
	IssuerSeed string `json:"issuer_seed"`
}

var kin_asset horizon.Asset
var issuerKP keypair.KP
var logger log.Logger

func main() {
	logger = initLogger()
	level.Debug(logger).Log("msg", "started local server")

	issuerSeed := getIssuerSeedFromConfig()

	//Create the issuer from const seed
	issuerKP = keypair.MustParse(issuerSeed)

	level.Debug(logger).Log("msg", fmt.Sprintf("Issuer Address: %s" ,issuerKP.Address()))

	//Send extra XLM to issuer / create the account if wasn't created.
	fundAccount(issuerKP.Address(), logger)

	//Create KIN asset related to our issuer.
	kin_asset = horizon.Asset{"credit_alphanum4", "KIN", issuerSeed}

	router := mux.NewRouter()
	router.HandleFunc("/send", sendKin).Methods("GET")
	level.Error(logger).Log("ListenAndServe:9000", http.ListenAndServe(":9000", router))
}

func getIssuerSeedFromConfig() string {
	raw, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var config Config
	json.Unmarshal(raw, &config)

	return config.IssuerSeed
}

func sendKin(writer http.ResponseWriter, request *http.Request) {
	toAddress := request.URL.Query().Get("addr")

	writer.Header().Set("Content-Type", "application/json")

	if toAddress == "" {
		writer.WriteHeader(http.StatusBadRequest)
		io.WriteString(writer, `{"msg": no address found}`)
	}

	toKP := keypair.MustParse(toAddress)

	err := transferKin(issuerKP, toKP, "1000", logger)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		io.WriteString(writer, `{"msg": kin transfer failed}`)
	} else {
		writer.WriteHeader(http.StatusOK)
		io.WriteString(writer, `{"msg": kin transfer succeed}`)
	}
}

func fundAccount(address string, logger log.Logger) {
	l := log.With(logger, "address", address[:5])

	level.Info(l).Log("msg", "sending funding request")

	fundCall := fmt.Sprintf("https://horizon-testnet.stellar.org/friendbot?addr=%s", address)
	level.Info(l).Log("fundCall:", fundCall)

	client := http.Client{Timeout: 20 * time.Second}

	res, err := client.Get(fundCall)

	if err != nil {
		level.Error(l).Log("msg", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			level.Error(l).Log("msg", err)
		}
	}()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		level.Error(l).Log("msg", err)
	}

	level.Debug(l).Log("msg", string(data))

	if res.StatusCode == http.StatusBadRequest {
		level.Error(l).Log("msg", "user already funded with 10k XLM")
	}

	level.Info(l).Log("msg", "funding success")
}

func transferKin(from, to keypair.KP, amount string, logger log.Logger) error {

	client := horizon.Client{
		URL:  horizonURL,
		HTTP: &http.Client{Timeout: 10 * time.Second},
	}

	tx := build.Transaction(
		build.SourceAccount{AddressOrSeed: from.Address()},
		build.TestNetwork,
		build.AutoSequence{SequenceProvider: horizon.DefaultTestNetClient},
		build.Payment(
			build.Destination{AddressOrSeed: to.Address()},
			build.CreditAmount{Code: kin_asset.Code, Issuer: kin_asset.Issuer, Amount: amount},
		),
	)
	txEnv := tx.Sign(from.(*keypair.Full).Seed())
	txEnvB64, err := txEnv.Base64()
	if err != nil {
		level.Error(logger).Log("msg", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "submitting transaction", "from", from.Address()[:5], "to", to.Address()[:5], "amount", amount)

	_, err = client.SubmitTransaction(txEnvB64)
	if err != nil {
		getTxErrorResponseCode(err, logger)
	}

	return err
}

func getTxErrorResponseCode(err error, logger log.Logger) *horizon.TransactionResultCodes {
	level.Error(logger).Log("msg", err)
	switch e := err.(type) {
	case *horizon.Error:
		code, err := e.ResultCodes()
		if err != nil {
			level.Error(logger).Log("msg", "failed to extract result codes from horizon response")
			return nil
		}
		level.Error(logger).Log("code", code.TransactionCode)
		for i, opCode := range code.OperationCodes {
			level.Error(logger).Log("opcode_index", i, "opcode", opCode)
		}

		return code
	}
	return nil
}

func initLogger() log.Logger {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = level.NewFilter(logger, level.AllowDebug())
	logger = log.With(logger, "time", log.DefaultTimestampUTC())
	return logger
}
