package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli/v2"
)

type CLI struct {
	bc  *Blockchain
	app *cli.App
}

func NewCLI() *CLI {
	bc := NewBlockchain()
	app := &cli.App{
		Name:  "blockchain",
		Usage: "manipulate blockchain",
		Action: func(cCtx *cli.Context) error {
			fmt.Fprintf(cCtx.App.Writer, "no arguments provided\n")
			cli.ShowAppHelpAndExit(cCtx, -1)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name: "add",
				Action: func(cCtx *cli.Context) error {
					data := cCtx.Args().First()
					if len(data) == 0 {
						return cli.Exit("data cannot be empty string", -1)
					}

					err := bc.AddBlock(cCtx.Args().First())
					if err != nil {
						return cli.Exit(err, -1)
					}
					return nil
				},
			},
			{
				Name: "list",
				Action: func(cCtx *cli.Context) error {
					w := cCtx.App.Writer
					iter := bc.Iterator()
					for {
						block := iter.Next()
						fmt.Fprintf(w, "Prev. hash: %x\n", block.PrevBlockHash)
						fmt.Fprintf(w, "Data: %s\n", block.Data)
						fmt.Fprintf(w, "Hash: %x\n", block.Hash)

						pow := NewProofOfWork(block)
						fmt.Fprintf(w, "POW: %s\n", strconv.FormatBool(pow.Validate()))
						fmt.Fprintln(w)

						if len(block.PrevBlockHash) == 0 {
							break
						}
					}
					return nil
				},
			},
		},
	}
	return &CLI{bc: bc, app: app}
}

func (c *CLI) Close() error {
	return c.bc.db.Close()
}

func (c *CLI) Run() error {
	return c.app.Run(os.Args)
}
