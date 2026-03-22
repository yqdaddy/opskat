import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { X, Loader2, CheckCircle2, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useSFTPStore, SFTPTransfer } from "@/stores/sftpStore";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + " " + units[i];
}

function TransferItem({ transfer }: { transfer: SFTPTransfer }) {
  const { t } = useTranslation();
  const cancelTransfer = useSFTPStore((s) => s.cancelTransfer);
  const clearTransfer = useSFTPStore((s) => s.clearTransfer);

  const percent =
    transfer.bytesTotal > 0
      ? Math.round((transfer.bytesDone / transfer.bytesTotal) * 100)
      : 0;

  return (
    <div className="flex items-center gap-2 text-xs">
      {transfer.status === "active" && (
        <Loader2 className="h-3 w-3 animate-spin text-primary shrink-0" />
      )}
      {transfer.status === "done" && (
        <CheckCircle2 className="h-3 w-3 text-green-500 shrink-0" />
      )}
      {(transfer.status === "error" || transfer.status === "cancelled") && (
        <XCircle className="h-3 w-3 text-destructive shrink-0" />
      )}

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-muted-foreground">
            {transfer.currentFile ||
              (transfer.direction === "upload"
                ? t("sftp.upload")
                : t("sftp.download"))}
          </span>
          <span className="shrink-0">
            {transfer.status === "active" && `${percent}%`}
            {transfer.status === "done" && t("sftp.completed")}
            {transfer.status === "error" && t("sftp.failed")}
            {transfer.status === "cancelled" && t("sftp.cancelled")}
          </span>
        </div>
        {transfer.status === "active" && (
          <div className="flex items-center gap-2 mt-0.5">
            <div className="flex-1 h-1 rounded-full bg-muted overflow-hidden">
              <div
                className="h-full rounded-full bg-primary transition-all duration-300"
                style={{ width: `${percent}%` }}
              />
            </div>
            <span className="text-muted-foreground shrink-0">
              {formatBytes(transfer.speed)}/s
            </span>
          </div>
        )}
        {transfer.filesTotal > 1 && transfer.status === "active" && (
          <span className="text-muted-foreground">
            {t("sftp.filesProgress", {
              completed: transfer.filesCompleted,
              total: transfer.filesTotal,
            })}
          </span>
        )}
        {transfer.status === "error" && transfer.error && (
          <span
            className="text-destructive truncate block"
            title={transfer.error}
          >
            {transfer.error}
          </span>
        )}
      </div>

      {transfer.status === "active" ? (
        <Button
          variant="ghost"
          size="icon-xs"
          className="h-4 w-4 shrink-0"
          onClick={() => cancelTransfer(transfer.transferId)}
          title={t("sftp.cancelTransfer")}
        >
          <X className="h-3 w-3" />
        </Button>
      ) : (
        <Button
          variant="ghost"
          size="icon-xs"
          className="h-4 w-4 shrink-0"
          onClick={() => clearTransfer(transfer.transferId)}
          title={t("sftp.clear")}
        >
          <X className="h-3 w-3" />
        </Button>
      )}
    </div>
  );
}

export function TransferProgress({ sessionId }: { sessionId: string }) {
  const allTransfers = useSFTPStore((s) => s.transfers);
  const transfers = useMemo(
    () => Object.values(allTransfers).filter((t) => t.sessionId === sessionId),
    [allTransfers, sessionId]
  );

  if (transfers.length === 0) return null;

  return (
    <div className="space-y-1">
      {transfers.map((t) => (
        <TransferItem key={t.transferId} transfer={t} />
      ))}
    </div>
  );
}
