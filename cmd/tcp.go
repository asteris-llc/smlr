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

// tcpCmd represents the tcp command
var tcpCmd = &cobra.Command{
	Use:   "tcp [url]:[port]",
	Short: "wait for a TCP health check",
	Long: `wait for a TCP health check. You can specify the content to wait for
	and to write with the --content and --write flags, respectively. You can
	specify read/write timeouts with the --iotimeout flag.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected one argument, the URL to wait for")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		tcp := smlr.TCPWaiter{
			URL:           args[0],
			Content:       viper.GetString("content"),
			Write:         viper.GetString("write"),
			IOTimeout:     viper.GetDuration("iotimeout"),
			EntireContent: viper.GetBool("complete"),
		}

		ctx, cancel := context.WithCancel(context.Background())
		for status := range tcp.Wait(ctx, viper.GetDuration("interval"), viper.GetDuration("timeout")) {
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
	RootCmd.AddCommand(tcpCmd)

	tcpCmd.Flags().StringP("content", "c", "", "content to check for")
	tcpCmd.Flags().StringP("write", "w", "", "write this to the connection before listening")
	tcpCmd.Flags().DurationP("iotimeout", "", 5*time.Second, "timeout of read/write operations")
	tcpCmd.Flags().Bool("complete", false, "if content + EOF is the complete expected response.")

	viper.BindPFlags(tcpCmd.Flags())
}
