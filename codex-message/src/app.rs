use anyhow::{Context, Result, anyhow};
use rand::Rng;
use serde_json::Map;
use serde_json::Value;
use serde_json::json;
use std::time::Duration;
use uuid::Uuid;

use crate::Config;
use crate::SandboxArg;
use crate::agent_message::{AgentMessageClient, Message, MessageWatch};
use crate::codex::{CodexAppServer, IncomingMessage};
use crate::log_ui::LogUi;
use crate::render::{ApprovalAction, approval_spec, report_spec};

fn request_suffix(to_username: &str, from_username: &str) -> String {
    format!(
        r#"

Operational requirements from the codex-message wrapper:
- Before composing the final user-facing result, run `agent-message catalog prompt` and use that output as the authoritative json-render catalog guidance.
- Use sender account `{from_username}` for direct outbound messages.
- Send the final user-facing result yourself by invoking the `agent-message` CLI.
- Deliver that result directly to the user `{to_username}`.
- Always pass `--from {from_username}` when invoking `agent-message send`.
- Prefer a visually readable `agent-message send {to_username} ... --kind json_render --from {from_username}` payload when appropriate.
- For final result messages, avoid wrapping the entire payload in a `Card` unless a card is clearly necessary; prefer a direct content-first layout such as `Stack`.
- Do not rely on the wrapper to forward your final assistant message as the primary user-facing result.
- After sending the direct result, keep your final assistant message minimal because the wrapper may still emit status metadata.
- If you need approval or clarification, ask clearly and briefly so the wrapper can relay it.
"#
    )
}
const READ_REACTION_EMOJI: &str = "👀";
const COMPLETE_REACTION_EMOJI: &str = "✅";
const WATCH_RETRY_DELAYS_SECS: [u64; 3] = [1, 2, 5];

fn resolve_hostname() -> String {
    for key in ["HOSTNAME", "HOST", "COMPUTERNAME"] {
        if let Ok(value) = std::env::var(key) {
            let trimmed = value.trim();
            if !trimmed.is_empty() {
                return trimmed.to_string();
            }
        }
    }

    std::process::Command::new("hostname")
        .output()
        .ok()
        .and_then(|output| String::from_utf8(output.stdout).ok())
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
        .unwrap_or_else(|| "unknown".to_string())
}

fn startup_text(
    chat_id: &str,
    username: &str,
    password: &str,
    cwd: &std::path::Path,
    hostname: &str,
) -> String {
    format!(
        "codex-message session started\nchat_id: {chat_id}\nusername: {username}\npassword: {password}\nCWD: {}\nHostname: {hostname}\n\nReply in this DM to run Codex.",
        cwd.display(),
    )
}

fn session_title(cwd: &std::path::Path, agent_name: &str) -> String {
    let project = cwd
        .file_name()
        .and_then(|value| value.to_str())
        .map(str::trim)
        .filter(|value| !value.is_empty());
    match project {
        Some(project) => format!("{agent_name} · {project}"),
        None => format!("{agent_name} session"),
    }
}

pub(crate) struct App {
    config: Config,
}

impl App {
    pub(crate) fn new(config: Config) -> Self {
        Self { config }
    }

    pub(crate) async fn run(self) -> Result<()> {
        let mut runtime = Runtime::bootstrap(self.config).await?;
        runtime.run_loop().await
    }
}

struct Runtime {
    config: Config,
    to_username: String,
    agent_client: AgentMessageClient,
    message_watch: MessageWatch,
    codex: CodexAppServer,
    thread_id: String,
    logger: LogUi,
}

