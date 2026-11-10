# jandan-top-snapshot

煎蛋热榜 JSON 快照，每天北京时间约 06:35、18:35 采集（Actions 可能延迟）。

浏览入口：[每日快照](https://page.qs5.org/jandan-top-snapshot/) · [煎蛋热榜](https://jandan.net/top)

仓库只存 JSON 数据和静态浏览器页面，图片与 MP4 打包存入每次快照对应的 GitHub Release。旧 Markdown/HTML 存档已移除，不再进行 AES 延时开放。

## 采集与发布

使用 Go 1.23.4 或更新版本运行：

```sh
go test ./...
go run .
```

默认请求 `https://jandan.net/api/top/post/26402?cdn_line=default`；该地址返回空数据时自动尝试去掉 `cdn_line`。接口错误、空列表或媒体下载失败会以非零状态退出，不更新索引。`API_URL` 可覆盖接口，`BOT_ID` 可设置蜘蛛标识，`GITHUB_REPOSITORY` 可覆盖 Release 所属仓库（默认 `imdong/jandan-top-snapshot`）。

输出：

- `docs/snapshots/YYYYMM/YYYYMMDD-HHMMSS.json`：采集时间、接口地址、原始 data 全部字段、媒体 URL 到包内文件名的映射。
- `docs/snapshots.json`：按时间倒序的入口索引及 Release 下载地址。
- `dist/media-YYYYMMDD-HHMMSS.zip`：无压缩 ZIP，包含正文与神吐槽的图片、视频，文件名按 URL 的 SHA-256 去重。此目录不提交 Git。

普通图片在路径末尾添加 `.webp`，GIF 映射到同域名 `/large/文件名.mp4`；现有 WebP、MP4、WebM 保留。保留各图片原有域名和查询参数。每个请求超时 60 秒，最多尝试三次。

定时和手动运行 `.github/workflows/cron.yml` 均会先测试、采集，再上传 Release 图片包，最后提交 JSON。需要 Actions 的 `contents: write` 权限。GitHub Pages 请从 `main` 分支的 `/docs` 发布。这里的代码修改本身不会立即运行远程工作流或发布 Release。

本地运行只生成文件，不上传 Release；可手动 `gh release create snapshot-时间戳 dist/media-时间戳.zip --latest=false`。首次使用可在 Actions 中手动运行“定时采集”。

## 浏览与图片包

`index.html` 动态加载时间列表，选择时间后渲染作者、ID、正文、神吐槽作者和内容、OO、XX、吐槽数。URL 的 hash 可分享指定快照。API 的 HTML 使用标签白名单重建，事件属性与脚本不会插入页面。

默认从原站 CDN 加载媒体。选择“Release 图片包”会下载对应 ZIP 并用 Blob URL 显示其中的图片与 MP4；切换快照会释放上一个包。整包需要载入内存，视频很多时首次加载可能较慢。读取失败时提示原因并暂用原站图片。

GitHub Release 下载在浏览器中可能受 CORS 限制，可选：

1. 安装 Tampermonkey 等 userscript 管理器，再点击页面上的“安装跨域辅助脚本”，安装后刷新页面。脚本仅接受本仓库 Release 图片包 URL，通过 `GM_xmlhttpRequest` 读取。
2. 点击“下载图片包”，再使用“导入图片包”选择 ZIP。浏览器本地读取无需跨域权限；会检查当前快照需要的文件是否齐全。

自定义域名或 fork 需要同步修改 `docs/release-bridge.user.js` 的 `@match` 和仓库 URL 白名单。辅助脚本只负责 Release ZIP，不代理原站图片。

本地预览（不要直接使用 file:// 打开）：

```sh
python3 -m http.server 8000 --directory docs
```

访问 `http://localhost:8000`。没有快照时页面会显示提示。
