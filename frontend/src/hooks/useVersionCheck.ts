import { useState, useEffect, useCallback, useRef } from 'react';

// Update result type matching the Go backend structure
export interface UpdateResult {
  has_update: boolean;
  current_version: string;
  latest_version: string;
  release_info?: {
    tag_name: string;
    name: string;
    body: string;
    html_url: string;
    published_at: string;
  };
  download_url?: string;
  release_notes?: string;
  published_at?: string;
  checked_at?: string;
  skipped?: boolean;
  skip_reason?: string;
}

// Legacy interface for backward compatibility
export interface VersionCheckResult {
  currentVersion: string;
  latestVersion: string;
  hasUpdate: boolean;
  releaseNotes: string;
  downloadUrl: string;
  publishedAt: string;
  error?: string;
}

export interface VersionCheckState {
  // State
  updateResult: UpdateResult | null;
  versionInfo: VersionCheckResult | null;
  isChecking: boolean;
  error: string | null;
  isDismissed: boolean;
  updateAvailable: boolean;
  
  // Actions
  checkForUpdates: () => Promise<void>;
  dismissUpdate: () => void;
  openReleasePage: () => void;
  resetDismissal: () => void;
}

// Export alias for compatibility with existing code
export type UseVersionCheckReturn = VersionCheckState;

// Storage keys
const STORAGE_KEY_DISMISSED = 'gline_update_dismissed';
const STORAGE_KEY_DISMISSED_VERSION = 'gline_update_dismissed_version';
const STORAGE_KEY_LAST_CHECK = 'gline_update_last_check';

// GitHub API URL for releases
const GITHUB_RELEASES_API = 'https://api.github.com/repos/liup215/gline/releases/latest';
const GITHUB_RELEASES_URL = 'https://github.com/liup215/gline/releases/latest';

// Current app version - will be replaced during build
const APP_VERSION = 'dev'; // This should match the version injected at build time

/**
 * Convert new UpdateResult to legacy VersionCheckResult format
 */
function toVersionCheckResult(result: UpdateResult | null): VersionCheckResult | null {
  if (!result) return null;
  return {
    currentVersion: result.current_version,
    latestVersion: result.latest_version,
    hasUpdate: result.has_update,
    releaseNotes: result.release_notes || result.release_info?.body || '',
    downloadUrl: result.download_url || result.release_info?.html_url || GITHUB_RELEASES_URL,
    publishedAt: result.published_at || result.release_info?.published_at || '',
  };
}

/**
 * Hook for checking app updates
 * Automatically checks on mount if enough time has passed since last check
 */
