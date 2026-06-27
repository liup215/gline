import { THEME } from '../theme';
import type { UpdateResult } from '../hooks/useVersionCheck';

interface UpdateNotificationProps {
  updateResult: UpdateResult;
  onDismiss: () => void;
  onDownload: () => void;
  isChecking?: boolean;
}

/**
 * UpdateNotification - A non-intrusive banner for version updates
 * 
 * Displays at the top of the app when a new version is available.
 * Supports both dark and light themes via CSS variables.
 */
export function UpdateNotification({
  updateResult,
  onDismiss,
  onDownload,
  isChecking = false,
}: UpdateNotificationProps) {
  const formatDate = (dateString: string): string => {
    try {
      const date = new Date(dateString);
      return date.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return dateString;
    }
  };

  // Truncate release notes to first paragraph or 200 chars
  const truncateReleaseNotes = (notes: string): string => {
    if (!notes) return '';
    const firstParagraph = notes.split('\n\n')[0];
    if (firstParagraph.length > 200) {
      return firstParagraph.substring(0, 200) + '...';
    }
    return firstParagraph;
  };

  const currentVersion = updateResult.current_version || 'dev';
  const latestVersion = updateResult.latest_version || '';
  const releaseNotes = updateResult.release_info?.body || updateResult.release_notes || '';
  const publishedAt = updateResult.release_info?.published_at || updateResult.published_at || '';

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        zIndex: 9999,
        background: `linear-gradient(135deg, ${THEME.accent}dd, ${THEME.accent})`,
        color: '#ffffff',
        padding: '12px 20px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '16px',
        boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)',
        animation: 'slideDown 0.3s ease-out',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      }}
    >
      {/* Icon */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: '32px',
          height: '32px',
          borderRadius: '50%',
          background: 'rgba(255, 255, 255, 0.2)',
          flexShrink: 0,
        }}
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M12 2v20M2 12h20M12 2l4 4M12 2l-4 4M12 22l4-4M12 22l-4-4M22 12l-4 4M22 12l-4-4M2 12l4 4M2 12l4-4" />
        </svg>
      </div>

      {/* Content */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '4px',
          flex: 1,
          minWidth: 0,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            flexWrap: 'wrap',
          }}
        >
          <span
            style={{
              fontWeight: 600,
              fontSize: '14px',
            }}
          >
            Update Available
          </span>
          <span
            style={{
              fontSize: '12px',
              opacity: 0.9,
              background: 'rgba(255, 255, 255, 0.2)',
              padding: '2px 8px',
              borderRadius: '12px',
            }}
          >
            {currentVersion} → {latestVersion}
          </span>
        </div>
        
        {releaseNotes && (
          <span
            style={{
              fontSize: '12px',
              opacity: 0.85,
              lineHeight: 1.4,
              maxWidth: '600px',
            }}
          >
            {truncateReleaseNotes(releaseNotes)}
          </span>
        )}
        
        {publishedAt && (
          <span
            style={{
              fontSize: '11px',
              opacity: 0.7,
            }}
          >
            Released {formatDate(publishedAt)}
          </span>
        )}
      </div>

      {/* Actions */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          flexShrink: 0,
        }}
      >
        <button
          onClick={onDownload}
          disabled={isChecking}
          style={{
            padding: '8px 16px',
            borderRadius: '6px',
            border: 'none',
            background: '#ffffff',
            color: THEME.accent,
            fontSize: '13px',
            fontWeight: 600,
            cursor: isChecking ? 'not-allowed' : 'pointer',
            opacity: isChecking ? 0.6 : 1,
            transition: 'all 0.2s ease',
            whiteSpace: 'nowrap',
          }}
          onMouseEnter={(e) => {
            if (!isChecking) {
              e.currentTarget.style.background = '#f0f0f0';
              e.currentTarget.style.transform = 'translateY(-1px)';
            }
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = '#ffffff';
            e.currentTarget.style.transform = 'translateY(0)';
          }}
        >
          {isChecking ? 'Checking...' : 'Download'}
        </button>
        
        <button
          onClick={onDismiss}
          style={{
            padding: '8px 12px',
            borderRadius: '6px',
            border: '1px solid rgba(255, 255, 255, 0.3)',
            background: 'transparent',
            color: '#ffffff',
            fontSize: '13px',
            cursor: 'pointer',
            opacity: 0.9,
            transition: 'all 0.2s ease',
            whiteSpace: 'nowrap',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = 'rgba(255, 255, 255, 0.1)';
            e.currentTarget.style.opacity = '1';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = 'transparent';
            e.currentTarget.style.opacity = '0.9';
          }}
        >
          Dismiss
        </button>
      </div>

      {/* Animation styles */}
      <style>{`
        @keyframes slideDown {
          from {
            transform: translateY(-100%);
            opacity: 0;
          }
          to {
            transform: translateY(0);
            opacity: 1;
          }
        }
      `}</style>
    </div>
  );
}

