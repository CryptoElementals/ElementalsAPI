#!/bin/bash

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script information
SCRIPT_NAME="Proto Builder"
VERSION="1.0.0"

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Help information
show_help() {
    echo -e "${CYAN}${SCRIPT_NAME} v${VERSION}${NC}"
    echo "Proto file compilation script"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help information"
    echo "  -v, --verbose           Verbose output mode"
    echo "  -c, --clean             Clean generated code"
    echo "  -f, --force             Force recompilation"
    echo "  -w, --watch             Watch file changes and auto-compile"
    echo "  -l, --list              List all proto files"
    echo "  -s, --status            Show compilation status"
    echo ""
    echo "Examples:"
    echo "  $0                      # Compile all proto files"
    echo "  $0 -v                   # Compile in verbose mode"
    echo "  $0 -c                   # Clean generated code"
    echo "  $0 -f                   # Force recompilation"
    echo "  $0 -w                   # Watch mode"
}

# Check dependencies
check_dependencies() {
    echo -e "${BLUE}Checking dependencies...${NC}"
    
    # Check if protoc is installed
    if ! command -v protoc &> /dev/null; then
        echo -e "${RED}Error: protoc is not installed${NC}"
        echo "Please install protoc:"
        echo "  Ubuntu/Debian: sudo apt install protobuf-compiler"
        echo "  macOS: brew install protobuf"
        echo "  Windows: Download protoc binary"
        exit 1
    fi
    
    # Check if protoc-gen-go is installed
    if ! command -v protoc-gen-go &> /dev/null; then
        echo -e "${YELLOW}Warning: protoc-gen-go is not installed${NC}"
        echo "Installing protoc-gen-go..."
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    fi
    
    # Check if protoc-gen-go-grpc is installed
    if ! command -v protoc-gen-go-grpc &> /dev/null; then
        echo -e "${YELLOW}Warning: protoc-gen-go-grpc is not installed${NC}"
        echo "Installing protoc-gen-go-grpc..."
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    fi
    
    # Display version information
    echo -e "${GREEN}protoc version: $(protoc --version)${NC}"
    echo -e "${GREEN}protoc-gen-go version: $(protoc-gen-go --version 2>/dev/null || echo "unknown")${NC}"
    echo -e "${GREEN}protoc-gen-go-grpc version: $(protoc-gen-go-grpc --version 2>/dev/null || echo "unknown")${NC}"
    
    echo -e "${GREEN}Dependency check complete${NC}"
}

