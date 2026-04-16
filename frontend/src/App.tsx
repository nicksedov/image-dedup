import { useCallback, useEffect, useState } from "react"
import { Toaster } from "sonner"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Header } from "@/components/layout/Header"
import { SettingsTab } from "@/components/tabs/SettingsTab"
import { GalleryTab } from "@/components/tabs/GalleryTab"
import { DeduplicationTab } from "@/components/tabs/DeduplicationTab"
import { fetchFolders } from "@/api/endpoints"
import { Settings, ImageIcon, FileScan, Shield, Users } from "lucide-react"
import { useTranslation } from "@/i18n"
import { useSettings } from "@/providers/useSettings"
import { useAuth } from "@/providers/AuthProvider"
import { LoginScreen } from "@/components/auth/LoginScreen"
import { UserProfile } from "@/components/auth/UserProfile"
import { AdminPanel } from "@/components/auth/AdminPanel"

type TabValue = "settings" | "gallery" | "deduplication" | "profile" | "admin"

export default function App() {
  const [activeTab, setActiveTab] = useState<TabValue>("deduplication")
  const [isCheckingGallery, setIsCheckingGallery] = useState(true)
  const { t } = useTranslation()
  const { isLoading: isLoadingSettings } = useSettings()
  const { user, isAuthenticated, isBootstrapMode, isLoading: isLoadingAuth } = useAuth()

  // On mount, check if gallery has folders. If not, force settings tab.
  useEffect(() => {
    async function checkGallery() {
      try {
        const result = await fetchFolders()
        if (result.totalFolders === 0) {
          setActiveTab("settings")
        }
      } catch {
        // If API fails, still allow normal navigation
      } finally {
        setIsCheckingGallery(false)
      }
    }
    if (isAuthenticated) {
      checkGallery()
    } else {
      setIsCheckingGallery(false)
    }
  }, [isAuthenticated])

  const handleFolderAdded = useCallback(() => {
    // Gallery is no longer empty -- user can now switch tabs freely
  }, [])

  // Loading state
  if (isLoadingAuth || (isAuthenticated && (isCheckingGallery || isLoadingSettings))) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="text-center">
          <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <p className="text-sm text-muted-foreground">Загрузка...</p>
        </div>
      </div>
    )
  }

  // Not authenticated - show login
  if (!isAuthenticated) {
    if (isBootstrapMode) {
      return (
        <>
          <LoginScreen />
          <Toaster richColors position="top-right" />
        </>
      )
    }
    return (
      <>
        <LoginScreen />
        <Toaster richColors position="top-right" />
      </>
    )
  }

  // Bootstrap setup redirect (handled by backend redirect, but we can also show a message)
  // This case should typically be handled via the bootstrap login flow

  return (
    <div className="min-h-screen bg-background">
      <Header />

      <main className="mx-auto max-w-7xl px-4 py-4 sm:px-6">
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as TabValue)}>
          <TabsList className="mb-4">
            <TabsTrigger value="deduplication" className="gap-1.5">
              <FileScan className="h-3.5 w-3.5" />
              {t("tabs.deduplication")}
            </TabsTrigger>
            <TabsTrigger value="gallery" className="gap-1.5">
              <ImageIcon className="h-3.5 w-3.5" />
              {t("tabs.gallery")}
            </TabsTrigger>
            <TabsTrigger value="settings" className="gap-1.5">
              <Settings className="h-3.5 w-3.5" />
              {t("tabs.settings")}
            </TabsTrigger>
            {user?.role === "admin" && (
              <TabsTrigger value="admin" className="gap-1.5">
                <Users className="h-3.5 w-3.5" />
                Админпанель
              </TabsTrigger>
            )}
            <TabsTrigger value="profile" className="gap-1.5">
              <Shield className="h-3.5 w-3.5" />
              Профиль
            </TabsTrigger>
          </TabsList>

          <TabsContent value="settings">
            <SettingsTab onFolderAdded={handleFolderAdded} />
          </TabsContent>

          <TabsContent value="gallery">
            <GalleryTab />
          </TabsContent>

          <TabsContent value="deduplication">
            <DeduplicationTab />
          </TabsContent>

          <TabsContent value="admin">
            {user?.role === "admin" ? <AdminPanel /> : (
              <div className="flex items-center justify-center py-20">
                <p className="text-muted-foreground">Недостаточно прав</p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="profile">
            <UserProfile />
          </TabsContent>
        </Tabs>
      </main>

      <Toaster richColors position="top-right" />
    </div>
  )
}
