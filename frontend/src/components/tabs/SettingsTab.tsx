import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { ThemeSelect, ThemeSelectContent, ThemeSelectItem, ThemeSelectTrigger, ThemeSelectValue } from "@/components/ui/theme-select"
import type { Theme } from "@/theme"
import { updateSettings } from "@/api/endpoints"
import { useSettings } from "@/providers/useSettings"
import { Globe, Sun, Moon } from "lucide-react"
import { useTranslation } from "@/i18n"

export function SettingsTab() {
  const { theme, setTheme, language, setLanguage } = useSettings()
  const { t } = useTranslation()

  const [selectedTheme, setSelectedTheme] = useState<Theme>(theme as Theme)
  const [selectedLanguage, setSelectedLanguage] = useState<"en" | "ru">(language)
  const [isSaving, setIsSaving] = useState(false)

  // Sync local state with settings when they change
  useEffect(() => {
    setSelectedTheme(theme)
  }, [theme])

  useEffect(() => {
    setSelectedLanguage(language)
  }, [language])

  const handleSavePreferences = useCallback(async () => {
    setIsSaving(true)
    try {
      await updateSettings({ theme: selectedTheme, language: selectedLanguage })
      setTheme(selectedTheme)
      setLanguage(selectedLanguage)
      toast.success(t("settings.preferencesSaved"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.saveFailed"))
    } finally {
      setIsSaving(false)
    }
  }, [selectedTheme, selectedLanguage, setTheme, setLanguage, t])

  return (
    <div className="space-y-6">
      {/* Preferences Header - Outside of card */}
      <div>
        <h2 className="text-lg font-semibold mb-1">{t("settings.preferences")}</h2>
        <p className="text-sm text-muted-foreground">
          {t("settings.preferencesDescription")}
        </p>
      </div>

      {/* Theme and Language Settings */}
      <div className="space-y-4">
        {/* Theme Setting */}
        <div className="space-y-2">
          <Label htmlFor="theme-select">{t("settings.theme")}</Label>
          <ThemeSelect value={selectedTheme} onValueChange={(v) => setSelectedTheme(v as Theme)}>
            <ThemeSelectTrigger id="theme-select">
              <ThemeSelectValue placeholder={t("settings.selectTheme")} />
            </ThemeSelectTrigger>
            <ThemeSelectContent>
              <ThemeSelectItem value="light-purple">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-purple-200" />
                  <Sun className="h-4 w-4 text-yellow-500" />
                  {t("settings.lightPurpleTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="dark-purple">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-purple-900" />
                  <Moon className="h-4 w-4 text-blue-400" />
                  {t("settings.darkPurpleTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="light-green">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-green-200" />
                  <Sun className="h-4 w-4 text-yellow-500" />
                  {t("settings.lightGreenTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="dark-green">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-green-900" />
                  <Moon className="h-4 w-4 text-blue-400" />
                  {t("settings.darkGreenTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="light-blue">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-blue-200" />
                  <Sun className="h-4 w-4 text-yellow-500" />
                  {t("settings.lightBlueTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="dark-blue">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-blue-900" />
                  <Moon className="h-4 w-4 text-blue-400" />
                  {t("settings.darkBlueTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="light-orange">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-orange-200" />
                  <Sun className="h-4 w-4 text-yellow-500" />
                  {t("settings.lightOrangeTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="dark-orange">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-orange-900" />
                  <Moon className="h-4 w-4 text-blue-400" />
                  {t("settings.darkOrangeTheme")}
                </span>
              </ThemeSelectItem>
              <ThemeSelectItem value="dark-contrast">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-gray-900" />
                  <Moon className="h-4 w-4 text-blue-400" />
                  {t("settings.darkContrastTheme")}
                </span>
              </ThemeSelectItem>
            </ThemeSelectContent>
          </ThemeSelect>
        </div>

        {/* Language Setting */}
        <div className="space-y-2">
          <Label htmlFor="language-select">{t("settings.language")}</Label>
          <Select value={selectedLanguage} onValueChange={(v) => setSelectedLanguage(v as "en" | "ru")}>
            <SelectTrigger id="language-select">
              <SelectValue>
                <span className="flex items-center gap-2">
                  <Globe className="h-4 w-4" />
                  {selectedLanguage === "en" ? "English" : "Русский"}
                </span>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="en">
                <span className="flex items-center gap-2">
                  <Globe className="h-4 w-4" />
                  English
                </span>
              </SelectItem>
              <SelectItem value="ru">
                <span className="flex items-center gap-2">
                  <Globe className="h-4 w-4" />
                  Русский
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Save Button - Left adjusted, not full width */}
        <Button
          onClick={handleSavePreferences}
          disabled={isSaving || (selectedTheme === theme && selectedLanguage === language)}
        >
          {isSaving ? t("common.saving") : t("settings.savePreferences")}
        </Button>
      </div>
    </div>
  )
}
