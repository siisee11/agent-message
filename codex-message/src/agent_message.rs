use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::time::Duration;

use anyhow::{Context, Result, anyhow, bail};
use serde::Deserialize;
use serde_json::Value;
use tokio::io::{AsyncBufReadExt, BufReader};
use tokio::process::{Child, ChildStderr, ChildStdout, Command};
use tokio::sync::mpsc;
use url::Url;

use crate::log_ui::LogUi;

const AGENT_MESSAGE_MODE_ENV: &str = "CODEX_MESSAGE_AGENT_MESSAGE_MODE";

#[derive(Debug, Clone)]
pub(crate) struct AgentMessageClient {
    command: AgentMessageCommand,
    from_profile: Option<String>,
    logger: LogUi,
}

#[derive(Debug, Clone)]
struct AgentMessageCommand {
    program: PathBuf,
    base_args: Vec<String>,
    cwd: Option<PathBuf>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum AgentMessageMode {
    Installed,
    Source,
}

impl AgentMessageClient {
    pub(crate) fn new(binary: PathBuf, logger: LogUi) -> Result<Self> {
        Ok(Self {
            command: resolve_agent_message_command(binary)?,
            from_profile: None,
            logger,
        })
    }

    pub(crate) fn set_from_profile(&mut self, profile: String) {
        self.from_profile = Some(profile);
    }

    pub(crate) async fn server_url(&self) -> Result<String> {
        let output = self
            .run(&["config", "get", "server_url"])
            .await
            .context("run `agent-message config get server_url`")?;
        let server_url = output.trim();
        if server_url.is_empty() {
            bail!("agent-message config get server_url returned an empty value");
        }
        Ok(server_url.to_string())
    }

    pub(crate) async fn master_username(&self) -> Result<String> {
        let output = self
            .run(&["config", "get", "master"])
            .await
            .context("run `agent-message config get master`")?;
        let master = output.trim();
        if master.is_empty() {
            bail!(
                "agent-message master is empty; pass --to or set one with `agent-message config set master <username>`"
            );
        }
        Ok(master.to_string())
    }

    pub(crate) async fn register(&self, username: &str, password: &str) -> Result<()> {
        let output = self
            .run(&["register", username, password])
            .await
            .context("run `agent-message register`")?;
        if !output.contains(&format!("registered {username}")) {
            bail!("unexpected register output: {output}");
        }
        Ok(())
    }

    pub(crate) async fn login(&self, username: &str, password: &str) -> Result<()> {
        let output = self
            .run(&["login", username, password])
            .await
            .context("run `agent-message login`")?;
        if !output.contains(&format!("logged in as {username}")) {
            bail!("unexpected login output: {output}");
        }
        Ok(())
    }

    pub(crate) async fn send_text_message(&self, username: &str, text: &str) -> Result<String> {
        let output = self
            .run(&["send", username, text])
            .await
            .context("run `agent-message send`")?;
        let message_id = parse_sent_message_id(&output)?;
        self.logger.send(
            "Message sent",
            [
                format!("From: {}", self.sender_label()),
                format!("To: {}", format_username(username)),
                "Kind: text".to_string(),
                format!("Message: {message_id}"),
                format!("Text: {}", preview_text(text)),
            ],
        );
        Ok(message_id)
    }

    pub(crate) async fn send_json_render_message(
        &self,
        username: &str,
        spec: Value,
    ) -> Result<String> {
        let payload = serde_json::to_string(&spec).context("encode json_render spec")?;
        let output = self
            .run(&["send", username, &payload, "--kind", "json_render"])
            .await
            .context("run `agent-message send --kind json_render`")?;
        let message_id = parse_sent_message_id(&output)?;
        self.logger.send(
            "Message sent",
            [
                format!("From: {}", self.sender_label()),
                format!("To: {}", format_username(username)),
                "Kind: json_render".to_string(),
                format!("Message: {message_id}"),
                format!("Payload bytes: {}", payload.len()),
            ],
        );
        Ok(message_id)
    }

