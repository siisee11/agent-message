use serde_json::{Map, Value};

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct ValidationIssue {
    pub(crate) path: String,
    pub(crate) message: String,
}

impl ValidationIssue {
    fn new(path: impl Into<String>, message: impl Into<String>) -> Self {
        Self {
            path: path.into(),
            message: message.into(),
        }
    }
}

pub(crate) fn validate_json_render_spec(spec: &Value) -> Vec<ValidationIssue> {
    let mut issues = Vec::new();
    let Some(root) = spec.as_object() else {
        issues.push(ValidationIssue::new("/", "expected object"));
        return issues;
    };

    if let Some(state) = root.get("state") {
        if !state.is_object() {
            issues.push(ValidationIssue::new("/state", "expected object"));
        }
    }

    let root_key = match root.get("root") {
        Some(Value::String(value)) if !value.trim().is_empty() => Some(value.as_str()),
        Some(_) => {
            issues.push(ValidationIssue::new("/root", "expected non-empty string"));
            None
        }
        None => {
            issues.push(ValidationIssue::new("/root", "required"));
            None
        }
    };

    let Some(elements) = root.get("elements") else {
        issues.push(ValidationIssue::new("/elements", "required"));
        return issues;
    };
    let Some(elements) = elements.as_object() else {
        issues.push(ValidationIssue::new("/elements", "expected object"));
        return issues;
    };

    if let Some(root_key) = root_key {
        if !elements.contains_key(root_key) {
            issues.push(ValidationIssue::new(
                "/root",
                format!("references missing element {root_key:?}"),
            ));
        }
    }

    for (key, element) in elements {
        let element_path = format!("/elements/{}", escape_json_pointer(key));
        validate_element(key, element, &element_path, elements, &mut issues);
    }

    issues
}

fn validate_element(
    key: &str,
    element: &Value,
    path: &str,
    elements: &Map<String, Value>,
    issues: &mut Vec<ValidationIssue>,
) {
    let Some(element) = element.as_object() else {
        issues.push(ValidationIssue::new(path, "expected object"));
        return;
    };

    let element_type = match element.get("type") {
        Some(Value::String(value)) if !value.trim().is_empty() => Some(value.as_str()),
        Some(_) => {
            issues.push(ValidationIssue::new(
                format!("{path}/type"),
                "expected non-empty string",
            ));
            None
        }
        None => {
            issues.push(ValidationIssue::new(format!("{path}/type"), "required"));
            None
        }
    };

    let props = match element.get("props") {
        None => None,
        Some(Value::Object(props)) => Some(props),
        Some(_) => {
            issues.push(ValidationIssue::new(
                format!("{path}/props"),
                "expected object",
            ));
            None
        }
    };

    if let Some(children) = element.get("children") {
        validate_children(children, path, elements, issues);
    }

    if let Some(repeat) = element.get("repeat") {
        validate_repeat(repeat, path, issues);
    }

    if let Some(on) = element.get("on") {
        validate_action_map(on, &format!("{path}/on"), issues);
    }

    if let Some(watch) = element.get("watch") {
        validate_action_map(watch, &format!("{path}/watch"), issues);
    }

    if let Some(visible) = element.get("visible") {
        validate_visible(visible, &format!("{path}/visible"), issues);
    }

    if let (Some(element_type), Some(props)) = (element_type, props) {
        validate_component_props(key, element_type, props, &format!("{path}/props"), issues);
    } else if let Some(element_type) = element_type {
        validate_component_props(
            key,
            element_type,
            &Map::new(),
            &format!("{path}/props"),
            issues,
        );
    }
}

