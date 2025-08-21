#!/bin/bash

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script information
SCRIPT_NAME="Client Test Runner"
VERSION="1.0.0"

# Default configuration
DEFAULT_TEST_PACKAGE="./rpc/client/..."
VERBOSE_FLAG=""
COVERAGE_FLAG=""
RACE_FLAG=""
SHORT_FLAG=""
TEST_FILTER=""
RUN_MODE="all"

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_PACKAGE="$PROJECT_ROOT/rpc/client/..."

# Help information
show_help() {
    echo -e "${CYAN}${SCRIPT_NAME} v${VERSION}${NC}"
    echo "A flexible Go test runner script"
    echo ""
    echo "Usage: $0 [options] [test names]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help information"
    echo "  -v, --verbose           Verbose output mode"
    echo "  -c, --coverage          Generate coverage report"
    echo "  -r, --race              Enable race detection"
    echo "  -s, --short             Run only short tests"
    echo "  -l, --list              List all available test cases"
    echo "  -f, --filter PATTERN    Filter tests by pattern"
    echo "  -m, --mode MODE         Run mode (all|unit|integration|real|custom)"
    echo "  -t, --timeout DURATION  Set test timeout (default: 5m)"
    echo "  -o, --output FILE       Save output to file"
    echo "  -j, --jobs N            Number of parallel test runs (default: 4)"
    echo ""
    echo "Run modes:"
    echo "  all            Run all tests (default)"
    echo "  unit           Run only unit tests (mock mode)"
    echo "  integration    Run only integration tests (real connection)"
    echo "  real           Run only real connection tests"
    echo "  custom         Run custom test cases"
    echo ""
    echo "Test case examples:"
    echo "  TestDefaultClientConfig"
    echo "  TestRealGRPCConnection"
    echo "  TestHealthClient_HealthCheck"
    echo "  TestClientManager.*"
    echo ""
    echo "Examples:"
    echo "  $0                           # Run all tests"
    echo "  $0 -v                        # Run all tests in verbose mode"
    echo "  $0 -c -r                     # Coverage + race detection"
    echo "  $0 -m real                   # Run only real connection tests"
    echo "  $0 TestRealGRPCConnection    # Run specific test"
    echo "  $0 -f TestHealth             # Run all health check related tests"
    echo "  $0 -m custom TestDefaultClientConfig TestNewStatClient"
}

# List all available test cases
list_tests() {
    echo -e "${BLUE}Available test cases:${NC}"
    echo ""
    
    echo -e "${CYAN}Basic functionality tests:${NC}"
    go test -list="TestDefault" $TEST_PACKAGE 2>/dev/null | grep -E "^Test" | sort
    echo ""
    
    echo -e "${CYAN}Health check tests:${NC}"
    go test -list="TestHealth" $TEST_PACKAGE 2>/dev/null | grep -E "^Test" | sort
    echo ""
    
    echo -e "${CYAN}Client manager tests:${NC}"
    go test -list="TestClientManager" $TEST_PACKAGE 2>/dev/null | grep -E "^Test" | sort
    echo ""
    
    echo -e "${CYAN}Real connection tests:${NC}"
    go test -list="TestReal" $TEST_PACKAGE 2>/dev/null | grep -E "^Test" | sort
    echo ""
    
    echo -e "${CYAN}Other tests:${NC}"
    go test -list="Test" $TEST_PACKAGE 2>/dev/null | grep -E "^Test" | grep -v -E "(TestDefault|TestHealth|TestClientManager|TestReal)" | sort
}

# Check dependencies
check_dependencies() {
    echo -e "${BLUE}Checking dependencies...${NC}"
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed${NC}"
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}Go version: ${GO_VERSION}${NC}"
    
    # Check test dependencies
    echo -e "${BLUE}Checking test dependencies...${NC}"
    go mod tidy
    go get -t ./...
    
    echo -e "${GREEN}Dependency check complete${NC}"
}

# Build tests
build_tests() {
    echo -e "${BLUE}Building tests...${NC}"
    
    if go build $TEST_PACKAGE; then
        echo -e "${GREEN}Build successful${NC}"
    else
        echo -e "${RED}Build failed${NC}"
        exit 1
    fi
}

