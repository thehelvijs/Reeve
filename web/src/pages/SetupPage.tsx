import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../auth';
import { ApiError } from '../api';
import { Button, ErrorText, Field, Input } from '../components/ui';

// SetupPage is the first-run screen: no users exist yet, so this creates the
// administrator account.
export default function SetupPage() {
  const { signup } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await signup(email, password);
      navigate('/');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'something went wrong');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-6">
      <div className="w-full max-w-md">
        <div className="mb-6 text-center">
          <span className="text-sm font-medium tracking-tight text-accent">Reeve</span>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight text-content">
            Set up your instance
          </h1>
          <p className="mt-2 text-sm text-muted">
            No accounts exist yet. The first account you create becomes the
            <span className="text-content"> administrator</span> with full control over users,
            tools, hosts, and settings.
          </p>
        </div>

        <form
          onSubmit={submit}
          className="space-y-4 rounded-card border border-hairline bg-surface-1 p-6"
        >
          <div className="flex items-center gap-2 rounded-button border border-accent/30 bg-surface-2 px-3 py-2">
            <span className="text-xs text-accent">●</span>
            <span className="text-xs text-muted">
              This is the admin account. Keep these credentials safe.
            </span>
          </div>
          <Field label="Admin email">
            <Input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </Field>
          <Field label="Password (min 8 characters)">
            <Input
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </Field>
          <ErrorText>{error}</ErrorText>
          <Button type="submit" className="w-full" disabled={busy}>
            {busy ? 'Creating…' : 'Create admin account'}
          </Button>
        </form>
      </div>
    </div>
  );
}
