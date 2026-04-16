import { useState } from "react"
import { useAuth } from "@/providers/AuthProvider"
import { changePassword as apiChangePassword, updateProfile as apiUpdateProfile } from "@/api/endpoints"
import { toast } from "sonner"
import { Loader2, User, Lock } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Badge } from "@/components/ui/badge"

export function UserProfile() {
  const { user, updateUser } = useAuth()
  const [displayName, setDisplayName] = useState(user?.displayName || "")
  const [isSavingProfile, setIsSavingProfile] = useState(false)

  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [isChangingPassword, setIsChangingPassword] = useState(false)

  if (!user) return null

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!displayName.trim()) return

    setIsSavingProfile(true)
    try {
      const response = await apiUpdateProfile({ displayName })
      updateUser(response.user)
      toast.success("Профиль обновлен")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Не удалось обновить профиль")
    } finally {
      setIsSavingProfile(false)
    }
  }

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()

    if (newPassword.length < 8) {
      toast.error("Пароль должен содержать не менее 8 символов")
      return
    }

    if (newPassword !== confirmPassword) {
      toast.error("Пароли не совпадают")
      return
    }

    setIsChangingPassword(true)
    try {
      await apiChangePassword({ oldPassword, newPassword })
      setOldPassword("")
      setNewPassword("")
      setConfirmPassword("")
      toast.success("Пароль изменен")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Не удалось изменить пароль")
    } finally {
      setIsChangingPassword(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Профиль</h2>
        <p className="text-muted-foreground">Управление учетной записью</p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <User className="h-5 w-5" />
            <CardTitle>Информация о пользователе</CardTitle>
          </div>
          <CardDescription>Ваша учетная запись и роль в системе</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">{user.displayName}</p>
              <p className="text-sm text-muted-foreground">{user.login}</p>
            </div>
            <Badge variant={user.role === "admin" ? "default" : "secondary"}>
              {user.role === "admin" ? "Администратор" : "Пользователь"}
            </Badge>
          </div>
          <Separator />
          <form onSubmit={handleSaveProfile} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="displayName">Отображаемое имя</Label>
              <Input
                id="displayName"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={isSavingProfile || displayName === user.displayName}>
              {isSavingProfile && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Сохранить
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Lock className="h-5 w-5" />
            <CardTitle>Смена пароля</CardTitle>
          </div>
          <CardDescription>Обновите свой пароль для безопасности</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleChangePassword} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="oldPassword">Текущий пароль</Label>
              <Input
                id="oldPassword"
                type="password"
                autoComplete="current-password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="newPassword">Новый пароль</Label>
              <Input
                id="newPassword"
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Подтверждение пароля</Label>
              <Input
                id="confirmPassword"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={isChangingPassword || !oldPassword || !newPassword || !confirmPassword}>
              {isChangingPassword && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Изменить пароль
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
