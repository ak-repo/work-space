import type { ToastItem } from '../../types/api'

const icons = {
  success: 'ti-circle-check',
  danger: 'ti-alert-circle',
  warning: 'ti-alert-triangle',
  info: 'ti-info-circle',
}

export function ToastViewport({ toasts }: { toasts: ToastItem[] }) {
  return (
    <div className="toast-wrap" aria-live="polite">
      {toasts.map((toast) => (
        <div className={`toast show ${toast.kind}`} key={toast.id}>
          <i className={`ti ${icons[toast.kind]}`} aria-hidden="true" />
          {toast.message}
        </div>
      ))}
    </div>
  )
}
