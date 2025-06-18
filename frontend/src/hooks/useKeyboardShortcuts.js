import { useEffect } from 'react';

export function useKeyboardShortcuts(shortcuts) {
  useEffect(() => {
    const handleKeyDown = (event) => {
      // Check if user is typing in an input field
      const isTyping = ['INPUT', 'TEXTAREA'].includes(event.target.tagName);
      
      shortcuts.forEach((shortcut) => {
        const { key, ctrlKey = false, metaKey = false, shiftKey = false, action, allowInInput = false } = shortcut;
        
        // Skip if typing and not allowed in input
        if (isTyping && !allowInInput) return;
        
        const isCtrlPressed = ctrlKey ? (event.ctrlKey || event.metaKey) : true;
        const isMetaPressed = metaKey ? event.metaKey : true;
        const isShiftPressed = shiftKey ? event.shiftKey : !event.shiftKey;
        
        if (
          event.key === key &&
          isCtrlPressed &&
          isMetaPressed &&
          isShiftPressed
        ) {
          event.preventDefault();
          action(event);
        }
      });
    };
    
    window.addEventListener('keydown', handleKeyDown);
    
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [shortcuts]);
}

// Common shortcuts
export const SHORTCUTS = {
  NEW_ENTRY: { key: 'n', ctrlKey: true },
  SEARCH: { key: 'k', ctrlKey: true },
  TOGGLE_FAVORITE: { key: 's', ctrlKey: true },
  ESCAPE: { key: 'Escape' },
  SAVE: { key: 's', ctrlKey: true },
  EDIT: { key: 'e', ctrlKey: true },
};