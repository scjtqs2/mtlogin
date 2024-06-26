# m-team 账号挂机保活脚本

> 根据自行配置的刷新频率进行账号刷新
>

### env环境变量参数

| Parameter    | Notes                                    |
|--------------|------------------------------------------|
| USERNAME     | 用户名                                      |
| PASSWORD     | 账号密码                                     |
| TOTPSECRET   | google 二次认证的secret                       |
| PROXY        | 代理服务器地址。例如: `http://192.168.50.123:7890` |
| CRONTAB      | 定时任务配置，例如: `2 */2 * * *`                 |
| QQPUSH       | 结果推送给的qq号                                |
| QQPUSH_TOKEN | 对应QQ号推送的token                            |
| M_TEAM_AUTH  | 直接填写m-team的auth字段，自行用浏览器登录，然后抓取到认证信息     |
| UA           | M_TEAM_AUTH 对应的user-agent                |

## docker

If you prefer the `docker cli` execute the following command.

```bash
docker run -d \
  --name=mtlogin \
  -v /yourpath/auth.db:/data \
  -e USERNAME=aaaaaa \
  -e PASSWORD=bbbbbbbb \
  -e TOTPSECRET=cccccccc \
  -e CRONTAB="2 */2 * * *" \
  --restart unless-stopped \
  ghcr.io/scjtqs2/mtlogin:edge
```

如果你用透传auth的方式、

```bash
docker run -d \
  --name=mtlogin \
  -v /yourpath/auth.db:/data \
  -e M_TEAM_AUTH="eyJhbGciOiJIUzUx9999.eyJzdWIiOiJzXXXXcXMiLCJ1aWQiOjMyNDI5MiwianRpIjoiY2JlNGE1MWUtZWMzOC00MTExLWEzNmYtY2E5N2RmMGI4NzdhIiwiaXNzIjoiaHR0cHM6Ly9hcGkubS10ZWFtLmNjIiwiaWF0IjoxNzE3MzkzMjk1LCJleHAiOjE3MTk5ODUyOTV9.B1dBTSNHcdSHziNqgGs8zlknxc84XXXXXaiRJNyvSLBkarHQiTzdhN-HA-BZf_AaVYhxwHRSmSDfV41PsRwH_Q" \
  -e UA="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0" \
  -e CRONTAB="2 */2 * * *" \
  --restart unless-stopped \
  ghcr.io/scjtqs2/mtlogin:edge
```