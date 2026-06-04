export type ThemeName = 'dark' | 'light';

export interface ThemeColors {
  bg: string;
  bgSidebar: string;
  bgChat: string;
  border: string;
  accent: string;
  accentHover: string;
  text: string;
  textMuted: string;
  textDim: string;
  assistantBg: string;
  userBg: string;
  toolBg: string;
  codeBg: string;
  inputBg: string;
  cardBg: string;
  overlayBg: string;
  toastBg: string;
  toastSuccess: string;
  toastSuccessBg: string;
  toastError: string;
  toastErrorBg: string;
  codeInlineBg: string;
  codeInlineText: string;
  optionBg: string;
  optionHoverBg: string;
  linkColor: string;
  spinner: string;
  tableHeadBg: string;
  tableBorder: string;
  footnoteText: string;
  footnoteBorder: string;
  highlightJsTheme: string;
  userTextColor: string;
  statusSuccessBg: string;
  statusSuccessBorder: string;
  statusSuccessText: string;
  statusPendingBg: string;
  statusPendingBorder: string;
  statusPendingText: string;
  logoGradientStart: string;
  logoGradientEnd: string;
}

export const THEME_DARK: ThemeColors = {
  bg: '#0b1220',
  bgSidebar: '#0f172a',
  bgChat: '#0b1220',
  border: '#1e293b',
  accent: '#3b82f6',
  accentHover: '#2563eb',
  text: '#e2e8f0',
  textMuted: '#94a3b8',
  textDim: '#64748b',
  assistantBg: '#111827',
  userBg: '#1e40af',
  toolBg: '#111827',
  codeBg: '#1e1e2e',
  inputBg: '#1e293b',
  cardBg: '#111827',
  overlayBg: 'rgba(0, 0, 0, 0.6)',
  toastBg: 'rgba(0, 0, 0, 0.5)',
  toastSuccess: '#4ade80',
  toastSuccessBg: 'rgba(34, 197, 94, 0.1)',
  toastError: '#f87171',
  toastErrorBg: 'rgba(239, 68, 68, 0.1)',
  codeInlineBg: '#2d2d3a',
  codeInlineText: '#cdd6f4',
  optionBg: 'rgba(59, 130, 246, 0.1)',
  optionHoverBg: 'rgba(59, 130, 246, 0.2)',
  linkColor: '#60a5fa',
  spinner: '#3b82f6',
  tableHeadBg: '#1e293b',
  tableBorder: '#334155',
  footnoteText: '#94a3b8',
  footnoteBorder: '#1e293b',
  highlightJsTheme: 'github-dark.css',
  userTextColor: '#ffffff',
  statusSuccessBg: 'rgba(74, 222, 128, 0.08)',
  statusSuccessBorder: 'rgba(74, 222, 128, 0.25)',
  statusSuccessText: '#4ade80',
  statusPendingBg: 'rgba(251, 191, 36, 0.08)',
  statusPendingBorder: 'rgba(251, 191, 36, 0.25)',
  statusPendingText: '#fbbf24',
  logoGradientStart: '#60a5fa',
  logoGradientEnd: '#a78bfa',
} as const;

export const THEME_LIGHT: ThemeColors = {
  bg: '#f8fafc',
  bgSidebar: '#f1f5f9',
  bgChat: '#ffffff',
  border: '#e2e8f0',
  accent: '#3b82f6',
  accentHover: '#2563eb',
  text: '#1e293b',
  textMuted: '#64748b',
  textDim: '#94a3b8',
  assistantBg: '#f8fafc',
  userBg: '#dbeafe',
  toolBg: '#f8fafc',
  codeBg: '#f1f5f9',
  inputBg: '#f8fafc',
  cardBg: '#ffffff',
  overlayBg: 'rgba(0, 0, 0, 0.3)',
  toastBg: 'rgba(255, 255, 255, 0.9)',
  toastSuccess: '#22c55e',
  toastSuccessBg: 'rgba(34, 197, 94, 0.1)',
  toastError: '#ef4444',
  toastErrorBg: 'rgba(239, 68, 68, 0.1)',
  codeInlineBg: '#e2e8f0',
  codeInlineText: '#334155',
  optionBg: 'rgba(59, 130, 246, 0.1)',
  optionHoverBg: 'rgba(59, 130, 246, 0.2)',
  linkColor: '#2563eb',
  spinner: '#3b82f6',
  tableHeadBg: '#f1f5f9',
  tableBorder: '#e2e8f0',
  footnoteText: '#475569',
  footnoteBorder: '#e2e8f0',
  highlightJsTheme: 'github.min.css',
  userTextColor: '#1e293b',
  statusSuccessBg: 'rgba(34, 197, 94, 0.1)',
  statusSuccessBorder: 'rgba(34, 197, 94, 0.3)',
  statusSuccessText: '#22c55e',
  statusPendingBg: 'rgba(245, 158, 11, 0.1)',
  statusPendingBorder: 'rgba(245, 158, 11, 0.3)',
  statusPendingText: '#f59e0b',
  logoGradientStart: '#2563eb',
  logoGradientEnd: '#7c3aed',
} as const;

