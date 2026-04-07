import { useMemo } from "react"
import { useTranslation } from "@/i18n"
import { Folder } from "lucide-react"
import type { GalleryImageDTO } from "@/types"

interface GalleryImageGridProps {
  images: GalleryImageDTO[]
  onImageClick: (image: GalleryImageDTO) => void
}

function getFolderName(dirPath: string): string {
  const normalized = dirPath.replace(/\\/g, "/").replace(/\/+$/, "")
  const lastSlash = normalized.lastIndexOf("/")
  return lastSlash >= 0 ? normalized.substring(lastSlash + 1) : normalized
}

export function GalleryImageGrid({ images, onImageClick }: GalleryImageGridProps) {
  const { t } = useTranslation()

  const groupedByFolder = useMemo(() => {
    const groups: { dirPath: string; folderName: string; images: GalleryImageDTO[] }[] = []
    const map = new Map<string, GalleryImageDTO[]>()
    const order: string[] = []

    for (const image of images) {
      const dir = image.dirPath
      if (!map.has(dir)) {
        map.set(dir, [])
        order.push(dir)
      }
      map.get(dir)!.push(image)
    }

    for (const dir of order) {
      groups.push({
        dirPath: dir,
        folderName: getFolderName(dir),
        images: map.get(dir)!,
      })
    }

    return groups
  }, [images])

  return (
    <div className="space-y-5">
      {groupedByFolder.map((group) => (
        <div key={group.dirPath}>
          <div className="flex items-center gap-2 mb-2 px-0.5">
            <Folder className="h-4 w-4 text-muted-foreground shrink-0" />
            <span className="text-sm font-medium truncate" title={group.dirPath}>
              {group.folderName}
            </span>
            <span className="text-xs text-muted-foreground shrink-0">
              {t("gallery.folderImageCount", { count: group.images.length.toString() })}
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
                  <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent p-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <div className="text-xs text-white/70">{image.sizeHuman}</div>
                  </div>
                </div>
                <p className="text-[11px] text-muted-foreground truncate mt-1 px-0.5 w-full text-center" title={image.fileName}>
                  {image.fileName}
                </p>
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
