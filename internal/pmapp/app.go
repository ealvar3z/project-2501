package pmapp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	VersionBase = "Puppet Master browser v0.1-dev"
)

const (
	debug       = false
	sandboxMode = "not sandboxed"
	pollMode    = "unknown poll"
)

var PMVersionStr = func() string {
	s := VersionBase + " ("
	if debug {
		s += "debug"
	} else {
		s += "release"
	}
	s += ", "
	s += sandboxMode
	s += ", "
	s += pollMode
	s += ")\n"
	return s
}()

var PMVersionStrLong = func() string {
	s := VersionBase + " ("
	if debug {
		s += "debug"
	} else {
		s += "release"
	}
	s += ", "
	s += sandboxMode
	s += ", poll uses "
	s += pollMode
	s += ")\n"
	return s
}()

var errNotImplemented = errors.New("not-implemented")

func die(s string) {
	fmt.Fprintln(os.Stderr, "pm: "+s)
	os.Exit(1)
}

func help(i int) {
	s := PMVersionStr + `
Usage: pm [options] [URL(s) or file(s)...]
Options:
    --                         Interpret all following arguments as URLs
    -c, --css <stylesheet>     Pass stylesheet (e.g. -c 'a {color: blue}')
    -d, --dump                 Print page to stdout
    -h, --help                 Print this usage message
    -o, --opt <config>         Pass config options (e.g. -o buffer.images=true)
    -r, --run <script/file>    Run passed script or file
    -v, --version              Print version information
    -C, --config <file>        Override config path
    -I, --input-charset <enc>  Specify document charset
    -M, --monochrome           Set color-mode to 'monochrome'
    -O, --output-charset <enc> Specify display charset
    -T, --type <type>          Specify content mime type
    -V, --visual               Visual startup mode
`
	if i == 0 {
		_, _ = os.Stdout.WriteString(s)
	} else {
		_, _ = os.Stderr.WriteString(s)
	}
	os.Exit(i)
}

func version() {
	_, _ = os.Stdout.WriteString(PMVersionStrLong)
	os.Exit(0)
}

type Params struct {
	ConfigPath    string
	ContentType   string
	InputCharset  string
	OutputCharset string
	Visual        bool
	Monochrome    bool
	Dump          bool
	RunScript     string
	Opts          []string
	Stylesheet    string
	Pages         []string
}

type Config struct {
	Dir        string
	DataDir    string
	TmpDir     string
	VisualHome string
	Headless   string
	Warnings   []string
	Viper      *viper.Viper
}

type Engine interface {
	Run(Params, Config) int
}

type Runtime struct{}

func (Runtime) Free() {}

type JSContext struct{}

func (JSContext) Free() {}

type ForkServer struct{}

type FileLoader struct{}

type Client struct{}

type stubEngine struct{}

func (stubEngine) Run(params Params, _ Config) int {
	if params.RunScript == "quit()" && len(params.Pages) == 0 {
		return 0
	}
	return 2
}

func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	return Run(args, stdin, stdout, stderr, stubEngine{})
}

func notImplemented(name string) error {
	return fmt.Errorf("%w: %s", errNotImplemented, name)
}

func initAtomFactory() error {
	return nil
}

func setupProcessEnv() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	binDir := filepath.Dir(bin)
	if err := os.Setenv("PM_BIN_DIR", binDir); err != nil {
		return err
	}
	return os.Setenv("PM_LIBEXEC_DIR", filepath.Clean(filepath.Join(binDir, "..", "libexec", "pm")))
}

func newGlobalJSRuntime() (Runtime, error) {
	return Runtime{}, nil
}

func (Runtime) newJSContext() (JSContext, error) {
	return JSContext{}, nil
}

func openURandom() (*os.File, error) {
	return os.Open("/dev/urandom")
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, engine Engine) error {
	return pmMain(args, stdin, stdout, stderr, engine)
}

func pmMain(args []string, stdin io.Reader, stdout, stderr io.Writer, engine Engine) error {
	if err := initAtomFactory(); err != nil {
		return err
	}
	if err := setupProcessEnv(); err != nil {
		return err
	}
	rt, err := newGlobalJSRuntime()
	if err != nil {
		return err
	}
	defer rt.Free()
	return main2(rt, args, stdin, stdout, stderr, engine)
}

func main2(rt Runtime, args []string, stdin io.Reader, stdout, stderr io.Writer, engine Engine) error {
	jsctx, err := rt.newJSContext()
	if err != nil {
		return err
	}
	defer jsctx.Free()
	urandom, err := openURandom()
	if err != nil {
		return err
	}
	defer urandom.Close()
	params, config, history, handled, err := initializeStartup(args, stdin, stdout, stderr)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}
	if engine == nil {
		return notImplemented("engine")
	}
	_ = jsctx
	_ = history
	code := engine.Run(params, config)
	if code != 0 {
		return notImplemented("browser engine")
	}
	return nil
}

