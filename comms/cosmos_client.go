package comms

// cosmos_client.go — replaces github.com/ignite/cli/ignite/pkg/cosmosclient
// Uses cosmos-sdk v0.53 primitives directly. Connects via gRPC (default :9090).

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bridgetypes "twilight-project/nyks/x/bridge/types"
	forktypes "twilight-project/nyks/x/forks/types"
)

// Client wraps cosmos-sdk v0.53 client infrastructure.
type Client struct {
	clientCtx client.Context
	txFactory clienttx.Factory
}

// Response holds the result of a broadcast.
type Response struct {
	TxHash string
	Raw    interface{}
}

type clientConfig struct {
	home           string
	keyringBackend string
}

// Option configures the Client.
type Option func(*clientConfig)

func WithHome(home string) Option {
	return func(c *clientConfig) { c.home = home }
}

func WithKeyringBackend(backend string) Option {
	return func(c *clientConfig) { c.keyringBackend = backend }
}

// New creates a cosmos client connected to the nyks chain via gRPC.
// Reads nyksd_grpc (default "localhost:9090") and chain_id (default "nyks") from viper.
func New(_ context.Context, opts ...Option) (Client, error) {
	cfg := &clientConfig{
		keyringBackend: "test",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.home == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Client{}, fmt.Errorf("get home dir: %w", err)
		}
		cfg.home = filepath.Join(home, ".nyks")
	}

	// bech32 prefix
	sdkCfg := sdk.GetConfig()
	sdkCfg.SetBech32PrefixForAccount("twilight", "twilightpub")

	// interface registry + codec
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	bridgetypes.RegisterInterfaces(registry)
	forktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	// tx config
	txCfg := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	// keyring
	kr, err := keyring.New("nyks", cfg.keyringBackend, cfg.home, nil, cdc)
	if err != nil {
		return Client{}, fmt.Errorf("open keyring: %w", err)
	}

	// gRPC connection
	grpcAddr := viper.GetString("nyksd_grpc")
	if grpcAddr == "" {
		grpcAddr = "localhost:9090"
	}
	grpcConn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return Client{}, fmt.Errorf("grpc dial %s: %w", grpcAddr, err)
	}

	chainID := viper.GetString("chain_id")
	if chainID == "" {
		chainID = "nyks"
	}

	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithInterfaceRegistry(registry).
		WithTxConfig(txCfg).
		WithKeyring(kr).
		WithGRPCClient(grpcConn).
		WithBroadcastMode("sync").
		WithAccountRetriever(authtypes.AccountRetriever{})

	txFactory := clienttx.Factory{}.
		WithKeybase(kr).
		WithTxConfig(txCfg).
		WithGas(300000).
		WithGasAdjustment(1.5).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithChainID(chainID)

	return Client{clientCtx: clientCtx, txFactory: txFactory}, nil
}

// Address returns the bech32 address for the given keyring account name.
func (c Client) Address(accountName string) (sdk.AccAddress, error) {
	rec, err := c.clientCtx.Keyring.Key(accountName)
	if err != nil {
		return nil, fmt.Errorf("keyring key %q: %w", accountName, err)
	}
	addr, err := rec.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("get address for %q: %w", accountName, err)
	}
	return addr, nil
}

// BroadcastTx builds, signs, and broadcasts a single message transaction.
func (c Client) BroadcastTx(accountName string, msg sdk.Msg) (Response, error) {
	addr, err := c.Address(accountName)
	if err != nil {
		return Response{}, err
	}

	ctx := c.clientCtx.
		WithFromName(accountName).
		WithFromAddress(addr)

	// Prepare fetches account number + sequence from chain via gRPC.
	txf, err := c.txFactory.Prepare(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("prepare tx factory: %w", err)
	}

	txBuilder, err := txf.BuildUnsignedTx(msg)
	if err != nil {
		return Response{}, fmt.Errorf("build unsigned tx: %w", err)
	}

	if err := clienttx.Sign(context.Background(), txf, accountName, txBuilder, true); err != nil {
		return Response{}, fmt.Errorf("sign tx: %w", err)
	}

	txBytes, err := ctx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return Response{}, fmt.Errorf("encode tx: %w", err)
	}

	res, err := ctx.BroadcastTx(txBytes)
	if err != nil {
		return Response{}, fmt.Errorf("broadcast tx: %w", err)
	}

	if res.Code != 0 {
		return Response{TxHash: res.TxHash}, fmt.Errorf("tx failed (code %d): %s", res.Code, res.RawLog)
	}

	log.Printf("tx broadcast ok, hash: %s", res.TxHash)
	return Response{TxHash: res.TxHash, Raw: res}, nil
}

// Context returns the underlying client.Context (used by GetCurrentSequence).
func (c Client) Context() client.Context {
	return c.clientCtx
}
