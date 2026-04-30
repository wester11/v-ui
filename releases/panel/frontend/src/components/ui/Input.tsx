import { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes, forwardRef } from 'react'

interface FieldProps {
  label?: string
  hint?: string
  error?: string
}

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement> & FieldProps>(
  function Input({ label, hint, error, className = '', ...rest }, ref) {
    return (
      <div className="stack-sm">
        {label && <label className="label">{label}</label>}
        <input ref={ref} className={`input ${className}`} {...rest} />
        {error ? <span className="text-xs text-danger">{error}</span> : hint ? <span className="text-xs text-mute">{hint}</span> : null}
      </div>
    )
  }
)

export function Select({
  label, hint, children, className = '', ...rest
}: SelectHTMLAttributes<HTMLSelectElement> & FieldProps) {
  return (
    <div className="stack-sm">
      {label && <label className="label">{label}</label>}
      <select className={`select ${className}`} {...rest}>
        {children}
      </select>
      {hint && <span className="text-xs text-mute">{hint}</span>}
    </div>
  )
}

export function Textarea({
  label, hint, className = '', ...rest
}: TextareaHTMLAttributes<HTMLTextAreaElement> & FieldProps) {
  return (
    <div className="stack-sm">
      {label && <label className="label">{label}</label>}
      <textarea className={`textarea ${className}`} {...rest} />
      {hint && <span className="text-xs text-mute">{hint}</span>}
    </div>
  )
}
