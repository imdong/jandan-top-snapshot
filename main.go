package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var aesCipher = "your-32-byte-long-key-here!00000"

type Row struct {
	ID      string // 唯一ID
	Code    string // 防伪码
	Name    string // 作者
	Time    string // 发布时间
	Type    string // 类型
	Content string // 内容
	OO      string // OO数
	XX      string // XX数
	Tucao   string // 吐槽数
}

// 获取当前时间
var now = time.Now()
var env = os.Getenv("ENV")

// 更新 list.json 文件
var listFileName = "./docs/index.md"

func main() {
	// 获取环境提供的密码
	if os.Getenv("AES_CIPHER") != "" {
		aesCipher = os.Getenv("AES_CIPHER")
	}

	// 扫描加密文件解密
	scanAesDecrypt()

	// 获取当前时间用于文件命名
	dateStr := now.Format("200601")
	hourStr := now.Format("0215")
	fileName := fmt.Sprintf("%s/%s.md", dateStr, hourStr)

	// 如果是本地环境且本地有缓存则直接读取
	var body []byte
	_, err := os.Stat("./docs/body.html")
	if env != "local" || err != nil {
		body, err = readHtmlBody()
		if err != nil {
			return
		}
	} else {
		body, err = os.ReadFile("./docs/body.html")
	}

	rows := matchRows(string(body))

	mdContent := makeMdDoc(rows)

	// 对 mdContent 进行 aes 加密后再写到文件
	mdContent = aesEncrypt(mdContent)

	// 保证储存目录存在
	err = os.MkdirAll("./docs/"+dateStr, 0755)

	// 保存到文件
	saveToFile(fileName, mdContent)

	// 检查列表是否需要跨年
	crossYear()

	// 更新列表
	appendList(fileName)

	fmt.Println("Comments successfully saved and index.md updated.")
}

// aesEncrypt encrypts the content using AES encryption.
func aesEncrypt(content string) string {
	key := []byte(aesCipher) // 32 bytes key for AES-256
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println("Error creating cipher:", err)
		return ""
	}

	// Pad the content to be a multiple of the block size
	padding := block.BlockSize() - len(content)%block.BlockSize()
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	content += string(padText)

	ciphertext := make([]byte, aes.BlockSize+len(content))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		fmt.Println("Error generating IV:", err)
		return ""
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], []byte(content))

	return base64.StdEncoding.EncodeToString(ciphertext)
}

// aesDecrypt decrypts the content using AES encryption.
func aesDecrypt(content string) string {
	key := []byte(aesCipher) // 32 bytes key for AES-256
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println("Error creating cipher:", err)
		return ""
	}

	ciphertext, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		fmt.Println("Error decoding content:", err)
		return ""
	}

	if len(ciphertext) < aes.BlockSize {
		fmt.Println("Ciphertext is too short")
		return ""
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Unpad the content
	padding := int(ciphertext[len(ciphertext)-1])
	content = string(ciphertext[:len(ciphertext)-padding])

	return content
}

// convertToMarkdown converts HTML content to Markdown format and removes unwanted HTML tags.
func convertToMarkdown(html string) string {
	// Convert the class="view_img_link" link to Markdown format
	html = regexp.MustCompile(`<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`).ReplaceAllString(html, "[$2]($1)")

	// Convert img tags to Markdown format
	html = regexp.MustCompile(`<img[^>]*src="([^"]+)"[^>]*>`).ReplaceAllString(html, "![](https:$1)")

	// Replace HTML line breaks with newlines
	html = strings.ReplaceAll(html, "<br />", "  \n\n")

	// Remove any remaining HTML tags
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")

	return html
}

// scanAesDecrypt scans the directory for .aes files and decrypts them.
func scanAesDecrypt() {
	// 扫描目录下 aes 文件解密
	_ = fs.WalkDir(os.DirFS("./docs"), ".", func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, ".aes") {
			return nil
		}

		// 读取文件内容
		content, err := os.ReadFile("./docs/" + path)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return err
		}

		// 解密文件内容
		decryptedContent := aesDecrypt(string(content))

		// 写入解密后的文件
		err = os.WriteFile("./docs/"+path[:len(path)-4], []byte(decryptedContent), 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return err
		}

		// 删除加密文件
		err = os.Remove("./docs/" + path)
		if err != nil {
			fmt.Println("Error deleting file:", err)
			return err
		}
		return nil
	})
}

