import type { I18nConfig } from 'fumadocs-core/i18n';

export const i18n: I18nConfig = {
  defaultLanguage: 'en',
  languages: ['en', 'ko'],
  parser: 'dir',
};

export type Locale = (typeof i18n)['languages'][number];

export const localeNames: Record<Locale, string> = {
  en: 'English',
  ko: '한국어',
};
