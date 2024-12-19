package utils

import (
	"context"
	"encoding/json"

	bin "github.com/gagliardetto/binary"
	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type GetParsedTransactionResultV2 struct {
	Slot        uint64
	BlockTime   *sol.UnixTimeSeconds
	Transaction *rpc.ParsedTransaction
	Meta        *rpc.ParsedTransactionMeta
}

func GetParsedTransactionV2(
	ctx context.Context,
	client *rpc.Client,
	txSig sol.Signature,
	opts *GetParsedTransactionOptsV2,
) (out *GetParsedTransactionResultV2, err error) {
	params := []interface{}{txSig}
	obj := rpc.M{}
	if opts != nil {
		if opts.Commitment != "" {
			obj["commitment"] = opts.Commitment
		}
		obj["maxSupportedTransactionVersion"] = opts.MaxSupportedTransactionVersion
	}
	obj["encoding"] = sol.EncodingJSONParsed
	params = append(params, obj)
	err = client.RPCCallForInto(ctx, &out, "getTransaction", params)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, rpc.ErrNotFound
	}
	return
}

func (obj GetParsedTransactionResultV2) MarshalWithEncoder(encoder *bin.Encoder) (err error) {
	err = encoder.WriteUint64(obj.Slot, bin.LE)
	if err != nil {
		return err
	}
	if obj.BlockTime == nil {
		err = encoder.WriteBool(false)
		if err != nil {
			return err
		}
	} else {
		err = encoder.WriteBool(true)
		if err != nil {
			return err
		}
		err = encoder.WriteInt64(int64(*obj.BlockTime), bin.LE)
		if err != nil {
			return err
		}
	}
	if obj.Transaction == nil {
		err = encoder.WriteBool(false)
		if err != nil {
			return err
		}
	} else {
		err = encoder.WriteBool(true)
		if err != nil {
			return err
		}
		err = encoder.Encode(obj.Transaction)
		if err != nil {
			return err
		}
	}

	return nil
}

func (obj GetParsedTransactionResultV2) UnmarshalWithDecoder(decoder *bin.Decoder) (err error) {
	// Deserialize Slot:
	obj.Slot, err = decoder.ReadUint64(bin.LE)
	if err != nil {
		return err
	}
	// Deserialize BlockTime (optional):
	ok, err := decoder.ReadBool()
	if err != nil {
		return err
	}
	if ok {
		err = decoder.Decode(&obj.BlockTime)
		if err != nil {
			return err
		}
	}
	// Deserialize Transaction (optional):
	ok, err = decoder.ReadBool()
	if err != nil {
		return err
	}
	if ok {
		// NOTE: storing as JSON bytes:
		buf, err := decoder.ReadByteSlice()
		if err != nil {
			return err
		}
		err = json.Unmarshal(buf, &obj.Transaction)
		if err != nil {
			return err
		}
	}

	return nil
}
