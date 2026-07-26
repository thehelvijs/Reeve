import { useState } from 'react';
import { Button, ErrorText, Form } from './ui';
import Modal from './Modal';

// ConfirmModal is the one gate in front of an action that cannot be undone with
// the same click that caused it: a reboot, a stop, a delete. It keeps the
// dialog open when the action fails, because a modal that closes on a 403 or an
// offline host hides the only report the caller gets.
//
// Enter confirms and Escape cancels, like every other modal in the app.
export default function ConfirmModal({
  title,
  body,
  confirmLabel,
  onConfirm,
  onClose,
}: {
  title: string;
  body: string;
  confirmLabel: string;
  onConfirm: () => Promise<void> | void;
  onClose: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const go = async () => {
    setBusy(true);
    setError('');
    try {
      await onConfirm();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'that did not work');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal title={title} onClose={onClose} size="md" onSubmit={go}>
      <Form onSubmit={go}>
        <p className="mt-2 text-sm text-muted">{body}</p>
        <ErrorText>{error}</ErrorText>
        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button type="submit" variant="danger" disabled={busy}>
            {confirmLabel}
          </Button>
        </div>
      </Form>
    </Modal>
  );
}
