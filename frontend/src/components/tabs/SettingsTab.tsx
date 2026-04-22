import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { AddFolderForm } from "@/components/settings/AddFolderForm"
import { FolderList } from "@/components/settings/FolderList"
import { ScanProgressBanner } from "@/components/ScanProgressBanner"
import { useGalleryFolders } from "@/hooks/useGalleryFolders"
import { useScanStatus } from "@/hooks/useScanStatus"
import { triggerScan, fetchTrashInfo, cleanTrash, updateSettings } from "@/api/endpoints"
import { useSettings } from "@/providers/useSettings"
import { RefreshCw, Trash2, Sun, Moon } from "lucide-react"
import { useTranslation } from "@/i18n"
import { useTheme } from "@/theme"

interface SettingsTabProps {
  onFolderAdded: () => void
}

export function SettingsTab({ onFolderAdded }: SettingsTabProps) {
  const { folders, isLoading, add, remove, refetch } = useGalleryFolders()
  const { status, startPolling, setOnScanComplete } = useScanStatus()
  const { trashDir, setTrashDir, theme, setTheme, language, setLanguage } = useSettings()
  const { t } = useTranslation()
  const { getThemeName } = useTheme()

  const [trashInput, setTrashInput] = useState(trashDir)
  const [trashFileCount, setTrashFileCount] = useState(0)
  const [trashTotalSize, setTrashTotalSize] = useState("")
  const [isCleaning, setIsCleaning] = useState(false)
  const [isSavingTrash, setIsSavingTrash] = useState(false)

  useEffect(() => {
    setTrashInput(trashDir)
  }, [trashDir])

  const loadTrashInfo = useCallback(() => {
    fetchTrashInfo()
      .then((info) => {
        setTrashFileCount(info.fileCount)
        setTrashTotalSize(info.totalSizeHuman)
      })
      .catch(() => {
        setTrashFileCount(0)
        setTrashTotalSize("")
      })
  }, [])

  useEffect(() => {
    loadTrashInfo()
  }, [loadTrashInfo])

  const handleSaveTrashDir = useCallback(async () => {
    setIsSavingTrash(true)
    try {
      const result = await updateSettings({ trashDir: trashInput.trim() })
      setTrashDir(result.trashDir)
      setTrashInput(result.trashDir)
      toast.success(t("trash.saved"))
      loadTrashInfo()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("trash.saveFailed"))
    } finally {
      setIsSavingTrash(false)
    }
  }, [trashInput, setTrashDir, loadTrashInfo, t])

  const handleSaveTheme = useCallback((value: string) => {
    setTheme(value as "light" | "dark")
  }, [setTheme])

  const handleSaveLanguage = useCallback((value: string) => {
    setLanguage(value as "en" | "ru")
  }, [setLanguage])

  const handleCleanTrash = useCallback(async () => {
    if (trashFileCount === 0) return
    if (!window.confirm(t("trash.cleanConfirm", { count: trashFileCount }))) return

    setIsCleaning(true)
    try {
      const result = await cleanTrash()
      toast.success(t("trash.cleanSuccess", { deleted: result.deleted }))
      loadTrashInfo()
    } catch {
      toast.error(t("trash.cleanFailed"))
    } finally {
      setIsCleaning(false)
    }
  }, [trashFileCount, loadTrashInfo, t])

  const handleAdd = useCallback(
    async (path: string) => {
      try {
        const result = await add(path)
        const message = result.message as string
        toast.success(message.includes(".") ? t(message as any) : message)
        if (result.scanStarted) {
          setOnScanComplete(() => {
            refetch()
            toast.success(t("settings.toastScanComplete"))
          })
          startPolling()
        }
        onFolderAdded()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : t("settings.toastAddFailed"))
      }
    },
    [add, startPolling, setOnScanComplete, refetch, onFolderAdded, t]
  )

  const handleRemove = useCallback(
    async (id: number) => {
      try {
        const result = await remove(id)
        const message = result.message as string
        toast.success(t("settings.toastFilesRemoved", { message: message.includes(".") ? t(message as any) : message, count: result.filesRemoved }))
      } catch (err) {
        toast.error(err instanceof Error ? err.message : t("settings.toastRemoveFailed"))
      }
    },
    [remove, t]
  )

  const handleRescanAll = useCallback(async () => {
    if (folders.length === 0) {
      toast.error(t("settings.toastNoFolders"))
      return
    }
    try {
      await triggerScan()
      toast.success(t("settings.toastRescanStarted"))
      setOnScanComplete(() => {
        refetch()
        toast.success(t("settings.toastRescanComplete"))
      })
      startPolling()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.toastRescanFailed"))
    }
  }, [folders.length, startPolling, setOnScanComplete, refetch, t])

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold mb-1">{t("settings.title")}</h2>
        <p className="text-sm text-muted-foreground">
          {t("settings.description")}
        </p>
      </div>

      {/* Theme and Language Settings */}
      <div className="border rounded-lg p-6">
        <div className="mb-4">
          <h2 className="text-lg font-semibold mb-1">{t("settings.preferences")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("settings.preferencesDescription")}
          </p>
        </div>

        <div className="space-y-6">
          {/* Theme Settings */}
          <div className="space-y-3">
            <Label>{t("settings.theme")}</Label>
            <RadioGroup value={theme} onValueChange={handleSaveTheme} className="flex gap-4">
              <div className="flex items-center space-x-2 rounded-md border p-3 cursor-pointer hover:bg-accent transition-colors">
                <RadioGroupItem value="light" id="theme-light" />
                <div className="flex items-center gap-2">
                  <Sun className="h-5 w-5 text-yellow-500" />
                  <span className="text-sm font-medium">{t("settings.lightTheme")}</span>
                </div>
              </div>
              <div className="flex items-center space-x-2 rounded-md border p-3 cursor-pointer hover:bg-accent transition-colors">
                <RadioGroupItem value="dark" id="theme-dark" />
                <div className="flex items-center gap-2">
                  <Moon className="h-5 w-5 text-blue-400" />
                  <span className="text-sm font-medium">{t("settings.darkTheme")}</span>
                </div>
              </div>
            </RadioGroup>
          </div>

          {/* Language Settings */}
          <div className="space-y-3">
            <Label>{t("settings.language")}</Label>
            <RadioGroup value={language} onValueChange={handleSaveLanguage} className="flex gap-4">
              <div className="flex items-center space-x-2 rounded-md border p-3 cursor-pointer hover:bg-accent transition-colors">
                <RadioGroupItem value="en" id="lang-en" />
                <span className="text-sm font-medium">English</span>
              </div>
              <div className="flex items-center space-x-2 rounded-md border p-3 cursor-pointer hover:bg-accent transition-colors">
                <RadioGroupItem value="ru" id="lang-ru" />
                <span className="text-sm font-medium">Русский</span>
              </div>
            </RadioGroup>
          </div>
        </div>
      </div>

      <AddFolderForm onAdd={handleAdd} disabled={status.scanning} />

      <ScanProgressBanner status={status} />

      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-muted-foreground">
          {folders.length === 1
            ? t("settings.folderCountOne", { count: folders.length })
            : t("settings.folderCount", { count: folders.length })}
        </h3>
        <Button
          variant="outline"
          size="sm"
          onClick={handleRescanAll}
          disabled={status.scanning || folders.length === 0}
        >
          <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${status.scanning ? "animate-spin" : ""}`} />
          {t("settings.rescanAll")}
        </Button>
      </div>

      <FolderList
        folders={folders}
        onRemove={handleRemove}
        isLoading={isLoading}
      />

      <div className="border-t pt-6">
        <div className="mb-4">
          <h2 className="text-lg font-semibold mb-1">{t("trash.title")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("trash.description")}
          </p>
        </div>

        <div className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="trash-dir-input">{t("trash.dirLabel")}</Label>
            <div className="flex gap-2">
              <Input
                id="trash-dir-input"
                placeholder={t("trash.dirPlaceholder")}
                value={trashInput}
                onChange={(e) => setTrashInput(e.target.value)}
                className="flex-1"
              />
              <Button
                onClick={handleSaveTrashDir}
                disabled={isSavingTrash || trashInput === trashDir}
                size="default"
              >
                {isSavingTrash ? t("trash.saving") : t("trash.save")}
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between rounded-md border p-3">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Trash2 className="h-4 w-4" />
              {!trashDir ? (
                <span>{t("trash.notConfigured")}</span>
              ) : trashFileCount === 0 ? (
                <span>{t("trash.empty")}</span>
              ) : (
                <span>{t("trash.fileCountWithSize", { count: trashFileCount, size: trashTotalSize })}</span>
              )}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCleanTrash}
              disabled={isCleaning || trashFileCount === 0 || !trashDir}
            >
              {isCleaning ? t("trash.cleaning") : t("trash.cleanButton")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
