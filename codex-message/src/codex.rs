use std::collections::HashMap;
use std::path::Path;
use std::process::Stdio;
use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};

use anyhow::{Context, Result, anyhow, bail};
use serde_json::Value;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::process::{Child, ChildStdin, ChildStdout, Command};
use tokio::sync::{Mutex, mpsc, oneshot};

#[derive(Debug)]
pub(crate) enum IncomingMessage {
    Request {
        method: String,
        id: Value,
        params: Value,
    },
    Notification {
        method: String,
        params: Value,
    },
}

#[derive(Debug, Clone)]
pub(crate) struct RpcError {
    pub(crate) code: Option<i64>,
    pub(crate) message: String,
}

impl std::fmt::Display for RpcError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self.code {
            Some(code) => write!(f, "rpc error {code}: {}", self.message),
            None => write!(f, "rpc error: {}", self.message),
        }
    }
}

impl std::error::Error for RpcError {}

pub(crate) struct CodexAppServer {
    child: Child,
    stdin: Arc<Mutex<ChildStdin>>,
    next_request_id: AtomicU64,
    pending: Arc<Mutex<HashMap<String, oneshot::Sender<Result<Value, RpcError>>>>>,
    events_rx: mpsc::UnboundedReceiver<IncomingMessage>,
}

impl CodexAppServer {
    pub(crate) async fn start(codex_bin: &Path, cwd: &Path) -> Result<Self> {
        let mut child = Command::new(codex_bin)
            .arg("app-server")
            .current_dir(cwd)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .kill_on_drop(true)
            .spawn()
            .with_context(|| format!("spawn `{}`", codex_bin.display()))?;

        let stdin = child.stdin.take().context("capture codex stdin")?;
        let stdout = child.stdout.take().context("capture codex stdout")?;
        let stderr = child.stderr.take().context("capture codex stderr")?;

        let pending = Arc::new(Mutex::new(HashMap::new()));
        let (events_tx, events_rx) = mpsc::unbounded_channel();

        spawn_stdout_pump(stdout, Arc::clone(&pending), events_tx);
        spawn_stderr_pump(stderr);

        Ok(Self {
            child,
            stdin: Arc::new(Mutex::new(stdin)),
            next_request_id: AtomicU64::new(1),
            pending,
            events_rx,
        })
    }

    pub(crate) async fn initialize(&self) -> Result<()> {
        let initialize = serde_json::json!({
            "clientInfo": {
                "name": "codex_message",
                "title": "Codex Message",
                "version": env!("CARGO_PKG_VERSION"),
            },
            "capabilities": {
                "experimentalApi": true,
            }
        });
        self.request("initialize", initialize).await?;
        self.notify("initialized", serde_json::json!({})).await
    }

    pub(crate) async fn request(&self, method: &str, params: Value) -> Result<Value> {
        let id = self.next_request_id.fetch_add(1, Ordering::SeqCst);
        let request_id = Value::from(id);
        let key = id_key(&request_id)?;
        let (tx, rx) = oneshot::channel();
        self.pending.lock().await.insert(key, tx);
        self.write_message(serde_json::json!({
            "id": request_id,
            "method": method,
            "params": params,
        }))
        .await?;

        match rx.await {
            Ok(Ok(result)) => Ok(result),
            Ok(Err(error)) => Err(error.into()),
            Err(_) => bail!("rpc response channel dropped for method `{method}`"),
        }
    }

    pub(crate) async fn notify(&self, method: &str, params: Value) -> Result<()> {
        self.write_message(serde_json::json!({
            "method": method,
            "params": params,
        }))
        .await
    }

    pub(crate) async fn respond(&self, id: Value, result: Value) -> Result<()> {
        self.write_message(serde_json::json!({
            "id": id,
            "result": result,
        }))
        .await
    }

    pub(crate) async fn next_event(&mut self) -> Result<IncomingMessage> {
        self.events_rx
            .recv()
            .await
            .ok_or_else(|| anyhow!("codex app-server event stream ended"))
    }

