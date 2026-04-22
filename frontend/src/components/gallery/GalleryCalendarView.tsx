import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { fetchGalleryCalendar } from "@/api/endpoints"
import { Skeleton } from "@/components/ui/skeleton"
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from "lucide-react"
import { useTranslation } from "@/i18n"
import type { GalleryImageDTO, CalendarDateGroup, CalendarDateRange, CalendarMonthInfo } from "@/types"

interface GalleryCalendarViewProps {
  onImageClick: (image: GalleryImageDTO) => void
}

const PAGE_SIZE = 50

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

  // Date filter state
  const [dateFilter, setDateFilter] = useState<{ start: string | null; end: string | null }>({
    start: null,
    end: null,
  })

  // Calendar widget state
  const [calendarViewDate, setCalendarViewDate] = useState(() => {
    return new Date()
  })

  const pageRef = useRef(1)
  const sentinelRef = useRef<HTMLDivElement>(null)
  const mainContentRef = useRef<HTMLDivElement>(null)

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
        dateFilter.start ?? undefined,
        dateFilter.end ?? undefined,
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
        if (!dateFilter.start && result.dateRange.minDate) {
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
  }, [isLoading, dateFilter.start, dateFilter.end, calendarMonthKey])

  // Initial load or reset when filter changes
  useEffect(() => {
    pageRef.current = 1
    setGroups([])
    setInitialized(false)
    loadPage(1, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dateFilter.start, dateFilter.end])

  // Load month info when calendar month changes
  useEffect(() => {
    fetchGalleryCalendar(1, 1, dateFilter.start ?? undefined, dateFilter.end ?? undefined, calendarMonthKey)
      .then((result) => {
        if (result.months.length > 0) {
          setMonthInfo(result.months[0])
        }
      })
      .catch(() => {})
  }, [calendarMonthKey, dateFilter.start, dateFilter.end])

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

    const days: { date: string; day: number; hasImages: boolean }[] = []
    for (let d = 1; d <= daysInMonth; d++) {
      const dateStr = `${year}-${String(month + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`
      days.push({
        date: dateStr,
        day: d,
        hasImages: daysWithImages.has(d),
      })
    }

    return days
  }, [calendarViewDate, monthInfo])

  const prevMonth = () => {
    setCalendarViewDate(new Date(calendarViewDate.getFullYear(), calendarViewDate.getMonth() - 1, 1))
  }

  const nextMonth = () => {
    setCalendarViewDate(new Date(calendarViewDate.getFullYear(), calendarViewDate.getMonth() + 1, 1))
  }

  const selectDate = (date: string) => {
    if (dateFilter.start === date && dateFilter.end === date) {
      setDateFilter({ start: null, end: null })
    } else {
      setDateFilter({ start: date, end: date })
    }
  }

  const clearDateFilter = () => {
    setDateFilter({ start: null, end: null })
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
        <div className="flex items-center justify-between mb-2">
          <button onClick={prevMonth} className="p-1 hover:bg-muted rounded">
            <ChevronLeft className="h-4 w-4" />
          </button>
          <span className="font-medium text-sm">
            {calendarViewDate.toLocaleDateString(undefined, { month: "long", year: "numeric" })}
          </span>
          <button onClick={nextMonth} className="p-1 hover:bg-muted rounded">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>

        {/* Horizontal scrollable day strip */}
        <div className="flex gap-1 overflow-x-auto pb-1 scrollbar-thin" style={{ scrollbarWidth: "thin" }}>
          {calendarDays.map((day) => (
            <button
              key={day.date}
              disabled={!day.date}
              className={`
                flex-shrink-0 w-9 h-9 flex flex-col items-center justify-center text-xs rounded-md
                transition-all
                ${day.hasImages ? "bg-primary/10 hover:bg-primary/20 font-medium text-primary cursor-pointer" : "text-muted-foreground/40"}
                ${dateFilter.start === day.date && dateFilter.end === day.date ? "bg-primary text-primary-foreground hover:bg-primary/90" : ""}
              `}
              onClick={() => selectDate(day.date)}
            >
              <span className="text-[11px] leading-none">{day.day}</span>
            </button>
          ))}
        </div>

        {/* Date filter controls */}
        {(dateFilter.start || dateFilter.end) && (
          <div className="mt-2 pt-2 border-t flex items-center justify-between">
            <span className="text-xs text-muted-foreground">
              {dateFilter.start === dateFilter.end
                ? dateFilter.start
                : `${dateFilter.start} \u2014 ${dateFilter.end}`}
            </span>
            <button
              onClick={clearDateFilter}
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
                {dateFilter.start ? t("gallery.calendar.noImagesForDate") : t("gallery.calendar.noDateInfo")}
              </p>
              <p className="text-xs text-muted-foreground/70">
                {dateFilter.start ? t("gallery.calendar.clearFilterHint") : t("gallery.calendar.noDateInfoHint")}
              </p>
            </div>
          ) : (
            <>
              {groups.map((group) => (
                <div key={group.date} className="mb-6">
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

        {/* Timeline sidebar — fixed to right edge, full viewport height */}
        {dateRange.minDate && dateRange.maxDate && groups.length > 0 && (
          <div
            className="fixed right-0 top-0 bottom-0 w-16 z-10 hidden lg:flex flex-col justify-center"
            style={{ pointerEvents: "none" }}
          >
            <div
              className="rounded-l-lg border-r border-y bg-card p-2 mx-0"
              style={{ pointerEvents: "auto", height: "calc(100vh - 2rem)", maxHeight: "calc(100vh - 2rem)" }}
            >
              <div className="text-xs font-medium mb-2 text-center">{t("gallery.calendar.timeline")}</div>
              <div className="relative flex-1" style={{ height: "calc(100% - 2rem)" }}>
                {/* Timeline track */}
                <div className="absolute left-1/2 -translate-x-1/2 w-0.5 h-full bg-muted" />

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
                      className="absolute left-1/2 -translate-x-1/2 w-2 h-2 rounded-full bg-primary"
                      style={{ top: `${topPercent}%` }}
                      title={group.date}
                    />
                  )
                })}

                {/* Oldest date label (top since ASC order) */}
                <div className="absolute -top-5 left-1/2 -translate-x-1/2 text-[10px] text-muted-foreground whitespace-nowrap">
                  {formatShortDate(dateRange.minDate)}
                </div>

                {/* Newest date label (bottom) */}
                <div className="absolute -bottom-5 left-1/2 -translate-x-1/2 text-[10px] text-muted-foreground whitespace-nowrap">
                  {formatShortDate(dateRange.maxDate)}
                </div>
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

function formatShortDate(date: string): string {
  const d = new Date(date + "T00:00:00")
  return d.toLocaleDateString(undefined, { month: "short", year: "2-digit" })
}
