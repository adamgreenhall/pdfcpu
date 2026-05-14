/*
Copyright 2025 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/spf13/cobra"
)

var (
	conf             string
	kpw              string
	needStackTrace   bool //= true
	offline          bool
	offlineSet       bool
	opw              string
	perm             string
	quiet            bool
	removeEncryption bool
	removeSignatures bool
	selectedPages    string
	unit             string
	upw              string
	verbose          int
)

var rootCmd = &cobra.Command{
	Use:           "pdfcpu",
	Short:         "A PDF processor written in Go",
	Long:          `pdfcpu is a tool for PDF manipulation written in Go.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&conf, "conf", "c", "", "set or disable config dir: $path | disable")
	rootCmd.PersistentFlags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable output")
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Increase verbosity. Use -v or -vv.")
}

func initConfig() {

	if verbose > 2 {
		verbose = 2
	}

	needStackTrace = verbose > 0

	if quiet {
		return
	}

	log.SetDefaultCLILogger()

	//log.SetDefaultParseLogger()

	if verbose > 0 {
		log.SetDefaultDebugLogger()
		log.SetDefaultInfoLogger()
		log.SetDefaultStatsLogger()
	}

	if verbose == 2 {
		log.SetDefaultTraceLogger()
		log.SetDefaultReadLogger()
		log.SetDefaultValidateLogger()
		log.SetDefaultOptimizeLogger()
		log.SetDefaultWriteLogger()
	}
}

func validateConfigDirFlag() {
	if len(conf) > 0 && conf != "disable" {
		info, err := os.Stat(conf)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "conf: %s does not exist\n\n", conf)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "conf: %s %v\n\n", conf, err)
			os.Exit(1)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "conf: %s not a directory\n\n", conf)
			os.Exit(1)
		}
		model.ConfigPath = conf
		return
	}
	if conf == "disable" {
		model.ConfigPath = "disable"
	}
}

func ensureDefaultConfig() (*model.Configuration, error) {
	validateConfigDirFlag()

	// Check if offline flag was explicitly set
	if cmd := rootCmd; cmd != nil {
		if f := cmd.Flag("offline"); f != nil {
			offlineSet = f.Changed
		}
	}

	if !types.MemberOf(model.ConfigPath, []string{"default", "disable"}) {
		if err := model.EnsureDefaultConfigAt(model.ConfigPath, false); err != nil {
			return nil, err
		}
	}
	return model.NewDefaultConfiguration(), nil
}

func getConfig() *model.Configuration {
	conf, err := ensureDefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pdfcpu: %v\n", err)
		os.Exit(1)
	}

	conf.OwnerPW = opw
	conf.UserPW = upw
	conf.PrivateKeyPW = kpw
	conf.RemoveSignatures = removeSignatures
	conf.RemoveEncryption = removeEncryption

	if offlineSet {
		conf.Offline = offline
	}

	return conf
}
