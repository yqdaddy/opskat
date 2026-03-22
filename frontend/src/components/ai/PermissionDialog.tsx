import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import { RespondPermission } from "../../../wailsjs/go/main/App";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Shield } from "lucide-react";

interface PermissionRequest {
  tool_name: string;
  input: Record<string, unknown>;
}

export function PermissionDialog() {
  const { t } = useTranslation();
  const [request, setRequest] = useState<PermissionRequest | null>(null);

  useEffect(() => {
    const eventName = "ai:permission";
    EventsOn(eventName, (req: PermissionRequest) => {
      setRequest(req);
    });
    return () => {
      EventsOff(eventName);
    };
  }, []);

  const respond = (behavior: string) => {
    RespondPermission(behavior, behavior === "deny" ? "用户拒绝" : "");
    setRequest(null);
  };

  if (!request) return null;

  // 格式化工具输入信息
  const formatInput = (req: PermissionRequest): string => {
    if (req.tool_name === "Bash" && req.input?.command) {
      return String(req.input.command);
    }
    if ((req.tool_name === "Write" || req.tool_name === "Edit") && req.input?.file_path) {
      return String(req.input.file_path);
    }
    if (req.tool_name === "Read" && req.input?.file_path) {
      return String(req.input.file_path);
    }
    return JSON.stringify(req.input, null, 2);
  };

  return (
    <AlertDialog open={!!request}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5 text-amber-500" />
            {t("ai.permissionTitle")}
          </AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2">
              <div className="text-sm font-medium text-foreground">
                {request.tool_name}
              </div>
              <pre className="text-xs bg-muted p-2 rounded-md overflow-auto max-h-48 whitespace-pre-wrap break-all">
                {formatInput(request)}
              </pre>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <Button variant="outline" onClick={() => respond("deny")}>
            {t("ai.permissionDeny")}
          </Button>
          <Button variant="secondary" onClick={() => respond("allowAll")}>
            {t("ai.permissionAllowAll")}
          </Button>
          <Button onClick={() => respond("allow")}>
            {t("ai.permissionAllow")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
