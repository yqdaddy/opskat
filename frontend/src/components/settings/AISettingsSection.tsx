import { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useAIStore } from "@/stores/aiStore";
import {
  DetectOpsctl,
  GetOpsctlInstallDir,
  InstallOpsctl,
  DetectSkills,
  InstallSkills,
  GetSkillPreview,
  GetDataDir,
  GetAppVersion,
  LoadAISetting,
} from "../../../wailsjs/go/app/App";
import {
  Check,
  Loader2,
  ExternalLink,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  Info,
} from "lucide-react";
import { toast } from "sonner";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";

const errMsg = (e: unknown) => (e instanceof Error ? e.message : String(e));

function maskApiKey(key: string): string {
  if (key.length <= 8) return "****";
  return key.slice(0, 4) + "****" + key.slice(-4);
}

function IntegrationSection() {
  const { t } = useTranslation();
  const [opsctlInfo, setOpsctlInfo] = useState<{
    installed: boolean;
    path: string;
    version: string;
    embedded: boolean;
  }>({ installed: false, path: "", version: "", embedded: false });
  const [skillTargets, setSkillTargets] = useState<{ name: string; installed: boolean; path: string }[]>([]);
  const [installDir, setInstallDir] = useState("");
  const [installing, setInstalling] = useState(false);
  const [skillInstalling, setSkillInstalling] = useState(false);
  const [skillPreview, setSkillPreview] = useState("");
  const [showPreview, setShowPreview] = useState(false);
  const [dataDir, setDataDir] = useState("");
  const [appVersion, setAppVersion] = useState("");

  const detect = useCallback(async () => {
    try {
      const [info, skills, dir, dd, ver] = await Promise.all([
        DetectOpsctl(),
        DetectSkills(),
        GetOpsctlInstallDir(),
        GetDataDir(),
        GetAppVersion(),
      ]);
      setOpsctlInfo(info);
      setSkillTargets(skills || []);
      setInstallDir(dir);
      setDataDir(dd);
      setAppVersion(ver);
    } catch {
      // detection is optional
    }
  }, []);

  useEffect(() => {
    detect();
  }, [detect]);

  const handleInstallCLI = async () => {
    setInstalling(true);
    try {
      await InstallOpsctl(installDir);
      toast.success(t("integration.installSuccess"));
      await detect();
      toast.info(`${t("integration.pathHint")}: ${installDir}`);
    } catch (e: unknown) {
      toast.error(`${t("integration.installFailed")}: ${errMsg(e)}`);
    } finally {
      setInstalling(false);
    }
  };

  const handleInstallSkill = async () => {
    setSkillInstalling(true);
    try {
      await InstallSkills();
      toast.success(t("integration.skillInstallSuccess"));
      await detect();
    } catch (e: unknown) {
      toast.error(errMsg(e));
    } finally {
      setSkillInstalling(false);
    }
  };

  const handlePreview = async () => {
    if (showPreview) {
      setShowPreview(false);
      return;
    }
    try {
      const content = await GetSkillPreview();
      setSkillPreview(content);
      setShowPreview(true);
    } catch {
      // preview is optional
    }
  };

  return (
    <>
      {/* opsctl CLI */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base">{t("integration.cli")}</CardTitle>
              <CardDescription>{t("integration.cliDesc")}</CardDescription>
            </div>
            <div className="flex items-center gap-2">
              {opsctlInfo.installed ? (
                <span className="inline-flex items-center gap-1 text-xs font-medium text-green-600 dark:text-green-400">
                  <Check className="h-3.5 w-3.5" />
                  {t("integration.installed")}
                </span>
              ) : (
                <span className="text-xs text-muted-foreground">{t("integration.notInstalled")}</span>
              )}
              <Button variant="ghost" size="icon" className="h-7 w-7" onClick={detect}>
                <RefreshCw className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {opsctlInfo.installed && (
            <div className="space-y-3">
              <div className="grid gap-1.5 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{t("integration.version")}</span>
                  <span className="font-mono text-xs">{opsctlInfo.version || "unknown"}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{t("integration.path")}</span>
                  <span className="font-mono text-xs truncate max-w-[300px]">{opsctlInfo.path}</span>
                </div>
              </div>
              {appVersion && opsctlInfo.version && opsctlInfo.version !== appVersion && (
                <div className="flex items-center gap-2 rounded-md bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 px-3 py-2 text-xs text-yellow-700 dark:text-yellow-300">
                  <Info className="h-3.5 w-3.5 shrink-0" />
                  <span>{t("integration.versionMismatch", { appVersion, cliVersion: opsctlInfo.version })}</span>
                </div>
              )}
              {opsctlInfo.embedded && (
                <div className="space-y-2">
                  <div className="grid gap-1.5">
                    <Label className="text-sm">{t("integration.installDir")}</Label>
                    <Input
                      value={installDir}
                      onChange={(e) => setInstallDir(e.target.value)}
                      className="font-mono text-xs h-8"
                    />
                  </div>
                  <Button onClick={handleInstallCLI} disabled={installing} size="sm" variant="outline">
                    {installing ? (
                      <>
                        <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                        {t("integration.installing")}
                      </>
                    ) : (
                      t("integration.reinstall")
                    )}
                  </Button>
                </div>
              )}
            </div>
          )}

          {!opsctlInfo.installed && (
            <div className="space-y-3">
              {opsctlInfo.embedded ? (
                <div className="space-y-2">
                  <div className="grid gap-1.5">
                    <Label className="text-sm">{t("integration.installDir")}</Label>
                    <Input
                      value={installDir}
                      onChange={(e) => setInstallDir(e.target.value)}
                      className="font-mono text-xs h-8"
                    />
                  </div>
                  <Button onClick={handleInstallCLI} disabled={installing} size="sm">
                    {installing ? (
                      <>
                        <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                        {t("integration.installing")}
                      </>
                    ) : (
                      t("integration.install")
                    )}
                  </Button>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">{t("integration.noEmbedded")}</p>
              )}
              <Separator />
              <div className="space-y-1">
                <p className="text-sm font-medium">{t("integration.manualInstall")}</p>
                <p className="text-xs text-muted-foreground">{t("integration.manualInstallHint")}</p>
                <Button
                  variant="link"
                  size="sm"
                  className="h-auto p-0 text-xs"
                  onClick={() => BrowserOpenURL("https://github.com/opskat/opskat/release")}
                >
                  <ExternalLink className="h-3 w-3 mr-1" />
                  GitHub Releases
                </Button>
              </div>
            </div>
          )}

          <Separator />
          <div className="space-y-1">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("integration.dataDir")}</span>
              <span className="font-mono text-xs truncate max-w-[300px]">{dataDir}</span>
            </div>
            <p className="text-xs text-muted-foreground">{t("integration.dataDirDesc")}</p>
          </div>
        </CardContent>
      </Card>

      {/* AI Skill */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base">{t("integration.skill")}</CardTitle>
              <CardDescription>{t("integration.skillDesc")}</CardDescription>
            </div>
            <div className="flex items-center gap-2">
              {skillTargets
                .filter((s) => s.installed)
                .map((s) => (
                  <span
                    key={s.name}
                    className="inline-flex items-center gap-1 text-xs font-medium text-green-600 dark:text-green-400"
                  >
                    <Check className="h-3.5 w-3.5" />
                    {s.name}
                  </span>
                ))}
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {skillTargets.some((s) => s.installed) && (
            <div className="space-y-1">
              {skillTargets
                .filter((s) => s.installed)
                .map((s) => (
                  <div key={s.name} className="flex justify-between text-sm">
                    <span className="text-muted-foreground">{s.name}</span>
                    <span className="font-mono text-xs truncate max-w-[300px]">{s.path}</span>
                  </div>
                ))}
            </div>
          )}

          <div className="flex gap-2">
            <Button onClick={handleInstallSkill} disabled={skillInstalling} size="sm">
              {skillInstalling ? (
                <>
                  <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                  {t("integration.skillInstalling")}
                </>
              ) : skillTargets.every((s) => s.installed) ? (
                t("integration.skillUpdate")
              ) : (
                t("integration.skillInstall")
              )}
            </Button>
            <Button variant="outline" size="sm" onClick={handlePreview}>
              {showPreview ? <ChevronUp className="h-3.5 w-3.5 mr-1" /> : <ChevronDown className="h-3.5 w-3.5 mr-1" />}
              {t("integration.skillPreview")}
            </Button>
          </div>

          {showPreview && (
            <pre className="text-xs bg-muted p-3 rounded-md overflow-auto max-h-[300px] whitespace-pre-wrap">
              {skillPreview}
            </pre>
          )}
        </CardContent>
      </Card>
    </>
  );
}

export interface AISettingsSectionProps {
  providerType: string;
  setProviderType: (v: string) => void;
  apiBase: string;
  setApiBase: (v: string) => void;
  apiKey: string;
  setApiKey: (v: string) => void;
  apiKeyPlaceholder: string;
  setApiKeyPlaceholder: (v: string) => void;
  model: string;
  setModel: (v: string) => void;
}

export function AISettingsSection({
  providerType,
  setProviderType,
  apiBase,
  setApiBase,
  apiKey,
  setApiKey,
  apiKeyPlaceholder,
  setApiKeyPlaceholder,
  model,
  setModel,
}: AISettingsSectionProps) {
  const { t } = useTranslation();
  const { configure, configured, detectCLIs, localCLIs } = useAIStore();
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    detectCLIs();
  }, [detectCLIs]);

  useEffect(() => {
    LoadAISetting()
      .then((info) => {
        if (info && info.configured) {
          setProviderType(info.providerType);
          setApiBase(info.apiBase);
          setModel(info.model);
          setApiKeyPlaceholder(info.maskedApiKey || "");
        }
      })
      .catch(() => {});
  }, [setProviderType, setApiBase, setModel, setApiKeyPlaceholder]);

  const handleSaveAI = async () => {
    try {
      await configure(providerType, apiBase, apiKey, model);
      if (apiKey) {
        setApiKeyPlaceholder(maskApiKey(apiKey));
        setApiKey("");
      }
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      toast.error(String(e));
    }
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">AI Provider</CardTitle>
          <CardDescription>
            {configured ? "✓ " + t("settings.configured") : t("ai.notConfigured")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label>{t("settings.providerType")}</Label>
            <Select value={providerType} onValueChange={setProviderType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai" disabled>
                  OpenAI Compatible ({t("setup.developing")})
                </SelectItem>
                <SelectItem value="local_cli">Local CLI</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {providerType === "openai" && (
            <>
              <div className="grid gap-2">
                <Label>API Base URL</Label>
                <Input value={apiBase} onChange={(e) => setApiBase(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label>API Key</Label>
                <Input
                  type="password"
                  value={apiKey}
                  placeholder={apiKeyPlaceholder || "sk-..."}
                  onChange={(e) => setApiKey(e.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label>{t("settings.model")}</Label>
                <Input value={model} onChange={(e) => setModel(e.target.value)} />
              </div>
            </>
          )}
          {providerType === "local_cli" && (
            <>
              <div className="grid gap-2">
                <Label>{t("settings.cliType")}</Label>
                <Select
                  value={model}
                  onValueChange={(v) => {
                    setModel(v);
                    const detected = localCLIs.find((c) => c.type === v);
                    setApiBase(detected ? detected.path : "");
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="claude">Claude Code</SelectItem>
                    <SelectItem value="codex">Codex</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>{t("settings.cliPath")}</Label>
                <Input
                  value={apiBase}
                  onChange={(e) => setApiBase(e.target.value)}
                  placeholder={localCLIs.find((c) => c.type === model)?.path || t("settings.cliPathHint")}
                />
                <p className="text-xs text-muted-foreground">{t("settings.cliPathHint")}</p>
              </div>
              {localCLIs.length > 0 && (
                <div className="text-sm text-muted-foreground">
                  {t("settings.detectedCLIs")}: {localCLIs.map((c) => `${c.name} (${c.path})`).join(", ")}
                </div>
              )}
            </>
          )}
          <Button onClick={handleSaveAI} className="gap-1">
            {saved ? <Check className="h-4 w-4" /> : null}
            {saved ? t("settings.saved") : t("action.save")}
          </Button>
        </CardContent>
      </Card>
      <IntegrationSection />
    </>
  );
}
