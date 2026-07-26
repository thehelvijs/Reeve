import { useState } from 'react';
import type { RevealedCredential } from '../api';
import { Button } from './ui';
import Modal from './Modal';

// RevealModal shows a decrypted secret. It is never persisted client-side beyond
// this component's lifetime.
export default function RevealModal({ cred, onClose }: { cred: RevealedCredential; onClose: () => void }) {
  const typePill = (
    <span className="rounded-pill border border-hairline bg-surface-2 px-2 py-0.5 text-xs text-muted">
      {cred.type}
    </span>
  );
  return (
    <Modal title={cred.label || cred.type} onClose={onClose} size="md" headerRight={typePill} onSubmit={onClose}>
      <div className="mt-4 space-y-3">
        {Object.entries(cred.secret).map(([k, v]) => (
          <SecretField key={k} name={k} value={v} />
        ))}
      </div>
      <div className="mt-5 flex justify-end">
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
}

function SecretField({ name, value }: { name: string; value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <div>
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-muted">{name.replace('_', ' ')}</span>
        <button onClick={copy} className="text-xs text-muted hover:text-accent">
          {copied ? 'copied' : 'copy'}
        </button>
      </div>
      <pre className="mt-1 overflow-x-auto rounded-button border border-hairline bg-surface-2 px-3 py-2 font-mono text-xs text-content">
        {value}
      </pre>
    </div>
  );
}
