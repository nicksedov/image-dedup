import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Checkbox } from "@/components/ui/checkbox"
import { AddFolderForm } from "@/components/settings/AddFolderForm"
import { FolderList } from "@/components/settings/FolderList"
import { ScanProgressBanner } from "@/components/ScanProgressBanner"
import { useGalleryFolders } from "@/hooks/useGalleryFolders"
import { useScanStatus } from "@/hooks/useScanStatus"
import { fetchTrashInfo, cleanTrash, updateSettings, fetchOCRStatus, startOcrClassification, startOcrClassificationChanges, stopOcrClassification, fetchOcrClassificationStatus, triggerScan, triggerFastScan, fetchLlmSettings, updateLlmSettings, fetchLlmModels } from "@/api/endpoints"
import { useSettings } from "@/providers/useSettings"
import { useAuth } from "@/providers/AuthProvider"
import { RefreshCw, Trash2, Shield, Loader2, Zap, Wand2, Play, Square } from "lucide-react"
import { useTranslation, type TranslationKey } from "@/i18n"
import type { OCRStatus, OcrClassificationStatusResponse, LlmSettingsDTO, LlmModelDTO } from "@/types"

export function AdminSettingsTab() {
  const { folders, isLoading, add, remove, refetch } = useGalleryFolders()
  const { status, startPolling, setOnScanComplete } = useScanStatus()
  const { trashDir, setTrashDir } = useSettings()
  const { user } = useAuth()
  const { t } = useTranslation()
  const isAdmin = user?.role === "admin"

  const [trashInput, setTrashInput] = useState(trashDir)
  const [trashFileCount, setTrashFileCount] = useState(0)
  const [trashTotalSize, setTrashTotalSize] = useState("")
  const [isCleaning, setIsCleaning] = useState(false)
  const [isSavingTrash, setIsSavingTrash] = useState(false)
  const [ocrStatus, setOcrStatus] = useState<OCRStatus | null>(null)
  const [isOcrLoading, setIsOcrLoading] = useState(false)
  const [ocrScanning, setOcrScanning] = useState(false)
  const [ocrScanStatus, setOcrScanStatus] = useState<OcrClassificationStatusResponse | null>(null)

  // LLM Settings state
  const [llmSettings, setLlmSettings] = useState<LlmSettingsDTO>({
    id: 0,
    provider: "ollama",
    apiUrl: "http://localhost:11434",
    apiKey: "",
    model: "minicpm-v",
    enabled: false,
  })
  const [isLlmLoading, setIsLlmLoading] = useState(false)
  const [isLlmSaving, setIsLlmSaving] = useState(false)
  const [llmFormDirty, setLlmFormDirty] = useState(false)
  const [availableModels, setAvailableModels] = useState<LlmModelDTO[]>([])
  const [isModelsLoading, setIsModelsLoading] = useState(false)
  const [showModelInput, setShowModelInput] = useState(false)

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

  const loadOCRStatus = useCallback(async () => {
    try {
      setIsOcrLoading(true)
      const response = await fetchOCRStatus()
      setOcrStatus(response.status)
    } catch {
      setOcrStatus(null)
    } finally {
      setIsOcrLoading(false)
    }
  }, [])

  // Poll OCR scan status when scanning
  useEffect(() => {
    if (!ocrScanning) return

    let cancelled = false

    const checkStatus = async () => {
      try {
        const status = await fetchOcrClassificationStatus()
        if (cancelled) return
        setOcrScanStatus(status)
        setOcrScanning(status.processing)
      } catch (err) {
        console.error("Failed to check OCR scan status:", err)
      }
    }

    checkStatus()
    const interval = setInterval(() => {
      if (!cancelled) {
        checkStatus()
      }
    }, 2000)

    return () => {
      cancelled = true
      clearInterval(interval)
    }
  }, [ocrScanning])

  const handleStartOcrScanAll = useCallback(async () => {
    try {
      await startOcrClassification()
      setOcrScanning(true)
      toast.success(t("api.ocr.started"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("api.ocr.failed"))
    }
  }, [t])

  const handleStartOcrScanChanges = useCallback(async () => {
    try {
      await startOcrClassificationChanges()
      setOcrScanning(true)
      toast.success(t("api.ocr.started"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("api.ocr.failed"))
    }
  }, [t])

  const handleStopOcrScan = useCallback(async () => {
    try {
      await stopOcrClassification()
      toast.info("OCR scanning stopping...")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("api.ocr.failed"))
    }
  }, [t])

  const loadLlmSettings = useCallback(async () => {
    try {
      setIsLlmLoading(true)
      const settings = await fetchLlmSettings()
      setLlmSettings(settings)
      setLlmFormDirty(false)
    } catch {
      setLlmSettings({
        id: 0,
        provider: "ollama",
        apiUrl: "http://localhost:11434",
        apiKey: "",
        model: "minicpm-v",
        enabled: false,
      })
    } finally {
      setIsLlmLoading(false)
    }
  }, [])

  const handleSaveLlmSettings = useCallback(async () => {
    setIsLlmSaving(true)
    try {
      await updateLlmSettings({
        provider: llmSettings.provider,
        apiUrl: llmSettings.apiUrl,
        apiKey: llmSettings.apiKey,
        model: llmSettings.model,
        enabled: llmSettings.enabled,
      })
      toast.success(t("llm_ocr.settingsSaved"))
      setLlmFormDirty(false)
      await loadLlmSettings()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("llm_ocr.settingsSaveFailed"))
    } finally {
      setIsLlmSaving(false)
    }
  }, [llmSettings, loadLlmSettings, t])

  const handleLlmFieldChange = useCallback((field: keyof LlmSettingsDTO, value: string | boolean) => {
    setLlmSettings((prev) => ({ ...prev, [field]: value }))
    setLlmFormDirty(true)
  }, [])

  const handleLoadModels = useCallback(async () => {
    setIsModelsLoading(true)
    try {
      const response = await fetchLlmModels()
      if (response.success && response.models.length > 0) {
        setAvailableModels(response.models)
        toast.success(`Загружено ${response.models.length} моделей`)
      } else {
        toast.error(response.error || "Не удалось загрузить список моделей")
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Не удалось загрузить список моделей")
    } finally {
      setIsModelsLoading(false)
    }
  }, [])

  useEffect(() => {
    loadTrashInfo()
  }, [loadTrashInfo])

  useEffect(() => {
    if (isAdmin) {
      loadOCRStatus()
      loadLlmSettings()
    }
  }, [isAdmin, loadOCRStatus, loadLlmSettings])

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
        toast.success(message.includes(".") ? t(message as TranslationKey) : message)
        if (result.scanStarted) {
          setOnScanComplete(() => {
            refetch()
            toast.success(t("settings.toastScanComplete"))
          })
          startPolling()
        }
      } catch (err) {
        toast.error(err instanceof Error ? err.message : t("settings.toastAddFailed"))
      }
    },
    [add, startPolling, setOnScanComplete, refetch, t]
  )

  const handleRemove = useCallback(
    async (id: number) => {
      try {
        const result = await remove(id)
        const message = result.message as string
        toast.success(t("settings.toastFilesRemoved", { message: message.includes(".") ? t(message as TranslationKey) : message, count: result.filesRemoved }))
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

  const handleFastRescanAll = useCallback(async () => {
    if (folders.length === 0) {
      toast.error(t("settings.toastNoFolders"))
      return
    }
    try {
      const result = await triggerFastScan()
      toast.success(t("settings.toastFastScanStarted"))
      setOnScanComplete(() => {
        refetch()
        const statsMsg = [
          t("settings.fastScanStats", { unchanged: result.unchanged }),
          t("settings.fastScanModified", { modified: result.modified }),
          t("settings.fastScanCreated", { created: result.created }),
          t("settings.fastScanDeleted", { deleted: result.deleted }),
        ].join(", ")
        toast.success(t("settings.toastFastScanComplete") + " (" + statsMsg + ")")
      })
      startPolling()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.toastRescanFailed"))
    }
  }, [folders.length, startPolling, setOnScanComplete, refetch, t])

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold mb-1">{t("adminPanel.adminSettings")}</h2>
        <p className="text-sm text-muted-foreground">
          {t("adminPanel.adminSettingsDescription")}
        </p>
      </div>

      {/* Gallery Folder Management */}
      {isAdmin && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>{t("settings.galleryFolders")}</CardTitle>
              <CardDescription>{t("settings.galleryFoldersDescription")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <AddFolderForm onAdd={handleAdd} disabled={status.scanning} />

              <ScanProgressBanner status={status} />

              <div className="flex items-center justify-between gap-2">
                <h3 className="text-sm font-medium text-muted-foreground">
                  {folders.length === 1
                    ? t("settings.folderCountOne", { count: folders.length })
                    : t("settings.folderCount", { count: folders.length })}
                </h3>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleFastRescanAll}
                    disabled={status.scanning || folders.length === 0}
                  >
                    <Zap className={`mr-1.5 h-3.5 w-3.5 ${status.scanning ? "animate-spin" : ""}`} />
                    {t("settings.fastScanChanges")}
                  </Button>
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
              </div>

              <FolderList
                folders={folders}
                onRemove={handleRemove}
                isLoading={isLoading}
              />
            </CardContent>
          </Card>
        </>
      )}

      {/* Trash Settings - Admin Only */}
      {isAdmin && (
        <Card>
          <CardHeader>
            <CardTitle>{t("trash.title")}</CardTitle>
            <CardDescription>{t("trash.description")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
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
          </CardContent>
        </Card>
      )}

      {/* OCR Document Search - Admin Only */}
      {isAdmin && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              {t("adminPanel.ocr.title")}
            </CardTitle>
            <CardDescription>{t("adminPanel.ocr.description")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* OCR Service Status */}
            <div className="flex items-center justify-between rounded-lg border p-3">
              <div className="space-y-1">
                <div className="text-sm font-medium">{t("adminPanel.ocr.status")}</div>
                {isOcrLoading ? (
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    {t("common.loading")}
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <span
                      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                        ocrStatus?.enabled && ocrStatus?.health === "healthy"
                          ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                          : ocrStatus?.error || (ocrStatus?.enabled && ocrStatus?.health !== "healthy")
                          ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                          : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                      }`}
                    >
                      {ocrStatus?.enabled && ocrStatus?.health === "healthy"
                        ? t("adminPanel.ocr.statusHealthy")
                        : ocrStatus?.error || (ocrStatus?.enabled && ocrStatus?.health !== "healthy")
                        ? t("adminPanel.ocr.statusError")
                        : t("adminPanel.ocr.statusDisabled")}
                    </span>
                  </div>
                )}
              </div>
              <Button variant="outline" size="sm" onClick={loadOCRStatus} disabled={isOcrLoading}>
                {isOcrLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
              </Button>
            </div>

            {/* OCR Scan Progress */}
            {ocrScanning && ocrScanStatus && (
              <div className="p-4 bg-muted rounded-lg">
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-sm">
                    {t("ocr.filesProcessed", {
                      count: ocrScanStatus.filesProcessed,
                      total: ocrScanStatus.totalFiles,
                    })}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground mt-1">{ocrScanStatus.progress}</p>
              </div>
            )}

            {/* Scan Buttons */}
            <div className="flex gap-2">
              <Button
                onClick={handleStartOcrScanChanges}
                disabled={ocrScanning}
                variant="outline"
                size="sm"
              >
                <Zap className={`mr-1.5 h-3.5 w-3.5 ${ocrScanning ? "animate-spin" : ""}`} />
                {t("adminPanel.ocr.scanChanges")}
              </Button>
              <Button
                onClick={handleStartOcrScanAll}
                disabled={ocrScanning}
                variant="outline"
                size="sm"
              >
                <Play className={`mr-1.5 h-3.5 w-3.5 ${ocrScanning ? "animate-spin" : ""}`} />
                {t("adminPanel.ocr.scanAll")}
              </Button>
              {ocrScanning && (
                <Button
                  onClick={handleStopOcrScan}
                  variant="destructive"
                  size="sm"
                >
                  <Square className="mr-1.5 h-3.5 w-3.5" />
                  {t("adminPanel.ocr.stopScanning")}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* LLM Settings - Admin Only */}
      {isAdmin && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wand2 className="h-5 w-5" />
              {t("llm_ocr.settings")}
            </CardTitle>
            <CardDescription>{t("llm_ocr.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            {isLlmLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : (
              <div className="space-y-4">
                {/* Provider Selection */}
                <div className="space-y-2">
                  <Label htmlFor="llm-provider">{t("llm_ocr.provider")}</Label>
                  <Select
                    value={llmSettings.provider}
                    onValueChange={(value) => handleLlmFieldChange("provider", value)}
                  >
                    <SelectTrigger id="llm-provider">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ollama">Ollama</SelectItem>
                      <SelectItem value="openai">OpenAI API</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {/* API URL */}
                <div className="space-y-2">
                  <Label htmlFor="llm-apiurl">API URL</Label>
                  <Input
                    id="llm-apiurl"
                    placeholder={llmSettings.provider === "ollama" ? "http://localhost:11434" : "https://api.openai.com"}
                    value={llmSettings.apiUrl}
                    onChange={(e) => handleLlmFieldChange("apiUrl", e.target.value)}
                  />
                </div>

                {/* API Key (only for OpenAI) */}
                {llmSettings.provider === "openai" && (
                  <div className="space-y-2">
                    <Label htmlFor="llm-apikey">API Key</Label>
                    <Input
                      id="llm-apikey"
                      type="password"
                      placeholder="sk-..."
                      value={llmSettings.apiKey}
                      onChange={(e) => handleLlmFieldChange("apiKey", e.target.value)}
                    />
                  </div>
                )}

                {/* Model */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="llm-model">{t("llm_ocr.model")}</Label>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleLoadModels}
                      disabled={isModelsLoading}
                      className="h-6 px-2 text-xs"
                    >
                      {isModelsLoading ? (
                        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                      ) : (
                        <RefreshCw className="mr-1 h-3 w-3" />
                      )}
                      Загрузить модели
                    </Button>
                  </div>

                  {/* Model dropdown */}
                  {availableModels.length > 0 && !showModelInput ? (
                    <div className="space-y-2">
                      <Select
                        value={llmSettings.model}
                        onValueChange={(value) => handleLlmFieldChange("model", value)}
                      >
                        <SelectTrigger id="llm-model">
                          <SelectValue placeholder="Выберите модель" />
                        </SelectTrigger>
                        <SelectContent>
                          {availableModels.map((model) => (
                            <SelectItem key={model.id} value={model.id}>
                              {model.name}
                              {model.size ? ` (${(model.size / 1073741824).toFixed(1)} GB)` : ""}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <Button
                        variant="link"
                        size="sm"
                        className="px-0 h-auto text-xs"
                        onClick={() => setShowModelInput(true)}
                      >
                        Ввести название модели вручную
                      </Button>
                    </div>
                  ) : (
                    <div className="space-y-2">
                      <Input
                        id="llm-model"
                        placeholder={llmSettings.provider === "ollama" ? "minicpm-v" : "gpt-4-vision-preview"}
                        value={llmSettings.model}
                        onChange={(e) => handleLlmFieldChange("model", e.target.value)}
                      />
                      {availableModels.length > 0 && showModelInput && (
                        <Button
                          variant="link"
                          size="sm"
                          className="px-0 h-auto text-xs"
                          onClick={() => setShowModelInput(false)}
                        >
                          Выбрать из списка доступных моделей
                        </Button>
                      )}
                    </div>
                  )}
                </div>

                {/* Enable/Disable Checkbox */}
                <div className="flex items-center space-x-2 rounded-lg border p-3">
                  <Checkbox
                    id="llm-enabled"
                    checked={llmSettings.enabled}
                    onCheckedChange={(checked) => handleLlmFieldChange("enabled", checked === true)}
                  />
                  <div className="space-y-0.5">
                    <Label htmlFor="llm-enabled">{t("llm_ocr.enableRecognition")}</Label>
                    <p className="text-xs text-muted-foreground">
                      {t("llm_ocr.enableDescription")}
                    </p>
                  </div>
                </div>

                {/* Save Button */}
                <div className="flex justify-end pt-2">
                  <Button
                    onClick={handleSaveLlmSettings}
                    disabled={isLlmSaving || !llmFormDirty}
                  >
                    {isLlmSaving ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        {t("common.saving")}
                      </>
                    ) : (
                      t("common.save")
                    )}
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
