export interface FileDTO {
  id: number
  path: string
  fileName: string
  dirPath: string
  modTime: string
}

export interface DuplicateGroupDTO {
  index: number
  hash: string
  size: number
  sizeHuman: string
  files: FileDTO[]
  thumbnail: string
}

export interface DuplicatesResponse {
  groups: DuplicateGroupDTO[]
  totalFiles: number
  pageFiles: number
  totalGroups: number
  scannedDirs: string[]
  currentPage: number
  pageSize: number
  totalPages: number
  hasPrevPage: boolean
  hasNextPage: boolean
  pageSizes: number[]
}

export interface ScanResponse {
  message: string
}

export interface ScanStatusResponse {
  scanning: boolean
  progress: string
  filesProcessed: number
}

export interface ThumbnailResponse {
  thumbnail: string
}

export interface DeleteFilesRequest {
  filePaths: string[]
  trashDir: string
}

export interface DeleteFilesResponse {
  success: number
  failed: number
  failedFiles?: string[]
}

export interface FolderPattern {
  id: string
  folders: string[]
  duplicateCount: number
  totalFiles: number
}

export interface FolderPatternsResponse {
  patterns: FolderPattern[]
}

export interface BatchDeleteRule {
  patternId: string
  keepFolder: string
}

export interface BatchDeleteRequest {
  rules: BatchDeleteRule[]
  trashDir: string
}

export interface BatchDeleteResponse {
  success: number
  failed: number
  failedFiles?: string[]
}

export interface ApiError {
  error: string
}

// --- Gallery Folder Types ---

export interface GalleryFolderDTO {
  id: number
  path: string
  fileCount: number
  createdAt: string
}

export interface GalleryFoldersResponse {
  folders: GalleryFolderDTO[]
  totalFolders: number
}

export interface AddFolderRequest {
  path: string
}

export interface AddFolderResponse {
  message: string
  folder: GalleryFolderDTO
  scanStarted: boolean
}

export interface RemoveFolderResponse {
  message: string
  filesRemoved: number
}

// --- Gallery Image Types ---

export interface GalleryImageDTO {
  id: number
  path: string
  fileName: string
  dirPath: string
  size: number
  sizeHuman: string
  modTime: string
  thumbnail?: string
}

export interface GalleryImagesResponse {
  images: GalleryImageDTO[]
  totalImages: number
  currentPage: number
  pageSize: number
  totalPages: number
  hasNextPage: boolean
}

// --- App Settings Types ---

export interface AppSettingsDTO {
  theme: "light" | "dark"
  language: "en" | "ru"
  trashDir: string
}

export interface UserSettingsDTO {
  theme: 
    | "light-purple" 
    | "dark-purple"
    | "light-green"
    | "dark-green"
    | "light-blue"
    | "dark-blue"
    | "light-orange"
    | "dark-orange"
    | "dark-contrast"
  language: "en" | "ru"
  trashDir: string
}

export interface UpdateSettingsRequest {
  theme?: 
    | "light-purple" 
    | "dark-purple"
    | "light-green"
    | "dark-green"
    | "light-blue"
    | "dark-blue"
    | "light-orange"
    | "dark-orange"
    | "dark-contrast"
  language?: "en" | "ru"
  trashDir?: string
}

// --- Trash Types ---

export interface TrashInfoResponse {
  fileCount: number
  totalSize: number
  totalSizeHuman: string
}

export interface CleanTrashResponse {
  deleted: number
  failed: number
}

// --- Image Metadata Types ---

export interface ImageMetadataDTO {
  width: number
  height: number
  dimensions: string
  cameraModel: string
  lensModel: string
  iso: number
  aperture: string
  shutterSpeed: string
  focalLength: string
  dateTaken: string
  orientation: number
  colorSpace: string
  software: string
  gpsLatitude: number | null
  gpsLongitude: number | null
  geoCountry: string
  geoCity: string
  hasGps: boolean
  hasExif: boolean
}

export interface ImageMetadataResponse {
  found: boolean
  metadata?: ImageMetadataDTO
}

// --- Gallery Calendar Types ---

export interface CalendarDateGroup {
  date: string       // "YYYY-MM-DD"
  label: string      // Human-readable label
  imageCount: number
  images: GalleryImageDTO[]
}

export interface CalendarDateRange {
  minDate: string    // "YYYY-MM-DD" or empty
  maxDate: string    // "YYYY-MM-DD" or empty
  totalWithDate: number
}

export interface CalendarMonthInfo {
  year: number
  month: number      // 1-12
  days: number[]     // Days that have images (1-31)
}

export interface GalleryCalendarResponse {
  groups: CalendarDateGroup[]
  totalImages: number
  totalGroups: number
  hasMore: boolean
  dateRange: CalendarDateRange
  months: CalendarMonthInfo[]
}

// --- Auth & User Types ---

export type UserRole = "admin" | "user"

export interface UserDTO {
  id: number
  login: string
  displayName: string
  role: UserRole
  isActive: boolean
  mustChangePassword: boolean
  createdAt: string
  lastLoginAt: string | null
}

export interface AuthStatusResponse {
  isAuthenticated: boolean
  isBootstrapMode: boolean
  user?: UserDTO
}

export interface LoginRequest {
  login: string
  password: string
}

export interface LoginResponse {
  user?: UserDTO
  isBootstrap?: boolean
  message?: string
}

export interface ChangePasswordRequest {
  oldPassword: string
  newPassword: string
}

export interface BootstrapSetupRequest {
  newPassword: string
  displayName: string
}

export interface UpdateProfileRequest {
  displayName: string
}

export interface ChangePasswordResponse {
  message: string
  mustLogin?: boolean
}

export interface CreateUserRequest {
  login: string
  displayName: string
  role: UserRole
  password: string
}

export interface UpdateUserRequest {
  displayName?: string
  role?: UserRole
  isActive?: boolean
}

export interface ResetPasswordRequest {
  newPassword: string
}

export interface UsersListResponse {
  users: UserDTO[]
  total: number
}

export interface AuditLogDTO {
  id: number
  actorUserId: number | null
  action: string
  targetType: string
  targetId: number | null
  meta: string
  createdAt: string
}

export interface AuditLogsResponse {
  logs: AuditLogDTO[]
  total: number
  page: number
}

// --- OCR Status Types ---

export interface OCRStatus {
  enabled: boolean
  health: string
  lastCheck?: string
  error?: string
  serviceUrl?: string
}

export interface OCRStatusResponse {
  status: OCRStatus
}

// --- OCR Classification Types ---

export interface OcrBoundingBoxDTO {
  x: number
  y: number
  width: number
  height: number
  word: string
  confidence: number
}

export interface OcrDocumentDTO {
  id: number
  imageFileId: number
  path: string
  fileName: string
  dirPath: string
  size: number
  sizeHuman: string
  modTime: string
  thumbnail?: string
  meanConfidence: number
  weightedConfidence: number
  tokenCount: number
  angle: number
  scaleFactor: number
}

export interface OcrDocumentsResponse {
  documents: OcrDocumentDTO[]
  total: number
  currentPage: number
  pageSize: number
  totalPages: number
  hasNextPage: boolean
}

export interface OcrDataResponse {
  imagePath: string
  angle: number
  scaleFactor: number
  boxes: OcrBoundingBoxDTO[]
}

export interface OcrClassificationStatusResponse {
  processing: boolean
  progress: string
  filesProcessed: number
  totalFiles: number
}
