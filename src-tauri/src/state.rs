use std::{
    path::PathBuf,
    sync::{atomic::AtomicBool, Mutex},
};

use crate::error::{AppError, AppResult};
use crate::menu::MenuLabels;

pub(crate) struct RootState {
    root: Mutex<PathBuf>,
}

impl RootState {
    pub(crate) fn new(root: PathBuf) -> Self {
        Self {
            root: Mutex::new(root),
        }
    }

    pub(crate) fn get(&self) -> AppResult<PathBuf> {
        self.root
            .lock()
            .map(|root| root.clone())
            .map_err(|_| AppError::new("workspaceLock", "Workspace root lock is unavailable."))
    }

    pub(crate) fn set(&self, root: PathBuf) -> AppResult<()> {
        *self
            .root
            .lock()
            .map_err(|_| AppError::new("workspaceLock", "Workspace root lock is unavailable."))? = root;
        Ok(())
    }
}

pub(crate) struct LanguageState {
    language: Mutex<String>,
}

impl LanguageState {
    pub(crate) fn new(language: String) -> Self {
        Self {
            language: Mutex::new(language),
        }
    }

    pub(crate) fn get(&self) -> String {
        self.language
            .lock()
            .map(|language| language.clone())
            .unwrap_or_else(|_| "en".to_string())
    }

    pub(crate) fn set(&self, language: String) -> AppResult<()> {
        *self
            .language
            .lock()
            .map_err(|_| AppError::new("languageLock", "Language lock is unavailable."))? = language;
        Ok(())
    }
}

pub(crate) fn normalize_language(value: &str) -> String {
    let normalized = value.trim().replace('_', "-");
    let valid = !normalized.is_empty()
        && normalized.len() <= 35
        && normalized.split('-').all(|part| {
            !part.is_empty()
                && part.len() <= 8
                && part
                    .chars()
                    .all(|character| character.is_ascii_alphanumeric())
        });
    if valid {
        normalized
    } else {
        "en".to_string()
    }
}

#[derive(Default)]
pub(crate) struct MenuLabelsState {
    labels: Mutex<MenuLabels>,
}

impl MenuLabelsState {
    pub(crate) fn new(labels: MenuLabels) -> Self {
        Self {
            labels: Mutex::new(labels),
        }
    }

    pub(crate) fn current(&self) -> MenuLabels {
        self.labels
            .lock()
            .map(|labels| labels.clone())
            .unwrap_or_default()
    }

    pub(crate) fn set(&self, labels: MenuLabels) -> AppResult<()> {
        *self
            .labels
            .lock()
            .map_err(|_| AppError::new("menuLabelsLock", "Menu labels lock is unavailable."))? = labels;
        Ok(())
    }
}

#[derive(Default)]
pub(crate) struct LifecycleState {
    pub(crate) allow_exit: AtomicBool,
    pub(crate) window_revealed: AtomicBool,
}

#[cfg(test)]
mod tests {
    use super::normalize_language;

    #[test]
    fn language_codes_are_not_limited_to_the_initial_locales() {
        assert_eq!(normalize_language("uk"), "uk");
        assert_eq!(normalize_language("pt_BR"), "pt-BR");
        assert_eq!(normalize_language("zh-Hant"), "zh-Hant");
        assert_eq!(normalize_language("../../bad"), "en");
        assert_eq!(normalize_language(""), "en");
    }
}
