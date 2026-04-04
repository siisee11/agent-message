package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"agent-message/cli/internal/api"
	"agent-message/cli/internal/config"

	"github.com/spf13/cobra"
)

func newRegisterCommand(rt *Runtime) *cobra.Command {
	var payload string
	var payloadFile string
	var payloadStdin bool
	cmd := &cobra.Command{
		Use:   "register <username> <password>",
		Short: "Register a new account",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			request, err := resolveAuthRequest(rt.Stdin, rawPayloadOptions{
				Payload:      payload,
				PayloadFile:  payloadFile,
				PayloadStdin: payloadStdin,
			}, args, "register")
			if err != nil {
				return err
			}
			return runRegisterRequest(rt, request)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "Raw JSON payload matching the register request body")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "Read the raw register JSON payload from a file")
	cmd.Flags().BoolVar(&payloadStdin, "payload-stdin", false, "Read the raw register JSON payload from stdin")
	return cmd
}

func newOnboardCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "onboard",
		Short: "Interactively log in or create an account, then set that account as master",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runOnboard(rt)
		},
	}
}

func runRegister(rt *Runtime, username, password string) error {
	return runRegisterRequest(rt, api.AuthRequest{
		Username: username,
		Password: password,
	})
}

func runRegisterRequest(rt *Runtime, request api.AuthRequest) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := rt.Client.SetServerURL(rt.Config.ServerURL); err != nil {
		return fmt.Errorf("set register server_url: %w", err)
	}

	resp, err := rt.Client.RegisterWithRequest(context.Background(), request)
	if err != nil {
		return err
	}

	if err := activateAuthenticatedProfile(rt, resp.User.Username, rt.Client.ServerURL(), resp.Token); err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("registered %s", resp.User.Username), map[string]any{
		"status": "registered",
		"user":   resp.User,
	})
}

func runOnboard(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := rt.Client.SetServerURL(rt.Config.ServerURL); err != nil {
		return fmt.Errorf("set onboard server_url: %w", err)
	}

	reader := bufio.NewReader(rt.Stdin)
	username, err := promptRequiredInput(reader, rt.Stdout, "username")
	if err != nil {
		return err
	}
	password, err := promptRequiredInput(reader, rt.Stdout, "password")
	if err != nil {
		return err
	}

	resp, err := loginOrRegister(rt.Client, username, password)
	if err != nil {
		return err
	}

	if err := activateAuthenticatedProfile(rt, resp.User.Username, rt.Client.ServerURL(), resp.Token); err != nil {
		return err
	}

	cfg := rt.Config
	cfg.Master = resp.User.Username
	if err := saveRuntimeConfig(rt, cfg); err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("onboarded %s", resp.User.Username), map[string]any{
		"status": "onboarded",
		"user":   resp.User,
		"master": rt.Config.Master,
	})
}

func newLoginCommand(rt *Runtime) *cobra.Command {
	var payload string
	var payloadFile string
	var payloadStdin bool
	cmd := &cobra.Command{
		Use:   "login <username> <password>",
		Short: "Log in with username and password",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("accepts at most 2 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			request, err := resolveAuthRequest(rt.Stdin, rawPayloadOptions{
				Payload:      payload,
				PayloadFile:  payloadFile,
				PayloadStdin: payloadStdin,
			}, args, "login")
			if err != nil {
				return err
			}
			return runLoginRequest(rt, request)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "Raw JSON payload matching the login request body")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "Read the raw login JSON payload from a file")
	cmd.Flags().BoolVar(&payloadStdin, "payload-stdin", false, "Read the raw login JSON payload from stdin")
	return cmd
}

func runLogin(rt *Runtime, username, password string) error {
	return runLoginRequest(rt, api.AuthRequest{
		Username: username,
		Password: password,
	})
}

func runLoginRequest(rt *Runtime, request api.AuthRequest) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}
	if err := rt.Client.SetServerURL(rt.Config.ServerURL); err != nil {
		return fmt.Errorf("set login server_url: %w", err)
	}

	resp, err := rt.Client.LoginWithRequest(context.Background(), request)
	if err != nil {
		return err
	}

	if err := activateAuthenticatedProfile(rt, resp.User.Username, rt.Client.ServerURL(), resp.Token); err != nil {
		return err
	}

	return writeTextOrJSON(rt, fmt.Sprintf("logged in as %s", resp.User.Username), map[string]any{
		"status": "logged_in",
		"user":   resp.User,
	})
}

