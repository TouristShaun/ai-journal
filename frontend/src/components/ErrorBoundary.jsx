import React from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    console.error('ErrorBoundary caught error:', error, errorInfo);
    this.setState({
      error,
      errorInfo
    });
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
    // Reload the page to ensure clean state
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
          <div className="max-w-md w-full p-6 bg-white dark:bg-gray-800 rounded-lg shadow-lg">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="w-8 h-8 text-red-500" />
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                Something went wrong
              </h1>
            </div>
            
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              An unexpected error occurred. This might be a temporary issue.
            </p>

            {this.state.error && (
              <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
                <p className="text-sm text-red-800 dark:text-red-200 font-mono">
                  {this.state.error.toString()}
                </p>
              </div>
            )}

            <div className="flex gap-3">
              <button
                onClick={this.handleReset}
                className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                <RefreshCw className="w-4 h-4" />
                Reload Application
              </button>
              
              <button
                onClick={() => window.history.back()}
                className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
              >
                Go Back
              </button>
            </div>

            {process.env.NODE_ENV === 'development' && this.state.errorInfo && (
              <details className="mt-4">
                <summary className="text-sm text-gray-500 cursor-pointer hover:text-gray-700 dark:hover:text-gray-300">
                  Error Details (Development Only)
                </summary>
                <pre className="mt-2 text-xs bg-gray-100 dark:bg-gray-900 p-3 rounded overflow-auto max-h-64">
                  {this.state.errorInfo.componentStack}
                </pre>
              </details>
            )}
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;