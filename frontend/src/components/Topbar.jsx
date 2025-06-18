import { useState } from 'react';
import { PanelLeftClose, PanelLeft, Folder, ChartBar, Keyboard, BookOpen } from 'lucide-react';
import CollectionsModal from './CollectionsModal';
import Evaluations from './Evaluations';

function Topbar({ isSidebarCollapsed, onToggleSidebar, onShowShortcuts }) {
  const [showCollections, setShowCollections] = useState(false);
  const [showEvaluations, setShowEvaluations] = useState(false);

  return (
    <>
      <div className="h-16 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 flex items-center px-4 gap-4">
        {/* Sidebar Toggle */}
        <button
          onClick={onToggleSidebar}
          className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          title={isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {isSidebarCollapsed ? (
            <PanelLeft className="w-5 h-5 text-gray-600 dark:text-gray-400" />
          ) : (
            <PanelLeftClose className="w-5 h-5 text-gray-600 dark:text-gray-400" />
          )}
        </button>

        {/* Journal Title */}
        <div className="flex items-center gap-2">
          <BookOpen className="w-6 h-6 text-blue-600 dark:text-blue-400" />
          <h1 className="text-xl font-bold text-gray-900 dark:text-white">Journal</h1>
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Action Buttons */}
        <div className="flex items-center gap-2">
          {/* Collections */}
          <button
            onClick={() => setShowCollections(true)}
            className="flex items-center gap-2 px-3 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            title="Manage collections"
          >
            <Folder className="w-5 h-5 text-gray-600 dark:text-gray-400" />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Collections</span>
          </button>

          {/* Evaluations */}
          <button
            onClick={() => setShowEvaluations(true)}
            className="flex items-center gap-2 px-3 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            title="Run search evaluations"
          >
            <ChartBar className="w-5 h-5 text-gray-600 dark:text-gray-400" />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Evaluations</span>
          </button>

          {/* Keyboard Shortcuts */}
          <button
            onClick={onShowShortcuts}
            className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            title="Keyboard shortcuts (Shift+?)"
          >
            <Keyboard className="w-5 h-5 text-gray-600 dark:text-gray-400" />
          </button>
        </div>
      </div>

      {/* Modals */}
      {showCollections && (
        <CollectionsModal
          isOpen={showCollections}
          onClose={() => setShowCollections(false)}
        />
      )}

      {showEvaluations && (
        <Evaluations
          isOpen={showEvaluations}
          onClose={() => setShowEvaluations(false)}
        />
      )}
    </>
  );
}

export default Topbar;