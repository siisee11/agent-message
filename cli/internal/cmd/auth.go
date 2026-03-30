package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newRegisterCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "register <username> <pin>",
		Short: "Register a new account",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runRegister(rt, args[0], args[1])
		},
	}
}

func runRegister(rt *Runtime, username, pin string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	resp, err := rt.Client.Register(context.Background(), username, pin)
	if err != nil {
		return err
	}

	rt.Config.Token = strings.TrimSpace(resp.Token)
	rt.Config.ServerURL = rt.Client.ServerURL()
	rt.Client.SetToken(rt.Config.Token)

	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	_, _ = fmt.Fprintf(rt.Stdout, "registered %s\n", resp.User.Username)
	return nil
}

func newLoginCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "login <username> <pin>",
		Short: "Log in with username and PIN",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runLogin(rt, args[0], args[1])
		},
	}
}

func runLogin(rt *Runtime, username, pin string) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	resp, err := rt.Client.Login(context.Background(), username, pin)
	if err != nil {
		return err
	}

	rt.Config.Token = strings.TrimSpace(resp.Token)
	rt.Config.ServerURL = rt.Client.ServerURL()
	rt.Client.SetToken(rt.Config.Token)

	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	_, _ = fmt.Fprintf(rt.Stdout, "logged in as %s\n", resp.User.Username)
	return nil
}

func newLogoutCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear local token",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runLogout(rt)
		},
	}
}

func newWhoAmICommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the currently authenticated user",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWhoAmI(rt)
		},
	}
}

func runWhoAmI(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := ensureLoggedIn(rt); err != nil {
		return err
	}

	user, err := rt.Client.Me(context.Background())
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(rt.Stdout, user.Username)
	return nil
}

func runLogout(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	var remoteErr error
	if strings.TrimSpace(rt.Config.Token) != "" {
		remoteErr = rt.Client.Logout(context.Background())
	}

	rt.Config.Token = ""
	rt.Client.SetToken("")
	rt.Config.ServerURL = rt.Client.ServerURL()
	if err := rt.ConfigStore.Save(rt.Config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	if remoteErr != nil {
		_, _ = fmt.Fprintf(rt.Stderr, "warning: server logout failed: %v\n", remoteErr)
	}
	_, _ = fmt.Fprintln(rt.Stdout, "logged out")
	return nil
}

func ensureRuntime(rt *Runtime) error {
	switch {
	case rt == nil:
		return errors.New("runtime is not initialized")
	case rt.ConfigStore == nil:
		return errors.New("config store is not initialized")
	case rt.Client == nil:
		return errors.New("api client is not initialized")
	case rt.Stdout == nil:
		return errors.New("stdout writer is not initialized")
	case rt.Stderr == nil:
		return errors.New("stderr writer is not initialized")
	default:
		return nil
	}
}
