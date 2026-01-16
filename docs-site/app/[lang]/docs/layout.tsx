import { source } from '@/lib/source';
import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import type { ReactNode } from 'react';
import { type Locale, i18n } from '@/lib/i18n';
import Image from 'next/image';

export default async function Layout({
  params,
  children,
}: {
  params: Promise<{ lang: Locale }>;
  children: ReactNode;
}) {
  const { lang } = await params;

  return (
    <DocsLayout
      tree={source.getPageTree(lang)}
      nav={{
        title: (
          <div className="flex items-center gap-2">
            <Image
              src={`${process.env.NEXT_PUBLIC_BASE_PATH || ''}/bingsan-logo.png`}
              alt="Bingsan"
              width={28}
              height={28}
              className="rounded"
            />
            <span>Bingsan</span>
          </div>
        ),
      }}
      i18n={i18n}
    >
      {children}
    </DocsLayout>
  );
}
