import React from 'react';
import { Clock, GitCommit, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useCanvasStore } from '@/stores/canvas.store';
import { cn } from '@/lib/utils';

export const VersionHistory: React.FC = () => {
  const {
    currentArtifact,
    artifactHistory,
    loadVersion,
  } = useCanvasStore();

  if (!currentArtifact) return null;

  const history = artifactHistory.get(currentArtifact.id);
  if (!history) return null;

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    
    return date.toLocaleDateString();
  };

  const calculateDiff = (v1: string, v2: string) => {
    const lines1 = v1.split('\n');
    const lines2 = v2.split('\n');
    
    let added = 0;
    let removed = 0;
    
    // Simple diff calculation (in production, use a proper diff library)
    const maxLen = Math.max(lines1.length, lines2.length);
    for (let i = 0; i < maxLen; i++) {
      if (i >= lines1.length) {
        added++;
      } else if (i >= lines2.length) {
        removed++;
      } else if (lines1[i] !== lines2[i]) {
        added++;
        removed++;
      }
    }
    
    return { added, removed };
  };

  return (
    <div className="w-80 border-l border-border-subtle bg-background-secondary overflow-auto">
      <div className="sticky top-0 bg-background-secondary border-b border-border-subtle px-4 py-3">
        <h3 className="text-sm font-medium text-foreground-primary flex items-center gap-2">
          <Clock size={16} />
          Version History
        </h3>
        <p className="text-xs text-foreground-secondary mt-1">
          {history.versions.length} versions
        </p>
      </div>

      <div className="p-2">
        {history.versions.slice().reverse().map((version, index) => {
          const isCurrentVersion = version.version === currentArtifact.version;
          const prevVersion = history.versions[history.versions.length - index - 2];
          const diff = prevVersion ? calculateDiff(version.content, prevVersion.content) : null;
          
          return (
            <div
              key={version.id}
              className={cn(
                "mb-2 rounded-lg border transition-all cursor-pointer",
                isCurrentVersion
                  ? "border-accent-blue bg-accent-blue/10"
                  : "border-border-subtle hover:border-border-default hover:bg-background-tertiary"
              )}
              onClick={() => !isCurrentVersion && loadVersion(version.version)}
            >
              <div className="p-3">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <GitCommit size={14} className="text-foreground-secondary mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-foreground-primary">
                        Version {version.version}
                      </p>
                      <p className="text-xs text-foreground-secondary">
                        {formatDate(version.updatedAt)}
                      </p>
                    </div>
                  </div>
                  
                  {isCurrentVersion && (
                    <span className="text-xs px-2 py-1 bg-accent-blue text-white rounded">
                      Current
                    </span>
                  )}
                </div>

                {diff && (
                  <div className="flex items-center gap-3 mt-2 text-xs">
                    <span className="text-accent-green">+{diff.added}</span>
                    <span className="text-accent-red">-{diff.removed}</span>
                  </div>
                )}

                {!isCurrentVersion && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full mt-2 justify-between"
                    onClick={(e) => {
                      e.stopPropagation();
                      loadVersion(version.version);
                    }}
                  >
                    <span>Restore this version</span>
                    <ChevronRight size={14} />
                  </Button>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};