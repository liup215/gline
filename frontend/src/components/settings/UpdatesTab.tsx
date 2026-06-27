import { THEME } from '../../theme';
import type { UpdateResult } from '../../hooks/useVersionCheck';

interface UpdatesTabProps {
  isChecking: boolean;
  updateResult: UpdateResult | null;
  error: string | null;
  onCheckForUpdates: () => void;
  onOpenReleasePage: () => void;
}

export function UpdatesTab({
  isChecking,
  updateResult,
  error,
  onCheckForUpdates,
  onOpenReleasePage,
}: UpdatesTabProps) {
  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  };

  const sectionStyle: React.CSSProperties = {
    padding: '16px',
    background: THEME.bgChat,
    borderRadius: '8px',
    border: `1px solid ${THEME.border}`,
  };

  const rowStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  };

  const versionLabelStyle: React.CSSProperties = {
    fontSize: '0.85rem',
    color: THEME.textMuted,
  };

  const versionValueStyle: React.CSSProperties = {
    fontSize: '0.9rem',
    color: THEME.text,
    fontWeight: 500,
    fontFamily: 'monospace',
  };

  const buttonStyle: React.CSSProperties = {
    padding: '10px 20px',
    borderRadius: '8px',
    border: 'none',
    background: THEME.accent,
    color: '#ffffff',
    fontSize: '0.9rem',
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'all 0.2s ease',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  };

  const secondaryButtonStyle: React.CSSProperties = {
    ...buttonStyle,
    background: 'transparent',
    color: THEME.accent,
    border: `1px solid ${THEME.accent}`,
  };

  const statusContainerStyle: React.CSSProperties = {
    padding: '12px',
    borderRadius: '6px',
    marginTop: '12px',
    display: 'flex',
    alignItems: 'flex-start',
    gap: '12px',
  };

  const successStatusStyle: React.CSSProperties = {
    ...statusContainerStyle,
    background: THEME.statusSuccessBg,
    border: `1px solid ${THEME.statusSuccessBorder}`,
  };

  const errorStatusStyle: React.CSSProperties = {
    ...statusContainerStyle,
    background: THEME.toastErrorBg,
    border: `1px solid ${THEME.toastError}`,
  };

  const infoStatusStyle: React.CSSProperties = {
    ...statusContainerStyle,
    background: THEME.statusPendingBg,
    border: `1px solid ${THEME.statusPendingBorder}`,
  };

  const statusTextStyle: React.CSSProperties = {
    fontSize: '0.9rem',
    lineHeight: 1.5,
  };

  const releaseNotesStyle: React.CSSProperties = {
    marginTop: '12px',
    padding: '12px',
    background: THEME.codeBg,
    borderRadius: '6px',
    fontSize: '0.85rem',
    color: THEME.textMuted,
    maxHeight: '200px',
    overflow: 'auto',
    whiteSpace: 'pre-wrap',
    fontFamily: 'monospace',
  };

  const hasUpdate = updateResult?.has_update ?? false;
  const isSkipped = updateResult?.skipped ?? false;

  return (
    <div style={containerStyle}>
      {/* Current Version */}
      <div style={sectionStyle}>
        <div style={rowStyle}>
          <span style={versionLabelStyle}>Current Version</span>
          <span style={versionValueStyle}>
            {updateResult?.current_version || 'dev'}
          </span>
        </div>
        
        {updateResult?.latest_version && (
          <div style={rowStyle}>
            <span style={versionLabelStyle}>Latest Version</span>
            <span style={{...versionValueStyle, color: hasUpdate ? THEME.toastSuccess : THEME.textMuted}}>
              {updateResult.latest_version}
            </span>
          </div>
        )}

        {updateResult?.checked_at && (
          <div style={rowStyle}>
            <span style={versionLabelStyle}>Last Checked</span>
            <span style={versionValueStyle}>
              {new Date(updateResult.checked_at).toLocaleString()}
            </span>
          </div>
        )}
      </div>

      {/* Check for Updates Button */}
      <div style={{ display: 'flex', gap: '12px' }}>
        <button
          onClick={onCheckForUpdates}
          disabled={isChecking}
          style={{
            ...buttonStyle,
            opacity: isChecking ? 0.7 : 1,
            cursor: isChecking ? 'not-allowed' : 'pointer',
          }}
          onMouseEnter={(e) => {
            if (!isChecking) {
              e.currentTarget.style.background = THEME.accentHover;
            }
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = THEME.accent;
          }}
        >
          {isChecking ? (
            <>
              <span style={{ animation: 'spin 1s linear infinite' }}>🔄</span>
              Checking...
            </>
          ) : (
            <>
              <span>🔍</span>
              Check for Updates
            </>
          )}
        </button>

        {hasUpdate && (
          <button
            onClick={onOpenReleasePage}
            style={secondaryButtonStyle}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = THEME.optionBg;
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = 'transparent';
            }}
          >
            <span>⬇️</span>
            Download Update
          </button>
        )}
      </div>

      {/* Status Messages */}
      {isChecking && (
        <div style={infoStatusStyle}>
          <span>🔄</span>
          <span style={statusTextStyle}>
            Checking for updates... This may take a few seconds.
          </span>
        </div>
      )}

      {error && !isChecking && (
        <div style={errorStatusStyle}>
          <span>⚠️</span>
          <span style={{...statusTextStyle, color: THEME.toastError}}>
            {error}
          </span>
        </div>
      )}

      {isSkipped && !isChecking && (
        <div style={infoStatusStyle}>
          <span>⏸️</span>
          <span style={statusTextStyle}>
            {updateResult?.skip_reason || 'Update check was skipped.'}
          </span>
        </div>
      )}

      {hasUpdate && !isChecking && (
        <div style={successStatusStyle}>
          <span>🎉</span>
          <div style={{ flex: 1 }}>
            <span style={{...statusTextStyle, color: THEME.statusSuccessText, fontWeight: 500}}>
              Update available! Version {updateResult?.latest_version} is ready to download.
            </span>
            {updateResult?.release_info?.published_at && (
              <div style={{ fontSize: '0.85rem', color: THEME.textMuted, marginTop: '4px' }}>
                Released: {new Date(updateResult.release_info.published_at).toLocaleDateString()}
              </div>
            )}
            {updateResult?.release_info?.body && (
              <div style={releaseNotesStyle}>
                {updateResult.release_info.body}
              </div>
            )}
          </div>
        </div>
      )}

      {!hasUpdate && !isChecking && !error && !isSkipped && updateResult && (
        <div style={{...statusContainerStyle, background: THEME.statusSuccessBg, border: `1px solid ${THEME.statusSuccessBorder}`}}>
          <span>✅</span>
          <span style={{...statusTextStyle, color: THEME.statusSuccessText}}>
            You are running the latest version.
          </span>
        </div>
      )}

      <style>{`
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  );
}

export default UpdatesTab;
