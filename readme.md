# m-team 账号挂机保活脚本

> 根据自行配置的刷新频率进行账号刷新
>

### env环境变量参数

| Parameter    | Notes                                    |
|--------------|------------------------------------------|
| USERNAME     | 用户名                                      |
| PASSWORD     | 账号密码                                     |
| OPTSECRET    | google 二次认证的secret                       |
| PROXY        | 代理服务器地址。例如: `http://192.168.50.123:7890` |
| CRONTAB      | 定时任务配置，例如: `2 */2 * * *`                 |
| QQPUSH       | 结果推送给的qq号                                |
| QQPUSH_TOKEN | 对应QQ号推送的token                            |

## docker

If you prefer the `docker cli` execute the following command.

```bash
docker run -d \
  --name=mtlogin \
  -e USERNAME=aaaaaa \
  -e PASSWORD=bbbbbbbb \
  -e CRONTAB="2 */2 * * *" \
  --restart unless-stopped \
  ghcr.io/scjtqs2/mtlogin:edge
```