# Run tests
run_tests() {
    local test_args=""
    local test_package="$TEST_PACKAGE"
    
    # Build basic test parameters
    if [[ -n "$VERBOSE_FLAG" ]]; then
        test_args="$test_args -v"
    fi
    
    if [[ -n "$COVERAGE_FLAG" ]]; then
        test_args="$test_args -cover"
    fi
    
    if [[ -n "$RACE_FLAG" ]]; then
        test_args="$test_args -race"
    fi
    
    if [[ -n "$SHORT_FLAG" ]]; then
        test_args="$test_args -short"
    fi
    
    if [[ -n "$TEST_FILTER" ]]; then
        test_args="$test_args -run $TEST_FILTER"
    fi
    
    # Set test parameters based on run mode
    case "$RUN_MODE" in
        "unit")
            echo -e "${BLUE}Running unit tests (mock mode)...${NC}"
            test_args="$test_args -run 'TestDefault|TestNewStatClient|TestClientConfig|TestHealthClient|TestHealthRequest|TestHealthResponse|TestClientManager|TestPrintClientInfo|TestStatClient'"
            ;;
        "integration")
            echo -e "${BLUE}Running integration tests (real connection)...${NC}"
            test_args="$test_args -run 'TestRealGRPCConnection|TestClientManagerRealConnection|TestBatchHealthCheckRealServer'"
            ;;
        "real")
            echo -e "${BLUE}Running real connection tests...${NC}"
            test_args="$test_args -run 'TestRealGRPCConnection|TestClientManagerRealConnection|TestBatchHealthCheckRealServer'"
            ;;
        "custom")
            if [[ -n "$TEST_FILTER" ]]; then
                echo -e "${BLUE}Running custom tests: $TEST_FILTER${NC}"
            else
                echo -e "${BLUE}Running custom test cases...${NC}"
            fi
            ;;
        "all")
            echo -e "${BLUE}Running all tests...${NC}"
            ;;
    esac
    
    # If there are specific test names, add them to the parameters
    if [[ $# -gt 0 ]]; then
        local test_names="$*"
        echo -e "${BLUE}Running specific tests: $test_names${NC}"
        # Convert test names to regex pattern
        local pattern=$(echo "$test_names" | tr ' ' '|')
        test_args="$test_args -run $pattern"
    fi
    
    echo -e "${CYAN}Executing command: go test $test_args $test_package${NC}"
    echo ""
    
    # Execute tests
    if go test $test_args $test_package; then
        echo ""
        echo -e "${GREEN}✅ All tests passed!${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}❌ Some tests failed${NC}"
        return 1
    fi
}

# Generate coverage report
generate_coverage_report() {
    if [[ -n "$COVERAGE_FLAG" ]]; then
        echo -e "${BLUE}Generating coverage report...${NC}"
        
        # Generate coverage file
        go test -coverprofile=coverage.out $TEST_PACKAGE
        
        # Generate HTML report
        go tool cover -html=coverage.out -o coverage.html
        
        # Display coverage statistics
        go tool cover -func=coverage.out
        
        echo -e "${GREEN}Coverage report generated:${NC}"
        echo "  - coverage.out (raw data)"
        echo "  - coverage.html (HTML report)"
    fi
}

# Clean up temporary files
cleanup() {
    echo -e "${BLUE}Cleaning up temporary files...${NC}"
    rm -f coverage.out coverage.html
    go clean
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Main function
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--verbose)
                VERBOSE_FLAG="-v"
                shift
                ;;
            -c|--coverage)
                COVERAGE_FLAG="-cover"
                shift
                ;;
            -r|--race)
                RACE_FLAG="-race"
                shift
                ;;
            -s|--short)
                SHORT_FLAG="-short"
                shift
                ;;
            -l|--list)
                list_tests
                exit 0
                ;;
            -f|--filter)
                TEST_FILTER="$2"
                shift 2
                ;;
            -m|--mode)
                RUN_MODE="$2"
                shift 2
                ;;
            -t|--timeout)
                export GOTEST_TIMEOUT="$2"
                shift 2
                ;;
            -o|--output)
                exec > "$2" 2>&1
                shift 2
                ;;
            -j|--jobs)
                export GOMAXPROCS="$2"
                shift 2
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
    echo -e "${BLUE}Run mode: $RUN_MODE${NC}"
    echo ""
    
    # Check dependencies
    check_dependencies
    
    # Build tests
    build_tests
    
    # Run tests
    run_tests "$@"
    test_result=$?
    
    # Generate coverage report
    generate_coverage_report
    
    # Clean up
    cleanup
    
    # Return test result
    exit $test_result
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
