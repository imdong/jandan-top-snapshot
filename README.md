# jandan-top-snapshot

煎蛋网热榜快照, 每天(06:35 \ 18:35))各创建一次快照.

查看每日快照: [https://page.qs5.org/jandan-top-snapshot/](https://page.qs5.org/jandan-top-snapshot/)

## 蜘蛛说明

爬虫蜘蛛每日有限次数运行, 如有打扰请以以下信息确定身份, 或联系我停止爬虫.

UA md5: `00000000000000000000000000000000`

指纹计算方法: 
```
var userAgent = request.headers.get('User-Agent'); // Mozilla/5.0 JandanTopSnapshot/1.0 repo(https://github.com/imdong/JandanTopSnapshot) BotID/***
var ua_md5 = CryptoJS.MD5(userAgent).toString(); // 7fcfb76a16d89e88ea698319f777f391
```

