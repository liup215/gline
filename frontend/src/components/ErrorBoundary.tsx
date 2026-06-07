import { Component, ErrorInfo, ReactNode } from 'react';
import { THEME } from '../theme';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('React Error Boundary caught an error:', error, errorInfo);
  }

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      return (
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100vh',
          width: '100vw',
          background: THEME.cardBg,
          color: THEME.text,
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
          padding: '40px',
          textAlign: 'center',
        }}>
          <div style={{ fontSize: '3rem', marginBottom: '20px' }}>⚠️</div>
          <h1 style={{ margin: '0 0 12px', fontSize: '1.5rem', fontWeight: 600 }}>
            Something went wrong
          </h1>
          <p style={{ margin: '0 0 24px', color: THEME.textMuted, maxWidth: '400px', lineHeight: 1.5 }}>
            gline encountered an unexpected error. Click reload to restart the application.
          </p>
          <pre style={{
            background: THEME.inputBg,
            padding: '16px',
            borderRadius: '8px',
            fontSize: '0.8rem',
            color: THEME.toastError,
            maxWidth: '600px',
            maxHeight: '200px',
            overflow: 'auto',
            marginBottom: '24px',
            textAlign: 'left',
          }}>
            {this.state.error?.message || 'Unknown error'}
          </pre>
          <button
            onClick={this.handleReload}
            style={{
              padding: '10px 24px',
              borderRadius: '8px',
              border: 'none',
              background: THEME.accent,
              color: THEME.userTextColor,
              fontSize: '0.95rem',
              fontWeight: 500,
              cursor: 'pointer',
            }}
          >
            🔄 Reload Application
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
