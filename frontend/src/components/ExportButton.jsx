import { useState } from 'react';
import { Download, FileJson, FileText, FileSpreadsheet } from 'lucide-react';

function ExportButton({ searchParams }) {
  const [isExporting, setIsExporting] = useState(false);
  const [showMenu, setShowMenu] = useState(false);

  const exportFormats = [
    { id: 'json', name: 'JSON', icon: FileJson, description: 'Complete data with metadata' },
    { id: 'markdown', name: 'Markdown', icon: FileText, description: 'Formatted for reading' },
    { id: 'csv', name: 'CSV', icon: FileSpreadsheet, description: 'For spreadsheet apps' },
  ];

  const handleExport = async (format) => {
    setIsExporting(true);
    setShowMenu(false);

    try {
      // Build query string
      const params = new URLSearchParams();
      params.append('format', format);
      
      if (searchParams.query) {
        params.append('query', searchParams.query);
      }
      
      if (searchParams.is_favorite) {
        params.append('is_favorite', 'true');
      }
      
      if (searchParams.collection_ids?.length > 0) {
        params.append('collection_ids', searchParams.collection_ids.join(','));
      }

      // Fetch export
      const response = await fetch(`http://localhost:8080/api/export?${params.toString()}`);
      if (!response.ok) {
        throw new Error('Export failed');
      }

      // Get filename from Content-Disposition header
      const contentDisposition = response.headers.get('Content-Disposition');
      const filenameMatch = contentDisposition?.match(/filename=(.+)/);
      const filename = filenameMatch ? filenameMatch[1] : `journal-export.${format}`;

      // Download file
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Export failed:', error);
      alert('Failed to export entries. Please try again.');
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div className="relative">
      <button
        onClick={() => setShowMenu(!showMenu)}
        disabled={isExporting}
        className="flex items-center gap-2 px-3 py-2 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
      >
        <Download className={`w-4 h-4 ${isExporting ? 'animate-bounce' : ''}`} />
        {isExporting ? 'Exporting...' : 'Export'}
      </button>

      {showMenu && (
        <div className="absolute right-0 mt-2 w-64 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-10">
          <div className="p-2">
            <h3 className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider px-2 py-1">
              Export Format
            </h3>
            {exportFormats.map((format) => {
              const Icon = format.icon;
              return (
                <button
                  key={format.id}
                  onClick={() => handleExport(format.id)}
                  className="w-full text-left p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                >
                  <div className="flex items-start gap-3">
                    <Icon className="w-5 h-5 text-gray-500 dark:text-gray-400 mt-0.5" />
                    <div>
                      <div className="font-medium text-sm text-gray-900 dark:text-white">
                        {format.name}
                      </div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {format.description}
                      </div>
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Click outside to close */}
      {showMenu && (
        <div
          className="fixed inset-0 z-0"
          onClick={() => setShowMenu(false)}
        />
      )}
    </div>
  );
}

export default ExportButton;