/**
 * Compact version of the update notification for use in settings or menus
 */
export function UpdateNotificationCompact({
  updateResult,
  onDismiss,
  onDownload,
  isChecking = false,
}: UpdateNotificationProps) {
  const latestVersion = updateResult.latest_version || '';
  const currentVersion = updateResult.current_version || 'dev';

  return (
    <div
      style={{
        background: THEME.cardBg,
        border: `1px solid ${THEME.border}`,
        borderRadius: '8px',
        padding: '16px',
        display: 'flex',
        alignItems: 'flex-start',
        gap: '12px',
        marginBottom: '16px',
      }}
    >
      {/* Icon */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: '36px',
          height: '36px',
          borderRadius: '50%',
          background: `${THEME.accent}20`,
          color: THEME.accent,
          flexShrink: 0,
        }}
      >
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M12 2v20M2 12h20M12 2l4 4M12 2l-4 4M12 22l4-4M12 22l-4-4M22 12l-4 4M22 12l-4-4M2 12l4 4M2 12l4-4" />
        </svg>
      </div>

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            marginBottom: '4px',
          }}
        >
          <span
            style={{
              fontWeight: 600,
              fontSize: '14px',
              color: THEME.text,
            }}
          >
            Update Available
          </span>
          <span
            style={{
              fontSize: '11px',
              color: THEME.textMuted,
              background: THEME.bgSidebar,
              padding: '2px 8px',
              borderRadius: '12px',
            }}
          >
            {latestVersion}
          </span>
        </div>
        
        <p
          style={{
            fontSize: '13px',
            color: THEME.textMuted,
            margin: '0 0 12px 0',
            lineHeight: 1.4,
          }}
        >
          A new version of gline is available. You're currently running{' '}
          {currentVersion}.
        </p>

        {/* Actions */}
        <div style={{ display: 'flex', gap: '8px' }}>
          <button
            onClick={onDownload}
            disabled={isChecking}
            style={{
              padding: '6px 12px',
              borderRadius: '6px',
              border: 'none',
              background: THEME.accent,
              color: '#ffffff',
              fontSize: '12px',
              fontWeight: 500,
              cursor: isChecking ? 'not-allowed' : 'pointer',
              opacity: isChecking ? 0.6 : 1,
            }}
          >
            {isChecking ? 'Checking...' : 'Download Update'}
          </button>
          
          <button
            onClick={onDismiss}
            style={{
              padding: '6px 12px',
              borderRadius: '6px',
              border: `1px solid ${THEME.border}`,
              background: 'transparent',
              color: THEME.textMuted,
              fontSize: '12px',
              cursor: 'pointer',
            }}
          >
            Dismiss
          </button>
        </div>
      </div>
    </div>
  );
}

/**
 * Loading state for version check
 */
export function UpdateCheckLoading() {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '8px 12px',
        color: THEME.textMuted,
        fontSize: '13px',
      }}
    >
      <div
        style={{
          width: '14px',
          height: '14px',
          border: `2px solid ${THEME.border}`,
          borderTopColor: THEME.accent,
          borderRadius: '50%',
          animation: 'spin 0.8s linear infinite',
        }}
      />
      Checking for updates...
      <style>{`
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  );
}

/**
 * Error state for version check
 */
export function UpdateCheckError({
  error,
  onRetry,
}: {
  error: string;
  onRetry: () => void;
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '8px 12px',
        color: THEME.toastError,
        fontSize: '13px',
      }}
    >
      <svg
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      {error}
      <button
        onClick={onRetry}
        style={{
          marginLeft: '8px',
          padding: '2px 8px',
          borderRadius: '4px',
          border: 'none',
          background: 'transparent',
          color: THEME.accent,
          fontSize: '12px',
          cursor: 'pointer',
          textDecoration: 'underline',
        }}
      >
        Retry
      </button>
    </div>
  );
}

/**
 * "Up to date" state
 */
export function UpdateUpToDate({ version }: { version: string }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '8px 12px',
        color: THEME.toastSuccess,
        fontSize: '13px',
      }}
    >
      <svg
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M20 6L9 17l-5-5" />
      </svg>
      You're up to date! ({version})
    </div>
  );
}
