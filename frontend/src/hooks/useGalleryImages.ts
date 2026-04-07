import { useCallback, useRef, useState } from "react"
import { fetchGalleryImages } from "@/api/endpoints"
import type { GalleryImageDTO, GalleryImagesResponse } from "@/types"

const PAGE_SIZE = 50

export function useGalleryImages(view: string) {
  const [images, setImages] = useState<GalleryImageDTO[]>([])
  const [totalImages, setTotalImages] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const pageRef = useRef(1)
  const viewRef = useRef(view)

  // Track if we've done at least one load
  const [initialized, setInitialized] = useState(false)

  // Prefetch buffer for the next page
  const prefetchRef = useRef<{
    page: number
    view: string
    promise: Promise<GalleryImagesResponse> | null
    data: GalleryImagesResponse | null
  }>({ page: 0, view: "", promise: null, data: null })

  const startPrefetch = useCallback((page: number, currentView: string) => {
    const buf = prefetchRef.current
    if (buf.page === page && buf.view === currentView && (buf.data || buf.promise)) {
      return // already prefetching/prefetched this page
    }
    buf.page = page
    buf.view = currentView
    buf.data = null
    buf.promise = fetchGalleryImages(page, PAGE_SIZE, currentView)
      .then((result) => {
        // Only store if still relevant (view hasn't changed)
        if (prefetchRef.current.page === page && prefetchRef.current.view === currentView) {
          prefetchRef.current.data = result
        }
        return result
      })
      .catch(() => {
        // Silently discard prefetch errors; the real load will handle them
        prefetchRef.current.promise = null
        return null as unknown as GalleryImagesResponse
      })
  }, [])

  const consumePrefetch = useCallback((page: number, currentView: string): GalleryImagesResponse | null => {
    const buf = prefetchRef.current
    if (buf.page === page && buf.view === currentView && buf.data) {
      const data = buf.data
      buf.page = 0
      buf.data = null
      buf.promise = null
      return data
    }
    return null
  }, [])

  const loadMore = useCallback(async () => {
    if (isLoading) return
    setIsLoading(true)
    setError(null)
    try {
      const currentPage = pageRef.current
      const currentView = viewRef.current

      // Use prefetched data if available
      const prefetched = consumePrefetch(currentPage, currentView)
      const result = prefetched ?? await fetchGalleryImages(currentPage, PAGE_SIZE, currentView)

      setImages((prev) => {
        // Avoid duplicates by checking IDs
        const existingIds = new Set(prev.map((img) => img.id))
        const newImages = result.images.filter((img) => !existingIds.has(img.id))
        return [...prev, ...newImages]
      })
      setTotalImages(result.totalImages)
      setHasMore(result.hasNextPage)
      pageRef.current += 1
      setInitialized(true)

      // Prefetch the next page in background
      if (result.hasNextPage) {
        startPrefetch(pageRef.current, currentView)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load images")
    } finally {
      setIsLoading(false)
    }
  }, [isLoading, consumePrefetch, startPrefetch])

  const reset = useCallback(
    (newView?: string) => {
      if (newView !== undefined) {
        viewRef.current = newView
      }
      setImages([])
      setTotalImages(0)
      setHasMore(true)
      setError(null)
      pageRef.current = 1
      setInitialized(false)
      // Invalidate prefetch buffer
      prefetchRef.current = { page: 0, view: "", promise: null, data: null }
    },
    []
  )

  return {
    images,
    totalImages,
    hasMore,
    isLoading,
    error,
    initialized,
    loadMore,
    reset,
  }
}
