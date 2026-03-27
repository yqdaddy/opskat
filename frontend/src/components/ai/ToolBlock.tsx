import { useState } from "react";
import {
  Terminal,
  FileText,
  FilePen,
  Search,
  ChevronRight,
  Loader2,
  CheckCircle2,
  XCircle,
  Shield,
} from "lucide-react";
import type { ContentBlock } from "@/stores/aiStore";

const toolIcons: Record<string, typeof Terminal> = {
  Bash: Terminal,
  Read: FileText,
  Write: FilePen,
  Edit: FilePen,
  Glob: Search,
  Grep: Search,
  run_command: Terminal,
  request_permission: Shield,
};

interface ToolBlockProps {
  block: ContentBlock;
}

export function ToolBlock({ block }: ToolBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const Icon = toolIcons[block.toolName || ""] || Terminal;
  const isRunning = block.status === "running";
  const isError = block.status === "error";
  const isCancelled = block.status === "cancelled";
  const hasOutput = block.content && block.content.length > 0;

  return (
    <div
      className={`my-1.5 rounded-lg border bg-background text-xs overflow-hidden ${
        isRunning ? "border-primary/30" : "border-border/60"
      }`}
    >
      <button
        className="flex items-center gap-2 w-full min-w-0 px-3 py-2 h-[34px] text-left hover:bg-muted/50 transition-colors"
        onClick={() => hasOutput && setExpanded(!expanded)}
        disabled={!hasOutput}
      >
        {hasOutput && (
          <ChevronRight
            className={`h-3 w-3 shrink-0 text-muted-foreground transition-transform duration-150 ${
              expanded ? "rotate-90 opacity-100" : "opacity-50"
            }`}
          />
        )}
        {isRunning ? (
          <Loader2 className="h-3.5 w-3.5 shrink-0 text-primary animate-spin" />
        ) : (
          <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        )}
        <span className="font-medium text-foreground/80">{block.toolName}</span>
        {block.toolInput && (
          <code className="min-w-0 truncate text-muted-foreground font-mono text-[10px] ml-0.5">{block.toolInput}</code>
        )}
        <span className="ml-auto shrink-0">
          {isError && <XCircle className="h-3.5 w-3.5 text-destructive/70" />}
          {isCancelled && <XCircle className="h-3.5 w-3.5 text-muted-foreground/50" />}
          {!isRunning && !isError && !isCancelled && hasOutput && (
            <CheckCircle2 className="h-3.5 w-3.5 text-green-500/70" />
          )}
        </span>
      </button>

      {expanded && hasOutput && (
        <div className="border-t border-border/40 px-3 py-2 max-h-48 overflow-auto">
          <pre className="whitespace-pre-wrap break-all font-mono text-[11px] text-muted-foreground leading-relaxed">
            {block.content}
          </pre>
        </div>
      )}
    </div>
  );
}
