import { useRef, useState } from 'react';
import { api, uploadIcon, ApiError } from '../api';
import { Button, ErrorText } from './ui';
import EntityIcon from './EntityIcon';

function message(e: unknown): string {
  if (e instanceof ApiError) {
    return e.message;
  }
  return 'upload failed';
}

// IconUploader shows the current icon with upload/remove controls. path is the
// entity's icon endpoint (POST to set, DELETE to clear).
export default function IconUploader({
  url,
  name,
  path,
  onChange,
}: {
  url?: string;
  name: string;
  path: string;
  onChange: () => void;
}) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [error, setError] = useState('');

  const pick = async (file: File) => {
    setError('');
    try {
      await uploadIcon(path, file);
      onChange();
    } catch (e) {
      setError(message(e));
    }
  };
  const remove = async () => {
    setError('');
    try {
      await api.del(path);
      onChange();
    } catch (e) {
      setError(message(e));
    }
  };

  return (
    <div>
      <div className="flex items-center gap-4">
        <EntityIcon url={url} name={name} size={56} />
        <div className="flex gap-2">
          <Button type="button" variant="secondary" onClick={() => fileRef.current?.click()}>
            Upload
          </Button>
          {url && (
            <Button type="button" variant="danger" onClick={remove}>
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
                pick(f);
              }
              e.target.value = '';
            }}
          />
        </div>
      </div>
      <p className="mt-3 text-xs text-muted">PNG, JPEG, or WebP, up to 1 MB.</p>
      <ErrorText>{error}</ErrorText>
    </div>
  );
}
