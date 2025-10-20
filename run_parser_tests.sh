#!/bin/bash

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}PlexLinker HTML Parser Test Suite${NC}"
echo "===================================="
echo ""

# Check if port 8888 is available
if lsof -Pi :8888 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}Port 8888 is already in use. Attempting to kill existing process...${NC}"
    kill $(lsof -t -i:8888) 2>/dev/null
    sleep 1
fi

# Start mock server in background
echo -e "${BLUE}Starting mock server...${NC}"
go run test_parser_mock_server.go &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to be ready..."
for i in {1..10}; do
    if curl -s http://localhost:8888 >/dev/null 2>&1; then
        echo -e "${GREEN}Server is ready!${NC}"
        break
    fi
    sleep 1
    if [ $i -eq 10 ]; then
        echo -e "${RED}Server failed to start${NC}"
        exit 1
    fi
done

echo ""

# Run tests
echo -e "${BLUE}Running tests...${NC}"
echo ""
go run test_parser.go
TEST_EXIT_CODE=$?

# Clean up
echo ""
echo -e "${YELLOW}Stopping mock server...${NC}"
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
else
    echo -e "${RED}Some tests failed.${NC}"
fi

exit $TEST_EXIT_CODE
