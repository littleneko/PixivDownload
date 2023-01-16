/*
Copyright © 2023 litao.little@gmail.com

*/

package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"pixiv/app"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var shortMsg = false
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
		client := buildPixivClient()
		showIllustInfo(client, illustIds)
	},
}

var userInfoCmd = &cobra.Command{
	Use:   "user",
	Short: "Get user info",
	Run: func(cmd *cobra.Command, args []string) {
		client := buildPixivClient()
		showUserInfo(client, userId)
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

func buildPixivClient() *app.PixivClient {
	proxy := viper.GetString("proxy")
	cookie := viper.GetString("cookie")
	ua := viper.GetString("user-agent")
	var client *app.PixivClient
	if len(proxy) > 0 {
		proxyUrl, err := url.Parse(proxy)
		cobra.CheckErr(err)
		client = app.NewPixivClientWithProxy(proxyUrl, 5000)
	} else {
		client = app.NewPixivClient(5000)
	}
	if len(cookie) > 0 {
		client.SetCookie(cookie)
	}
	if len(ua) > 0 {
		client.SetUserAgent(ua)
	}
	return client
}

func showIllustInfo(pixivClient *app.PixivClient, illusts []string) {
	for _, pid := range illusts {
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
	j, _ := json.Marshal(pids)
	fmt.Println("illusts: ", len(pids), string(j))
}
