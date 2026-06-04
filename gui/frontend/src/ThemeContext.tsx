import React, { createContext, useContext, useEffect, useState } from 'react';
import type { ThemeColors, ThemeName } from './theme';
import { THEME_DARK, THEME_LIGHT, applyThemeColors } from './theme';

const STORAGE_KEY = 'gline-theme';

interface ThemeContextValue {
  theme: ThemeColors;
  themeName: ThemeName;
  setTheme: (name: ThemeName) => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: THEME_DARK,
  themeName: 'dark',
  setTheme: () => {},
});

export function useTheme(): ThemeContextValue {
  return useContext(ThemeContext);
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [themeName, setThemeName] = useState<ThemeName>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY) as ThemeName | null;
      return stored === 'light' ? 'light' : 'dark';
    } catch {
      return 'dark';
    }
  });

  const theme = themeName === 'light' ? THEME_LIGHT : THEME_DARK;

  // Apply CSS custom properties whenever the theme changes
  useEffect(() => {
    applyThemeColors(theme);
  }, [theme]);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, themeName);
    } catch {
      // ignore storage errors in restricted environments
    }
  }, [themeName]);

  return (
    <ThemeContext.Provider value={{ theme, themeName, setTheme: setThemeName }}>
      {children}
    </ThemeContext.Provider>
  );
}