impl Runtime {
    async fn bootstrap(config: Config) -> Result<Self> {
        let logger = LogUi::new("codex-message");
        let chat_id = new_chat_id();
        let username = format!("agent-{chat_id}");
        let password = new_password();
        let mut agent_client =
            AgentMessageClient::new(std::path::PathBuf::from("agent-message"), logger.clone())?;
        let to_username = resolve_target_username(&config, &agent_client).await?;
        let server_url = agent_client
            .server_url()
            .await
            .context("read agent-message server_url")?;

        logger.system(
            "Bootstrapping session",
            [
                format!("Target: @{to_username}"),
                format!("Chat: {chat_id}"),
                format!("Agent: {username}"),
                format!("CWD: {}", config.cwd.display()),
                format!("Server: {server_url}"),
            ],
        );

        register_agent_account(&mut agent_client, &username, &password).await?;
        agent_client.set_from_profile(username.clone());
        logger.success(
            "Agent profile registered",
            [format!("Profile: {username}"), format!("Chat: {chat_id}")],
        );

        let message_watch = agent_client
            .watch_messages(&to_username)
            .await
            .context("start agent-message watch stream")?;
        logger.system(
            "Message watch stream ready",
            [format!("Watching replies from @{to_username}")],
        );

        let hostname = resolve_hostname();
        let startup_text = startup_text(&chat_id, &username, &password, &config.cwd, &hostname);
        let startup_message_id = agent_client
            .send_text_message(&to_username, &startup_text)
            .await
            .context("send startup message")?;
        let conversation_title = session_title(&config.cwd, "Codex");
        agent_client
            .set_conversation_title(&to_username, &conversation_title)
            .await
            .context("set conversation title")?;
        logger.success(
            "Startup message sent",
            [
                format!("Recipient: @{to_username}"),
                format!("Message: {startup_message_id}"),
                format!("Title: {conversation_title}"),
            ],
        );

        let codex = CodexAppServer::start(&config.codex_bin, &config.cwd, logger.clone())
            .await
            .context("start codex app-server")?;
        codex
            .initialize()
            .await
            .context("initialize codex app-server")?;
        let thread_id = start_thread(&codex, &config).await?;
        logger.success("Codex app-server ready", [format!("Thread: {thread_id}")]);

        Ok(Self {
            config,
            to_username,
            agent_client,
            message_watch,
            codex,
            thread_id,
            logger,
        })
    }

    async fn run_loop(&mut self) -> Result<()> {
        loop {
            tokio::select! {
                _ = tokio::signal::ctrl_c() => {
                    self.message_watch.shutdown().await?;
                    self.codex.shutdown().await?;
                    return Ok(());
                }
                next = self.next_target_message() => {
                    let message = next?;
                    let Some(request) = extract_request_text(&message) else {
                        continue;
                    };

                    self.logger.request(
                        "User request received",
                        [
                            format!("Message: {}", message.id),
                            format!("From: @{}", message.sender_username),
                            format!("Text: {}", request_preview(&request)),
                        ],
                    );

                    match self.run_turn(&request).await {
                        Ok(outcome) => {
                            let mut lines = vec![
                                format!("Turn: {}", outcome.turn_id),
                                format!("Status: {}", outcome.status),
                                format!("Request: {}", request_preview(&request)),
                            ];
                            if let Some(error_text) = outcome.error_text.as_deref() {
                                lines.push(format!("Error: {}", request_preview(error_text)));
                                self.logger.error("Turn finished with error", lines);
                                self.send_turn_failure_notice(&request, error_text).await;
                            } else if !outcome.status.eq_ignore_ascii_case("completed") {
                                self.logger.warning("Turn finished", lines.clone());
                                self.send_turn_failure_notice(
                                    &request,
                                    &format!("Turn ended with status {}.", outcome.status),
                                )
                                .await;
                            } else {
                                self.logger.success("Turn finished", lines);
                            }
                            if should_mark_message_complete(&outcome) {
                                self.mark_message_complete(&message).await;
                            }
                        }
                        Err(error) => {
                            self.logger.error(
                                "Codex request failed",
                                [
                                    format!("Request: {}", request_preview(&request)),
                                    format!("Error: {error:#}"),
                                ],
                            );
                            self.send_turn_failure_notice(&request, &format!("{error:#}"))
                                .await;
                        }
                    }
                }
            }
        }
    }

    async fn next_target_message(&mut self) -> Result<Message> {
        let mut watch_retry_attempt = 0usize;
        loop {
            let message = match self.message_watch.next_message().await {
                Ok(message) => {
                    watch_retry_attempt = 0;
                    message
                }
                Err(error) => {
                    self.reconnect_message_watch(watch_retry_attempt, &error)
                        .await?;
                    watch_retry_attempt = watch_retry_attempt.saturating_add(1);
                    continue;
                }
            };
            if !message
                .sender_username
                .eq_ignore_ascii_case(&self.to_username)
            {
                continue;
            }
            if extract_request_text(&message).is_none() {
                continue;
            }
            self.logger.recv(
                "Message received",
                [
                    format!("Message: {}", message.id),
                    format!("From: @{}", message.sender_username),
                    format!("To: {}", self.agent_client.sender_label()),
                    format!("Kind: {}", message.kind),
                    format!("Text: {}", request_preview(&message.text)),
                ],
            );
            if let Err(error) = self
                .agent_client
                .react_to_message(&message, READ_REACTION_EMOJI)
                .await
            {
                self.logger.warning(
                    "Failed to add read reaction",
                    [
                        format!("Message: {}", message.id),
                        format!("Error: {error}"),
                    ],
                );
            }
            return Ok(message);
        }
    }

