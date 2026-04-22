import { useCallback, useEffect, useRef, useState, type ReactNode } from "react"
import { ThemeProvider, type Theme } from "@/theme"
import { I18nProvider, type Language } from "@/i18n"
import { fetchSettings, updateSettings, fetchUserSettings, updateUserSettings } from "@/api/endpoints"
import { SettingsContext } from "./settingsContext"
import { useAuth } from "./AuthProvider"

interface SettingsProviderProps {
  children: ReactNode
}

export function SettingsProvider({ children }: SettingsProviderProps) {
  const [theme, setThemeState] = useState<Theme>("light")
  const [language, setLanguageState] = useState<Language>("en")
  const [trashDir, setTrashDirState] = useState("")
  const [isLoading, setIsLoading] = useState(true)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const { isAuthenticated } = useAuth()

  useEffect(() => {
    if (!isAuthenticated) {
      setIsLoading(false)
      return
    }
    
    // Load user settings first, fallback to global settings
    Promise.all([
      fetchUserSettings().catch(() => null),
      fetchSettings().catch(() => null),
    ]).then(([userSettings, globalSettings]) => {
      // Prefer user settings, fallback to global settings
      const effectiveTheme = userSettings?.theme || globalSettings?.theme || "light"
      const effectiveLanguage = userSettings?.language || globalSettings?.language || "en"
      const effectiveTrashDir = userSettings?.trashDir || globalSettings?.trashDir || ""
      
      setThemeState(effectiveTheme)
      setLanguageState(effectiveLanguage)
      setTrashDirState(effectiveTrashDir)
    }).catch(() => {
      // Use defaults on failure
    }).finally(() => setIsLoading(false))
  }, [isAuthenticated])

  const persistSettings = useCallback((newTheme: Theme, newLanguage: Language, newTrashDir?: string) => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }
    debounceRef.current = setTimeout(() => {
      const req: { theme: Theme; language: Language; trashDir?: string } = {
        theme: newTheme,
        language: newLanguage,
      }
      if (newTrashDir !== undefined) {
        req.trashDir = newTrashDir
      }
      updateUserSettings(req).catch(() => {
        // Silently fail - UI already updated
      })
    }, 300)
  }, [])

  const setTheme = useCallback(
    (newTheme: Theme) => {
      setThemeState(newTheme)
      setLanguageState((lang) => {
        persistSettings(newTheme, lang)
        return lang
      })
    },
    [persistSettings]
  )

  const toggleTheme = useCallback(() => {
    setThemeState((prev) => {
      const next = prev === "light" ? "dark" : "light"
      setLanguageState((lang) => {
        persistSettings(next, lang)
        return lang
      })
      return next
    })
  }, [persistSettings])

  const setLanguage = useCallback(
    (newLanguage: Language) => {
      setLanguageState(newLanguage)
      setThemeState((th) => {
        persistSettings(th, newLanguage)
        return th
      })
    },
    [persistSettings]
  )

  const setTrashDir = useCallback(
    (newTrashDir: string) => {
      setTrashDirState(newTrashDir)
      setThemeState((th) => {
        setLanguageState((lang) => {
          persistSettings(th, lang, newTrashDir)
          return lang
        })
        return th
      })
    },
    [persistSettings]
  )

  return (
    <SettingsContext.Provider
      value={{ theme, setTheme, toggleTheme, language, setLanguage, trashDir, setTrashDir, isLoading }}
    >
      <ThemeProvider theme={theme}>
        <I18nProvider language={language}>
          {children}
        </I18nProvider>
      </ThemeProvider>
    </SettingsContext.Provider>
  )
}
