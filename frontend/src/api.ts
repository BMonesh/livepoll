export type User = { email: string }

export type PollOption = { id: string; label: string; count: number }

export type Poll = {
  slug: string
  question: string
  options: PollOption[]
  total: number
  closed: boolean
  revision: number
  createdAt: string
}

export type PollView = Poll & { selectedOptionId?: string }

export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message)
  }
}

async function request<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch('/api' + path, body === undefined ? undefined : {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    throw new ApiError(payload?.error ?? 'The request failed. Please try again.', response.status)
  }
  return response.status === 204 ? undefined as T : response.json()
}

export const api = {
  auth: (mode: 'signup' | 'login', email: string, password: string) =>
    request<User>('/auth/' + mode, { email, password }),
  me: () => request<User>('/auth/me'),
  logout: () => request<void>('/auth/logout', {}),
  list: (before?: string) => request<Poll[]>('/creator/polls' + (before ? '?before=' + encodeURIComponent(before) : '')),
  create: (question: string, options: string[]) => request<Poll>('/creator/polls', { question, options }),
  close: (slug: string) => request<Poll>('/creator/polls/' + encodeURIComponent(slug) + '/close', {}),
  poll: async (slug: string): Promise<PollView> => {
    const load = () => request<PollView>('/polls/' + encodeURIComponent(slug))
    // Bootstrap the shared voter cookie once if two tabs open at the same time.
    return navigator.locks ? navigator.locks.request('livepoll-voter-cookie', load) : load()
  },
  vote: (slug: string, optionId: string) =>
    request<PollView>('/polls/' + encodeURIComponent(slug) + '/vote', { optionId }),
}

export function message(error: unknown) {
  return error instanceof ApiError ? error.message : 'Could not reach LivePoll. Please try again.'
}
