import type * as React from "react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ProfileSection } from "@/components/settings/ProfileSection";
import { AppearanceSection } from "@/components/settings/AppearanceSection";
import { ConnectionSection } from "@/components/settings/ConnectionSection";
import { NotificationSection } from "@/components/settings/NotificationSection";
import { AccountSection } from "@/components/settings/AccountSection";

export interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Modal settings hub with one tab per configuration area. */
export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps): React.JSX.Element {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="border-b border-border px-6 py-4">
          <DialogTitle>Settings</DialogTitle>
          <DialogDescription>
            Manage your Cordis account and how the app behaves.
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="profile" className="flex min-h-0 flex-1 flex-col">
          <TabsList className="mx-6 mt-4 grid grid-cols-5">
            <TabsTrigger value="profile">Profile</TabsTrigger>
            <TabsTrigger value="appearance">Appearance</TabsTrigger>
            <TabsTrigger value="connection">Connection</TabsTrigger>
            <TabsTrigger value="notifications">Notifications</TabsTrigger>
            <TabsTrigger value="account">Account</TabsTrigger>
          </TabsList>

          <div className="cordis-scroll min-h-0 flex-1 overflow-y-auto px-6 pb-6">
            <TabsContent value="profile">
              <ProfileSection />
            </TabsContent>
            <TabsContent value="appearance">
              <AppearanceSection />
            </TabsContent>
            <TabsContent value="connection">
              <ConnectionSection />
            </TabsContent>
            <TabsContent value="notifications">
              <NotificationSection />
            </TabsContent>
            <TabsContent value="account">
              <AccountSection />
            </TabsContent>
          </div>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}