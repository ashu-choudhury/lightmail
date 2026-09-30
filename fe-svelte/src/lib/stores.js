import { writable } from 'svelte/store';

// Authentication & user state
const storedUser = localStorage.getItem('lightmail_user');
export const currentUser = writable(storedUser ? JSON.parse(storedUser) : null);

currentUser.subscribe((user) => {
  if (user) {
    localStorage.setItem('lightmail_user', JSON.stringify(user));
  } else {
    localStorage.removeItem('lightmail_user');
  }
});

// Navigation state
export const currentFolder = writable('inbox'); // 'inbox', 'sent', 'drafts', 'trash', 'spam', 'settings', 'admin-users', 'admin-domains'
export const selectedEmailId = writable(null);
export const searchQuery = writable('');

// Sidebar state
export const sidebarOpen = writable(true);

// Compose widget state
export const composeState = writable({
  open: false,
  minimized: false,
  to: '',
  cc: '',
  bcc: '',
  subject: '',
  body: '',
  files: [],
  replyToId: null,
});

// Toast notifications
export const toasts = writable([]);

export function showToast(message, type = 'info', timeout = 4000) {
  const id = Date.now() + Math.random();
  toasts.update((all) => [...all, { id, message, type }]);
  setTimeout(() => {
    toasts.update((all) => all.filter((t) => t.id !== id));
  }, timeout);
}

// Mail list refresh trigger
export const mailRefreshTrigger = writable(0);
export function triggerMailRefresh() {
  mailRefreshTrigger.update((n) => n + 1);
}
