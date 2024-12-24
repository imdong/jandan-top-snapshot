# jandan-top-snapshot

[煎蛋网](https://jandan.net/) 热榜快照, 每天 (06:35 \ 18:35) 左右各创建一次快照.

查看每日快照: [https://page.qs5.org/jandan-top-snapshot/](https://page.qs5.org/jandan-top-snapshot/)

查看热榜实况: [https://jandan.net/top](https://jandan.net/top)

## 蜘蛛说明

爬虫蜘蛛每日有限次数运行, 如有打扰请以以下信息确定身份(BotID 不变), 或联系我停止爬虫.

UA md5: `e936f8a4a1180641f46d25e01038ad34` (2024-12-24 更新)

指纹计算方法: 
```
var userAgent = request.headers.get('User-Agent'); // Mozilla/5.0 JandanTopSnapshot/1.0 repo(https://github.com/imdong/JandanTopSnapshot) BotID/***
var ua_md5 = CryptoJS.MD5(userAgent).toString(); // 7fcfb76a16d89e88ea698319f777f391
```

已加入自毁机制, 如果连续 14 次遇到 403 响应则自动停止爬虫.
