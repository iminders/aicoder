package ui

import (
	"fmt"
	"strings"
)

// Renderer handles streaming output to the terminal with basic ANSI formatting.
type Renderer struct {
	buf        strings.Builder
	inCodeBlock bool
	codeLang   string
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

// Write appends a delta to the internal buffer and prints it.
func (r *Renderer) Write(delta string) {
	r.buf.WriteString(delta)
	// Print inline — for a real TUI we'd use bubbletea here
	fmt.Print(delta)
}

// Flush finalises any open formatting contexts.
func (r *Renderer) Flush() {
	if r.inCodeBlock {
		fmt.Print("\033[0m")
		r.inCodeBlock = false
	}
}

// PrintInfo prints an informational message in cyan.
func PrintInfo(msg string) {
	fmt.Printf("\033[36m%s\033[0m\n", msg)
}

// PrintError prints an error message in red.
func PrintError(msg string) {
	fmt.Printf("\033[31m✗ %s\033[0m\n", msg)
}

// PrintSuccess prints a success message in green.
func PrintSuccess(msg string) {
	fmt.Printf("\033[32m✓ %s\033[0m\n", msg)
}

// PrintWarn prints a warning in yellow.
func PrintWarn(msg string) {
	fmt.Printf("\033[33m⚠ %s\033[0m\n", msg)
}

// PrintDivider prints a horizontal divider.
func PrintDivider() {
	fmt.Println("\033[90m" + strings.Repeat("─", 60) + "\033[0m")
}

// PrintBanner prints the aicoder startup banner.
func PrintBanner(version, model string) {
	fmt.Println("\033[1;34m")
	fmt.Println("  ██████╗  ██╗ ██████╗ ██████╗ ██████╗ ███████╗██████╗ ")
	fmt.Println(" ██╔══██╗ ██║██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔══██╗")
	fmt.Println(" ███████║ ██║██║     ██║   ██║██║  ██║█████╗  ██████╔╝")
	fmt.Println(" ██╔══██║ ██║██║     ██║   ██║██║  ██║██╔══╝  ██╔══██╗")
	fmt.Println(" ██║  ██║ ██║╚██████╗╚██████╔╝██████╔╝███████╗██║  ██║")
	fmt.Println(" ╚═╝  ╚═╝ ╚═╝ ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝")
	fmt.Println("\033[0m")
	fmt.Printf("  \033[90mVersion: %s  Model: %s\033[0m\n", version, model)
	fmt.Printf("  \033[90m输入 /help 查看命令，Ctrl+D 或 /exit 退出\033[0m\n\n")
}
