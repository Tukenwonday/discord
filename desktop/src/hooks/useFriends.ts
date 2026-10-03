import { useCallback, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as friendsApi from "@/lib/api/friends";
import type { Friend } from "@/types";

export interface UseFriendsResult {
  friends: Friend[];
  incoming: Friend[];
  outgoing: Friend[];
  isLoading: boolean;
  error: string | null;
  add: (userId: string) => void;
  accept: (friendId: string) => void;
  remove: (friendId: string) => void;
}

export function useFriends(): UseFriendsResult {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["friends"],
    queryFn: () => friendsApi.listFriends(),
    staleTime: 30_000,
  });

  // Every mutation refetches the trio so both sides stay consistent.
  const invalidate = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["friends"] });
  }, [queryClient]);

  const addMutation = useMutation({
    mutationFn: (userId: string) => friendsApi.addFriend(userId),
    onSuccess: invalidate,
  });

  const acceptMutation = useMutation({
    mutationFn: (friendId: string) => friendsApi.acceptFriend(friendId),
    onSuccess: invalidate,
  });

  const removeMutation = useMutation({
    mutationFn: (friendId: string) => friendsApi.removeFriend(friendId),
    onSuccess: invalidate,
  });

  const actions = useMemo(
    () => ({
      add: (userId: string) => addMutation.mutate(userId),
      accept: (friendId: string) => acceptMutation.mutate(friendId),
      remove: (friendId: string) => removeMutation.mutate(friendId),
    }),
    [addMutation, acceptMutation, removeMutation],
  );

  return {
    friends: query.data?.friends ?? [],
    incoming: query.data?.incoming ?? [],
    outgoing: query.data?.outgoing ?? [],
    isLoading: query.isLoading,
    error: query.error?.message ?? null,
    ...actions,
  };
}

/** Invalidates the friends query after a socket friend event arrives. */
export function useFriendsRefetch(): () => void {
  const queryClient = useQueryClient();
  return useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["friends"] });
  }, [queryClient]);
}