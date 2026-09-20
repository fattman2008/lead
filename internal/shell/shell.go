package shell

import (
	"fmt"
	"path/filepath"
)

// InitScript returns eval-able shell integration for the given shell name.
func InitScript(shellName string) string {
	switch shellName {
	case "zsh", "bash":
		return bashZshInit
	case "fish":
		return fishInit
	default:
		return bashZshInit
	}
}

const bashZshInit = `# lead (pt) shell integration — directory switching
pt() {
  local cd_file exit_code=0
  cd_file="$(mktemp)"
  LEAD_CD_FILE="$cd_file" command pt "$@" || exit_code=$?
  if [[ -s "$cd_file" ]]; then
    builtin cd -- "$(<"$cd_file")"
    local cd_exit=$?
    if [[ $exit_code -eq 0 ]]; then
      exit_code=$cd_exit
    fi
  fi
  command rm -f "$cd_file"
  return "$exit_code"
}
`

const fishInit = `# lead (pt) shell integration — directory switching
function pt
  set -l cd_file (mktemp)
  set -l exit_code 0
  env LEAD_CD_FILE="$cd_file" command pt $argv
  or set exit_code $status
  if test -s "$cd_file"
    cd (cat "$cd_file")
    set -l cd_exit $status
    if test $exit_code -eq 0
      set exit_code $cd_exit
    end
  end
  rm -f "$cd_file"
  return $exit_code
end
`

// DetectFromEnv returns a shell name hint from $SHELL basename.
func DetectFromEnv(shellPath string) string {
	if shellPath == "" {
		return "zsh"
	}
	base := shellPath
	for i := len(shellPath) - 1; i >= 0; i-- {
		if shellPath[i] == '/' {
			base = shellPath[i+1:]
			break
		}
	}
	switch base {
	case "bash", "zsh", "fish":
		return base
	default:
		return "zsh"
	}
}

// PrintInit writes the init script to stdout-style string with a trailing newline.
func PrintInit(shellName string) string {
	return InitScript(shellName) + "\n"
}

// InstallHint returns a one-liner for docs / shell rc files.
func InstallHint(shellName string) string {
	switch shellName {
	case "fish":
		return `pt shell init fish | source`
	default:
		if shellName == "" {
			shellName = "zsh"
		}
		return fmt.Sprintf(`eval "$(pt shell init %s)"`, shellName)
	}
}

// RCPath returns the usual startup file path for shellName under home.
func RCPath(home, shellName string) string {
	switch shellName {
	case "bash":
		return filepath.Join(home, ".bashrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return filepath.Join(home, ".zshrc")
	}
}
