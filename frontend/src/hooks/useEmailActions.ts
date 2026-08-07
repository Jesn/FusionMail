import { useState, useCallback } from 'react';

interface DialogState {
  markAllRead: boolean;
  delete: boolean;
  permanentDelete: boolean;
  emptyTrash: boolean;
}

const initialDialogState: DialogState = {
  markAllRead: false,
  delete: false,
  permanentDelete: false,
  emptyTrash: false,
};

export function useEmailActions() {
  const [dialogs, setDialogs] = useState<DialogState>(initialDialogState);
  const [isDeleting, setIsDeleting] = useState(false);

  const openDialog = useCallback((key: keyof DialogState) => {
    setDialogs((prev) => ({ ...prev, [key]: true }));
  }, []);

  const closeDialog = useCallback((key: keyof DialogState) => {
    setDialogs((prev) => ({ ...prev, [key]: false }));
  }, []);

  return {
    dialogs,
    isDeleting,
    setIsDeleting,
    openDialog,
    closeDialog,
  };
}