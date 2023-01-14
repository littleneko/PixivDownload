/*
Copyright © 2023 litao.little@gmail.com

*/

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"pixiv/app"
)

var shortMsg bool = false
var illustIds []string
var userId string

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get the illust/user info",
}

var illustInfoCmd = &cobra.Command{
	Use:   "illust",
	Short: "Get illust info",
	Run: func(cmd *cobra.Command, args []string) {
		pixivClient := app.NewPixivClient(viper.GetString("cookie"), viper.GetString("user-agent"), 5000)
		showIllustInfo(pixivClient, illustIds)
	},
}

var userInfoCmd = &cobra.Command{
	Use:   "user",
	Short: "Get user info",
	Run: func(cmd *cobra.Command, args []string) {
		pixivClient := app.NewPixivClient(viper.GetString("cookie"), viper.GetString("user-agent"), 5000)
		showUserInfo(pixivClient, userId)
	},
}

func init() {
	infoCmd.PersistentFlags().BoolVarP(&shortMsg, "short-msg", "s", false, "Show the short msg")

	illustInfoCmd.Flags().StringSliceVar(&illustIds, "ids", []string{}, "Get the illust info of this pid")
	err := illustInfoCmd.MarkFlagRequired("ids")
	cobra.CheckErr(err)

	userInfoCmd.Flags().StringVar(&userId, "uid", "", "Get the user info of this uid")
	err = userInfoCmd.MarkFlagRequired("uid")
	cobra.CheckErr(err)

	infoCmd.AddCommand(illustInfoCmd)
	infoCmd.AddCommand(userInfoCmd)
}

func showIllustInfo(pixivClient *app.PixivClient, illusts []string) {
	for _, pid := range illustIds {
		illusts, err := pixivClient.GetIllustInfo(app.PixivID(pid), false)
		if err != nil {
			fmt.Printf("ID: %s,\tERROE: %s\n", pid, err)
			continue
		}
		for _, illust := range illusts {
			if shortMsg {
				fmt.Println(illust.DigestStringWithUrl())
			} else {
				j, err := json.MarshalIndent(illust, "", "  ")
				if err != nil {
					fmt.Printf("ID: %s, ERROE: %s\n", pid, err)
					continue
				}
				fmt.Println(string(j))
			}
		}
	}
}

func showUserInfo(PixivClient *app.PixivClient, uid string) {
	pids, err := PixivClient.GetUserIllusts(uid)
	cobra.CheckErr(err)
	fmt.Println("illusts: ", pids)
}