    pub(crate) async fn set_conversation_title(&self, username: &str, title: &str) -> Result<()> {
        let output = self
            .run(&["title", "set", username, title])
            .await
            .context("run `agent-message title set`")?;
        if !output.contains(&format!("title set for {username}")) {
            bail!("unexpected title output: {output}");
        }
        Ok(())
    }

    pub(crate) async fn watch_messages(&self, username: &str) -> Result<MessageWatch> {
        let server_url = self
            .server_url()
            .await
            .context("read agent-message server_url for watch")?;
        let mut command = self.build_command();
        command
            .args(["watch", username, "--json"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .kill_on_drop(true);

        let mut child = command
            .spawn()
            .with_context(|| format!("spawn `{}`", self.command.describe()))?;
        let stdout = child
            .stdout
            .take()
            .context("capture agent-message watch stdout")?;
        let stderr = child
            .stderr
            .take()
            .context("capture agent-message watch stderr")?;
        let (messages_tx, messages_rx) = mpsc::unbounded_channel();
        spawn_watch_stdout_pump(stdout, messages_tx, self.logger.clone(), server_url);
        spawn_watch_stderr_pump(stderr, self.logger.clone());

        Ok(MessageWatch {
            child,
            messages_rx,
            logger: self.logger.clone(),
        })
    }

    async fn run(&self, args: &[&str]) -> Result<String> {
        let mut command = self.build_command();
        command
            .args(args)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());

        let child = command
            .spawn()
            .with_context(|| format!("spawn `{}`", self.command.describe()))?;
        let output = tokio::time::timeout(Duration::from_secs(30), child.wait_with_output())
            .await
            .context("agent-message command timed out")?
            .context("wait for agent-message command")?;

        let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        if !output.status.success() {
            let detail = if stderr.is_empty() {
                stdout.clone()
            } else {
                stderr.clone()
            };
            return Err(anyhow!("agent-message command failed: {detail}"));
        }
        Ok(stdout)
    }

    fn build_command(&self) -> Command {
        let mut command = Command::new(&self.command.program);
        if let Some(cwd) = &self.command.cwd {
            command.current_dir(cwd);
        }
        command.args(&self.command.base_args);
        if let Some(profile) = &self.from_profile {
            command.args(["--from", profile]);
        }
        command
    }

    pub(crate) fn sender_label(&self) -> String {
        self.from_profile
            .as_deref()
            .map(format_username)
            .unwrap_or_else(|| "<active-profile>".to_string())
    }
}

impl AgentMessageCommand {
    fn describe(&self) -> String {
        let mut parts = vec![self.program.display().to_string()];
        parts.extend(self.base_args.iter().cloned());
        parts.join(" ")
    }
}

fn format_username(username: &str) -> String {
    let trimmed = username.trim();
    if trimmed.is_empty() {
        "<empty>".to_string()
    } else {
        format!("@{trimmed}")
    }
}

fn preview_text(text: &str) -> String {
    const LIMIT: usize = 120;

    let compact = text.split_whitespace().collect::<Vec<_>>().join(" ");
    let mut preview = compact.chars().take(LIMIT).collect::<String>();
    if compact.chars().count() > LIMIT {
        preview.push_str("...");
    }
    preview
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct Message {
    pub(crate) id: String,
    pub(crate) sender_username: String,
    pub(crate) kind: String,
    pub(crate) text: String,
    pub(crate) json_render_spec: Option<Value>,
}

pub(crate) struct MessageWatch {
    child: Child,
    messages_rx: mpsc::UnboundedReceiver<Message>,
    logger: LogUi,
}

impl MessageWatch {
    pub(crate) async fn next_message(&mut self) -> Result<Message> {
        self.messages_rx
            .recv()
            .await
            .ok_or_else(|| anyhow!("agent-message watch stream ended"))
    }

    pub(crate) async fn shutdown(&mut self) -> Result<()> {
        if let Err(error) = self.child.start_kill() {
            self.logger.warning(
                "Watch shutdown warning",
                [format!("failed to signal watch shutdown: {error}")],
            );
        }
        let _ = self.child.wait().await;
        Ok(())
    }
}

fn resolve_agent_message_command(binary: PathBuf) -> Result<AgentMessageCommand> {
    match resolve_agent_message_mode()? {
        AgentMessageMode::Installed => {
            if binary.as_os_str().is_empty() {
                bail!("agent-message binary path is empty");
            }

            Ok(AgentMessageCommand {
                program: binary,
                base_args: Vec::new(),
                cwd: None,
            })
        }
        AgentMessageMode::Source => {
            let Some(cli_dir) = source_cli_dir() else {
                bail!(
                    "{AGENT_MESSAGE_MODE_ENV}=source requires the repo CLI at ../cli relative to the codex-message sources"
                );
            };

            Ok(AgentMessageCommand {
                program: PathBuf::from("go"),
                base_args: vec!["run".to_string(), ".".to_string()],
                cwd: Some(cli_dir),
            })
        }
    }
}

fn resolve_agent_message_mode() -> Result<AgentMessageMode> {
    resolve_agent_message_mode_from_env(std::env::var(AGENT_MESSAGE_MODE_ENV).ok().as_deref())
}

fn resolve_agent_message_mode_from_env(value: Option<&str>) -> Result<AgentMessageMode> {
    match value.map(str::trim).filter(|value| !value.is_empty()) {
        None | Some("installed") => Ok(AgentMessageMode::Installed),
        Some("source") => Ok(AgentMessageMode::Source),
        Some(other) => bail!(
            "invalid {AGENT_MESSAGE_MODE_ENV} value {other:?}; expected \"installed\" or \"source\""
        ),
    }
}

fn source_cli_dir() -> Option<PathBuf> {
    let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));
    let cli_dir = manifest_dir.parent()?.join("cli");
    if cli_dir.join("go.mod").is_file() {
        Some(cli_dir)
    } else {
        None
    }
}

fn parse_sent_message_id(output: &str) -> Result<String> {
    let trimmed = output.trim();
    let Some(rest) = trimmed.strip_prefix("sent ") else {
        bail!("unexpected send output: {trimmed}");
    };
    let id = rest.trim();
    if id.is_empty() {
        bail!("send output did not contain a message id");
    }
    Ok(id.to_string())
}

fn parse_watch_event(line: &str, server_url: &str) -> Result<Option<Message>> {
    let trimmed = line.trim();
    if trimmed.is_empty() {
        return Ok(None);
    }

    let event: WatchJSONEvent =
        serde_json::from_str(trimmed).context("decode agent-message watch JSON line")?;
    if event.event_type.trim() != "message.new" {
        return Ok(None);
    }

    let message = event.message;
    Ok(Some(Message {
        id: message.id.trim().to_string(),
        sender_username: message.sender.username.trim().to_string(),
        kind: message.kind.trim().to_string(),
        text: watch_message_text(&message, server_url),
        json_render_spec: message.json_render_spec,
    }))
}

#[derive(Debug, Deserialize)]
struct WatchJSONEvent {
    #[serde(rename = "type")]
    event_type: String,
    message: WatchJSONMessage,
}

#[derive(Debug, Deserialize)]
struct WatchJSONMessage {
    id: String,
    sender: WatchJSONSender,
    content: Option<String>,
    kind: String,
    #[serde(default)]
    json_render_spec: Option<Value>,
    attachment_url: Option<String>,
    attachment_type: Option<String>,
    deleted: bool,
}

#[derive(Debug, Deserialize)]
struct WatchJSONSender {
    username: String,
}

fn watch_message_text(message: &WatchJSONMessage, server_url: &str) -> String {
    if message.deleted {
        return "deleted message".to_string();
    }
    if message.kind.trim() == "json_render" {
        return "[json-render]".to_string();
    }
    let content = message
        .content
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty());
    let attachment_url = message
        .attachment_url
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(|value| normalize_attachment_url(server_url, value));

