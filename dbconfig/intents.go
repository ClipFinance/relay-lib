package dbconfig

import (
	"context"
	"database/sql"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

// InsertIntent inserts or updates an intent in the database.
//
// Parameters:
// - ctx: the context for managing the request.
// - intent: the intent object containing transaction details.
//
// Returns:
// - error: an error if the database operation fails.
func (dc *DBConfig) InsertIntent(ctx context.Context, intent *types.Intent) error {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return ErrDatabaseConnect
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
       INSERT INTO intent (
           quote_id,            
           from_chain_id,       
           from_token_address,  
           from_amount,         
           to_chain_id,         
           to_token_address,    
           to_amount,           
           user_address,        
           recipient_address,   
           from_tx,            
           from_nonce,          
           status,              
           sub_status,          
           quote_requested_at,  
           from_tx_mined_at,    
           to_tx_set_at,        
           to_tx_mined_at,      
           refund,
           refund_tx,
           refund_tx_set_at,
           refund_tx_mined_at,
           block_hash,          
           quorum               
       ) VALUES (
           $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
           $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, 1
       )
       ON CONFLICT (quote_id, block_hash) 
       DO UPDATE SET quorum = intent.quorum + 1`,
		intent.QuoteID,
		intent.FromChain,
		intent.FromToken,
		intent.FromAmount.String(),
		intent.ToChain,
		intent.ToToken,
		intent.ToAmount.String(),
		intent.UserAddress,
		intent.RecipientAddress,
		intent.FromTx,
		intent.FromNonce,
		intent.Status,
		intent.SubStatus,
		intent.RequestedAt,
		intent.FromTxMinedAt,
		intent.ToTxSetAt,
		intent.ToTxMinedAt,
		intent.Refund,
		intent.RefundTx,
		intent.RefundTxSetAt,
		intent.RefundTxMinedAt,
		intent.BlockHash,
	)

	return err
}

// SetCreatedIntentStatus updates the status of an intent to created and sets the to_tx field to null.
func (dc *DBConfig) SetCreatedIntentStatus(ctx context.Context, quoteID string) error {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	query := `
		UPDATE intent 
			SET status = $1, 
			    to_tx = NULL, 
			    to_tx_set_at = NULL, 
			    to_nonce = NULL, 
			    retries = retries + 1
		WHERE quote_id = $2
    `

	result, err := db.ExecContext(ctx, query, types.StatusCreated, quoteID)
	if err != nil {
		return errors.Wrap(err, "failed to update intent status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.Errorf("intent with quoteID %s not found", quoteID)
	}

	return nil
}

func (dc *DBConfig) GetCreatedIntents(ctx context.Context) ([]*types.Intent, error) {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	// Сначала получаем и блокируем записи
	query := `
        WITH selected_intents AS (
		SELECT 
			id, quote_id, from_chain_id, from_token_address, from_amount, 
                to_chain_id, to_token_address, to_amount, user_address, recipient_address,
                from_tx, to_tx, status, sub_status, quote_requested_at,
                from_tx_mined_at, to_tx_set_at, to_tx_mined_at, refund,
                refund_tx, refund_tx_set_at, refund_tx_mined_at, block_hash, quorum
            FROM intent 
            WHERE status = $1 AND quorum >= 1
		AND from_tx_mined_at > $2
            FOR UPDATE SKIP LOCKED
            LIMIT 100
        )
        UPDATE intent i
        SET status = $3
        FROM selected_intents s
        WHERE i.id = s.id
        RETURNING s.*`

	// Calculate expiration time.
	expirationTime := time.Now().Add(-ExpirationTime)

	rows, err := tx.QueryContext(ctx, query, types.StatusCreated, expirationTime, types.StatusPending)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query created intents")
	}
	defer rows.Close()

	var intents []*types.Intent
	for rows.Next() {
		var i types.Intent
		var fromAmount, toAmount string

		err := rows.Scan(
			&i.ID, &i.QuoteID, &i.FromChain, &i.FromToken, &fromAmount,
			&i.ToChain, &i.ToToken, &toAmount, &i.UserAddress, &i.RecipientAddress,
			&i.FromTx, &i.ToTx, &i.Status, &i.SubStatus, &i.RequestedAt,
			&i.FromTxMinedAt, &i.ToTxSetAt, &i.ToTxMinedAt, &i.Refund,
			&i.RefundTx, &i.RefundTxSetAt, &i.RefundTxMinedAt, &i.BlockHash, &i.Quorum,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan intent")
		}

		// Convert string amounts to big.Int
		i.FromAmount = new(big.Int)
		i.FromAmount.SetString(fromAmount, 10)
		i.ToAmount = new(big.Int)
		i.ToAmount.SetString(toAmount, 10)

		intents = append(intents, &i)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return intents, nil
}

func (dc *DBConfig) GetPendingIntents(ctx context.Context) ([]*types.Intent, error) {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	// Start transaction
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()

	query := `
		SELECT 
			id, quote_id, from_chain_id, from_token_address, from_amount, 
			to_chain_id, to_token_address, to_amount, user_address, recipient_address,
			from_tx, to_tx, status, sub_status, quote_requested_at,
			from_tx_mined_at, to_tx_set_at, to_tx_mined_at, refund,
			refund_tx, refund_tx_set_at, refund_tx_mined_at, block_hash, quorum
		FROM intent 
		WHERE status = $1
		AND from_tx_mined_at > $2
        FOR UPDATE SKIP LOCKED
        LIMIT 100
	`

	expirationTime := time.Now().Add(-ExpirationTime)

	rows, err := tx.QueryContext(ctx, query, types.StatusPending, expirationTime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query pending intents")
	}
	defer rows.Close()

	var intents []*types.Intent
	for rows.Next() {
		var i types.Intent
		var fromAmount, toAmount string

		err := rows.Scan(
			&i.ID, &i.QuoteID, &i.FromChain, &i.FromToken, &fromAmount,
			&i.ToChain, &i.ToToken, &toAmount, &i.UserAddress, &i.RecipientAddress,
			&i.FromTx, &i.ToTx, &i.Status, &i.SubStatus, &i.RequestedAt,
			&i.FromTxMinedAt, &i.ToTxSetAt, &i.ToTxMinedAt, &i.Refund,
			&i.RefundTx, &i.RefundTxSetAt, &i.RefundTxMinedAt, &i.BlockHash, &i.Quorum,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan intent")
		}

		i.FromAmount = new(big.Int)
		i.FromAmount.SetString(fromAmount, 10)
		i.ToAmount = new(big.Int)
		i.ToAmount.SetString(toAmount, 10)

		intents = append(intents, &i)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	// Commit transaction to release locks.
	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return intents, nil
}

// SetDoneIntentStatus updates the status of an intent to done and sets the sub_status field for the intent.
func (dc *DBConfig) SetDoneIntentStatus(ctx context.Context, quoteID string, nonce uint64) error {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	query := `
		UPDATE intent 
		SET 
		    status = $1, 
		    sub_status = $2, 
		    to_nonce = $3, 
		    to_tx_mined_at = NOW()
		WHERE quote_id = $4
	`

	result, err := db.ExecContext(ctx, query, types.StatusDone, types.Completed, nonce, quoteID)
	if err != nil {
		return errors.Wrap(err, "failed to update intent status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("no intent found with quote_id: " + quoteID)
	}

	return nil
}

// SetFailedIntentStatus updates the status of an intent to failed and sets the sub_status field for the intent.
func (dc *DBConfig) SetFailedIntentStatus(ctx context.Context, quoteID string, subStatus types.SubStatus) error {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	query := `
		UPDATE intent 
		SET status = $1, sub_status = $2
		WHERE quote_id = $3
	`

	result, err := db.ExecContext(ctx, query, types.StatusFailed, subStatus, quoteID)
	if err != nil {
		return errors.Wrap(err, "failed to update intent status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("no intent found with quote_id: " + quoteID)
	}

	return nil

}

// SetPendingIntentStatus updates the status of an intent to pending and sets the to_tx field for the intent.
func (dc *DBConfig) SetPendingIntentStatus(ctx context.Context, quoteID, toTx string, nonce uint64) error {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	query := `
		UPDATE intent 
		SET to_tx = $1, to_tx_set_at = NOW(), status = $2, to_nonce = $3
		WHERE quote_id = $4
	`

	result, err := db.ExecContext(ctx, query, toTx, types.StatusPending, nonce, quoteID)
	if err != nil {
		return errors.Wrap(err, "failed to update intent status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.New("no intent found with quote_id: " + quoteID)
	}

	return nil
}

func (dc *DBConfig) GetIntentByQuoteID(ctx context.Context, quoteID string) (*types.Intent, error) {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	query := `
		SELECT 
			id, quote_id, from_chain_id, from_token_address, from_amount, 
			to_chain_id, to_token_address, to_amount, user_address, recipient_address,
			from_tx, to_tx, status, sub_status, quote_requested_at,
			from_tx_mined_at, to_tx_set_at, to_tx_mined_at, refund,
			refund_tx, refund_tx_set_at, refund_tx_mined_at, block_hash, quorum
		FROM intent 
		WHERE quote_id = $1
	`

	var i types.Intent
	var fromAmount, toAmount string

	if err = db.QueryRowContext(ctx, query, quoteID).Scan(
		&i.ID, &i.QuoteID, &i.FromChain, &i.FromToken, &fromAmount,
		&i.ToChain, &i.ToToken, &toAmount, &i.UserAddress, &i.RecipientAddress,
		&i.FromTx, &i.ToTx, &i.Status, &i.SubStatus, &i.RequestedAt,
		&i.FromTxMinedAt, &i.ToTxSetAt, &i.ToTxMinedAt, &i.Refund,
		&i.RefundTx, &i.RefundTxSetAt, &i.RefundTxMinedAt, &i.BlockHash, &i.Quorum,
	); err != nil {
		return nil, errors.Wrap(err, "failed to scan intent")
	}

	i.FromAmount = new(big.Int)
	i.FromAmount.SetString(fromAmount, 10)
	i.ToAmount = new(big.Int)
	i.ToAmount.SetString(toAmount, 10)

	return &i, nil

}

// GetPendingTransactionsByChain returns a map of chain IDs to sorted transactions.
func (dc *DBConfig) GetPendingTransactionsByChain(ctx context.Context) (map[uint64][]*types.Transaction, error) {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	// Get transactions grouped by chain and sorted by nonce
	query := `
        SELECT 
            i.to_chain_id as chain_id,
            i.to_tx as hash,
            i.quote_id,
            i.to_nonce as nonce,
            i.user_address as "from",
            i.recipient_address as "to",
            i.to_token_address as token,
            i.to_amount,
            i.from_amount
        FROM intent i
        WHERE i.status = $1 
        AND i.to_tx_set_at > $2
        ORDER BY i.to_chain_id, i.to_nonce
    `

	expirationTime := time.Now().Add(-ExpirationTime)
	rows, err := db.QueryContext(ctx, query, types.StatusPending, expirationTime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query pending transactions")
	}
	defer rows.Close()

	pendingTxsByChain := make(map[uint64][]*types.Transaction)

	for rows.Next() {
		tx := &types.Transaction{}
		var chainID uint64

		if err := rows.Scan(
			&chainID,
			&tx.Hash,
			&tx.QuoteID,
			&tx.Nonce,
			&tx.Token,
			&tx.From,
			&tx.To,
			&tx.ToAmount,
			&tx.FromAmount,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan transaction")
		}

		tx.ChainID = chainID
		pendingTxsByChain[chainID] = append(pendingTxsByChain[chainID], tx)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return pendingTxsByChain, nil
}
