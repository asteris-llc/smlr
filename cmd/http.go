// Copyright © 2016 Asteris
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/asteris-llc/smlr/smlr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

// httpCmd represents the http command
var httpCmd = &cobra.Command{
	Use:   "http [url]",
	Short: "wait for an HTTP health check",
	Long: `wait for an HTTP health check. You can specify the method and status to
wait for with the --method and --status flags, respectively.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected one argument, the URL to wait for")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		http := smlr.HTTPWaiter{
			Method:         viper.GetString("method"),
			URL:            args[0],
			ExpectedStatus: viper.GetInt("status"),
			Content:        viper.GetString("content"),
			EntireContent:  viper.GetBool("complete"),
		}

		ctx, cancel := context.WithCancel(context.Background())
		for status := range http.Wait(ctx, viper.GetDuration("interval"), viper.GetDuration("timeout")) {
			logger := logrus.WithFields(logrus.Fields{
				"message": status.Message,
				"done":    status.Done,
			})

			if status.Error != nil {
				cancel()
				logger.WithError(status.Error).Error("exiting")
				time.AfterFunc(1*time.Second, func() { os.Exit(1) })
			} else {
				logger.Info("update")
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(httpCmd)

	httpCmd.Flags().StringP("method", "m", "GET", "method to use")
	httpCmd.Flags().Int32P("status", "s", 200, "status to check for")
	httpCmd.Flags().StringP("content", "c", "", "content to check for")
	httpCmd.Flags().Bool("complete", true, "if content is the complete expected response")

	viper.BindPFlags(httpCmd.Flags())
}