const CSS_VAR_MAP: Record<keyof ThemeColors, string> = {
  bg: '--theme-bg',
  bgSidebar: '--theme-bg-sidebar',
  bgChat: '--theme-bg-chat',
  border: '--theme-border',
  accent: '--theme-accent',
  accentHover: '--theme-accent-hover',
  text: '--theme-text',
  textMuted: '--theme-text-muted',
  textDim: '--theme-text-dim',
  assistantBg: '--theme-assistant-bg',
  userBg: '--theme-user-bg',
  toolBg: '--theme-tool-bg',
  codeBg: '--theme-code-bg',
  inputBg: '--theme-input-bg',
  cardBg: '--theme-card-bg',
  overlayBg: '--theme-overlay-bg',
  toastBg: '--theme-toast-bg',
  toastSuccess: '--theme-toast-success',
  toastSuccessBg: '--theme-toast-success-bg',
  toastError: '--theme-toast-error',
  toastErrorBg: '--theme-toast-error-bg',
  codeInlineBg: '--theme-code-inline-bg',
  codeInlineText: '--theme-code-inline-text',
  optionBg: '--theme-option-bg',
  optionHoverBg: '--theme-option-hover-bg',
  linkColor: '--theme-link-color',
  spinner: '--theme-spinner',
  tableHeadBg: '--theme-table-head-bg',
  tableBorder: '--theme-table-border',
  footnoteText: '--theme-footnote-text',
  footnoteBorder: '--theme-footnote-border',
  highlightJsTheme: '--theme-hljs-theme',
  userTextColor: '--theme-user-text-color',
  statusSuccessBg: '--theme-status-success-bg',
  statusSuccessBorder: '--theme-status-success-border',
  statusSuccessText: '--theme-status-success-text',
  statusPendingBg: '--theme-status-pending-bg',
  statusPendingBorder: '--theme-status-pending-border',
  statusPendingText: '--theme-status-pending-text',
  logoGradientStart: '--theme-logo-gradient-start',
  logoGradientEnd: '--theme-logo-gradient-end',
};

/** THEME exported for backward compatibility.
 *  All values are CSS var() references so that switching themes
 *  only requires updating the CSS variables on :root.
 *  New code should prefer useTheme() when it needs the raw colors.
 */
export const THEME: Record<keyof ThemeColors, string> = Object.fromEntries(
  Object.entries(CSS_VAR_MAP).map(([key, varName]) => [key, `var(${varName})`])
) as Record<keyof ThemeColors, string>;

/** Apply a set of theme colors to CSS custom properties on :root. */
export function applyThemeColors(colors: ThemeColors): void {
  const root = document.documentElement;
  (Object.keys(CSS_VAR_MAP) as Array<keyof ThemeColors>).forEach((key) => {
    root.style.setProperty(CSS_VAR_MAP[key], colors[key]);
  });

  // Sync highlight.js stylesheet with current theme
  const hljsTheme = (colors as any).highlightJsTheme || 'github-dark.css';
  const href = hljsTheme.includes('github-dark')
    ? '/styles/hljs-github-dark.css'
    : '/styles/hljs-github-light.css';
  let link = document.getElementById('hljs-theme') as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement('link');
    link.id = 'hljs-theme';
    link.rel = 'stylesheet';
    document.head.appendChild(link);
  }
  if (link.href !== href) {
    link.href = href;
  }
}
