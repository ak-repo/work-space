import { apiUrl } from '../config'
import type { StatusUpdate } from '../types/api'

export function connectEvents(onUpdate: (update: StatusUpdate) => void, onError: () => void) {
  const source = new EventSource(apiUrl('/events'))

  source.addEventListener('update', (event) => {
    try {
      onUpdate(JSON.parse(event.data) as StatusUpdate)
    } catch {
      // Ignore malformed events and wait for the next backend update.
    }
  })

  source.onerror = () => {
    source.close()
    onError()
  }

  return () => source.close()
}
