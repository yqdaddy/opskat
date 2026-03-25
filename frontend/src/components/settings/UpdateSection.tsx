import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  GetAppVersion,
  GetUpdateChannel,
  SetUpdateChannel,
  CheckForUpdate,
  DownloadAndInstallUpdate,
} from "../../../wailsjs/go/app/App";
import { Download, Loader2, ExternalLink, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { BrowserOpenURL, Quit } from "../../../wailsjs/runtime/runtime";
import { EventsOn } from "../../../wailsjs/runtime/runtime";

const errMsg = (e: unknown) => (e instanceof Error ? e.message : String(e));

export function UpdateSection() {
  const { t } = useTranslation();
  const [currentVersion, setCurrentVersion] = useState("");
  const [channel, setChannel] = useState("stable");
  const [checking, setChecking] = useState(false);
  const [updateInfo, setUpdateInfo] = useState<{
    hasUpdate: boolean;
    latestVersion: string;
    releaseNotes: string;
    releaseURL: string;
    publishedAt: string;
  } | null>(null);
  const [updating, setUpdating] = useState(false);
  const [progress, setProgress] = useState(0);
  const [updateDone, setUpdateDone] = useState(false);

  useEffect(() => {
    GetAppVersion()
      .then(setCurrentVersion)
      .catch(() => {});
    GetUpdateChannel()
      .then(setChannel)
      .catch(() => {});
  }, []);

  const handleChannelChange = async (value: string) => {
    setChannel(value);
    setUpdateInfo(null);
    try {
      await SetUpdateChannel(value);
    } catch (e: unknown) {
      toast.error(errMsg(e));
    }
  };

  useEffect(() => {
    const cancelProgress = EventsOn("update:progress", (data: { downloaded: number; total: number }) => {
      if (data.total > 0) {
        setProgress(Math.round((data.downloaded / data.total) * 100));
      }
    });
    const cancelOpsctlErr = EventsOn("update:opsctl-error", (errMsg: string) => {
      toast.error(t("appUpdate.opsctlUpdateFailed", { error: errMsg }));
    });
    const cancelSkillErr = EventsOn("update:skill-error", (errMsg: string) => {
      toast.error(t("appUpdate.skillUpdateFailed", { error: errMsg }));
    });
    return () => {
      cancelProgress();
      cancelOpsctlErr();
      cancelSkillErr();
    };
  }, [t]);

  const handleCheck = async () => {
    setChecking(true);
    setUpdateInfo(null);
    try {
      const info = await CheckForUpdate();
      setUpdateInfo(info);
      if (!info.hasUpdate) {
        toast.success(t("appUpdate.latestVersion"));
      }
    } catch (e: unknown) {
      toast.error(`${t("appUpdate.checkFailed")}: ${errMsg(e)}`);
    } finally {
      setChecking(false);
    }
  };

  const handleUpdate = async () => {
    setUpdating(true);
    setProgress(0);
    try {
      await DownloadAndInstallUpdate();
      setUpdateDone(true);
      toast.success(t("appUpdate.updateSuccess"));
    } catch (e: unknown) {
      toast.error(`${t("appUpdate.updateFailed")}: ${errMsg(e)}`);
    } finally {
      setUpdating(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t("appUpdate.title")}</CardTitle>
        <CardDescription>{t("appUpdate.desc")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex justify-between items-center text-sm">
          <span className="text-muted-foreground">{t("appUpdate.currentVersion")}</span>
          <span className="font-mono text-xs">{currentVersion || "dev"}</span>
        </div>

        <div className="flex justify-between items-center text-sm">
          <span className="text-muted-foreground">{t("appUpdate.updateChannel")}</span>
          <Select value={channel} onValueChange={handleChannelChange}>
            <SelectTrigger className="w-[160px] h-8 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="stable">{t("appUpdate.channelStable")}</SelectItem>
              <SelectItem value="beta">{t("appUpdate.channelBeta")}</SelectItem>
              <SelectItem value="nightly">{t("appUpdate.channelNightly")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {channel === "nightly" && <p className="text-xs text-muted-foreground">{t("appUpdate.nightlyWarning")}</p>}
        {channel === "beta" && <p className="text-xs text-muted-foreground">{t("appUpdate.betaWarning")}</p>}

        <div className="flex gap-2">
          <Button onClick={handleCheck} disabled={checking || updating} size="sm" variant="outline">
            {checking ? (
              <>
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                {t("appUpdate.checking")}
              </>
            ) : (
              <>
                <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
                {t("appUpdate.checkUpdate")}
              </>
            )}
          </Button>
        </div>

        {updateInfo?.hasUpdate && (
          <div className="space-y-3 border rounded-md p-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">
                {t("appUpdate.newVersion")}: {updateInfo.latestVersion}
              </span>
              <Button
                variant="link"
                size="sm"
                className="h-auto p-0 text-xs"
                onClick={() => BrowserOpenURL(updateInfo.releaseURL)}
              >
                <ExternalLink className="h-3 w-3 mr-1" />
                {t("appUpdate.viewRelease")}
              </Button>
            </div>

            {updateInfo.releaseNotes && (
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground">{t("appUpdate.releaseNotes")}</p>
                <pre className="text-xs bg-muted p-2 rounded-md overflow-auto max-h-[200px] whitespace-pre-wrap">
                  {updateInfo.releaseNotes}
                </pre>
              </div>
            )}

            {updating && (
              <div className="space-y-1">
                <div className="w-full bg-muted rounded-full h-2">
                  <div className="bg-primary h-2 rounded-full transition-all" style={{ width: `${progress}%` }} />
                </div>
                <p className="text-xs text-muted-foreground text-center">
                  {t("appUpdate.downloadProgress", { percent: progress })}
                </p>
              </div>
            )}

            {!updateDone ? (
              <Button onClick={handleUpdate} disabled={updating} size="sm">
                {updating ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                    {t("appUpdate.downloading")}
                  </>
                ) : (
                  <>
                    <Download className="h-3.5 w-3.5 mr-1.5" />
                    {t("appUpdate.download")}
                  </>
                )}
              </Button>
            ) : (
              <div className="flex gap-2">
                <Button onClick={() => Quit()} size="sm">
                  {t("appUpdate.restartNow")}
                </Button>
                <Button variant="outline" size="sm" onClick={() => setUpdateDone(false)}>
                  {t("appUpdate.restartLater")}
                </Button>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
