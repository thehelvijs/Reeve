import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { api, type User } from './api';

interface AuthStatus {
  setup_required: boolean;
  signup_enabled: boolean;
  password_reset_enabled: boolean;
  google_enabled: boolean;
}

interface AuthState {
  user: User | null;
  loading: boolean;
  status: AuthStatus;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
  setUser: (u: User | null) => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<AuthStatus>({
    setup_required: false,
    signup_enabled: true,
    password_reset_enabled: false,
    google_enabled: false,
  });

  useEffect(() => {
    Promise.all([
      api.get<User>('/api/me').then(setUser).catch(() => setUser(null)),
      api.get<AuthStatus>('/api/auth/status').then(setStatus).catch(() => undefined),
    ]).finally(() => setLoading(false));
  }, []);

  const login = async (email: string, password: string) => {
    setUser(await api.post<User>('/api/auth/login', { email, password }));
  };
  const signup = async (email: string, password: string) => {
    setUser(await api.post<User>('/api/auth/signup', { email, password }));
    setStatus((s) => ({ ...s, setup_required: false }));
  };
  const logout = async () => {
    await api.post('/api/auth/logout');
    setUser(null);
  };
  const refreshUser = async () => {
    setUser(await api.get<User>('/api/me'));
  };

  return (
    <AuthContext.Provider value={{ user, loading, status, login, signup, logout, refreshUser, setUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return ctx;
}
