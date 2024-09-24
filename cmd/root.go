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

const ModelFlash = "gemini-1.5-flash-latest"
const ModelPro = "gemini-1.5-pro-latest"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "graderace-data-summarizer URL",
	Short: "指定したJRAのページの重賞データを要約します",
	Long: `指定したJRAのページの重賞データを要約します。
その後クリップボードへデータを要約を保存します。`,
	Run: func(cmd *cobra.Command, args []string) {
		tm, _ := cmd.Flags().GetString("model")

		model := ModelFlash
		if strings.Contains(tm, "pro") {
			model = ModelPro
		}
		summarizeAndClipped(model)
	},
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
}

func init() {
	rootCmd.Flags().StringP("model", "m", ModelFlash, fmt.Sprintf("Modelを指定します。デフォルトだと%sが利用されます。\n%sを使いたいときにはproと入力してください。\nproは制約によりエラーになる場合があります。", ModelFlash, ModelPro))
}

// summarizeAndClipped はデータを取得して、要約しクリップボードへ書き出す
func summarizeAndClipped(model string) {
	if err := godotenv.Load(".env"); err != nil {
		log.Panic(err)
	}

	url := os.Args[1]
	text, err := getAnalyzedRaceData(url)
	if err != nil {
		log.Panic(err)
	}

	ctx := context.Background()
	summarize, err := runGemini(ctx, text, model)
	if err != nil {
		log.Panic(err)
	}

	// 標準出力に出力して、clipboardへ書き込みも行う
	fmt.Print(summarize)
	clipboard.WriteAll(summarize)
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
// 利用するmodelはユーザーの指定による
func runGemini(ctx context.Context, text, model string) (string, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return "", err
	}
	defer client.Close()

	genModel := client.GenerativeModel(model)
	resp, err := genModel.GenerateContent(ctx, genai.Text(fmt.Sprintf("%s %s", "次の文章を要約してください。", text)))
	if err != nil {
		return "", err
	}

	fmt.Print("\n----- USED MODEL -----\n")
	fmt.Print(model)
	fmt.Print("\n----- USED MODEL -----\n")

	return textnaizeCandinates(resp.Candidates), nil
}
