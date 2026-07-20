// Central toast helper. The group key is `type:message`, so ONLY identical
// toasts stack (with a count) — a success can never merge into an error, and
// vice versa. One place to enrich toast messaging later.
import { Notify } from 'quasar'

type ToastType = 'positive' | 'negative' | 'warning' | 'info'

export function toast(type: ToastType, message: string) {
  Notify.create({ type, message, group: `${type}:${message}` })
}
