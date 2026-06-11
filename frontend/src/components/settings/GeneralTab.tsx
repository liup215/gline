import { useTheme } from '../../ThemeContext';
import { selectStyle, labelStyle } from './sharedStyles';

interface GeneralTabProps {
  uiTheme: string;
  setUiTheme: (v: string) => void;
}

export function GeneralTab({ uiTheme, setUiTheme }: GeneralTabProps) {
  const { themeName: currentTheme, setTheme } = useTheme();

  return (
    <>
      <div style={{ marginBottom: '14px' }}>
        <label style={labelStyle}>Chat Theme</label>
        <select
          style={selectStyle}
          value={currentTheme}
          onChange={e => {
            const name = e.target.value as 'dark' | 'light';
            setTheme(name);
            setUiTheme(name);
          }}
        >
          <option value="dark">Dark</option>
          <option value="light">Light</option>
        </select>
      </div>
    </>
  );
}
