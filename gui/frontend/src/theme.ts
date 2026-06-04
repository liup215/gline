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
}
