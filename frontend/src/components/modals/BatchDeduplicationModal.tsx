import { useEffect, useState, useMemo } from "react"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useFolderPatterns } from "@/hooks/useFolderPatterns"
import { batchDelete } from "@/api/endpoints"
import { useSettings } from "@/providers/useSettings"
import { useTranslation } from "@/i18n"
import type { BatchDeleteRule, FolderPattern } from "@/types"

interface BatchDeduplicationModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: (message: string) => void
  onError: (message: string) => void
  onComplete: () => void
}

export function BatchDeduplicationModal({
  open,
  onOpenChange,
  onSuccess,
  onError,
  onComplete,
}: BatchDeduplicationModalProps) {
  const { patterns, isLoading, error, load } = useFolderPatterns()
  const [currentStep, setCurrentStep] = useState(0)
  const [selectedFolders, setSelectedFolders] = useState<Record<string, string>>({})
  const [useTrash, setUseTrash] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isCompleted, setIsCompleted] = useState(false)
  const { trashDir } = useSettings()
  const { t } = useTranslation()

  const skippedPatterns = useMemo(() => {
    return patterns
      .filter((_, index) => index < currentStep && !selectedFolders[patterns[index].id])
      .map(p => p.id)
  }, [patterns, currentStep, selectedFolders])

  useEffect(() => {
    if (open) {
      load()
      setCurrentStep(0)
      setSelectedFolders({})
      setUseTrash(true)
      setIsCompleted(false)
    }
  }, [open, load])

  const currentPattern: FolderPattern | undefined = useMemo(() => {
    return patterns[currentStep]
  }, [patterns, currentStep])

  const totalSteps = patterns.length
  const canGoBack = currentStep > 0 || skippedPatterns.length > 0

  const handleNext = () => {
    if (currentStep < totalSteps - 1) {
      setCurrentStep(currentStep + 1)
    } else {
      setCurrentStep(totalSteps)
    }
  }

  const handleBack = () => {
    if (skippedPatterns.length > 0 && currentStep === totalSteps) {
      setCurrentStep(skippedPatterns.length - 1)
      return
    }
    if (currentStep > 0 && skippedPatterns.length > 0 && currentStep > skippedPatterns.length - 1) {
      setCurrentStep(skippedPatterns.length - 1)
    } else {
      setCurrentStep(currentStep - 1)
    }
  }

  const handlePatternSkip = (patternId: string) => {
    setSelectedFolders(prev => ({ ...prev, [patternId]: "" }))
  }

  const handleApplyStep = async (currentStepIndex: number) => {
    const rules: BatchDeleteRule[] = Object.entries(selectedFolders)
      .filter(([, folder]) => folder)
      .map(([patternId, keepFolder]) => ({ patternId, keepFolder }))

    if (rules.length === 0) {
      onError(t("batchDedup.errorNoRules"))
      return
    }

    if (!useTrash || !trashDir) {
      if (!window.confirm(t("batchDedup.confirmPermanent"))) {
        return
      }
    } else {
      if (!window.confirm(t("batchDedup.confirmApply", { count: rules.length }))) {
        return
      }
    }

    setIsSubmitting(true)
    try {
      const result = await batchDelete({
        rules,
        trashDir: useTrash ? trashDir : "",
      })
      let message: string
      if (result.failed > 0) {
        message = t("batchDedup.successWithFailed", { count: result.success, failed: result.failed })
      } else {
        message = t("batchDedup.success", { count: result.success })
      }
      onSuccess(message)
      setIsCompleted(true)
      onComplete()
    } catch (err) {
      onError(err instanceof Error ? err.message : t("batchDedup.errorFailed"))
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleFinalApply = () => {
    handleApplyStep(currentStep)
  }

  const handleBackToSkipped = () => {
    const nextSkippedIndex = skippedPatterns.length - (totalSteps - currentStep)
    if (nextSkippedIndex >= 0) {
      setCurrentStep(nextSkippedIndex)
    }
  }

  const handleCompleteClose = () => {
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col overflow-hidden">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>{t("batchDedup.title")}</DialogTitle>
          <DialogDescription>
            {t("batchDedup.description")}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 overflow-hidden flex-1 min-h-0">
          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
            </div>
          ) : error ? (
            <p className="text-sm text-destructive">{error}</p>
          ) : isCompleted ? (
            <div className="py-8 text-center space-y-2">
              <p className="text-muted-foreground">
                {t("batchDedup.success", { count: Object.keys(selectedFolders).filter(k => selectedFolders[k]).length })}
              </p>
            </div>
          ) : patterns.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">{t("batchDedup.noPatterns")}</p>
          ) : currentPattern ? (
            <div className="space-y-3">
              <div className="flex items-center justify-between text-sm text-muted-foreground pb-2 border-b">
                <span>{t("batchDedup.step", { current: currentStep + 1, total: totalSteps })}</span>
                {skippedPatterns.length > 0 && (
                  <span className="text-xs">
                    {t("batchDedup.skipped", { count: skippedPatterns.length })}
                  </span>
                )}
              </div>
              <div className="rounded-md border p-3 space-y-2">
                <div className="flex items-center gap-2 flex-wrap">
                  <Badge variant="secondary" className="text-xs">
                    {t("batchDedup.groups", { count: currentPattern.duplicateCount })}
                  </Badge>
                  <Badge variant="outline" className="text-xs">
                    {t("batchDedup.files", { count: currentPattern.totalFiles })}
                  </Badge>
                </div>
                <RadioGroup
                  value={selectedFolders[currentPattern.id] || ""}
                  onValueChange={(value) =>
                    setSelectedFolders((prev) => ({ ...prev, [currentPattern.id]: value }))
                  }
                >
                  {currentPattern.folders.map((folder) => (
                    <div key={folder} className="flex items-center gap-2 min-w-0">
                      <RadioGroupItem value={folder} id={`${currentPattern.id}-${folder}`} className="flex-shrink-0" />
                      <Label
                        htmlFor={`${currentPattern.id}-${folder}`}
                        className="text-xs font-mono cursor-pointer truncate"
                      >
                        {folder}
                      </Label>
                    </div>
                  ))}
                </RadioGroup>
              </div>
              {skippedPatterns.length > 0 && currentStep === totalSteps - 1 && (
                <div className="text-xs text-muted-foreground py-2">
                  {t("batchDedup.returnBack", { count: skippedPatterns.length })}
                </div>
              )}
            </div>
          ) : null}
          <div className="flex items-center gap-2 flex-shrink-0 pt-2 border-t">
            <Checkbox
              id="batch-use-trash"
              checked={useTrash}
              onCheckedChange={(checked) => setUseTrash(checked === true)}
            />
            <Label htmlFor="batch-use-trash" className="text-sm cursor-pointer">
              {t("batchDedup.useTrash")}
            </Label>
          </div>
          {useTrash && !trashDir && (
            <p className="text-xs text-destructive flex-shrink-0">
              {t("batchDedup.trashNotConfigured")}
            </p>
          )}
        </div>
        <DialogFooter className="flex-shrink-0">
          {isCompleted ? (
            <Button variant="outline" onClick={handleCompleteClose}>{t("common.close")}</Button>
          ) : currentStep < totalSteps - 1 ? (
            <>
              {skippedPatterns.length > 0 && currentStep === totalSteps - 2 && (
                <Button variant="ghost" onClick={handleBackToSkipped} className="mr-auto">
                  {t("batchDedup.backToSkipped")}
                </Button>
              )}
              {canGoBack ? (
                <Button variant="secondary" onClick={handleBack}>
                  {skippedPatterns.length > 0 ? t("batchDedup.backToSkipped") : t("batchDedup.back")}
                </Button>
              ) : null}
              <Button onClick={handleNext}>
                {skippedPatterns.length > 0 || currentStep < totalSteps - 2 ? t("batchDedup.forward") : t("batchDedup.finish")}
              </Button>
            </>
          ) : (
            <div className="flex items-center gap-2 w-full">
              {canGoBack ? (
                <Button variant="secondary" onClick={handleBack} disabled={isSubmitting}>
                  {skippedPatterns.includes(currentPattern?.id || "") ? t("batchDedup.backToPattern") : t("batchDedup.back")}
                </Button>
              ) : null}
              {skippedPatterns.length > 0 ? (
                <Button variant="outline" onClick={() => handlePatternSkip(currentPattern?.id || "")} disabled={isSubmitting}>
                  {t("batchDedup.skipThis")}
                </Button>
              ) : null}
              <Button variant="destructive" onClick={handleFinalApply} disabled={isSubmitting || isLoading} className="ml-auto">
                {isSubmitting ? t("batchDedup.applying") : t("batchDedup.applyRules")}
              </Button>
            </div>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
