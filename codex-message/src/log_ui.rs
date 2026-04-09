use std::sync::Mutex;

static OUTPUT_LOCK: Mutex<()> = Mutex::new(());

const ANSI_RESET: &str = "\x1b[0m";
const ANSI_BOLD: &str = "\x1b[1m";
const ANSI_DIM: &str = "\x1b[2m";
const ANSI_CYAN: &str = "\x1b[36m";
const ANSI_BLUE: &str = "\x1b[34m";
const ANSI_GREEN: &str = "\x1b[32m";
const ANSI_YELLOW: &str = "\x1b[33m";
const ANSI_RED: &str = "\x1b[31m";
const ANSI_MAGENTA: &str = "\x1b[35m";
const ANSI_WHITE: &str = "\x1b[37m";
const ANSI_BRIGHT_BLACK: &str = "\x1b[90m";

#[derive(Debug, Clone)]
pub(crate) struct LogUi {
    app_name: &'static str,
}

#[derive(Debug, Clone, Copy)]
enum LogTone {
    System,
    Receive,
    Request,
    Turn,
    Send,
    Success,
    Warning,
    Error,
    Child,
}

impl LogUi {
    pub(crate) const fn new(app_name: &'static str) -> Self {
        Self { app_name }
    }

    pub(crate) fn system<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::System, title, lines);
    }

    pub(crate) fn request<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Request, title, lines);
    }

    pub(crate) fn recv<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Receive, title, lines);
    }

    pub(crate) fn turn<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Turn, title, lines);
    }

    pub(crate) fn send<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Send, title, lines);
    }

    pub(crate) fn success<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Success, title, lines);
    }

    pub(crate) fn warning<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Warning, title, lines);
    }

    pub(crate) fn error<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Error, title, lines);
    }

    pub(crate) fn child<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Child, title, lines);
    }

    fn print<S, I>(&self, tone: LogTone, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        let rendered = render_line(
            self.app_name,
            tone,
            title,
            &collect_lines(lines.into_iter().map(Into::into)),
        );
        let guard = match OUTPUT_LOCK.lock() {
            Ok(guard) => guard,
            Err(poisoned) => poisoned.into_inner(),
        };
        eprintln!("{rendered}");
        drop(guard);
    }
}

fn collect_lines(lines: impl IntoIterator<Item = String>) -> Vec<String> {
    lines
        .into_iter()
        .map(|line| compact_whitespace(&line.replace('\t', "  ")))
        .filter(|line| !line.is_empty())
        .collect()
}

fn render_line(app_name: &str, tone: LogTone, title: &str, lines: &[String]) -> String {
    let color = color_enabled();
    let separator = paint(" | ", ANSI_BRIGHT_BLACK, color);

    let mut parts = vec![
        paint(app_name, ANSI_CYAN, color),
        paint(tone.label(), tone.color_code(), color),
        paint(&compact_whitespace(title), ANSI_BOLD, color),
    ];

    parts.extend(lines.iter().map(|line| format_detail(line, color)));
    parts.join(&separator)
}

fn format_detail(line: &str, color: bool) -> String {
    if let Some((key, value)) = line.split_once(':') {
        let key = normalize_key(key);
        let value = quote_if_needed(value.trim());
        return format!(
            "{}={}",
            paint(&key, ANSI_DIM, color),
            paint(&value, ANSI_WHITE, color)
        );
    }

    paint(&quote_if_needed(line), ANSI_WHITE, color)
}

fn normalize_key(key: &str) -> String {
    key.trim()
        .chars()
        .map(|ch| match ch {
            'A'..='Z' => ch.to_ascii_lowercase(),
            'a'..='z' | '0'..='9' => ch,
            _ => '_',
        })
        .collect::<String>()
        .trim_matches('_')
        .to_string()
}

fn quote_if_needed(value: &str) -> String {
    let compact = compact_whitespace(value);
    if compact.is_empty() {
        return "\"\"".to_string();
    }
    if compact.chars().all(|ch| {
        ch.is_ascii_alphanumeric() || matches!(ch, '@' | '/' | '.' | '-' | '_' | ':' | '#')
    }) {
        return compact;
    }

    format!("\"{}\"", compact.replace('\\', "\\\\").replace('"', "\\\""))
}

fn compact_whitespace(value: &str) -> String {
    value.split_whitespace().collect::<Vec<_>>().join(" ")
}

fn paint(text: &str, code: &str, enabled: bool) -> String {
    if enabled {
        format!("{code}{text}{ANSI_RESET}")
    } else {
        text.to_string()
    }
}

fn color_enabled() -> bool {
    std::env::var_os("NO_COLOR").is_none()
}

impl LogTone {
    fn label(self) -> &'static str {
        match self {
            Self::System => "SYSTEM",
            Self::Receive => "RECV",
            Self::Request => "REQUEST",
            Self::Turn => "TURN",
            Self::Send => "SEND",
            Self::Success => "SUCCESS",
            Self::Warning => "WARN",
            Self::Error => "ERROR",
            Self::Child => "CHILD",
        }
    }

    fn color_code(self) -> &'static str {
        match self {
            Self::System => ANSI_BLUE,
            Self::Receive => ANSI_MAGENTA,
            Self::Request => ANSI_MAGENTA,
            Self::Turn => ANSI_CYAN,
            Self::Send => ANSI_GREEN,
            Self::Success => ANSI_GREEN,
            Self::Warning => ANSI_YELLOW,
            Self::Error => ANSI_RED,
            Self::Child => ANSI_BRIGHT_BLACK,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn renders_structured_single_line_log() {
        let rendered = render_line(
            "codex-message",
            LogTone::Request,
            "User request received",
            &["Message: m-1".to_string(), "Text: build it".to_string()],
        );

        assert!(rendered.contains("codex-message"));
        assert!(rendered.contains("REQUEST"));
        assert!(rendered.contains("User request received"));
        assert!(rendered.contains("message="));
        assert!(rendered.contains("text=\"build it\""));
        assert!(!rendered.contains('\n'));
    }

    #[test]
    fn renders_send_log_tone() {
        let rendered = render_line(
            "codex-message",
            LogTone::Send,
            "Message sent",
            &[
                "From: @agent-123".to_string(),
                "To: @jay".to_string(),
                "Kind: text".to_string(),
            ],
        );

        assert!(rendered.contains("SEND"));
        assert!(rendered.contains("from=@agent-123"));
        assert!(rendered.contains("to=@jay"));
        assert!(rendered.contains("kind=text"));
    }

    #[test]
    fn renders_recv_log_tone() {
        let rendered = render_line(
            "codex-message",
            LogTone::Receive,
            "Message received",
            &[
                "From: @jay".to_string(),
                "To: @agent-123".to_string(),
                "Kind: text".to_string(),
            ],
        );

        assert!(rendered.contains("RECV"));
        assert!(rendered.contains("from=@jay"));
        assert!(rendered.contains("to=@agent-123"));
        assert!(rendered.contains("kind=text"));
    }
}
