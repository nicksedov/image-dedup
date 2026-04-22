import { useCallback } from "react"
import { useTranslation } from "@/i18n"
import { Settings, ImageIcon, FileScan, Shield, Users } from "lucide-react"
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

  const tabs: TabItem[] = [
    { value: "deduplication", icon: FileScan, label: t("tabs.deduplication") },
    { value: "gallery", icon: ImageIcon, label: t("tabs.gallery") },
    { value: "settings", icon: Settings, label: t("tabs.settings") },
    { value: "profile", icon: Shield, label: t("adminPanel.updateProfile") },
  ]

  const adminTabs: TabItem[] = [
    { value: "admin", icon: Users, label: t("adminPanel.title") },
  ]

  const handleTabChange = useCallback(
    (tab: string) => {
      onTabChange(tab)
    },
    [onTabChange]
  )

  const isTabActive = (tabValue: string) => activeTab === tabValue

  return (
    <aside className="flex h-screen w-64 flex-col border-r bg-background">
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
        {tabs.map((tab) => {
          const isActive = isTabActive(tab.value)

          return (
            <Button
              key={tab.value}
              variant={isActive ? "default" : "ghost"}
              className={cn("w-full justify-start gap-3", isActive && "bg-primary text-primary-foreground hover:bg-primary/90")}
              onClick={() => handleTabChange(tab.value)}
            >
              <tab.icon className="h-4 w-4" />
              <span className="font-medium">{tab.label}</span>
            </Button>
          )
        })}
      </nav>

      {/* Admin-only items at the bottom */}
      {user?.role === "admin" && adminTabs.length > 0 && (
        <div className="border-t px-2 py-4 space-y-1">
          {adminTabs.map((tab) => {
            const isActive = isTabActive(tab.value)

            return (
              <Button
                key={tab.value}
                variant={isActive ? "default" : "ghost"}
                className={cn("w-full justify-start gap-3", isActive && "bg-primary text-primary-foreground hover:bg-primary/90")}
                onClick={() => handleTabChange(tab.value)}
              >
                <tab.icon className="h-4 w-4" />
                <span className="font-medium">{tab.label}</span>
              </Button>
            )
          })}
        </div>
      )}
    </aside>
  )
}
