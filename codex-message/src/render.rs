use serde_json::{Map, Value, json};

use crate::json_render_validation::ValidationIssue;

pub(crate) struct ApprovalAction<'a> {
    pub(crate) label: &'a str,
    pub(crate) value: &'a str,
    pub(crate) variant: &'a str,
}

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
    actions: &[ApprovalAction<'_>],
) -> Value {
    let action_values: Vec<Value> = actions
        .iter()
        .map(|action| {
            json!({
                "label": action.label,
                "value": action.value,
                "variant": action.variant,
            })
        })
        .collect();

    json!({
        "root": "approval",
        "elements": {
            "approval": {
                "type": "ApprovalCard",
                "props": {
                    "badge": badge,
                    "title": title,
                    "details": details,
                    "replyHint": reply_hint,
                    "actions": action_values,
                },
            },
        },
    })
}

pub(crate) fn invalid_json_render_spec(
    title: &str,
    summary: &str,
    issues: &[ValidationIssue],
) -> Value {
    let rows: Vec<Vec<String>> = issues
        .iter()
        .map(|issue| vec![issue.path.clone(), issue.message.clone()])
        .collect();

    json!({
        "root": "root",
        "elements": {
            "root": {
                "type": "Stack",
                "props": { "gap": "sm" },
                "children": ["badge", "title", "summary", "table"],
            },
            "badge": {
                "type": "Badge",
                "props": {
                    "text": "Invalid",
                    "variant": "destructive",
                },
                "children": [],
            },
            "title": {
                "type": "Text",
                "props": {
                    "text": title,
                    "variant": "lead",
                },
                "children": [],
            },
            "summary": {
                "type": "Text",
                "props": {
                    "text": summary,
                    "variant": "muted",
                },
                "children": [],
            },
            "table": {
                "type": "Table",
                "props": {
                    "caption": "Validation issues",
                    "columns": ["Path", "Issue"],
                    "rows": rows,
                },
                "children": [],
            },
        },
    })
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

    #[test]
    fn approval_spec_has_expected_shape() {
        let spec = approval_spec(
            "Approval Needed",
            "Command approval requested",
            &["Command: npm test".to_string()],
            "approve | session | deny | cancel",
            &[
                ApprovalAction {
                    label: "Approve",
                    value: "approve",
                    variant: "primary",
                },
                ApprovalAction {
                    label: "Deny",
                    value: "deny",
                    variant: "destructive",
                },
            ],
        );

        assert_eq!(spec["root"], "approval");
        assert_eq!(spec["elements"]["approval"]["type"], "ApprovalCard");
        assert_eq!(
            spec["elements"]["approval"]["props"]["actions"][0]["value"],
            "approve"
        );
    }

    #[test]
    fn invalid_json_render_spec_renders_issue_table() {
        let spec = invalid_json_render_spec(
            "Invalid json-render input",
            "The payload was ignored.",
            &[ValidationIssue {
                path: "/elements/main/props/title".to_string(),
                message: "required".to_string(),
            }],
        );

        assert_eq!(spec["elements"]["table"]["type"], "Table");
        assert_eq!(
            spec["elements"]["table"]["props"]["rows"][0][0],
            "/elements/main/props/title"
        );
    }
}
