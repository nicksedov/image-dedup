import { useState, useEffect, useRef, useCallback } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { X, Loader2, FileText } from "lucide-react"
import { useTranslation } from "@/i18n"
import { fetchOcrData } from "@/api/endpoints"
import type { OcrDataResponse } from "@/types"

const API_BASE_URL = import.meta.env.VITE_API_URL || ""

interface OcrLightboxProps {
  imagePath: string | null
  onClose: () => void
}

export function OcrLightbox({ imagePath, onClose }: OcrLightboxProps) {
  const { t } = useTranslation()
  const [ocrData, setOcrData] = useState<OcrDataResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [imageLoaded, setImageLoaded] = useState(false)
  const [imageDimensions, setImageDimensions] = useState<{ width: number; height: number } | null>(null)
  const [displayDimensions, setDisplayDimensions] = useState<{ width: number; height: number } | null>(null)
  const imageRef = useRef<HTMLImageElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // Load OCR data when lightbox opens
  useEffect(() => {
    if (!imagePath) {
      setOcrData(null)
      setImageLoaded(false)
      setImageDimensions(null)
      setDisplayDimensions(null)
      return
    }

    setLoading(true)
    fetchOcrData(imagePath)
      .then((data) => {
        setOcrData(data)
      })
      .catch((err) => {
        console.error("Failed to load OCR data:", err)
        setOcrData(null)
      })
      .finally(() => {
        setLoading(false)
      })
  }, [imagePath])

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
  // scaleFactor converts from OCR-processed image coords to original image coords
  // scaleX/scaleY convert from original image coords to display coords
  const baseScaleX = imageDimensions && displayDimensions ? displayDimensions.width / imageDimensions.width : 1
  const baseScaleY = imageDimensions && displayDimensions ? displayDimensions.height / imageDimensions.height : 1
  const scaleX = baseScaleX
  const scaleY = baseScaleY

  const angle = ocrData?.angle || 0
  const ocrScaleFactor = ocrData?.scaleFactor || 1
  const imageUrl = imagePath
    ? `${API_BASE_URL}/api/ocr-image?path=${encodeURIComponent(imagePath)}&angle=${angle}&scaleFactor=${ocrScaleFactor}`
    : ""

  return (
    <Dialog open={imagePath !== null} onOpenChange={() => onClose()}>
      <DialogContent className="max-w-[95vw] w-[95vw] h-[90vh] p-0 bg-black/95 border-0">
        <DialogHeader className="absolute top-4 left-4 right-16 z-50">
          <DialogTitle className="text-white text-lg">
            {t("lightbox.ocrTitle")}
          </DialogTitle>
        </DialogHeader>

        <button
          onClick={onClose}
          className="absolute top-4 right-4 z-50 p-2 rounded-full bg-black/50 hover:bg-black/70 text-white transition-colors"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex h-full">
          {/* Image with bounding boxes */}
          <div className="flex-1 flex items-center justify-center p-8 relative" ref={containerRef}>
            {loading && (
              <div className="absolute inset-0 flex items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-white" />
              </div>
            )}

            <div
              className="relative inline-block"
            >
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
                    >
                      {/* Word label */}
                      {box.width * scaleX > 30 && box.height * scaleY > 15 && (
                        <span className="absolute -top-5 left-0 text-[10px] text-yellow-400 whitespace-nowrap bg-black/70 px-1 rounded">
                          {box.word}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Metadata panel */}
          {ocrData && (
            <div className="w-80 bg-card border-l p-4 overflow-y-auto hidden lg:block">
              <h3 className="text-lg font-semibold mb-4">{t("metadata.title")}</h3>

              {/* OCR metadata */}
              <div className="space-y-3">
                <div className="p-3 bg-muted rounded-lg">
                  <h4 className="text-sm font-medium text-muted-foreground mb-2">OCR</h4>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-sm">{t("ocr.angle")}</span>
                      <span className="text-sm font-medium">{ocrData.angle}°</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm">{t("lightbox.ocrTokens", { count: 0 }).split(":")[0]}</span>
                      <span className="text-sm font-medium">{ocrData.boxes.length}</span>
                    </div>
                  </div>
                </div>

                {/* File info */}
                <div className="p-3 bg-muted rounded-lg">
                  <h4 className="text-sm font-medium text-muted-foreground mb-2">File</h4>
                  <p className="text-sm break-all">{imagePath}</p>
                </div>

                {/* Bounding boxes list */}
                {ocrData.boxes.length > 0 && (
                  <div className="p-3 bg-muted rounded-lg">
                    <h4 className="text-sm font-medium text-muted-foreground mb-2">
                      Text Regions ({ocrData.boxes.length})
                    </h4>
                    <div className="space-y-1 max-h-64 overflow-y-auto">
                      {ocrData.boxes.slice(0, 50).map((box, index) => (
                        <div key={index} className="text-xs flex justify-between">
                          <span className="truncate flex-1" title={box.word}>
                            {box.word || "(empty)"}
                          </span>
                          <span className="text-muted-foreground ml-2">
                            {(box.confidence * 100).toFixed(0)}%
                          </span>
                        </div>
                      ))}
                      {ocrData.boxes.length > 50 && (
                        <p className="text-xs text-muted-foreground">
                          ... and {ocrData.boxes.length - 50} more
                        </p>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* No OCR data */}
          {!ocrData && !loading && (
            <div className="w-80 bg-card border-l p-4 flex items-center justify-center hidden lg:flex">
              <div className="text-center">
                <FileText className="h-12 w-12 text-muted-foreground mx-auto mb-2" />
                <p className="text-sm text-muted-foreground">{t("metadata.noData")}</p>
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
