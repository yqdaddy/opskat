import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Upload, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RemoteFileBrowser } from "@/components/terminal/RemotePathDialog";
import { TransferProgress } from "@/components/terminal/TransferProgress";
import { useSFTPStore } from "@/stores/sftpStore";
import { useTerminalStore } from "@/stores/terminalStore";

interface TerminalToolbarProps {
  tabId: string;
}

export function TerminalToolbar({ tabId }: TerminalToolbarProps) {
  const { t } = useTranslation();
  const startUpload = useSFTPStore((s) => s.startUpload);
  const startDownload = useSFTPStore((s) => s.startDownload);
  const startDownloadDir = useSFTPStore((s) => s.startDownloadDir);
  const tab = useTerminalStore((s) => s.tabs.find((t) => t.id === tabId));

  const [browserOpen, setBrowserOpen] = useState(false);
  const [browserMode, setBrowserMode] = useState<"upload" | "download">("upload");

  if (!tab) return null;

  const sessionId = tab.activePaneId;

  const openBrowser = (mode: "upload" | "download") => {
    setBrowserMode(mode);
    setBrowserOpen(true);
  };

  const handleBrowserConfirm = async (remotePath: string, isDir: boolean) => {
    if (!sessionId) return;
    if (browserMode === "upload") {
      // 上传到选中的目录，后端会弹出本地文件/目录选择
      await startUpload(sessionId, remotePath);
    } else {
      // 下载：根据选中的是文件还是目录
      if (isDir) {
        await startDownloadDir(sessionId, remotePath);
      } else {
        await startDownload(sessionId, remotePath);
      }
    }
  };

  return (
    <>
      <div className="flex items-center gap-1 px-2 py-1 border-t bg-background shrink-0">
        <Button
          variant="ghost"
          size="icon-xs"
          title={t("sftp.upload")}
          onClick={() => openBrowser("upload")}
        >
          <Upload className="h-3.5 w-3.5" />
        </Button>

        <Button
          variant="ghost"
          size="icon-xs"
          title={t("sftp.download")}
          onClick={() => openBrowser("download")}
        >
          <Download className="h-3.5 w-3.5" />
        </Button>

        {/* Transfer progress inline */}
        <div className="flex-1 min-w-0">
          <TransferProgress sessionId={sessionId} />
        </div>
      </div>

      <RemoteFileBrowser
        open={browserOpen}
        onOpenChange={setBrowserOpen}
        sessionId={sessionId}
        mode={browserMode}
        onConfirm={handleBrowserConfirm}
      />
    </>
  );
}
