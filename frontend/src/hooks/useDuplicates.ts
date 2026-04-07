import { useCallback, useEffect, useRef, useState } from "react"
import { fetchDuplicates } from "@/api/endpoints"
import type { DuplicatesResponse } from "@/types"

interface PrefetchEntry {
  page: number
  pageSize: number
  data: DuplicatesResponse | null
  promise: Promise<DuplicatesResponse> | null
}

export function useDuplicates(page: number, pageSize: number) {
  const [data, setData] = useState<DuplicatesResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Prefetch buffer for the next page
  const prefetchRef = useRef<PrefetchEntry>({ page: 0, pageSize: 0, data: null, promise: null })

  const startPrefetch = useCallback((nextPage: number, size: number) => {
    const buf = prefetchRef.current
    if (buf.page === nextPage && buf.pageSize === size && (buf.data || buf.promise)) {
      return // already prefetching/prefetched
    }
    buf.page = nextPage
    buf.pageSize = size
    buf.data = null
    buf.promise = fetchDuplicates(nextPage, size)
      .then((result) => {
        if (prefetchRef.current.page === nextPage && prefetchRef.current.pageSize === size) {
          prefetchRef.current.data = result
        }
        return result
      })
      .catch(() => {
        prefetchRef.current.promise = null
        return null as unknown as DuplicatesResponse
      })
  }, [])

  const consumePrefetch = useCallback((targetPage: number, size: number): DuplicatesResponse | null => {
    const buf = prefetchRef.current
    if (buf.page === targetPage && buf.pageSize === size && buf.data) {
      const result = buf.data
      buf.page = 0
      buf.data = null
      buf.promise = null
      return result
    }
    return null
  }, [])

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      // Use prefetched data if available
      const prefetched = consumePrefetch(page, pageSize)
      const result = prefetched ?? await fetchDuplicates(page, pageSize)
      setData(result)

      // Prefetch the next page in background
      if (result.hasNextPage) {
        startPrefetch(page + 1, pageSize)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load duplicates")
    } finally {
      setIsLoading(false)
    }
  }, [page, pageSize, consumePrefetch, startPrefetch])

  useEffect(() => {
    load()
  }, [load])

  return { data, isLoading, error, refetch: load }
}