    async fn reconnect_message_watch(
        &mut self,
        attempt: usize,
        error: &anyhow::Error,
    ) -> Result<()> {
        let delay = watch_retry_delay(attempt);
        self.logger.warning(
            "Watch stream disconnected",
            [
                format!("Retry in: {}s", delay.as_secs()),
                format!("Error: {error:#}"),
            ],
        );

        self.message_watch
            .shutdown()
            .await
            .context("shutdown stale agent-message watch stream")?;
        tokio::time::sleep(delay).await;

        self.message_watch = self
            .agent_client
            .watch_messages(&self.to_username)
            .await
            .context("restart agent-message watch stream")?;
        self.logger.success(
            "Watch stream reconnected",
            [format!("User: @{}", self.to_username)],
        );
        Ok(())
    }

    async fn run_turn(&mut self, request: &str) -> Result<TurnOutcome> {
        let composed_request = format!(
            "{}\n{}",
            request.trim(),
            request_suffix(&self.to_username, &self.agent_client.sender_label())
        );
        let turn_params =
            build_turn_start_params(&self.config, &self.thread_id, &composed_request)?;
        let response = self
            .codex
            .request("turn/start", turn_params)
            .await
            .context("start codex turn")?;
        let turn_id = response
            .get("turn")
            .and_then(|turn| turn.get("id"))
            .and_then(Value::as_str)
            .map(ToOwned::to_owned)
            .ok_or_else(|| anyhow!("turn/start response missing turn.id"))?;

        self.logger.turn(
            "Turn started",
            [
                format!("Turn: {turn_id}"),
                format!("Request: {}", request_preview(request)),
            ],
        );

        let (turn_status, turn_error) = loop {
            match self.codex.next_event().await? {
                IncomingMessage::Notification { method, params } => match method.as_str() {
                    "item/agentMessage/delta" => {}
                    "turn/completed" => {
                        if params
                            .get("turn")
                            .and_then(|turn| turn.get("id"))
                            .and_then(Value::as_str)
                            != Some(turn_id.as_str())
                        {
                            continue;
                        }

                        let status = params
                            .get("turn")
                            .and_then(|turn| turn.get("status"))
                            .and_then(Value::as_str)
                            .map(ToOwned::to_owned)
                            .unwrap_or_else(|| "unknown".to_string());
                        let error = params
                            .get("turn")
                            .and_then(|turn| turn.get("error"))
                            .and_then(|error| error.get("message"))
                            .and_then(Value::as_str)
                            .map(ToOwned::to_owned);
                        break (status, error);
                    }
                    _ => {}
                },
                IncomingMessage::Request { method, id, params } => {
                    self.handle_server_request(method.as_str(), id, params)
                        .await?;
                }
            }
        };

        Ok(TurnOutcome {
            turn_id,
            status: turn_status,
            error_text: turn_error,
        })
    }

    async fn mark_message_complete(&self, message: &Message) {
        if let Err(error) = self
            .agent_client
            .unreact_to_message(message, READ_REACTION_EMOJI)
            .await
        {
            self.logger.warning(
                "Failed to remove read reaction",
                [
                    format!("Message: {}", message.id),
                    format!("Error: {error}"),
                ],
            );
        }
        if let Err(error) = self
            .agent_client
            .react_to_message(message, COMPLETE_REACTION_EMOJI)
            .await
        {
            self.logger.warning(
                "Failed to add complete reaction",
                [
                    format!("Message: {}", message.id),
                    format!("Error: {error}"),
                ],
            );
        }
    }