func initializeStartup(args []string, stdin io.Reader, stdout, stderr io.Writer) (Params, Config, bool, bool, error) {
	params, showHelp, showVersion, err := parseArgs(args)
	if err != nil {
		return Params{}, Config{}, false, false, err
	}
	if showHelp {
		_, err := io.WriteString(stdout, usage())
		return Params{}, Config{}, false, true, err
	}
	if showVersion {
		_, err := fmt.Fprintln(stdout, versionLong())
		return Params{}, Config{}, false, true, err
	}
	config, err := initConfig(params)
	if err != nil {
		return Params{}, Config{}, false, false, err
	}
	history := true
	if len(params.Pages) == 0 && isTerminal(stdin) {
		addDefaultPages(&params, config, &history)
	}
	if len(params.Pages) == 0 && config.Headless != "true" && !params.Dump && isTerminal(stdin) {
		_, _ = io.WriteString(stderr, usage())
		return Params{}, Config{}, false, false, errors.New("missing URL or file")
	}
	if err := ensureTmpDir(config.TmpDir); err != nil {
		return Params{}, Config{}, false, false, err
	}
	return params, config, history, false, nil
}

func parseArgs(args []string) (Params, bool, bool, error) {
	var params Params
	var showHelp bool
	var showVersion bool
	cmd := &cobra.Command{
		Use:                "pm [options] [URL(s) or file(s)...]",
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, raw []string) error {
			p, help, version, err := parsePMArgs(raw)
			params = p
			showHelp = help
			showVersion = version
			return err
		},
	}
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return Params{}, false, false, err
	}
	return params, showHelp, showVersion, nil
}

func parsePMArgs(args []string) (Params, bool, bool, error) {
	var params Params
	escapeAll := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if escapeAll {
			params.Pages = append(params.Pages, arg)
			continue
		}
		if arg == "" {
			continue
		}
		if arg == "--" {
			escapeAll = true
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			params.Pages = append(params.Pages, arg)
			continue
		}
		if arg == "-" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			value := func(name string) (string, error) {
				if i+1 >= len(args) {
					return "", fmt.Errorf("%s requires an argument", name)
				}
				i++
				return args[i], nil
			}
			switch arg {
			case "--config":
				v, err := value(arg)
				if err != nil {
					return Params{}, false, false, err
				}
				if v == "" {
					return Params{}, false, false, errors.New("config override cannot be empty")
				}
				params.ConfigPath = v
			case "--input-charset":
				v, err := value(arg)
				if err != nil {
					return Params{}, false, false, err
				}
				params.InputCharset = v
			case "--monochrome":
				params.Monochrome = true
				params.Opts = append(params.Opts, "display.color-mode = monochrome")
			case "--output-charset":
				v, err := value(arg)
				if err != nil {
					return Params{}, false, false, err
				}
				params.OutputCharset = v
				params.Opts = append(params.Opts, "encoding.display-charset = '"+v+"'")
			case "--type":
				v, err := value(arg)
				if err != nil {
					return Params{}, false, false, err
				}
				params.ContentType = v
			case "--visual":
				params.Visual = true
			case "--css":
				v, err := value(arg)
				if err != nil {
					return Params{}, false, false, err
				}
				params.Stylesheet += v
			case "--dump":
				params.Dump = true
				params.Opts = append(params.Opts, "start.headless = 'dump'")
			case "--help":
				return params, true, false, nil
			case "--opt":
				v, err := value(arg)
				if err != nil {
					return Params{}, false, false, err
				}
				params.Opts = append(params.Opts, v)
			case "--run":
				v, err := value(arg)
				if err != nil {
					return Params{}, false, false, err
				}
				params.RunScript = v
				params.Opts = append(params.Opts, "start.startup-script = "+quoteTriple(v), "start.headless = true")
			case "--version":
				return params, false, true, nil
			default:
				return Params{}, false, false, fmt.Errorf("unknown option %s", arg)
			}
			continue
		}
		for j := 1; j < len(arg); j++ {
			flag := arg[j]
			needsValue := strings.ContainsRune("CIOTcor", rune(flag))
			next := ""
			if needsValue {
				if j+1 < len(arg) {
					next = arg[j+1:]
					j = len(arg)
				} else {
					if i+1 >= len(args) {
						return Params{}, false, false, fmt.Errorf("-%c requires an argument", flag)
					}
					i++
					next = args[i]
				}
			}
			switch flag {
			case 'C':
				if next == "" {
					return Params{}, false, false, errors.New("config override cannot be empty")
				}
				params.ConfigPath = next
			case 'I':
				params.InputCharset = next
			case 'M':
				params.Monochrome = true
				params.Opts = append(params.Opts, "display.color-mode = monochrome")
			case 'O':
				params.OutputCharset = next
				params.Opts = append(params.Opts, "encoding.display-charset = '"+next+"'")
			case 'T':
				params.ContentType = next
			case 'V':
				params.Visual = true
			case 'c':
				params.Stylesheet += next
			case 'd':
				params.Dump = true
				params.Opts = append(params.Opts, "start.headless = 'dump'")
			case 'h':
				return params, true, false, nil
			case 'o':
				params.Opts = append(params.Opts, next)
			case 'r':
				params.RunScript = next
				params.Opts = append(params.Opts, "start.startup-script = "+quoteTriple(next), "start.headless = true")
			case 'v':
				return params, false, true, nil
			default:
				return Params{}, false, false, fmt.Errorf("unknown option -%c", flag)
			}
		}
	}
	return params, false, false, nil
}

