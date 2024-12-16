package dbconfig

import (
	"context"
	"database/sql"
	"github.com/ClipFinance/relay-lib/common/types"
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