    async fn handle_server_request(
        &mut self,
        method: &str,
        id: Value,
        params: Value,
    ) -> Result<()> {
        match method {
            "item/commandExecution/requestApproval" => {
                let details = summarize_command_approval(&params);
                self.logger
                    .turn("Command approval requested", details.clone());
                let spec = approval_spec(
                    "Approval Needed",
                    "Command approval requested",
                    &details,
                    "approve | session | deny | cancel",
                    &[
                        ApprovalAction {
                            label: "Approve",
                            value: "approve",
                            variant: "primary",
                        },
                        ApprovalAction {
                            label: "This session",
                            value: "session",
                            variant: "secondary",
                        },
                        ApprovalAction {
                            label: "Deny",
                            value: "deny",
                            variant: "destructive",
                        },
                        ApprovalAction {
                            label: "Cancel",
                            value: "cancel",
                            variant: "secondary",
                        },
                    ],
                );
                self.agent_client
                    .send_json_render_message(&self.to_username, spec)
                    .await
                    .context("send command approval request")?;

                let decision = loop {
                    let reply = self.next_target_message().await?;
                    let Some(text) = extract_request_text(&reply) else {
                        continue;
                    };
                    if let Some(decision) = parse_command_decision(&text) {
                        break decision;
                    }
                    self.agent_client
                        .send_text_message(
                            &self.to_username,
                            "Reply with one of: approve, session, deny, cancel.",
                        )
                        .await
                        .context("send command approval clarification")?;
                };

                self.codex
                    .respond(id, json!({ "decision": decision }))
                    .await
                    .context("respond to command approval")?;
            }
            "item/fileChange/requestApproval" => {
                let details = summarize_file_approval(&params);
                self.logger
                    .turn("File change approval requested", details.clone());
                let spec = approval_spec(
                    "Approval Needed",
                    "File change approval requested",
                    &details,
                    "approve | session | deny | cancel",
                    &[
                        ApprovalAction {
                            label: "Approve",
                            value: "approve",
                            variant: "primary",
                        },
                        ApprovalAction {
                            label: "This session",
                            value: "session",
                            variant: "secondary",
                        },
                        ApprovalAction {
                            label: "Deny",
                            value: "deny",
                            variant: "destructive",
                        },
                        ApprovalAction {
                            label: "Cancel",
                            value: "cancel",
                            variant: "secondary",
                        },
                    ],
                );
                self.agent_client
                    .send_json_render_message(&self.to_username, spec)
                    .await
                    .context("send file approval request")?;

                let decision = loop {
                    let reply = self.next_target_message().await?;
                    let Some(text) = extract_request_text(&reply) else {
                        continue;
                    };
                    if let Some(decision) = parse_file_decision(&text) {
                        break decision;
                    }
                    self.agent_client
                        .send_text_message(
                            &self.to_username,
                            "Reply with one of: approve, session, deny, cancel.",
                        )
                        .await
                        .context("send file approval clarification")?;
                };

                self.codex
                    .respond(id, json!({ "decision": decision }))
                    .await
                    .context("respond to file approval")?;
            }
            "item/permissions/requestApproval" => {
                let details = summarize_permissions_request(&params);
                self.logger
                    .turn("Additional permissions requested", details.clone());
                let requested_permissions = params
                    .get("permissions")
                    .cloned()
                    .unwrap_or_else(|| json!({}));
                let spec = approval_spec(
                    "Permission Needed",
                    "Additional permissions requested",
                    &details,
                    "allow | allow session | deny",
                    &[
                        ApprovalAction {
                            label: "Allow",
                            value: "allow",
                            variant: "primary",
                        },
                        ApprovalAction {
                            label: "Allow session",
                            value: "allow session",
                            variant: "secondary",
                        },
                        ApprovalAction {
                            label: "Deny",
                            value: "deny",
                            variant: "destructive",
                        },
                    ],
                );
                self.agent_client
                    .send_json_render_message(&self.to_username, spec)
                    .await
                    .context("send permissions request")?;

                let response = loop {
                    let reply = self.next_target_message().await?;
                    let Some(text) = extract_request_text(&reply) else {
                        continue;
                    };
                    if let Some(response) = parse_permission_response(&text, &requested_permissions)
                    {
                        break response;
                    }
                    self.agent_client
                        .send_text_message(
                            &self.to_username,
                            "Reply with one of: allow, allow session, deny.",
                        )
                        .await
                        .context("send permissions clarification")?;
                };

                self.codex
                    .respond(id, response)
                    .await
                    .context("respond to permissions request")?;
            }
            "item/tool/requestUserInput" => {
                let questions = params
                    .get("questions")
                    .and_then(Value::as_array)
                    .cloned()
                    .unwrap_or_default();
                let details = summarize_tool_questions(&questions);
                self.logger
                    .turn("Codex requested user input", details.clone());
                let mut prompt_lines = details.clone();
                prompt_lines.push(
                    "Reply: JSON object keyed by question id, or plain text if there is only one question"
                        .to_string(),
                );
                let spec = report_spec(
                    "Input Needed",
                    "Codex requested user input",
                    &prompt_lines,
                    None,
                );
                self.agent_client
                    .send_json_render_message(&self.to_username, spec)
                    .await
                    .context("send request_user_input prompt")?;

                let response = loop {
                    let reply = self.next_target_message().await?;
                    let Some(text) = extract_request_text(&reply) else {
                        continue;
                    };
                    if let Some(response) = parse_tool_user_input_response(&text, &questions) {
                        break response;
                    }
                    self.agent_client
                        .send_text_message(
                            &self.to_username,
                            "Reply with JSON like {\"question_id\":\"answer\"}. For a single question, plain text also works.",
                        )
                        .await
                        .context("send request_user_input clarification")?;
                };

                self.codex
                    .respond(id, response)
                    .await
                    .context("respond to request_user_input")?;
            }
            "mcpServer/elicitation/request" => {
                let details = summarize_mcp_elicitation(&params);
                self.logger
                    .turn("MCP elicitation requested", details.clone());
                let spec = approval_spec(
                    "MCP Input",
                    "An MCP server requested interaction",
                    &details,
                    "accept | decline | cancel, optionally followed by JSON content",
                    &[
                        ApprovalAction {
                            label: "Accept",
                            value: "accept",
                            variant: "primary",
                        },
                        ApprovalAction {
                            label: "Decline",
                            value: "decline",
                            variant: "destructive",
                        },
                        ApprovalAction {
                            label: "Cancel",
                            value: "cancel",
                            variant: "secondary",
                        },
                    ],
                );
                self.agent_client
                    .send_json_render_message(&self.to_username, spec)
                    .await
                    .context("send MCP elicitation prompt")?;

                let response = loop {
                    let reply = self.next_target_message().await?;
                    let Some(text) = extract_request_text(&reply) else {
                        continue;
                    };
                    if let Some(response) = parse_mcp_elicitation_response(&text) {
                        break response;
                    }
                    self.agent_client
                        .send_text_message(
                            &self.to_username,
                            "Reply with `accept`, `decline`, or `cancel`. You can append JSON after `accept` if the MCP server needs content.",
                        )
                        .await
                        .context("send MCP clarification")?;
                };

                self.codex
                    .respond(id, response)
                    .await
                    .context("respond to MCP elicitation")?;
            }
            other => {
                self.logger
                    .warning("Unhandled Codex interaction", [format!("Method: {other}")]);
                let spec = report_spec(
                    "Unsupported",
                    "Unhandled Codex interaction",
                    &[format!("Method: {other}")],
                    Some("codex-message does not implement this server request type yet."),
                );
                self.agent_client
                    .send_json_render_message(&self.to_username, spec)
                    .await
                    .context("send unsupported interaction notice")?;
                self.codex
                    .respond(id, json!({}))
                    .await
                    .context("respond to unsupported request with empty object")?;
            }
        }

        Ok(())
    }

