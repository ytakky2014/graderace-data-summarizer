/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/atotto/clipboard"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"google.golang.org/api/option"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "graderace-data-summarizer",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute は
// .envからGeminiAIに必要なAPIKEYを取得する
// 引数に指定したURLのコンテンツのデータを取得する
// 取得したコンテンツデータをGeminiAIで要約させる
// clipboard へ書き込む
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}

	if err := godotenv.Load(".env"); err != nil {
		log.Panic(err)
	}

	url := os.Args[1]
	text, err := getAnalyzedRaceData(url)
	if err != nil {
		log.Panic(err)
	}

	ctx := context.Background()
	summarize, err := runGemini(ctx, text)
	if err != nil {
		log.Panic(err)
	}

	// 標準出力に出力して、clipboardへ書き込みも行う
	fmt.Print(summarize)
	clipboard.WriteAll(summarize)

}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.graderace-data-summarizer.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// getAnalyzedRaceData は指定したデータ分析ページからテキストを抽出する
func getAnalyzedRaceData(url string) (string, error) {
	var text string
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// 対象ページはShiftJIS
	utfBody := transform.NewReader(bufio.NewReader(res.Body), japanese.ShiftJIS.NewDecoder())
	if err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(utfBody)
	if err != nil {
		return "", err
	}

	doc.Find("div#main_contents").Each(func(i int, s *goquery.Selection) {
		text = strings.Join(strings.Fields(s.Text()), " ")
	})
	return text, nil
}

// textnaizeCandinates はGeminiAIの結果を改行した文字列で返す
func textnaizeCandinates(cs []*genai.Candidate) string {
	var ss []string
	for _, c := range cs {
		for _, p := range c.Content.Parts {
			ss = append(ss, fmt.Sprint(p))
		}
	}

	return strings.Join(ss, "\n")
}

// runGemini はGeminiAIを実行する
// 解析結果のテキストを返却する
func runGemini(ctx context.Context, text string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return "", err
	}
	defer client.Close()

	//model := client.GenerativeModel("gemini-1.5-pro-latest")
	model := client.GenerativeModel("gemini-1.5-flash-latest")

	resp, err := model.GenerateContent(ctx, genai.Text(fmt.Sprintf("%s %s", "次の文章を要約してください。", text)))
	if err != nil {
		return "", err
	}

	return textnaizeCandinates(resp.Candidates), nil
}