// readHtmlBody 读取 HTML 内容
func readHtmlBody() (body []byte, err error) {
	// 创建一个新的请求
	url := "https://jandan.net/top"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// 自定义 UA, 如果持续一周 403 则终止该仓库
	req.Header.Set("User-Agent", "Mozilla/5.0 JandanTopSnapshot/1.0 repo(https://github.com/imdong/JandanTopSnapshot)")

	// 发送请求并获取响应
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// 将 body 缓存到本地, 并用于调试
	if env == "local" {
		err = os.WriteFile("./docs/body.html", body, 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}

	return body, nil
}

// matchRows 从 HTML 中匹配出评论数据
func matchRows(html string) []Row {

	// 使用正则匹配主要部分
	mainBodyRegex := regexp.MustCompile(`(?s)<div id="comments">.*<!-- end comments -->`)
	mainBody := mainBodyRegex.FindString(html)

	// 提取单条记录的详细数据
	rowRegex := regexp.MustCompile(`(?ms)<li id="comment-([^"]+)">[^/]+/a><strong\s+title="[^:：]+[：:]([^"]+)"[^>]*>([^<]+)<[^>]+>\s+<br>\s+<small>([^<]+)<[^>]+>\s+<[^<]+>[^@]+(@[^<]+)</b></small>\s+<br>\s+<p>(.*?)</p>\s+</div>[^[]+[^>]+>([^<]+)<[^[]+\[[^>]+>([^<]+)<[^[]+\[([^]]+)]\s*</a>[^"]+"\S+\s+</li>`)

	// 匹配所有评论并提取数据
	var rows []Row
	for _, match := range rowRegex.FindAllStringSubmatch(mainBody, -1) {
		// 将提取到的匹配项存入结构体
		comment := Row{
			ID:      match[1],
			Code:    match[2],
			Name:    match[3],
			Time:    match[4],
			Type:    match[5],
			Content: convertToMarkdown(match[6]),
			OO:      match[7],
			XX:      match[8],
			Tucao:   match[9],
		}
		rows = append(rows, comment)

		//fmt.Printf("%d, %s\n", i, match[0])
	}

	return rows
}

// makeMdDoc 生成 Markdown 文档
func makeMdDoc(rows []Row) string {
	// 写入 Markdown 文件
	mdContent := fmt.Sprintf("# 煎蛋热榜快照\n\n> 采集时间: %s\n\n", now.Format("2006-01-02 15:04:05"))
	for _, comment := range rows {
		mdContent += fmt.Sprintf("## %s (%s) \n\n", comment.Name, comment.Time)
		mdContent += fmt.Sprintf("- **ID**: %s #%s, **防伪码**: %s\n", comment.Type, comment.ID, comment.Code)
		mdContent += fmt.Sprintf("- **正文**: %s\n", comment.Content)
		mdContent += fmt.Sprintf(
			"- **OO**: [%s](https://jandan.net/t/%s#tucao-like), **XX**: [%s](https://jandan.net/t/%s#tucao-unlike), **Tucao**: [%s](https://jandan.net/t/%s#tucao-list)\n\n",
			comment.OO, comment.ID, comment.XX, comment.ID, comment.Tucao, comment.ID,
		)
	}

	return mdContent
}

// saveToFile 保存到文件
func saveToFile(fileName string, mdContent string) {
	// 写入加密文件
	err := os.WriteFile("./docs/"+fileName+".aes", []byte(mdContent), 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	// 写入未加密文件
	err = os.WriteFile("./docs/"+fileName, []byte("# 煎蛋热榜快照\n\n快照内容将在12小时内开放, 请到 [jandan.net/top](https://jandan.net/top) 查看最新内容"), 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
}

// appendList 追加到 list.md 文件
func appendList(mdFile string) {
	listContent, err := os.ReadFile(listFileName)
	if err != nil {
		fmt.Println("Error reading index.md:", err)
		return
	}

	// 删除 index.md 文件中的 **12小时后开放**
	listContent = []byte(strings.ReplaceAll(string(listContent), " **12小时后开放**", ""))

	// 先判断是否有当前月份行 ^- 12月$ 如果没有就添加
	month := now.Format("01")
	monthRegex := regexp.MustCompile(fmt.Sprintf(`(?m)^- %s月`, month))
	if !monthRegex.Match(listContent) {
		// 添加当前月份行
		listContent = append(listContent, []byte(fmt.Sprintf("\n- %s月", month))...)
	}

	// 检查是否有当前日期行 ^-- 01/02$ 如果没有就添加
	date := now.Format("01月02日")
	dateRegex := regexp.MustCompile(fmt.Sprintf(`(?m)^  - %s:`, date))
	if !dateRegex.Match(listContent) {
		// 添加当前日期行
		listContent = append(listContent, []byte(fmt.Sprintf(
			"\n  - %s: [%s](./docs/%s) **12小时后开放**",
			date, now.Format("15:04:05"), mdFile,
		))...)
	} else {
		dayRegex := regexp.MustCompile(fmt.Sprintf(`(?m)  - %s:([^$]+)$`, date))

		// 在匹配到的日期行后添加新的链接,但不换行, 使用 / 与前一个链接分隔, 用正则表达式 ^  - 01月02日:([^$]+)$ 替换
		listContent = []byte(
			dayRegex.ReplaceAllString(string(listContent), fmt.Sprintf(
				"  - %s:$1 / [%s](./docs/%s) **12小时后开放**",
				date, now.Format("15:04:05"), mdFile,
			)),
		)
	}

	// 最终写入 index.md 文件
	err = os.WriteFile(listFileName, listContent, 0644)
	if err != nil {
		fmt.Println("Error writing to index.md:", err)
		return
	}
}

// 添加一个跨年时的任务逻辑, 当 index 中没有 ## 2024年 时,认为已经跨年,将该文档改为 2024.md, 并创建新的 index.md, 然后将 2024.md 加入到 years.md 中
func crossYear() {
	// 读取 index.md 文件
	indexContent, err := os.ReadFile(listFileName)
	if err != nil {
		fmt.Println("Error reading index.md:", err)
		return
	}

	// 检查是否有当前年份行 ^## [0-9]{4}年$ 并取出匹配到的年份
	yearRegex := regexp.MustCompile(`(?m)^## ([0-9]{4})年`)
	matches := yearRegex.FindStringSubmatch(string(indexContent))
	if len(matches) > 1 && matches[1] != now.Format("2006") {
		// 认为需要跨年,移动 index 到 对应年份
		err := os.Rename(listFileName, fmt.Sprintf("./docs/%s.md", matches[1]))
		if err != nil {
			fmt.Println("Error renaming file:", err)
			return
		}
		// 追加到 years.md 文件
		yearsFileName := "./docs/years.md"
		yearsContent, err := os.ReadFile(yearsFileName)
		if err != nil {
			fmt.Println("Error reading years.md:", err)
			return
		}
		// 在 years.md 文件中追加新的年份
		yearsContent = append(yearsContent, []byte(fmt.Sprintf("- [%s 年](./%s.md)\n", matches[1], matches[1]))...)
		err = os.WriteFile(yearsFileName, yearsContent, 0644)
		if err != nil {
			fmt.Println("Error writing to years.md:", err)
			return
		}

		// 创建新的 index.md 文件
		indexContent = []byte(fmt.Sprintf("# 每日快照索引\n\n> 历年快照请查看 [历年快照](years.md)。\n\n## %s年\n", now.Format("2006")))
		err = os.WriteFile(listFileName, indexContent, 0644)
		if err != nil {
			fmt.Println("Error writing to index.md:", err)
			return
		}
	}
}
