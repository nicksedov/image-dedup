import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { fetchGalleryCalendar, fetchCalendarMonthInfo } from "@/api/endpoints"
import { Skeleton } from "@/components/ui/skeleton"
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from "lucide-react"
import { useTranslation } from "@/i18n"
import type { GalleryImageDTO, CalendarDateGroup, CalendarDateRange, CalendarMonthInfo } from "@/types"

interface GalleryCalendarViewProps {
  onImageClick: (image: GalleryImageDTO) => void
}

const PAGE_SIZE = 50
const HEADER_HEIGHT = 56 // px - height of the header to offset timeline

const MONTHS = [
  { value: 0, label: "Jan" },
  { value: 1, label: "Feb" },
  { value: 2, label: "Mar" },
  { value: 3, label: "Apr" },
  { value: 4, label: "May" },
  { value: 5, label: "Jun" },
  { value: 6, label: "Jul" },
  { value: 7, label: "Aug" },
  { value: 8, label: "Sep" },
  { value: 9, label: "Oct" },
  { value: 10, label: "Nov" },
  { value: 11, label: "Dec" },
]

export function GalleryCalendarView({ onImageClick }: GalleryCalendarViewProps) {
  const { t } = useTranslation()

  const [groups, setGroups] = useState<CalendarDateGroup[]>([])
  const [totalImages, setTotalImages] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [initialized, setInitialized] = useState(false)
  const [dateRange, setDateRange] = useState<CalendarDateRange>({ minDate: "", maxDate: "", totalWithDate: 0 })
  const [monthInfo, setMonthInfo] = useState<CalendarMonthInfo | null>(null)
  const [dayCounts, setDayCounts] = useState<Map<number, number>>(new Map()) // day -> count

  // Date filter state - now supports range selection
  const [dateRangeFilter, setDateRangeFilter] = useState<{ start: string | null; end: string | null }>({
    start: null,
    end: null,
  })
  const [rangeSelecting, setRangeSelecting] = useState(false)

  // Calendar widget state
  const [calendarViewDate, setCalendarViewDate] = useState(() => {
    return new Date()
  })

  const pageRef = useRef(1)
  const sentinelRef = useRef<HTMLDivElement>(null)
  const mainContentRef = useRef<HTMLDivElement>(null)
  
  // Image preloading
  const preloadImageCache = useRef<Map<string, HTMLImageElement>>(new Map())

  const calendarMonthKey = useMemo(() => {
    const y = calendarViewDate.getFullYear()
    const m = calendarViewDate.getMonth() + 1
    return `${y}-${String(m).padStart(2, "0")}`
  }, [calendarViewDate])

  // Fetch calendar data
  const loadPage = useCallback(async (page: number, reset = false) => {
    if (isLoading) return
    setIsLoading(true)
    setError(null)
    try {
      const result = await fetchGalleryCalendar(
        page,
        PAGE_SIZE,
        dateRangeFilter.start ?? undefined,
        dateRangeFilter.end ?? undefined,
        calendarMonthKey
      )

      if (reset) {
        setGroups(result.groups)
      } else {
        setGroups((prev) => [...prev, ...result.groups])
      }
      setTotalImages(result.totalImages)
      setHasMore(result.hasMore)

      // Update date range on first load
      if (page === 1) {
        setDateRange(result.dateRange)
        // Set calendar to the month of the oldest image (minDate) if not filtered
        if (!dateRangeFilter.start && result.dateRange.minDate) {
          const minDate = new Date(result.dateRange.minDate + "T00:00:00")
          setCalendarViewDate(new Date(minDate.getFullYear(), minDate.getMonth(), 1))
        }
      }

      // Update month info
      if (result.months.length > 0) {
        setMonthInfo(result.months[0])
      }

      setInitialized(true)
      pageRef.current = page + 1
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load images")
    } finally {
      setIsLoading(false)
    }
  }, [isLoading, dateRangeFilter.start, dateRangeFilter.end, calendarMonthKey])

  // Preload images for smoother scrolling
  const preloadImages = useCallback((imageUrls: string[]) => {
    const MAX_CACHE_SIZE = 100
    
    imageUrls.forEach((url) => {
      if (!url || preloadImageCache.current.has(url)) return
      
      // Limit cache size
      if (preloadImageCache.current.size >= MAX_CACHE_SIZE) {
        const firstKey = preloadImageCache.current.keys().next().value
        if (firstKey) {
          preloadImageCache.current.delete(firstKey)
        }
      }
      
      const img = new Image()
      img.src = url
      preloadImageCache.current.set(url, img)
    })
  }, [])

  // Preload images when groups change
  useEffect(() => {
    const imageUrls = groups.flatMap((group) => 
      group.images.map((img) => img.thumbnail).filter(Boolean) as string[]
    )
    
    // Preload with slight delay to not block initial render
    const timer = setTimeout(() => {
      preloadImages(imageUrls)
    }, 100)
    
    return () => clearTimeout(timer)
  }, [groups, preloadImages])

  // Preload next page images when approaching end of current content
  useEffect(() => {
    if (!hasMore || isLoading) return
    
    const checkAndPreloadNextPage = () => {
      if (!mainContentRef.current) return
      
      const scrollHeight = mainContentRef.current.scrollHeight
      const scrollTop = mainContentRef.current.scrollTop || window.scrollY
      const clientHeight = window.innerHeight
      
      // If user has scrolled past 50% of content, preload next page
      if (scrollTop + clientHeight > scrollHeight * 0.5) {
        const nextPage = pageRef.current
        
        // Preload next page images in background
        fetchGalleryCalendar(
          nextPage,
          PAGE_SIZE,
          dateRangeFilter.start ?? undefined,
          dateRangeFilter.end ?? undefined,
          calendarMonthKey
        )
          .then((result) => {
            const imageUrls = result.groups.flatMap((group) =>
              group.images.map((img) => img.thumbnail).filter(Boolean) as string[]
            )
            preloadImages(imageUrls)
          })
          .catch(() => {
            // Silently fail - next page will load normally when needed
          })
      }
    }
    
    window.addEventListener("scroll", checkAndPreloadNextPage, { passive: true })
    return () => window.removeEventListener("scroll", checkAndPreloadNextPage)
  }, [hasMore, isLoading, dateRangeFilter.start, dateRangeFilter.end, calendarMonthKey, preloadImages])

  // Initial load or reset when filter changes
  useEffect(() => {
    pageRef.current = 1
    setGroups([])
    setInitialized(false)
    loadPage(1, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dateRangeFilter.start, dateRangeFilter.end])

  // Load month info when calendar month changes (using lightweight endpoint)
  useEffect(() => {
    fetchCalendarMonthInfo(calendarMonthKey)
      .then((result) => {
        setMonthInfo({ year: result.year, month: result.month, days: result.days })
        // Build day count map
        const countMap = new Map<number, number>()
        result.dayCounts.forEach((dc) => {
          countMap.set(dc.day, dc.count)
        })
        setDayCounts(countMap)
      })
      .catch(() => {
        // Reset on error
        setMonthInfo(null)
        setDayCounts(new Map())
      })
  }, [calendarMonthKey])

  // Infinite scroll
  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !isLoading) {
          loadPage(pageRef.current)
        }
      },
      { threshold: 0.1 }
    )

    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [hasMore, isLoading, loadPage])

  // Horizontal calendar: generate all days of the month as a scrollable strip
  const calendarDays = useMemo(() => {
    const year = calendarViewDate.getFullYear()
    const month = calendarViewDate.getMonth()
    const lastDay = new Date(year, month + 1, 0)
    const daysInMonth = lastDay.getDate()

    const daysWithImages = new Set(monthInfo?.days ?? [])

    const days: { date: string; day: number; hasImages: boolean; imageCount: number }[] = []
    for (let d = 1; d <= daysInMonth; d++) {
      const dateStr = `${year}-${String(month + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`
      const count = dayCounts.get(d) ?? 0
      days.push({
        date: dateStr,
        day: d,
        hasImages: daysWithImages.has(d),
        imageCount: count,
      })
    }

    return days
  }, [calendarViewDate, monthInfo, dayCounts])

  const prevMonth = () => {
    setCalendarViewDate(new Date(calendarViewDate.getFullYear(), calendarViewDate.getMonth() - 1, 1))
  }

  const nextMonth = () => {
    setCalendarViewDate(new Date(calendarViewDate.getFullYear(), calendarViewDate.getMonth() + 1, 1))
  }

  const selectDate = (date: string) => {
    if (!rangeSelecting) {
      // Start range selection
      setDateRangeFilter({ start: date, end: null })
      setRangeSelecting(true)
    } else {
      // Complete range selection
      if (dateRangeFilter.start) {
        const startDate = dateRangeFilter.start
        // Ensure start <= end
        if (date >= startDate) {
          setDateRangeFilter({ start: startDate, end: date })
        } else {
          setDateRangeFilter({ start: date, end: startDate })
        }
      } else {
        setDateRangeFilter({ start: date, end: date })
      }
      setRangeSelecting(false)
    }
  }

  const isInSelectedRange = (date: string) => {
    if (!dateRangeFilter.start) return false
    if (!dateRangeFilter.end) return date === dateRangeFilter.start
    return date >= dateRangeFilter.start && date <= dateRangeFilter.end
  }

  const clearDateRangeFilter = () => {
    setDateRangeFilter({ start: null, end: null })
    setRangeSelecting(false)
  }

  // Visible date range for timeline
  const visibleDateRange = useMemo(() => {
    if (groups.length === 0) return { start: null, end: null }
    return {
      start: groups[0].date,
      end: groups[groups.length - 1].date,
    }
  }, [groups])

  return (
    <div className="space-y-4">
      {/* Header with image count */}
      <div className="flex items-center gap-2">
        <CalendarIcon className="h-5 w-5 text-muted-foreground" />
        <span className="text-sm text-muted-foreground">
          {dateRange.totalWithDate > 0
            ? (dateRange.totalWithDate === 1
              ? t("gallery.imageCountOne", { count: dateRange.totalWithDate.toLocaleString() })
              : t("gallery.imageCount", { count: dateRange.totalWithDate.toLocaleString() }))
            : t("gallery.calendar.noDateInfo")
          }
        </span>
      </div>

      {/* Horizontal Calendar Widget */}
      <div className="rounded-lg border bg-card p-3">
        {/* Month/Year selector */}
        <div className="flex items-center justify-between mb-2 gap-2">
          <button onClick={prevMonth} className="p-1 hover:bg-muted rounded flex-shrink-0">
            <ChevronLeft className="h-4 w-4" />
          </button>
          
          <div className="flex items-center gap-2">
            {/* Month dropdown */}
            <select
              value={calendarViewDate.getMonth()}
              onChange={(e) => {
                const newMonth = parseInt(e.target.value)
                setCalendarViewDate(new Date(calendarViewDate.getFullYear(), newMonth, 1))
              }}
              className="text-sm font-medium bg-background dark:bg-zinc-800 text-foreground dark:text-zinc-100 border border-border rounded px-2 py-1 outline-none cursor-pointer"
            >
              {MONTHS.map((m) => (
                <option key={m.value} value={m.value} className="bg-background dark:bg-zinc-800 text-foreground dark:text-zinc-100">
                  {new Date(2000, m.value, 1).toLocaleDateString(undefined, { month: "long" })}
                </option>
              ))}
            </select>

            {/* Year dropdown */}
            <select
              value={calendarViewDate.getFullYear()}
              onChange={(e) => {
                const newYear = parseInt(e.target.value)
                setCalendarViewDate(new Date(newYear, calendarViewDate.getMonth(), 1))
              }}
              className="text-sm font-medium bg-background dark:bg-zinc-800 text-foreground dark:text-zinc-100 border border-border rounded px-2 py-1 outline-none cursor-pointer"
            >
              {(() => {
                // Generate year range from dateRange or fallback to current year ±5
                const currentYear = calendarViewDate.getFullYear()
                let startYear = currentYear - 5
                let endYear = currentYear + 5
                
                // Use actual data range if available
                if (dateRange.minDate && dateRange.maxDate) {
                  const minYear = new Date(dateRange.minDate + "T00:00:00").getFullYear()
                  const maxYear = new Date(dateRange.maxDate + "T00:00:00").getFullYear()
                  startYear = minYear
                  endYear = maxYear
                }
                
                const years = []
                for (let y = startYear; y <= endYear; y++) {
                  years.push(y)
                }
                
                return years.map((year) => (
                  <option key={year} value={year} className="bg-background dark:bg-zinc-800 text-foreground dark:text-zinc-100">
                    {year}
                  </option>
                ))
              })()}
            </select>
          </div>

          <button onClick={nextMonth} className="p-1 hover:bg-muted rounded flex-shrink-0">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>

        {/* Horizontal scrollable day strip */}
        <div className="flex gap-1 overflow-x-auto pb-1 pt-2 scrollbar-thin" style={{ scrollbarWidth: "thin" }}>
          {calendarDays.map((day) => {
            const isSelected = isInSelectedRange(day.date)
            const isRangeStart = day.date === dateRangeFilter.start
            const isRangeEnd = day.date === dateRangeFilter.end && dateRangeFilter.end !== null
            
            return (
              <button
                key={day.date}
                disabled={!day.date}
                className={`
                  flex-shrink-0 w-9 h-9 flex flex-col items-center justify-center text-xs rounded-md
                  transition-all relative
                  ${isSelected
                    ? "bg-primary text-primary-foreground hover:bg-primary/90 font-medium cursor-pointer"
                    : day.hasImages 
                      ? "bg-emerald-100 dark:bg-emerald-900/30 hover:bg-emerald-200 dark:hover:bg-emerald-900/50 text-emerald-700 dark:text-emerald-300 font-medium cursor-pointer"
                      : "bg-red-50 dark:bg-red-900/20 text-muted-foreground/40 hover:bg-red-100 dark:hover:bg-red-900/30"
                  }
                  ${isRangeStart || isRangeEnd ? "ring-2 ring-primary ring-offset-2" : ""}
                  ${isSelected && !isRangeStart && !isRangeEnd ? "opacity-80" : ""}
                `}
                onClick={() => day.date && selectDate(day.date)}
                title={day.hasImages ? `${day.imageCount} ${day.imageCount === 1 ? "image" : "images"}` : "No images"}
              >
                <span className="text-[11px] leading-none">{day.day}</span>
              </button>
            )
          })}
        </div>

        {/* Date filter controls */}
        {(dateRangeFilter.start || dateRangeFilter.end) && (
          <div className="mt-2 pt-2 border-t flex items-center justify-between">
            <span className="text-xs text-muted-foreground">
              {rangeSelecting
                ? `Selecting: ${dateRangeFilter.start}${dateRangeFilter.end ? ` — ${dateRangeFilter.end}` : " (click end date)"}`
                : dateRangeFilter.start === dateRangeFilter.end
                  ? dateRangeFilter.start
                  : `${dateRangeFilter.start} \u2014 ${dateRangeFilter.end}`}
            </span>
            <button
              onClick={clearDateRangeFilter}
              className="text-xs text-primary hover:underline"
            >
              {t("gallery.calendar.clearFilter")}
            </button>
          </div>
        )}
      </div>

      {/* Main content area with images and timeline */}
      <div className="flex gap-4" style={{ position: "relative" }}>
        {/* Images area */}
        <div ref={mainContentRef} className="flex-1 min-w-0">
          {error && (
            <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-4 text-sm text-destructive">
              {error}
            </div>
          )}

          {!initialized && isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-40 w-full rounded-lg" />
              ))}
            </div>
          ) : groups.length === 0 && !isLoading ? (
            <div className="rounded-lg border border-dashed p-12 text-center">
              <CalendarIcon className="mx-auto h-10 w-10 text-muted-foreground/50" />
              <p className="mt-2 text-sm font-medium text-muted-foreground">
                {dateRangeFilter.start ? t("gallery.calendar.noImagesForDate") : t("gallery.calendar.noDateInfo")}
              </p>
              <p className="text-xs text-muted-foreground/70">
                {dateRangeFilter.start ? t("gallery.calendar.clearFilterHint") : t("gallery.calendar.noDateInfoHint")}
              </p>
            </div>
          ) : (
            <>
              {groups.map((group) => (
                <div key={group.date} id={`date-group-${group.date}`} className="mb-6">
                  <div className="flex items-center gap-2 mb-2 px-0.5">
                    <CalendarIcon className="h-4 w-4 text-muted-foreground shrink-0" />
                    <span className="text-sm font-medium">{group.label}</span>
                    <span className="text-xs text-muted-foreground shrink-0">
                      ({group.imageCount})
                    </span>
                  </div>
                  <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-7 xl:grid-cols-8 gap-1.5">
                    {group.images.map((image) => (
                      <button
                        key={image.id}
                        className="group flex flex-col cursor-pointer"
                        onClick={() => onImageClick(image)}
                      >
                        <div className="relative aspect-square overflow-hidden rounded-lg border bg-muted hover:ring-2 hover:ring-ring transition-all">
                          {image.thumbnail ? (
                            <img
                              src={image.thumbnail}
                              alt={image.fileName}
                              className="h-full w-full object-cover"
                              loading="lazy"
                            />
                          ) : (
                            <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
                              {t("gallery.noPreview")}
                            </div>
                          )}
                        </div>
                        <p className="text-[11px] text-muted-foreground truncate mt-1 px-0.5 w-full text-center" title={image.fileName}>
                          {image.fileName}
                        </p>
                      </button>
                    ))}
                  </div>
                </div>
              ))}

              <div ref={sentinelRef} className="h-4" />

              {isLoading && (
                <div className="flex justify-center py-4">
                  <div className="text-sm text-muted-foreground">{t("gallery.loadingMore")}</div>
                </div>
              )}

              {!hasMore && groups.length > 0 && (
                <div className="text-center text-xs text-muted-foreground py-4">
                  {t("gallery.allLoaded", { count: totalImages.toLocaleString() })}
                </div>
              )}
            </>
          )}
        </div>

        {/* Timeline sidebar — fixed to right edge, offset from header */}
        {dateRange.minDate && dateRange.maxDate && groups.length > 0 && (
          <div
            className="fixed right-0 w-16 z-10 hidden lg:flex flex-col justify-center"
            style={{ 
              pointerEvents: "none",
              top: `${HEADER_HEIGHT}px`,
              bottom: 0
            }}
          >
            <div
              className="rounded-l-lg border-r border-y border-l-0 bg-card p-2 mx-0"
              style={{ 
                pointerEvents: "auto", 
                height: "calc(100vh - 2rem)", 
                maxHeight: "calc(100vh - 2rem)" 
              }}
            >
              <div 
                className="relative flex-1" 
                style={{ height: "calc(100% - 2rem)" }}
                onClick={(e) => {
                  // Navigate to date when clicking on timeline
                  const rect = e.currentTarget.getBoundingClientRect()
                  const clickY = e.clientY - rect.top
                  const clickPercent = clickY / rect.height
                  
                  const totalDays = daysBetween(dateRange.minDate, dateRange.maxDate)
                  const targetOffset = Math.floor(clickPercent * totalDays)
                  const targetDate = new Date(dateRange.minDate + "T00:00:00")
                  targetDate.setDate(targetDate.getDate() + targetOffset)
                  
                  const dateStr = `${targetDate.getFullYear()}-${String(targetDate.getMonth() + 1).padStart(2, "0")}-${String(targetDate.getDate()).padStart(2, "0")}`
                  
                  // Scroll to the date group
                  const element = document.getElementById(`date-group-${dateStr}`)
                  if (element) {
                    element.scrollIntoView({ behavior: "smooth", block: "start" })
                  }
                }}
              >
                {/* Timeline track */}
                <div className="absolute left-1/2 -translate-x-1/2 w-0.5 h-full bg-muted" />

                {/* Year/Month scale markers */}
                {(() => {
                  const markers = []
                  const start = new Date(dateRange.minDate + "T00:00:00")
                  const end = new Date(dateRange.maxDate + "T00:00:00")
                  const totalDays = daysBetween(dateRange.minDate, dateRange.maxDate)
                  
                  // Generate year markers
                  let current = new Date(start.getFullYear(), 0, 1)
                  if (current < start) current = new Date(start.getFullYear() + 1, 0, 1)
                  
                  while (current <= end) {
                    const offset = daysBetween(dateRange.minDate, current.toISOString().split("T")[0])
                    const topPercent = totalDays > 0 ? (offset / totalDays) * 100 : 0
                    
                    markers.push(
                      <div
                        key={`year-${current.getFullYear()}`}
                        className="absolute left-0 right-0 border-t border-primary/50"
                        style={{ top: `${topPercent}%` }}
                      >
                        <span className="absolute left-0 text-[9px] font-semibold text-foreground whitespace-nowrap -translate-x-1/2" style={{ left: "50%" }}>
                          {current.getFullYear()}
                        </span>
                      </div>
                    )
                    
                    current.setFullYear(current.getFullYear() + 1)
                    current = new Date(current.getFullYear(), 0, 1)
                  }
                  
                  return markers
                })()}

                {/* Visible range indicator */}
                {visibleDateRange.start && visibleDateRange.end && (
                  (() => {
                    const totalDays = daysBetween(dateRange.minDate, dateRange.maxDate)
                    const startOffset = daysBetween(dateRange.minDate, visibleDateRange.start)
                    const endOffset = daysBetween(dateRange.minDate, visibleDateRange.end)
                    const topPercent = totalDays > 0 ? (startOffset / totalDays) * 100 : 0
                    const heightPercent = totalDays > 0 ? ((endOffset - startOffset) / totalDays) * 100 : 0

                    return (
                      <div
                        className="absolute left-1/2 -translate-x-1/2 w-3 bg-primary/20 rounded-sm"
                        style={{
                          top: `${topPercent}%`,
                          height: `${Math.max(heightPercent, 5)}%`,
                        }}
                      />
                    )
                  })()
                )}

                {/* Date markers for visible groups */}
                {groups.map((group) => {
                  const offset = daysBetween(dateRange.minDate, group.date)
                  const totalDays = daysBetween(dateRange.minDate, dateRange.maxDate)
                  const topPercent = totalDays > 0 ? (offset / totalDays) * 100 : 0

                  return (
                    <div
                      key={group.date}
                      className="absolute left-1/2 -translate-x-1/2 w-2 h-2 rounded-full bg-primary/40 cursor-pointer hover:scale-150 transition-transform"
                      style={{ top: `${topPercent}%` }}
                      title={group.date}
                      onClick={(e) => {
                        e.stopPropagation()
                        const element = document.getElementById(`date-group-${group.date}`)
                        if (element) {
                          element.scrollIntoView({ behavior: "smooth", block: "start" })
                        }
                      }}
                    />
                  )
                })}

              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function daysBetween(date1: string, date2: string): number {
  const d1 = new Date(date1 + "T00:00:00")
  const d2 = new Date(date2 + "T00:00:00")
  return Math.floor((d2.getTime() - d1.getTime()) / (1000 * 60 * 60 * 24))
}