    async fn send_turn_failure_notice(&self, request: &str, error_text: &str) {
        let spec = report_spec(
            "Failed",
            "Codex request failed",
            &[
                format!("Request: {}", request.trim()),
                format!("Error: {}", error_text.trim()),
            ],
            None,
        );
        if let Err(error) = self
            .agent_client
            .send_json_render_message(&self.to_username, spec)
            .await
        {
            self.logger.warning(
                "Failed to send failure notice",
                [format!("Error: {error:#}")],
            );
        }
    }
}

#[derive(Debug)]
struct TurnOutcome {
    turn_id: String,
    status: String,
    error_text: Option<String>,
}

fn should_mark_message_complete(outcome: &TurnOutcome) -> bool {
    outcome.status.eq_ignore_ascii_case("completed") && outcome.error_text.is_none()
}

fn request_preview(text: &str) -> String {
    const LIMIT: usize = 160;

    let compact = text.split_whitespace().collect::<Vec<_>>().join(" ");
    let mut preview = compact.chars().take(LIMIT).collect::<String>();
    if compact.chars().count() > LIMIT {
        preview.push_str("...");
    }
    preview
}

async fn register_agent_account(
    client: &mut AgentMessageClient,
    username: &str,
    password: &str,
) -> Result<()> {
    client
        .register(username, password)
        .await
        .context("register agent-message account")?;
    client
        .login(username, password)
        .await
        .context("refresh agent-message session after register")
}

