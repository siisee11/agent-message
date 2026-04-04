mod agent_message;
mod app;
mod codex;
mod render;

use std::path::PathBuf;

use anyhow::Context;
use app::App;
use clap::Parser;

#[derive(Debug, Clone, Copy, PartialEq, Eq, clap::ValueEnum)]
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

#[derive(Debug, Clone, Copy, PartialEq, Eq, clap::ValueEnum)]
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

    #[arg(
        long,
        env = "CODEX_MESSAGE_YOLO",
        default_value_t = false,
        conflicts_with = "approval_policy",
        conflicts_with = "sandbox",
        help = "Run with --approval-policy never and --sandbox danger-full-access"
    )]
    yolo: bool,
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
        let (approval_policy, sandbox) = resolve_execution_mode(&value);
        let cwd = match value.cwd {
            Some(path) => path,
            None => std::env::current_dir().context("resolve current working directory")?,
        };

        Ok(Self {
            to_username: value.to_username,
            codex_bin: value.codex_bin,
            model: value.model,
            cwd,
            approval_policy: approval_policy.map(|policy| policy.as_app_server_value().to_string()),
            sandbox,
            network_access: value.network_access,
        })
    }
}

fn resolve_execution_mode(value: &Cli) -> (Option<ApprovalPolicyArg>, SandboxArg) {
    if value.yolo {
        return (Some(ApprovalPolicyArg::Never), SandboxArg::DangerFullAccess);
    }

    (value.approval_policy, value.sandbox)
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
    fn yolo_sets_never_and_danger_full_access() {
        let cli = Cli::parse_from(["codex-message", "--yolo"]);
        let (approval_policy, sandbox) = resolve_execution_mode(&cli);

        assert_eq!(approval_policy, Some(ApprovalPolicyArg::Never));
        assert_eq!(sandbox, SandboxArg::DangerFullAccess);
    }

    #[test]
    fn yolo_conflicts_with_manual_execution_flags() {
        let error = Cli::try_parse_from([
            "codex-message",
            "--yolo",
            "--approval-policy",
            "on-request",
        ])
        .expect_err("expected clap conflict");

        let rendered = error.to_string();
        assert!(rendered.contains("--yolo"));
        assert!(rendered.contains("--approval-policy"));

        let error = Cli::try_parse_from(["codex-message", "--yolo", "--sandbox", "read-only"])
            .expect_err("expected clap conflict");

        let rendered = error.to_string();
        assert!(rendered.contains("--yolo"));
        assert!(rendered.contains("--sandbox"));
    }

    #[test]
    fn help_mentions_yolo() {
        let help = Cli::command().render_long_help().to_string();
        assert!(help.contains("--yolo"));
        assert!(help.contains("danger-full-access"));
    }
}