    match (content, attachment_url) {
        (Some(content), Some(attachment_url)) => format!(
            "{content}\n\nAttached {}: {attachment_url}",
            attachment_label(message.attachment_type.as_deref())
        ),
        (Some(content), None) => content.to_string(),
        (None, Some(attachment_url)) => format!(
            "Attached {}: {attachment_url}",
            attachment_label(message.attachment_type.as_deref())
        ),
        (None, None) => String::new(),
    }
}

fn attachment_label(attachment_type: Option<&str>) -> &'static str {
    match attachment_type
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        Some("image") => "image",
        Some("file") => "file",
        _ => "attachment",
    }
}

fn normalize_attachment_url(server_url: &str, attachment_url: &str) -> String {
    let Ok(base_url) = Url::parse(server_url) else {
        return attachment_url.to_string();
    };
    base_url
        .join(attachment_url)
        .map(|value| value.to_string())
        .unwrap_or_else(|_| attachment_url.to_string())
}

fn spawn_watch_stdout_pump(
    stdout: ChildStdout,
    messages_tx: mpsc::UnboundedSender<Message>,
    logger: LogUi,
    server_url: String,
) {
    tokio::spawn(async move {
        let mut lines = BufReader::new(stdout).lines();
        loop {
            let next = lines.next_line().await;
            let line = match next {
                Ok(Some(line)) => line,
                Ok(None) => break,
                Err(error) => {
                    logger.warning(
                        "Watch stream read failed",
                        [format!("stdout read error: {error}")],
                    );
                    break;
                }
            };

            match parse_watch_event(&line, &server_url) {
                Ok(Some(message)) => {
                    if messages_tx.send(message).is_err() {
                        break;
                    }
                }
                Ok(None) => {}
                Err(error) => {
                    logger.warning("Watch event decode failed", [format!("error: {error:#}")]);
                }
            }
        }
    });
}

