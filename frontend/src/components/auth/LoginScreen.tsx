import { useEffect, useState } from "react"
import { useAuth } from "@/providers/AuthProvider"
import { login as apiLogin } from "@/api/endpoints"
import { toast } from "sonner"
import { Loader2, ShieldAlert } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useTranslation } from "@/i18n"

export function LoginScreen() {
  const { login, isBootstrapMode, setBootstrapVerified } = useAuth()
  const { t } = useTranslation()
  const [loginInput, setLoginInput] = useState("")
  const [password, setPassword] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")

  useEffect(() => {
    const handleNavigateToProfile = () => {
      toast.info(t("adminPanel.loginAgain"))
    }
    window.addEventListener("navigate-to-profile", handleNavigateToProfile as EventListener)
    return () => {
      window.removeEventListener("navigate-to-profile", handleNavigateToProfile as EventListener)
    }
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")

    if (!loginInput.trim() || !password.trim()) {
      setError(t("adminPanel.wrongFields"))
      return
    }

    setIsLoading(true)
    try {
      const response = await apiLogin({ login: loginInput, password })

      if (response.isBootstrap) {
        // Show bootstrap setup screen
        setBootstrapVerified(true)
        return
      }

      if (response.user) {
        login(response.user)
        toast.success(t("adminPanel.loginSuccess"))
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : t("adminPanel.loginFailed")
      setError(message)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-2 text-center">
          <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <ShieldAlert className="h-6 w-6 text-primary" />
          </div>
          <CardTitle className="text-2xl font-bold">
            {isBootstrapMode ? t("adminPanel.bootstrapTitle") : t("adminPanel.loginTitle")}
          </CardTitle>
          <CardDescription>
            {isBootstrapMode
              ? t("adminPanel.bootstrapDescAdmin")
              : t("adminPanel.loginDesc")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="login">{t("adminPanel.login")}</Label>
              <Input
                id="login"
                type="text"
                autoComplete="username"
                placeholder={t("adminPanel.loginPlaceholder")}
                value={loginInput}
                onChange={(e) => setLoginInput(e.target.value)}
                disabled={isLoading}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t("adminPanel.passwordPlaceholder")}</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                placeholder={t("adminPanel.passwordPlaceholder")}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isLoading}
              />
            </div>
            {error && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {error}
              </div>
            )}
            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isLoading ? t("adminPanel.setup") : t("adminPanel.loginTitle")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
