import { useTranslation } from "@/i18n"
import { ThemeToggle } from "./ThemeToggle"
import { LanguageToggle } from "./LanguageToggle"
import { useAuth } from "@/providers/AuthProvider"
import { LogOut, User } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

export function Header() {
  const { t } = useTranslation()
  const { user, logout } = useAuth()

  return (
    <header className="border-b bg-gradient-to-r from-blue-600 to-indigo-700 text-white">
      <div className="mx-auto max-w-7xl px-4 py-4 sm:px-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("header.title")}</h1>
          <p className="text-sm text-blue-100">
            {t("header.subtitle")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {user && (
            <div className="flex items-center gap-3 mr-2">
              <div className="flex items-center gap-2 rounded-lg bg-white/10 px-3 py-1.5">
                <User className="h-4 w-4" />
                <span className="text-sm font-medium">{user.displayName}</span>
                <Badge variant="secondary" className="text-xs">
                  {user.role === "admin" ? t("adminPanel.roleAdmin") : t("adminPanel.roleUser")}
                </Badge>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={logout}
                className="text-white hover:bg-white/20 hover:text-white"
              >
                <LogOut className="mr-1.5 h-4 w-4" />
                {t("adminPanel.logout")}
              </Button>
            </div>
          )}
          <ThemeToggle />
          <LanguageToggle />
        </div>
      </div>
    </header>
  )
}
