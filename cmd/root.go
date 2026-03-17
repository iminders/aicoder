package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/iminders/aicoder/internal/agent"
	"github.com/iminders/aicoder/internal/config"
	"github.com/iminders/aicoder/internal/llm"
	anthropicprovider "github.com/iminders/aicoder/internal/llm/anthropic"
	openaiprovider "github.com/iminders/aicoder/internal/llm/openai"
	"github.com/iminders/aicoder/internal/logger"
	"github.com/iminders/aicoder/internal/slash"
	"github.com/iminders/aicoder/internal/ui"
	"github.com/iminders/aicoder/pkg/version"

	// Register tools
	_ "github.com/iminders/aicoder/internal/tools/filesystem"
	_ "github.com/iminders/aicoder/internal/tools/search"
	_ "github.com/iminders/aicoder/internal/tools/shell"
)

// flags holds CLI flag values.
var flags struct {
	model           string
	provider        string
	verbose         bool
	dangerouslySkip bool
	noAutoApprove   bool
	file            string
	version         bool
}

// Execute is the main entry point called from main.go.
func Execute() {
	// Parse minimal flags manually (avoids cobra/pflag dependency)
	args := os.Args[1:]
	var positional []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version", "-v":
			flags.version = true
		case "--verbose":
			flags.verbose = true
		case "--dangerously-skip-permissions":
			flags.dangerouslySkip = true
		case "--no-auto-approve":
			flags.noAutoApprove = true
		case "--model", "-m":
			if i+1 < len(args) {
				i++
				flags.model = args[i]
			}
		case "--provider", "-p":
			if i+1 < len(args) {
				i++
				flags.provider = args[i]
			}
		case "--file", "-f":
			if i+1 < len(args) {
				i++
				flags.file = args[i]
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				positional = append(positional, args[i])
			}
		}
	}

	if flags.version {
		fmt.Println(version.String())
		os.Exit(0)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI overrides
	if flags.model != "" {
		cfg.Model = flags.model
	}
	if flags.provider != "" {
		cfg.Provider = flags.provider
	}
	if flags.dangerouslySkip {
		cfg.DangerouslySkip = true
	}
	if flags.noAutoApprove {
		cfg.AutoApprove = false
	}
	cfg.Verbose = flags.verbose

	// Init logger
	logger.Init(flags.verbose)

	// Build provider
	provider, err := buildProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m%v\033[0m\n", err)
		fmt.Fprintf(os.Stderr, "请设置环境变量，例如: export ANTHROPIC_API_KEY=sk-ant-...\n")
		os.Exit(1)
	}

	// Determine run mode
	isPipe := isPipeInput()
	prompt := strings.Join(positional, " ")

	if isPipe {
		// Pipe mode: read stdin and prepend to prompt
		stdinData, _ := io.ReadAll(os.Stdin)
		if len(stdinData) > 0 {
			if prompt == "" {
				prompt = string(stdinData)
			} else {
				prompt = string(stdinData) + "\n\n" + prompt
			}
		}
	}

	// Load file attachment if given
	if flags.file != "" {
		data, err := os.ReadFile(flags.file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read file %s: %v\n", flags.file, err)
			os.Exit(1)
		}
		prompt = fmt.Sprintf("文件内容 (%s):\n```\n%s\n```\n\n%s", flags.file, string(data), prompt)
	}

	a := agent.New(cfg, provider)

	if isPipe || prompt != "" {
		// One-shot / pipe mode
		runOneShot(a, prompt)
	} else {
		// Interactive REPL mode
		ui.PrintBanner(version.Version, cfg.Model)
		runInteractive(a, cfg)
	}
}

func runOneShot(a *agent.Agent, prompt string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	if err := a.Run(ctx, prompt); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}
}

func runInteractive(a *agent.Agent, cfg *config.Config) {
	slashHandler := slash.NewHandler(a.Session(), cfg)

	// Setup readline with tab completion
	completer := readline.NewPrefixCompleter()
	for _, cmd := range slash.AllCommands() {
		completer.Children = append(completer.Children, readline.PcItem(cmd.Name))
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[1;34m> \033[0m",
		HistoryFile:     getHistoryFile(),
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		// Fallback to basic reader if readline fails
		runInteractiveBasic(a, cfg)
		return
	}
	defer rl.Close()

	// Setup signal handling for Ctrl+C (interrupt current task, not exit)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	var cancelCurrent context.CancelFunc

	go func() {
		for range sigCh {
			if cancelCurrent != nil {
				fmt.Println("\n\033[33m[任务已中断，输入新的指令继续]\033[0m")
				cancelCurrent()
				cancelCurrent = nil
			}
		}
	}()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if cancelCurrent != nil {
					cancelCurrent()
					cancelCurrent = nil
				}
				continue
			} else if err == io.EOF {
				fmt.Println("\n再见！")
				break
			}
			continue
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			handled, shouldExit := slashHandler.Handle(input)
			if shouldExit {
				fmt.Println("再见！")
				return
			}
			if handled {
				continue
			}
		}

		// Run agent
		ctx, cancel := context.WithCancel(context.Background())
		cancelCurrent = cancel

		ui.PrintDivider()
		if err := a.Run(ctx, input); err != nil && ctx.Err() == nil {
			ui.PrintError(err.Error())
		}
		cancel()
		cancelCurrent = nil
		ui.PrintDivider()
	}
}

func isPipeInput() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func buildProvider(cfg *config.Config) (llm.Provider, error) {
	apiKey := config.APIKey(cfg.Provider)

	switch cfg.Provider {
	case "anthropic":
		if apiKey == "" {
			return nil, fmt.Errorf("未找到 Anthropic API Key，请设置 ANTHROPIC_API_KEY 环境变量")
		}
		return anthropicprovider.New(apiKey, cfg.BaseURL, cfg.Model), nil
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("未找到 OpenAI API Key，请设置 OPENAI_API_KEY 环境变量")
		}
		return openaiprovider.New(apiKey, cfg.BaseURL, cfg.Model), nil
	default:
		return nil, fmt.Errorf("不支持的 provider: %s (支持: anthropic, openai)", cfg.Provider)
	}
}

// getHistoryFile returns the path to the readline history file.
func getHistoryFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	historyDir := fmt.Sprintf("%s/.aicoder", home)
	os.MkdirAll(historyDir, 0755)
	return fmt.Sprintf("%s/history", historyDir)
}

// runInteractiveBasic is a fallback interactive mode without readline support.
func runInteractiveBasic(a *agent.Agent, cfg *config.Config) {
	slashHandler := slash.NewHandler(a.Session(), cfg)
	reader := bufio.NewReader(os.Stdin)

	// Setup signal handling for Ctrl+C (interrupt current task, not exit)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	var cancelCurrent context.CancelFunc

	go func() {
		for range sigCh {
			if cancelCurrent != nil {
				fmt.Println("\n\033[33m[任务已中断，输入新的指令继续]\033[0m")
				cancelCurrent()
				cancelCurrent = nil
			}
		}
	}()

	for {
		fmt.Print("\033[1;34m> \033[0m")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\n再见！")
				break
			}
			continue
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			handled, shouldExit := slashHandler.Handle(input)
			if shouldExit {
				fmt.Println("再见！")
				return
			}
			if handled {
				continue
			}
		}

		// Run agent
		ctx, cancel := context.WithCancel(context.Background())
		cancelCurrent = cancel

		ui.PrintDivider()
		if err := a.Run(ctx, input); err != nil && ctx.Err() == nil {
			ui.PrintError(err.Error())
		}
		cancel()
		cancelCurrent = nil
		ui.PrintDivider()
	}
}
