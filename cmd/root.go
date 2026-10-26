package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stellar/kelp/support/utils"
)

// build flags
var version string
var gitBranch string
var gitHash string
var buildDate string
var env string
var amplitudeAPIKey string
var goarm string

const envRelease = "release"
const envDev = "dev"
const rootShort = "Kelp is a free and open-source trading bot for the Stellar universal marketplace."
const rootLong = `Kelp is a free and open-source trading bot for the Stellar universal marketplace (https://stellar.org).

Learn more about Stellar : https://www.stellar.org
Learn more about Kelp    : https://github.com/stellar/kelp`
const kelpExamples = tradeExamples + "\n  kelp trade --help"

// RootCmd is the main command for this repo
var RootCmd = &cobra.Command{
	Use:     "kelp",
	Short:   rootShort,
	Long:    rootLong,
	Example: kelpExamples,
	Run: func(ccmd *cobra.Command, args []string) {
		intro := `
  __        _______ _     ____ ___  __  __ _____    _____ ___      _  _______ _     ____  
  \ \      / / ____| |   / ___/ _ \|  \/  | ____|  |_   _/ _ \    | |/ / ____| |   |  _ \ 
   \ \ /\ / /|  _| | |  | |  | | | | |\/| |  _|      | || | | |   | ' /|  _| | |   | |_) |
    \ V  V / | |___| |__| |__| |_| | |  | | |___     | || |_| |   | . \| |___| |___|  __/ 
     \_/\_/  |_____|_____\____\___/|_|  |_|_____|    |_| \___/    |_|\_\_____|_____|_|    
																			cli=` + version + `
`
		fmt.Println(intro)

		e := ccmd.Help()
		if e != nil {
			panic(e)
		}
	},
}

func init() {
	validateBuild()

	RootCmd.AddCommand(tradeCmd)
	RootCmd.AddCommand(strategiesCmd)
	RootCmd.AddCommand(terminateCmd)
	RootCmd.AddCommand(versionCmd)
}

func validateBuild() {
	if version == "" || buildDate == "" || gitBranch == "" || gitHash == "" {
		fmt.Println("version information not included, please build using the build script (scripts/build.sh)")
		os.Exit(1)
	}

	if amplitudeAPIKey == "" && env == envRelease {
		utils.PrintErrorHintf("amplitude API key not included, please export AMPLITUDE_API_KEY before running build script (scripts/build.sh)")
		os.Exit(1)
	}
}
