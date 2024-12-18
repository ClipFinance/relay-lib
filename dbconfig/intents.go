package dbconfig

import (
	"context"
	"database/sql"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/pkg/errors"
)

// InsertIntent inserts or updates an intent in the database.
//
// Parameters:
// - ctx: the context for managing the request.
// - intent: the intent object containing transaction details.
//
// Returns:
// - error: an error if the database operation fails.
func (r *DBConfig) InsertIntent(ctx context.Context, intent *types.Intent) error {
	db, err := sql.Open("postgres", r.dbConnStr)
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
func (r *DBConfig) SetCreatedIntentStatus(ctx context.Context, quoteID string) error {
	db, err := sql.Open("postgres", r.dbConnStr)
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
