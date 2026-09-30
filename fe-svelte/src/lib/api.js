// Lightmail API client

export async function request(url, options = {}) {
  const defaultHeaders = {
    'Accept': 'application/json',
  };

  if (!(options.body instanceof FormData)) {
    defaultHeaders['Content-Type'] = 'application/json';
  }

  const res = await fetch(url, {
    ...options,
    headers: {
      ...defaultHeaders,
      ...options.headers,
    },
    credentials: 'same-origin',
  });

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${res.statusText}`);
  }

  const data = await res.json();
  if (data.errorNo !== 0) {
    throw new Error(data.errorMsg || 'Unknown server error');
  }
  return data.data;
}

export function normalizeEmailItem(item) {
  if (!item) return {};

  const senderName = item.sender?.Name || item.from_name || '';
  const senderAddress = item.sender?.EmailAddress || item.from_address || '';
  const displaySender = senderName || senderAddress || 'Unknown Sender';

  let toDisplay = '';
  if (Array.isArray(item.to)) {
    toDisplay = item.to.map((r) => r.Name || r.EmailAddress || '').filter(Boolean).join(', ');
  } else if (typeof item.to === 'string') {
    try {
      const parsed = JSON.parse(item.to);
      if (Array.isArray(parsed)) {
        toDisplay = parsed.map((r) => r.Name || r.EmailAddress || '').filter(Boolean).join(', ');
      } else {
        toDisplay = item.to;
      }
    } catch (_) {
      toDisplay = item.to;
    }
  }

  const subject = item.title || item.subject || '(no subject)';
  const snippet = item.desc !== undefined ? item.desc : (item.text || '');
  const date = item.datetime || item.send_date || item.create_time || '';
  const isRead = item.is_read === true || item.is_read === 1;

  return {
    ...item,
    id: item.id,
    subject,
    title: subject,
    text: item.text !== undefined ? item.text : snippet,
    desc: snippet,
    from_name: senderName,
    from_address: senderAddress,
    display_sender: displaySender,
    to: toDisplay || item.to,
    to_display: toDisplay || 'me',
    send_date: date,
    datetime: date,
    is_read: isRead,
    attachments: item.attachments || [],
  };
}

// Authentication
export const apiLogin = (account, password) =>
  request('/api/login', {
    method: 'POST',
    body: JSON.stringify({ account, password }),
  });

export const apiLogout = () =>
  request('/api/logout', { method: 'GET' });

export const apiGetUserInfo = () =>
  request('/api/user/info', { method: 'GET' });

// Email Management
// Maps frontend folder names to server SearchTag values.
// Server SearchTag: type (-1=any, 0=received, 1=sent), status (-1=any, 3=deleted, 4=drafts, 5=junk), group_id (-1=any, -2=inbox special)
function folderToTag(group) {
  switch (group) {
    case 'inbox':   return { type: 0,  status: -1, group_id: -1 }; // received, normal status, default inbox group
    case 'sent':    return { type: 1,  status: -1, group_id: -1 }; // sent type
    case 'drafts':  return { type: -1, status: 4,  group_id: -1 }; // draft status
    case 'trash':   return { type: -1, status: 3,  group_id: -1 }; // deleted status
    case 'spam':    return { type: -1, status: 5,  group_id: -1 }; // junk status
    default:        return { type: -1, status: -1, group_id: -1 };
  }
}

export const apiGetEmailList = async ({ group = 'inbox', page = 1, pageSize = 25, search = '' }) => {
  const tag = folderToTag(group);
  const data = await request('/api/email/list', {
    method: 'POST',
    body: JSON.stringify({
      tag: JSON.stringify(tag),
      current_page: page,
      page_size: pageSize,
      keyword: search,
    }),
  });
  const list = (data.list || []).map(normalizeEmailItem);
  const totalPages = data.total_page || 0;
  return {
    ...data,
    list,
    total: data.total || (totalPages > 0 ? totalPages * pageSize : list.length),
  };
};

export const apiGetEmailDetail = async (id) => {
  const data = await request('/api/email/detail', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
  return normalizeEmailItem(data);
};

export const apiMarkRead = (ids, isRead = true) =>
  request('/api/email/read', {
    method: 'POST',
    body: JSON.stringify({ ids: Array.isArray(ids) ? ids : [ids], isRead }),
  });

export const apiDeleteEmail = (ids) =>
  request('/api/email/del', {
    method: 'POST',
    body: JSON.stringify({ ids: Array.isArray(ids) ? ids : [ids] }),
  });

export const apiMoveEmail = (ids, targetGroup) =>
  request('/api/email/move', {
    method: 'POST',
    body: JSON.stringify({
      ids: Array.isArray(ids) ? ids : [ids],
      group: targetGroup,
    }),
  });

// apiSendEmail sends a JSON body to the server.
// body should be a JSON string: { to, cc, bcc, subject, html, text, attrs }
// where to/cc/bcc are [{name, email}] arrays and attrs are [{name, data}] base64 attachments.
export const apiSendEmail = (body) =>
  request('/api/email/send', {
    method: 'POST',
    body: typeof body === 'string' ? body : JSON.stringify(body),
  });

// User Management (Admin)
export const normalizeUserItem = (item) => {
  if (!item) return {};
  return {
    ...item,
    id: item.id !== undefined ? item.id : item.ID,
    account: item.account || item.Account || '',
    name: item.name || item.Name || '',
    is_admin: item.is_admin !== undefined ? item.is_admin : (item.IsAdmin !== undefined ? item.IsAdmin : 0),
    disabled: item.disabled !== undefined ? item.disabled : (item.Disabled !== undefined ? item.Disabled : 0),
    gender: item.gender !== undefined ? item.gender : (item.Gender || ''),
  };
};

export const apiGetUserList = async (page = 1, pageSize = 20) => {
  const data = await request('/api/user/list', {
    method: 'POST',
    body: JSON.stringify({ currentPage: page, pageSize }),
  });
  const list = (data.list || []).map(normalizeUserItem);
  return {
    ...data,
    list,
  };
};

export const apiCreateUser = ({ account, domain, username, password, isAdmin = 0, gender = '' }) =>
  request('/api/user/create', {
    method: 'POST',
    body: JSON.stringify({
      account,
      domain,
      username,
      password,
      is_admin: isAdmin,
      gender,
    }),
  });

export const apiEditUser = (data) =>
  request('/api/user/edit', {
    method: 'POST',
    body: JSON.stringify(data),
  });

export const apiChangePassword = (oldPassword, newPassword) =>
  request('/api/settings/modify_password', {
    method: 'POST',
    body: JSON.stringify({
      oldPassword,
      password: newPassword,
    }),
  });

// Domain Management (Admin)
export const apiGetDomainList = () =>
  request('/api/domain/list', { method: 'GET' });

export const apiAddDomain = (domain) =>
  request('/api/domain/add', {
    method: 'POST',
    body: JSON.stringify({ domain }),
  });

export const apiDeleteDomain = (domain) =>
  request('/api/domain/del', {
    method: 'POST',
    body: JSON.stringify({ domain }),
  });

export const apiCheckDomainDNS = (domain) =>
  request(`/api/domain/check?domain=${encodeURIComponent(domain)}`, {
    method: 'GET',
  });

export const apiApplyCloudflare = (token, domain, email = '', zoneId = '') =>
  request('/api/domain/cloudflare', {
    method: 'POST',
    body: JSON.stringify({ token, email, domain, zone_id: zoneId }),
  });

export const apiGetCloudflareSettings = () =>
  request('/api/settings/cloudflare/get', { method: 'GET' });

export const apiSaveCloudflareSettings = (token, email = '') =>
  request('/api/settings/cloudflare/save', {
    method: 'POST',
    body: JSON.stringify({ token, email }),
  });

// Folders / Groups
export const apiGetGroups = () =>
  request('/api/group/list', { method: 'GET' });

export const apiAddGroup = (name) =>
  request('/api/group/add', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });

export const apiDeleteGroup = (id) =>
  request('/api/group/del', {
    method: 'POST',
    body: JSON.stringify({ id }),
  });
