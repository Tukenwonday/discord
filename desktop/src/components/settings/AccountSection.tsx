import { useEffect, useState, type React } from "react";
import { LogOut, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { useAuthStore } from "@/stores/auth.store";
import { useSettingsStore } from "@/stores/settings.store";
import { socket } from "@/lib/ws/client";
import { appVersion, deviceId, isTauri } from "@/lib/tauri/desktop";
import type { UserStatus } from "@/types";

const STATUSES: UserStatus[] = ["online", "idle", "dnd", "invisible"];

/** Presence, session information and the sign-out / reset controls. */
export function AccountSection(): React.JSX.Element {
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const status = useSettingsStore((state) => state.status);
  const setStatus = useSettingsStore((state) => state.setStatus);

  const [logoutOpen, setLogoutOpen] = useState(false);
  const [version, setVersion] = useState<string | null>(null);
  const [device, setDevice] = useState<string | null>(null);

  useEffect(() => {
    void appVersion().then(setVersion);
    void deviceId().then(setDevice);
  }, []);

  return (
    <div className="space-y-6 py-4">
      <div className="space-y-2">
        <Label htmlFor="account-status">Presence</Label>
        <Select value={status} onValueChange={(value) => setStatus(value as UserStatus)}>
          <SelectTrigger id="account-status" className="max-w-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {STATUSES.map((option) => (
              <SelectItem key={option} value={option}>
                {option === "dnd" ? "Do Not Disturb" : option[0].toUpperCase() + option.slice(1)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Separator />

      <div className="space-y-1 text-xs text-muted-foreground">
        <p>
          Signed in as <span className="text-foreground">@{user?.username}</span>
        </p>
        <p>Cordis {version ?? "…"}</p>
        {isTauri() ? <p>Device {device ?? "…"}</p> : <p>Running in a browser</p>}
      </div>

      <Separator />

      <div className="space-y-2">
        <Button variant="secondary" onClick={() => setLogoutOpen(true)}>
          <LogOut className="mr-2 h-4 w-4" />
          Sign out
        </Button>

        <Button
          variant="ghost"
          onClick={() => {
            socket.close();
            void logout();
          }}
          className="text-destructive hover:bg-destructive/10 hover:text-destructive"
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Sign out and disconnect
        </Button>
      </div>

      <ConfirmDialog
        open={logoutOpen}
        onOpenChange={setLogoutOpen}
        title="Sign out of Cordis?"
        description="You will need your password to sign back in."
        confirmLabel="Sign out"
        onConfirm={() => {
          socket.close();
          void logout();
        }}
      />
    </div>
  );
}