fn validate_children(
    children: &Value,
    path: &str,
    elements: &Map<String, Value>,
    issues: &mut Vec<ValidationIssue>,
) {
    let Some(children) = children.as_array() else {
        issues.push(ValidationIssue::new(
            format!("{path}/children"),
            "expected array of strings",
        ));
        return;
    };

    for (index, child) in children.iter().enumerate() {
        let child_path = format!("{path}/children/{index}");
        let Some(child_key) = child.as_str() else {
            issues.push(ValidationIssue::new(child_path, "expected string"));
            continue;
        };
        if !elements.contains_key(child_key) {
            issues.push(ValidationIssue::new(
                child_path,
                format!("references missing element {child_key:?}"),
            ));
        }
    }
}

fn validate_repeat(repeat: &Value, path: &str, issues: &mut Vec<ValidationIssue>) {
    let repeat_path = format!("{path}/repeat");
    let Some(repeat) = repeat.as_object() else {
        issues.push(ValidationIssue::new(repeat_path, "expected object"));
        return;
    };

    match repeat.get("statePath") {
        Some(Value::String(value)) if !value.trim().is_empty() => {}
        Some(_) => issues.push(ValidationIssue::new(
            format!("{repeat_path}/statePath"),
            "expected non-empty string",
        )),
        None => issues.push(ValidationIssue::new(
            format!("{repeat_path}/statePath"),
            "required",
        )),
    }

    if let Some(key) = repeat.get("key") {
        expect_optional_non_empty_string(Some(key), &format!("{repeat_path}/key"), issues);
    }
}

fn validate_action_map(action_map: &Value, path: &str, issues: &mut Vec<ValidationIssue>) {
    let Some(action_map) = action_map.as_object() else {
        issues.push(ValidationIssue::new(path, "expected object"));
        return;
    };

    for (name, binding) in action_map {
        let binding_path = format!("{path}/{}", escape_json_pointer(name));
        let Some(binding) = binding.as_object() else {
            issues.push(ValidationIssue::new(binding_path, "expected object"));
            continue;
        };

        match binding.get("action") {
            Some(Value::String(value)) if !value.trim().is_empty() => {}
            Some(_) => issues.push(ValidationIssue::new(
                format!("{binding_path}/action"),
                "expected non-empty string",
            )),
            None => issues.push(ValidationIssue::new(
                format!("{binding_path}/action"),
                "required",
            )),
        }

        if let Some(params) = binding.get("params") {
            if !params.is_object() {
                issues.push(ValidationIssue::new(
                    format!("{binding_path}/params"),
                    "expected object",
                ));
            }
        }
    }
}

fn validate_visible(visible: &Value, path: &str, issues: &mut Vec<ValidationIssue>) {
    if visible.is_boolean() || visible.is_object() || visible.is_array() {
        return;
    }

    issues.push(ValidationIssue::new(
        path,
        "expected boolean, object, or array of conditions",
    ));
}