fn spawn_watch_stderr_pump(stderr: ChildStderr, logger: LogUi) {
    tokio::spawn(async move {
        let mut lines = BufReader::new(stderr).lines();
        loop {
            match lines.next_line().await {
                Ok(Some(line)) => {
                    logger.child("agent-message stderr", [line]);
                }
                Ok(None) => break,
                Err(error) => {
                    logger.warning(
                        "Watch stderr read failed",
                        [format!("stderr read error: {error}")],
                    );
                    break;
                }
            }
        }
    });
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_watch_text_message() {
        let parsed = parse_watch_event(
            r#"{"type":"message.new","conversation_id":"c-1","message":{"id":"m-123","conversation_id":"c-1","sender":{"id":"u-1","username":"jay"},"content":"hello world","kind":"text","edited":false,"deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}"#,
            "https://agent.example.test",
        )
        .expect("parse watch event")
        .expect("message event");
        assert_eq!(
            parsed,
            Message {
                id: "m-123".to_string(),
                sender_username: "jay".to_string(),
                kind: "text".to_string(),
                text: "hello world".to_string(),
                json_render_spec: None,
            }
        );
    }

    #[test]
    fn parses_send_output() {
        assert_eq!(
            parse_sent_message_id("sent m-42").expect("parse send output"),
            "m-42"
        );
    }

    #[test]
    fn formats_usernames_for_logs() {
        assert_eq!(format_username("jay"), "@jay");
        assert_eq!(format_username("  jay  "), "@jay");
        assert_eq!(format_username("   "), "<empty>");
    }

    #[test]
    fn previews_text_for_logs() {
        assert_eq!(preview_text("hello   world"), "hello world");
        assert!(preview_text(&"a".repeat(130)).ends_with("..."));
    }

    #[test]
    fn parses_watch_json_render_message() {
        let parsed = parse_watch_event(
            r#"{"type":"message.new","conversation_id":"c-1","message":{"id":"m-124","conversation_id":"c-1","sender":{"id":"u-1","username":"jay"},"kind":"json_render","json_render_spec":{"root":"main","elements":{"main":{"type":"Stack"}}},"deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}"#,
            "https://agent.example.test",
        )
        .expect("parse watch event")
        .expect("message event");

        assert_eq!(parsed.id, "m-124");
        assert_eq!(parsed.sender_username, "jay");
        assert_eq!(parsed.kind, "json_render");
        assert_eq!(parsed.text, "[json-render]");
        assert_eq!(
            parsed.json_render_spec,
            Some(serde_json::json!({
                "root": "main",
                "elements": {
                    "main": {
                        "type": "Stack"
                    }
                }
            }))
        );
    }

    #[test]
    fn parses_watch_message_with_text_and_attachment() {
        let parsed = parse_watch_event(
            r#"{"type":"message.new","conversation_id":"c-1","message":{"id":"m-125","conversation_id":"c-1","sender":{"id":"u-1","username":"jay"},"content":"look at this","kind":"text","attachment_url":"/static/uploads/diagram.png","attachment_type":"image","deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}"#,
            "https://agent.example.test/base",
        )
        .expect("parse watch event")
        .expect("message event");

        assert_eq!(parsed.id, "m-125");
        assert_eq!(
            parsed.text,
            "look at this\n\nAttached image: https://agent.example.test/static/uploads/diagram.png"
        );
    }

    #[test]
    fn parses_watch_message_with_attachment_only() {
        let parsed = parse_watch_event(
            r#"{"type":"message.new","conversation_id":"c-1","message":{"id":"m-126","conversation_id":"c-1","sender":{"id":"u-1","username":"jay"},"kind":"text","attachment_url":"https://cdn.example.test/diagram.png","attachment_type":"image","deleted":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}"#,
            "https://agent.example.test",
        )
        .expect("parse watch event")
        .expect("message event");

        assert_eq!(
            parsed.text,
            "Attached image: https://cdn.example.test/diagram.png"
        );
    }

    #[test]
    fn defaults_to_installed_agent_message_binary() {
        let command = resolve_agent_message_command_with_mode(
            PathBuf::from("agent-message"),
            AgentMessageMode::Installed,
        )
        .expect("resolve command");
        assert_eq!(command.program, PathBuf::from("agent-message"));
        assert!(command.base_args.is_empty());
        assert!(command.cwd.is_none());
    }

    #[test]
    fn source_mode_uses_repo_cli_when_available() {
        let command = resolve_agent_message_command_with_mode(
            PathBuf::from("agent-message"),
            AgentMessageMode::Source,
        )
        .expect("resolve command");
        assert_eq!(command.program, PathBuf::from("go"));
        assert_eq!(command.base_args, vec!["run".to_string(), ".".to_string()]);
        assert!(command.cwd.as_ref().is_some_and(|cwd| cwd.ends_with("cli")));
    }

    #[test]
    fn parses_agent_message_mode_from_env() {
        assert_eq!(
            resolve_agent_message_mode_from_env(None).expect("default mode"),
            AgentMessageMode::Installed
        );
        assert_eq!(
            resolve_agent_message_mode_from_env(Some("installed")).expect("installed mode"),
            AgentMessageMode::Installed
        );
        assert_eq!(
            resolve_agent_message_mode_from_env(Some("source")).expect("source mode"),
            AgentMessageMode::Source
        );
        let error = resolve_agent_message_mode_from_env(Some("weird")).expect_err("invalid mode");
        assert!(
            error
                .to_string()
                .contains("expected \"installed\" or \"source\"")
        );
    }

    fn resolve_agent_message_command_with_mode(
        binary: PathBuf,
        mode: AgentMessageMode,
    ) -> Result<AgentMessageCommand> {
        match mode {
            AgentMessageMode::Installed => {
                if binary.as_os_str().is_empty() {
                    bail!("agent-message binary path is empty");
                }

                Ok(AgentMessageCommand {
                    program: binary,
                    base_args: Vec::new(),
                    cwd: None,
                })
            }
            AgentMessageMode::Source => {
                let Some(cli_dir) = source_cli_dir() else {
                    bail!("missing repo cli")
                };
                Ok(AgentMessageCommand {
                    program: PathBuf::from("go"),
                    base_args: vec!["run".to_string(), ".".to_string()],
                    cwd: Some(cli_dir),
                })
            }
        }
    }
}
