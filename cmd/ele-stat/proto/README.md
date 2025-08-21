# Proto File Management

This directory contains protobuf definition files for gRPC services and compilation scripts.

## 📁 File Structure

```
proto/
├── health.proto          # Health check service definition
├── build.sh              # Proto compilation script
└── README.md             # This documentation
```

## 🚀 Quick Start

### Compile all proto files
```bash
./proto/build.sh
```

### View help information
```bash
./proto/build.sh --help
```

### List all proto files
```bash
./proto/build.sh --list
```

### Check compilation status
```bash
./proto/build.sh --status
```

### Clean generated code
```bash
./proto/build.sh --clean
```

### Force recompilation
```bash
./proto/build.sh --force
```

## 🔧 Script Features

### Main Features
- ✅ **Automatic dependency checking** - Check protoc, protoc-gen-go, protoc-gen-go-grpc
- ✅ **Smart path resolution** - Automatically find project root and proto directory
- ✅ **Detailed output** - Support verbose mode and status display
- ✅ **Auto cleanup** - Can clean generated files
- ✅ **Force compilation** - Support forced recompilation
- ✅ **Code formatting** - Automatically format generated Go code

### Compilation Options
- `--proto_path` - Set proto file search path
- `--go_out` - Set Go code output directory
- `--go-grpc_out` - Set gRPC code output directory
- `--go_opt=paths=source_relative` - Use relative path imports

## 📋 Dependency Requirements

### Required Tools
1. **protoc** - Protocol Buffers compiler
   ```bash
   # Ubuntu/Debian
   sudo apt install protobuf-compiler
   
   # macOS
   brew install protobuf
   ```

2. **protoc-gen-go** - Go code generator
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   ```

3. **protoc-gen-go-grpc** - gRPC Go code generator
   ```bash
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

## 🎯 Usage Scenarios

### Development Stage
```bash
# Quick compilation
./proto/build.sh

# Detailed output
./proto/build.sh -v

# Force recompilation
./proto/build.sh --force
```

### Maintenance Stage
```bash
# Check status
```

### Integration into CI/CD
```bash
# Use in build scripts
./proto/build.sh --force

# Check compilation results
if ./proto/build.sh --status | grep -q "✅ All files compiled"; then
    echo "Proto compilation successful"
else
    echo "Proto compilation failed"
    exit 1
fi
```

## 🔍 Troubleshooting

### Common Issues

1. **protoc not found**
   ```bash
   # Install protobuf compiler
   sudo apt install protobuf-compiler
   ```

2. **protoc-gen-go not found**
   ```bash
   # Install Go code generator
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   ```

3. **Compilation failed**
   ```bash
   # Check proto file syntax
   protoc --proto_path=proto proto/*.proto --dry-run
   
   # Check output directory permissions
   ls -la proto/
   ```

4. **Generated file location error**
   - Ensure script is run in the correct directory
   - Check `--go_out` and `--go-grpc_out` path settings

### Debugging Tips

1. **Enable verbose mode**
   ```bash
   ./proto/build.sh -v
   ```

2. **Manually run protoc**
   ```bash
   protoc --proto_path=proto --go_out=proto --go_opt=paths=source_relative \
          --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
          proto/*.proto
   ```

3. **Check file permissions**
   ```bash
   ls -la proto/
   chmod +x proto/build.sh
   ```

## 📝 Notes

1. **Output path** - Generated files are output to the proto directory
2. **Relative paths** - Use `paths=source_relative` to ensure correct import paths
3. **Go modules** - Ensure `go_package` option matches project module path
4. **File cleanup** - Cleanup function deletes all generated `.pb.go` and `_grpc.pb.go` files

## 🤝 Contributing

To add new proto files:
1. Create `.proto` files in the `proto/` directory
2. Ensure `go_package` option is correct
3. Run `./proto/build.sh` to test compilation
4. Update this document

## 📚 Related Links

- [Protocol Buffers Official Documentation](https://developers.google.com/protocol-buffers)
- [gRPC Go Guide](https://grpc.io/docs/languages/go/)
- [protoc-gen-go Documentation](https://github.com/golang/protobuf)
- [protoc-gen-go-grpc Documentation](https://github.com/grpc/grpc-go)
