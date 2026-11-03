use serde::Serialize;
use std::collections::BTreeMap;

/// Structured command error: `code` is stable and translated on the frontend,
/// `message` stays as an English fallback for logs and unknown codes, and
/// `details` carries values for message interpolation (path, name, numbers…).
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct AppError {
    pub(crate) code: String,
    pub(crate) message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) details: Option<BTreeMap<String, String>>,
}

impl AppError {
    pub(crate) fn new(code: &str, message: impl Into<String>) -> Self {
        Self {
            code: code.to_string(),
            message: message.into(),
            details: None,
        }
    }

    pub(crate) fn detail(mut self, key: &str, value: impl ToString) -> Self {
        self.details
            .get_or_insert_with(BTreeMap::new)
            .insert(key.to_string(), value.to_string());
        self
    }
}

impl std::fmt::Display for AppError {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(&self.message)
    }
}

impl std::error::Error for AppError {}

impl From<std::io::Error> for AppError {
    fn from(error: std::io::Error) -> Self {
        Self::new("ioError", error.to_string())
    }
}

pub(crate) type AppResult<T> = Result<T, AppError>;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn serializes_stable_codes_and_details() {
        let error = AppError::new("itemExists", "An item with this name already exists.")
            .detail("name", "note.md");
        let json = serde_json::to_value(&error).unwrap();
        assert_eq!(json["code"], "itemExists");
        assert_eq!(json["message"], "An item with this name already exists.");
        assert_eq!(json["details"]["name"], "note.md");
    }

    #[test]
    fn omits_empty_details() {
        let json = serde_json::to_value(AppError::new("notMarkdown", "not markdown")).unwrap();
        assert!(json.get("details").is_none());
    }
}
