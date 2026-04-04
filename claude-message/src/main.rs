mod agent_message;
mod app;
mod claude;
mod render;

use std::path::PathBuf;

use anyhow::Context;
use app::App;
use clap::{CommandFactory, Parser};

#[derive(Debug, Clone, clap::ValueEnum)]
enum PermissionModeArg {
    AcceptEdits,
    BypassPermissions,
    Default,
    DontAsk,
    Plan,
    Auto,
}

impl PermissionModeArg {
    fn as_claude_value(&self) -> &'static str {
        match self {
            Self::AcceptEdits => "acceptEdits",
            Self::BypassPermissions => "bypassPermissions",
            Self::Default => "default",
            Self::DontAsk => "dontAsk",
            Self::Plan => "plan",
            Self::Auto => "auto",
        }
    }
}

#[derive(Debug, Clone, Parser)]
#[command(author, version = env!("APP_VERSION"), about)]
struct Cli {
    #[arg(
        long = "to",
        env = "CLAUDE_MESSAGE_TO",
        help = "Recipient username; defaults to `agent-message config get master`"
    )]
    to_username: Option<String>,

    #[arg(long, env = "CLAUDE_MESSAGE_CLAUDE_BIN", default_value = "claude")]
    claude_bin: PathBuf,

    #[arg(long, env = "CLAUDE_MESSAGE_MODEL")]
    model: Option<String>,

    #[arg(long, env = "CLAUDE_MESSAGE_CWD")]
    cwd: Option<PathBuf>,

    #[arg(long, env = "CLAUDE_MESSAGE_PERMISSION_MODE")]
    permission_mode: Option<PermissionModeArg>,

    #[arg(long, env = "CLAUDE_MESSAGE_ALLOWED_TOOLS", value_delimiter = ',')]
    allowed_tools: Vec<String>,

    #[arg(long, env = "CLAUDE_MESSAGE_BARE", default_value_t = false)]
    bare: bool,

    #[arg(long, env = "CLAUDE_MESSAGE_TIMEOUT_SECS", default_value_t = 1800)]
    timeout_secs: u64,
}

#[derive(Debug, Clone)]
pub(crate) struct Config {
    pub(crate) to_username: Option<String>,
    pub(crate) claude_bin: PathBuf,
    pub(crate) model: Option<String>,
    pub(crate) cwd: PathBuf,
    pub(crate) permission_mode: Option<String>,
    pub(crate) allowed_tools: Vec<String>,
    pub(crate) bare: bool,
    pub(crate) timeout_secs: u64,
}

impl TryFrom<Cli> for Config {
    type Error = anyhow::Error;

    fn try_from(value: Cli) -> Result<Self, Self::Error> {
        let cwd = match value.cwd {
            Some(path) => path,
            None => std::env::current_dir().context("resolve current working directory")?,
        };

        Ok(Self {
            to_username: value.to_username,
            claude_bin: value.claude_bin,
            model: value.model,
            cwd,
            permission_mode: value
                .permission_mode
                .map(|mode| mode.as_claude_value().to_string()),
            allowed_tools: value.allowed_tools,
            bare: value.bare,
            timeout_secs: value.timeout_secs,
        })
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let config = Config::try_from(Cli::parse())?;
    App::new(config).run().await
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn to_username_defaults_to_none() {
        let cli = Cli::parse_from(["claude-message"]);
        assert_eq!(cli.to_username, None);
    }

    #[test]
    fn explicit_to_username_is_preserved() {
        let cli = Cli::parse_from(["claude-message", "--to", "alice"]);
        assert_eq!(cli.to_username.as_deref(), Some("alice"));
    }

    #[test]
    fn help_mentions_to_flag() {
        let help = Cli::command().render_long_help().to_string();
        assert!(help.contains("--to"));
        assert!(help.contains("agent-message config get master"));
    }
}