fn validate_component_props(
    _key: &str,
    element_type: &str,
    props: &Map<String, Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match element_type {
        "ApprovalCard" => {
            expect_optional_string_prop(props, "badge", path, issues);
            expect_required_string_prop(props, "title", path, issues);
            expect_optional_string_array_prop(props, "details", path, issues);
            expect_optional_string_prop(props, "replyHint", path, issues);
            expect_optional_array_prop(
                props,
                "actions",
                path,
                issues,
                |value, item_path, issues| {
                    let Some(action) = value.as_object() else {
                        issues.push(ValidationIssue::new(item_path, "expected object"));
                        return;
                    };
                    expect_required_string_value(
                        action.get("label"),
                        &format!("{item_path}/label"),
                        issues,
                    );
                    expect_required_string_value(
                        action.get("value"),
                        &format!("{item_path}/value"),
                        issues,
                    );
                    expect_optional_enum_value(
                        action.get("variant"),
                        &format!("{item_path}/variant"),
                        &["primary", "secondary", "destructive"],
                        issues,
                    );
                },
            );
        }
        "Alert" => {
            expect_required_string_prop(props, "title", path, issues);
            expect_optional_string_prop(props, "message", path, issues);
            expect_optional_enum_prop(
                props,
                "type",
                path,
                &["info", "success", "warning", "error"],
                issues,
            );
        }
        "AskQuestion" => {
            expect_optional_string_prop(props, "confirmLabel", path, issues);
            expect_optional_string_prop(props, "freeformPlaceholder", path, issues);
            expect_optional_array_prop(
                props,
                "options",
                path,
                issues,
                |value, item_path, issues| {
                    let Some(option) = value.as_object() else {
                        issues.push(ValidationIssue::new(item_path, "expected object"));
                        return;
                    };
                    expect_required_string_value(
                        option.get("label"),
                        &format!("{item_path}/label"),
                        issues,
                    );
                    expect_optional_string_value(
                        option.get("value"),
                        &format!("{item_path}/value"),
                        issues,
                    );
                },
            );
            expect_required_string_prop(props, "question", path, issues);
        }
        "Avatar" => {
            expect_optional_string_prop(props, "src", path, issues);
            expect_required_string_prop(props, "name", path, issues);
            expect_optional_enum_prop(props, "size", path, &["sm", "md", "lg"], issues);
        }
        "Badge" => {
            expect_required_string_prop(props, "text", path, issues);
            expect_optional_enum_prop(
                props,
                "variant",
                path,
                &["default", "secondary", "destructive", "outline"],
                issues,
            );
        }
        "BarGraph" | "LineGraph" => {
            expect_optional_string_prop(props, "currency", path, issues);
            expect_optional_string_prop(props, "description", path, issues);
            expect_required_array_prop(props, "data", path, issues, |value, item_path, issues| {
                let Some(datum) = value.as_object() else {
                    issues.push(ValidationIssue::new(item_path, "expected object"));
                    return;
                };
                expect_optional_string_value(
                    datum.get("color"),
                    &format!("{item_path}/color"),
                    issues,
                );
                expect_required_string_value(
                    datum.get("label"),
                    &format!("{item_path}/label"),
                    issues,
                );
                expect_required_number_value(
                    datum.get("value"),
                    &format!("{item_path}/value"),
                    issues,
                );
            });
            expect_optional_enum_prop(
                props,
                "format",
                path,
                &["currency", "number", "percent"],
                issues,
            );
            expect_optional_number_prop(props, "maxValue", path, issues);
            expect_optional_string_prop(props, "title", path, issues);
        }
        "Card" => {
            expect_optional_string_prop(props, "title", path, issues);
            expect_optional_string_prop(props, "description", path, issues);
            expect_optional_enum_prop(props, "maxWidth", path, &["sm", "md", "lg", "full"], issues);
            expect_optional_bool_prop(props, "centered", path, issues);
            expect_optional_string_prop(props, "className", path, issues);
        }
        "GitCommitLog" => {
            expect_optional_string_prop(props, "branch", path, issues);
            expect_required_array_prop(
                props,
                "commits",
                path,
                issues,
                |value, item_path, issues| {
                    let Some(commit) = value.as_object() else {
                        issues.push(ValidationIssue::new(item_path, "expected object"));
                        return;
                    };
                    expect_optional_string_value(
                        commit.get("authorName"),
                        &format!("{item_path}/authorName"),
                        issues,
                    );
                    expect_optional_string_value(
                        commit.get("authoredAt"),
                        &format!("{item_path}/authoredAt"),
                        issues,
                    );
                    expect_optional_string_value(
                        commit.get("body"),
                        &format!("{item_path}/body"),
                        issues,
                    );
                    expect_optional_bool_value(
                        commit.get("isHead"),
                        &format!("{item_path}/isHead"),
                        issues,
                    );
                    expect_optional_bool_value(
                        commit.get("isMerge"),
                        &format!("{item_path}/isMerge"),
                        issues,
                    );
                    match commit.get("refs") {
                        Some(value) => expect_string_array_value(
                            Some(value),
                            &format!("{item_path}/refs"),
                            issues,
                        ),
                        None => {}
                    }
                    expect_required_string_value(
                        commit.get("sha"),
                        &format!("{item_path}/sha"),
                        issues,
                    );
                    if let Some(stats) = commit.get("stats") {
                        let stats_path = format!("{item_path}/stats");
                        let Some(stats) = stats.as_object() else {
                            issues.push(ValidationIssue::new(stats_path, "expected object"));
                            return;
                        };
                        expect_optional_nonnegative_int_value(
                            stats.get("deletions"),
                            &format!("{stats_path}/deletions"),
                            issues,
                        );
                        expect_optional_nonnegative_int_value(
                            stats.get("filesChanged"),
                            &format!("{stats_path}/filesChanged"),
                            issues,
                        );
                        expect_optional_nonnegative_int_value(
                            stats.get("insertions"),
                            &format!("{stats_path}/insertions"),
                            issues,
                        );
                    }
                    expect_required_string_value(
                        commit.get("subject"),
                        &format!("{item_path}/subject"),
                        issues,
                    );
                },
            );
            expect_optional_string_prop(props, "description", path, issues);
            expect_optional_string_prop(props, "repository", path, issues);
            expect_optional_string_prop(props, "title", path, issues);
        }
        "Grid" => {
            expect_optional_number_prop(props, "columns", path, issues);
            expect_optional_enum_prop(props, "gap", path, &["sm", "md", "lg", "xl"], issues);
            expect_optional_string_prop(props, "className", path, issues);
        }
        "Gif" => {
            expect_optional_string_prop(props, "src", path, issues);
            expect_required_string_prop(props, "alt", path, issues);
            expect_optional_number_prop(props, "width", path, issues);
            expect_optional_number_prop(props, "height", path, issues);
        }
        "Heading" => {
            expect_required_string_prop(props, "text", path, issues);
            expect_optional_enum_prop(props, "level", path, &["h1", "h2", "h3", "h4"], issues);
        }
        "Image" => {
            expect_optional_string_prop(props, "src", path, issues);
            expect_required_string_prop(props, "alt", path, issues);
            expect_optional_number_prop(props, "width", path, issues);
            expect_optional_number_prop(props, "height", path, issues);
        }
        "Markdown" => {
            expect_required_string_prop(props, "content", path, issues);
        }
        "Progress" => {
            expect_required_number_prop(props, "value", path, issues);
            expect_optional_number_prop(props, "max", path, issues);
            expect_optional_string_prop(props, "label", path, issues);
        }
        "Separator" => {
            expect_optional_enum_prop(
                props,
                "orientation",
                path,
                &["horizontal", "vertical"],
                issues,
            );
        }
        "Skeleton" => {
            expect_optional_string_prop(props, "width", path, issues);
            expect_optional_string_prop(props, "height", path, issues);
            expect_optional_bool_prop(props, "rounded", path, issues);
        }
        "Spinner" => {
            expect_optional_enum_prop(props, "size", path, &["sm", "md", "lg"], issues);
            expect_optional_string_prop(props, "label", path, issues);
        }
        "Stack" => {
            expect_optional_enum_prop(
                props,
                "direction",
                path,
                &["horizontal", "vertical"],
                issues,
            );
            expect_optional_enum_prop(
                props,
                "gap",
                path,
                &["none", "sm", "md", "lg", "xl"],
                issues,
            );
            expect_optional_enum_prop(
                props,
                "align",
                path,
                &["start", "center", "end", "stretch"],
                issues,
            );
            expect_optional_enum_prop(
                props,
                "justify",
                path,
                &["start", "center", "end", "between", "around"],
                issues,
            );
            expect_optional_string_prop(props, "className", path, issues);
        }
        "Table" => {
            expect_required_string_array_prop(props, "columns", path, issues);
            expect_required_array_prop(props, "rows", path, issues, |value, item_path, issues| {
                expect_string_array_value(Some(value), &item_path, issues);
            });
            expect_optional_string_prop(props, "caption", path, issues);
        }
        "Text" => {
            expect_required_string_prop(props, "text", path, issues);
            expect_optional_enum_prop(
                props,
                "variant",
                path,
                &["body", "caption", "muted", "lead", "code"],
                issues,
            );
        }
        _ => {
            let type_path = path.strip_suffix("/props").unwrap_or(path);
            issues.push(ValidationIssue::new(
                format!("{type_path}/type"),
                format!("unsupported component type {element_type:?}"),
            ));
        }
    }
}

