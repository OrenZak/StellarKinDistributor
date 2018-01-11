package ingest

import (
	"encoding/json"
	"fmt"
	"time"

	"math"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/db2/sqx"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ClearAll clears the entire history database
func (ingest *Ingestion) ClearAll() error {
	return ingest.Clear(0, math.MaxInt64)
}

// Clear removes a range of data from the history database, exclusive of the end
// id provided.
func (ingest *Ingestion) Clear(start int64, end int64) error {
	clear := ingest.DB.DeleteRange

	err := clear(start, end, "history_effects", "history_operation_id")
	if err != nil {
		return err
	}
	err = clear(start, end, "history_operation_participants", "history_operation_id")
	if err != nil {
		return err
	}
	err = clear(start, end, "history_operations", "id")
	if err != nil {
		return err
	}
	err = clear(start, end, "history_transaction_participants", "history_transaction_id")
	if err != nil {
		return err
	}
	err = clear(start, end, "history_transactions", "id")
	if err != nil {
		return err
	}
	err = clear(start, end, "history_ledgers", "id")
	if err != nil {
		return err
	}
	err = clear(start, end, "history_trades", "history_operation_id")
	if err != nil {
		return err
	}
	err = clear(start, end, "asset_stats", "id")
	if err != nil {
		return err
	}

	return nil
}

// Close finishes the current transaction and finishes this ingestion.
func (ingest *Ingestion) Close() error {
	return ingest.commit()
}

// Effect adds a new row into the `history_effects` table.
func (ingest *Ingestion) Effect(aid int64, opid int64, order int, typ history.EffectType, details interface{}) error {
	djson, err := json.Marshal(details)
	if err != nil {
		return err
	}

	sql := ingest.effects.Values(aid, opid, order, typ, djson)

	_, err = ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// Flush writes the currently buffered rows to the db, and if successful
// starts a new transaction.
func (ingest *Ingestion) Flush() error {
	err := ingest.commit()
	if err != nil {
		return err
	}

	return ingest.Start()
}

// Ledger adds a ledger to the current ingestion
func (ingest *Ingestion) Ledger(
	id int64,
	header *core.LedgerHeader,
	txs int,
	ops int,
) error {

	sql := ingest.ledgers.Values(
		CurrentVersion,
		id,
		header.Sequence,
		header.LedgerHash,
		null.NewString(header.PrevHash, header.Sequence > 1),
		header.Data.TotalCoins,
		header.Data.FeePool,
		header.Data.BaseFee,
		header.Data.BaseReserve,
		header.Data.MaxTxSetSize,
		time.Unix(header.CloseTime, 0).UTC(),
		time.Now().UTC(),
		time.Now().UTC(),
		txs,
		ops,
		header.Data.LedgerVersion,
	)

	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// Operation ingests the provided operation data into a new row in the
// `history_operations` table
func (ingest *Ingestion) Operation(
	id int64,
	txid int64,
	order int32,
	source xdr.AccountId,
	typ xdr.OperationType,
	details map[string]interface{},

) error {
	djson, err := json.Marshal(details)
	if err != nil {
		return err
	}

	sql := ingest.operations.Values(id, txid, order, source.Address(), typ, djson)
	_, err = ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// OperationParticipants ingests the provided accounts `aids` as participants of
// operation with id `op`, creating a new row in the
// `history_operation_participants` table.
func (ingest *Ingestion) OperationParticipants(op int64, aids []xdr.AccountId) error {
	sql := ingest.operation_participants
	q := history.Q{Session: ingest.DB}

	for _, aid := range aids {
		haid, err := q.GetCreateAccountID(aid)
		if err != nil {
			return err
		}
		sql = sql.Values(op, haid)
	}

	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// Rollback aborts this ingestions transaction
func (ingest *Ingestion) Rollback() (err error) {
	err = ingest.DB.Rollback()
	return
}

// Start makes the ingestion reeady, initializing the insert builders and tx
func (ingest *Ingestion) Start() (err error) {
	err = ingest.DB.Begin()
	if err != nil {
		return
	}

	ingest.createInsertBuilders()

	return
}

// transactionInsertBuilder returns sql.InsertBuilder for a single transaction
func (ingest *Ingestion) transactionInsertBuilder(id int64, tx *core.Transaction, fee *core.TransactionFee) sq.InsertBuilder {
	// Enquote empty signatures
	signatures := tx.Base64Signatures()

	return ingest.transactions.Values(
		id,
		tx.TransactionHash,
		tx.LedgerSequence,
		tx.Index,
		tx.SourceAddress(),
		tx.Sequence(),
		tx.Fee(),
		len(tx.Envelope.Tx.Operations),
		tx.EnvelopeXDR(),
		tx.ResultXDR(),
		tx.ResultMetaXDR(),
		fee.ChangesXDR(),
		sqx.StringArray(signatures),
		ingest.formatTimeBounds(tx.Envelope.Tx.TimeBounds),
		tx.MemoType(),
		tx.Memo(),
		time.Now().UTC(),
		time.Now().UTC(),
	)
}

// Trade records a trade into the history_trades table
func (ingest *Ingestion) Trade(
	opid int64,
	order int32,
	buyer xdr.AccountId,
	trade xdr.ClaimOfferAtom,
	ledgerClosedAt int64,
) error {

	q := history.Q{Session: ingest.DB}

	sellerAccountId, err := q.GetCreateAccountID(trade.SellerId)
	if err != nil {
		return errors.Wrap(err, "failed to load seller account id")
	}

	buyerAccountId, err := q.GetCreateAccountID(buyer)
	if err != nil {
		return errors.Wrap(err, "failed to load buyer account id")
	}
	soldAssetId, err := q.GetCreateAssetID(trade.AssetSold)
	if err != nil {
		return errors.Wrap(err, "failed to get sold asset id")
	}

	boughtAssetId, err := q.GetCreateAssetID(trade.AssetBought)
	if err != nil {
		return errors.Wrap(err, "failed to get bought asset id")
	}
	var baseAssetId, counterAssetId int64
	var baseAccountId, counterAccountId int64
	var baseAmount, counterAmount xdr.Int64

	//map seller and buyer to base and counter based on ordering of ids
	if soldAssetId < boughtAssetId {
		baseAccountId, baseAssetId, baseAmount, counterAccountId, counterAssetId, counterAmount =
			sellerAccountId, soldAssetId, trade.AmountSold, buyerAccountId, boughtAssetId, trade.AmountBought
	} else {
		baseAccountId, baseAssetId, baseAmount, counterAccountId, counterAssetId, counterAmount =
			buyerAccountId, boughtAssetId, trade.AmountBought, sellerAccountId, soldAssetId, trade.AmountSold
	}

	sql := ingest.trades.Values(
		opid,
		order,
		time.Unix(ledgerClosedAt, 0).UTC(),
		trade.OfferId,
		baseAccountId,
		baseAssetId,
		baseAmount,
		counterAccountId,
		counterAssetId,
		counterAmount,
		soldAssetId < boughtAssetId,
	)
	_, err = ingest.DB.Exec(sql)
	if err != nil {
		return errors.Wrap(err, "failed to exec sql")
	}

	return nil
}

// Transaction ingests the provided transaction data into a new row in the
// `history_transactions` table
func (ingest *Ingestion) Transaction(
	id int64,
	tx *core.Transaction,
	fee *core.TransactionFee,
) error {

	sql := ingest.transactionInsertBuilder(id, tx, fee)
	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// TransactionParticipants ingests the provided account ids as participants of
// transaction with id `tx`, creating a new row in the
// `history_transaction_participants` table.
func (ingest *Ingestion) TransactionParticipants(tx int64, aids []xdr.AccountId) error {
	sql := ingest.transaction_participants
	q := history.Q{Session: ingest.DB}

	for _, aid := range aids {
		haid, err := q.GetCreateAccountID(aid)
		if err != nil {
			return err
		}
		sql = sql.Values(tx, haid)
	}

	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

func (ingest *Ingestion) createInsertBuilders() {
	ingest.ledgers = sq.Insert("history_ledgers").Columns(
		"importer_version",
		"id",
		"sequence",
		"ledger_hash",
		"previous_ledger_hash",
		"total_coins",
		"fee_pool",
		"base_fee",
		"base_reserve",
		"max_tx_set_size",
		"closed_at",
		"created_at",
		"updated_at",
		"transaction_count",
		"operation_count",
		"protocol_version",
	)

	ingest.accounts = sq.Insert("history_accounts").Columns(
		"address",
	)

	ingest.transactions = sq.Insert("history_transactions").Columns(
		"id",
		"transaction_hash",
		"ledger_sequence",
		"application_order",
		"account",
		"account_sequence",
		"fee_paid",
		"operation_count",
		"tx_envelope",
		"tx_result",
		"tx_meta",
		"tx_fee_meta",
		"signatures",
		"time_bounds",
		"memo_type",
		"memo",
		"created_at",
		"updated_at",
	)

	ingest.transaction_participants = sq.Insert("history_transaction_participants").Columns(
		"history_transaction_id",
		"history_account_id",
	)

	ingest.operations = sq.Insert("history_operations").Columns(
		"id",
		"transaction_id",
		"application_order",
		"source_account",
		"type",
		"details",
	)

	ingest.operation_participants = sq.Insert("history_operation_participants").Columns(
		"history_operation_id",
		"history_account_id",
	)

	ingest.effects = sq.Insert("history_effects").Columns(
		"history_account_id",
		"history_operation_id",
		"\"order\"",
		"type",
		"details",
	)

	ingest.trades = sq.Insert("history_trades").Columns(
		"history_operation_id",
		"\"order\"",
		"ledger_closed_at",
		"offer_id",
		"base_account_id",
		"base_asset_id",
		"base_amount",
		"counter_account_id",
		"counter_asset_id",
		"counter_amount",
		"base_is_seller",
	)

	ingest.assetStats = sq.Insert("asset_stats").Columns(
		"id",
		"amount",
		"num_accounts",
		"flags",
		"toml",
	)
}

func (ingest *Ingestion) commit() error {
	err := ingest.DB.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (ingest *Ingestion) formatTimeBounds(bounds *xdr.TimeBounds) interface{} {
	if bounds == nil {
		return nil
	}

	if bounds.MaxTime == 0 {
		return sq.Expr("?::int8range", fmt.Sprintf("[%d,]", bounds.MinTime))
	}

	return sq.Expr("?::int8range", fmt.Sprintf("[%d,%d]", bounds.MinTime, bounds.MaxTime))
}
