import { useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, uploadAvatar, ApiError } from '../api';
import { useAuth } from '../auth';
import { Button, Card, Field, Form, Input, ErrorText } from '../components/ui';
import PageHeader from '../components/PageHeader';
import Avatar from '../components/Avatar';

function message(e: unknown): string {
  if (e instanceof ApiError) {
    return e.message;
  }
  return 'something went wrong';
}

export default function Profile() {
  const { user, refreshUser, setUser } = useAuth();
  const navigate = useNavigate();
  const fileRef = useRef<HTMLInputElement>(null);

  const [name, setName] = useState(user?.display_name ?? '');
  const [nameErr, setNameErr] = useState('');
  const [nameSaved, setNameSaved] = useState(false);

  const [avatarErr, setAvatarErr] = useState('');

  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [pwErr, setPwErr] = useState('');
  const [pwSaved, setPwSaved] = useState(false);

  const [confirmDelete, setConfirmDelete] = useState('');
  const [deleteErr, setDeleteErr] = useState('');

  if (!user) {
    return null;
  }

  const saveName = async () => {
    setNameErr('');
    setNameSaved(false);
    try {
      await api.patch('/api/v1/me', { display_name: name });
      await refreshUser();
      setNameSaved(true);
    } catch (e) {
      setNameErr(message(e));
    }
  };

  const onPickFile = async (file: File) => {
    setAvatarErr('');
    try {
      await uploadAvatar(file);
      await refreshUser();
    } catch (e) {
      setAvatarErr(message(e));
    }
  };

  const removeAvatar = async () => {
    setAvatarErr('');
    try {
      await api.del('/api/v1/me/avatar');
      await refreshUser();
    } catch (e) {
      setAvatarErr(message(e));
    }
  };

  const changePassword = async () => {
    setPwErr('');
    setPwSaved(false);
    if (next !== confirm) {
      setPwErr('new passwords do not match');
      return;
    }
    try {
      await api.post('/api/v1/me/password', { current_password: current, new_password: next });
      setCurrent('');
      setNext('');
      setConfirm('');
      setPwSaved(true);
    } catch (e) {
      setPwErr(message(e));
    }
  };

  const deleteAccount = async () => {
    setDeleteErr('');
    try {
      await api.del('/api/v1/me');
      setUser(null);
      navigate('/');
    } catch (e) {
      setDeleteErr(message(e));
    }
  };

  return (
    <div>
      <PageHeader title="Profile" subtitle="Your account, avatar, and password." />

      <div className="mt-6 space-y-4">
        <Card className="p-5">
          <h2 className="text-sm font-medium text-content">Avatar</h2>
          <div className="mt-4 flex items-center gap-4">
            <Avatar url={user.avatar_url} name={user.display_name} email={user.email} size={64} />
            <div className="flex gap-2">
              <Button variant="secondary" onClick={() => fileRef.current?.click()}>
                Upload
              </Button>
              {user.avatar_url && (
                <Button variant="danger" onClick={removeAvatar}>
                  Remove
                </Button>
              )}
              <input
                ref={fileRef}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (f) {
                    onPickFile(f);
                  }
                  e.target.value = '';
                }}
              />
            </div>
          </div>
          <p className="mt-3 text-xs text-muted">PNG, JPEG, or WebP, up to 1 MB.</p>
          <ErrorText>{avatarErr}</ErrorText>
        </Card>

        <Card className="p-5">
          <h2 className="text-sm font-medium text-content">Display name</h2>
          <Form onSubmit={saveName} className="mt-4 flex items-end gap-3">
            <div className="max-w-xs flex-1">
              <Field label="Shown across the app; falls back to your email">
                <Input
                  value={name}
                  maxLength={60}
                  placeholder={user.email}
                  onChange={(e) => {
                    setName(e.target.value);
                    setNameSaved(false);
                  }}
                />
              </Field>
            </div>
            <Button type="submit">Save</Button>
          </Form>
          {nameSaved && <p className="mt-2 text-xs text-muted">Saved.</p>}
          <ErrorText>{nameErr}</ErrorText>
        </Card>

        <Card className="p-5">
          <h2 className="text-sm font-medium text-content">Change password</h2>
          <Form onSubmit={changePassword} className="mt-4 max-w-xs space-y-3">
            <Field label="Current password">
              <Input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} />
            </Field>
            <Field label="New password">
              <Input type="password" value={next} onChange={(e) => setNext(e.target.value)} />
            </Field>
            <Field label="Confirm new password">
              <Input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
            </Field>
            <Button type="submit">Update password</Button>
          </Form>
          {pwSaved && <p className="mt-2 text-xs text-muted">Password updated.</p>}
          <ErrorText>{pwErr}</ErrorText>
        </Card>

        <Card className="border-red-500/20 p-5">
          <h2 className="text-sm font-medium text-content">Delete account</h2>
          <p className="mt-1 text-sm text-muted">
            Permanently deletes your account. Tools and credentials you own are reassigned to an admin.
            This cannot be undone.
          </p>
          <Form onSubmit={deleteAccount} className="mt-4 flex items-end gap-3">
            <div className="max-w-xs flex-1">
              <Field label={`Type ${user.email} to confirm`}>
                <Input value={confirmDelete} onChange={(e) => setConfirmDelete(e.target.value)} />
              </Field>
            </div>
            <Button type="submit" variant="danger" disabled={confirmDelete !== user.email}>
              Delete account
            </Button>
          </Form>
          <ErrorText>{deleteErr}</ErrorText>
        </Card>
      </div>
    </div>
  );
}
