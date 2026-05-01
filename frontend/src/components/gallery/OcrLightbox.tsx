import { useState, useEffect, useRef, useCallback } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { X, Loader2, Wand2 } from "lucide-react"
import { useTranslation } from "@/i18n"
import { fetchOcrData, fetchLlmRecognition, recognizeWithLlm, fetchLlmRecognizeStatus } from "@/api/endpoints"
import type { OcrDataResponse, LlmOcrDataResponse } from "@/types"
import ReactMarkdown, { type Components } from "react-markdown"

const API_BASE_URL = import.meta.env.VITE_API_URL || ""

interface OcrLightboxProps {
  imagePath: string | null
  onClose: () => void
}

export function OcrLightbox({ imagePath, onClose }: OcrLightboxProps) {
  const { t } = useTranslation()
  const [ocrData, setOcrData] = useState<OcrDataResponse | null>(null)
  const [llmData, setLlmData] = useState<LlmOcrDataResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [recognizing, setRecognizing] = useState(false)
  const [imageLoaded, setImageLoaded] = useState(false)
  const [imageDimensions, setImageDimensions] = useState<{ width: number; height: number } | null>(null)
  const [displayDimensions, setDisplayDimensions] = useState<{ width: number; height: number } | null>(null)
  const imageRef = useRef<HTMLImageElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const prevImagePath = useRef<string | null>(null)

  const resetState = useCallback(() => {
    setOcrData(null)
    setLlmData(null)
    setImageLoaded(false)
    setImageDimensions(null)
    setDisplayDimensions(null)
    setLoading(false)
    prevImagePath.current = null
  }, [])

  const handleClose = useCallback(() => {
    resetState()
    onClose()
  }, [resetState, onClose])

  // Load OCR data when lightbox opens
  useEffect(() => {
    if (!imagePath) {
      return
    }

    // Only fetch if imagePath changed
    if (prevImagePath.current === imagePath) {
      return
    }
    prevImagePath.current = imagePath

    let isMounted = true
    setLoading(true)
    
    Promise.all([
      fetchOcrData(imagePath).catch(() => null),
      fetchLlmRecognition(imagePath).catch(() => null),
    ]).then(([ocr, llm]) => {
      if (isMounted) {
        setOcrData(ocr)
        setLlmData(llm)
        setLoading(false)
      }
    }).catch(() => {
      if (isMounted) {
        setLoading(false)
      }
    })

    return () => {
      isMounted = false
    }
  }, [imagePath])

  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Stop polling on unmount or image change
  useEffect(() => {
    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current)
        pollingRef.current = null
      }
    }
  }, [imagePath])

  // Handle recognize button click - starts async recognition with polling
  const handleRecognize = useCallback(() => {
    if (!imagePath || recognizing) return

    const hasExistingResult = llmData?.found && llmData.success
    setRecognizing(true)

    // Stop any existing polling
    if (pollingRef.current) {
      clearInterval(pollingRef.current)
      pollingRef.current = null
    }

    recognizeWithLlm({ imagePath, force: hasExistingResult || undefined })
      .then((result) => {
        // If the backend returned a cached result (status 200 with success)
        if (result.status === "completed" && result.markdownContent) {
          setLlmData({
            found: true,
            markdownContent: result.markdownContent,
            language: result.language,
            provider: result.provider,
            model: result.model,
            processingTimeMs: result.processingTimeMs,
            success: true,
          })
          setRecognizing(false)
          return
        }

        // Start polling for async task status
        const currentPath = imagePath
        pollingRef.current = setInterval(() => {
          fetchLlmRecognizeStatus(currentPath)
            .then((status) => {
              if (status.status === "completed") {
                if (pollingRef.current) {
                  clearInterval(pollingRef.current)
                  pollingRef.current = null
                }
                setLlmData({
                  found: true,
                  markdownContent: status.markdownContent,
                  language: status.language,
                  provider: status.provider,
                  model: status.model,
                  processingTimeMs: status.processingTimeMs,
                  success: true,
                })
                setRecognizing(false)
              } else if (status.status === "failed") {
                if (pollingRef.current) {
                  clearInterval(pollingRef.current)
                  pollingRef.current = null
                }
                setLlmData({
                  found: false,
                  error: status.error,
                  success: false,
                })
                setRecognizing(false)
              }
              // status === "processing" — keep polling
            })
            .catch(() => {
              if (pollingRef.current) {
                clearInterval(pollingRef.current)
                pollingRef.current = null
              }
              setRecognizing(false)
            })
        }, 2000)
      })
      .catch((err) => {
        console.error("LLM recognition failed:", err)
        setLlmData({
          found: false,
          error: err.message,
          success: false,
        })
        setRecognizing(false)
      })
  }, [imagePath, recognizing, llmData])

  // Calculate display dimensions when image loads
  const handleImageLoad = useCallback(() => {
    if (imageRef.current) {
      const { naturalWidth, naturalHeight, clientWidth, clientHeight } = imageRef.current
      setImageDimensions({ width: naturalWidth, height: naturalHeight })
      setDisplayDimensions({ width: clientWidth, height: clientHeight })
      setImageLoaded(true)
    }
  }, [])

  // Update display dimensions on resize
  useEffect(() => {
    if (!imageLoaded || !imageRef.current) return

    const updateDimensions = () => {
      if (imageRef.current) {
        setDisplayDimensions({
          width: imageRef.current.clientWidth,
          height: imageRef.current.clientHeight,
        })
      }
    }

    updateDimensions()
    window.addEventListener("resize", updateDimensions)
    return () => window.removeEventListener("resize", updateDimensions)
  }, [imageLoaded])

  // Calculate scale factor for bounding boxes
  const baseScaleX = imageDimensions && displayDimensions ? displayDimensions.width / imageDimensions.width : 1
  const baseScaleY = imageDimensions && displayDimensions ? displayDimensions.height / imageDimensions.height : 1
  const scaleX = baseScaleX
  const scaleY = baseScaleY

  const angle = ocrData?.angle || 0
  const ocrScaleFactor = ocrData?.scaleFactor || 1
  const imageUrl = imagePath
    ? `${API_BASE_URL}/api/ocr-image?path=${encodeURIComponent(imagePath)}&angle=${angle}&scaleFactor=${ocrScaleFactor}`
    : ""

  // Format processing time
  const formatProcessingTime = (ms?: number) => {
    if (!ms) return ""
    if (ms < 1000) return t("llm_ocr.milliseconds", { ms })
    return t("llm_ocr.seconds", { seconds: (ms / 1000).toFixed(1) })
  }

  return (
    <Dialog open={imagePath !== null} onOpenChange={() => handleClose()}>
      <DialogContent className="max-w-[95vw] w-[95vw] h-[90vh] p-0 bg-black/95 border-0 overflow-hidden">
        <DialogHeader className="absolute top-4 left-4 right-16 z-50">
          <DialogTitle className="text-white text-lg">
            {t("lightbox.ocrTitle")}
          </DialogTitle>
          <DialogDescription className="sr-only">
            {t("lightbox.ocrDescription")}
          </DialogDescription>
        </DialogHeader>

        <button
          onClick={handleClose}
          className="absolute top-4 right-4 z-50 p-2 rounded-full bg-black/50 hover:bg-black/70 text-white transition-colors"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex h-full">
          {/* Image with bounding boxes - 50% width */}
          <div className="w-[50%] flex items-center justify-center p-8 relative" ref={containerRef}>
            {loading && (
              <div className="absolute inset-0 flex items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-white" />
              </div>
            )}

            <div className="relative inline-block">
              {/* Image */}
              {imagePath && (
                <img
                  ref={imageRef}
                  src={imageUrl}
                  alt={t("lightbox.alt")}
                  className="max-w-full max-h-[75vh] object-contain"
                  onLoad={handleImageLoad}
                />
              )}

              {/* Bounding boxes overlay */}
              {ocrData && ocrData.boxes.length > 0 && imageLoaded && displayDimensions && (
                <div
                  className="absolute inset-0 pointer-events-none"
                  style={{
                    width: displayDimensions.width,
                    height: displayDimensions.height,
                  }}
                >
                  {ocrData.boxes.map((box, index) => (
                    <div
                      key={index}
                      className="absolute border-2 border-yellow-400/70 bg-yellow-400/10"
                      style={{
                        left: box.x * scaleX,
                        top: box.y * scaleY,
                        width: box.width * scaleX,
                        height: box.height * scaleY,
                      }}
                      title={`${box.word} (${(box.confidence * 100).toFixed(0)}%)`}
                    ></div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Right panel - 50% width */}
          <div className="w-[50%] bg-card border-l p-4 h-full overflow-y-auto">
            {recognizing ? (
              <div className="flex flex-col items-center justify-center h-full">
                <Loader2 className="h-12 w-12 animate-spin text-primary mb-4" />
                <p className="text-lg font-medium">{t("llm_ocr.recognizing")}</p>
              </div>
            ) : llmData?.found && llmData.success && llmData.markdownContent ? (
              /* LLM recognition result */
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold">{t("llm_ocr.title")}</h3>
                  <button
                    onClick={handleRecognize}
                    className="flex items-center gap-2 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90"
                  >
                    <Wand2 className="h-4 w-4" />
                    {t("llm_ocr.recognizeButton")}
                  </button>
                </div>

                {/* Metadata */}
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t("llm_ocr.language")}:</span>
                    <span className="font-medium">{llmData.language === "ru" ? "Русский" : "English"}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t("llm_ocr.provider")}:</span>
                    <span className="font-medium">{llmData.provider}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t("llm_ocr.model")}:</span>
                    <span className="font-medium">{llmData.model}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{t("llm_ocr.processingTime")}:</span>
                    <span className="font-medium">{formatProcessingTime(llmData.processingTimeMs)}</span>
                  </div>
                  {imagePath && (
                    <div>
                      <span className="text-muted-foreground">{t("llm_ocr.filePath")}:</span>
                      <p className="text-xs break-all mt-1">{imagePath}</p>
                    </div>
                  )}
                </div>

                {/* Markdown content */}
                <div className="mt-4 p-4 bg-muted rounded-lg markdown-body">
                  <ReactMarkdown
                    components={
                      {
                        h1: (props) => <h1 className="text-xl font-bold mt-4 mb-2" {...props} />,
                        h2: (props) => <h2 className="text-lg font-bold mt-3 mb-2" {...props} />,
                        h3: (props) => <h3 className="text-base font-bold mt-2 mb-1" {...props} />,
                        p: (props) => <p className="mb-2 leading-relaxed" {...props} />,
                        ul: (props) => <ul className="list-disc list-inside mb-2 space-y-1" {...props} />,
                        ol: (props) => <ol className="list-decimal list-inside mb-2 space-y-1" {...props} />,
                        li: (props) => <li className="text-sm" {...props} />,
                        code: ({ className, ...props }) => {
                          const isInline = !className
                          return isInline
                            ? <code className="bg-muted-foreground/20 px-1.5 py-0.5 rounded text-sm font-mono" {...props} />
                            : <code className="block bg-muted-foreground/20 p-2 rounded text-sm font-mono overflow-x-auto" {...props} />
                        },
                        blockquote: (props) => <blockquote className="border-l-4 border-primary pl-3 italic my-2" {...props} />,
                        table: (props) => <table className="min-w-full border-collapse border border-border my-2" {...props} />,
                        th: (props) => <th className="border border-border px-3 py-1.5 font-semibold text-left" {...props} />,
                        td: (props) => <td className="border border-border px-3 py-1.5" {...props} />,
                        a: (props) => <a className="text-primary underline hover:opacity-80" {...props} />,
                        strong: (props) => <strong className="font-semibold" {...props} />,
                        em: (props) => <em className="italic" {...props} />,
                        hr: (props) => <hr className="border-border my-3" {...props} />,
                        img: (props) => <img className="max-w-full h-auto rounded my-2" {...props} />,
                      } as Components
                    }
                  >
                    {llmData.markdownContent}
                  </ReactMarkdown>
                </div>
              </div>
            ) : llmData?.error ? (
              /* Error state */
              <div className="space-y-4">
                <h3 className="text-lg font-semibold text-destructive">{t("llm_ocr.failed")}</h3>
                <p className="text-sm text-muted-foreground">{llmData.error}</p>
                <button
                  onClick={handleRecognize}
                  className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90"
                >
                  <Wand2 className="h-4 w-4" />
                  {t("llm_ocr.recognizeButton")}
                </button>
              </div>
            ) : (
              /* No LLM recognition yet */
              <div className="space-y-4">
                <h3 className="text-lg font-semibold">{t("llm_ocr.title")}</h3>

                {/* Basic info */}
                <div className="space-y-2 text-sm">
                  {imagePath && (
                    <div>
                      <span className="text-muted-foreground">{t("llm_ocr.filePath")}:</span>
                      <p className="text-xs break-all mt-1">{imagePath}</p>
                    </div>
                  )}
                  {ocrData && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">{t("llm_ocr.language")}:</span>
                      <span className="font-medium">
                        {(() => {
                          // Detect language from OCR data
                          let ruCount = 0
                          let enCount = 0
                          for (const box of ocrData.boxes) {
                            for (const ch of box.word.toLowerCase()) {
                              if (ch.charCodeAt(0) >= 0x0400 && ch.charCodeAt(0) <= 0x04FF) ruCount++
                              if (ch.charCodeAt(0) >= 0x0061 && ch.charCodeAt(0) <= 0x007A) enCount++
                            }
                          }
                          return ruCount > enCount ? "Русский" : "English"
                        })()}
                      </span>
                    </div>
                  )}
                </div>

                {/* Recognize button */}
                <button
                  onClick={handleRecognize}
                  className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
                >
                  <Wand2 className="h-5 w-5" />
                  {t("llm_ocr.recognizeButton")}
                </button>

                <p className="text-xs text-muted-foreground text-center">
                  {t("llm_ocr.description")}
                </p>
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
