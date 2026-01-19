import { useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';

export default function Callback() {
  const [searchParams] = useSearchParams();

  useEffect(() => {
    const provider = searchParams.get('provider');
    const status = searchParams.get('status');

    if (window.opener && status === 'success' && provider) {
      window.opener.postMessage({ type: 'oauth-success', provider }, '*');
      window.close();
    }
  }, [searchParams]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-slate-950 p-4 text-white">
      <h1 className="mb-2 font-bold text-2xl">Authentication Successful</h1>
      <p className="text-slate-400">You can close this window now.</p>
    </div>
  );
}
