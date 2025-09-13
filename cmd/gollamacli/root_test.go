package gollamacli

import (
    "bytes"
    "os"
    "strings"
    "testing"

    "github.com/spf13/cobra"
)

func TestRoot_SubcommandsPresent(t *testing.T) {
    have := map[string]bool{}
    for _, c := range rootCmd.Commands() {
        have[c.Name()] = true
        if c.Name() == "list" {
            // list should have subcommands 'models' and 'commands'
            sub := map[string]bool{}
            for _, sc := range c.Commands() {
                sub[sc.Name()] = true
            }
            if !sub["models"] || !sub["commands"] {
                t.Fatalf("list subcommands missing: %v", sub)
            }
        }
        if c.Name() == "pull" || c.Name() == "delete" || c.Name() == "sync" || c.Name() == "unload" {
            sub := map[string]bool{}
            for _, sc := range c.Commands() {
                sub[sc.Name()] = true
            }
            if !sub["models"] {
                t.Fatalf("%s must have models subcommand", c.Name())
            }
        }
    }
    for _, want := range []string{"chat", "list", "pull", "delete", "sync", "unload"} {
        if !have[want] {
            t.Fatalf("missing subcommand %s", want)
        }
    }
}

func TestCommands_HaveDescriptions(t *testing.T) {
    var check func(*cobra.Command)
    check = func(cmd *cobra.Command) {
        if cmd.Short == "" || cmd.Long == "" {
            t.Fatalf("command %s missing Short/Long", cmd.Name())
        }
        for _, sc := range cmd.Commands() {
            check(sc)
        }
    }
    check(rootCmd)
}

func TestListCommands_PrintsTree(t *testing.T) {
    old := os.Stdout
    defer func() { os.Stdout = old }()
    var buf bytes.Buffer
    os.Stdout = &buf

    listAllCommands(rootCmd)
    out := buf.String()
    if !strings.Contains(out, "gollamacli chat") {
        t.Fatalf("expected command path in output, got: %s", out)
    }
}
