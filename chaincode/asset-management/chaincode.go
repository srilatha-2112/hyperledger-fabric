package main

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type Account struct {
	DEALERID    string `json:"DEALERID"`
	MSISDN      string `json:"MSISDN"`
	MPIN        string `json:"MPIN"`
	BALANCE     int64  `json:"BALANCE"`
	STATUS      string `json:"STATUS"`
	TRANSAMOUNT int64  `json:"TRANSAMOUNT"`
	TRANSTYPE   string `json:"TRANSTYPE"`
	REMARKS     string `json:"REMARKS"`
}

type SmartContract struct {
	contractapi.Contract
}

func (s *SmartContract) exists(ctx contractapi.TransactionContextInterface, key string) (bool, error) {
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return false, err
	}
	return b != nil, nil
}

func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, dealerID, msisdn, mpin, balance, status, transAmount, transType, remarks string) error {
	ok, err := s.exists(ctx, msisdn)
	if err != nil {
		return err
	}
	if ok {
		return errors.New("asset exists")
	}
	bal, err := strconv.ParseInt(balance, 10, 64)
	if err != nil {
		return err
	}
	tamt, err := strconv.ParseInt(transAmount, 10, 64)
	if err != nil {
		return err
	}
	acc := Account{DEALERID: dealerID, MSISDN: msisdn, MPIN: mpin, BALANCE: bal, STATUS: status, TRANSAMOUNT: tamt, TRANSTYPE: transType, REMARKS: remarks}
	raw, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(msisdn, raw)
}

func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, msisdn string) (*Account, error) {
	b, err := ctx.GetStub().GetState(msisdn)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errors.New("not found")
	}
	var acc Account
	if err := json.Unmarshal(b, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, dealerID, msisdn, mpin, balance, status, transAmount, transType, remarks string) error {
	ok, err := s.exists(ctx, msisdn)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not found")
	}
	bal, err := strconv.ParseInt(balance, 10, 64)
	if err != nil {
		return err
	}
	tamt, err := strconv.ParseInt(transAmount, 10, 64)
	if err != nil {
		return err
	}
	acc := Account{DEALERID: dealerID, MSISDN: msisdn, MPIN: mpin, BALANCE: bal, STATUS: status, TRANSAMOUNT: tamt, TRANSTYPE: transType, REMARKS: remarks}
	raw, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(msisdn, raw)
}

func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, msisdn string) error {
	ok, err := s.exists(ctx, msisdn)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not found")
	}
	return ctx.GetStub().DelState(msisdn)
}

func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*Account, error) {
	it, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer it.Close()
	out := []*Account{}
	for it.HasNext() {
		kv, err := it.Next()
		if err != nil {
			return nil, err
		}
		var a Account
		if err := json.Unmarshal(kv.Value, &a); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, nil
}

type History struct {
	TxID      string   `json:"txId"`
	Value     *Account `json:"value,omitempty"`
	IsDelete  bool     `json:"isDelete"`
	Timestamp int64    `json:"timestamp"`
}

func (s *SmartContract) GetAssetHistory(ctx contractapi.TransactionContextInterface, msisdn string) ([]*History, error) {
	it, err := ctx.GetStub().GetHistoryForKey(msisdn)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	h := []*History{}
	for it.HasNext() {
		rec, err := it.Next()
		if err != nil {
			return nil, err
		}
		var val *Account
		if rec.Value != nil && !rec.IsDelete {
			var a Account
			if err := json.Unmarshal(rec.Value, &a); err != nil {
				return nil, err
			}
			val = &a
		}
		h = append(h, &History{TxID: rec.TxId, Value: val, IsDelete: rec.IsDelete, Timestamp: rec.Timestamp.GetSeconds()})
	}
	return h, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		panic(err)
	}
	if err := chaincode.Start(); err != nil {
		panic(err)
	}
}
