import { useState, type React } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { searchUsers } from "@/lib/api/users";
import { createDm } from "@/lib/api/dms";
import { errorMessage, avatarHue, initials } from "@/lib/utils";
import type { User } from "@/types";

export interface NewDmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onOpenDm: (dmId: string) => void;
}

/** Searches users and opens, or creates, the conversation with them. */
export function NewDmDialog({
  open,
  onOpenChange,
  onOpenDm,
}: NewDmDialogProps): React.JSX.Element {
  const [term, setTerm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const query = useQuery({
    queryKey: ["user-search", term],
    queryFn: () => searchUsers(term, 20),
    enabled: open && term.trim().length > 0,
  });

  async function start(user: User): Promise<void> {
    setBusyId(user.id);
    setError(null);
    try {
      const dm = await createDm(user.id);
      onOpenDm(dm.id);
      onOpenChange(false);
      setTerm("");
    } catch (dmError) {
      setError(errorMessage(dmError, "Could not open the conversation"));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Select a friend</DialogTitle>
          <DialogDescription>
            Start a direct message by searching for a username.
          </DialogDescription>
        </DialogHeader>

        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            autoFocus
            value={term}
            onChange={(event) => setTerm(event.target.value)}
            placeholder="Search for people"
            className="pl-8"
          />
        </div>

        <div className="cordis-scroll max-h-72 space-y-px overflow-y-auto">
          {query.data?.items.map((user) => (
            <button
              key={user.id}
              type="button"
              disabled={busyId === user.id}
              onClick={() => void start(user)}
              className="flex w-full items-center gap-3 rounded px-2 py-2 text-left transition hover:bg-accent disabled:opacity-50"
            >
              <Avatar className="h-8 w-8">
                {user.avatarUrl ? <AvatarImage src={user.avatarUrl} alt="" /> : null}
                <AvatarFallback
                  style={{ backgroundColor: `hsl(${avatarHue(user.id)} 45% 35%)` }}
                  className="text-[10px] font-semibold text-white"
                >
                  {initials(user.displayName || user.username)}
                </AvatarFallback>
              </Avatar>

              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium">
                  {user.displayName || user.username}
                </span>
                <span className="block truncate text-xs text-muted-foreground">
                  @{user.username}
                </span>
              </span>
            </button>
          ))}

          {term.trim() && !query.isLoading && (query.data?.items.length ?? 0) === 0 ? (
            <p className="px-2 py-4 text-center text-sm text-muted-foreground">
              No people match “{term.trim()}”
            </p>
          ) : null}

          {term.trim().length === 0 ? (
            <p className="px-2 py-4 text-center text-sm text-muted-foreground">
              Type a name to search
            </p>
          ) : null}
        </div>

        {error ? <p className="text-xs text-destructive">{error}</p> : null}

        <Button variant="secondary" onClick={() => onOpenChange(false)}>
          Cancel
        </Button>
      </DialogContent>
    </Dialog>
  );
}