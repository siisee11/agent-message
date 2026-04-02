use std::path::PathBuf;
use std::process::Stdio;
use std::time::Duration;

use anyhow::{Context, Result, anyhow, bail};
use serde::Deserialize;
use serde_json::Value;
use tokio::io::{AsyncBufReadExt, BufReader};
use tokio::process::{Child, ChildStderr, ChildStdout, Command};
use tokio::sync::mpsc;

#[derive(Debug, Clone)]
pub(crate) struct AgentMessageClient {
    binary: PathBuf,
}

impl AgentMessageClient {
    pub(crate) fn new(binary: PathBuf) -> Self {
        Self { binary }
    }

    pub(crate) async fn server_url(&self) -> Result<String> {
        let output = self
            .run(["config", "get", "server_url"])
            .await
            .context("run `agent-message config get server_url`")?;
        let server_url = output.trim();
        if server_url.is_empty() {
            bail!("agent-message config get server_url returned an empty value");
        }
        Ok(server_url.to_string())
    }

    pub(crate) async fn register(&self, username: &str, pin: &str) -> Result<()> {
        let output = self
            .run(["register", username, pin])
            .await
            .context("run `agent-message register`")?;
        if !output.contains(&format!("registered {username}")) {
            bail!("unexpected register output: {output}");
        }
        Ok(())
    }

    pub(crate) async fn login(&self, username: &str, pin: &str) -> Result<()> {
        let output = self
            .run(["login", username, pin])
            .await
            .context("run `agent-message login`")?;
        if !output.contains(&format!("logged in as {username}")) {
            bail!("unexpected login output: {output}");
        }
        Ok(())
    }

    pub(crate) async fn send_text_message(&self, username: &str, text: &str) -> Result<String> {
        let output = self
            .run(["send", username, text])
            .await
            .context("run `agent-message send`")?;
        parse_sent_message_id(&output)
    }

    pub(crate) async fn send_json_render_message(
        &self,
        username: &str,
        spec: Value,
    ) -> Result<String> {
        let payload = serde_json::to_string(&spec).context("encode json_render spec")?;
        let output = self
            .run(["send", username, &payload, "--kind", "json_render"])
            .await
            .context("run `agent-message send --kind json_render`")?;
        parse_sent_message_id(&output)
    }

    pub(crate) async fn watch_messages(&self, username: &str) -> Result<MessageWatch> {
        let mut command = Command::new(&self.binary);
        command
            .args(["watch", username, "--json"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .kill_on_drop(true);

        let mut child = command
            .spawn()
            .with_context(|| format!("spawn `{}`", self.binary.display()))?;
        let stdout = child
            .stdout
            .take()
            .context("capture agent-message watch stdout")?;
        let stderr = child
            .stderr
            .take()
            .context("capture agent-message watch stderr")?;
        let (messages_tx, messages_rx) = mpsc::unbounded_channel();
        spawn_watch_stdout_pump(stdout, messages_tx);
        spawn_watch_stderr_pump(stderr);

        Ok(MessageWatch { child, messages_rx })
    }

    pub(crate) async fn react_to_message(&self, message: &Message, emoji: &str) -> Result<()> {
        let output = self
            .run(["react", &message.id, emoji])
            .await
            .context("run `agent-message react`")?;
        if !output.contains(&message.id) {
            bail!("unexpected react output: {output}");
        }
        Ok(())
    }

    pub(crate) async fn unreact_to_message(&self, message: &Message, emoji: &str) -> Result<()> {
        let output = self
            .run(["unreact", &message.id, emoji])
            .await
            .context("run `agent-message unreact`")?;
        if !output.contains(&message.id) {
            bail!("unexpected unreact output: {output}");
        }
        Ok(())
    }

    async fn run<const N: usize>(&self, args: [&str; N]) -> Result<String> {
        let mut command = Command::new(&self.binary);
        command
            .args(args)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());

        let child = command
            .spawn()
            .with_context(|| format!("spawn `{}`", self.binary.display()))?;
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
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct Message {
    pub(crate) id: String,
    pub(crate) sender_username: String,
    pub(crate) text: String,
}

pub(crate) struct MessageWatch {
    child: Child,
    messages_rx: mpsc::UnboundedReceiver<Message>,
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
            eprintln!("[agent-message] failed to signal watch shutdown: {error}");
        }
        let _ = self.child.wait().await;
        Ok(())
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

fn parse_watch_event(line: &str) -> Result<Option<Message>> {
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
        text: watch_message_text(&message),
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
    attachment_url: Option<String>,
    deleted: bool,
}

#[derive(Debug, Deserialize)]
struct WatchJSONSender {
    username: String,
}

fn watch_message_text(message: &WatchJSONMessage) -> String {
    if message.deleted {
        return "deleted message".to_string();
    }
    if message.kind.trim() == "json_render" {
        return "[json-render]".to_string();
    }
    if let Some(content) = &message.content {
        let content = content.trim();
        if !content.is_empty() {
            return content.to_string();
        }
    }
    if let Some(attachment_url) = &message.attachment_url {
        let attachment_url = attachment_url.trim();
        if !attachment_url.is_empty() {
            return attachment_url.to_string();
        }
    }
    String::new()
}

fn spawn_watch_stdout_pump(stdout: ChildStdout, messages_tx: mpsc::UnboundedSender<Message>) {
    tokio::spawn(async move {
        let mut lines = BufReader::new(stdout).lines();
        loop {
            match lines.next_line().await {
                Ok(Some(line)) => match parse_watch_event(&line) {
                    Ok(Some(message)) => {
                        if messages_tx.send(message).is_err() {
                            break;
                        }
                    }
                    Ok(None) => {}
                    Err(error) => eprintln!("[agent-message] invalid watch event: {error:#}"),
                },
                Ok(None) => break,
                Err(error) => {
                    eprintln!("[agent-message] failed to read watch stdout: {error}");
                    break;
                }
            }
        }
    });
}

fn spawn_watch_stderr_pump(stderr: ChildStderr) {
    tokio::spawn(async move {
        let mut lines = BufReader::new(stderr).lines();
        loop {
            match lines.next_line().await {
                Ok(Some(line)) => eprintln!("[agent-message] {line}"),
                Ok(None) => break,
                Err(error) => {
                    eprintln!("[agent-message] failed to read watch stderr: {error}");
                    break;
                }
            }
        }
    });
}
