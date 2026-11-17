import { useEffect, useState } from 'react'
import type { Poll } from '../api'

export type Connection = 'connecting' | 'live' | 'reconnecting'

export default function usePollEvents(slug: string, enabled: boolean, onResult: (poll: Poll) => void) {
  const [connection, setConnection] = useState<Connection>('connecting')

  useEffect(() => {
    if (!enabled) return
    const source = new EventSource('/api/polls/' + encodeURIComponent(slug) + '/events')
    source.addEventListener('results', event => {
      const poll = JSON.parse(event.data) as Poll
      onResult(poll)
      setConnection('live')
    })
    source.onerror = () => setConnection('reconnecting')
    return () => source.close()
  }, [slug, enabled, onResult])

  return connection
}
