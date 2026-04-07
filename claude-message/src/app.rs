use anyhow::{Context, Result};
use rand::Rng;
use std::time::Duration;
use uuid::Uuid;

use crate::Config;
use crate::agent_message::{AgentMessageClient, Message, MessageWatch};
use crate::claude::{ClaudeRunResult, ClaudeRunner};
use crate::log_ui::LogUi;
use crate::render::response_spec;

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
        "claude-message session started\nchat_id: {chat_id}\nusername: {username}\npassword: {password}\nCWD: {}\nHostname: {hostname}\n\nReply in this DM to run Claude Code.",
        cwd.display(),
    )
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
    to_username: String,
    agent_client: AgentMessageClient,
    message_watch: MessageWatch,
    claude: ClaudeRunner,
    session_id: Option<String>,
    logger: LogUi,
}

impl Runtime {
    async fn bootstrap(config: Config) -> Result<Self> {
        let logger = LogUi::new("claude-message");
        let chat_id = new_chat_id();
        let username = format!("agent-{chat_id}");
        let password = new_password();
        let mut agent_client =
            AgentMessageClient::new(std::path::PathBuf::from("agent-message"), logger.clone());
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

        register_agent_account(&agent_client, &username, &password).await?;
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
        logger.success(
            "Startup message sent",
            [
                format!("Recipient: @{to_username}"),
                format!("Message: {startup_message_id}"),
            ],
        );

        let claude = ClaudeRunner::new(&config);
        logger.success(
            "Claude runner ready",
            [format!("CWD: {}", config.cwd.display())],
        );

        Ok(Self {
            to_username,
            agent_client,
            message_watch,
            claude,
            session_id: None,
            logger,
        })
    }

    async fn run_loop(&mut self) -> Result<()> {
        loop {
            tokio::select! {
                _ = tokio::signal::ctrl_c() => {
                    self.message_watch.shutdown().await?;
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
                                format!("Request: {}", request_preview(&request)),
                                format!("Success: {}", outcome.success),
                            ];
                            if let Some(session_id) = outcome.session_id.as_deref() {
                                lines.push(format!("Session: {session_id}"));
                            }
                            if let Some(status) = outcome.status.as_deref() {
                                lines.push(format!("Status: {status}"));
                            }
                            if outcome.success {
                                self.logger.success("Turn finished", lines);
                            } else {
                                self.logger.warning("Turn finished", lines);
                            }
                            if outcome.success {
                                self.mark_message_complete(&message).await;
                            }
                        }
                        Err(error) => {
                            self.logger.error(
                                "Claude request failed",
                                [
                                    format!("Request: {}", request_preview(&request)),
                                    format!("Error: {error:#}"),
                                ],
                            );
                            let spec = response_spec(
                                "Failed",
                                "destructive",
                                "Claude request failed",
                                &[format!("Error: {error:#}")],
                                None,
                            );
                            let _ = self
                                .agent_client
                                .send_json_render_message(&self.to_username, spec)
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
        self.logger.turn(
            "Turn started",
            [format!("Request: {}", request_preview(request))],
        );
        let response = self
            .claude
            .run(request.trim(), self.session_id.as_deref())
            .await
            .context("run claude turn")?;

        if let Some(session_id) = &response.session_id {
            self.session_id = Some(session_id.clone());
        }

        let success = response.is_success();
        let spec = spec_for_turn(&response);
        self.agent_client
            .send_json_render_message(&self.to_username, spec)
            .await
            .context("send claude result message")?;

        Ok(TurnOutcome {
            success,
            session_id: response.session_id.clone(),
            status: response.subtype.clone(),
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

#[derive(Debug)]
struct TurnOutcome {
    success: bool,
    session_id: Option<String>,
    status: Option<String>,
}

fn spec_for_turn(response: &ClaudeRunResult) -> serde_json::Value {
    let error_body = response.error_text();
    let (badge_text, badge_variant, title, body) = if response.is_success() {
        (
            "Completed",
            "default",
            "Claude finished the request",
            Some(response.result_text.as_str()),
        )
    } else {
        (
            "Failed",
            "destructive",
            "Claude could not complete the request",
            error_body.as_deref(),
        )
    };

    response_spec(
        badge_text,
        badge_variant,
        title,
        &response.summary_lines(),
        body,
    )
}

async fn register_agent_account(
    client: &AgentMessageClient,
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

fn request_preview(text: &str) -> String {
    const LIMIT: usize = 160;

    let compact = text.split_whitespace().collect::<Vec<_>>().join(" ");
    let mut preview = compact.chars().take(LIMIT).collect::<String>();
    if compact.chars().count() > LIMIT {
        preview.push_str("...");
    }
    preview
}

fn new_chat_id() -> String {
    Uuid::new_v4().simple().to_string()[..12].to_string()
}

fn new_password() -> String {
    let mut rng = rand::rng();
    format!("{:06}", rng.random_range(0..=999_999))
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
    use crate::claude::ClaudeUsage;

    #[test]
    fn watch_retry_delay_caps_at_last_value() {
        assert_eq!(watch_retry_delay(0), Duration::from_secs(1));
        assert_eq!(watch_retry_delay(1), Duration::from_secs(2));
        assert_eq!(watch_retry_delay(2), Duration::from_secs(5));
        assert_eq!(watch_retry_delay(10), Duration::from_secs(5));
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

    #[test]
    fn spec_for_turn_uses_markdown_body_on_success() {
        let spec = spec_for_turn(&ClaudeRunResult {
            session_id: Some("sess-1".to_string()),
            subtype: Some("success".to_string()),
            result_text: "## Done".to_string(),
            errors: Vec::new(),
            usage: Some(ClaudeUsage {
                input_tokens: Some(1),
                output_tokens: Some(2),
                total_cost_usd: Some(0.01),
            }),
        });

        assert_eq!(spec["elements"]["body"]["type"], "Markdown");
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
}
