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
import { useTranslation } from "@/i18n"
import { translateApiMessage } from "@/api/client"

export function AdminPanel() {
  const { user: currentUser } = useAuth()
  const { t } = useTranslation()
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
      toast.error(t("adminPanel.toastUsersLoadFailed"))
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
        <p className="text-muted-foreground">{t("adminPanel.accessDenied")}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">{t("adminPanel.title")}</h2>
          <p className="text-muted-foreground">{t("adminPanel.description")}</p>
        </div>
        <Button onClick={() => setIsCreateOpen(true)}>
          <UserPlus className="mr-2 h-4 w-4" />
          {t("adminPanel.createButton")}
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
            <p className="text-lg font-medium">{t("adminPanel.noUsers")}</p>
            <p className="text-sm text-muted-foreground">{t("adminPanel.noUsersHint")}</p>
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
                if (!confirm(t("adminPanel.deleteConfirm", { displayName: u.displayName }))) return
                try {
                  await deleteUser(u.id)
                  toast.success(t("adminPanel.deleteSuccess"))
                  loadUsers()
                } catch {
                  toast.error(t("adminPanel.deleteFailed"))
                }
              }}
              onToggleActive={async () => {
                try {
                  await updateUser(u.id, { isActive: !u.isActive })
                  toast.success(u.isActive ? t("adminPanel.deactivateSuccess") : t("adminPanel.activateSuccess"))
                  loadUsers()
                } catch {
                  toast.error(t("adminPanel.updateFailed"))
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
  const { t } = useTranslation()
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
              {isCurrentUser && <span className="ml-2 text-xs text-muted-foreground">{t("adminPanel.you")}</span>}
            </p>
            <p className="text-sm text-muted-foreground">{user.login}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={user.role === "admin" ? "default" : "secondary"}>
            {user.role === "admin" ? t("adminPanel.roleAdmin") : t("adminPanel.roleUser")}
          </Badge>
          <Badge variant={user.isActive ? "outline" : "destructive"}>
            {user.isActive ? t("adminPanel.statusActive") : t("adminPanel.statusDisabled")}
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
  const { t } = useTranslation()
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
      toast.success(t("adminPanel.deleteSuccess"))
      setLogin("")
      setDisplayName("")
      setPassword("")
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      const errorMessage = err instanceof Error ? translateApiMessage(err.message) : t("adminPanel.create")
      toast.error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("adminPanel.createUserTitle")}</DialogTitle>
          <DialogDescription>{t("adminPanel.createUserDesc")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="create-login">{t("adminPanel.login")}</Label>
            <Input id="create-login" value={login} onChange={(e) => setLogin(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="create-displayName">{t("adminPanel.displayName")}</Label>
            <Input id="create-displayName" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>{t("adminPanel.role")}</Label>
            <Select value={role} onValueChange={(v) => setRole(v as UserRole)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">{t("adminPanel.roleUser")}</SelectItem>
                <SelectItem value="admin">{t("adminPanel.adminRole")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="create-password">{t("adminPanel.tempPassword")}</Label>
            <Input id="create-password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("adminPanel.cancel")}
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t("adminPanel.create")}
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
  const { t } = useTranslation()
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
      toast.success(t("adminPanel.profileUpdated"))
      onSuccess()
      onClose()
    } catch (err) {
      const errorMessage = err instanceof Error ? translateApiMessage(err.message) : t("adminPanel.updateFailed")
      toast.error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={true} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("adminPanel.editUserTitle")}</DialogTitle>
          <DialogDescription>{t("adminPanel.editUserDesc")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label>{t("adminPanel.login")}</Label>
            <Input value={user.login} disabled />
          </div>
          <div className="space-y-2">
            <Label>{t("adminPanel.displayName")}</Label>
            <Input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>{t("adminPanel.role")}</Label>
            <Select value={role} onValueChange={(v) => setRole(v as UserRole)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">{t("adminPanel.roleUser")}</SelectItem>
                <SelectItem value="admin">{t("adminPanel.adminRole")}</SelectItem>
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
            <Label htmlFor="isActive">{t("adminPanel.statusActive")}</Label>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t("adminPanel.cancel")}
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t("adminPanel.save")}
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
  const { t } = useTranslation()
  const [newPassword, setNewPassword] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword.length < 8) return

    setIsLoading(true)
    try {
      await resetUserPassword(user.id, { newPassword })
      toast.success(t("adminPanel.resetPasswordSuccess"))
      onClose()
    } catch (err) {
      const errorMessage = err instanceof Error ? translateApiMessage(err.message) : t("adminPanel.resetPasswordFailed")
      toast.error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={true} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("adminPanel.resetPasswordTitle")}</DialogTitle>
          <DialogDescription>{t("adminPanel.resetPasswordDesc", { displayName: user.displayName })}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="reset-password">{t("adminPanel.newPassword")}</Label>
            <Input
              id="reset-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">{t("adminPanel.minPasswordLength")}</p>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t("adminPanel.cancel")}
            </Button>
            <Button type="submit" disabled={isLoading || newPassword.length < 8}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t("adminPanel.resetPassword")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
