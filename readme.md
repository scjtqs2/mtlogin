# m-team 账号挂机保活脚本

> 根据自行配置的刷新频率进行账号刷新
>
> 连续失败5次才清理cookie重新获取。如果要强制重新拉取cookie可以删掉 data里面的`cookie.db`再重新运行。
>

## 使用之前，需要把`二次认证`从`邮箱验证`更换成`动态验证码二次验证`。`TOTPSECRET`是从二维码解析出来的字符串提取的

`secret`字段。

### env环境变量参数

| Parameter                     | Notes                                                                                                                       |
|-------------------------------|-----------------------------------------------------------------------------------------------------------------------------|
| USERNAME                      | 用户名                                                                                                                         |
| PASSWORD                      | 账号密码                                                                                                                        |
| TOTPSECRET                    | google 二次认证的secret                                                                                                          |
| PROXY                         | 代理服务器地址。例如: `http://192.168.50.123:7890`                                                                                    |
| CRONTAB                       | 定时任务配置，例如: `2 */2 * * *`                                                                                                    |
| QQPUSH                        | 结果推送给的qq号                                                                                                                   |
| QQPUSH_TOKEN                  | 对应QQ号推送的token                                                                                                               |
| M_TEAM_AUTH                   | 直接填写m-team的auth字段，自行用浏览器登录，然后抓取到认证信息                                                                                        |
| UA                            | M_TEAM_AUTH 对应的user-agent                                                                                                   |
| API_HOST                      | api的域名，如果和你的不一样，就换成你自己的。默认值为`api.m-team.io`                                                                                 |
| TIME_OUT                      | api访问的超时时间，单位秒。默认值为60                                                                                                       |
| API_REFERER                   | api的请求的referer值,如果和你的不一样，就换成你自己的。默认为`https://kp.m-team.cc/`                                                                 |
| WXCORPID                      | 企业微信推送通道用。企业ID                                                                                                              |
| WXAGENTSECRET                 | 企业微信推送通道用。应用秘钥                                                                                                              |
| WXAGENTID                     | 企业微信推送通道用。应用ID                                                                                                              |
| WXUSERID                      | 企业微信推送通道用。指定接收消息的成员ID，多个接收者用\|分隔。为空则发送给所有成员                                                                                 |
| MINDELAY                      | 定时任务执行随机延迟，最小延迟（分钟）。默认值0                                                                                                    |
| MAXDELAY                      | 定时任务执行随机延迟，最大延迟（分钟）。默认值0                                                                                                    |
| COOKIE_MODE                   | cookie更新模式，"normal"(默认）,连续失败6次才删。"strict"，每次失败都会删掉cookie尝试重新登录                                                              |
| VERSION                       | http_header里面的version版本号，eg 1.1.2                                                                                           |
| WEB_VERSION                   | http_header里面的webversion版本号, eg 1120                                                                                        |
| M_TEAM_DID                    | http_header里面的did参数。和M_TEAM_AUTH绑定。仅在使用M_TEAM_AUTH的时候需要填                                                                    |
| DING_TALK_ROBOT_WEBHOOK_TOKEN | 钉钉机器人推送地址的access_token                                                                                                      |
| DING_TALK_ROBOT_SECRET        | 钉钉机器人“安全设置”里面的“加签”秘钥，目前仅适配“加签”的方式                                                                                           |
| DING_TALK_ROBOT_AT_MOBILES    | 钉钉机器人消息里面的艾特配置，填对方手机号。多个接受者用`\|`分隔，eg: `138xxxx1234\|137xxxx5678`。留空@all                                                    |
| TGBOT_TOKEN                   | Telegram机器人token，需要在[BotFather](https://t.me/BotFather)注册后获取。eg:`123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`                 |
| TGBOT_CHAT_ID                 | Telegram机器人的聊天ID。去找你的 Bot 聊天一次，然后访问：`https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`替换 <YOUR_TOKEN> 后查看响应数据里的 chat.id。 |
| TGBOT_PROXY                   | Telegram机器人接口用到的代理。如果你是国外环境，留空不用填，国内环境填`http://192.168.50.111:7890`或者`socks5://192.168.50.111:1080`这种代理                     |

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