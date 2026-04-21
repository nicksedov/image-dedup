import { useCallback, useEffect, type ReactNode } from "react"
import { translations } from "./translations"
import { I18nContext } from "./context"
import type { Language, TranslationKey } from "./types"
import { setGlobalTranslate } from "@/api/client"

interface I18nProviderProps {
  language: Language
  children: ReactNode
}

export function I18nProvider({ language, children }: I18nProviderProps) {
  const t = useCallback(
    (key: TranslationKey, params?: Record<string, string | number>): string => {
      const dict = translations[language] ?? translations.en
      let value = (dict as Record<string, string>)[key] ?? (translations.en as Record<string, string>)[key] ?? key
      if (params) {
        for (const [k, v] of Object.entries(params)) {
          value = value.replaceAll(`{${k}}`, String(v))
        }
      }
      return value
    },
    [language]
  )

  // Register global translate function for API calls
  useEffect(() => {
    const wrapperT = (key: string) => t(key as any)
    setGlobalTranslate(wrapperT)
  }, [t])

  return (
    <I18nContext.Provider value={{ language, t }}>
      {children}
    </I18nContext.Provider>
  )
}
