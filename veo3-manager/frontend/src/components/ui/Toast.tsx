import { toast as sonnerToast, Toaster } from 'sonner';

export function toast(type: 'success' | 'error' | 'info', message: string) {
  switch (type) {
    case 'success':
      sonnerToast.success(message);
      break;
    case 'error':
      sonnerToast.error(message);
      break;
    case 'info':
      sonnerToast.info(message);
      break;
  }
}

export function ToastContainer() {
  return (
    <Toaster
      position="top-right"
      offset={48}
      toastOptions={{
        duration: 4000,
        style: {
          background: 'var(--ds-bg-elevated)',
          border: '1px solid var(--ds-border)',
          color: 'var(--ds-text-primary)',
          boxShadow: 'var(--ds-shadow-lg)',
          borderRadius: 'var(--ds-radius-lg)',
          fontSize: '0.8125rem',
        },
      }}
    />
  );
}
