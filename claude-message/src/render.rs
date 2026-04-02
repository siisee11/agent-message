use serde_json::{Map, Value, json};

pub(crate) fn response_spec(
    badge_text: &str,
    badge_variant: &str,
    title: &str,
    lines: &[String],
    body_markdown: Option<&str>,
) -> Value {
    let mut elements = Map::new();
    let mut children = Vec::new();

    push_badge(
        &mut elements,
        &mut children,
        "badge",
        badge_text,
        badge_variant,
    );
    push_text(&mut elements, &mut children, "title", title, "lead");

    if !lines.is_empty() {
        push_separator(&mut elements, &mut children, "sep-meta");
        for (index, line) in lines.iter().enumerate() {
            let key = format!("line-{index}");
            push_text(&mut elements, &mut children, &key, line, "muted");
        }
    }

    if let Some(body) = non_empty(body_markdown) {
        push_separator(&mut elements, &mut children, "sep-body");
        children.push("body".to_string());
        elements.insert(
            "body".to_string(),
            json!({
                "type": "Markdown",
                "props": { "content": body },
                "children": [],
            }),
        );
    }

    elements.insert(
        "root".to_string(),
        json!({
            "type": "Stack",
            "props": { "gap": "sm" },
            "children": children,
        }),
    );

    json!({
        "root": "root",
        "elements": elements,
    })
}

fn push_badge(
    elements: &mut Map<String, Value>,
    children: &mut Vec<String>,
    key: &str,
    text: &str,
    variant: &str,
) {
    children.push(key.to_string());
    elements.insert(
        key.to_string(),
        json!({
            "type": "Badge",
            "props": {
                "text": text,
                "variant": variant,
            },
            "children": [],
        }),
    );
}

fn push_text(
    elements: &mut Map<String, Value>,
    children: &mut Vec<String>,
    key: &str,
    text: &str,
    variant: &str,
) {
    children.push(key.to_string());
    elements.insert(
        key.to_string(),
        json!({
            "type": "Text",
            "props": {
                "text": text,
                "variant": variant,
            },
            "children": [],
        }),
    );
}

fn push_separator(elements: &mut Map<String, Value>, children: &mut Vec<String>, key: &str) {
    children.push(key.to_string());
    elements.insert(
        key.to_string(),
        json!({
            "type": "Separator",
            "children": [],
        }),
    );
}

fn non_empty(value: Option<&str>) -> Option<&str> {
    let trimmed = value?.trim();
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
    fn response_spec_preserves_markdown_body() {
        let spec = response_spec(
            "Completed",
            "default",
            "Claude finished",
            &["Session: abc".to_string()],
            Some("## Heading\n\nParagraph"),
        );

        assert_eq!(spec["root"], "root");
        assert_eq!(spec["elements"]["badge"]["props"]["text"], "Completed");
        assert_eq!(spec["elements"]["body"]["type"], "Markdown");
    }
}