# List all proto files
list_proto_files() {
    echo -e "${BLUE}Proto file list:${NC}"
    echo ""
    
    local proto_dir="$SCRIPT_DIR"
    local count=0
    
    for file in "$proto_dir"/*.proto; do
        if [[ -f "$file" ]]; then
            local filename=$(basename "$file")
            local size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo "unknown")
            echo -e "  ${CYAN}$filename${NC} (${size} bytes)"
            ((count++))
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        echo -e "${YELLOW}No .proto files found${NC}"
    else
        echo ""
        echo -e "${GREEN}Total: $count proto files${NC}"
    fi
}

# Show compilation status
show_status() {
    echo -e "${BLUE}Compilation status:${NC}"
    echo ""
    
    local proto_dir="$SCRIPT_DIR"
    local project_root="$PROJECT_ROOT"
    local proto_count=0
    local pb_count=0
    local grpc_count=0
    
    # Count proto files
    for file in "$proto_dir"/*.proto; do
        if [[ -f "$file" ]]; then
            ((proto_count++))
        fi
    done
    
    # Count generated Go files
    for file in "$proto_dir"/*.pb.go; do
        if [[ -f "$file" ]]; then
            ((pb_count++))
        fi
    done
    
    for file in "$proto_dir"/*_grpc.pb.go; do
        if [[ -f "$file" ]]; then
            ((grpc_count++))
        fi
    done
    
    echo -e "  Proto files: ${CYAN}$proto_count${NC}"
    echo -e "  Generated .pb.go files: ${CYAN}$pb_count${NC}"
    echo -e "  Generated _grpc.pb.go files: ${CYAN}$grpc_count${NC}"
    
    if [[ $proto_count -gt 0 && $pb_count -gt 0 && $grpc_count -gt 0 ]]; then
        echo -e "${GREEN}✅ All files compiled${NC}"
    elif [[ $proto_count -gt 0 ]]; then
        echo -e "${YELLOW}⚠️  Need compilation${NC}"
    else
        echo -e "${RED}❌ No proto files found${NC}"
    fi
}

# Clean generated code
clean_generated() {
    echo -e "${BLUE}Cleaning generated code...${NC}"
    
    local proto_dir="$SCRIPT_DIR"
    local removed_count=0
    
    # Clean .pb.go files
    for file in "$proto_dir"/*.pb.go; do
        if [[ -f "$file" ]]; then
            rm -f "$file"
            echo -e "   Deleted: $(basename "$file")"
            ((removed_count++))
        fi
    done
    
    # Clean _grpc.pb.go files
    for file in "$proto_dir"/*_grpc.pb.go; do
        if [[ -f "$file" ]]; then
            rm -f "$file"
            echo -e "   Deleted: $(basename "$file")"
            ((removed_count++))
        fi
    done
    
    # Clean Go cache
    go clean
    
    if [[ $removed_count -eq 0 ]]; then
        echo -e "${YELLOW}No files to clean found${NC}"
    else
        echo -e "${GREEN}Cleaning complete, deleted $removed_count files${NC}"
    fi
}

# Compile proto files
build_proto() {
    local verbose_flag=""
    local force_flag=""
    
    if [[ -n "$VERBOSE_FLAG" ]]; then
        verbose_flag="-v"
    fi
    
    if [[ -n "$FORCE_FLAG" ]]; then
        echo -e "${BLUE}Force recompilation...${NC}"
        clean_generated
    fi
    
    echo -e "${BLUE}Compiling proto files...${NC}"
    
    # Switch to project root directory
    cd "$PROJECT_ROOT"
    
    # Check if there are proto files
    local proto_files=("$SCRIPT_DIR"/*.proto)
    if [[ ! -f "${proto_files[0]}" ]]; then
        echo -e "${RED}Error: No .proto files found${NC}"
        exit 1
    fi
    
    # Compile command
    local cmd="protoc --proto_path=$SCRIPT_DIR --go_out=$SCRIPT_DIR --go_opt=paths=source_relative --go-grpc_out=$SCRIPT_DIR --go-grpc_opt=paths=source_relative $SCRIPT_DIR/*.proto"
    
    if [[ -n "$verbose_flag" ]]; then
        echo -e "${CYAN}Executing command: $cmd${NC}"
    fi
    
    # Execute compilation
    if eval "$cmd"; then
        echo -e "${GREEN}✅ Proto compilation successful!${NC}"
        
        # Display generated files
        local generated_count=0
        echo -e "${BLUE}Checking generated files...${NC}"
        
        # Check .pb.go files
        for file in "$SCRIPT_DIR"/*.pb.go; do
            if [[ -f "$file" ]]; then
                echo -e "   Generated: $(basename "$file")"
                ((generated_count++))
            fi
        done
        
        # Check _grpc.pb.go files
        for file in "$SCRIPT_DIR"/*_grpc.pb.go; do
            if [[ -f "$file" ]]; then
                echo -e "   Generated: $(basename "$file")"
                ((generated_count++))
            fi
        done
        
        if [[ $generated_count -eq 0 ]]; then
            echo -e "${YELLOW}Warning: No files generated${NC}"
            echo -e "${BLUE}Debug info:${NC}"
            echo -e "  Current directory: $(pwd)"
            echo -e "  Proto directory: $SCRIPT_DIR"
            echo -e "  Proto files: $(ls -la "$SCRIPT_DIR"/*.proto 2>/dev/null | wc -l)"
            echo -e "  Generated Go files: $(ls -la "$SCRIPT_DIR"/*.go 2>/dev/null | wc -l)"
        else
            echo -e "${GREEN}Total generated: $generated_count files${NC}"
        fi
        
        # Format generated Go code
        if [[ $generated_count -gt 0 ]]; then
            echo -e "${BLUE}Formatting generated Go code...${NC}"
            if command -v gofmt &> /dev/null; then
                gofmt -w "$SCRIPT_DIR"/*.go
                echo -e "${GREEN}Code formatting complete${NC}"
            fi
        fi
        
    else
        echo -e "${RED}❌ Proto compilation failed${NC}"
        exit 1
    fi
}

# Watch mode
watch_mode() {
    echo -e "${BLUE}Watch mode started, press Ctrl+C to stop...${NC}"
    echo -e "${CYAN}Watching directory: $SCRIPT_DIR${NC}"
    echo ""
    
    # Check if inotify-tools (Linux) or fswatch (macOS) is installed
    local watcher=""
    if command -v inotifywait &> /dev/null; then
        watcher="inotifywait"
    elif command -v fswatch &> /dev/null; then
        watcher="fswatch"
    else
        echo -e "${RED}Error: File watching tool not found${NC}"
        echo "Please install inotify-tools (Linux) or fswatch (macOS)"
        exit 1
    fi
    
    # Watch for file changes
    if [[ "$watcher" == "inotifywait" ]]; then
        inotifywait -m -r -e modify,create,delete "$SCRIPT_DIR" | while read -r directory events filename; do
            if [[ "$filename" == *.proto ]]; then
                echo -e "${YELLOW}Detected change: $filename${NC}"
                build_proto
            fi
        done
    elif [[ "$watcher" == "fswatch" ]]; then
        fswatch -o "$SCRIPT_DIR" | while read -r; do
            echo -e "${YELLOW}Detected change, recompiling...${NC}"
            build_proto
        done
    fi
}

# Main function
main() {
    local verbose_flag=""
    local clean_flag=""
    local force_flag=""
    local watch_flag=""
    local list_flag=""
    local status_flag=""
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--verbose)
                verbose_flag="-v"
                VERBOSE_FLAG="-v"
                shift
                ;;
            -c|--clean)
                clean_flag="-c"
                shift
                ;;
            -f|--force)
                force_flag="-f"
                FORCE_FLAG="-f"
                shift
                ;;
            -w|--watch)
                watch_flag="-w"
                shift
                ;;
            -l|--list)
                list_flag="-l"
                shift
                ;;
            -s|--status)
                status_flag="-s"
                shift
                ;;
            -*)
                echo -e "${RED}Unknown option: $1${NC}"
                show_help
                exit 1
                ;;
            *)
                break
                ;;
        esac
    done
    
    # Display script information
    echo -e "${CYAN}${SCRIPT_NAME} v${VERSION}${NC}"
    echo -e "${BLUE}Project root: $PROJECT_ROOT${NC}"
    echo -e "${BLUE}Proto directory: $SCRIPT_DIR${NC}"
    echo ""
    
    # Execute corresponding operation based on parameters
    if [[ -n "$list_flag" ]]; then
        list_proto_files
        exit 0
    fi
    
    if [[ -n "$status_flag" ]]; then
        show_status
        exit 0
    fi
    
    if [[ -n "$clean_flag" ]]; then
        clean_generated
        exit 0
    fi
    
    if [[ -n "$watch_flag" ]]; then
        check_dependencies
        watch_mode
        exit 0
    fi
    
    # Default compilation
    check_dependencies
    build_proto
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
