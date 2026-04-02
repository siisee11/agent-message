use std::path::PathBuf;
use std::process::Stdio;
use std::time::Duration;

use anyhow::{Context, Result, anyhow, bail};
use serde_json::Value;
use tokio::process::Command;

#[derive(Debug, Clone)]
pub(crate) struct AgentMessageClient {
    binary: PathBuf,
}

impl AgentMessageClient {
    pub(crate) fn new(binary: PathBuf) -> Self {
        Self { binary }
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

    pub(crate) async fn read_messages(&self, username: &str, limit: usize) -> Result<Vec<Message>> {
        let limit_string = limit.to_string();
        let output = self
            .run(["read", username, "-n", &limit_string])
            .await
            .context("run `agent-message read`")?;
        parse_read_output(&output)
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

fn parse_read_output(output: &str) -> Result<Vec<Message>> {
    let mut messages = Vec::new();
    let mut current: Option<Message> = None;

    for raw_line in output.lines() {
        let line = raw_line.trim_end();
        if line.trim().is_empty() {
            continue;
        }

        if line.starts_with('[') {
            if let Some(message) = current.take() {
                messages.push(message);
            }
            current = Some(parse_read_line(line)?);
            continue;
        }

        let Some(message) = current.as_mut() else {
            bail!("unexpected read line: {line}");
        };
        message.text.push('\n');
        message.text.push_str(line);
    }

    if let Some(message) = current {
        messages.push(message);
    }

    Ok(messages)
}

fn parse_read_line(line: &str) -> Result<Message> {
    let Some(after_index) = line.split_once("] ").map(|(_, rest)| rest) else {
        bail!("unexpected read line: {line}");
    };
    let Some((message_id, rest)) = after_index.split_once(' ') else {
        bail!("unexpected read line missing message id: {line}");
    };
    let Some((sender, text)) = rest.split_once(": ") else {
        bail!("unexpected read line missing sender/text separator: {line}");
    };
    if message_id.trim().is_empty() || sender.trim().is_empty() {
        bail!("unexpected read line with empty fields: {line}");
    }

    Ok(Message {
        id: message_id.trim().to_string(),
        sender_username: sender.trim().to_string(),
        text: text.to_string(),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_read_line() {
        let parsed = parse_read_line("[1] m-123 jay: hello world").expect("parse line");
        assert_eq!(
            parsed,
            Message {
                id: "m-123".to_string(),
                sender_username: "jay".to_string(),
                text: "hello world".to_string(),
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
    fn parses_multiline_read_output() {
        let output = "\
[1] m-123 jay: first line
chat_id: abc123
pin: 654321

[2] m-124 jay: second message";

        let messages = parse_read_output(output).expect("parse output");

        assert_eq!(messages.len(), 2);
        assert_eq!(messages[0].id, "m-123");
        assert_eq!(messages[0].sender_username, "jay");
        assert_eq!(messages[0].text, "first line\nchat_id: abc123\npin: 654321");
        assert_eq!(messages[1].id, "m-124");
        assert_eq!(messages[1].text, "second message");
    }
}
