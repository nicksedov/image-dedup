import { useCallback, useEffect, useState } from "react"
import { Toaster } from "sonner"
import { Tabs, TabsContent } from "@/components/ui/tabs"
import { Sidebar } from "@/components/layout/Sidebar"
import { SettingsTab } from "@/components/tabs/SettingsTab"
import { GalleryTab } from "@/components/tabs/GalleryTab"
import { DeduplicationTab } from "@/components/tabs/DeduplicationTab"
import { fetchFolders } from "@/api/endpoints"
import { useTranslation } from "@/i18n"
import { useSettings } from "@/providers/useSettings"
import { useAuth } from "@/providers/AuthProvider"
import { LoginScreen } from "@/components/auth/LoginScreen"
import { BootstrapSetupScreen } from "@/components/auth/BootstrapSetupScreen"
import { UserProfile } from "@/components/auth/UserProfile"
import { AdminPanel } from "@/components/auth/AdminPanel"

type TabValue = "settings" | "gallery" | "deduplication" | "profile" | "admin"

export default function App() {
  const [activeTab, setActiveTab] = useState<TabValue>("deduplication")
  const [isCheckingGallery, setIsCheckingGallery] = useState(true)
  const [forceLogout, setForceLogout] = useState(false)
  const { t } = useTranslation()
  const { isLoading: isLoadingSettings } = useSettings()
  const { user, isAuthenticated, isBootstrapMode, isBootstrapVerified, isLoading: isLoadingAuth } = useAuth()

  // Listen for navigate-to-login event
  useEffect(() => {
    const handleNavigateToLogin = () => {
      setForceLogout(true)
    }
    window.addEventListener("navigate-to-login", handleNavigateToLogin as EventListener)
    return () => {
      window.removeEventListener("navigate-to-login", handleNavigateToLogin as EventListener)
    }
  }, [])

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
          <p className="text-sm text-muted-foreground">{t("common.loading")}</p>
        </div>
      </div>
    )
  }

  // Not authenticated or forced logout - show login or bootstrap setup
  if (!isAuthenticated || forceLogout) {
    if (isBootstrapMode && isBootstrapVerified) {
      return (
        <>
          <BootstrapSetupScreen />
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

  const handleTabChange = (tab: string) => {
    setActiveTab(tab as TabValue)
  }

  return (
    <div className="flex min-h-screen bg-background">
      <Sidebar activeTab={activeTab} onTabChange={handleTabChange} />

      <div className="flex flex-1 flex-col overflow-hidden">
        <main className="flex-1 overflow-auto px-8 py-6">
          <div className="mx-auto max-w-7xl">
            <Tabs value={activeTab} onValueChange={(v) => handleTabChange(v)}>
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
                    <p className="text-muted-foreground">{t("adminPanel.accessDenied")}</p>
                  </div>
                )}
              </TabsContent>

              <TabsContent value="profile">
                <UserProfile />
              </TabsContent>
            </Tabs>
          </div>
        </main>
      </div>

      <Toaster richColors position="top-right" />
    </div>
  )
}
