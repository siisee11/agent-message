use std::cmp::{max, min};
use std::sync::Mutex;

use ratatui::buffer::Buffer;
use ratatui::layout::Rect;
use ratatui::text::{Line, Text};
use ratatui::widgets::{Block, BorderType, Borders, Paragraph, Widget, Wrap};

static OUTPUT_LOCK: Mutex<()> = Mutex::new(());

#[derive(Debug, Clone)]
pub(crate) struct LogUi {
    app_name: &'static str,
}

#[derive(Debug, Clone, Copy)]
enum LogTone {
    System,
    Request,
    Turn,
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

    pub(crate) fn turn<S, I>(&self, title: &str, lines: I)
    where
        S: Into<String>,
        I: IntoIterator<Item = S>,
    {
        self.print(LogTone::Turn, title, lines);
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
        let rendered = render_card(
            self.app_name,
            tone,
            title,
            &collect_lines(lines.into_iter().map(Into::into)),
        );
        let guard = match OUTPUT_LOCK.lock() {
            Ok(guard) => guard,
            Err(poisoned) => poisoned.into_inner(),
        };
        eprintln!("{rendered}\n");
        drop(guard);
    }
}

fn collect_lines(lines: impl IntoIterator<Item = String>) -> Vec<String> {
    lines
        .into_iter()
        .map(|line| line.replace('\t', "  "))
        .filter(|line| !line.trim().is_empty())
        .collect()
}

fn render_card(app_name: &str, tone: LogTone, title: &str, lines: &[String]) -> String {
    let width = preferred_width();
    let inner_width = width.saturating_sub(2) as usize;

    let mut body_lines = vec![Line::from(title.trim().to_string())];
    if !lines.is_empty() {
        body_lines.push(Line::from(String::new()));
        body_lines.extend(lines.iter().map(|line| Line::from(format!("* {line}"))));
    }

    let content_height = estimate_text_height(&body_lines, inner_width);
    let area = Rect::new(0, 0, width, content_height.saturating_add(2));
    let mut buffer = Buffer::empty(area);

    let block = Block::default()
        .borders(Borders::ALL)
        .border_type(BorderType::Rounded)
        .title(format!(" {app_name} | {} ", tone.label()));
    let inner = block.inner(area);
    block.render(area, &mut buffer);

    Paragraph::new(Text::from(body_lines))
        .wrap(Wrap { trim: false })
        .render(inner, &mut buffer);

    buffer_to_string(&buffer)
}

fn preferred_width() -> u16 {
    let env_width = std::env::var("COLUMNS")
        .ok()
        .and_then(|value| value.trim().parse::<u16>().ok())
        .unwrap_or(96);
    min(max(env_width, 56), 108)
}

fn estimate_text_height(lines: &[Line<'_>], inner_width: usize) -> u16 {
    if inner_width == 0 {
        return 1;
    }

    let mut height = 0usize;
    for line in lines {
        let text = line.to_string();
        let logical_width = max(text.chars().count(), 1);
        height += logical_width.div_ceil(inner_width);
    }

    max(height, 1) as u16
}

fn buffer_to_string(buffer: &Buffer) -> String {
    let area = buffer.area;
    let mut output = String::new();

    for y in area.y..area.y.saturating_add(area.height) {
        let mut line = String::new();
        for x in area.x..area.x.saturating_add(area.width) {
            line.push_str(buffer[(x, y)].symbol());
        }
        let trimmed = line.trim_end_matches(' ');
        output.push_str(trimmed);
        if y + 1 < area.y.saturating_add(area.height) {
            output.push('\n');
        }
    }

    output
}

impl LogTone {
    fn label(self) -> &'static str {
        match self {
            Self::System => "SYSTEM",
            Self::Request => "REQUEST",
            Self::Turn => "TURN",
            Self::Success => "SUCCESS",
            Self::Warning => "WARNING",
            Self::Error => "ERROR",
            Self::Child => "CHILD",
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn renders_log_card_with_title_and_lines() {
        let rendered = render_card(
            "claude-message",
            LogTone::Request,
            "User request received",
            &["Message: m-1".to_string(), "Text: build it".to_string()],
        );

        assert!(rendered.contains("claude-message"));
        assert!(rendered.contains("REQUEST"));
        assert!(rendered.contains("User request received"));
        assert!(rendered.contains("* Text: build it"));
    }
}
