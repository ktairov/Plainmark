export type PluralCategory = Intl.LDMLPluralRule;

/**
 * Plural forms keyed by CLDR category. Languages may use any subset
 * (Arabic: zero/two, Welsh: six forms, Japanese: other only); `other` is the
 * required fallback.
 */
export type PluralForms = Partial<Record<PluralCategory, string>> & { other: string };

export type Message = string | PluralForms;
