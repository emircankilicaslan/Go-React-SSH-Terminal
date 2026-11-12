export const API_BASE = import.meta.env.VITE_API_URL || `http://${window.location.hostname}:8080`

export const WS_BASE = API_BASE.replace(/^http/, 'ws')

/** Kimlik doğrulamalı istek; 401 gelirse onUnauthorized çağrılır (oturum süresi dolmuş). */
export async function request(path, { token, method = 'GET', body, onUnauthorized } = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      ...(body ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401 && token && onUnauthorized) onUnauthorized()
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error || `İstek başarısız (${res.status})`)
  return data
}
