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
    <div className="flex flex-col items-center justify-center min-h-screen bg-slate-950 text-white p-4">
      <h1 className="text-2xl font-bold mb-2">Authentication Successful</h1>
      <p className="text-slate-400">You can close this window now.</p>
    </div>
  );
}
