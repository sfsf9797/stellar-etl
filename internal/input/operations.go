package input

import (
	"context"
	"fmt"
	"io"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stellar/stellar-etl/internal/utils"
)

// OperationTransformInput is a representation of the input for the TransformOperation function
type OperationTransformInput struct {
	Operation       xdr.Operation
	OperationIndex  int32
	Transaction     ingest.LedgerTransaction
	LedgerSeqNum    int32
	LedgerCloseMeta xdr.LedgerCloseMeta
}

func panicIf(err error) {
	if err != nil {
		panic(fmt.Errorf("An error occurred, panicking: %s\n", err))
	}
}

// GetOperations returns a slice of operations for the ledgers in the provided range (inclusive on both ends)
func GetOperations(start, end uint32, limit int64, env utils.EnvironmentDetails) ([]OperationTransformInput, error) {
	ctx := context.Background()

	backend, err := env.CreateCaptiveCoreBackend()
	if err != nil {
		return []OperationTransformInput{}, err
	}

	opSlice := []OperationTransformInput{}
	err = backend.PrepareRange(ctx, ledgerbackend.BoundedRange(start, end))
	panicIf(err)
	for seq := start; seq <= end; seq++ {
		// txReader, err := ingest.NewLedgerTransactionReader(ctx, backend, publicPassword, seq)
		changeReader, err := ingest.NewLedgerChangeReader(ctx, backend, env.NetworkPassphrase, seq)
		if err != nil {
			return []OperationTransformInput{}, err
		}
		txReader := changeReader.LedgerTransactionReader

		ledgerCloseMeta, err := backend.GetLedger(ctx, seq)
		if err != nil {
			return nil, errors.Wrap(err, "error getting ledger from the backend")
		}

		for int64(len(opSlice)) < limit || limit < 0 {
			tx, err := txReader.Read()
			if err == io.EOF {
				break
			}

			for index, op := range tx.Envelope.Operations() {
				opSlice = append(opSlice, OperationTransformInput{
					Operation:       op,
					OperationIndex:  int32(index),
					Transaction:     tx,
					LedgerSeqNum:    int32(seq),
					LedgerCloseMeta: ledgerCloseMeta,
				})

				if int64(len(opSlice)) >= limit && limit >= 0 {
					break
				}
			}
		}

		txReader.Close()
		if int64(len(opSlice)) >= limit && limit >= 0 {
			break
		}
	}

	return opSlice, nil
}
