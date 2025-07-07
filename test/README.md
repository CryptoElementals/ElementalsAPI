## 目录结构

```
test/
├── README.md          # 本文件
├── run_tests.sh       # 测试运行脚本
└── db_test.go         # 数据库测试
```

## 运行测试

### 方法1：使用测试脚本
```bash
./test/run_tests.sh
```

### 方法2：直接运行
```bash
cd test
go test -v
```

### 方法3：运行特定测试
```bash
cd test
go test -v -run TestDatabaseConnection
```

## 测试文件说明

### db_test.go
- 测试数据库连接
- 验证配置加载
- 检查数据库连通性


## 添加新测试

当您需要添加新的测试时：

1. 在 `test/` 目录下创建新的测试文件
2. 使用正确的导入路径（相对于项目根目录）
3. 更新 `run_tests.sh` 脚本（如果需要）

## 注意事项

- 测试文件使用 `package test`
- 配置文件路径使用 `../config.yaml`
- 确保数据库和Redis服务正在运行 