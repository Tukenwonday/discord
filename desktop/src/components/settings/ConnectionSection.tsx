import { useEffect, useState, type React } from "react";
import { Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useSettingsStore } from "@/stores/settings.store";
import { useAuthStore } from "@/stores/auth.store";
import { socket, resolveWsUrl } from "@/lib/ws/client";
import type { ConnectionState } from "@/lib/ws/events";

/** Editable API and WebSocket endpoints, persisted to local storage. */
export function ConnectionSection(): React.JSX.Element {
  const apiBaseUrl = useSettingsStore((state) => state.apiBaseUrl);
  const wsUrl = useSettingsStore((state) => state.wsUrl);
  const setApiBaseUrl = useSettingsStore((state) => state.setApiBaseUrl);
  const setWsUrl = useSettingsStore((state) => state.setWsUrl);
  const accessToken = useAuthStore((state) => state.accessToken);

  const [apiDraft, setApiDraft] = useState(apiBaseUrl);
  const [wsDraft, setWsDraft] = useState(wsUrl);
  const [status, setStatus] = useState<ConnectionState>(socket.getState());

  // Follow reconnects so the indicator reflects the live socket state.
  useEffect(() => socket.onStatus(setStatus), []);

  const dirty = apiDraft !== apiBaseUrl || wsDraft !== wsUrl;

  function save(): void {
    setApiBaseUrl(apiDraft);
    setWsUrl(wsDraft);
    // A changed endpoint needs a fresh connection to take effect.
    if (accessToken) socket.connect(accessToken, wsDraft || resolveWsUrl());
  }

  return (
    <div className="space-y-6 py-4">
      <div className="space-y-2">
        <Label htmlFor="api-base-url">API base URL</Label>
        <Input
          id="api-base-url"
          value={apiDraft}
          onChange={(event) => setApiDraft(event.target.value)}
          placeholder="http://127.0.0.1:8080"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="ws-url">WebSocket URL</Label>
        <Input
          id="ws-url"
          value={wsDraft}
          onChange={(event) => setWsDraft(event.target.value)}
          placeholder="ws://127.0.0.1:8080/ws"
        />
      </div>

      <Separator />

      <div className="flex items-center gap-3">
        <Button onClick={save} disabled={!dirty}>
          <Save className="mr-2 h-4 w-4" />
          Save and reconnect
        </Button>
        <span className="text-xs text-muted-foreground">
          Socket: {status}
          {dirty ? " · unsaved changes" : ""}
        </span>
      </div>

      <Alert>
        <AlertDescription className="text-xs">
          These values are stored on this device only and override the values
          bundled at build time.
        </AlertDescription>
      </Alert>
    </div>
  );
}