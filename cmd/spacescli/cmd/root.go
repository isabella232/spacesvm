// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "spaces-cli" implements spacesvm client operation interface.
package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	requestTimeout = 30 * time.Second
	fsModeWrite    = 0o600
)

var (
	privateKeyFile string
	uri            string
	workDir        string

	rootCmd = &cobra.Command{
		Use:        "spaces-cli",
		Short:      "SpacesVM client CLI",
		SuggestFor: []string{"spaces-cli", "spacescli", "spacesctl"},
	}
)

func init() {
	p, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	workDir = p

	cobra.EnablePrefixMatching = true
	rootCmd.AddCommand(
		createCmd,
		genesisCmd,
		claimCmd,
		lifelineCmd,
		setCmd,
		deleteCmd,
		resolveCmd,
		infoCmd,
	)

	rootCmd.PersistentFlags().StringVar(
		&privateKeyFile,
		"private-key-file",
		".spaces-cli-pk",
		"private key file path",
	)
	rootCmd.PersistentFlags().StringVar(
		&uri,
		"endpoint",
		"http://127.0.0.1:9650",
		"RPC Endpoint for VM",
	)
}

func Execute() error {
	return rootCmd.Execute()
}
