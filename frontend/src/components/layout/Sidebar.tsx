import { useCallback, useState } from "react"
import { useTranslation } from "@/i18n"
import { Settings, ImageIcon, FileScan, Shield, Users, ChevronDown, ChevronRight, Folder, Calendar } from "lucide-react"
import { useAuth } from "@/providers/AuthProvider"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface SidebarProps {
  activeTab: string
  onTabChange: (tab: string) => void
}

type TabItem = {
  value: string
  icon: React.ElementType
  label: string
  showForAdminOnly?: boolean
}

export function Sidebar({ activeTab, onTabChange }: SidebarProps) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [galleryExpanded, setGalleryExpanded] = useState(true)
  const [adminExpanded, setAdminExpanded] = useState(false)

  const tabs: TabItem[] = [
    { value: "settings", icon: Settings, label: t("tabs.preferences") },
    { value: "deduplication", icon: FileScan, label: t("tabs.deduplication") },
    { value: "profile", icon: Shield, label: t("adminPanel.updateProfile") },
  ]

  const adminTabs: TabItem[] = [
    { value: "admin-users", icon: Users, label: t("adminPanel.title") },
    { value: "admin-settings", icon: Settings, label: t("adminPanel.adminSettings") },
  ]

  const gallerySubModes = [
    { value: "gallery-folders", icon: Folder, label: t("gallery.subModes.folders") },
    { value: "gallery-calendar", icon: Calendar, label: t("gallery.subModes.calendar") },
  ]

  const isGalleryActive = activeTab.startsWith("gallery")
  const isAdminActive = activeTab === "admin-users" || activeTab === "admin-settings"

  const handleGalleryModeChange = useCallback(
    (mode: string) => {
      onTabChange(mode)
    },
    [onTabChange]
  )

  const handleTabChange = useCallback(
    (tab: string) => {
      onTabChange(tab)
    },
    [onTabChange]
  )

  const isTabActive = (tabValue: string) => activeTab === tabValue

  return (
    <aside className="sticky top-0 flex h-screen w-64 flex-shrink-0 flex-col border-r bg-sidebar">
      <div className="flex h-16 items-center px-4 border-b">
        <div className="flex items-center gap-3 overflow-hidden">
          <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <ImageIcon className="h-5 w-5" />
          </div>
          <div className="overflow-hidden">
            <h1 className="truncate font-semibold text-lg">{t("header.title")}</h1>
            <p className="truncate text-xs text-muted-foreground">{t("header.subtitle")}</p>
          </div>
        </div>
      </div>

      <nav className="flex-1 px-2 py-4 space-y-1 overflow-y-auto">
        {/* Gallery group with sub-items */}
        <div className="space-y-1">
          <Button
            variant={isGalleryActive ? "default" : "ghost"}
            className={cn("w-full justify-start gap-3", isGalleryActive && "bg-primary text-primary-foreground hover:bg-primary/90")}
            onClick={() => {
              setGalleryExpanded(!galleryExpanded)
              if (!isGalleryActive) {
                handleGalleryModeChange("gallery-folders")
              }
            }}
          >
            <ImageIcon className="h-4 w-4 flex-shrink-0" />
            <span className="flex-1 font-medium text-left">{t("tabs.gallery")}</span>
            {galleryExpanded ? (
              <ChevronDown className="h-3 w-3 flex-shrink-0" />
            ) : (
              <ChevronRight className="h-3 w-3 flex-shrink-0" />
            )}
          </Button>

          {galleryExpanded && (
            <div className="ml-6 space-y-0.5 border-l pl-2">
              {gallerySubModes.map((subMode) => {
                const isActive = isTabActive(subMode.value)
                return (
                  <Button
                    key={subMode.value}
                    variant={isActive ? "default" : "ghost"}
                    size="sm"
                    className={cn("w-full justify-start gap-2 h-8", isActive && "bg-primary text-primary-foreground hover:bg-primary/90")}
                    onClick={() => handleGalleryModeChange(subMode.value)}
                  >
                    <subMode.icon className="h-3.5 w-3.5 flex-shrink-0" />
                    <span className="text-sm font-medium">{subMode.label}</span>
                  </Button>
                )
              })}
            </div>
          )}
        </div>

        {/* Other tabs */}
        {tabs.map((tab) => {
          const isActive = isTabActive(tab.value)

          return (
            <Button
              key={tab.value}
              variant={isActive ? "default" : "ghost"}
              className={cn("w-full justify-start gap-3", isActive && "bg-primary text-primary-foreground hover:bg-primary/90")}
              onClick={() => handleTabChange(tab.value)}
            >
              <tab.icon className="h-4 w-4 flex-shrink-0" />
              <span className="font-medium">{tab.label}</span>
            </Button>
          )
        })}

        {/* Admin administration group */}
        {user?.role === "admin" && (
          <div className="mt-4">
            <Button
              variant={isAdminActive ? "default" : "ghost"}
              className={cn("w-full justify-start gap-3", isAdminActive && "bg-primary text-primary-foreground hover:bg-primary/90")}
              onClick={() => {
                setAdminExpanded(!adminExpanded)
                if (!isAdminActive) {
                  handleTabChange("admin-settings")
                }
              }}
            >
              <Shield className="h-4 w-4 flex-shrink-0" />
              <span className="flex-1 font-medium text-left">{t("adminPanel.administration")}</span>
              {adminExpanded ? (
                <ChevronDown className="h-3 w-3 flex-shrink-0" />
              ) : (
                <ChevronRight className="h-3 w-3 flex-shrink-0" />
              )}
            </Button>

            {adminExpanded && (
              <div className="ml-6 space-y-0.5 border-l pl-2">
                {adminTabs.map((tab) => {
                  const isActive = isTabActive(tab.value)
                  return (
                    <Button
                      key={tab.value}
                      variant={isActive ? "default" : "ghost"}
                      size="sm"
                      className={cn("w-full justify-start gap-2 h-8", isActive && "bg-primary text-primary-foreground hover:bg-primary/90")}
                      onClick={() => handleTabChange(tab.value)}
                    >
                      <tab.icon className="h-3.5 w-3.5 flex-shrink-0" />
                      <span className="text-sm font-medium">{tab.label}</span>
                    </Button>
                  )
                })}
              </div>
            )}
          </div>
        )}
      </nav>
    </aside>
  )
}