async fn start_thread(codex: &CodexAppServer, config: &Config) -> Result<String> {
    let mut params = Map::new();
    params.insert(
        "cwd".to_string(),
        Value::String(config.cwd.to_string_lossy().into_owned()),
    );
    if let Some(model) = &config.model {
        params.insert("model".to_string(), Value::String(model.clone()));
    }

    let response = codex
        .request("thread/start", Value::Object(params))
        .await
        .context("start codex thread")?;
    response
        .get("thread")
        .and_then(|thread| thread.get("id"))
        .and_then(Value::as_str)
        .map(ToOwned::to_owned)
        .ok_or_else(|| anyhow!("thread/start response missing thread.id"))
}

fn build_turn_start_params(config: &Config, thread_id: &str, text: &str) -> Result<Value> {
    let mut params = Map::new();
    params.insert("threadId".to_string(), Value::String(thread_id.to_string()));
    params.insert(
        "input".to_string(),
        Value::Array(vec![json!({
            "type": "text",
            "text": text,
        })]),
    );
    params.insert(
        "cwd".to_string(),
        Value::String(config.cwd.to_string_lossy().into_owned()),
    );
    if let Some(model) = &config.model {
        params.insert("model".to_string(), Value::String(model.clone()));
    }
    if let Some(policy) = &config.approval_policy {
        params.insert("approvalPolicy".to_string(), Value::String(policy.clone()));
    }
    params.insert("sandboxPolicy".to_string(), sandbox_policy(config));
    Ok(Value::Object(params))
}

fn sandbox_policy(config: &Config) -> Value {
    match config.sandbox {
        SandboxArg::ReadOnly => json!({
            "type": "readOnly",
            "networkAccess": config.network_access,
        }),
        SandboxArg::WorkspaceWrite => json!({
            "type": "workspaceWrite",
            "writableRoots": [config.cwd.to_string_lossy()],
            "networkAccess": config.network_access,
        }),
        SandboxArg::DangerFullAccess => json!({
            "type": "dangerFullAccess",
        }),
    }
}

fn extract_request_text(message: &Message) -> Option<String> {
    let trimmed = message.text.trim();
    if trimmed.is_empty() {
        return None;
    }
    if trimmed == "[json-render]" || trimmed == "deleted message" {
        return None;
    }
    Some(trimmed.to_string())
}

fn summarize_command_approval(params: &Value) -> Vec<String> {
    let mut details = Vec::new();
    if let Some(reason) = params.get("reason").and_then(Value::as_str) {
        details.push(format!("Reason: {reason}"));
    }
    if let Some(command) = params.get("command").and_then(Value::as_str) {
        details.push(format!("Command: {command}"));
    }
    if let Some(cwd) = params.get("cwd").and_then(Value::as_str) {
        details.push(format!("CWD: {cwd}"));
    }
    if let Some(permissions) = params.get("additionalPermissions") {
        details.push(format!(
            "Additional permissions: {}",
            serde_json::to_string_pretty(permissions).unwrap_or_else(|_| permissions.to_string())
        ));
    }
    if details.is_empty() {
        details.push("Codex requested command execution approval.".to_string());
    }
    details
}

fn summarize_file_approval(params: &Value) -> Vec<String> {
    let mut details = Vec::new();
    if let Some(reason) = params.get("reason").and_then(Value::as_str) {
        details.push(format!("Reason: {reason}"));
    }
    if let Some(root) = params.get("grantRoot").and_then(Value::as_str) {
        details.push(format!("Grant root: {root}"));
    }
    if details.is_empty() {
        details.push("Codex requested approval for file changes.".to_string());
    }
    details
}

fn summarize_permissions_request(params: &Value) -> Vec<String> {
    let mut details = Vec::new();
    if let Some(reason) = params.get("reason").and_then(Value::as_str) {
        details.push(format!("Reason: {reason}"));
    }
    if let Some(permissions) = params.get("permissions") {
        details.push(format!(
            "Requested permissions: {}",
            serde_json::to_string_pretty(permissions).unwrap_or_else(|_| permissions.to_string())
        ));
    }
    if details.is_empty() {
        details.push("Codex requested additional permissions.".to_string());
    }
    details
}

