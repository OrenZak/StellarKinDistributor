package ingest

import (
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

// Add writes an effect to the database while automatically tracking the index
// to use.
func (ei *EffectIngestion) Add(aid xdr.AccountId, typ history.EffectType, details interface{}) bool {
	q := history.Q{Session: ei.parent.DB}

	if ei.err != nil {
		return false
	}

	ei.added++
	var haid int64

	haid, ei.err = q.GetCreateAccountID(aid)
	if ei.err != nil {
		return false
	}

	ei.err = ei.Dest.Effect(haid, ei.OperationID, ei.added, typ, details)
	if ei.err != nil {
		return false
	}

	return true
}

// Finish marks this ingestion as complete, returning any error that was recorded.
func (ei *EffectIngestion) Finish() error {
	err := ei.err
	ei.err = nil
	return err
}
