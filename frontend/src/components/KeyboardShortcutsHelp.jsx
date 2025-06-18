import { Keyboard, X } from 'lucide-react';

function KeyboardShortcutsHelp({ isOpen, onClose }) {
  if (!isOpen) return null;

  const shortcuts = [
    { keys: ['Ctrl', 'N'], description: 'Create new entry' },
    { keys: ['Ctrl', 'K'], description: 'Focus search' },
    { keys: ['Ctrl', 'E'], description: 'Edit entry' },
    { keys: ['Ctrl', 'S'], description: 'Save entry' },
    { keys: ['Ctrl', 'Enter'], description: 'Toggle fullscreen' },
    { keys: ['Esc'], description: 'Close/Cancel' },
    { keys: ['?'], description: 'Show keyboard shortcuts' },
  ];

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md mx-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <Keyboard className="w-5 h-5" />
            Keyboard Shortcuts
          </h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="space-y-3">
          {shortcuts.map((shortcut, idx) => (
            <div key={idx} className="flex items-center justify-between">
              <div className="flex gap-1">
                {shortcut.keys.map((key, keyIdx) => (
                  <span key={keyIdx}>
                    <kbd className="px-2 py-1 text-xs font-semibold text-gray-800 bg-gray-100 border border-gray-200 rounded-lg dark:bg-gray-700 dark:text-gray-100 dark:border-gray-600">
                      {key}
                    </kbd>
                    {keyIdx < shortcut.keys.length - 1 && (
                      <span className="mx-1 text-gray-500">+</span>
                    )}
                  </span>
                ))}
              </div>
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {shortcut.description}
              </span>
            </div>
          ))}
        </div>

        <div className="mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
          <p className="text-xs text-gray-500 dark:text-gray-400 text-center">
            Press <kbd className="px-1 py-0.5 text-xs font-semibold text-gray-800 bg-gray-100 border border-gray-200 rounded dark:bg-gray-700 dark:text-gray-100 dark:border-gray-600">?</kbd> anytime to show this help
          </p>
        </div>
      </div>
    </div>
  );
}

export default KeyboardShortcutsHelp;