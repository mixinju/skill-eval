import { useEffect, useRef } from 'react'
import { BACKEND_GAPS } from '../lib/gaps'

export default function GapsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const ref = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  return (
    <dialog ref={ref} className="se-dialog" onClose={onClose}>
      <header>
        <h2>数据从哪来</h2>
        <button type="button" className="se-icon-btn" onClick={onClose}>关闭</button>
      </header>
      <div className="se-gap-list">
        {BACKEND_GAPS.map((gap) => (
          <section key={gap.title}>
            <h3>{gap.title}</h3>
            <p>{gap.body}</p>
          </section>
        ))}
      </div>
    </dialog>
  )
}
