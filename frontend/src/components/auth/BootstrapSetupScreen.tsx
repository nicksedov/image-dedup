import { useState } from "react"
import { useAuth } from "@/providers/AuthProvider"
import { bootstrapSetup as apiBootstrapSetup } from "@/api/endpoints"
import { toast } from "sonner"
import { Loader2, Settings } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function BootstrapSetupScreen() {
  const { login } = useAuth()
  const [displayName, setDisplayName] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")

    if (!displayName.trim()) {
      setError("Введите отображаемое имя")
      return
    }

    if (newPassword.length < 8) {
      setError("Пароль должен содержать не менее 8 символов")
      return
    }

    if (newPassword !== confirmPassword) {
      setError("Пароли не совпадают")
      return
    }

    setIsLoading(true)
    try {
      const response = await apiBootstrapSetup({ newPassword, displayName })
      login(response.user)
      toast.success("Первичная настройка завершена")
    } catch (err) {
      const message = err instanceof Error ? err.message : "Не удалось завершить настройку"
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
            <Settings className="h-6 w-6 text-primary" />
          </div>
          <CardTitle className="text-2xl font-bold">Первичная настройка</CardTitle>
          <CardDescription>
            Создайте учетную запись администратора и задайте постоянный пароль
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="displayName">Отображаемое имя</Label>
              <Input
                id="displayName"
                type="text"
                placeholder="Администратор"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                disabled={isLoading}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="newPassword">Новый пароль</Label>
              <Input
                id="newPassword"
                type="password"
                autoComplete="new-password"
                placeholder="Минимум 8 символов"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                disabled={isLoading}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Подтверждение пароля</Label>
              <Input
                id="confirmPassword"
                type="password"
                autoComplete="new-password"
                placeholder="Повторите пароль"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
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
              {isLoading ? "Настройка..." : "Завершить настройку"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
