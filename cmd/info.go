/*
Copyright © 2023 litao.little@gmail.com

*/

package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	pixiv "github.com/littleneko/pixiv-api-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var shortMsg = false

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get the illust/user info",
}

var illustInfoCmd = &cobra.Command{
	Use:   "illust [illust id list]",
	Short: "Get illust info",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cobra.CheckErr("Must give at least one illust id")
		}
		client := buildPixivClient()
		illustIds := processListArgs(args)
		showIllustInfo(client, illustIds)
	},
}

var userInfoCmd = &cobra.Command{
	Use:   "user [user id]",
	Short: "Get user info",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cobra.CheckErr("Must give a user id")
		}
		client := buildPixivClient()
		uid := strings.TrimSpace(args[0])
		showUserInfo(client, uid)
	},
}

func init() {
	infoCmd.PersistentFlags().BoolVarP(&shortMsg, "short-msg", "s", false, "Show the short msg")

	infoCmd.AddCommand(illustInfoCmd)
	infoCmd.AddCommand(userInfoCmd)
}

func buildPixivClient() *pixiv.PixivClient {
	proxy := viper.GetString("proxy")
	cookie := viper.GetString("cookie")
	ua := viper.GetString("user-agent")

	var client *pixiv.PixivClient
	if len(proxy) > 0 {
		proxyUrl, err := url.Parse(proxy)
		cobra.CheckErr(err)
		client = pixiv.NewPixivClientWithProxy(proxyUrl, 5000)
	} else {
		client = pixiv.NewPixivClient(5000)
	}
	if len(cookie) > 0 {
		cookieKV := strings.Split(cookie, "=")
		if len(cookieKV) == 2 {
			client.AddCookie(cookieKV[0], cookieKV[1])
		} else {
			client.SetCookiePHPSESSID(cookie)
		}
	}
	if len(ua) > 0 {
		client.SetUserAgent(ua)
	}
	return client
}

func showIllustInfo(pixivClient *pixiv.PixivClient, illusts []string) {
	for _, pid := range illusts {
		illusts, err := pixivClient.GetIllustInfo(pixiv.PixivID(pid), false)
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

func showUserInfo(PixivClient *pixiv.PixivClient, uid string) {
	illustIds, err := PixivClient.GetUserIllusts(uid)
	cobra.CheckErr(err)
	j, _ := json.Marshal(illustIds)
	fmt.Println("illusts: ", len(illustIds), string(j))
}