fn summarize_tool_questions(questions: &[Value]) -> Vec<String> {
    let mut details = Vec::new();
    for question in questions {
        let id = question
            .get("id")
            .and_then(Value::as_str)
            .unwrap_or("question");
        let text = question
            .get("question")
            .and_then(Value::as_str)
            .unwrap_or("No question text provided");
        details.push(format!("{id}: {text}"));
        if let Some(options) = question.get("options").and_then(Value::as_array) {
            let labels: Vec<String> = options
                .iter()
                .filter_map(|option| option.get("label").and_then(Value::as_str))
                .map(ToOwned::to_owned)
                .collect();
            if !labels.is_empty() {
                details.push(format!("Options for {id}: {}", labels.join(", ")));
            }
        }
    }
    if details.is_empty() {
        details.push("Codex requested additional user input.".to_string());
    }
    details
}

fn summarize_mcp_elicitation(params: &Value) -> Vec<String> {
    let mut details = Vec::new();
    if let Some(server_name) = params.get("serverName").and_then(Value::as_str) {
        details.push(format!("Server: {server_name}"));
    }
    if let Some(mode) = params.get("mode").and_then(Value::as_str) {
        details.push(format!("Mode: {mode}"));
    }
    if let Some(message) = params.get("message").and_then(Value::as_str) {
        details.push(format!("Message: {message}"));
    }
    if let Some(url) = params.get("url").and_then(Value::as_str) {
        details.push(format!("URL: {url}"));
    }
    if details.is_empty() {
        details.push("An MCP server requested interaction.".to_string());
    }
    details
}

fn parse_command_decision(text: &str) -> Option<Value> {
    let normalized = normalize_reply(text);
    if normalized.contains("cancel") || normalized.contains("abort") {
        return Some(json!("cancel"));
    }
    if normalized.contains("session") {
        return Some(json!("acceptForSession"));
    }
    if normalized.contains("deny") || normalized.contains("decline") || normalized == "no" {
        return Some(json!("decline"));
    }
    if normalized.contains("approve")
        || normalized.contains("accept")
        || normalized.contains("allow")
        || normalized == "yes"
    {
        return Some(json!("accept"));
    }
    None
}

fn parse_file_decision(text: &str) -> Option<Value> {
    parse_command_decision(text)
}

fn parse_permission_response(text: &str, requested_permissions: &Value) -> Option<Value> {
    let normalized = normalize_reply(text);
    if normalized.contains("deny") || normalized.contains("decline") || normalized == "no" {
        return Some(json!({
            "scope": "turn",
            "permissions": {},
        }));
    }
    if normalized.contains("allow")
        || normalized.contains("approve")
        || normalized.contains("accept")
        || normalized == "yes"
    {
        let scope = if normalized.contains("session") {
            "session"
        } else {
            "turn"
        };
        return Some(json!({
            "scope": scope,
            "permissions": requested_permissions,
        }));
    }
    None
}

fn parse_tool_user_input_response(text: &str, questions: &[Value]) -> Option<Value> {
    if questions.is_empty() {
        return Some(json!({ "answers": {} }));
    }

    if let Ok(value) = serde_json::from_str::<Value>(text) {
        let answers_object = value.get("answers").unwrap_or(&value);
        if let Some(answer_map) = answers_object.as_object() {
            let mut answers = Map::new();
            for question in questions {
                let id = question.get("id").and_then(Value::as_str)?;
                let answer_value = answer_map.get(id)?;
                let answers_array = match answer_value {
                    Value::String(text) => vec![Value::String(text.clone())],
                    Value::Array(values) => values.clone(),
                    _ => return None,
                };
                answers.insert(id.to_string(), json!({ "answers": answers_array }));
            }
            return Some(json!({ "answers": answers }));
        }
    }

    if questions.len() == 1 {
        let id = questions[0].get("id").and_then(Value::as_str)?;
        let answer = text.trim();
        if answer.is_empty() {
            return None;
        }
        return Some(json!({
            "answers": {
                id: {
                    "answers": [answer],
                }
            }
        }));
    }

    None
}

fn parse_mcp_elicitation_response(text: &str) -> Option<Value> {
    let trimmed = text.trim();
    if trimmed.is_empty() {
        return None;
    }
    let normalized = normalize_reply(trimmed);
    if normalized == "decline" || normalized == "deny" {
        return Some(json!({ "action": "decline", "content": Value::Null }));
    }
    if normalized == "cancel" || normalized == "abort" {
        return Some(json!({ "action": "cancel", "content": Value::Null }));
    }
    if normalized.starts_with("accept") {
        let rest = trimmed["accept".len()..].trim();
        let content = if rest.is_empty() {
            Value::Null
        } else if let Ok(json_value) = serde_json::from_str::<Value>(rest) {
            json_value
        } else {
            Value::String(rest.to_string())
        };
        return Some(json!({ "action": "accept", "content": content }));
    }
    None
}

