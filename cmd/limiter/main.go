package main

import (
	"fmt"

	"github.com/remnawave/limiter/internal/version"
)

func main() {
	fmt.Printf("remnawave-limiter v%s\n", version.Version)
}
