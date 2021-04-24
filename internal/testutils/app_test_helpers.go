package testutils

import (
	"encoding/json"
	"math/big"
	"os"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	"github.com/smartbch/moeingevm/ebp"
	motypes "github.com/smartbch/moeingevm/types"
	"github.com/smartbch/smartbch/app"
	"github.com/smartbch/smartbch/internal/bigutils"
	"github.com/smartbch/smartbch/param"
)

const (
	adsDir  = "./testdbdata"
	modbDir = "./modbdata"
)

// var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
var nopLogger = log.NewNopLogger()

type TestApp struct {
	*app.App
}

func CreateTestApp(keys ...string) *TestApp {
	return CreateTestApp0(bigutils.NewU256(10000000), keys...)
}

func CreateTestApp0(testInitAmt *uint256.Int, keys ...string) *TestApp {
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
	params := param.DefaultConfig()
	params.AppDataPath = adsDir
	params.ModbDataPath = modbDir
	testValidatorPubKey := ed25519.GenPrivKey().PubKey()
	_app := app.NewApp(params, bigutils.NewU256(1), nopLogger,
		testValidatorPubKey)
	_app.Init(nil)
	//_app.txEngine = ebp.NewEbpTxExec(10, 100, 1, 100, _app.signer)
	genesisData := app.GenesisData{
		Alloc: KeysToGenesisAlloc(testInitAmt, keys),
	}
	appStateBytes, _ := json.Marshal(genesisData)

	_app.InitChain(abci.RequestInitChain{AppStateBytes: appStateBytes})
	_app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		ProposerAddress: testValidatorPubKey.Address(),
	}})
	_app.Commit()
	return &TestApp{_app}
}

func (_app *TestApp) Destroy() {
	_app.Stop()
	_ = os.RemoveAll(adsDir)
	_ = os.RemoveAll(modbDir)
}

func (_app *TestApp) GetBalance(addr common.Address) *big.Int {
	ctx := _app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	b, err := ctx.GetBalance(addr, -1)
	if err != nil {
		panic(err)
	}
	return b.ToBig()
}

func (_app *TestApp) GetCode(addr common.Address) []byte {
	ctx := _app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	codeInfo := ctx.GetCode(addr)
	if codeInfo == nil {
		return nil
	}
	return codeInfo.BytecodeSlice()
}

func (_app *TestApp) GetStorageAt(addr common.Address, key []byte) []byte {
	ctx := _app.GetContext(app.RpcMode)
	defer ctx.Close(false)

	acc := ctx.GetAccount(addr)
	if acc == nil {
		return nil
	}
	return ctx.GetStorageAt(acc.Sequence(), string(key))
}

func (_app *TestApp) GetBlock(h uint64) *motypes.Block {
	ctx := _app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	if ctx.GetLatestHeight() != int64(h) {
		time.Sleep(500 * time.Millisecond)
	}
	b, err := ctx.GetBlockByHeight(h)
	if err != nil {
		panic(err)
	}
	return b
}

func (_app *TestApp) GetTx(h common.Hash) *motypes.Transaction {
	ctx := _app.GetContext(app.RpcMode)
	defer ctx.Close(false)
	tx, err := ctx.GetTxByHash(h)
	if err != nil {
		panic(err)
	}
	return tx
}

func (_app *TestApp) GetTxsByAddr(addr common.Address) []*motypes.Transaction {
	ctx := _app.GetContext(app.HistoryOnlyMode)
	defer ctx.Close(false)
	txs, err := ctx.QueryTxByAddr(addr, 1, uint32(_app.BlockNum())+1)
	if err != nil {
		panic(err)
	}
	return txs
}

func (_app *TestApp) Call(sender common.Address, tx *gethtypes.Transaction) (int, string, []byte) {
	runner, _ := _app.RunTxForRpc(tx, sender, false)
	return runner.Status, ebp.StatusToStr(runner.Status), runner.OutData
}
func (_app *TestApp) EstimateGas(sender common.Address, tx *gethtypes.Transaction) (int, string, int64) {
	runner, estimatedGas := _app.RunTxForRpc(tx, sender, true)
	return runner.Status, ebp.StatusToStr(runner.Status), estimatedGas
}