fn normalize_reply(text: &str) -> String {
    text.trim().to_ascii_lowercase()
}

fn new_chat_id() -> String {
    Uuid::new_v4().simple().to_string()[..12].to_string()
}

fn new_password() -> String {
    let mut rng = rand::rng();
    format!("{:06}", rng.random_range(0..=999_999))
}

async fn resolve_target_username(
    config: &Config,
    agent_client: &AgentMessageClient,
) -> Result<String> {
    if let Some(username) = config
        .to_username
        .as_deref()
        .map(str::trim)
        .filter(|username| !username.is_empty())
    {
        return Ok(username.to_string());
    }

    agent_client
        .master_username()
        .await
        .context("resolve target username from agent-message master")
}

fn watch_retry_delay(attempt: usize) -> Duration {
    let seconds = WATCH_RETRY_DELAYS_SECS
        .get(attempt)
        .copied()
        .unwrap_or(*WATCH_RETRY_DELAYS_SECS.last().unwrap_or(&5));
    Duration::from_secs(seconds)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn watch_retry_delay_caps_at_last_value() {
        assert_eq!(watch_retry_delay(0), Duration::from_secs(1));
        assert_eq!(watch_retry_delay(1), Duration::from_secs(2));
        assert_eq!(watch_retry_delay(2), Duration::from_secs(5));
        assert_eq!(watch_retry_delay(10), Duration::from_secs(5));
    }

    #[test]
    fn command_approval_reply_is_parsed() {
        assert_eq!(parse_command_decision("approve"), Some(json!("accept")));
        assert_eq!(
            parse_command_decision("allow session"),
            Some(json!("acceptForSession"))
        );
        assert_eq!(parse_command_decision("deny"), Some(json!("decline")));
        assert_eq!(parse_command_decision("cancel"), Some(json!("cancel")));
    }

    #[test]
    fn permission_reply_grants_requested_subset() {
        let requested = json!({
            "network": { "enabled": true },
            "fileSystem": { "write": ["/tmp/demo"] }
        });
        assert_eq!(
            parse_permission_response("allow session", &requested),
            Some(json!({
                "scope": "session",
                "permissions": requested,
            }))
        );
    }

    #[test]
    fn request_user_input_accepts_single_plain_text_reply() {
        let questions = vec![json!({
            "id": "workspace",
            "question": "Which workspace should I use?",
        })];

        assert_eq!(
            parse_tool_user_input_response("repo-a", &questions),
            Some(json!({
                "answers": {
                    "workspace": {
                        "answers": ["repo-a"]
                    }
                }
            }))
        );
    }

    #[test]
    fn extract_request_text_filters_special_messages() {
        assert_eq!(
            extract_request_text(&Message {
                id: "m1".to_string(),
                sender_username: "jay".to_string(),
                kind: "json_render".to_string(),
                text: "[json-render]".to_string(),
            }),
            None
        );
    }

    #[test]
    fn startup_text_includes_cwd_line() {
        let text = startup_text(
            "chat-1",
            "agent-chat-1",
            "1234",
            std::path::Path::new("/tmp/demo"),
            "demo-host",
        );
        assert!(text.contains("CWD: /tmp/demo"));
        assert!(text.contains("Hostname: demo-host"));
    }

    #[test]
    fn marks_message_complete_only_for_successful_completed_turns() {
        assert!(should_mark_message_complete(&TurnOutcome {
            turn_id: "turn-1".to_string(),
            status: "completed".to_string(),
            error_text: None,
        }));
        assert!(!should_mark_message_complete(&TurnOutcome {
            turn_id: "turn-2".to_string(),
            status: "failed".to_string(),
            error_text: Some("boom".to_string()),
        }));
        assert!(!should_mark_message_complete(&TurnOutcome {
            turn_id: "turn-3".to_string(),
            status: "completed".to_string(),
            error_text: Some("boom".to_string()),
        }));
    }

    #[test]
    fn request_suffix_discourages_wrapping_final_results_in_card() {
        let suffix = request_suffix("jay", "agent-123");
        assert!(suffix.contains("avoid wrapping the entire payload in a `Card`"));
        assert!(suffix.contains("prefer a direct content-first layout such as `Stack`"));
        assert!(suffix.contains("Use sender account `agent-123`"));
        assert!(suffix.contains("Always pass `--from agent-123`"));
    }
}