fn expect_required_string_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    expect_required_string_value(props.get(key), &format!("{path}/{key}"), issues);
}

fn expect_optional_string_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    expect_optional_string_value(props.get(key), &format!("{path}/{key}"), issues);
}

fn expect_required_number_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    expect_required_number_value(props.get(key), &format!("{path}/{key}"), issues);
}

fn expect_optional_number_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    expect_optional_number_value(props.get(key), &format!("{path}/{key}"), issues);
}

fn expect_optional_bool_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    expect_optional_bool_value(props.get(key), &format!("{path}/{key}"), issues);
}

fn expect_optional_enum_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    allowed: &[&str],
    issues: &mut Vec<ValidationIssue>,
) {
    expect_optional_enum_value(props.get(key), &format!("{path}/{key}"), allowed, issues);
}

fn expect_required_string_array_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    if props.get(key).is_none() {
        issues.push(ValidationIssue::new(format!("{path}/{key}"), "required"));
        return;
    }
    expect_string_array_value(props.get(key), &format!("{path}/{key}"), issues);
}

fn expect_optional_string_array_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    if let Some(value) = props.get(key) {
        expect_string_array_value(Some(value), &format!("{path}/{key}"), issues);
    }
}

fn expect_required_array_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
    item_validator: impl Fn(&Value, String, &mut Vec<ValidationIssue>),
) {
    let value = props.get(key);
    if value.is_none() {
        issues.push(ValidationIssue::new(format!("{path}/{key}"), "required"));
        return;
    }
    expect_array_value(value, &format!("{path}/{key}"), issues, item_validator);
}

