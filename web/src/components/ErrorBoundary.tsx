import { Component, type ErrorInfo, type ReactNode } from 'react';

type Props = { children: ReactNode };
type State = { hasError: boolean };

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Uncaught application error', error, info.componentStack);
  }

  render() {
    if (!this.state.hasError) return this.props.children;
    return (
      <main className="error-boundary" role="alert">
        <h1>Something went wrong</h1>
        <p>Reload the page to try again.</p>
        <button className="header-btn" onClick={() => window.location.reload()}>
          Reload
        </button>
      </main>
    );
  }
}
