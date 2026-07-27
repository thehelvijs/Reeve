import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api, ApiError } from '../api';
import { Button, ErrorText, Field, Input } from '../components/ui';

// Shared chrome for the two off-session password screens.
function AuthShell({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center px-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <span className="rounded-button bg-accent px-1.5 text-sm font-semibold text-accent-fg">Reeve</span>
          <h1 className="mt-3 text-2xl font-semibold tracking-tight text-content">{title}</h1>
          <p className="mt-1.5 text-sm text-muted">{subtitle}</p>
        </div>
        {children}
      </div>
    </div>
  );
}

export function ForgotPassword() {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await api.post('/api/auth/forgot', { email });
      setSent(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'something went wrong');
    } finally {
      setBusy(false);
    }
  };

  if (sent) {
    return (
      <AuthShell title="Check your email" subtitle="If that address has an account, a reset link is on its way.">
        <p className="rounded-card border border-hairline bg-surface-1 p-6 text-sm text-muted">
          The link works once and expires in an hour. No email? Ask an admin to reset your password instead.
        </p>
        <p className="mt-4 text-center text-sm text-muted">
          <Link to="/login" className="text-link underline underline-offset-2 hover:text-link-hover">
            Back to sign in
          </Link>
        </p>
      </AuthShell>
    );
  }

  return (
    <AuthShell title="Reset your password" subtitle="We'll email you a link to choose a new one.">
      <form onSubmit={submit} className="space-y-4 rounded-card border border-hairline bg-surface-1 p-6">
        <Field label="Email">
          <Input type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </Field>
        <ErrorText>{error}</ErrorText>
        <Button type="submit" className="w-full" disabled={busy}>
          {busy ? 'Sending…' : 'Send reset link'}
        </Button>
      </form>
      <p className="mt-4 text-center text-sm text-muted">
        <Link to="/login" className="text-link underline underline-offset-2 hover:text-link-hover">
          Back to sign in
        </Link>
      </p>
    </AuthShell>
  );
}

export function ResetPassword() {
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';
  const navigate = useNavigate();
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await api.post('/api/auth/reset', { token, password });
      navigate('/login');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'something went wrong');
    } finally {
      setBusy(false);
    }
  };

  if (!token) {
    return (
      <AuthShell title="Link incomplete" subtitle="That reset link is missing its token.">
        <p className="mt-4 text-center text-sm text-muted">
          <Link to="/forgot" className="text-link underline underline-offset-2 hover:text-link-hover">
            Request a new link
          </Link>
        </p>
      </AuthShell>
    );
  }

  return (
    <AuthShell title="Choose a new password" subtitle="This signs you out everywhere else.">
      <form onSubmit={submit} className="space-y-4 rounded-card border border-hairline bg-surface-1 p-6">
        <Field label="New password">
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
          {busy ? 'Saving…' : 'Set password'}
        </Button>
      </form>
      <p className="mt-4 text-center text-sm text-muted">
        <Link to="/login" className="text-link underline underline-offset-2 hover:text-link-hover">
          Back to sign in
        </Link>
      </p>
    </AuthShell>
  );
}