fn expect_optional_array_prop(
    props: &Map<String, Value>,
    key: &str,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
    item_validator: impl Fn(&Value, String, &mut Vec<ValidationIssue>),
) {
    expect_array_value(
        props.get(key),
        &format!("{path}/{key}"),
        issues,
        item_validator,
    );
}

fn expect_required_string_value(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        Some(value) if value.is_string() || is_dynamic_expression(value) => {}
        Some(_) => issues.push(ValidationIssue::new(path, "expected string")),
        None => issues.push(ValidationIssue::new(path, "required")),
    }
}

fn expect_optional_string_value(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        None => {}
        Some(Value::String(_)) => {}
        Some(value) if is_dynamic_expression(value) => {}
        Some(_) => issues.push(ValidationIssue::new(path, "expected string")),
    }
}

fn expect_required_number_value(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        Some(value) if value.is_number() || is_dynamic_expression(value) => {}
        Some(_) => issues.push(ValidationIssue::new(path, "expected number")),
        None => issues.push(ValidationIssue::new(path, "required")),
    }
}

fn expect_optional_number_value(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        None => {}
        Some(Value::Number(_)) => {}
        Some(value) if is_dynamic_expression(value) => {}
        Some(_) => issues.push(ValidationIssue::new(path, "expected number")),
    }
}

fn expect_optional_bool_value(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        None => {}
        Some(Value::Bool(_)) => {}
        Some(value) if is_dynamic_expression(value) => {}
        Some(_) => issues.push(ValidationIssue::new(path, "expected boolean")),
    }
}

fn expect_optional_non_empty_string(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        None => {}
        Some(Value::String(text)) if !text.trim().is_empty() => {}
        Some(_) => issues.push(ValidationIssue::new(path, "expected non-empty string")),
    }
}

fn expect_optional_enum_value(
    value: Option<&Value>,
    path: &str,
    allowed: &[&str],
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        None => {}
        Some(Value::String(text)) if allowed.contains(&text.as_str()) => {}
        Some(value) if is_dynamic_expression(value) => {}
        Some(_) => issues.push(ValidationIssue::new(
            path,
            format!("expected one of {}", allowed.join(", ")),
        )),
    }
}

