import { useState } from 'react';
import type { RevealedCredential } from '../api';
import { Button } from './ui';
import Modal from './Modal';
import { isSensitiveField, orderedSecretFields } from '../lib/secretFields';

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
        {orderedSecretFields(cred.secret).map(([k, v]) => (
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
  // A reveal is often done with someone looking over your shoulder, or on a
  // shared screen: the secret is fetched but not shown until it is asked for.
  const [shown, setShown] = useState(!isSensitiveField(name));
  const copy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <div>
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-muted">{name.replace('_', ' ')}</span>
        <div className="flex items-center gap-3">
          {isSensitiveField(name) && (
            <button type="button" onClick={() => setShown(!shown)} className="text-xs text-muted hover:text-link">
              {shown ? 'hide' : 'show'}
            </button>
          )}
          <button type="button" onClick={copy} className="text-xs text-muted hover:text-link">
            {copied ? 'copied' : 'copy'}
          </button>
        </div>
      </div>
      <pre className="mt-1 overflow-x-auto rounded-button border border-hairline bg-surface-2 px-3 py-2 font-mono text-xs text-content">
        {shown ? value : '•'.repeat(Math.min(value.length, 32))}
      </pre>
    </div>
  );
}
