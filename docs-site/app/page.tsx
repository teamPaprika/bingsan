'use client';

import { useEffect } from 'react';

export default function RootPage() {
  useEffect(() => {
    // Client-side redirect for when JavaScript is enabled
    const basePath = process.env.NEXT_PUBLIC_BASE_PATH || '';
    window.location.href = `${basePath}/en`;
  }, []);

  const basePath = process.env.NEXT_PUBLIC_BASE_PATH || '';

  return (
    <>
      {/* Meta refresh for when JavaScript is disabled */}
      <meta httpEquiv="refresh" content={`0; url=${basePath}/en`} />
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <p>Redirecting to documentation...</p>
        <p>
          If you are not redirected automatically, please{' '}
          <a href={`${basePath}/en`}>click here</a>.
        </p>
      </div>
    </>
  );
}
