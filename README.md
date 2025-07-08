# element backend common library

## API 列表

### 用户相关 API

#### GetUserProfile
获取用户个人信息，包括基本信息、游戏统计数据和卡牌使用统计。

- **Action**: `GetUserProfile`
- **认证**: 无需认证
- **文档**: [GetUserProfile API 文档](docs/GetUserProfile_API.md)

#### SetUserProfile
设置用户个人信息。

- **Action**: `SetUserProfile`
- **认证**: Cookie 认证
- **文档**: 参考现有代码

### 登录相关 API

- `GetLoginCode`: 获取登录验证码
- `LoginWeb3`: Web3 登录
- `RefreshTokens`: 刷新令牌
- `IsWalletLoggedIn`: 检查钱包登录状态

## 测试

### E2E 测试

Set redis host and password in "./sever/e2e_test/test_server/test_config.yml"

Then run: `cd ./sever/e2e_test && bash -c e2e_test.sh`

### API 测试

```bash
# 运行所有 API 测试
./test/api/script/run_all_tests.sh

# 单独测试各个 API
# 测试 GetUserProfile API
./test/api/script/test_get_user_profile.sh

# 测试登录相关 API
./test/api/script/test_login.sh

# 测试 SetUserProfile API
./test/api/script/test_set_user_profile.sh
```

## 开发

### 编译
```bash
go build ./...
```

### 运行服务器
```bash
# 配置服务器参数后运行
go run main.go
```