    pub(crate) async fn shutdown(&mut self) -> Result<()> {
        if let Err(error) = self.child.start_kill() {
            eprintln!("[codex] failed to signal shutdown: {error}");
        }
        let _ = self.child.wait().await;
        Ok(())
    }

    async fn write_message(&self, message: Value) -> Result<()> {
        let encoded = serde_json::to_vec(&message).context("encode rpc message")?;
        let mut stdin = self.stdin.lock().await;
        stdin
            .write_all(&encoded)
            .await
            .context("write rpc payload to codex stdin")?;
        stdin
            .write_all(b"\n")
            .await
            .context("write rpc newline to codex stdin")?;
        stdin.flush().await.context("flush codex stdin")
    }
}

fn spawn_stdout_pump(
    stdout: ChildStdout,
    pending: Arc<Mutex<HashMap<String, oneshot::Sender<Result<Value, RpcError>>>>>,
    events_tx: mpsc::UnboundedSender<IncomingMessage>,
) {
    tokio::spawn(async move {
        let mut lines = BufReader::new(stdout).lines();
        loop {
            let next = lines.next_line().await;
            let line = match next {
                Ok(Some(line)) => line,
                Ok(None) => break,
                Err(error) => {
                    eprintln!("[codex] failed to read stdout: {error}");
                    break;
                }
            };
            let trimmed = line.trim();
            if trimmed.is_empty() {
                continue;
            }

            let value: Value = match serde_json::from_str(trimmed) {
                Ok(value) => value,
                Err(error) => {
                    eprintln!("[codex] invalid JSON from app-server: {error}: {trimmed}");
                    continue;
                }
            };

            if let Some(id) = value.get("id") {
                if value.get("method").is_some() {
                    let Some(method) = value.get("method").and_then(Value::as_str) else {
                        eprintln!("[codex] request missing method string: {trimmed}");
                        continue;
                    };
                    let params = value.get("params").cloned().unwrap_or(Value::Null);
                    if events_tx
                        .send(IncomingMessage::Request {
                            method: method.to_string(),
                            id: id.clone(),
                            params,
                        })
                        .is_err()
                    {
                        break;
                    }
                    continue;
                }

                let key = match id_key(id) {
                    Ok(key) => key,
                    Err(error) => {
                        eprintln!("[codex] invalid response id: {error:#}");
                        continue;
                    }
                };
                let sender = pending.lock().await.remove(&key);
                let Some(sender) = sender else {
                    eprintln!("[codex] dropped response for unknown request id: {trimmed}");
                    continue;
                };

                if let Some(result) = value.get("result") {
                    let _ = sender.send(Ok(result.clone()));
                    continue;
                }

                let error = value.get("error").cloned().unwrap_or(Value::Null);
                let rpc_error = RpcError {
                    code: error.get("code").and_then(Value::as_i64),
                    message: error
                        .get("message")
                        .and_then(Value::as_str)
                        .unwrap_or("unknown rpc error")
                        .to_string(),
                };
                let _ = sender.send(Err(rpc_error));
                continue;
            }

            if let Some(method) = value.get("method").and_then(Value::as_str) {
                let params = value.get("params").cloned().unwrap_or(Value::Null);
                if events_tx
                    .send(IncomingMessage::Notification {
                        method: method.to_string(),
                        params,
                    })
                    .is_err()
                {
                    break;
                }
            }
        }
    });
}

fn spawn_stderr_pump(stderr: tokio::process::ChildStderr) {
    tokio::spawn(async move {
        let mut lines = BufReader::new(stderr).lines();
        loop {
            match lines.next_line().await {
                Ok(Some(line)) => eprintln!("[codex] {line}"),
                Ok(None) => break,
                Err(error) => {
                    eprintln!("[codex] failed to read stderr: {error}");
                    break;
                }
            }
        }
    });
}

fn id_key(id: &Value) -> Result<String> {
    serde_json::to_string(id).context("serialize request id")
}
