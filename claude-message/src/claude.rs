use std::path::PathBuf;
use std::process::Stdio;
use std::time::Duration;

use anyhow::{Context, Result, anyhow, bail};
use serde_json::Value;
use tokio::process::Command;

use crate::Config;

#[derive(Debug, Clone)]
pub(crate) struct ClaudeRunner {
    binary: PathBuf,
    cwd: PathBuf,
    model: Option<String>,
    permission_mode: Option<String>,
    allowed_tools: Vec<String>,
    bare: bool,
    timeout_secs: u64,
}

#[derive(Debug, Clone)]
pub(crate) struct ClaudeRunResult {
    pub(crate) session_id: Option<String>,
    pub(crate) subtype: Option<String>,
    pub(crate) result_text: String,
    pub(crate) errors: Vec<String>,
    pub(crate) usage: Option<ClaudeUsage>,
}

#[derive(Debug, Clone, PartialEq)]
pub(crate) struct ClaudeUsage {
    pub(crate) input_tokens: Option<u64>,
    pub(crate) output_tokens: Option<u64>,
    pub(crate) total_cost_usd: Option<f64>,
}

impl ClaudeRunResult {
    pub(crate) fn is_success(&self) -> bool {
        self.errors.is_empty()
            && !matches!(
                self.subtype.as_deref(),
                Some(subtype) if subtype.starts_with("error")
            )
    }

    pub(crate) fn summary_lines(&self) -> Vec<String> {
        let mut lines = Vec::new();
        if let Some(session_id) = &self.session_id {
            lines.push(format!("Session: {session_id}"));
        }
        if let Some(subtype) = &self.subtype {
            lines.push(format!("Status: {subtype}"));
        }
        if let Some(usage) = &self.usage {
            let mut usage_parts = Vec::new();
            if let Some(input_tokens) = usage.input_tokens {
                usage_parts.push(format!("input {input_tokens}"));
            }
            if let Some(output_tokens) = usage.output_tokens {
                usage_parts.push(format!("output {output_tokens}"));
            }
            if !usage_parts.is_empty() {
                lines.push(format!("Tokens: {}", usage_parts.join(", ")));
            }
            if let Some(total_cost_usd) = usage.total_cost_usd {
                lines.push(format!("Cost: ${total_cost_usd:.4}"));
            }
        }
        lines
    }

    pub(crate) fn error_text(&self) -> Option<String> {
        if self.errors.is_empty() {
            None
        } else {
            Some(self.errors.join("\n"))
        }
    }
}

impl ClaudeRunner {
    pub(crate) fn new(config: &Config) -> Self {
        Self {
            binary: config.claude_bin.clone(),
            cwd: config.cwd.clone(),
            model: config.model.clone(),
            permission_mode: config.permission_mode.clone(),
            allowed_tools: config.allowed_tools.clone(),
            bare: config.bare,
            timeout_secs: config.timeout_secs,
        }
    }

    pub(crate) async fn run(
        &self,
        prompt: &str,
        resume_session_id: Option<&str>,
    ) -> Result<ClaudeRunResult> {
        let mut command = Command::new(&self.binary);
        command
            .current_dir(&self.cwd)
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());

        if self.bare {
            command.arg("--bare");
        }
        if let Some(model) = &self.model {
            command.args(["--model", model]);
        }
        if let Some(permission_mode) = &self.permission_mode {
            command.args(["--permission-mode", permission_mode]);
        }
        command.arg("--dangerously-skip-permissions");
        if !self.allowed_tools.is_empty() {
            command.arg("--allowed-tools");
            for tool in &self.allowed_tools {
                command.arg(tool);
            }
        }
        if let Some(session_id) = resume_session_id {
            command.args(["--resume", session_id]);
        }

        command.args(["-p", "--output-format", "json", prompt]);

        let child = command
            .spawn()
            .with_context(|| format!("spawn `{}`", self.binary.display()))?;
        let output = tokio::time::timeout(
            Duration::from_secs(self.timeout_secs),
            child.wait_with_output(),
        )
        .await
        .with_context(|| format!("claude command timed out after {}s", self.timeout_secs))?
        .context("wait for claude command")?;

        let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();

        if output.status.success() {
            return parse_success_output(&stdout);
        }

        if let Ok(parsed) = parse_json_output(&stdout) {
            return Ok(parsed);
        }

        let detail = if !stderr.is_empty() {
            stderr
        } else if !stdout.is_empty() {
            stdout
        } else {
            format!("claude exited with status {}", output.status)
        };
        Err(anyhow!("claude command failed: {detail}"))
    }
}

