use anyhow::{Context, Result};
use rand::Rng;
use std::time::Duration;
use uuid::Uuid;

use crate::Config;
use crate::agent_message::{AgentMessageClient, Message, MessageWatch};
use crate::claude::{ClaudeRunResult, ClaudeRunner};
use crate::render::response_spec;

const READ_REACTION_EMOJI: &str = "👀";
const COMPLETE_REACTION_EMOJI: &str = "✅";
const WATCH_RETRY_DELAYS_SECS: [u64; 3] = [1, 2, 5];

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
}

impl Runtime {
    async fn bootstrap(config: Config) -> Result<Self> {
        let chat_id = new_chat_id();
        let username = format!("agent-{chat_id}");
        let password = new_password();
        let mut agent_client = AgentMessageClient::new(std::path::PathBuf::from("agent-message"));
        let to_username = resolve_target_username(&config, &agent_client).await?;
        let server_url = agent_client
            .server_url()
            .await
            .context("read agent-message server_url")?;

        println!("agent-message server_url: {server_url}");

        register_agent_account(&agent_client, &username, &password).await?;
        agent_client.set_from_profile(username.clone());
        println!("registered agent profile: {username} (chat_id: {chat_id})");

        let message_watch = agent_client
            .watch_messages(&to_username)
            .await
            .context("start agent-message watch stream")?;
        println!("agent-message watch stream ready for {to_username}");

        let startup_text = format!(
            "claude-message session started\nchat_id: {chat_id}\nusername: {username}\npassword: {password}\n\nReply in this DM to run Claude Code."
        );
        let startup_message_id = agent_client
            .send_text_message(&to_username, &startup_text)
            .await
            .context("send startup message")?;
        println!("startup message sent to {to_username}: {startup_message_id}");

        let claude = ClaudeRunner::new(&config);
        println!("claude-message ready in {}", config.cwd.display());

        Ok(Self {
            to_username,
            agent_client,
            message_watch,
            claude,
            session_id: None,
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

                    match self.run_turn(&request).await {
                        Ok(outcome) => {
                            if outcome.success {
                                self.mark_message_complete(&message).await;
                            }
                        }
                        Err(error) => {
                            eprintln!("claude request failed for {:?}: {error:#}", request.trim());
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
                eprintln!(
                    "warning: failed to add read reaction to {}: {error}",
                    message.id
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
        eprintln!(
            "warning: agent-message watch stream failed: {error:#}. retrying in {}s",
            delay.as_secs()
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
        eprintln!(
            "agent-message watch stream reconnected for {}",
            self.to_username
        );
        Ok(())
    }

    async fn run_turn(&mut self, request: &str) -> Result<TurnOutcome> {
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

        Ok(TurnOutcome { success })
    }

    async fn mark_message_complete(&self, message: &Message) {
        if let Err(error) = self
            .agent_client
            .unreact_to_message(message, READ_REACTION_EMOJI)
            .await
        {
            eprintln!(
                "warning: failed to remove read reaction from {}: {error}",
                message.id
            );
        }
        if let Err(error) = self
            .agent_client
            .react_to_message(message, COMPLETE_REACTION_EMOJI)
            .await
        {
            eprintln!(
                "warning: failed to add complete reaction to {}: {error}",
                message.id
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
}
