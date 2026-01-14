import { RootProvider } from 'fumadocs-ui/provider/next';
import { defineI18nUI } from 'fumadocs-ui/i18n';
import type { ReactNode } from 'react';
import { i18n, type Locale } from '@/lib/i18n';

const { provider } = defineI18nUI(i18n, {
  translations: {
    en: {
      displayName: 'English',
    },
    ko: {
      displayName: '한국어',
      search: '문서 검색...',
      toc: '이 페이지의 내용',
    },
  },
});

export function generateStaticParams() {
  return i18n.languages.map((lang) => ({ lang }));
}

export default async function LangLayout({
  params,
  children,
}: {
  params: Promise<{ lang: Locale }>;
  children: ReactNode;
}) {
  const { lang } = await params;

  return (
    <RootProvider i18n={provider(lang)}>
      {children}
    </RootProvider>
  );
}
