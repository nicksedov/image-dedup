const API_BASE_URL = import.meta.env.VITE_API_URL || ""
import { translations } from "../i18n/translations"

// Global i18n registry for API calls outside React components
let globalTranslate: ((key: string) => string) | null = null

export function setGlobalTranslate(fn: (key: string) => string) {
  globalTranslate = fn
}

function handleUnauthorized(): never {
  window.dispatchEvent(new CustomEvent("navigate-to-login"))
  throw new Error("Authorization required")
}

// Function to translate API response error/message keys
export function translateApiMessage(message: string | undefined): string {
  if (!message) return "Unknown error"
  
  // If message contains dots (like "auth.invalid_credentials"), treat as i18n key
  if (message.includes(".")) {
    // Try global translate first
    if (globalTranslate) {
      const translated = globalTranslate(message)
      if (translated !== message) {
        return translated
      }
    }
    // Fallback to translation if available
    const lang = localStorage.getItem("language") as "en" | "ru" || "en"
    const dict = translations[lang] || translations.en
    const translated = (dict as Record<string, string>)[message] || message
    if (translated !== message) {
      return translated
    }
  }
  
  // Fallback to original message
  return message
}

export async function apiGet<T>(path: string, params?: Record<string, string>): Promise<T> {
  const url = new URL(`${API_BASE_URL}${path}`, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.set(key, value)
    })
  }

  const response = await fetch(url.toString(), {
    credentials: "include",
  })
  const data = await response.json()

  if (!response.ok) {
    if (response.status === 401) {
      handleUnauthorized()
    }
    const errorMessage = translateApiMessage(data.error || data.message)
    throw new Error(errorMessage)
  }

  return data as T
}

export async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: body ? JSON.stringify(body) : undefined,
  })

  const data = await response.json()

  if (!response.ok) {
    if (response.status === 401) {
      handleUnauthorized()
    }
    const errorMessage = translateApiMessage(data.error || data.message)
    throw new Error(errorMessage)
  }

  return data as T
}

export async function apiDelete<T>(path: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "DELETE",
    credentials: "include",
  })

  const data = await response.json()

  if (!response.ok) {
    if (response.status === 401) {
      handleUnauthorized()
    }
    const errorMessage = translateApiMessage(data.error || data.message)
    throw new Error(errorMessage)
  }

  return data as T
}

export async function apiPut<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: body ? JSON.stringify(body) : undefined,
  })

  const data = await response.json()

  if (!response.ok) {
    if (response.status === 401) {
      handleUnauthorized()
    }
    const errorMessage = translateApiMessage(data.error || data.message)
    throw new Error(errorMessage)
  }

  return data as T
}

export async function apiPatch<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: body ? JSON.stringify(body) : undefined,
  })

  const data = await response.json()

  if (!response.ok) {
    if (response.status === 401) {
      handleUnauthorized()
    }
    const errorMessage = translateApiMessage(data.error || data.message)
    throw new Error(errorMessage)
  }

  return data as T
}
