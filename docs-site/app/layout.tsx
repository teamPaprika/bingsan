import './global.css';
import type { ReactNode } from 'react';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: {
    default: 'Bingsan Documentation',
    template: '%s | Bingsan',
  },
  description: 'High-performance Apache Iceberg REST Catalog documentation',
  keywords: ['Apache Iceberg', 'REST Catalog', 'Data Lake', 'Go', 'PostgreSQL', 'Spark', 'Trino', 'PyIceberg'],
  authors: [{ name: 'Bingsan Team' }],
  openGraph: {
    title: 'Bingsan Documentation',
    description: 'High-performance Apache Iceberg REST Catalog implemented in Go',
    type: 'website',
    locale: 'en_US',
  },
  icons: {
    icon: '/favicon.svg',
  },
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="flex min-h-screen flex-col">
        {children}
      </body>
    </html>
  );
}
