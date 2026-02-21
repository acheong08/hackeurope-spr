import { Upload, X, FileJson } from 'lucide-react';

interface FileUploadProps {
  uploadedFile: File | null;
  onFileUpload: (file: File) => void;
  onFileRemove: () => void;
}

export function FileUpload({ uploadedFile, onFileUpload, onFileRemove }: FileUploadProps) {
  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      if (file.name.endsWith('.json') || file.type === 'application/json') {
        onFileUpload(file);
      } else {
        alert('Please upload a valid package.json file');
      }
    }
  };

  return (
    <div className="flex items-center gap-3">
      {!uploadedFile ? (
        <label 
          className="flex items-center gap-2 px-4 py-2 rounded-lg cursor-pointer transition-all hover:opacity-80"
          style={{
            background: '#22c55e',
            color: '#000',
          }}
        >
          <Upload className="w-4 h-4" />
          <span className="text-sm font-medium">Upload File</span>
          <input
            type="file"
            accept=".json,application/json"
            onChange={handleFileChange}
            className="hidden"
          />
        </label>
      ) : (
        <div 
          className="flex items-center gap-3 px-4 py-2 rounded-lg"
          style={{
            background: '#111827',
            border: '1px solid #22c55e',
          }}
        >
          <FileJson className="w-4 h-4" style={{ color: '#22c55e' }} />
          <span className="text-sm" style={{ color: '#22c55e' }}>
            {uploadedFile.name}
          </span>
          <span className="text-xs" style={{ color: '#9ca3af' }}>
            ({(uploadedFile.size / 1024).toFixed(2)} KB)
          </span>
          <button
            onClick={onFileRemove}
            className="p-1 rounded transition-colors hover:bg-red-500/20"
            style={{ color: '#ef4444' }}
            aria-label="Remove file"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
}