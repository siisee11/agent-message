use serde_json::{Map, Value, json};

pub(crate) fn report_spec(badge: &str, title: &str, lines: &[String], body: Option<&str>) -> Value {
    let mut elements = Map::new();
    let mut children = Vec::new();

    push_badge(&mut elements, &mut children, "badge", badge);
    push_text(&mut elements, &mut children, "title", title);
    push_separator(&mut elements, &mut children, "sep-top");

    for (index, line) in lines.iter().enumerate() {
        let key = format!("line-{index}");
        push_text(&mut elements, &mut children, &key, line);
    }

    if let Some(body) = body.and_then(non_empty) {
        if !lines.is_empty() {
            push_separator(&mut elements, &mut children, "sep-body");
        }
        for (index, paragraph) in paragraphs(body).into_iter().enumerate() {
            let key = format!("body-{index}");
            push_text(&mut elements, &mut children, &key, &paragraph);
        }
    }

    elements.insert(
        "root".to_string(),
        json!({
            "type": "Stack",
            "children": children,
        }),
    );

    json!({
        "root": "root",
        "elements": elements,
    })
}

pub(crate) fn approval_spec(
    badge: &str,
    title: &str,
    details: &[String],
    reply_hint: &str,
) -> Value {
    let mut lines = details.to_vec();
    lines.push(format!("Reply: {reply_hint}"));
    report_spec(badge, title, &lines, None)
}

fn push_badge(
    elements: &mut Map<String, Value>,
    children: &mut Vec<String>,
    key: &str,
    text: &str,
) {
    children.push(key.to_string());
    elements.insert(
        key.to_string(),
        json!({
            "type": "Badge",
            "props": { "text": text },
        }),
    );
}

fn push_text(elements: &mut Map<String, Value>, children: &mut Vec<String>, key: &str, text: &str) {
    children.push(key.to_string());
    elements.insert(
        key.to_string(),
        json!({
            "type": "Text",
            "props": { "text": text },
        }),
    );
}

fn push_separator(elements: &mut Map<String, Value>, children: &mut Vec<String>, key: &str) {
    children.push(key.to_string());
    elements.insert(
        key.to_string(),
        json!({
            "type": "Separator",
        }),
    );
}

fn paragraphs(text: &str) -> Vec<String> {
    text.split("\n\n")
        .map(str::trim)
        .filter(|part| !part.is_empty())
        .map(ToOwned::to_owned)
        .collect()
}

fn non_empty(value: &str) -> Option<&str> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(trimmed)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn report_spec_has_expected_shape() {
        let spec = report_spec(
            "Completed",
            "Request finished",
            &["Status: ok".to_string()],
            Some("First paragraph\n\nSecond paragraph"),
        );
        assert_eq!(spec["root"], "root");
        assert_eq!(spec["elements"]["badge"]["type"], "Badge");
        assert_eq!(spec["elements"]["title"]["type"], "Text");
        assert_eq!(
            spec["elements"]["body-1"]["props"]["text"],
            "Second paragraph"
        );
    }
}