func resolveAuthRequest(stdin io.Reader, payloadOptions rawPayloadOptions, args []string, action string) (api.AuthRequest, error) {
	rawPayload, err := resolveRawPayload(stdin, payloadOptions)
	if err != nil {
		return api.AuthRequest{}, err
	}
	if rawPayload != nil {
		if len(args) != 0 {
			return api.AuthRequest{}, fmt.Errorf("%s accepts no positional args when raw payload flags are used", action)
		}
		return decodeStrictJSONObject[api.AuthRequest](rawPayload, action+" payload")
	}
	if len(args) != 2 {
		return api.AuthRequest{}, fmt.Errorf("%s requires <username> <password>", action)
	}
	return api.AuthRequest{
		Username: args[0],
		Password: args[1],
	}, nil
}

func loginOrRegister(client *api.Client, username, password string) (api.AuthResponse, error) {
	resp, err := client.Login(context.Background(), username, password)
	if err == nil {
		return resp, nil
	}
	if !isAPIStatus(err, http.StatusUnauthorized) {
		return api.AuthResponse{}, err
	}

	resp, registerErr := client.Register(context.Background(), username, password)
	if registerErr != nil {
		if isAPIStatus(registerErr, http.StatusConflict) {
			return api.AuthResponse{}, err
		}
		return api.AuthResponse{}, registerErr
	}
	return resp, nil
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

	return writeTextOrJSON(rt, user.Username, map[string]any{
		"username": user.Username,
		"user":     user,
	})
}

func runLogout(rt *Runtime) error {
	if err := ensureRuntime(rt); err != nil {
		return err
	}

	var remoteErr error
	if strings.TrimSpace(rt.Config.Token) != "" {
		remoteErr = rt.Client.Logout(context.Background())
	}

	cfg := rt.Config
	cfg.Token = ""
	cfg.ActiveProfileServerURL = rt.Client.ServerURL()
	cfg.ReadSessions = make(map[string]config.ReadSession)
	cfg.LastReadConversationID = ""
	if err := saveRuntimeConfig(rt, cfg); err != nil {
		return err
	}

	if remoteErr != nil {
		_, _ = fmt.Fprintf(rt.Stderr, "warning: server logout failed: %v\n", remoteErr)
	}
	return writeTextOrJSON(rt, "logged out", map[string]any{
		"status": "logged_out",
	})
}

func ensureRuntime(rt *Runtime) error {
	switch {
	case rt == nil:
		return errors.New("runtime is not initialized")
	case rt.ConfigStore == nil:
		return errors.New("config store is not initialized")
	case rt.Client == nil:
		return errors.New("api client is not initialized")
	case rt.Stdin == nil:
		return errors.New("stdin reader is not initialized")
	case rt.Stdout == nil:
		return errors.New("stdout writer is not initialized")
	case rt.Stderr == nil:
		return errors.New("stderr writer is not initialized")
	default:
		return nil
	}
}

func activateAuthenticatedProfile(rt *Runtime, username, serverURL, token string) error {
	profileName := strings.TrimSpace(username)
	if profileName == "" {
		return errors.New("username is required")
	}

	cfg := rt.Config
	existingProfile := cfg.Profiles[profileName]
	cfg.ActiveProfile = profileName
	cfg.ActiveProfileServerURL = serverURL
	cfg.Token = strings.TrimSpace(token)
	cfg.Master = existingProfile.Master
	cfg.ReadSessions = cloneReadSessionsMap(existingProfile.ReadSessions)
	cfg.LastReadConversationID = existingProfile.LastReadConversationID

	if err := saveRuntimeConfig(rt, cfg); err != nil {
		return err
	}
	rt.Client.SetToken(rt.Config.Token)
	return nil
}

func isAPIStatus(err error, statusCode int) bool {
	var apiErr *api.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == statusCode
}

func promptRequiredInput(reader *bufio.Reader, stdout io.Writer, label string) (string, error) {
	trimmedLabel := strings.TrimSpace(label)
	if trimmedLabel == "" {
		return "", errors.New("prompt label is required")
	}

	for {
		if _, err := fmt.Fprintf(stdout, "%s: ", trimmedLabel); err != nil {
			return "", fmt.Errorf("write %s prompt: %w", trimmedLabel, err)
		}

		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read %s: %w", trimmedLabel, err)
		}

		value := strings.TrimSpace(line)
		if value != "" {
			return value, nil
		}
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%s is required", trimmedLabel)
		}
	}
}
