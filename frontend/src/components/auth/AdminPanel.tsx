import { useCallback, useEffect, useState } from "react"
import { useAuth } from "@/providers/AuthProvider"
import { fetchUsers, createUser, updateUser, deleteUser, resetUserPassword } from "@/api/endpoints"
import { toast } from "sonner"
import { Loader2, Trash2, KeyRound, Pencil, Save, X, Users, UserPlus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { UserDTO, UserRole } from "@/types"

export function AdminPanel() {
  const { user: currentUser } = useAuth()
  const [users, setUsers] = useState<UserDTO[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<UserDTO | null>(null)
  const [resettingUser, setResettingUser] = useState<UserDTO | null>(null)

  const loadUsers = useCallback(async () => {
    try {
      const response = await fetchUsers()
      setUsers(response.users)
    } catch {
      toast.error("Не удалось загрузить список пользователей")
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    loadUsers()
  }, [loadUsers])

  if (currentUser?.role !== "admin") {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-muted-foreground">Доступ запрещен</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Управление пользователями</h2>
          <p className="text-muted-foreground">Создание и настройка учетных записей</p>
        </div>
        <Button onClick={() => setIsCreateOpen(true)}>
          <UserPlus className="mr-2 h-4 w-4" />
          Создать пользователя
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : users.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Users className="mb-4 h-12 w-12 text-muted-foreground" />
            <p className="text-lg font-medium">Нет пользователей</p>
            <p className="text-sm text-muted-foreground">Создайте первую учетную запись</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {users.map((u) => (
            <UserCard
              key={u.id}
              user={u}
              isCurrentUser={u.id === currentUser?.id}
              onEdit={() => setEditingUser(u)}
              onResetPassword={() => setResettingUser(u)}
              onDelete={async () => {
                if (!confirm(`Удалить пользователя "${u.displayName}"?`)) return
                try {
                  await deleteUser(u.id)
                  toast.success("Пользователь удален")
                  loadUsers()
                } catch {
                  toast.error("Не удалось удалить пользователя")
                }
              }}
              onToggleActive={async () => {
                try {
                  await updateUser(u.id, { isActive: !u.isActive })
                  toast.success(u.isActive ? "Пользователь деактивирован" : "Пользователь активирован")
                  loadUsers()
                } catch {
                  toast.error("Не удалось обновить пользователя")
                }
              }}
            />
          ))}
        </div>
      )}

      <CreateUserDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} onSuccess={loadUsers} />
      {editingUser && (
        <EditUserDialog user={editingUser} onClose={() => setEditingUser(null)} onSuccess={loadUsers} />
      )}
      {resettingUser && (
        <ResetPasswordDialog user={resettingUser} onClose={() => setResettingUser(null)} />
      )}
    </div>
  )
}

function UserCard({
  user,
  isCurrentUser,
  onEdit,
  onResetPassword,
  onDelete,
  onToggleActive,
}: {
  user: UserDTO
  isCurrentUser: boolean
  onEdit: () => void
  onResetPassword: () => void
  onDelete: () => Promise<void>
  onToggleActive: () => Promise<void>
}) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between p-4">
        <div className="flex items-center gap-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
            <span className="font-medium text-primary">{user.displayName.charAt(0).toUpperCase()}</span>
          </div>
          <div>
            <p className="font-medium">
              {user.displayName}
              {isCurrentUser && <span className="ml-2 text-xs text-muted-foreground">(Вы)</span>}
            </p>
            <p className="text-sm text-muted-foreground">{user.login}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={user.role === "admin" ? "default" : "secondary"}>
            {user.role === "admin" ? "Админ" : "Пользователь"}
          </Badge>
          <Badge variant={user.isActive ? "outline" : "destructive"}>
            {user.isActive ? "Активен" : "Отключен"}
          </Badge>
          {!isCurrentUser && (
            <>
              <Button variant="ghost" size="icon" onClick={onEdit}>
                <Pencil className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" onClick={onResetPassword}>
                <KeyRound className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" onClick={onToggleActive}>
                {user.isActive ? <X className="h-4 w-4" /> : <Save className="h-4 w-4" />}
              </Button>
              <Button variant="ghost" size="icon" onClick={onDelete}>
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function CreateUserDialog({
  open,
  onOpenChange,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}) {
  const [login, setLogin] = useState("")
  const [displayName, setDisplayName] = useState("")
  const [role, setRole] = useState<UserRole>("user")
  const [password, setPassword] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!login.trim() || !displayName.trim() || !password.trim()) return

    setIsLoading(true)
    try {
      await createUser({ login, displayName, role, password })
      toast.success("Пользователь создан")
      setLogin("")
      setDisplayName("")
      setPassword("")
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Не удалось создать пользователя")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Создать пользователя</DialogTitle>
          <DialogDescription>Создайте новую учетную запись для пользователя</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="create-login">Логин</Label>
            <Input id="create-login" value={login} onChange={(e) => setLogin(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="create-displayName">Отображаемое имя</Label>
            <Input id="create-displayName" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>Роль</Label>
            <Select value={role} onValueChange={(v) => setRole(v as UserRole)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">Пользователь</SelectItem>
                <SelectItem value="admin">Администратор</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="create-password">Временный пароль</Label>
            <Input id="create-password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Создать
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function EditUserDialog({
  user,
  onClose,
  onSuccess,
}: {
  user: UserDTO
  onClose: () => void
  onSuccess: () => void
}) {
  const [displayName, setDisplayName] = useState(user.displayName)
  const [role, setRole] = useState<UserRole>(user.role)
  const [isActive, setIsActive] = useState(user.isActive)
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!displayName.trim()) return

    setIsLoading(true)
    try {
      await updateUser(user.id, { displayName, role, isActive })
      toast.success("Пользователь обновлен")
      onSuccess()
      onClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Не удалось обновить пользователя")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={true} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Редактировать пользователя</DialogTitle>
          <DialogDescription>Измените данные учетной записи</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label>Логин</Label>
            <Input value={user.login} disabled />
          </div>
          <div className="space-y-2">
            <Label>Отображаемое имя</Label>
            <Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>Роль</Label>
            <Select value={role} onValueChange={(v) => setRole(v as UserRole)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">Пользователь</SelectItem>
                <SelectItem value="admin">Администратор</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="isActive"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="h-4 w-4"
            />
            <Label htmlFor="isActive">Активен</Label>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Отмена
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Сохранить
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function ResetPasswordDialog({
  user,
  onClose,
}: {
  user: UserDTO
  onClose: () => void
}) {
  const [newPassword, setNewPassword] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword.length < 8) return

    setIsLoading(true)
    try {
      await resetUserPassword(user.id, { newPassword })
      toast.success("Пароль сброшен")
      onClose()
    } catch {
      toast.error("Не удалось сбросить пароль")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={true} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Сброс пароля</DialogTitle>
          <DialogDescription>Сбросьте пароль для пользователя {user.displayName}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="reset-password">Новый пароль</Label>
            <Input
              id="reset-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">Минимум 8 символов</p>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Отмена
            </Button>
            <Button type="submit" disabled={isLoading || newPassword.length < 8}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Сбросить пароль
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
