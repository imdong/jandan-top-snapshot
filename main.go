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

	err = os.WriteFile("./docs/"+fileName+".aes", []byte(mdContent), 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	err = os.WriteFile("./docs/"+fileName, []byte("# 煎蛋热榜快照\n\n快照内容将在12小时内开放, 请到 [jandan.net/top](https://jandan.net/top) 查看最新内容"), 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	// 更新 list.json 文件
	listFileName := "./docs/index.md"

	// 删除 index.md 文件中的 **12小时后开放**
	listContent, err := os.ReadFile(listFileName)
	if err != nil {
		fmt.Println("Error reading index.md:", err)
		return
	}
	listContent = []byte(strings.ReplaceAll(string(listContent), " **12小时后开放**", ""))
	err = os.WriteFile(listFileName, listContent, 0644)
	if err != nil {
		fmt.Println("Error writing to index.md:", err)
		return
	}

	// 向 index.md 文件追加内容
	listFile, err := os.OpenFile(listFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening index.md:", err)
		return
	}
	defer func(listFile *os.File) {
		_ = listFile.Close()
	}(listFile)

	// 写入 index.md 文件
	_, err = listFile.WriteString(fmt.Sprintf("- [%s](%s) **12小时后开放**\n", now.Format("01/02 15:04:05"), fileName))
	if err != nil {
		fmt.Println("Error writing to index.md:", err)
		return
	}

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
			 comment.OO,comment.ID, comment.XX, comment.ID,comment.Tucao, comment.ID,
		)
	

	return mdContent
}
