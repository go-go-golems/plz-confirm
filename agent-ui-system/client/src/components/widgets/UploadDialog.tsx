import React, { useState, useRef } from 'react';
import { UploadInput, UploadOutput } from '@/types/schemas';
import { Button } from '@/components/ui/button';
import { Loader2, Upload, File, X, CheckCircle } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Props {
  requestId: string;
  input: UploadInput;
  onSubmit: (output: UploadOutput) => Promise<void>;
  loading?: boolean;
}

interface FileStatus {
  file: File;
  progress: number;
  status: 'pending' | 'uploading' | 'completed' | 'error';
  path?: string;
}

export const UploadDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [files, setFiles] = useState<FileStatus[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      addFiles(Array.from(e.target.files));
    }
  };

  const addFiles = (newFiles: File[]) => {
    const validFiles = newFiles.filter(file => {
      // Check max size
      if (input.maxSize && file.size > input.maxSize) return false;
      // Check type (simple check)
      if (input.accept && input.accept.length > 0) {
        const ext = '.' + file.name.split('.').pop()?.toLowerCase();
        // This is a simplified check, a real one would be more robust
        return input.accept.some(type => type === ext || file.type.match(type));
      }
      return true;
    });

    const fileStatuses: FileStatus[] = validFiles.map(file => ({
      file,
      progress: 0,
      status: 'pending'
    }));

    if (input.multiple) {
      setFiles(prev => [...prev, ...fileStatuses]);
    } else {
      setFiles(fileStatuses.slice(0, 1));
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    if (e.dataTransfer.files) {
      addFiles(Array.from(e.dataTransfer.files));
    }
  };

  const removeFile = (index: number) => {
    setFiles(prev => prev.filter((_, i) => i !== index));
  };

  const simulateUpload = async () => {
    setSubmitting(true);
    
    // Simulate upload for each file
    const uploadedFiles = await Promise.all(files.map(async (fileStatus, index) => {
      // Update status to uploading
      setFiles(prev => prev.map((f, i) => i === index ? { ...f, status: 'uploading' } : f));
      
      // Simulate progress
      for (let i = 0; i <= 100; i += 10) {
        await new Promise(r => setTimeout(r, 100));
        setFiles(prev => prev.map((f, i) => i === index ? { ...f, progress: i } : f));
      }
      
      // Complete
      setFiles(prev => prev.map((f, i) => i === index ? { ...f, status: 'completed' } : f));
      
      return {
        name: fileStatus.file.name,
        size: fileStatus.file.size,
        path: `/tmp/uploads/${fileStatus.file.name}`,
        mimeType: fileStatus.file.type
      };
    }));

    await onSubmit({ files: uploadedFiles });
    setSubmitting(false);
  };

  return (
    <div className="bg-background p-6 md:p-8 min-h-[400px] flex flex-col relative">
      <div className="space-y-4 mb-6">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title}
        </h2>
        <div className="h-px w-full bg-border" />
      </div>

      <div 
        className={cn(
          "border-2 border-dashed border-border p-8 text-center transition-all duration-200 cursor-pointer mb-6",
          isDragging && "border-primary bg-primary/5",
          "hover:border-primary/50 hover:bg-primary/5"
        )}
        onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
      >
        <input 
          type="file" 
          ref={fileInputRef} 
          className="hidden" 
          multiple={input.multiple}
          accept={input.accept?.join(',')}
          onChange={handleFileSelect}
        />
        <div className="flex flex-col items-center gap-4">
          <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
            <Upload className="h-8 w-8 text-primary" />
          </div>
          <div className="space-y-1">
            <p className="font-mono text-sm font-bold uppercase">
              DROP_FILES_HERE_OR_CLICK
            </p>
            <p className="text-xs text-muted-foreground font-mono">
              {input.accept ? `ACCEPTED: ${input.accept.join(', ')}` : 'ALL_FILES_ACCEPTED'}
              {input.maxSize && ` • MAX: ${(input.maxSize / 1024 / 1024).toFixed(1)}MB`}
            </p>
          </div>
        </div>
      </div>

      {files.length > 0 && (
        <div className="space-y-3 mb-6">
          {files.map((file, idx) => (
            <div key={idx} className="flex items-center gap-3 p-3 border border-border bg-card">
              <File className="h-5 w-5 text-primary" />
              <div className="flex-1 min-w-0">
                <div className="flex justify-between mb-1">
                  <span className="text-sm font-mono truncate">{file.file.name}</span>
                  <span className="text-xs text-muted-foreground font-mono">
                    {(file.file.size / 1024).toFixed(1)}KB
                  </span>
                </div>
                <div className="h-1 w-full bg-secondary overflow-hidden">
                  <div 
                    className="h-full bg-primary transition-all duration-300"
                    style={{ width: `${file.progress}%` }}
                  />
                </div>
              </div>
              {file.status === 'completed' ? (
                <CheckCircle className="h-5 w-5 text-primary" />
              ) : (
                <button 
                  onClick={(e) => { e.stopPropagation(); removeFile(idx); }}
                  className="text-muted-foreground hover:text-destructive transition-colors"
                  disabled={submitting}
                >
                  <X className="h-5 w-5" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      <div className="flex justify-end pt-4 border-t border-border mt-auto">
        <Button 
          className="cyber-button min-w-[140px]"
          onClick={simulateUpload}
          disabled={loading || submitting || files.length === 0}
        >
          {submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          {submitting ? 'UPLOADING...' : 'START_UPLOAD'}
        </Button>
      </div>
    </div>
  );
};