export function useVersionCheck(autoCheck: boolean = true): VersionCheckState {
  const [updateResult, setUpdateResult] = useState<UpdateResult | null>(null);
  const [isChecking, setIsChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isDismissed, setIsDismissed] = useState(false);
  const checkInProgress = useRef(false);

  // Load dismissed state from localStorage on mount
  useEffect(() => {
    try {
      const dismissedVersion = localStorage.getItem(STORAGE_KEY_DISMISSED_VERSION);
      const dismissedTime = localStorage.getItem(STORAGE_KEY_DISMISSED);
      
      if (dismissedVersion && dismissedTime) {
        // Only consider dismissed if it's for the same version
        const parsedTime = parseInt(dismissedTime, 10);
        const oneDay = 24 * 60 * 60 * 1000;
        
        // Reset dismissal after 24 hours or if checking different version
        if (Date.now() - parsedTime < oneDay) {
          setIsDismissed(true);
        } else {
          // Clear old dismissal
          localStorage.removeItem(STORAGE_KEY_DISMISSED);
          localStorage.removeItem(STORAGE_KEY_DISMISSED_VERSION);
        }
      }
    } catch {
      // Ignore storage errors
    }
  }, []);

  /**
   * Parse semantic version string (e.g., "v1.2.3" or "1.2.3")
   */
  const parseVersion = (version: string): { major: number; minor: number; patch: number } => {
    const clean = version.replace(/^v/, '');
    const parts = clean.split('.').map(Number);
    return {
      major: parts[0] || 0,
      minor: parts[1] || 0,
      patch: parts[2] || 0,
    };
  };

  /**
   * Compare two version strings
   * Returns: -1 if v1 < v2, 0 if equal, 1 if v1 > v2
   */
  const compareVersions = (v1: string, v2: string): number => {
    const p1 = parseVersion(v1);
    const p2 = parseVersion(v2);
    
    if (p1.major !== p2.major) return p1.major > p2.major ? 1 : -1;
    if (p1.minor !== p2.minor) return p1.minor > p2.minor ? 1 : -1;
    if (p1.patch !== p2.patch) return p1.patch > p2.patch ? 1 : -1;
    return 0;
  };

  /**
   * Check for updates from GitHub releases
   */
  const checkForUpdates = useCallback(async (): Promise<void> => {
    if (checkInProgress.current) return;
    checkInProgress.current = true;
    
    setIsChecking(true);
    setError(null);
    
    try {
      // Fetch latest release from GitHub API
      const response = await fetch(GITHUB_RELEASES_API, {
        headers: {
          'Accept': 'application/vnd.github.v3+json',
        },
      });
      
      if (!response.ok) {
        if (response.status === 403 || response.status === 429) {
          // Rate limited
          setUpdateResult({
            has_update: false,
            current_version: APP_VERSION,
            latest_version: '',
            skipped: true,
            skip_reason: 'Rate limited by GitHub API. Please try again later.',
          });
          return;
        }
        throw new Error(`GitHub API error: ${response.status}`);
      }
      
      const release = await response.json();
      const latestVersion = release.tag_name || release.name || '';
      
      if (!latestVersion) {
        throw new Error('Invalid release data');
      }
      
      // Compare versions
      const hasUpdate = compareVersions(latestVersion, APP_VERSION) > 0;
      
      const result: UpdateResult = {
        has_update: hasUpdate,
        current_version: APP_VERSION,
        latest_version: latestVersion,
        release_info: {
          tag_name: release.tag_name,
          name: release.name,
          body: release.body,
          html_url: release.html_url,
          published_at: release.published_at,
        },
        download_url: release.html_url,
        release_notes: release.body?.substring(0, 500) + (release.body?.length > 500 ? '...' : ''),
        published_at: release.published_at,
        checked_at: new Date().toISOString(),
      };
      
      setUpdateResult(result);
      
      // Store last check time
      try {
        localStorage.setItem(STORAGE_KEY_LAST_CHECK, Date.now().toString());
      } catch {
        // Ignore storage errors
      }
      
      // Reset dismissal if new version is available
      if (hasUpdate) {
        const dismissedVersion = localStorage.getItem(STORAGE_KEY_DISMISSED_VERSION);
        if (dismissedVersion !== latestVersion) {
          setIsDismissed(false);
          localStorage.removeItem(STORAGE_KEY_DISMISSED);
          localStorage.removeItem(STORAGE_KEY_DISMISSED_VERSION);
        }
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to check for updates';
      setError(errorMessage);
      console.error('Update check failed:', err);
    } finally {
      setIsChecking(false);
      checkInProgress.current = false;
    }
  }, []);

  /**
   * Dismiss the update notification
   */
  const dismissUpdate = useCallback(() => {
    setIsDismissed(true);
    try {
      localStorage.setItem(STORAGE_KEY_DISMISSED, Date.now().toString());
      if (updateResult?.latest_version) {
        localStorage.setItem(STORAGE_KEY_DISMISSED_VERSION, updateResult.latest_version);
      }
    } catch {
      // Ignore storage errors
    }
  }, [updateResult?.latest_version]);

  /**
   * Open the release page in browser
   */
  const openReleasePage = useCallback(() => {
    const url = updateResult?.release_info?.html_url || GITHUB_RELEASES_URL;
    
    // Try to use Wails BrowserOpenURL if available
    if (typeof window !== 'undefined' && (window as any).go?.main?.App?.BrowserOpenURL) {
      (window as any).go.main.App.BrowserOpenURL(url);
    } else {
      // Fallback: open in default browser
      window.open(url, '_blank');
    }
  }, [updateResult?.release_info?.html_url]);

  /**
   * Reset dismissal state (for testing or manual check)
   */
  const resetDismissal = useCallback(() => {
    setIsDismissed(false);
    try {
      localStorage.removeItem(STORAGE_KEY_DISMISSED);
      localStorage.removeItem(STORAGE_KEY_DISMISSED_VERSION);
    } catch {
      // Ignore storage errors
    }
  }, []);

  // Auto-check on mount if enabled
  useEffect(() => {
    if (!autoCheck) return;
    
    // Check if we should auto-check (respect check interval)
    try {
      const lastCheck = localStorage.getItem(STORAGE_KEY_LAST_CHECK);
      const checkInterval = 24 * 60 * 60 * 1000; // 24 hours
      
      if (!lastCheck || Date.now() - parseInt(lastCheck, 10) > checkInterval) {
        // Delay auto-check to not block app startup
        const timer = setTimeout(() => {
          checkForUpdates();
        }, 5000);
        
        return () => clearTimeout(timer);
      }
    } catch {
      // If storage fails, still try to check
      const timer = setTimeout(() => {
        checkForUpdates();
      }, 5000);
      
      return () => clearTimeout(timer);
    }
  }, [autoCheck, checkForUpdates]);

  // Compute derived values
  const versionInfo = toVersionCheckResult(updateResult);
  const updateAvailable = updateResult?.has_update ?? false;

  return {
    updateResult,
    versionInfo,
    isChecking,
    error,
    isDismissed,
    updateAvailable,
    checkForUpdates,
    dismissUpdate,
    openReleasePage,
    resetDismissal,
  };
}

export default useVersionCheck;
