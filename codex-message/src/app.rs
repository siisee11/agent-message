use anyhow::{Context, Result, anyhow};
use rand::Rng;
use serde_json::Map;
use serde_json::Value;
use serde_json::json;
use uuid::Uuid;

use crate::Config;
use crate::SandboxArg;
use crate::agent_message::{AgentMessageClient, Message, MessageWatch};
use crate::codex::{CodexAppServer, IncomingMessage};
use crate::render::{ApprovalAction, approval_spec, report_spec};

const REQUEST_SUFFIX: &str = r#"

Operational requirements from the codex-message wrapper:
- The final result will be forwarded to the user through agent-message.
- Prefer a visually readable report format suitable for an agent-message json-render response.
- If you need approval or clarification, ask clearly and briefly so the wrapper can relay it.
"#;
const READ_REACTION_EMOJI: &str = "👀";

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
    chat_id: String,
    to_username: String,
    agent_client: AgentMessageClient,
    message_watch: MessageWatch,
    codex: CodexAppServer,
    thread_id: String,
}

impl Runtime {
    async fn bootstrap(config: Config) -> Result<Self> {
        let chat_id = new_chat_id();
        let username = format!("agent-{chat_id}");
        let pin = new_pin();
        let to_username = config.to_username.clone();
        let agent_client = AgentMessageClient::new(std::path::PathBuf::from("agent-message"));
        let server_url = agent_client
            .server_url()
            .await
            .context("read agent-message server_url")?;

        println!("agent-message server_url: {server_url}");

        register_agent_account(&agent_client, &username, &pin).await?;
        println!("registered agent profile: {username} (chat_id: {chat_id})");

        let message_watch = agent_client
            .watch_messages(&to_username)
            .await
            .context("start agent-message watch stream")?;
        println!("agent-message watch stream ready for {to_username}");

        let startup_text = format!(
            "codex-message session started\nchat_id: {chat_id}\nusername: {username}\npin: {pin}\n\nReply in this DM to run Codex."
        );
        let startup_message_id = agent_client
            .send_text_message(&to_username, &startup_text)
            .await
            .context("send startup message")?;
        println!("startup message sent to {to_username}: {startup_message_id}");

        let codex = CodexAppServer::start(&config.codex_bin, &config.cwd)
            .await
            .context("start codex app-server")?;
        codex
            .initialize()
            .await
            .context("initialize codex app-server")?;
        let thread_id = start_thread(&codex, &config).await?;
        println!("codex app-server ready (thread_id: {thread_id})");

        Ok(Self {
            config,
            chat_id,
            to_username,
            agent_client,
            message_watch,
            codex,
            thread_id,
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

                    match self.run_turn(&request).await {
                        Ok(outcome) => {
                            let spec = report_spec(
                                match outcome.status.as_str() {
                                    "completed" => "Completed",
                                    "interrupted" => "Interrupted",
                                    _ => "Failed",
                                },
                                "Codex report",
                                &[
                                    format!("Chat ID: {}", self.chat_id),
                                    format!("Request: {}", request.trim()),
                                    format!("Status: {}", outcome.status),
                                ],
                                Some(&outcome.report_body()),
                            );
                            self.agent_client
                                .send_json_render_message(&self.to_username, spec)
                                .await
                                .context("send turn report")?;
                        }
                        Err(error) => {
                            let spec = report_spec(
                                "Error",
                                "Codex request failed",
                                &[
                                    format!("Chat ID: {}", self.chat_id),
                                    format!("Request: {}", request.trim()),
                                ],
                                Some(&error.to_string()),
                            );
                            self.agent_client
                                .send_json_render_message(&self.to_username, spec)
                                .await
                                .context("send failure report")?;
                        }
                    }
                }
            }
        }
    }

    async fn next_target_message(&mut self) -> Result<Message> {
        loop {
            let message = self
                .message_watch
                .next_message()
                .await
                .context("read next event from agent-message watch stream")?;
            if !message.sender_username.eq_ignore_ascii_case(&self.to_username) {
                continue;
            }
            if extract_request_text(&message).is_none() {
                continue;
            }
            if let Err(error) = self
                .agent_client
                .react_to_message(&message, READ_REACTION_EMOJI)
                .await
            {
                eprintln!(
                    "warning: failed to add read reaction to {}: {error}",
                    message.id
                );
            }
            return Ok(message);
        }
    }

    async fn run_turn(&mut self, request: &str) -> Result<TurnOutcome> {
        let composed_request = format!("{}\n{}", request.trim(), REQUEST_SUFFIX);
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

        let mut agent_text = String::new();

        let (turn_status, turn_error) = loop {
            match self.codex.next_event().await? {
                IncomingMessage::Notification { method, params } => match method.as_str() {
                    "item/agentMessage/delta" => {
                        if params.get("turnId").and_then(Value::as_str) == Some(turn_id.as_str()) {
                            if let Some(delta) = params.get("delta").and_then(Value::as_str) {
                                agent_text.push_str(delta);
                            }
                        }
                    }
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
            status: turn_status,
            agent_text: agent_text.trim().to_string(),
            error_text: turn_error,
        })
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
}

#[derive(Debug)]
struct TurnOutcome {
    status: String,
    agent_text: String,
    error_text: Option<String>,
}

impl TurnOutcome {
    fn report_body(&self) -> String {
        match (&self.agent_text.is_empty(), &self.error_text) {
            (false, Some(error)) => format!("{}\n\nError: {}", self.agent_text, error),
            (false, None) => self.agent_text.clone(),
            (true, Some(error)) => error.clone(),
            (true, None) => "Codex completed without an assistant message.".to_string(),
        }
    }
}

async fn register_agent_account(
    client: &AgentMessageClient,
    username: &str,
    pin: &str,
) -> Result<()> {
    client
        .register(username, pin)
        .await
        .context("register agent-message account")?;
    client
        .login(username, pin)
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

fn new_pin() -> String {
    let mut rng = rand::rng();
    format!("{:06}", rng.random_range(0..=999_999))
}

#[cfg(test)]
mod tests {
    use super::*;

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
                text: "[json-render]".to_string(),
            }),
            None
        );
    }
}
