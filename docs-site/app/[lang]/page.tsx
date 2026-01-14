import Link from 'next/link';
import Image from 'next/image';
import { type Locale, i18n } from '@/lib/i18n';

const content: Record<Locale, {
  title: string;
  subtitle: string;
  description: string;
  getStarted: string;
  apiReference: string;
  features: { title: string; description: string }[];
}> = {
  en: {
    title: 'Bingsan',
    subtitle: 'High-Performance Apache Iceberg REST Catalog',
    description:
      'A production-ready Iceberg REST catalog implementation in Go, designed for performance and scalability.',
    getStarted: 'Get Started',
    apiReference: 'API Reference',
    features: [
      {
        title: 'High Performance',
        description: 'Built with Go for maximum throughput and minimal latency',
      },
      {
        title: 'Iceberg Compatible',
        description: 'Full REST catalog API compliance with Spark, Trino, and PyIceberg',
      },
      {
        title: 'Production Ready',
        description: 'PostgreSQL backend with connection pooling and metrics',
      },
    ],
  },
  ko: {
    title: 'Bingsan',
    subtitle: '고성능 Apache Iceberg REST 카탈로그',
    description:
      'Go로 구현된 프로덕션 레디 Iceberg REST 카탈로그입니다. 성능과 확장성을 위해 설계되었습니다.',
    getStarted: '시작하기',
    apiReference: 'API 레퍼런스',
    features: [
      {
        title: '고성능',
        description: '최대 처리량과 최소 지연 시간을 위해 Go로 구축',
      },
      {
        title: 'Iceberg 호환',
        description: 'Spark, Trino, PyIceberg와 완벽한 REST 카탈로그 API 호환',
      },
      {
        title: '프로덕션 레디',
        description: '커넥션 풀링과 메트릭이 포함된 PostgreSQL 백엔드',
      },
    ],
  },
};

export function generateStaticParams() {
  return i18n.languages.map((lang) => ({ lang }));
}

export default async function HomePage({
  params,
}: {
  params: Promise<{ lang: Locale }>;
}) {
  const { lang } = await params;
  const t = content[lang] ?? content.en;

  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-8">
      <div className="max-w-4xl text-center">
        <div className="mb-6 flex justify-center">
          <Image
            src="/bingsan-logo.png"
            alt="Bingsan Logo"
            width={120}
            height={120}
            className="rounded-2xl"
            priority
          />
        </div>
        <h1 className="mb-4 text-5xl font-bold">{t.title}</h1>
        <p className="mb-2 text-xl text-fd-muted-foreground">{t.subtitle}</p>
        <p className="mb-8 text-fd-muted-foreground">{t.description}</p>

        <div className="mb-12 flex justify-center gap-4">
          <Link
            href={`/${lang}/docs/getting-started`}
            className="rounded-lg bg-fd-primary px-6 py-3 font-medium text-fd-primary-foreground hover:bg-fd-primary/90"
          >
            {t.getStarted}
          </Link>
          <Link
            href={`/${lang}/docs/api`}
            className="rounded-lg border border-fd-border px-6 py-3 font-medium hover:bg-fd-accent"
          >
            {t.apiReference}
          </Link>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          {t.features.map((feature, index) => (
            <div
              key={index}
              className="rounded-lg border border-fd-border p-6 text-left"
            >
              <h3 className="mb-2 font-semibold">{feature.title}</h3>
              <p className="text-sm text-fd-muted-foreground">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