fn expect_optional_nonnegative_int_value(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
) {
    match value {
        None => {}
        Some(Value::Number(number)) if number.as_u64().is_some() => {}
        Some(value) if is_dynamic_expression(value) => {}
        Some(_) => issues.push(ValidationIssue::new(path, "expected non-negative integer")),
    }
}

fn expect_string_array_value(value: Option<&Value>, path: &str, issues: &mut Vec<ValidationIssue>) {
    expect_array_value(
        value,
        path,
        issues,
        |value, item_path, issues| match value {
            Value::String(_) => {}
            value if is_dynamic_expression(value) => {}
            _ => issues.push(ValidationIssue::new(item_path, "expected string")),
        },
    );
}

fn expect_array_value(
    value: Option<&Value>,
    path: &str,
    issues: &mut Vec<ValidationIssue>,
    item_validator: impl Fn(&Value, String, &mut Vec<ValidationIssue>),
) {
    let Some(value) = value else {
        return;
    };
    if is_dynamic_expression(value) {
        return;
    }
    let Some(array) = value.as_array() else {
        issues.push(ValidationIssue::new(path, "expected array"));
        return;
    };

    for (index, item) in array.iter().enumerate() {
        item_validator(item, format!("{path}/{index}"), issues);
    }
}

fn is_dynamic_expression(value: &Value) -> bool {
    let Some(object) = value.as_object() else {
        return false;
    };

    object.contains_key("$state")
        || object.contains_key("$bindState")
        || object.contains_key("$bindItem")
        || object.contains_key("$cond")
        || object.contains_key("$template")
        || object.contains_key("$item")
        || object.contains_key("$index")
}

fn escape_json_pointer(value: &str) -> String {
    value.replace('~', "~0").replace('/', "~1")
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn accepts_minimal_stack_spec() {
        let issues = validate_json_render_spec(&json!({
            "root": "main",
            "elements": {
                "main": {
                    "type": "Stack",
                    "children": ["title"],
                },
                "title": {
                    "type": "Text",
                    "props": {
                        "text": "Hello"
                    }
                }
            }
        }));

        assert!(issues.is_empty(), "{issues:?}");
    }

    #[test]
    fn reports_missing_component_props_and_child_references() {
        let issues = validate_json_render_spec(&json!({
            "root": "main",
            "elements": {
                "main": {
                    "type": "Stack",
                    "children": ["missing"]
                },
                "question": {
                    "type": "AskQuestion",
                    "props": {
                        "options": [
                            {"label": 123}
                        ]
                    }
                }
            }
        }));

        assert!(issues.contains(&ValidationIssue::new(
            "/elements/main/children/0",
            "references missing element \"missing\"",
        )));
        assert!(issues.contains(&ValidationIssue::new(
            "/elements/question/props/question",
            "required",
        )));
        assert!(issues.contains(&ValidationIssue::new(
            "/elements/question/props/options/0/label",
            "expected string",
        )));
    }

    #[test]
    fn accepts_dynamic_expressions_for_props() {
        let issues = validate_json_render_spec(&json!({
            "root": "main",
            "elements": {
                "main": {
                    "type": "Text",
                    "props": {
                        "text": { "$template": "Hello, ${/name}!" }
                    }
                }
            }
        }));

        assert!(issues.is_empty(), "{issues:?}");
    }

    #[test]
    fn accepts_gif_component() {
        let issues = validate_json_render_spec(&json!({
            "root": "gif",
            "elements": {
                "gif": {
                    "type": "Gif",
                    "props": {
                        "src": "https://example.test/animation.gif",
                        "alt": "Animated status",
                        "width": 480,
                        "height": 360
                    }
                }
            }
        }));

        assert!(issues.is_empty(), "{issues:?}");
    }
}
