import { useEffect, useState, type React } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { updateMe } from "@/lib/api/users";
import { useAuthStore } from "@/stores/auth.store";
import { avatarHue, errorMessage, initials } from "@/lib/utils";

/** Display name, bio and avatar, persisted through PATCH /users/me. */
export function ProfileSection(): React.JSX.Element {
  const user = useAuthStore((state) => state.user);
  const setUser = useAuthStore((state) => state.setUser);

  const [displayName, setDisplayName] = useState(user?.displayName ?? "");
  const [bio, setBio] = useState(user?.bio ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  // Re-seed when a different account is loaded.
  useEffect(() => {
    setDisplayName(user?.displayName ?? "");
    setBio(user?.bio ?? "");
  }, [user?.id, user?.displayName, user?.bio]);

  async function save(): Promise<void> {
    setSaving(true);
    setError(null);
    setSaved(false);

    try {
      const updated = await updateMe({ displayName: displayName.trim(), bio: bio.trim() });
      setUser(updated);
      setSaved(true);
    } catch (saveError) {
      setError(errorMessage(saveError, "Could not save your profile"));
    } finally {
      setSaving(false);
    }
  }

  if (!user) {
    return <p className="py-6 text-sm text-muted-foreground">Loading your profile…</p>;
  }

  return (
    <div className="space-y-6 py-4">
      <div className="flex items-center gap-4">
        <Avatar className="h-16 w-16">
          {user.avatarUrl ? <AvatarImage src={user.avatarUrl} alt="" /> : null}
          <AvatarFallback
            style={{ backgroundColor: `hsl(${avatarHue(user.id)} 45% 35%)` }}
            className="text-lg font-semibold text-white"
          >
            {initials(user.displayName || user.username)}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold">{user.displayName}</p>
          <p className="truncate text-xs text-muted-foreground">@{user.username}</p>
          <p className="truncate text-xs text-muted-foreground">{user.email}</p>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="profile-display-name">Display name</Label>
        <Input
          id="profile-display-name"
          value={displayName}
          maxLength={64}
          onChange={(event) => setDisplayName(event.target.value)}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="profile-bio">About me</Label>
        <Textarea
          id="profile-bio"
          value={bio}
          onChange={(event) => setBio(event.target.value)}
          placeholder="Tell people a little about yourself"
          className="min-h-[80px]"
        />
      </div>

      {error ? (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="flex items-center gap-3">
        <Button onClick={() => void save()} disabled={saving}>
          {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          Save changes
        </Button>
        {saved ? (
          <span className="text-xs text-cordis-online">Profile saved</span>
        ) : null}
      </div>
    </div>
  );
}