package ws

import "sync"

// registry maps user ids to their open sockets and channel ids to the sockets
// that joined them. A single mutex guards both maps so a socket is never
// written to twice by two goroutines.
type registry struct {
	mu      sync.RWMutex
	byUser  map[string]map[*client]struct{}
	byChan  map[string]map[*client]struct{}
	joins   map[*client]map[string]struct{}
}

func newRegistry() *registry {
	return &registry{
		byUser: make(map[string]map[*client]struct{}),
		byChan: make(map[string]map[*client]struct{}),
		joins:  make(map[*client]map[string]struct{}),
	}
}

// add registers a socket under its user.
func (r *registry) add(c *client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sockets, ok := r.byUser[c.userID]
	if !ok {
		sockets = make(map[*client]struct{})
		r.byUser[c.userID] = sockets
	}
	sockets[c] = struct{}{}
	r.joins[c] = make(map[string]struct{})
}

// remove drops a socket from every index it appears in.
func (r *registry) remove(c *client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sockets, ok := r.byUser[c.userID]; ok {
		delete(sockets, c)
		if len(sockets) == 0 {
			delete(r.byUser, c.userID)
		}
	}
	for channelID := range r.joins[c] {
		if sockets, ok := r.byChan[channelID]; ok {
			delete(sockets, c)
			if len(sockets) == 0 {
				delete(r.byChan, channelID)
			}
		}
	}
	delete(r.joins, c)
}

// join subscribes a socket to a channel or direct message conversation.
func (r *registry) join(c *client, channelID string) {
	if channelID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	sockets, ok := r.byChan[channelID]
	if !ok {
		sockets = make(map[*client]struct{})
		r.byChan[channelID] = sockets
	}
	sockets[c] = struct{}{}
	if _, ok := r.joins[c]; !ok {
		r.joins[c] = make(map[string]struct{})
	}
	r.joins[c][channelID] = struct{}{}
}

// leave removes a socket from one channel, used when a client stops typing.
func (r *registry) leave(c *client, channelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sockets, ok := r.byChan[channelID]; ok {
		delete(sockets, c)
		if len(sockets) == 0 {
			delete(r.byChan, channelID)
		}
	}
	if joined, ok := r.joins[c]; ok {
		delete(joined, channelID)
	}
}

// userClients returns a snapshot of the sockets of one user.
func (r *registry) userClients(userID string) []*client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sockets := r.byUser[userID]
	out := make([]*client, 0, len(sockets))
	for c := range sockets {
		out = append(out, c)
	}
	return out
}

// channelClients returns a snapshot of the sockets joined to a destination.
func (r *registry) channelClients(channelID string) []*client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sockets := r.byChan[channelID]
	out := make([]*client, 0, len(sockets))
	for c := range sockets {
		out = append(out, c)
	}
	return out
}

// onlineUsers lists every user id with at least one open socket.
func (r *registry) onlineUsers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byUser))
	for id := range r.byUser {
		out = append(out, id)
	}
	return out
}

// socketCount reports how many sockets a user currently holds.
func (r *registry) socketCount(userID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byUser[userID])
}