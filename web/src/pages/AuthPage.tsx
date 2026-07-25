import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../auth';
import { ApiError } from '../api';
import { Button, ErrorText, Field, Input } from '../components/ui';

export default function AuthPage({ mode }: { mode: 'login' | 'signup' }) {
  const { login, signup, status } = useAuth();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState(oauthErrorMessage(params.get('oauth_error')));
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      if (mode === 'signup') {
        await signup(email, password);
      } else {
        await login(email, password);
      }
      navigate('/');
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'something went wrong');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <span className="text-sm font-medium tracking-tight text-accent">Reeve</span>
          <h1 className="mt-3 text-2xl font-semibold tracking-tight text-content">
            {mode === 'signup' ? 'Create your account' : 'Sign in'}
          </h1>
          <p className="mt-1.5 text-sm text-muted">Your infrastructure catalog, owned and secured locally.</p>
        </div>
        <form onSubmit={submit} className="space-y-4 rounded-card border border-hairline bg-surface-1 p-6">
          <Field label="Email">
            <Input
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </Field>
          <Field label="Password">
            <Input
              type="password"
              autoComplete={mode === 'signup' ? 'new-password' : 'current-password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </Field>
          <ErrorText>{error}</ErrorText>
          <Button type="submit" className="w-full" disabled={busy}>
            {busy ? 'Please wait…' : mode === 'signup' ? 'Create account' : 'Sign in'}
          </Button>
          {mode === 'login' && status.google_enabled && (
            <>
              <div className="flex items-center gap-3">
                <span className="h-px flex-1 bg-hairline" />
                <span className="text-xs text-muted">or</span>
                <span className="h-px flex-1 bg-hairline" />
              </div>
              <a
                href="/api/v1/auth/google/start"
                className="flex w-full items-center justify-center gap-2 rounded-button border border-hairline px-3 py-2 text-sm text-content transition-colors hover:bg-surface-2"
              >
                <GoogleMark />
                Continue with Google
              </a>
            </>
          )}
          {mode === 'login' && status.password_reset_enabled && (
            <p className="text-center text-sm">
              <Link to="/forgot" className="text-muted hover:text-accent">
                Forgot your password?
              </Link>
            </p>
          )}
        </form>
        <p className="mt-4 text-center text-sm text-muted">
          {mode === 'signup' ? (
            <>
              Already have an account?{' '}
              <Link to="/login" className="text-content hover:text-accent">
                Sign in
              </Link>
            </>
          ) : (
            status.signup_enabled && (
              <>
                Need an account?{' '}
                <Link to="/signup" className="text-content hover:text-accent">
                  Sign up
                </Link>
              </>
            )
          )}
        </p>
      </div>
    </div>
  );
}

// oauthErrorMessage turns a callback error code into something readable; the
// OAuth flow returns through a browser redirect, not the JSON client.
function oauthErrorMessage(code: string | null): string {
  if (!code) {
    return '';
  }
  const messages: Record<string, string> = {
    google_not_configured: 'Google sign-in is not configured on this server.',
    state_mismatch: 'That sign-in attempt expired. Try again.',
    state_failed: 'Could not start sign-in. Try again.',
    provider_denied: 'Google cancelled the sign-in.',
    exchange_failed: 'Google rejected this server’s credentials.',
    profile_failed: 'Google did not return a verified email address.',
    domain_not_allowed: 'That email domain is not allowed to sign in here.',
    no_account: 'No account exists for that address, and sign-up is closed.',
    account_disabled: 'That account is deactivated.',
    provision_failed: 'Could not create an account for that address.',
  };
  return messages[code] ?? 'Sign-in failed.';
}

function GoogleMark() {
  return (
    <svg className="h-4 w-4" viewBox="0 0 18 18" aria-hidden>
      <path
        fill="#4285F4"
        d="M17.64 9.2c0-.64-.06-1.25-.16-1.84H9v3.48h4.84a4.14 4.14 0 0 1-1.8 2.72v2.26h2.92c1.71-1.57 2.68-3.89 2.68-6.62Z"
      />
      <path
        fill="#34A853"
        d="M9 18c2.43 0 4.47-.8 5.96-2.18l-2.92-2.26c-.81.54-1.84.86-3.04.86-2.34 0-4.32-1.58-5.03-3.71H.96v2.34A8.99 8.99 0 0 0 9 18Z"
      />
      <path fill="#FBBC05" d="M3.97 10.71a5.4 5.4 0 0 1 0-3.42V4.95H.96a9 9 0 0 0 0 8.1l3.01-2.34Z" />
      <path
        fill="#EA4335"
        d="M9 3.58c1.32 0 2.5.45 3.44 1.35l2.58-2.59C13.46.89 11.43 0 9 0A8.99 8.99 0 0 0 .96 4.95l3.01 2.34C4.68 5.16 6.66 3.58 9 3.58Z"
      />
    </svg>
  );
}