func initConfig(params Params) (Config, error) {
	configDir, dataDir, path, err := resolveConfigPath(params.ConfigPath)
	if err != nil {
		return Config{}, err
	}
	v := viper.New()
	v.SetConfigType("toml")
	setDefaults(v)
	var warnings []string
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("failed to read config file %s: %w", path, err)
		}
	} else if params.ConfigPath != "" {
		return Config{}, fmt.Errorf("failed to open config file %s", params.ConfigPath)
	}
	if err := os.Setenv("PM_DIR", configDir); err != nil {
		return Config{}, err
	}
	if err := os.Setenv("PM_DATA_DIR", dataDir); err != nil {
		return Config{}, err
	}
	if bin, err := os.Executable(); err == nil {
		binDir := filepath.Dir(bin)
		_ = os.Setenv("PM_BIN_DIR", binDir)
		_ = os.Setenv("PM_LIBEXEC_DIR", filepath.Clean(filepath.Join(binDir, "..", "libexec", "pm")))
	}
	headless := fmt.Sprint(v.Get("start.headless"))
	if params.Dump {
		headless = "dump"
	}
	if params.RunScript != "" {
		headless = "true"
	}
	return Config{
		Dir:        configDir,
		DataDir:    dataDir,
		TmpDir:     v.GetString("external.tmpdir"),
		VisualHome: v.GetString("start.visual-home"),
		Headless:   headless,
		Warnings:   warnings,
		Viper:      v,
	}, nil
}

func resolveConfigPath(override string) (configDir, dataDir, path string, err error) {
	home, _ := os.UserHomeDir()
	configDir = filepath.Join(home, ".pm")
	dataDir = configDir
	if override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", "", "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return filepath.Dir(abs), filepath.Dir(abs), "", nil
		}
		return filepath.Dir(abs), filepath.Dir(abs), abs, nil
	}
	if env := os.Getenv("PM_DIR"); env != "" {
		configDir = env
		dataDir = env
		candidate := filepath.Join(env, "config.toml")
		if fileExists(candidate) {
			return configDir, dataDir, candidate, nil
		}
		return configDir, dataDir, "", nil
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}
	candidate := filepath.Join(xdg, "pm", "config.toml")
	if fileExists(candidate) {
		return filepath.Dir(candidate), filepath.Dir(candidate), candidate, nil
	}
	candidate = filepath.Join(home, ".pm", "config.toml")
	if fileExists(candidate) {
		return filepath.Dir(candidate), filepath.Dir(candidate), candidate, nil
	}
	return configDir, dataDir, "", nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("start.visual-home", "about:pm")
	v.SetDefault("start.headless", false)
	v.SetDefault("external.tmpdir", filepath.Join(os.TempDir(), fmt.Sprintf("pm-%d", os.Getuid())))
}

func addDefaultPages(params *Params, config Config, history *bool) {
	if params.Visual {
		*history = false
		params.Pages = append(params.Pages, config.VisualHome)
		return
	}
	if home := os.Getenv("HTTP_HOME"); home != "" {
		*history = false
		params.Pages = append(params.Pages, home)
		return
	}
	if home := os.Getenv("WWW_HOME"); home != "" {
		*history = false
		params.Pages = append(params.Pages, home)
	}
}

func ensureTmpDir(path string) error {
	if path == "" {
		return errors.New("tmpdir cannot be empty")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("failed to set permissions of %s: %w", path, err)
	}
	return nil
}

func usage() string {
	return VersionBase + ` (development)
Usage: pm [options] [URL(s) or file(s)...]
Options:
    --                         Interpret all following arguments as URLs
    -c, --css <stylesheet>     Pass stylesheet (e.g. -c 'a {color: blue}')
    -d, --dump                 Print page to stdout
    -h, --help                 Print this usage message
    -o, --opt <config>         Pass config options (e.g. -o buffer.images=true)
    -r, --run <script/file>    Run passed script or file
    -v, --version              Print version information
    -C, --config <file>        Override config path
    -I, --input-charset <enc>  Specify document charset
    -M, --monochrome           Set color-mode to 'monochrome'
    -O, --output-charset <enc> Specify display charset
    -T, --type <type>          Specify content mime type
    -V, --visual               Visual startup mode
`
}

func versionLong() string {
	return VersionBase + " (development, not sandboxed)"
}

func quoteTriple(s string) string {
	return `"""` + strings.ReplaceAll(s, `"""`, `\"\"\"`) + `"""`
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func isTerminal(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
