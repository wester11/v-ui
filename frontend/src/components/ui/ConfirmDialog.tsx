import { ReactNode } from 'react'
import { Modal } from './Modal'
import { Button } from './Button'

interface ConfirmProps {
  open: boolean
  title: string
  body?: ReactNode
  confirmText?: string
  destructive?: boolean
  onConfirm: () => void
  onClose: () => void
  loading?: boolean
}

export function ConfirmDialog({
  open, title, body, confirmText = 'Confirm', destructive, onConfirm, onClose, loading,
}: ConfirmProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={loading}>Cancel</Button>
          <Button
            variant={destructive ? 'danger' : 'primary'}
            onClick={onConfirm}
            loading={loading}
          >
            {confirmText}
          </Button>
        </>
      }
    >
      <div className="text-dim">{body}</div>
    </Modal>
  )
}
