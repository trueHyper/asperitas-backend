#!/bin/bash

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [ "$1" = "docker" ]; then
    START=".env.docker"
    docker-compose up -d
else
    START=".env.local"
fi

export START=$START
echo -e "${YELLOW}Using environment file: $START${NC}"

echo -e "\n${GREEN}Start test...${NC}"
go test ./...
go run ./cmd/redditclone

