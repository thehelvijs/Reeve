import { useRef, useState } from 'react';
import { api, uploadIcon, ApiError } from '../api';
import { Button, ErrorText } from './ui';

function message(e: unknown): string {
  if (e instanceof ApiError) {
    return e.message;
  }
  return 'upload failed';
}

// ThumbnailUploader manages an entity's larger preview image (a wide banner),
// distinct from the small square icon. path is the thumbnail endpoint.
export default function ThumbnailUploader({
  url,
  path,
  onChange,
}: {
  url?: string;
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
      <div className="flex items-start gap-4">
        {url ? (
          <img
            src={url}
            alt=""
            className="h-24 w-40 shrink-0 rounded-card border border-hairline object-cover"
          />
        ) : (
          <div className="flex h-24 w-40 shrink-0 items-center justify-center rounded-card border border-dashed border-hairline text-xs text-muted">
            No thumbnail
          </div>
        )}
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
      <p className="mt-3 text-xs text-muted">A wider preview image. PNG, JPEG, or WebP, up to 1 MB.</p>
      <ErrorText>{error}</ErrorText>
    </div>
  );
}