fn parse_success_output(stdout: &str) -> Result<ClaudeRunResult> {
    if stdout.trim().is_empty() {
        bail!("claude returned an empty JSON response");
    }
    parse_json_output(stdout)
}

fn parse_json_output(stdout: &str) -> Result<ClaudeRunResult> {
    let value: Value = serde_json::from_str(stdout).context("decode claude JSON output")?;
    ClaudeRunResult::try_from(value)
}

impl TryFrom<Value> for ClaudeRunResult {
    type Error = anyhow::Error;

    fn try_from(value: Value) -> Result<Self, Self::Error> {
        let session_id = value
            .get("session_id")
            .and_then(Value::as_str)
            .map(ToOwned::to_owned);
        let subtype = value
            .get("subtype")
            .and_then(Value::as_str)
            .map(ToOwned::to_owned);
        let result_text = value
            .get("result")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_string();

        let mut errors = collect_errors(&value);
        if value
            .get("is_error")
            .and_then(Value::as_bool)
            .unwrap_or(false)
            && errors.is_empty()
        {
            errors.push(
                subtype
                    .clone()
                    .unwrap_or_else(|| "Claude returned an unspecified error".to_string()),
            );
        }

        let usage = parse_usage(value.get("usage"), value.get("total_cost_usd"))?;

        Ok(Self {
            session_id,
            subtype,
            result_text,
            errors,
            usage,
        })
    }
}

fn collect_errors(value: &Value) -> Vec<String> {
    value
        .get("errors")
        .and_then(Value::as_array)
        .map(|items| {
            items
                .iter()
                .filter_map(|item| match item {
                    Value::String(text) => Some(text.trim().to_string()),
                    other => Some(other.to_string()),
                })
                .filter(|text| !text.is_empty())
                .collect()
        })
        .unwrap_or_default()
}

fn parse_usage(
    usage_value: Option<&Value>,
    total_cost_value: Option<&Value>,
) -> Result<Option<ClaudeUsage>> {
    let input_tokens = usage_value.and_then(|value| parse_optional_u64(value.get("input_tokens")));
    let output_tokens =
        usage_value.and_then(|value| parse_optional_u64(value.get("output_tokens")));
    let total_cost_usd = parse_optional_f64(total_cost_value);

    if input_tokens.is_none() && output_tokens.is_none() && total_cost_usd.is_none() {
        return Ok(None);
    }

    Ok(Some(ClaudeUsage {
        input_tokens,
        output_tokens,
        total_cost_usd,
    }))
}

fn parse_optional_u64(value: Option<&Value>) -> Option<u64> {
    value.and_then(|item| match item {
        Value::Number(number) => number.as_u64(),
        _ => None,
    })
}

fn parse_optional_f64(value: Option<&Value>) -> Option<f64> {
    value.and_then(|item| match item {
        Value::Number(number) => number.as_f64(),
        _ => None,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn parses_success_output() {
        let parsed = ClaudeRunResult::try_from(json!({
            "session_id": "sess-123",
            "subtype": "success",
            "result": "Hello",
            "is_error": false,
            "usage": {
                "input_tokens": 10,
                "output_tokens": 20
            },
            "total_cost_usd": 0.1234
        }))
        .expect("parse success output");

        assert_eq!(parsed.session_id.as_deref(), Some("sess-123"));
        assert_eq!(parsed.result_text, "Hello");
        assert!(parsed.is_success());
    }

    #[test]
    fn parses_error_output() {
        let parsed = ClaudeRunResult::try_from(json!({
            "session_id": "sess-err",
            "subtype": "error_max_turns",
            "is_error": true,
            "errors": ["Max turns exceeded"]
        }))
        .expect("parse error output");

        assert!(!parsed.is_success());
        assert_eq!(parsed.error_text().as_deref(), Some("Max turns exceeded"));
    }

    #[test]
    fn summary_lines_include_usage() {
        let result = ClaudeRunResult {
            session_id: Some("sess-1".to_string()),
            subtype: Some("success".to_string()),
            result_text: "Done".to_string(),
            errors: Vec::new(),
            usage: Some(ClaudeUsage {
                input_tokens: Some(12),
                output_tokens: Some(34),
                total_cost_usd: Some(0.01),
            }),
        };

        let lines = result.summary_lines();
        assert!(lines.iter().any(|line| line.contains("Session: sess-1")));
        assert!(
            lines
                .iter()
                .any(|line| line.contains("Tokens: input 12, output 34"))
        );
        assert!(lines.iter().any(|line| line.contains("Cost: $0.0100")));
    }
}
