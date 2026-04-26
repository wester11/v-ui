import { ButtonHTMLAttributes, ReactNode } from 'react'

type Variant = 'default' | 'primary' | 'success' | 'danger' | 'ghost'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: 'sm' | 'md'
  icon?: ReactNode
  loading?: boolean
}

const variantClass: Record<Variant, string> = {
  default: 'btn',
  primary: 'btn btn-primary',
  success: 'btn btn-success',
  danger:  'btn btn-danger',
  ghost:   'btn btn-ghost',
}

export function Button({
  variant = 'default',
  size = 'md',
  icon,
  loading,
  children,
  disabled,
  className = '',
  ...rest
}: ButtonProps) {
  const cls = [variantClass[variant], size === 'sm' ? 'btn-sm' : '', className].filter(Boolean).join(' ')
  return (
    <button {...rest} disabled={disabled || loading} className={cls}>
      {loading ? <span className="spin">⟳</span> : icon}
      {children}
    </button>
  )
}

export function IconButton({
  children,
  className = '',
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button {...rest} className={`btn-icon ${className}`}>
      {children}
    </button>
  )
}
