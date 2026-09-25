export function EmptyState({ icon, message }: { icon: string; message: string }) {
  return (
    <div className="empty-state">
      <i className={`ti ${icon}`} aria-hidden="true" />
      <p>{message}</p>
    </div>
  )
}
