import { useState } from 'react'
import { IconButton } from './Button'
import { toast } from './Toast'

export function CopyField({ value, mask }: { value: string; mask?: boolean }) {
  const [shown, setShown] = useState(!mask)
  const display = shown ? value : '•'.repeat(Math.min(value.length, 24))
  return (
    <div className="copy-field">
      <code title={value}>{display}</code>
      {mask && (
        <IconButton onClick={() => setShown((s) => !s)} title={shown ? 'Hide' : 'Show'}>
          {shown ? '🙈' : '👁'}
        </IconButton>
      )}
      <IconButton
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(value)
            toast.success('Copied')
          } catch {
            toast.error('Copy failed')
          }
        }}
        title="Copy"
      >
        ⧉
      </IconButton>
    </div>
  )
}

export async function copyToClipboard(text: string, label = 'Copied') {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(label)
  } catch {
    toast.error('Copy failed')
  }
}

export function downloadFile(filename: string, content: string, mime = 'text/plain;charset=utf-8') {
  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
