import { apiGet, apiPost, apiDelete, apiPut, apiPatch } from "./client"
import type {
  DuplicatesResponse,
  ScanResponse,
  ScanStatusResponse,
  ThumbnailResponse,
  DeleteFilesRequest,
  DeleteFilesResponse,
  FolderPatternsResponse,
  BatchDeleteRequest,
  BatchDeleteResponse,
  GalleryFoldersResponse,
  AddFolderRequest,
  AddFolderResponse,
  RemoveFolderResponse,
  GalleryImagesResponse,
  AppSettingsDTO,
  UserSettingsDTO,
  UpdateSettingsRequest,
  TrashInfoResponse,
  CleanTrashResponse,
  ImageMetadataResponse,
  AuthStatusResponse,
  LoginRequest,
  LoginResponse,
  ChangePasswordRequest,
  ChangePasswordResponse,
  BootstrapSetupRequest,
  UpdateProfileRequest,
  CreateUserRequest,
  UpdateUserRequest,
  ResetPasswordRequest,
  UsersListResponse,
  AuditLogsResponse,
} from "@/types"

export function fetchDuplicates(page: number, pageSize: number): Promise<DuplicatesResponse> {
  return apiGet<DuplicatesResponse>("/api/duplicates", {
    page: String(page),
    pageSize: String(pageSize),
  })
}

export function triggerScan(): Promise<ScanResponse> {
  return apiPost<ScanResponse>("/api/scan")
}

export function fetchScanStatus(): Promise<ScanStatusResponse> {
  return apiGet<ScanStatusResponse>("/api/status")
}

export function fetchThumbnail(path: string): Promise<ThumbnailResponse> {
  return apiGet<ThumbnailResponse>("/api/thumbnail", { path })
}

export function deleteFiles(req: DeleteFilesRequest): Promise<DeleteFilesResponse> {
  return apiPost<DeleteFilesResponse>("/api/delete-files", req)
}

export function fetchFolderPatterns(): Promise<FolderPatternsResponse> {
  return apiGet<FolderPatternsResponse>("/api/folder-patterns")
}

export function batchDelete(req: BatchDeleteRequest): Promise<BatchDeleteResponse> {
  return apiPost<BatchDeleteResponse>("/api/batch-delete", req)
}

// --- Gallery Folders ---

export function fetchFolders(): Promise<GalleryFoldersResponse> {
  return apiGet<GalleryFoldersResponse>("/api/folders")
}

export function addFolder(req: AddFolderRequest): Promise<AddFolderResponse> {
  return apiPost<AddFolderResponse>("/api/folders", req)
}

export function removeFolder(id: number): Promise<RemoveFolderResponse> {
  return apiDelete<RemoveFolderResponse>(`/api/folders/${id}`)
}

// --- Gallery Images ---

export function fetchGalleryImages(
  page: number,
  pageSize: number,
  view: string
): Promise<GalleryImagesResponse> {
  return apiGet<GalleryImagesResponse>("/api/gallery", {
    page: String(page),
    pageSize: String(pageSize),
    view,
  })
}

// --- App Settings ---

export function fetchSettings(): Promise<AppSettingsDTO> {
  return apiGet<AppSettingsDTO>("/api/settings")
}

export function updateSettings(req: UpdateSettingsRequest): Promise<AppSettingsDTO> {
  return apiPut<AppSettingsDTO>("/api/settings", req)
}

// --- User Settings ---

export function fetchUserSettings(): Promise<UserSettingsDTO> {
  return apiGet<UserSettingsDTO>("/api/user-settings")
}

export function updateUserSettings(req: UpdateSettingsRequest): Promise<UserSettingsDTO> {
  return apiPut<UserSettingsDTO>("/api/user-settings", req)
}

// --- Trash ---

export function fetchTrashInfo(): Promise<TrashInfoResponse> {
  return apiGet<TrashInfoResponse>("/api/trash-info")
}

export function cleanTrash(): Promise<CleanTrashResponse> {
  return apiPost<CleanTrashResponse>("/api/trash-clean")
}

// --- Image Metadata ---

export function fetchImageMetadata(path: string): Promise<ImageMetadataResponse> {
  return apiGet<ImageMetadataResponse>("/api/image-metadata", { path })
}

// --- Auth ---

export function fetchAuthStatus(): Promise<AuthStatusResponse> {
  return apiGet<AuthStatusResponse>("/api/auth/status")
}

export function login(req: LoginRequest): Promise<LoginResponse> {
  return apiPost<LoginResponse>("/api/auth/login", req)
}

export function logout(): Promise<{ message: string }> {
  return apiPost<{ message: string }>("/api/auth/logout")
}

export function fetchCurrentUser(): Promise<{ user: import("@/types").UserDTO }> {
  return apiGet<{ user: import("@/types").UserDTO }>("/api/auth/me")
}

export function changePassword(req: ChangePasswordRequest): Promise<ChangePasswordResponse> {
  return apiPost<ChangePasswordResponse>("/api/auth/change-password", req)
}

export function bootstrapSetup(req: BootstrapSetupRequest): Promise<{ user: import("@/types").UserDTO; message: string }> {
  return apiPost<{ user: import("@/types").UserDTO; message: string }>("/api/auth/bootstrap/setup", req)
}

export function updateProfile(req: UpdateProfileRequest): Promise<{ user: import("@/types").UserDTO }> {
  return apiPatch<{ user: import("@/types").UserDTO }>("/api/users/me", req)
}

// --- Admin ---

export function fetchUsers(): Promise<UsersListResponse> {
  return apiGet<UsersListResponse>("/api/admin/users")
}

export function createUser(req: CreateUserRequest): Promise<{ user: import("@/types").UserDTO; message: string }> {
  return apiPost<{ user: import("@/types").UserDTO; message: string }>("/api/admin/users", req)
}

export function updateUser(id: number, req: UpdateUserRequest): Promise<{ user: import("@/types").UserDTO; message: string }> {
  return apiPatch<{ user: import("@/types").UserDTO; message: string }>(`/api/admin/users/${id}`, req)
}

export function deleteUser(id: number): Promise<{ message: string }> {
  return apiDelete<{ message: string }>(`/api/admin/users/${id}`)
}

export function resetUserPassword(id: number, req: ResetPasswordRequest): Promise<{ message: string }> {
  return apiPost<{ message: string }>(`/api/admin/users/${id}/reset-password`, req)
}

export function fetchAuditLogs(page: number): Promise<AuditLogsResponse> {
  return apiGet<AuditLogsResponse>("/api/admin/audit", { page: String(page) })
}
