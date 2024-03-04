package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/text"

	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/tiny-blob/tinyblob/pkg/routes"
	"github.com/tiny-blob/tinyblob/pkg/services"
)

func main() {
	// Initiate a new container
	c := services.NewContainer()
	defer func() {
		if err := c.Shutdown(); err != nil {
			c.Web.Logger.Fatal(err)
		}
	}()

	// Build the router
	routes.BuildRouter(c)

	// Start web server
	go func() {
		srv := http.Server{
			Addr:    fmt.Sprintf("%s:%d", c.Config.HTTP.Hostname, c.Config.HTTP.Port),
			Handler: c.Web,
		}

		// TODO: Handle TLS certificates
		if err := c.Web.StartServer(&srv); err != http.ErrServerClosed {
			c.Web.Logger.Fatalf("shutting down the server: %v", err)
		}
	}()

	// Create a new WS client (used for confirming transactions)
	wsClient, err := ws.Connect(context.Background(), rpc.DevNet_WS)
	if err != nil {
		panic(err)
	}

	// TODO Make around here……
	accountFrom, err := solana.PrivateKeyFromSolanaKeygenFile("/Users/james/code/tinyblob/programs/target/deploy/tiny_blob-keypair.json")
	if err != nil {
		spew.Dump(err)
	}

	// triggerKeyPair := solana.NewWallet()
	triggerKeyPair := solana.MustPublicKeyFromBase58("H2kY5LXxxBjiBR91dH9FisrLA597Njb4X6sbf8UhcdeN")
	triggerKeyPairPrivateKey := solana.MustPrivateKeyFromBase58("2ZKnvMBKb2qiZTGnGYZBXkVdqho5zqrCf76SsNiwGcckJFrSbUKx7TEsmbJN9JPp4ujRjcA8CqSHN8MUT4sbr4Dt")
	// triggerKeyPair := solana.MustPublicKeyFromBase58("4sVHU1NqyqNGRnE93tBRZovU2d4s1XKn9ogFjgobiFrR")
	// spew.Dump(triggerKeyPair.PublicKey(), "<----------------------------")

	// if true {
	// 	// Airdrop 1 sol to the account so it will have something to transfer:
	// 	out, err := c.SolRPC.RequestAirdrop(
	// 		context.TODO(),
	// 		accountFrom.PublicKey(),
	// 		solana.LAMPORTS_PER_SOL*1,
	// 		rpc.CommitmentFinalized,
	// 	)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Println("airdrop transaction signature:", out)
	// 	time.Sleep(time.Second * 5)
	// }

	recentBlockHash, err := c.SolRPC.GetRecentBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		panic(err)
	}

	programID := accountFrom.PublicKey()

	accounts := []*solana.AccountMeta{
		{PublicKey: triggerKeyPair, IsSigner: true, IsWritable: true},
		// {PublicKey: accountTo, IsSigner: false, IsWritable: true},
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			solana.NewInstruction(programID, accounts, []byte{}),
		},
		recentBlockHash.Value.Blockhash,
	)
	if err != nil {
		panic(err)
	}

	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if triggerKeyPair.Equals(key) {
				return &triggerKeyPairPrivateKey
			}
			return nil
		},
	)
	if err != nil {
		panic(fmt.Errorf("unable to sign transaction: %w", err))
	}
	// spew.Dump(tx)
	// Pretty print the transaction:
	tx.EncodeTree(text.NewTreeEncoder(os.Stdout, "Transfer SOL"))

	// Send transaction, and wait for confirmation:
	sig, err := confirm.SendAndConfirmTransaction(
		context.TODO(),
		c.SolRPC,
		wsClient,
		tx,
	)
	if err != nil {
		spew.Dump("+++++++++++++++++")
		panic(err)
	}
	spew.Dump(sig)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Web.Shutdown(ctx); err != nil {
		c.Web.Logger.Fatal(err)
	}
}
