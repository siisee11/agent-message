mod agent_message;
mod app;
mod codex;
mod render;

use std::path::PathBuf;

use anyhow::Context;
use app::App;
use clap::Parser;

#[derive(Debug, Clone, clap::ValueEnum)]
enum ApprovalPolicyArg {
    Untrusted,
    OnFailure,
    OnRequest,
    Never,
}

impl ApprovalPolicyArg {
    fn as_app_server_value(&self) -> &'static str {
        match self {
            Self::Untrusted => "untrusted",
            Self::OnFailure => "on-failure",
            Self::OnRequest => "on-request",
            Self::Never => "never",
        }
    }
}

#[derive(Debug, Clone, clap::ValueEnum)]
enum SandboxArg {
    ReadOnly,
    WorkspaceWrite,
    DangerFullAccess,
}

#[derive(Debug, Clone, Parser)]
#[command(author, version = env!("APP_VERSION"), about)]
struct Cli {
    #[arg(long = "to", env = "CODEX_MESSAGE_TO", default_value = "jay")]
    to_username: String,

    #[arg(long, env = "CODEX_MESSAGE_CODEX_BIN", default_value = "codex")]
    codex_bin: PathBuf,

    #[arg(long, env = "CODEX_MESSAGE_MODEL")]
    model: Option<String>,

    #[arg(long, env = "CODEX_MESSAGE_CWD")]
    cwd: Option<PathBuf>,

    #[arg(long, env = "CODEX_MESSAGE_APPROVAL_POLICY")]
    approval_policy: Option<ApprovalPolicyArg>,

    #[arg(long, env = "CODEX_MESSAGE_SANDBOX", default_value = "workspace-write")]
    sandbox: SandboxArg,

    #[arg(long, env = "CODEX_MESSAGE_NETWORK_ACCESS", default_value_t = false)]
    network_access: bool,
}

#[derive(Debug, Clone)]
pub(crate) struct Config {
    pub(crate) to_username: String,
    pub(crate) codex_bin: PathBuf,
    pub(crate) model: Option<String>,
    pub(crate) cwd: PathBuf,
    pub(crate) approval_policy: Option<String>,
    pub(crate) sandbox: SandboxArg,
    pub(crate) network_access: bool,
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
            codex_bin: value.codex_bin,
            model: value.model,
            cwd,
            approval_policy: value
                .approval_policy
                .map(|policy| policy.as_app_server_value().to_string()),
            sandbox: value.sandbox,
            network_access: value.network_access,
        })
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let config = Config::try_from(Cli::parse())?;
    App::new(config).run().await
}
