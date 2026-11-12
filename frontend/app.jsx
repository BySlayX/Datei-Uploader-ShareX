import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';

const demoFiles = [
  { id: 'demo-1', name: 'workspace-preview.png', size: 2840000, contentType: 'image/png', url: 'https://files.example.com/uploads/workspace-preview.png', uploadedAt: '2026-10-19T08:48:00Z', demo: true },
  { id: 'demo-2', name: 'release-notes.pdf', size: 812000, contentType: 'application/pdf', url: 'https://files.example.com/uploads/release-notes.pdf', uploadedAt: '2026-10-18T16:21:00Z', demo: true },
  { id: 'demo-3', name: 'screen-recording.mp4', size: 18400000, contentType: 'video/mp4', url: 'https://files.example.com/uploads/screen-recording.mp4', uploadedAt: '2026-10-18T10:05:00Z', demo: true },
];

const demoUser = { id: 'demo-admin', email: import.meta.env.VITE_DEMO_EMAIL || '', name: import.meta.env.VITE_DEMO_NAME || 'Demo', role: 'admin', demo: true };
const demoPassword = import.meta.env.VITE_DEMO_PASSWORD || '';
const demoModeEnabled = import.meta.env.DEV || import.meta.env.VITE_DEMO_MODE === 'true';

function readPreference(key, fallback) {
  try {
    return localStorage.getItem(key) || fallback;
  } catch {
    return fallback;
  }
}

function writePreference(key, value) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Browser privacy settings may disable local storage; preferences are optional.
  }
}

async function copyText(value) {
  const text = String(value || '');
  if (!text) return false;
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {
    // HTTP pages or browser permissions may block the modern Clipboard API.
  }
  try {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.setAttribute('readonly', '');
    textarea.setAttribute('aria-hidden', 'true');
    textarea.style.position = 'fixed';
    textarea.style.top = '0';
    textarea.style.left = '0';
    textarea.style.opacity = '0';
    textarea.style.pointerEvents = 'none';
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    textarea.setSelectionRange(0, textarea.value.length);
    const copied = document.execCommand('copy');
    textarea.remove();
    return copied;
  } catch {
    return false;
  }
}

function Icon({ name, size = 18 }) {
  const paths = {
    user: <><circle cx="12" cy="8" r="3.5" /><path d="M5 20a7 7 0 0 1 14 0" /></>,
    grid: <><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /></>,
    upload: <><path d="M12 16V4" /><path d="m7 9 5-5 5 5" /><path d="M5 20h14" /></>,
    download: <><path d="M12 4v12" /><path d="m7 11 5 5 5-5" /><path d="M5 20h14" /></>,
    image: <><rect x="3" y="4" width="18" height="16" rx="2" /><circle cx="8.5" cy="9" r="1.5" /><path d="m4 17 4.5-4.5 3 3 2-2 4.5 4.5" /></>,
    settings: <><path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z" /><path d="m19.4 15 .7 1.8-1.8 1.5-1.5-1a7.8 7.8 0 0 1-1.8.8l-.3 1.8h-2.4l-.3-1.8a7.8 7.8 0 0 1-1.8-.8l-1.5 1-1.8-1.5.7-1.8a7.4 7.4 0 0 1-.8-1.8L5 12.9v-2.4l1.8-.3a7.4 7.4 0 0 1 .8-1.8l-.7-1.8L8.7 5l1.5 1a7.8 7.8 0 0 1 1.8-.8l.3-1.8h2.4l.3 1.8a7.8 7.8 0 0 1 1.8.8l1.5-1 1.8 1.5-.7 1.8a7.4 7.4 0 0 1 .8 1.8l1.8.3v2.4l-1.8.3a7.4 7.4 0 0 1-.8 1.8Z" /></>,
    settingsHeader: <><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0L6.2 6.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.09a2 2 0 0 1 1 1.74v.5a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.09a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2Z" /><circle cx="12" cy="12" r="3" /></>,
    copy: <><rect x="8" y="8" width="11" height="11" rx="2" /><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2" /></>,
    trash: <><path d="M4 7h16" /><path d="M10 11v6M14 11v6" /><path d="M6 7l1 13h10l1-13M9 7V4h6v3" /></>,
    search: <><circle cx="10.8" cy="10.8" r="6.8" /><path d="m16 16 5 5" /></>,
    more: <><circle cx="5" cy="12" r="1" fill="currentColor" /><circle cx="12" cy="12" r="1" fill="currentColor" /><circle cx="19" cy="12" r="1" fill="currentColor" /></>,
    chevron: <path d="m6 9 6 6 6-6" />,
    chevronUp: <path d="m18 15-6-6-6 6" />,
    file: <><path d="M15 2H8a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7Z" /><path d="M14 2v5h5" /><path d="M4 6v12a2 2 0 0 0 2 2h10" /><path d="M9 12h6M9 15h4" /></>,
    video: <><rect x="2" y="6" width="14" height="12" rx="2.2" /><path d="m16 9.2 6-3.4v12.4l-6-3.4Z" fill="currentColor" stroke="none" /></>,
    pdf: <><path d="M6 3h8l4 4v14H6z" /><path d="M14 3v5h5" /><path d="M8 13h8M8 17h5" /></>,
    audio: <><path d="M9 18V6l11-2v12" /><circle cx="7" cy="18" r="2.4" /><circle cx="18" cy="16" r="2.4" /></>,
    archive: <><path d="M4 8h16v12H4z" /><path d="M4 8V5h16v3" /><path d="M10 5v15" /><path d="M12 12h3" /></>,
    textFile: <><path d="M6 3h8l4 4v14H6z" /><path d="M14 3v5h5" /><path d="M8 12h8M8 16h6" /></>,
    link: <><path d="M10 13a5 5 0 0 0 7.07.07l2-2a5 5 0 0 0-7.07-7.07l-1.14 1.14" /><path d="M14 11a5 5 0 0 0-7.07-.07l-2 2A5 5 0 0 0 7 20l1.14-1.14" /></>,
    lock: <><rect x="5" y="10" width="14" height="10" rx="2" /><path d="M8 10V7a4 4 0 0 1 8 0v3" /><path d="M12 14v2" /></>,
    check: <path d="m5 12 4 4L19 6" />,
    shield: <><path d="M12 3 20 6v5c0 5-3.5 8.5-8 10-4.5-1.5-8-5-8-10V6l8-3Z" /><path d="m8.5 12 2.2 2.2 4.8-5" /></>,
    users: <><path d="M16 20v-1.4a3.6 3.6 0 0 0-3.6-3.6H7.6A3.6 3.6 0 0 0 4 18.6V20" /><circle cx="10" cy="7" r="3" /><path d="M17 11a3 3 0 1 0-1.2-5.8M20 20v-1.4a3.6 3.6 0 0 0-2.6-3.45" /></>,
    activity: <><path d="M3 12h4l2-7 4 14 2-7h6" /></>,
    database: <><ellipse cx="12" cy="5" rx="7" ry="3" /><path d="M5 5v7c0 1.7 3.1 3 7 3s7-1.3 7-3V5" /><path d="M5 12v7c0 1.7 3.1 3 7 3s7-1.3 7-3v-7" /></>,
    logout: <><path d="M10 5H5v14h5" /><path d="m15 8 4 4-4 4" /><path d="M19 12H9" /></>,
    arrowUpRight: <><path d="M7 17 17 7" /><path d="M8 7h9v9" /></>,
    arrowLeft: <><path d="M19 12H5" /><path d="m12 19-7-7 7-7" /></>,
    eye: <><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" /><circle cx="12" cy="12" r="3" /></>,
    eyeOff: <><path d="M10.6 10.6a2 2 0 0 0 2.8 2.8" /><path d="M16.4 16.4A9.8 9.8 0 0 1 12 19c-6.5 0-10-7-10-7a17.4 17.4 0 0 1 5.1-5.4" /><path d="M9.9 4.2A9.9 9.9 0 0 1 12 5c6.5 0 10 7 10 7a17.6 17.6 0 0 1-2.2 3.1" /><path d="m2 2 20 20" /></>,
  };
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name] || paths.file}</svg>;
}

function Brand({ className = '', cacheKey = '' }) {
  const [revision, setRevision] = useState('');
  useEffect(() => {
    const refreshBranding = () => setRevision(String(Date.now()));
    window.addEventListener('branding-changed', refreshBranding);
    return () => window.removeEventListener('branding-changed', refreshBranding);
  }, []);
  const logoSource = cacheKey || revision ? `/api/branding/logo?v=${cacheKey || revision}` : '/api/branding/logo';
  return <div className={`brand ${className}`.trim()}><img className="brand-logo" src={logoSource} onError={(event) => { event.currentTarget.onerror = null; event.currentTarget.src = '/drop-logo.png'; }} alt="ShareX Uploader" /><div><strong>SHAREX</strong><small>UPLOADER</small></div></div>;
}

function formatSize(bytes) {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = Number(bytes);
  let index = 0;
  // Switch early (at 1000) so 1022.9 MB becomes ~1.0 GB instead of staying in MB.
  while (value >= 1000 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index ? 1 : 0)} ${units[index]}`;
}

function formatSpeed(bytesPerSecond) {
  if (!Number(bytesPerSecond) || bytesPerSecond <= 0) return '';
  return `${formatSize(bytesPerSecond)}/s`;
}

function uploadProgressCaption(progress) {
  const label = progress?.kind === 'download' ? 'Download' : 'Datei';
  const parts = [`${label} ${progress.current} von ${progress.total}`];
  const speed = formatSpeed(progress.speed);
  if (speed) parts.push(speed);
  return parts.join(' · ');
}

function formatDate(date) {
  return new Intl.DateTimeFormat('de-DE', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(date));
}

function fileKind(file) {
  const type = String(file?.contentType || '').toLowerCase();
  const name = String(file?.name || '').toLowerCase();
  const ext = name.includes('.') ? name.slice(name.lastIndexOf('.') + 1) : '';
  if (type.startsWith('image/') || /^(png|jpe?g|gif|webp|bmp|svg|ico|heic|tiff?)$/.test(ext)) return 'image';
  if (type.startsWith('video/') || /^(mp4|webm|mkv|mov|avi|m4v)$/.test(ext)) return 'video';
  if (type.startsWith('audio/') || /^(mp3|wav|ogg|flac|m4a|aac)$/.test(ext)) return 'audio';
  if (type === 'application/pdf' || ext === 'pdf') return 'pdf';
  if (type.includes('zip') || type.includes('compressed') || type.includes('tar') || /^(zip|rar|7z|tar|gz)$/.test(ext)) return 'archive';
  if (type.startsWith('text/') || type.includes('json') || type.includes('xml') || /^(txt|log|md|json|xml|csv|sxcu)$/.test(ext)) return 'text';
  return 'file';
}

function fileIconName(kind) {
  if (kind === 'image') return 'image';
  if (kind === 'video') return 'video';
  if (kind === 'pdf') return 'pdf';
  if (kind === 'audio') return 'audio';
  if (kind === 'archive') return 'archive';
  if (kind === 'text') return 'textFile';
  return 'file';
}

function typeLabel(file) {
  const kind = fileKind(file);
  if (kind === 'image') return 'IMAGE';
  if (kind === 'video') return 'VIDEO';
  if (kind === 'audio') return 'AUDIO';
  if (kind === 'pdf') return 'PDF';
  if (kind === 'archive') return 'ARCHIV';
  if (kind === 'text') return 'TEXT';
  return 'FILE';
}

function AdminFileTypeMark({ file }) {
  const kind = fileKind(file);
  const [broken, setBroken] = useState(false);
  const thumb = kind === 'image' && !broken && !file.passwordProtected ? (file.url || file.previewUrl) : '';
  return (
    <div className={`admin-file-type kind-${kind}${thumb ? ' has-thumb' : ''}`}>
      {thumb ? <img src={thumb} alt="" onError={() => setBroken(true)} /> : <Icon name={fileIconName(kind)} size={24} />}
    </div>
  );
}

function initials(name = 'User') {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join('').toUpperCase();
}

function downloadShareXConfig(baseUrl, uploadKey) {
  if (!uploadKey) return;
  const root = String(baseUrl || window.location.origin).replace(/\/$/, '');
  const config = {
    Version: '17.0.0',
    Name: 'ShareX Uploader',
    DestinationType: 'ImageUploader, TextUploader, FileUploader',
    RequestMethod: 'POST',
    RequestURL: `${root}/api/upload`,
    Headers: { 'X-API-Key': uploadKey },
    Body: 'MultipartFormData',
    FileFormName: 'file',
    URL: '{json:rawUrl}',
    ThumbnailURL: '{json:rawUrl}',
    ErrorMessage: '{json:error}',
  };
  const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = 'ShareX-Uploader.sxcu';
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

const appRoutes = {
  dashboard: '/dashboard',
  files: '/galerie',
  settings: '/einstellungen',
  admin: '/admin',
};

function viewForPath(pathname) {
  const path = String(pathname || '').replace(/\/$/, '') || '/';
  if (path === '/galerie') return 'files';
  if (path === '/einstellungen') return 'settings';
  if (path === '/admin') return 'admin';
  if (path === '/dashboard' || path === '/') return 'dashboard';
  return null;
}

function isAppPath(pathname) {
  const path = String(pathname || '').replace(/\/$/, '') || '/';
  return path === '/setup'
    || path === '/login'
    || path === '/logout'
    || path === '/forgot-password'
    || path === '/reset-password'
    || viewForPath(path) !== null;
}

function NotFoundPage({ onHome }) {
  return <main className="auth-screen not-found-screen">
    <LoginAmbient />
    <section className="not-found-card">
      <Brand className="auth-brand" />
      <p className="eyebrow">FEHLER 404</p>
      <h1>Seite nicht gefunden.</h1>
      <p className="not-found-copy">Diese Adresse gibt es hier nicht oder sie wurde verschoben.</p>
      <div className="not-found-actions">
        <button className="primary-button" type="button" onClick={onHome}>Zum Dashboard<span>→</span></button>
        <button className="secondary-button" type="button" onClick={() => window.history.back()}>Zurück</button>
      </div>
      <code className="not-found-path">{window.location.pathname}</code>
    </section>
  </main>;
}

function pathForView(view) {
  return appRoutes[view] || appRoutes.dashboard;
}

function App() {
  const inputRef = useRef(null);
  const [files, setFiles] = useState(demoFiles);
  const [query, setQuery] = useState('');
  const [notFound, setNotFound] = useState(() => !isAppPath(window.location.pathname));
  const [active, setActive] = useState(() => {
    const routeView = viewForPath(window.location.pathname);
    if (routeView) return routeView;
    if (!isAppPath(window.location.pathname)) return 'dashboard';
    const savedView = readPreference('drop-active-view', 'dashboard');
    return ['dashboard', 'files', 'settings', 'admin'].includes(savedView) ? savedView : 'dashboard';
  });
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(null);
  const [downloadProgress, setDownloadProgress] = useState(null);
  const [toast, setToast] = useState('');
  const [apiOnline, setApiOnline] = useState(false);
  const [healthStatus, setHealthStatus] = useState(null);
  const [adminOverview, setAdminOverview] = useState(null);
  const [adminUsers, setAdminUsers] = useState(null);
  const [adminRefreshKey, setAdminRefreshKey] = useState(0);
  const [sessionUser, setSessionUser] = useState(null);
  const sessionUserRef = useRef(null);
  useEffect(() => { sessionUserRef.current = sessionUser; }, [sessionUser]);
  const [authLoading, setAuthLoading] = useState(true);
  const [authOffline, setAuthOffline] = useState(false);
  const [loginError, setLoginError] = useState('');
  const [setupStatus, setSetupStatus] = useState(null);
  const [setupCredentials, setSetupCredentials] = useState(null);
  const [authView, setAuthView] = useState('login');
  const [viewMode, setViewMode] = useState(() => readPreference('drop-view', 'grid'));
  const [galleryPage, setGalleryPage] = useState(1);
  const [previewFile, setPreviewFile] = useState(null);
  const [passwordFile, setPasswordFile] = useState(null);
  const [deleteFile, setDeleteFile] = useState(null);
  const [profileMenuOpen, setProfileMenuOpen] = useState(false);
  const profileMenuRef = useRef(null);

  function navigateTo(view, { replace = false } = {}) {
    const nextPath = pathForView(view);
    if (window.location.pathname !== nextPath || window.location.search || window.location.hash) {
      const method = replace ? 'replaceState' : 'pushState';
      window.history[method]({}, '', nextPath);
    }
    setNotFound(false);
    setActive(view);
  }

  function applyPath(path) {
    const normalized = String(path || '').replace(/\/$/, '') || '/';
    const user = sessionUserRef.current;

    if (!user) {
      if (normalized === '/forgot-password') {
        setAuthView('forgot');
        setNotFound(false);
        return;
      }
      if (normalized === '/login' || normalized === '/' || normalized === '/dashboard' || normalized === '/galerie' || normalized === '/einstellungen' || normalized === '/admin' || normalized === '/logout') {
        setAuthView('login');
        setNotFound(false);
        return;
      }
      if (!isAppPath(normalized)) {
        setNotFound(true);
        return;
      }
      setAuthView('login');
      setNotFound(false);
      return;
    }

    if (normalized === '/login' || normalized === '/forgot-password' || normalized === '/logout' || normalized === '/') {
      setNotFound(false);
      setActive('dashboard');
      if (normalized !== '/dashboard') {
        window.history.replaceState({}, '', '/dashboard');
      }
      return;
    }

    if (!isAppPath(normalized)) {
      setNotFound(true);
      return;
    }

    setNotFound(false);
    const routeView = viewForPath(normalized);
    if (routeView) setActive(routeView);
  }

  useEffect(() => {
    function handleHistoryNavigation() {
      applyPath(window.location.pathname);
    }
    window.addEventListener('popstate', handleHistoryNavigation);
    return () => window.removeEventListener('popstate', handleHistoryNavigation);
  }, []);

  useEffect(() => {
    if (authLoading || window.location.pathname === '/reset-password') return;
    applyPath(window.location.pathname);
  }, [sessionUser, authLoading]);


  useEffect(() => {
    writePreference('drop-view', viewMode);
  }, [viewMode]);

  useEffect(() => {
    writePreference('drop-active-view', active);
  }, [active]);

  useEffect(() => {
    if (sessionUser && active === 'admin' && sessionUser.role !== 'admin') navigateTo('dashboard', { replace: true });
  }, [active, sessionUser]);

  useEffect(() => {
    if (!profileMenuOpen) return undefined;
    function closeOnOutsideClick(event) {
      if (!profileMenuRef.current?.contains(event.target)) setProfileMenuOpen(false);
    }
    document.addEventListener('pointerdown', closeOnOutsideClick);
    return () => document.removeEventListener('pointerdown', closeOnOutsideClick);
  }, [profileMenuOpen]);

  useEffect(() => {
    setProfileMenuOpen(false);
  }, [active]);

  useEffect(() => {
    function refreshSessionUser() {
      fetch('/api/auth/me', { credentials: 'include', cache: 'no-store' })
        .then((response) => response.ok ? response.json() : null)
        .then((data) => { if (data?.user) setSessionUser(data.user); })
        .catch(() => {});
    }
    window.addEventListener('sharex-limits-changed', refreshSessionUser);
    return () => window.removeEventListener('sharex-limits-changed', refreshSessionUser);
  }, []);

  useEffect(() => {
    if (setupStatus?.configured !== true) return undefined;
    let mounted = true;
    function refreshHealth() {
      fetch('/api/health', { cache: 'no-store' })
        .then(async (response) => {
          const data = await response.json().catch(() => ({}));
          if (!mounted) return;
          const online = response.ok;
          setApiOnline(online);
          setHealthStatus(online ? {
            api: data.api !== false,
            apiKeyEnabled: data.apiKeyEnabled === true,
            database: data.database === true,
            databaseType: data.databaseType || '',
            uploadReady: data.uploadReady === true,
          } : { api: false, apiKeyEnabled: false, database: false, databaseType: '', uploadReady: false });
        })
        .catch(() => {
          if (!mounted) return;
          setApiOnline(false);
          setHealthStatus({ api: false, apiKeyEnabled: false, database: false, databaseType: '', uploadReady: false });
        });
    }
    refreshHealth();
    const timer = window.setInterval(refreshHealth, 15000);
    return () => { mounted = false; window.clearInterval(timer); };
  }, [setupStatus?.configured]);

  useEffect(() => {
    let mounted = true;
    fetch('/api/setup/status', { cache: 'no-store' }).then((response) => {
      if (!response.ok) throw new Error('setup status unavailable');
      return response.json();
    }).then((data) => {
      if (mounted) setSetupStatus({ configured: data.configured === true });
    }).catch(() => {
      if (mounted) setSetupStatus({ configured: true, unavailable: true });
    });
    return () => { mounted = false; };
  }, []);

  useEffect(() => {
    if (setupStatus?.configured !== true) return undefined;
    let mounted = true;
    fetch('/api/auth/me', { credentials: 'include' }).then((response) => {
      if (response.status === 401) return null;
      if (!response.ok) throw new Error('auth unavailable');
      return response.json();
    }).then((data) => {
      if (!mounted) return;
      if (data?.user) setSessionUser(data.user);
      setAuthOffline(false);
      setAuthLoading(false);
    }).catch(() => {
      if (!mounted) return;
      setAuthOffline(demoModeEnabled);
      setAuthLoading(false);
    });
    return () => { mounted = false; };
  }, [setupStatus]);

  useEffect(() => {
    if (!sessionUser) return;
    if (sessionUser.demo) {
      setFiles(demoFiles);
      return;
    }
    fetch('/api/files', { credentials: 'include' }).then((response) => response.ok ? response.json() : Promise.reject()).then((data) => setFiles(data.files || [])).catch(() => setFiles([]));
  }, [sessionUser]);

  useEffect(() => {
    if (active !== 'admin' && active !== 'settings') return undefined;
    fetch('/api/admin/overview', { credentials: 'include' }).then((response) => response.ok ? response.json() : Promise.reject()).then((data) => setAdminOverview(data)).catch(() => setAdminOverview(null));
    if (active === 'admin') {
      fetch('/api/admin/users', { credentials: 'include' }).then((response) => response.ok ? response.json() : Promise.reject()).then((data) => setAdminUsers(data.users || [])).catch(() => setAdminUsers(null));
    }
    return undefined;
  }, [active, adminRefreshKey]);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = setTimeout(() => setToast(''), 2400);
    return () => clearTimeout(timer);
  }, [toast]);

  const filteredFiles = useMemo(() => {
    const searchTerm = active === 'files' ? query : '';
    const matchingFiles = files.filter((file) => file.name.toLowerCase().includes(searchTerm.toLowerCase()));
    const newestFirst = [...matchingFiles].sort((first, second) => new Date(second.uploadedAt || 0) - new Date(first.uploadedAt || 0));
    if (active !== 'dashboard') return newestFirst;
    return newestFirst.slice(0, 4);
  }, [files, query, active]);
  const galleryPageSize = 8;
  const galleryPageCount = Math.max(1, Math.ceil(filteredFiles.length / galleryPageSize));
  const galleryFiles = active === 'files' ? filteredFiles.slice((galleryPage - 1) * galleryPageSize, galleryPage * galleryPageSize) : filteredFiles;
  const previewFiles = filteredFiles;
  const galleryPageNumbers = useMemo(() => {
    const maxVisiblePages = 7;
    if (galleryPageCount <= maxVisiblePages) {
      return Array.from({ length: galleryPageCount }, (_, index) => index + 1);
    }

    const visiblePages = new Set([1, galleryPageCount]);
    const start = Math.max(2, galleryPage - 2);
    const end = Math.min(galleryPageCount - 1, galleryPage + 2);
    for (let page = start; page <= end; page += 1) visiblePages.add(page);

    const pages = [];
    let previousPage = 0;
    Array.from(visiblePages).sort((first, second) => first - second).forEach((page) => {
      if (page - previousPage > 1) pages.push(`ellipsis-${page}`);
      pages.push(page);
      previousPage = page;
    });
    return pages;
  }, [galleryPage, galleryPageCount]);

  useEffect(() => {
    setGalleryPage(1);
  }, [active, query]);

  useEffect(() => {
    setGalleryPage((current) => Math.min(current, galleryPageCount));
  }, [galleryPageCount]);

  useEffect(() => {
    if (active !== 'files' || previewFile || passwordFile || deleteFile) return undefined;
    function handleGalleryKeys(event) {
      const tag = String(event.target?.tagName || '').toLowerCase();
      if (tag === 'input' || tag === 'textarea' || event.target?.isContentEditable) return;
      if (event.key === 'ArrowLeft') {
        event.preventDefault();
        setGalleryPage((current) => Math.max(1, current - 1));
      }
      if (event.key === 'ArrowRight') {
        event.preventDefault();
        setGalleryPage((current) => Math.min(galleryPageCount, current + 1));
      }
    }
    document.addEventListener('keydown', handleGalleryKeys);
    return () => document.removeEventListener('keydown', handleGalleryKeys);
  }, [active, previewFile, passwordFile, deleteFile, galleryPageCount]);
  const totalSize = files.reduce((sum, file) => sum + (file.size || 0), 0);
  const defaultStorageLimit = 5 * 1024 * 1024 * 1024;
  const unlimitedStorage = sessionUser?.unlimitedStorage === true;
  const storageLimit = sessionUser?.maxStorageBytes || defaultStorageLimit;
  const storageUsed = formatSize(totalSize);
  const storagePercent = unlimitedStorage ? 12 : Math.min(100, Math.max(3, totalSize / storageLimit * 100));
  const userUploadLimit = Number(sessionUser?.maxUploadBytes) > 0 ? Number(sessionUser.maxUploadBytes) : (adminOverview?.maxUploadBytes || 100 * 1024 * 1024);
  const uploadLimitLabel = formatSize(userUploadLimit);
  const isResetPath = window.location.pathname === '/reset-password';
  const resetToken = new URLSearchParams(window.location.search).get('token') || '';

  async function login(email, password, remember = true) {
    setLoginError('');
    if (authOffline) {
      if (demoUser.email && demoPassword && email.trim().toLowerCase() === demoUser.email && password === demoPassword) {
        setSessionUser(demoUser);
        window.history.replaceState({}, '', '/dashboard');
        setNotFound(false);
        setActive('dashboard');
        return true;
      }
      setLoginError('Demo-Zugang ist nicht konfiguriert oder die Daten sind falsch.');
      return false;
    }
    try {
      const response = await fetch('/api/auth/login', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ email, password, remember }) });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Anmeldung fehlgeschlagen');
      setSessionUser(data.user);
      window.history.replaceState({}, '', '/dashboard');
      setNotFound(false);
      setActive('dashboard');
      return true;
    } catch (error) {
      setLoginError(error.message || 'Anmeldung fehlgeschlagen');
      return false;
    }
  }

  async function logout() {
    if (!sessionUser?.demo) await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' }).catch(() => {});
    setSessionUser(null);
    setAdminOverview(null);
    setAuthView('login');
    window.history.replaceState({}, '', '/login');
    setNotFound(false);
  }

  function uploadFileWithProgress(file, onProgress) {
    return new Promise((resolve, reject) => {
      const request = new XMLHttpRequest();
      request.open('POST', '/api/upload');
      request.withCredentials = true;
      request.upload.onprogress = (event) => {
        if (!event.lengthComputable) return;
        onProgress({
          percent: Math.round((event.loaded / event.total) * 100),
          loaded: event.loaded,
          total: event.total,
        });
      };
      request.onload = () => {
        let data = {};
        try {
          data = JSON.parse(request.responseText || '{}');
        } catch {
          data = {};
        }
        if (request.status >= 200 && request.status < 300 && data.id) resolve(data);
        else reject(new Error(data.error || 'API nicht erreichbar'));
      };
      request.onerror = () => reject(new Error('API nicht erreichbar'));
      const body = new FormData();
      body.append('file', file);
      request.send(body);
    });
  }

  const uploadReady = Boolean(apiOnline && healthStatus?.uploadReady);
  const uploadBlockedReason = !apiOnline
    ? 'Go API ist offline'
    : healthStatus?.apiKeyEnabled === false
      ? 'API-Key-Schutz ist nicht aktiv'
      : healthStatus?.database === false
        ? 'Datenbank ist nicht erreichbar'
        : (!uploadReady ? 'Upload derzeit nicht möglich' : '');

  async function uploadFiles(fileList) {
    const selected = Array.from(fileList || []);
    if (!selected.length) return;
    if (!uploadReady) {
      setToast(uploadBlockedReason || 'Upload derzeit nicht möglich');
      return;
    }

    const maxBytes = Number(userUploadLimit) > 0 ? Number(userUploadLimit) : 100 * 1024 * 1024;
    const tooLarge = selected.filter((file) => Number(file.size) > maxBytes);
    const allowed = selected.filter((file) => Number(file.size) <= maxBytes);
    if (tooLarge.length && !allowed.length) {
      const first = tooLarge[0];
      setToast(`Upload blockiert: ${first.name} ist ${formatSize(first.size)}, Limit ist ${formatSize(maxBytes)}`);
      return;
    }
    if (tooLarge.length) {
      setToast(`${tooLarge.length} Datei${tooLarge.length > 1 ? 'en' : ''} über dem Limit (${formatSize(maxBytes)}) – werden übersprungen`);
    }
    if (!allowed.length) return;

    setUploading(true);
    setUploadProgress({ current: 1, total: allowed.length, percent: 0, name: allowed[0].name });
    const nextFiles = [];
    const failures = [];
    for (const [index, file] of allowed.entries()) {
      if (Number(file.size) > maxBytes) {
        failures.push(`${file.name}: größer als Upload-Limit (${formatSize(maxBytes)})`);
        continue;
      }
      try {
        setUploadProgress({ current: index + 1, total: allowed.length, percent: Math.round((index / allowed.length) * 100), name: file.name, filePercent: 0, speed: 0 });
        let lastLoaded = 0;
        let lastAt = performance.now();
        let speedBps = 0;
        nextFiles.push(await uploadFileWithProgress(file, (progress) => {
          const filePercent = progress.percent;
          const now = performance.now();
          const elapsed = (now - lastAt) / 1000;
          if (elapsed >= 0.12 && progress.loaded >= lastLoaded) {
            const instant = (progress.loaded - lastLoaded) / elapsed;
            speedBps = speedBps > 0 ? (speedBps * 0.7 + instant * 0.3) : instant;
            lastLoaded = progress.loaded;
            lastAt = now;
          }
          const percent = Math.min(100, Math.round(((index + (filePercent / 100)) / allowed.length) * 100));
          setUploadProgress({ current: index + 1, total: allowed.length, percent, name: file.name, filePercent, speed: speedBps });
        }));
        setUploadProgress({ current: index + 1, total: allowed.length, percent: Math.round(((index + 1) / allowed.length) * 100), name: file.name, filePercent: 100 });
      } catch (error) {
        if (demoModeEnabled) {
          nextFiles.push({ id: `local-${Date.now()}-${file.name}`, name: file.name, size: file.size, contentType: file.type || 'application/octet-stream', url: URL.createObjectURL(file), uploadedAt: new Date().toISOString(), demo: true });
        } else {
          failures.push(`${file.name}: ${error.message || 'Upload fehlgeschlagen'}`);
        }
        setUploadProgress({ current: index + 1, total: allowed.length, percent: Math.round(((index + 1) / allowed.length) * 100), name: file.name, filePercent: 0 });
      }
    }
    if (!sessionUser?.demo && nextFiles.length) {
      try {
        const response = await fetch('/api/files', { credentials: 'include', cache: 'no-store' });
        if (response.ok) {
          const data = await response.json();
          setFiles(data.files || nextFiles);
        } else {
          setFiles((current) => [...nextFiles, ...current]);
        }
      } catch {
        setFiles((current) => [...nextFiles, ...current]);
      }
    } else {
      setFiles((current) => [...nextFiles, ...current]);
    }
    setUploading(false);
    setUploadProgress(null);
    if (failures.length && nextFiles.length) {
      setToast(`${nextFiles.length} Datei${nextFiles.length > 1 ? 'en' : ''} hochgeladen, ${failures.length} fehlgeschlagen`);
    } else if (failures.length) {
      setToast(failures[0]);
    } else {
      setToast(`${nextFiles.length} Datei${nextFiles.length > 1 ? 'en' : ''} hochgeladen`);
    }
  }

  async function removeFile(file) {
    if (!file.demo) {
      const response = await fetch(`/api/files/${file.id}`, { method: 'DELETE' });
      if (!response.ok) {
        setToast('Datei konnte nicht gelöscht werden');
        return false;
      }
    }
    setFiles((current) => current.filter((item) => item.id !== file.id));
    setToast('Datei gelöscht');
    return true;
  }

  async function saveFilePassword(file, password) {
    if (file.demo) {
      setFiles((current) => current.map((item) => item.id === file.id ? { ...item, passwordProtected: Boolean(password) } : item));
      setPasswordFile(null);
      setToast(password ? 'Datei geschützt' : 'Dateipasswort entfernt');
      return true;
    }
    try {
      const response = await fetch(`/api/files/${encodeURIComponent(file.id)}/password`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Passwort konnte nicht gespeichert werden');
      setFiles((current) => current.map((item) => item.id === file.id ? { ...item, passwordProtected: data.passwordProtected === true } : item));
      setPasswordFile(null);
      setToast(password ? 'Datei geschützt' : 'Dateipasswort entfernt');
      return true;
    } catch (error) {
      setToast(error.message || 'Passwort konnte nicht gespeichert werden');
      return false;
    }
  }

  async function copyURL(url) {
    setToast((await copyText(url)) ? 'URL kopiert' : 'URL konnte nicht kopiert werden');
  }

  function downloadFileWithProgress(file, onProgress) {
    return new Promise((resolve, reject) => {
      const request = new XMLHttpRequest();
      request.open('GET', file.url);
      request.withCredentials = true;
      request.responseType = 'blob';
      request.onprogress = (event) => {
        const total = event.lengthComputable ? event.total : Number(file.size) || 0;
        if (!total) return;
        onProgress({
          percent: Math.min(100, Math.round((event.loaded / total) * 100)),
          loaded: event.loaded,
          total,
        });
      };
      request.onload = () => {
        if (request.status >= 200 && request.status < 300 && request.response) resolve(request.response);
        else reject(new Error('Download fehlgeschlagen'));
      };
      request.onerror = () => reject(new Error('Download fehlgeschlagen'));
      request.send();
    });
  }

  async function downloadFile(file) {
    if (!file?.url) {
      setToast('Download nicht verfügbar');
      return;
    }
    if (uploading || downloadProgress) return;
    setDownloadProgress({ kind: 'download', current: 1, total: 1, percent: 0, name: file.name, speed: 0 });
    try {
      let lastLoaded = 0;
      let lastAt = performance.now();
      let speedBps = 0;
      const blob = await downloadFileWithProgress(file, (progress) => {
        const now = performance.now();
        const elapsed = (now - lastAt) / 1000;
        if (elapsed >= 0.12 && progress.loaded >= lastLoaded) {
          const instant = (progress.loaded - lastLoaded) / elapsed;
          speedBps = speedBps > 0 ? speedBps * 0.7 + instant * 0.3 : instant;
          lastLoaded = progress.loaded;
          lastAt = now;
        }
        setDownloadProgress({ kind: 'download', current: 1, total: 1, percent: progress.percent, name: file.name, speed: speedBps });
      });
      const href = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = href;
      link.download = file.name || 'download';
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(href), 1500);
      setToast('Download gestartet');
    } catch {
      setToast('Download fehlgeschlagen');
    }
    setDownloadProgress(null);
  }

  function onDrop(event) {
    event.preventDefault();
    setIsDragging(false);
    if (uploading || !uploadReady) return;
    uploadFiles(event.dataTransfer.files);
  }

  function refreshAdmin() {
    setAdminOverview(null);
    setAdminUsers(null);
    setAdminRefreshKey((current) => current + 1);
    fetch('/api/auth/me', { credentials: 'include', cache: 'no-store' }).then((response) => response.ok ? response.json() : Promise.reject()).then((data) => {
      if (data?.user) setSessionUser(data.user);
    }).catch(() => {});
    navigateTo('admin', { replace: true });
  }

  if (!setupStatus) return <AuthLoading />;
  if (!setupStatus.configured) return <SetupPage onComplete={(credentials) => { setSetupCredentials(credentials); setAuthLoading(true); setSetupStatus({ configured: true }); }} />;
  if (setupCredentials) return <SetupCompletePage credentials={setupCredentials} onContinue={() => {
    setSetupCredentials(null);
    setSessionUser(null);
    setLoginError('');
    setAuthView('login');
    setAuthLoading(false);
    window.history.replaceState({}, '', '/login');
  }} />;
  if (isResetPath) return <ResetPasswordPage token={resetToken} onDone={() => { window.history.replaceState({}, '', '/login'); setSessionUser(null); setAuthLoading(false); setAuthView('login'); }} />;
  if (authLoading) return <AuthLoading />;
  if (authOffline && (!demoUser.email || !demoPassword)) return <BackendOfflinePage />;
  if (!sessionUser && (authView === 'forgot' || window.location.pathname.replace(/\/$/, '') === '/forgot-password')) {
    return <ForgotPasswordPage onBack={() => { setLoginError(''); setAuthView('login'); window.history.pushState({}, '', '/login'); }} />;
  }
  if (!sessionUser) {
    return <LoginPage onSubmit={login} offline={authOffline} error={loginError} onForgot={() => { setLoginError(''); setAuthView('forgot'); window.history.pushState({}, '', '/forgot-password'); }} />;
  }
  if (notFound) return <NotFoundPage onHome={() => navigateTo('dashboard', { replace: true })} />;

  if (active === 'admin' && sessionUser.role === 'admin') {
    return <AdminWorkspace user={sessionUser} overview={adminOverview} users={adminUsers} files={files} apiOnline={apiOnline} onCopy={copyURL} onDelete={removeFile} onRefresh={refreshAdmin} onUserUpdate={setSessionUser} onDashboard={() => navigateTo('dashboard')} onSettings={() => navigateTo('settings')} onLogout={logout} />;
  }

  const transferProgress = uploadProgress || downloadProgress;
  const transferTitle = uploadProgress ? 'Upload läuft …' : 'Download läuft …';

  return <div className="app-shell">
    <aside className="sidebar">
      <Brand />
      <div className="workspace-label">WORKSPACE <span>⌄</span></div>
      <nav className="nav-list">
        <button className={active === 'dashboard' ? 'nav-item active' : 'nav-item'} onClick={() => navigateTo('dashboard')}><Icon name="grid" />Dashboard</button>
        <button className={active === 'files' ? 'nav-item active' : 'nav-item'} onClick={() => navigateTo('files')}><Icon name="image" />Galerie <em>{files.length}</em></button>
      </nav>
      <div className="sidebar-bottom">
        <div className={`storage-card${unlimitedStorage ? ' unlimited' : ''}`}>
          <div className="storage-title">
            <span>Speicher</span>
            {unlimitedStorage ? <em className="storage-pill">Unbegrenzt</em> : <span className="storage-limit">{formatSize(storageLimit)}</span>}
          </div>
          <strong className="storage-used">{storageUsed}</strong>
          <div className={unlimitedStorage ? 'storage-bar unlimited' : 'storage-bar'}><span style={{ width: `${storagePercent}%` }} /></div>
        </div>
        <div className="sidebar-bottom-nav">
          {sessionUser.role === 'admin' && <button className={active === 'admin' ? 'nav-item nav-item-admin active' : 'nav-item nav-item-admin'} onClick={() => navigateTo('admin')}><Icon name="shield" />Admin</button>}
        </div>
        <div className="profile-wrap" ref={profileMenuRef}>
          <button className={profileMenuOpen ? 'profile open' : 'profile'} type="button" aria-expanded={profileMenuOpen} aria-haspopup="menu" onClick={() => setProfileMenuOpen((open) => !open)}>
            <div className="avatar">{initials(sessionUser.name)}</div>
            <div className="profile-meta"><strong>{sessionUser.name}</strong><small>{sessionUser.role === 'admin' ? 'Administrator' : 'Benutzer'}</small></div>
            <Icon name="chevronUp" size={16} />
          </button>
          {profileMenuOpen && <div className="profile-menu" role="menu"><button type="button" role="menuitem" onClick={() => navigateTo('settings')}><Icon name="settings" size={15} />Einstellungen</button><button type="button" role="menuitem" onClick={logout}><Icon name="logout" size={15} />Abmelden</button></div>}
        </div>
      </div>
    </aside>

    <main className={active === 'settings' ? 'main-content settings-content-shell' : 'main-content'}>
      <input ref={inputRef} type="file" multiple hidden onChange={(event) => uploadFiles(event.target.files)} />
      {active === 'settings' ? <Settings apiOnline={apiOnline} overview={adminOverview} user={sessionUser} onUserUpdate={setSessionUser} onToast={setToast} /> : <>
        <section className={active === 'files' ? 'page-heading gallery-page-heading' : 'page-heading'}><div><p className="eyebrow">{active === 'files' ? 'GALERIE' : 'MONTAG, 19. OKTOBER 2026'}</p><h1>{active === 'files' ? <>Meine <span>Galerie.</span></> : <>Dein Upload-<span>Hub.</span></>}</h1><p className="subheading">{active === 'files' ? 'Alle deine Uploads übersichtlich an einem Ort.' : 'Schnell hochladen. Einfach teilen. Voll unter deiner Kontrolle.'}</p></div>{active === 'files' && transferProgress && <section className="gallery-upload-status"><div className="upload-progress" role="status" aria-live="polite"><div className="upload-progress-heading"><h2>{transferTitle}</h2><strong>{transferProgress.percent}%</strong></div><p className="upload-progress-name" title={transferProgress.name}>{transferProgress.name}</p><div className="upload-progress-track"><span style={{ width: `${transferProgress.percent}%` }} /></div><small>{uploadProgressCaption(transferProgress)}</small></div></section>}<div className="heading-actions"><button className="primary-button" disabled={!uploadReady || uploading} title={uploadBlockedReason || undefined} onClick={() => { if (uploadReady && !uploading) inputRef.current?.click(); }}><Icon name="upload" size={16} />Datei hochladen</button></div></section>
        {active === 'dashboard' && <section className="stats-grid"><StatCard label="DATEIEN GESAMT" value={files.length} detail="seit deinem Start" icon="image" /><StatCard label="GESPEICHERT" value={formatSize(totalSize)} detail={unlimitedStorage ? 'unbegrenzter Speicher' : `von ${formatSize(storageLimit)} verfügbar`} icon="file" /><StatCard label="API STATUS" value={uploadReady ? 'Online' : (apiOnline ? 'Eingeschränkt' : 'Offline')} detail={uploadReady ? 'bereit für ShareX' : (uploadBlockedReason || 'Upload gesperrt')} icon="check" /></section>}
        {active === 'dashboard' && <section className={`${isDragging && uploadReady ? 'dropzone dragging' : 'dropzone'}${uploadReady ? '' : ' dropzone-disabled'}`} onDragOver={(event) => { event.preventDefault(); if (!uploading && uploadReady) setIsDragging(true); }} onDragLeave={() => setIsDragging(false)} onDrop={onDrop} onClick={() => { if (!uploading && uploadReady) inputRef.current?.click(); }}><div className="drop-icon"><Icon name="upload" size={25} /></div>{transferProgress ? <div className="upload-progress" role="status" aria-live="polite"><div className="upload-progress-heading"><h2>{transferTitle}</h2><strong>{transferProgress.percent}%</strong></div><p className="upload-progress-name" title={transferProgress.name}>{transferProgress.name}</p><div className="upload-progress-track"><span style={{ width: `${transferProgress.percent}%` }} /></div><small>{uploadProgressCaption(transferProgress)}</small></div> : uploadReady ? <><h2>Dateien hier ablegen</h2><p>oder <strong>durchsuchen</strong> · max. {uploadLimitLabel} pro Datei</p><div className="drop-hint"><span className="shortcut">⌘</span><span>Du kannst mehrere Dateien gleichzeitig auswählen</span></div></> : <><h2>Upload gesperrt</h2><p>{uploadBlockedReason}</p><div className="drop-hint"><span>Go API, API-Key-Schutz und Datenbank müssen aktiv sein</span></div></>}</section>}
        <section className="files-section gallery-section"><div className="section-heading"><div><p className="eyebrow">{active === 'files' ? 'GALERIE' : 'DEINE DATEIEN'}</p><h2>{active === 'files' ? 'Meine Galerie' : 'Zuletzt hochgeladen'}</h2></div>{active === 'files' && <div className="section-tools"><div className="search-box"><Icon name="search" size={16} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Dateien suchen …" /></div><button className={viewMode === 'grid' ? 'view-button active' : 'view-button'} aria-label="Rasteransicht" aria-pressed={viewMode === 'grid'} onClick={() => setViewMode('grid')}>▦</button><button className={viewMode === 'list' ? 'view-button active' : 'view-button'} aria-label="Listenansicht" aria-pressed={viewMode === 'list'} onClick={() => setViewMode('list')}>☷</button></div>}</div><div className={viewMode === 'list' ? 'file-grid list-view' : 'file-grid'}>{galleryFiles.length ? galleryFiles.map((file) => <FileCard key={file.id} file={file} gallery onOpen={() => setPreviewFile(file)} onCopy={copyURL} onDownload={() => downloadFile(file)} onPassword={() => setPasswordFile(file)} onDelete={() => setDeleteFile(file)} />) : <div className="empty-state"><Icon name="file" size={30} /><h3>Keine Dateien gefunden</h3><p>Versuche einen anderen Suchbegriff.</p></div>}</div>{active === 'files' && galleryPageCount > 1 && <nav className="gallery-pagination" aria-label="Galerie-Seiten"><button type="button" className="gallery-page-arrow" disabled={galleryPage === 1} onClick={() => setGalleryPage((current) => Math.max(1, current - 1))}>← Zurück</button><div className="gallery-pages">{galleryPageNumbers.map((page, index) => typeof page === 'string' ? <span className="gallery-page-ellipsis" key={`${page}-${index}`} aria-hidden="true">…</span> : <button type="button" key={page} className={galleryPage === page ? 'gallery-page active' : 'gallery-page'} aria-current={galleryPage === page ? 'page' : undefined} onClick={() => setGalleryPage(page)}>{page}</button>)}</div><button type="button" className="gallery-page-arrow" disabled={galleryPage === galleryPageCount} onClick={() => setGalleryPage((current) => Math.min(galleryPageCount, current + 1))}>Weiter →</button></nav>}</section>
      </>}
    </main>
    {previewFile && <GalleryPreview file={previewFile} files={previewFiles} onChange={setPreviewFile} onClose={() => setPreviewFile(null)} />}
    {passwordFile && <FilePasswordDialog file={passwordFile} onClose={() => setPasswordFile(null)} onSave={(password) => saveFilePassword(passwordFile, password)} />}
    {deleteFile && <DeleteFileDialog file={deleteFile} onClose={() => setDeleteFile(null)} onConfirm={async () => { const deleted = await removeFile(deleteFile); if (deleted) setDeleteFile(null); return deleted; }} />}
    {toast && <div className="toast"><span className="toast-check"><Icon name="check" size={14} /></span>{toast}</div>}
  </div>;
}

function AuthLoading() {
  return <main className="auth-screen loading-screen"><LoginAmbient /><Brand className="auth-brand" /><div className="loading-pulse" /></main>;
}

function LoginAmbient() {
  return <>
    <div className="login-glow login-glow-a" aria-hidden="true" />
    <div className="login-glow login-glow-b" aria-hidden="true" />
    <div className="login-glow login-glow-c" aria-hidden="true" />
  </>;
}

function BackendOfflinePage() {
  return <main className="auth-screen"><LoginAmbient /><section className="login-card"><div className="login-lock"><Icon name="activity" size={19} /></div><p className="eyebrow">BACKEND NICHT ERREICHBAR</p><h2>Go-API ist offline.</h2><p className="setup-complete-copy">Die Oberfläche läuft, aber Login, Uploads und Admin-Funktionen benötigen das Go-Backend.</p><div className="login-error"><Icon name="activity" size={14} />Starte im Projektordner <code>docker compose up --build -d</code> und öffne anschließend Port 8080.</div><button className="login-submit" type="button" onClick={() => window.location.reload()}>Erneut versuchen<span>↻</span></button></section></main>;
}

function SetupPage({ onComplete }) {
  const [form, setForm] = useState({
    databaseType: 'sqlite',
    mysqlHost: '',
    mysqlPort: '',
    mysqlDatabase: '',
    mysqlUser: '',
    mysqlPassword: '',
    mysqlTls: 'true',
    baseUrl: '',
    adminEmail: '',
    adminPassword: '',
    adminPasswordConfirm: '',
    corsOrigin: '',
    smtpHost: '',
    smtpPort: '',
    smtpUser: '',
    smtpPassword: '',
    smtpFrom: '',
    smtpTls: 'true',
  });
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  function update(name, value) {
    setForm((current) => ({ ...current, [name]: value }));
  }

  async function submit(event, options = {}) {
    event.preventDefault();
    setError('');
    if (form.adminPassword !== form.adminPasswordConfirm) {
      setError('Die beiden Admin-Passwörter stimmen nicht überein.');
      return;
    }
    const setupForm = options.skipEmail
      ? { ...form, smtpHost: '', smtpPort: '', smtpUser: '', smtpPassword: '', smtpFrom: '' }
      : form;
    setSubmitting(true);
    try {
      const response = await fetch('/api/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(setupForm),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Setup konnte nicht abgeschlossen werden.');
      onComplete({ uploadApiKey: data.uploadApiKey, adminApiKey: data.adminApiKey, baseUrl: data.baseUrl });
    } catch (setupError) {
      setError(setupError.message || 'Setup konnte nicht abgeschlossen werden.');
    } finally {
      setSubmitting(false);
    }
  }

  return <SetupFormView form={form} update={update} error={error} submitting={submitting} submit={submit} />;
}


function SetupFormView({ form, update, error, submitting, submit }) {
  const [step, setStep] = useState(0);
  const [stepError, setStepError] = useState('');
  const [customSMTPPort, setCustomSMTPPort] = useState(false);
  const isSQLite = form.databaseType === 'sqlite';
  const steps = [
    { id: '01', title: 'Datenbank', detail: 'Datenbank für deine Installation auswählen' },
    { id: '02', title: 'Admin-Zugang', detail: 'Erster Benutzer mit Vollzugriff' },
    { id: '03', title: 'E-Mail-Versand', detail: 'Optional für Reset und Warnungen' },
  ];

  function validateStep(current) {
    if (current === 0) {
      if (!isSQLite) {
        if (!form.mysqlHost.trim() || !form.mysqlPort.trim() || !form.mysqlDatabase.trim() || !form.mysqlUser.trim() || !form.mysqlPassword) {
          return 'Bitte alle MySQL-Felder ausfüllen.';
        }
      }
      return '';
    }
    if (current === 1) {
      if (!form.adminEmail.trim() || !form.baseUrl.trim() || !form.adminPassword || !form.adminPasswordConfirm) {
        return 'Bitte alle Admin-Felder ausfüllen.';
      }
      if (form.adminPassword.length < 12) {
        return 'Das Admin-Passwort muss mindestens 12 Zeichen haben.';
      }
      if (form.adminPassword !== form.adminPasswordConfirm) {
        return 'Die beiden Admin-Passwörter stimmen nicht überein.';
      }
      return '';
    }
    return '';
  }

  function goNext(event) {
    event.preventDefault();
    const message = validateStep(step);
    if (message) {
      setStepError(message);
      return;
    }
    setStepError('');
    setStep((value) => Math.min(value + 1, steps.length - 1));
  }

  function goBack(event) {
    event.preventDefault();
    setStepError('');
    setStep((value) => Math.max(value - 1, 0));
  }

  function changeSMTPPort(event) {
    const value = event.target.value;
    if (value === 'custom') {
      setCustomSMTPPort(true);
      update('smtpPort', '');
      return;
    }
    setCustomSMTPPort(false);
    update('smtpPort', value);
  }

  function onSubmit(event) {
    event.preventDefault();
    const message = validateStep(step);
    if (message) {
      setStepError(message);
      return;
    }
    setStepError('');
    submit(event);
  }

  function finishWithoutEmail(event) {
    setStepError('');
    submit(event, { skipEmail: true });
  }

  return <main className="auth-screen setup-screen">
    <LoginAmbient />
    <section className="login-showcase">
      <Brand className="auth-brand" />
      <div className="showcase-copy">
        <p className="eyebrow">ERSTEINRICHTUNG</p>
        <h1>Dein eigener<br /><span>Upload-Hub.</span></h1>
        <p>Schritt für Schritt: Datenbank, Admin-Zugang und optional E-Mail.</p>
      </div>
    </section>
    <section className="login-card setup-card">
      <div className="login-card-heading">
        <div className="login-lock"><Icon name="database" size={19} /></div>
        <p className="eyebrow">SETUP-ASSISTENT</p>
        <h2>Instanz einrichten</h2>
        <p>Schritt {steps[step].id} von {steps[steps.length - 1].id}</p>
      </div>
      <div className="setup-stepper" aria-label="Setup-Schritte">
        {steps.map((item, index) => (
          <button
            key={item.id}
            type="button"
            className={`setup-step${index === step ? ' active' : ''}${index < step ? ' done' : ''}`}
            onClick={() => {
              if (index <= step) {
                setStepError('');
                setStep(index);
              }
            }}
          >
            <span>{item.id}</span>
            <strong>{item.title}</strong>
          </button>
        ))}
      </div>
      <form onSubmit={step === steps.length - 1 ? onSubmit : goNext} className="setup-form">
        {step === 0 && <div className="setup-section setup-section-active">
          <div className="setup-section-heading"><span>01</span><div><strong>Datenbank</strong><small>Datenbank für deine Installation auswählen</small></div></div>
          <div className="setup-database-choice">
            <label className={isSQLite ? 'selected' : ''}><input type="radio" name="databaseType" value="sqlite" checked={isSQLite} onChange={(event) => update('databaseType', event.target.value)} /><span><strong>Interne Datenbank</strong><small>SQLite wird automatisch auf dem Server angelegt</small></span></label>
            <label className={!isSQLite ? 'selected' : ''}><input type="radio" name="databaseType" value="mysql" checked={!isSQLite} onChange={(event) => update('databaseType', event.target.value)} /><span><strong>Externer Datenbankserver</strong><small>MySQL oder MariaDB · für einen vorhandenen Datenbankserver</small></span></label>
          </div>
          {isSQLite
            ? <div className="setup-database-note">SQLite wird automatisch erstellt und benötigt keinen separaten Datenbankserver.</div>
            : <div className="setup-grid">
                <label>Host<input value={form.mysqlHost} onChange={(event) => update('mysqlHost', event.target.value)} placeholder="db.example.com" autoComplete="off" /></label>
                <label>Port<input value={form.mysqlPort} onChange={(event) => update('mysqlPort', event.target.value)} placeholder="3306" inputMode="numeric" /></label>
                <label>Datenbankname<input value={form.mysqlDatabase} onChange={(event) => update('mysqlDatabase', event.target.value)} placeholder="datenbank_name" autoComplete="off" /></label>
                <label>Benutzer<input value={form.mysqlUser} onChange={(event) => update('mysqlUser', event.target.value)} placeholder="db_benutzer" autoComplete="username" /></label>
                <label className="setup-wide">Passwort<input type="password" value={form.mysqlPassword} onChange={(event) => update('mysqlPassword', event.target.value)} placeholder="••••••••" autoComplete="current-password" /></label>
                <label>TLS<select value={form.mysqlTls} onChange={(event) => update('mysqlTls', event.target.value)}><option value="true">Aktiviert</option><option value="false">Deaktiviert</option></select></label>
              </div>}
        </div>}

        {step === 1 && <div className="setup-section setup-section-active">
          <div className="setup-section-heading"><span>02</span><div><strong>Admin-Zugang</strong><small>Der erste Benutzer mit Vollzugriff</small></div></div>
          <div className="setup-grid">
            <label>E-Mail-Adresse<input type="email" value={form.adminEmail} onChange={(event) => update('adminEmail', event.target.value)} placeholder="admin@example.com" autoComplete="email" /></label>
            <label>Öffentliche URL<input type="url" value={form.baseUrl} onChange={(event) => update('baseUrl', event.target.value)} placeholder="https://files.example.com" /><small className="setup-field-note">Endgültige HTTPS-Adresse für deine Upload-Links. Die Admin-E-Mail wird für TLS-Benachrichtigungen verwendet.</small></label>
            <label>Admin-Passwort<input type="password" value={form.adminPassword} onChange={(event) => update('adminPassword', event.target.value)} placeholder="••••••••••••" minLength="12" autoComplete="new-password" /></label>
            <label>Passwort wiederholen<input type="password" value={form.adminPasswordConfirm} onChange={(event) => update('adminPasswordConfirm', event.target.value)} placeholder="••••••••••••" minLength="12" autoComplete="new-password" /></label>
          </div>
        </div>}

        {step === 2 && <div className="setup-section setup-section-active">
          <div className="setup-section-heading"><span>03</span><div><strong>E-Mail-Versand</strong><small>Optional für Passwort-Reset und Sicherheitswarnungen</small></div></div>
          <div className="setup-grid">
            <label>SMTP-Host<input value={form.smtpHost} onChange={(event) => update('smtpHost', event.target.value)} placeholder="smtp.example.com" autoComplete="off" /></label>
            <label>SMTP-Port<select value={customSMTPPort ? 'custom' : form.smtpPort} onChange={changeSMTPPort}><option value="">Port auswählen</option><option value="587">587 – STARTTLS</option><option value="465">465 – TLS</option><option value="25">25 – Standard</option><option value="custom">Eigener Port</option></select></label>
            {customSMTPPort && <label className="setup-wide">Eigener SMTP-Port<input value={form.smtpPort} onChange={(event) => update('smtpPort', event.target.value)} placeholder="z. B. 2525" inputMode="numeric" /></label>}
            <label>SMTP-Benutzer<input value={form.smtpUser} onChange={(event) => update('smtpUser', event.target.value)} placeholder="smtp_benutzer" autoComplete="off" /></label>
            <label>SMTP-Passwort<input type="password" value={form.smtpPassword} onChange={(event) => update('smtpPassword', event.target.value)} placeholder="••••••••" autoComplete="new-password" /></label>
            <label>Absender-Adresse<input type="email" value={form.smtpFrom} onChange={(event) => update('smtpFrom', event.target.value)} placeholder="admin@example.com" autoComplete="email" /></label>
            <label>TLS<select value={form.smtpTls} onChange={(event) => update('smtpTls', event.target.value)}><option value="true">Aktiviert</option><option value="false">Deaktiviert</option></select></label>
          </div>
          <div className="setup-database-note">E-Mail-Versand ist optional. Du kannst die Installation auch ohne SMTP abschließen.</div>
        </div>}

        {(stepError || error) && <div className="login-error"><Icon name="activity" size={14} />{stepError || error}</div>}
        <div className="setup-nav">
          {step > 0
            ? <button className="secondary-button" type="button" onClick={goBack}>Zurück</button>
            : <span />}
          {step < steps.length - 1
            ? <button className="login-submit setup-next" type="submit">Weiter<span>→</span></button>
            : <div className="setup-final-actions">
                <button className="secondary-button setup-skip-email" type="button" onClick={finishWithoutEmail} disabled={submitting}>Vorerst ohne E-Mail</button>
                <button className="login-submit setup-next" type="submit" disabled={submitting}>{submitting ? 'Verbindung wird geprüft …' : 'Installation abschließen'}<span>→</span></button>
              </div>}
        </div>
      </form>
      <p className="login-footnote">Der Setup-Assistent wird nach erfolgreicher Einrichtung automatisch gesperrt.</p>
    </section>
  </main>;
}

function SetupCompletePage({ credentials, onContinue }) {
  const [copied, setCopied] = useState('');

  async function copy(name, value) {
    if (await copyText(value)) {
      setCopied(name);
      window.setTimeout(() => setCopied(''), 1800);
    }
  }

  return <main className="auth-screen"><LoginAmbient /><section className="login-card setup-complete-card"><div className="login-lock setup-success-icon"><Icon name="check" size={21} /></div><p className="eyebrow">INSTALLATION ABGESCHLOSSEN</p><h2>Bereit für ShareX.</h2><p className="setup-complete-copy">Speichere diese Schlüssel jetzt sicher. Sie werden aus Sicherheitsgründen nur in dieser Abschlussansicht angezeigt.</p><div className="setup-key-list"><label>ShareX Upload-Key<div className="setup-key-row"><code>{credentials.uploadApiKey}</code><button type="button" onClick={() => copy('upload', credentials.uploadApiKey)}>{copied === 'upload' ? 'Kopiert' : 'Kopieren'}</button></div></label><label>Admin API-Key<div className="setup-key-row"><code>{credentials.adminApiKey}</code><button type="button" onClick={() => copy('admin', credentials.adminApiKey)}>{copied === 'admin' ? 'Kopiert' : 'Kopieren'}</button></div></label></div><button className="secondary-button setup-download-button" type="button" onClick={() => downloadShareXConfig(credentials.baseUrl, credentials.uploadApiKey)}>ShareX-Konfiguration herunterladen</button><button className="login-submit" type="button" onClick={onContinue}>Zur Anmeldung<span>→</span></button><p className="login-footnote">Die Setup-Seite ist jetzt dauerhaft gesperrt.</p></section></main>;
}


function PasswordInput({ value, onChange, placeholder, autoComplete, minLength, required }) {
  const [visible, setVisible] = useState(false);
  return <div className="password-field">
    <input
      type={visible ? 'text' : 'password'}
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      autoComplete={autoComplete}
      minLength={minLength}
      required={required}
    />
    <button
      type="button"
      className="password-toggle"
      aria-label={visible ? 'Passwort verbergen' : 'Passwort anzeigen'}
      title={visible ? 'Passwort verbergen' : 'Passwort anzeigen'}
      onClick={() => setVisible((current) => !current)}
    >
      <Icon name={visible ? 'eyeOff' : 'eye'} size={16} />
    </button>
  </div>;
}

function LoginPage({ onSubmit, offline, error, onForgot }) {
  const [email, setEmail] = useState(offline ? demoUser.email : '');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setSubmitting(true);
    await onSubmit(email, password, remember);
    setSubmitting(false);
  }

  return <main className="auth-screen login-screen">
    <LoginAmbient />
    <section className="login-panel">
      <div className="login-panel-top">
        <Brand className="auth-brand" />
        <div className="login-panel-copy">
          <p className="eyebrow">PRIVATER DATEIUPLOADER</p>
          <h1>Willkommen zurück.</h1>
          <p>Logge dich ein, um dein Dashboard zu öffnen.</p>
        </div>
      </div>
      <form onSubmit={submit} className="login-form">
        <label>E-Mail-Adresse
          <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="admin@example.com" autoComplete="username" required />
        </label>
        <label>Passwort
          <PasswordInput value={password} onChange={(event) => setPassword(event.target.value)} placeholder="••••••••••••" autoComplete="current-password" required />
        </label>
        <div className="login-options">
          <label className="checkbox-label">
            <input type="checkbox" checked={remember} onChange={(event) => setRemember(event.target.checked)} />
            <span>Angemeldet bleiben</span>
          </label>
          <button type="button" className="text-button" onClick={onForgot}>Passwort vergessen?</button>
        </div>
        {error && <div className="login-error"><Icon name="activity" size={14} />{error}</div>}
        <button className="login-submit" type="submit" disabled={submitting}>
          {submitting ? 'Anmeldung läuft …' : 'Einloggen'}
          <span>→</span>
        </button>
      </form>
      {offline && demoUser.email && demoPassword && <div className="demo-login">
        <div className="demo-login-title"><span className="demo-badge">DEMO</span> Vorschau-Modus</div>
        <p>Backend nicht aktiv. Demo-Zugang ist nur für lokale Entwicklung konfiguriert.</p>
        <code>{demoUser.email}</code>
        <code>Demo-Passwort gesetzt</code>
      </div>}
      <p className="login-meta">Self-hosted ShareX CDN</p>
    </section>
  </main>;
}

function ForgotPasswordPage({ onBack }) {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      const response = await fetch('/api/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      });
      if (!response.ok) throw new Error('Anfrage konnte nicht verarbeitet werden.');
      setSent(true);
    } catch (requestError) {
      setError(requestError.message || 'Anfrage konnte nicht verarbeitet werden.');
    } finally {
      setSubmitting(false);
    }
  }

  return <main className="auth-screen login-screen">
    <LoginAmbient />
    <section className="login-panel">
      <div className="login-panel-top">
        <Brand className="auth-brand" />
        <div className="login-panel-copy">
          <p className="eyebrow">PASSWORT VERGESSEN</p>
          <h1>Reset-Link anfordern.</h1>
          <p>Gib deine E-Mail-Adresse ein. Wir senden dir einen sicheren Link.</p>
        </div>
      </div>
      {sent ? <>
        <div className="login-success"><Icon name="check" size={15} />Wenn ein Konto mit dieser E-Mail existiert, wurde eine Nachricht versendet.</div>
        <button className="login-submit" type="button" onClick={onBack}>Zur Anmeldung<span>→</span></button>
      </> : <form onSubmit={submit} className="login-form">
        <label>E-Mail-Adresse
          <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="du@beispiel.de" autoComplete="email" required />
        </label>
        {error && <div className="login-error"><Icon name="activity" size={14} />{error}</div>}
        <button className="login-submit" type="submit" disabled={submitting}>
          {submitting ? 'Wird versendet …' : 'Reset-Link senden'}
          <span>→</span>
        </button>
        <button type="button" className="text-button auth-back-button" onClick={onBack}>Zurück zur Anmeldung</button>
      </form>}
      <p className="login-meta">Der Reset-Link ist 60 Minuten gültig und nur einmal verwendbar.</p>
    </section>
  </main>;
}

function ResetPasswordPage({ token, onDone }) {
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [done, setDone] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setError('');
    if (password.length < 12) {
      setError('Das Passwort muss mindestens 12 Zeichen haben.');
      return;
    }
    if (password !== confirm) {
      setError('Die beiden Passwörter stimmen nicht überein.');
      return;
    }
    setSubmitting(true);
    try {
      const response = await fetch('/api/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, password }),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Reset-Link ist ungültig oder abgelaufen.');
      setDone(true);
    } catch (requestError) {
      setError(requestError.message || 'Passwort konnte nicht geändert werden.');
    } finally {
      setSubmitting(false);
    }
  }

  return <main className="auth-screen"><LoginAmbient /><section className="login-card"><div className="login-card-heading"><div className="login-lock"><Icon name={done ? 'check' : 'shield'} size={19} /></div><p className="eyebrow">PASSWORT RESET</p><h2>{done ? 'Passwort geändert' : 'Neues Passwort'}</h2><p>{done ? 'Du kannst dich jetzt mit deinem neuen Passwort anmelden.' : 'Wähle ein neues, sicheres Passwort für deinen Account.'}</p></div>{done ? <button className="login-submit" type="button" onClick={onDone}>Zur Anmeldung<span>→</span></button> : !token ? <div className="login-error"><Icon name="activity" size={14} />Dieser Reset-Link ist ungültig.</div> : <form onSubmit={submit} className="login-form"><label>Neues Passwort<PasswordInput value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" minLength={12} required /></label><label>Passwort wiederholen<PasswordInput value={confirm} onChange={(event) => setConfirm(event.target.value)} autoComplete="new-password" minLength={12} required /></label>{error && <div className="login-error"><Icon name="activity" size={14} />{error}</div>}<button className="login-submit" type="submit" disabled={submitting}>{submitting ? 'Wird gespeichert …' : 'Passwort ändern'}<span>→</span></button></form>}<p className="login-footnote">Mindestens 12 Zeichen · aktive Sessions werden abgemeldet</p></section></main>;
}

function StatCard({ label, value, detail, icon }) {
  return <div className="stat-card"><div className="stat-icon"><Icon name={icon} size={18} /></div><div><p>{label}</p><strong>{value}</strong><small>{detail}</small></div></div>;
}

function GalleryPreview({ file, files, onChange, onClose }) {
  const source = file.url || file.previewUrl;
  const kind = fileKind(file);
  const isImage = kind === 'image';
  const isVideo = kind === 'video';
  const isAudio = kind === 'audio';
  const isPDF = kind === 'pdf';
  const currentIndex = files.findIndex((item) => item.id === file.id);
  const hasPrevious = currentIndex > 0;
  const hasNext = currentIndex >= 0 && currentIndex < files.length - 1;

  function showPrevious() {
    if (hasPrevious) onChange(files[currentIndex - 1]);
  }

  function showNext() {
    if (hasNext) onChange(files[currentIndex + 1]);
  }

  useEffect(() => {
    function handleKeyboard(event) {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
        event.preventDefault();
        if (hasPrevious) onChange(files[currentIndex - 1]);
        return;
      }
      if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
        event.preventDefault();
        if (hasNext) onChange(files[currentIndex + 1]);
      }
    }
    document.addEventListener('keydown', handleKeyboard, true);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', handleKeyboard, true);
      document.body.style.overflow = previousOverflow;
    };
  }, [onClose, onChange, currentIndex, files, hasPrevious, hasNext]);

  let media = <div className={`gallery-lightbox-file kind-${kind}`}><Icon name={fileIconName(kind)} size={48} /><strong>{typeLabel(file)}</strong><a href={source || '#'} target="_blank" rel="noreferrer" onClick={(event) => { if (!source) event.preventDefault(); }}>Datei öffnen</a></div>;
  if (isImage && source) media = <img className="gallery-lightbox-media" src={source} alt={file.name} />;
  if (isVideo && source) media = <video className="gallery-lightbox-media" src={source} controls autoPlay playsInline />;
  if (isAudio && source) media = <audio className="gallery-lightbox-audio" src={source} controls autoPlay />;
  if (isPDF && source) media = <iframe className="gallery-lightbox-document" src={source} title={file.name} />;

  return <div className="gallery-lightbox" role="dialog" aria-modal="true" aria-label={`${file.name} Vorschau`} onClick={onClose}><div className="gallery-lightbox-panel" onClick={(event) => event.stopPropagation()}><div className="gallery-lightbox-header"><div><strong>{file.name}</strong><span>{formatSize(file.size)} · {typeLabel(file)}{currentIndex >= 0 && files.length > 1 ? ` · ${currentIndex + 1} von ${files.length}` : ''}</span></div><button type="button" className="gallery-lightbox-close" onClick={onClose} aria-label="Vorschau schließen">×</button></div><div className="gallery-lightbox-stage"><button type="button" className="gallery-lightbox-nav previous" onClick={showPrevious} disabled={!hasPrevious} aria-label="Vorherige Datei">←</button>{media}<button type="button" className="gallery-lightbox-nav next" onClick={showNext} disabled={!hasNext} aria-label="Nächste Datei">→</button></div></div></div>;
}

function FilePasswordDialog({ file, onClose, onSave }) {
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setError('');
    if (password && password.length < 8) return setError('Das Passwort muss mindestens 8 Zeichen lang sein.');
    if (password && password !== confirm) return setError('Die Passwörter stimmen nicht überein.');
    setSaving(true);
    const saved = await onSave(password);
    if (!saved) setSaving(false);
  }

  return <div className="file-password-backdrop" role="presentation" onClick={onClose}><section className="file-password-dialog" role="dialog" aria-modal="true" aria-labelledby="file-password-title" onClick={(event) => event.stopPropagation()}><div className="file-password-heading"><div className="file-password-icon"><Icon name="shield" size={20} /></div><div><p className="eyebrow">DATEISCHUTZ</p><h2 id="file-password-title">Datei schützen</h2><p title={file.name}>{file.name}</p></div><button type="button" className="file-password-close" onClick={onClose} aria-label="Dialog schließen">×</button></div><form onSubmit={submit} className="file-password-form"><p>Lege ein Passwort fest. Der Link öffnet die Datei erst nach erfolgreicher Eingabe.</p><label>Neues Passwort<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} minLength={password ? 8 : undefined} autoComplete="new-password" placeholder={file.passwordProtected ? 'Neues Passwort eingeben' : 'Mindestens 8 Zeichen'} /></label><label>Passwort wiederholen<input type="password" value={confirm} onChange={(event) => setConfirm(event.target.value)} autoComplete="new-password" placeholder="Passwort wiederholen" /></label>{error && <div className="file-password-error"><Icon name="activity" size={14} />{error}</div>}<div className="file-password-actions"><button type="button" className="secondary-button" onClick={onClose}>Abbrechen</button>{file.passwordProtected && <button type="button" className="file-password-clear" onClick={() => onSave('')}>Schutz entfernen</button>}<button type="submit" className="primary-button" disabled={saving}>{saving ? 'Wird gespeichert …' : 'Passwort speichern'}</button></div></form></section></div>;
}

function DeleteFileDialog({ file, onClose, onConfirm }) {
  const [deleting, setDeleting] = useState(false);

  async function confirmDelete() {
    setDeleting(true);
    const deleted = await onConfirm();
    if (!deleted) setDeleting(false);
  }

  return <div className="file-password-backdrop" role="presentation" onClick={onClose}><section className="file-delete-dialog" role="dialog" aria-modal="true" aria-labelledby="delete-file-title" onClick={(event) => event.stopPropagation()}><div className="file-delete-icon"><Icon name="trash" size={22} /></div><p className="eyebrow">DATEI LÖSCHEN</p><h2 id="delete-file-title">Datei wirklich löschen?</h2><p className="file-delete-copy"><strong title={file.name}>{file.name}</strong><br />Bist du dir sicher, dass du diese Datei löschen willst? Dieser Vorgang kann nicht rückgängig gemacht werden.</p><div className="file-delete-actions"><button type="button" className="secondary-button" onClick={onClose} disabled={deleting}>Abbrechen</button><button type="button" className="file-delete-confirm" onClick={confirmDelete} disabled={deleting}>{deleting ? 'Wird gelöscht …' : 'Datei löschen'}</button></div></section></div>;
}

function viewURLForFile(file) {
  return file.previewUrl || file.url || '';
}

function rawURLForFile(file) {
  const kind = fileKind(file);
  if ((kind === 'image' || kind === 'video' || kind === 'audio') && file.rawUrl) return file.rawUrl;
  if ((kind === 'image' || kind === 'video' || kind === 'audio') && file.url) return file.url;
  return file.previewUrl || file.url || '';
}

function FileCard({ file, gallery, onOpen, onCopy, onDownload, onPassword, onDelete }) {
  const kind = fileKind(file);
  const isImage = kind === 'image';
  const canRenderImage = isImage && Boolean(file.url) && (!file.demo || file.url.startsWith('blob:'));
  const viewURL = viewURLForFile(file);
  const rawURL = rawURLForFile(file);
  return <article className={gallery ? 'file-card gallery-card' : 'file-card'}><div className={isImage ? 'file-preview image-preview' : `file-preview document-preview kind-${kind}`}><button className="file-preview-link" type="button" onClick={onOpen} aria-label={`${file.name} öffnen`}>{canRenderImage ? <img src={file.url} alt={file.name} loading="lazy" /> : isImage ? <div className="preview-art"><span>IMAGE</span><b>◒</b></div> : <div className={`document-art kind-${kind}`}><span className="document-art-icon"><Icon name={fileIconName(kind)} size={42} /></span><span>{typeLabel(file)}</span></div>}</button><span className={`file-type kind-${kind}`}>{typeLabel(file)}</span></div><div className="file-info"><div><h3 title={file.name}>{file.name}</h3><p>{formatSize(file.size)} <span>·</span> {formatDate(file.uploadedAt)}{file.passwordProtected && <span className="protected-file-mark" title="Passwortgeschützt"><Icon name="lock" size={12} /></span>}</p></div><div className="file-actions"><button type="button" className={file.passwordProtected ? 'security-button active' : 'security-button'} onClick={onPassword} title={file.passwordProtected ? 'Passwort ändern oder entfernen' : 'Passwort setzen'} aria-label={file.passwordProtected ? 'Dateipasswort ändern' : 'Datei mit Passwort schützen'}><Icon name="lock" size={15} /></button><button type="button" className="copy-button" onClick={() => onCopy(viewURL)} title="View-Link kopieren (Discord-Einbettung)" aria-label="View-Link kopieren"><Icon name="link" size={16} /></button><button type="button" className="copy-button raw-copy-button" onClick={() => onCopy(rawURL)} title="Raw-Link kopieren (nur Bild)" aria-label="Raw-Link kopieren"><Icon name="image" size={16} /></button><button type="button" className="download-button" onClick={onDownload} title="Datei herunterladen" aria-label="Datei herunterladen"><Icon name="download" size={16} /></button><button type="button" className="file-delete-button" onClick={onDelete} title="Datei löschen" aria-label="Datei löschen"><Icon name="trash" size={15} /></button></div></div></article>;
}

function AdminWorkspace({ user, overview, users, files, apiOnline, onCopy, onDelete, onRefresh, onUserUpdate, onDashboard, onSettings, onLogout }) {
  const [profileMenuOpen, setProfileMenuOpen] = useState(false);
  const [adminSection, setAdminSection] = useState(() => {
    const hash = String(window.location.hash || '').replace(/^#/, '');
    return ['overview', 'users', 'security', 'branding'].includes(hash) ? hash : 'overview';
  });
  const profileMenuRef = useRef(null);

  useEffect(() => {
    if (!profileMenuOpen) return undefined;
    function closeOnOutsideClick(event) {
      if (!profileMenuRef.current?.contains(event.target)) setProfileMenuOpen(false);
    }
    document.addEventListener('pointerdown', closeOnOutsideClick);
    return () => document.removeEventListener('pointerdown', closeOnOutsideClick);
  }, [profileMenuOpen]);

  function openSection(section) {
    setAdminSection(section);
    const next = section === 'overview' ? '/admin' : `/admin#${section}`;
    if (`${window.location.pathname}${window.location.hash}` !== next) {
      window.history.replaceState({}, '', next);
    }
  }

  return <div className="admin-shell">
    <aside className="admin-sidebar">
      <div className="admin-brand-row">
        <Brand />
        <span className="admin-badge"><Icon name="shield" size={12} />Admin</span>
      </div>
      <div className="admin-sidebar-label">ADMINISTRATION</div>
      <nav className="admin-nav-list" aria-label="Admin-Navigation">
        <button className={adminSection === 'overview' ? 'admin-nav-item active' : 'admin-nav-item'} type="button" onClick={() => openSection('overview')}><Icon name="grid" />Übersicht</button>
        <button className={adminSection === 'users' ? 'admin-nav-item active' : 'admin-nav-item'} type="button" onClick={() => openSection('users')}><Icon name="users" />Benutzer</button>
        <button className={adminSection === 'security' ? 'admin-nav-item active' : 'admin-nav-item'} type="button" onClick={() => openSection('security')}><Icon name="shield" />Sicherheit</button>
        <button className={adminSection === 'branding' ? 'admin-nav-item active' : 'admin-nav-item'} type="button" onClick={() => openSection('branding')}><Icon name="image" />Branding</button>
      </nav>
      <div className="admin-sidebar-bottom">
        <button className="admin-dashboard-link" type="button" onClick={onDashboard}><Icon name="grid" />Zum Dashboard</button>
        <div className="profile-wrap" ref={profileMenuRef}>
          <button className={profileMenuOpen ? 'profile open' : 'profile'} type="button" aria-expanded={profileMenuOpen} aria-haspopup="menu" onClick={() => setProfileMenuOpen((open) => !open)}>
            <div className="avatar">{initials(user.name)}</div>
            <div className="profile-meta"><strong>{user.name}</strong><small>Administrator</small></div>
            <Icon name="chevronUp" size={16} />
          </button>
          {profileMenuOpen && <div className="profile-menu" role="menu"><button type="button" role="menuitem" onClick={onDashboard}><Icon name="grid" size={15} />Dashboard</button><button type="button" role="menuitem" onClick={onSettings}><Icon name="settings" size={15} />Einstellungen</button><button type="button" role="menuitem" onClick={onLogout}><Icon name="logout" size={15} />Abmelden</button></div>}
        </div>
      </div>
    </aside>
    <main className="admin-main">
      <AdminPage section={adminSection} onSection={openSection} user={user} overview={overview} users={users} files={files} apiOnline={apiOnline} onCopy={onCopy} onDelete={onDelete} onRefresh={onRefresh} onUserUpdate={onUserUpdate} />
    </main>
  </div>;
}

function AdminPage({ section, onSection, user, overview, users, files, apiOnline, onCopy, onDelete, onRefresh, onUserUpdate }) {
  const totalSize = files.reduce((sum, file) => sum + (file.size || 0), 0);
  const fileCount = overview?.fileCount ?? files.length;
  const totalBytes = overview?.totalBytes ?? totalSize;
  const typeCount = overview?.typeCount ?? new Set(files.map((file) => typeLabel(file))).size;
  const maxUpload = overview?.maxUploadBytes ?? 100 * 1024 * 1024;
  const recentSource = overview?.recent?.length ? overview.recent : files;
  const recent = [...recentSource]
    .sort((first, second) => new Date(second.uploadedAt || 0) - new Date(first.uploadedAt || 0))
    .slice(0, 10);
  const apiKeyEnabled = overview?.apiKeyEnabled ?? false;
  const database = overview?.database ?? 'mysql';
  const databaseOnline = overview?.databaseOnline ?? apiOnline;
  const userCount = Array.isArray(users) ? users.length : (overview?.userCount ?? '—');

  const titles = {
    overview: { title: 'Übersicht', description: 'Status, Speicher und letzte Uploads.' },
    users: { title: 'Benutzer', description: 'Accounts anlegen und Speicherlimits setzen.' },
    security: { title: 'Sicherheit', description: 'API-Schlüssel und Schutzstatus.' },
    branding: { title: 'Branding', description: 'Logo und Discord-Einbettung.' },
  };
  const current = titles[section] || titles.overview;

  return <section className={`admin-page${section === 'overview' ? ' admin-overview' : ''}`}>
    <header className="admin-header">
      <div>
        <p className="eyebrow">ADMIN</p>
        <h1>{current.title}</h1>
        {section !== 'overview' && <p className="admin-header-copy">{current.description}</p>}
      </div>
      <div className="admin-header-actions">
        <div className="admin-live-pill"><span className={apiOnline ? 'admin-dot online' : 'admin-dot'} />{apiOnline ? 'API online' : 'API offline'}</div>
        <button className="secondary-button" type="button" onClick={onRefresh}><Icon name="activity" size={18} />Aktualisieren</button>
      </div>
    </header>

    {section === 'overview' && <div className="admin-overview-body">
      <div className="admin-stats">
        <AdminStat label="Uploads" value={fileCount} detail="gespeicherte Dateien" icon="image" />
        <AdminStat label="Speicher" value={formatSize(totalBytes)} detail="belegter Speicher" icon="database" />
        <AdminStat label="Benutzer" value={userCount} detail="aktive Accounts" icon="users" />
        <AdminStat label="Max. Upload" value={formatSize(maxUpload)} detail="Standardlimit pro Datei" icon="upload" />
      </div>
      <div className="admin-layout">
        <section className="admin-panel admin-panel-wide">
          <div className="admin-panel-heading"><div><p className="eyebrow">AKTIVITÄT</p><h2>Letzte Uploads</h2></div><span className="admin-count">{recent.length} angezeigt</span></div>
          {recent.length ? <div className="admin-file-list">{recent.map((file) => <div className="admin-file-row" key={file.id}><AdminFileTypeMark file={file} /><div className="admin-file-name"><strong title={file.name}>{file.name}</strong><span>{typeLabel(file)} · {formatDate(file.uploadedAt)}</span></div><span className="admin-file-size">{formatSize(file.size)}</span><button className="admin-row-button" onClick={() => onCopy(viewURLForFile(file))} title="View-Link kopieren (Discord-Einbettung)"><Icon name="link" size={18} /></button><button className="admin-row-button danger" onClick={() => onDelete(file)} title="Datei löschen"><Icon name="trash" size={18} /></button></div>)}</div> : <div className="admin-empty"><Icon name="file" size={36} /><strong>Noch keine Uploads</strong><p>Neue Dateien erscheinen hier automatisch.</p></div>}
        </section>
        <div className="admin-overview-side">
          <section className="admin-panel">
            <div className="admin-panel-heading"><div><p className="eyebrow">SYSTEM</p><h2>Instanzstatus</h2></div><span className={`status-pill ${apiOnline ? 'good' : ''}`}>{apiOnline ? 'Online' : 'Offline'}</span></div>
            <div className="system-list">
              <SystemRow label="Go API" value={apiOnline ? 'Verbunden' : 'Nicht erreichbar'} good={apiOnline} />
              <SystemRow label="Datenbank" value={databaseOnline ? String(database).toUpperCase() : 'Offline'} good={databaseOnline} />
              <SystemRow label="API-Key-Schutz" value={apiKeyEnabled ? 'Aktiv' : 'Nicht gesetzt'} good={apiKeyEnabled} />
              <SystemRow label="ShareX Endpoint" value={apiKeyEnabled && databaseOnline && apiOnline ? '/api/upload' : 'gesperrt'} good={apiKeyEnabled && databaseOnline && apiOnline} />
            </div>
          </section>
          <section className="admin-panel">
            <div className="admin-panel-heading"><div><p className="eyebrow">SCHNELLZUGRIFF</p><h2>Weiter</h2></div><Icon name="activity" size={26} /></div>
            <div className="admin-actions">
              <button type="button" onClick={() => onSection('users')}><Icon name="users" size={20} />Benutzer verwalten</button>
              <button type="button" onClick={() => onSection('security')}><Icon name="shield" size={20} />API-Schlüssel</button>
              <button type="button" onClick={() => onSection('branding')}><Icon name="image" size={20} />Branding anpassen</button>
            </div>
          </section>
        </div>
      </div>
    </div>}

    {section === 'users' && <div className="admin-single-panel admin-users-wrap"><AdminUserManagement users={users} onRefresh={onRefresh} /></div>}
    {section === 'security' && <div className="admin-security-layout"><AdminKeyManagement /><section className="admin-panel"><div className="admin-panel-heading"><div><p className="eyebrow">SYSTEM</p><h2>Schutzstatus</h2></div><Icon name="shield" size={26} /></div><div className="system-list"><SystemRow label="Go API" value={apiOnline ? 'Verbunden' : 'Nicht erreichbar'} good={apiOnline} /><SystemRow label="API-Key-Schutz" value={apiKeyEnabled ? 'Aktiv' : 'Nicht gesetzt'} good={apiKeyEnabled} /><SystemRow label="Datenbank" value={databaseOnline ? String(database).toUpperCase() : 'Offline'} good={databaseOnline} /></div><p className="admin-note">Neue Schlüssel werden nur einmal angezeigt. Danach ShareX-Konfiguration aktualisieren. Ohne aktive API, API-Key und Datenbank sind Uploads gesperrt.</p></section></div>}
    {section === 'branding' && <div className="admin-single-panel"><AdminBranding user={user} onUserUpdate={onUserUpdate} /></div>}
  </section>;
}

function applyEmbedPreview(template, fallback, values) {
  const raw = String(template || '').trim() || fallback;
  const filled = raw
    .replaceAll('{name}', values.name)
    .replaceAll('{site}', values.site)
    .replaceAll('{type}', values.type)
    .replaceAll('{size}', values.size)
    .trim();
  return filled || fallback;
}

function AdminBranding({ user, onUserUpdate }) {
  const [logoUrl, setLogoUrl] = useState('');
  const [siteName, setSiteName] = useState('404ProjectNET');
  const [embedColor, setEmbedColor] = useState('#718dff');
  const [embedTitle, setEmbedTitle] = useState('{name}');
  const [embedExtra, setEmbedExtra] = useState('');
  const [embedCard, setEmbedCard] = useState('large');
  const [embedImage, setEmbedImage] = useState('auto');
  const [showSiteName, setShowSiteName] = useState(true);
  const [showDate, setShowDate] = useState(true);
  const [showSize, setShowSize] = useState(true);
  const [showViews, setShowViews] = useState(true);
  const [showType, setShowType] = useState(false);
  const [showDimensions, setShowDimensions] = useState(true);
  const [showIcons, setShowIcons] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState(false);
  const [glowColor, setGlowColor] = useState(user?.glowColor || '#ffffff');
  const [savingGlow, setSavingGlow] = useState(false);

  useEffect(() => {
    setGlowColor(user?.glowColor || '#ffffff');
  }, [user?.id, user?.glowColor]);

  useEffect(() => {
    fetch('/api/admin/branding', { credentials: 'include', cache: 'no-store' }).then((response) => response.ok ? response.json() : Promise.reject()).then((data) => {
      setLogoUrl(data.logoUrl || '');
      setSiteName(data.siteName || '404ProjectNET');
      setEmbedColor(data.embedColor || '#718dff');
      setEmbedTitle(data.embedTitle || '{name}');
      setEmbedExtra(data.embedExtra || '');
      setEmbedCard(data.embedCard === 'compact' ? 'compact' : 'large');
      setEmbedImage(['auto', 'file', 'logo', 'none'].includes(data.embedImage) ? data.embedImage : 'auto');
      setShowSiteName(data.showSiteName !== false);
      setShowDate(data.showDate !== false);
      setShowSize(data.showSize !== false);
      setShowViews(data.showViews !== false);
      setShowType(data.showType === true);
      setShowDimensions(data.showDimensions !== false);
      setShowIcons(data.showIcons !== false);
    }).catch(() => setError('Branding konnte nicht geladen werden'));
  }, []);

  async function uploadLogo(event) {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    if (!['image/png', 'image/jpeg', 'image/gif'].includes(file.type)) {
      setError('Bitte eine PNG-, JPG- oder GIF-Datei auswählen');
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      setError('Das Logo darf maximal 5 MB groß sein');
      return;
    }
    setError('');
    setBusy(true);
    try {
      const form = new FormData();
      form.append('logo', file);
      const response = await fetch('/api/admin/branding', { method: 'PUT', credentials: 'include', body: form });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Logo konnte nicht gespeichert werden');
      setLogoUrl(data.logoUrl || '');
      window.dispatchEvent(new Event('branding-changed'));
    } catch (requestError) {
      setError(requestError.message || 'Logo konnte nicht gespeichert werden');
    } finally {
      setBusy(false);
    }
  }

  async function removeLogo() {
    setError('');
    setBusy(true);
    try {
      const response = await fetch('/api/admin/branding', { method: 'DELETE', credentials: 'include' });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Logo konnte nicht entfernt werden');
      setLogoUrl('');
      window.dispatchEvent(new Event('branding-changed'));
    } catch (requestError) {
      setError(requestError.message || 'Logo konnte nicht entfernt werden');
    } finally {
      setBusy(false);
    }
  }

  async function saveEmbed(event) {
    event.preventDefault();
    setError('');
    setNotice('');
    setBusy(true);
    try {
      const response = await fetch('/api/admin/branding', {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          siteName,
          embedColor,
          embedTitle,
          embedExtra,
          embedCard,
          embedImage,
          showSiteName,
          showDate,
          showSize,
          showViews,
          showType,
          showDimensions,
          showIcons,
        }),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Discord-Einstellungen konnten nicht gespeichert werden');
      setSiteName(data.siteName || '404ProjectNET');
      setEmbedColor(data.embedColor || '#718dff');
      setEmbedTitle(data.embedTitle || '{name}');
      setEmbedExtra(data.embedExtra || '');
      setEmbedCard(data.embedCard === 'compact' ? 'compact' : 'large');
      setEmbedImage(['auto', 'file', 'logo', 'none'].includes(data.embedImage) ? data.embedImage : 'auto');
      setShowSiteName(data.showSiteName !== false);
      setShowDate(data.showDate !== false);
      setShowSize(data.showSize !== false);
      setShowViews(data.showViews !== false);
      setShowType(data.showType === true);
      setShowDimensions(data.showDimensions !== false);
      setShowIcons(data.showIcons !== false);
      setNotice('Discord-Einbettung gespeichert');
    } catch (requestError) {
      setError(requestError.message || 'Discord-Einstellungen konnten nicht gespeichert werden');
    } finally {
      setBusy(false);
    }
  }

  async function saveGlowColor() {
    if (!/^#[0-9a-f]{6}$/i.test(glowColor)) {
      setError('Bitte eine gültige Farbe wählen');
      return;
    }
    setError('');
    setNotice('');
    setSavingGlow(true);
    try {
      const response = await fetch('/api/account/settings', { method: 'PUT', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ glowColor }) });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Glow-Effekt konnte nicht gespeichert werden');
      if (data.user && onUserUpdate) onUserUpdate(data.user);
      setGlowColor(data.glowColor || glowColor);
      setNotice('Glow-Effekt gespeichert');
    } catch (requestError) {
      setError(requestError.message || 'Glow-Effekt konnte nicht gespeichert werden');
    } finally {
      setSavingGlow(false);
    }
  }

  return <div className="branding-layout">
    <section className="admin-panel branding-panel"><div className="admin-panel-heading"><div><p className="eyebrow">LOGO & TEXT</p><h2>Erscheinungsbild</h2></div><Icon name="image" size={26} /></div><p className="admin-note branding-intro">Wird im Dashboard, Admin, auf den Anmeldeseiten und als Discord-Emblem verwendet.</p><div className="branding-preview"><img src={logoUrl || '/drop-logo.png'} alt="Logo-Vorschau" /></div><div className="branding-actions"><label className="secondary-button branding-upload-button">{busy ? 'Wird verarbeitet …' : 'Logo hochladen'}<input type="file" accept="image/png,image/jpeg,image/gif" onChange={uploadLogo} disabled={busy} /></label><button className="secondary-button" type="button" onClick={removeLogo} disabled={busy || !logoUrl}>Logo entfernen</button></div><p className="branding-help">PNG, JPG oder GIF · maximal 5 MB</p><form id="branding-site-form" className="branding-site-form" onSubmit={saveEmbed}><label>Powered by<input value={siteName} onChange={(event) => setSiteName(event.target.value)} maxLength={64} placeholder="404ProjectNET" /></label><p className="branding-help">Dieser Name erscheint auf Datei-Seiten hinter „Powered by“.</p><div className="branding-powered-preview" aria-label="Vorschau der Datei-Seite"><span className="branding-powered-preview-label">VORSCHAU DER DATEI-SEITE</span><div className="branding-powered-preview-stage"><img className="branding-powered-preview-image" src="/branding-preview.jpg" alt="Beispiel einer Datei-Seite" /><div className="branding-powered-preview-live"><span>Powered by {siteName.trim() || '404ProjectNET'}</span><img src={logoUrl || '/drop-logo.png'} alt="" /></div></div></div></form><div className="branding-glow-section"><p className="admin-note branding-glow-intro">Lege hier die Farbe des Glow-Effekts auf deinen Datei-Seiten fest.</p><div className="branding-glow-settings"><div><strong>Glow-Effekt</strong><p>Der Glow wird nur auf den Datei-Seiten verwendet.</p></div><div className="branding-glow-control"><span className="branding-glow-swatch" style={{ backgroundColor: glowColor, boxShadow: `0 0 18px ${glowColor}` }} /><input aria-label="Glow-Farbe" type="color" value={glowColor} onChange={(event) => setGlowColor(event.target.value)} /><code>{glowColor.toUpperCase()}</code><button className="secondary-button" type="button" onClick={saveGlowColor} disabled={savingGlow}>{savingGlow ? 'Wird gespeichert …' : 'Glow speichern'}</button></div></div></div><button className="primary-button branding-appearance-save" type="submit" form="branding-site-form" disabled={busy}>{busy ? 'Wird gespeichert …' : 'Erscheinungsbild speichern'}</button></section>
    <section className="admin-panel branding-panel"><div className="admin-panel-heading"><div><p className="eyebrow">DISCORD</p><h2>Einbettung</h2></div><Icon name="link" size={26} /></div><p className="admin-note branding-intro">Titel, Text, Bild und Farbe der Discord-Karte. Platzhalter: {'{name}'} {'{site}'} {'{type}'} {'{size}'}.</p>
      {(() => {
        const previewSite = siteName || '404ProjectNET';
        const previewTitle = applyEmbedPreview(embedTitle, 'beispiel.png', { name: 'beispiel.png', site: previewSite, type: 'IMAGE', size: '1.2 MB' });
        const previewBits = [
          embedExtra ? { text: embedExtra } : null,
          showType ? { icon: '🖼️', text: 'IMAGE' } : null,
          showDate ? { icon: '📅', text: '21. Okt., 12:00' } : null,
          showSize ? { icon: '📦', text: '1.2 MB' } : null,
          showDimensions ? { icon: '📐', text: '1920×1080' } : null,
          showViews ? { icon: '👁️', text: '1 Aufruf' } : null,
        ].filter(Boolean);
        const showImage = embedImage !== 'none';
        const imageSrc = embedImage === 'file' ? (logoUrl || '/drop-logo.png') : (logoUrl || '/drop-logo.png');
        return (
          <div className={embedCard === 'large' && showImage ? 'discord-embed discord-embed-large' : 'discord-embed'} style={{ borderLeftColor: embedColor || '#718dff' }}>
            <div className="discord-embed-top">
              <div className="discord-embed-copy">
                {showSiteName && <small>{previewSite}</small>}
                <strong>{previewTitle}</strong>
                {previewBits.length > 0 && <span className="discord-embed-meta">{previewBits.map((bit, index) => <span className="discord-embed-bit" key={`${bit.text}-${index}`}>{showIcons && bit.icon ? <span className="discord-embed-emoji">{bit.icon}</span> : null}{bit.text}</span>)}</span>}
              </div>
              {showImage && embedCard !== 'large' && <img className="discord-embed-mark" src={imageSrc} alt="" />}
            </div>
            {showImage && embedCard === 'large' && <img className="discord-embed-hero" src={imageSrc} alt="" />}
          </div>
        );
      })()}
      <form className="branding-embed-form" onSubmit={saveEmbed}>
        <label>Titel<input value={embedTitle} onChange={(event) => setEmbedTitle(event.target.value)} maxLength={120} placeholder="{name}" /></label>
        <label>Zusatztext<textarea value={embedExtra} onChange={(event) => setEmbedExtra(event.target.value)} maxLength={200} rows={2} placeholder="Optionaler Text über den Datei-Infos" /></label>
        <label>Embed-Farbe<div className="branding-color-row"><input aria-label="Embed-Farbe" type="color" value={/^#[0-9a-fA-F]{6}$/.test(embedColor) ? embedColor : '#718dff'} onChange={(event) => setEmbedColor(event.target.value)} /><input value={embedColor} onChange={(event) => setEmbedColor(event.target.value)} placeholder="#718dff" maxLength={7} /></div></label>
        <label>Kartengröße<select value={embedCard} onChange={(event) => setEmbedCard(event.target.value)}><option value="large">Großes Bild</option><option value="compact">Kompakt</option></select></label>
        <label>Bildquelle<select value={embedImage} onChange={(event) => setEmbedImage(event.target.value)}><option value="auto">Auto: Datei bei Bildern, sonst Logo</option><option value="file">Immer Datei / Logo-Fallback</option><option value="logo">Immer Logo</option><option value="none">Kein Bild</option></select></label>
        <div className="branding-embed-toggles">
          <label><input type="checkbox" checked={showSiteName} onChange={(event) => setShowSiteName(event.target.checked)} />Site-Name anzeigen</label>
          <label><input type="checkbox" checked={showType} onChange={(event) => setShowType(event.target.checked)} />Dateityp</label>
          <label><input type="checkbox" checked={showDate} onChange={(event) => setShowDate(event.target.checked)} />Datum</label>
          <label><input type="checkbox" checked={showSize} onChange={(event) => setShowSize(event.target.checked)} />Größe</label>
          <label><input type="checkbox" checked={showDimensions} onChange={(event) => setShowDimensions(event.target.checked)} />Auflösung</label>
          <label><input type="checkbox" checked={showViews} onChange={(event) => setShowViews(event.target.checked)} />Aufrufe</label>
          <label><input type="checkbox" checked={showIcons} onChange={(event) => setShowIcons(event.target.checked)} />Icons im Text</label>
        </div>
        <button className="primary-button" type="submit" disabled={busy}>{busy ? 'Wird gespeichert …' : 'Discord speichern'}</button>
      </form>
    </section>
    {notice && <div className="admin-user-notice"><Icon name="check" size={18} />{notice}</div>}
    {error && <div className="login-error branding-error"><Icon name="activity" size={18} />{error}</div>}
  </div>;
}

function bytesToMb(value, fallback = 100) {
  const bytes = Number(value);
  if (!Number.isFinite(bytes) || bytes < 1) return String(fallback);
  return String(Math.max(1, Math.round(bytes / 1048576)));
}

function draftFromUser(item) {
  return {
    maxUploadMb: bytesToMb(item.maxUploadBytes, 100),
    maxStorageMb: bytesToMb(item.maxStorageBytes, 5120),
    unlimitedStorage: item.unlimitedStorage === true,
  };
}

function draftsEqual(a, b) {
  if (!a || !b) return false;
  return String(a.maxUploadMb) === String(b.maxUploadMb)
    && String(a.maxStorageMb) === String(b.maxStorageMb)
    && Boolean(a.unlimitedStorage) === Boolean(b.unlimitedStorage);
}

function AdminUserManagement({ users, onRefresh }) {
  const [newUser, setNewUser] = useState({ name: '', email: '', password: '', maxUploadMb: '100', maxStorageMb: '5120', unlimitedStorage: false });
  const [drafts, setDrafts] = useState({});
  const [baseline, setBaseline] = useState({});
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState('');
  const [query, setQuery] = useState('');
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    if (!users) return;
    const next = Object.fromEntries(users.map((item) => [item.id, draftFromUser(item)]));
    setDrafts(next);
    setBaseline(next);
  }, [users]);

  function updateNew(name, value) {
    setNewUser((current) => ({ ...current, [name]: value }));
  }

  function updateDraft(id, name, value) {
    setDrafts((current) => ({ ...current, [id]: { ...current[id], [name]: value } }));
  }

  async function createUser(event) {
    event.preventDefault();
    setError('');
    setNotice('');
    setBusy('create');
    try {
      const response = await fetch('/api/admin/users', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(newUser) });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Benutzer konnte nicht angelegt werden.');
      setNewUser({ name: '', email: '', password: '', maxUploadMb: '100', maxStorageMb: '5120', unlimitedStorage: false });
      setShowCreate(false);
      setNotice('Benutzer wurde angelegt.');
      onRefresh();
    } catch (requestError) {
      setError(requestError.message || 'Benutzer konnte nicht angelegt werden.');
    } finally {
      setBusy('');
    }
  }

  async function saveLimits(userItem) {
    const draft = drafts[userItem.id];
    if (!draft) return;
    setError('');
    setNotice('');
    setBusy(userItem.id);
    try {
      const response = await fetch(`/api/admin/users/${encodeURIComponent(userItem.id)}`, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(draft) });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Limits konnten nicht gespeichert werden.');
      setBaseline((current) => ({ ...current, [userItem.id]: { ...draft } }));
      setNotice(`Limits für ${userItem.name} gespeichert.`);
      onRefresh();
      window.dispatchEvent(new Event('sharex-limits-changed'));
    } catch (requestError) {
      setError(requestError.message || 'Limits konnten nicht gespeichert werden.');
    } finally {
      setBusy('');
    }
  }

  function resetDraft(userItem) {
    const original = baseline[userItem.id] || draftFromUser(userItem);
    setDrafts((current) => ({ ...current, [userItem.id]: { ...original } }));
  }

  const filtered = !users ? null : users.filter((item) => {
    const haystack = `${item.name || ''} ${item.email || ''} ${item.role || ''}`.toLowerCase();
    return haystack.includes(query.trim().toLowerCase());
  });

  return <section id="admin-users" className="admin-panel admin-panel-wide user-management">
    <div className="admin-panel-heading user-management-heading">
      <div>
        <p className="eyebrow">ACCOUNTS</p>
        <h2>Benutzerverwaltung</h2>
      </div>
      <button className="secondary-button" type="button" onClick={() => setShowCreate((open) => !open)}>
        <Icon name="users" size={18} />
        {showCreate ? 'Formular schließen' : 'Benutzer hinzufügen'}
      </button>
    </div>

    {showCreate && <form className="admin-create-form" onSubmit={createUser}>
      <div className="admin-form-heading">
        <strong>Neuen Benutzer anlegen</strong>
        <span>Passwort mindestens 12 Zeichen</span>
      </div>
      <div className="admin-user-fields admin-user-fields-identity">
        <label>Name<input value={newUser.name} onChange={(event) => updateNew('name', event.target.value)} placeholder="Max Mustermann" required /></label>
        <label>E-Mail<input type="email" value={newUser.email} onChange={(event) => updateNew('email', event.target.value)} placeholder="max@beispiel.de" required /></label>
        <label>Passwort<input type="password" value={newUser.password} onChange={(event) => updateNew('password', event.target.value)} autoComplete="new-password" minLength="12" required /></label>
      </div>
      <div className="admin-user-fields admin-user-fields-limits">
        <label>Upload-Limit (MB)<input value={newUser.maxUploadMb} onChange={(event) => updateNew('maxUploadMb', event.target.value)} inputMode="numeric" min="1" required /></label>
        <label>Speicher-Limit (MB)<input value={newUser.maxStorageMb} onChange={(event) => updateNew('maxStorageMb', event.target.value)} inputMode="numeric" min="1" disabled={newUser.unlimitedStorage} required /></label>
        <label className="admin-checkbox"><input type="checkbox" checked={newUser.unlimitedStorage} onChange={(event) => updateNew('unlimitedStorage', event.target.checked)} /><span>Unlimited Speicher</span></label>
      </div>
      <div className="admin-create-actions">
        <button className="secondary-button" type="button" onClick={() => setShowCreate(false)}>Abbrechen</button>
        <button className="primary-button admin-create-submit" type="submit" disabled={busy === 'create'}>{busy === 'create' ? 'Wird angelegt …' : 'Benutzer anlegen'}<span>→</span></button>
      </div>
    </form>}

    <div className="user-management-toolbar">
      <div className="search-box admin-user-search">
        <Icon name="search" size={18} />
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Benutzer suchen …" />
      </div>
      <span className="admin-count">{filtered ? `${filtered.length} Benutzer` : 'Laden …'}</span>
    </div>

    {error && <div className="login-error admin-user-error"><Icon name="activity" size={18} />{error}</div>}
    {notice && <div className="admin-user-notice"><Icon name="check" size={18} />{notice}</div>}

    <div className="managed-user-list">
      {filtered === null ? <div className="admin-empty"><div className="loading-pulse" /><p>Benutzer werden geladen …</p></div>
        : filtered.length ? <>
          <div className="managed-user-columns" aria-hidden="true"><span>Benutzer</span><span>Upload (MB)</span><span>Speicher (MB)</span><span>Optionen</span><span /></div>
          {filtered.map((item) => {
            const draft = drafts[item.id] || draftFromUser(item);
            const dirty = !draftsEqual(draft, baseline[item.id] || draftFromUser(item));
            return <div className={`managed-user-row${dirty ? ' dirty' : ''}`} key={item.id}>
              <div className="managed-user-identity">
                <div className="avatar">{initials(item.name)}</div>
                <div>
                  <strong>{item.name}</strong>
                  <span>{item.email}</span>
                  <em className={`role-pill ${item.role === 'admin' ? 'admin' : ''}`}>{item.role === 'admin' ? 'Administrator' : 'Benutzer'}</em>
                </div>
              </div>
              <label><span className="field-label">Upload (MB)</span><input value={draft.maxUploadMb || ''} onChange={(event) => updateDraft(item.id, 'maxUploadMb', event.target.value)} inputMode="numeric" min="1" /></label>
              <label><span className="field-label">Speicher (MB)</span><input value={draft.maxStorageMb || ''} onChange={(event) => updateDraft(item.id, 'maxStorageMb', event.target.value)} inputMode="numeric" min="1" disabled={draft.unlimitedStorage} /></label>
              <label className="admin-checkbox"><input type="checkbox" checked={draft.unlimitedStorage === true} onChange={(event) => updateDraft(item.id, 'unlimitedStorage', event.target.checked)} /><span>Unlimited</span></label>
              <div className="managed-user-actions">
                {dirty && <button className="ghost-button" type="button" onClick={() => resetDraft(item)}>Zurück</button>}
                <button className={`save-limits-button${dirty ? ' ready' : ''}`} type="button" onClick={() => saveLimits(item)} disabled={!dirty || busy === item.id}>{busy === item.id ? 'Speichert …' : 'Speichern'}</button>
              </div>
            </div>;
          })}
        </> : <div className="admin-empty"><Icon name="users" size={36} /><strong>{query ? 'Keine Treffer' : 'Keine Benutzer'}</strong><p>{query ? 'Suche anpassen oder zurücksetzen.' : 'Lege den ersten Benutzer an.'}</p></div>}
    </div>
  </section>;
}

function AdminKeyManagement() {
  const [busy, setBusy] = useState('');
  const [revealed, setRevealed] = useState(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');

  async function rotate(type) {
    const label = type === 'upload' ? 'ShareX Upload-Key' : 'Admin API-Key';
    if (!window.confirm(`${label} wirklich neu generieren? Der bisherige Schlüssel wird sofort ungültig.`)) return;
    setBusy(type);
    setError('');
    setRevealed(null);
    setCopied(false);
    try {
      const response = await fetch('/api/admin/keys', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ type }) });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Schlüssel konnte nicht neu generiert werden.');
      setRevealed({ label, key: data.key, type, baseUrl: data.baseUrl });
      if (type === 'upload') downloadShareXConfig(data.baseUrl, data.key);
    } catch (requestError) {
      setError(requestError.message || 'Schlüssel konnte nicht neu generiert werden.');
    } finally {
      setBusy('');
    }
  }

  async function copyKey() {
    if (!revealed?.key) return;
    if (await copyText(revealed.key)) {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    }
  }

  return <section id="admin-keys" className="admin-panel key-management"><div className="admin-panel-heading"><div><p className="eyebrow">API</p><h2>Schlüssel</h2></div><Icon name="shield" size={26} /></div><p className="admin-note" style={{ marginTop: 0, marginBottom: 14 }}>ShareX bekommt den Upload-Key. Beim Erzeugen einer neuen <code>.sxcu</code> wird der alte Upload-Key ungültig.</p><div className="key-management-actions"><button type="button" onClick={() => rotate('upload')} disabled={Boolean(busy)}><span>{busy === 'upload' ? 'Wird erzeugt …' : 'ShareX-Key neu generieren + .sxcu'}</span><span>→</span></button><button type="button" onClick={() => rotate('admin')} disabled={Boolean(busy)}><span>{busy === 'admin' ? 'Wird erzeugt …' : 'Admin API-Key neu generieren'}</span><span>→</span></button></div>{revealed && <div className="key-management-result"><strong>{revealed.label} – nur jetzt sichtbar</strong><code>{revealed.key}</code><button className="secondary-button" type="button" onClick={copyKey}>{copied ? 'Kopiert' : 'Schlüssel kopieren'}</button>{revealed.type === 'upload' && <button className="secondary-button" type="button" onClick={() => downloadShareXConfig(revealed.baseUrl, revealed.key)}>ShareX-Konfiguration erneut herunterladen</button>}</div>}{error && <div className="login-error key-management-error"><Icon name="activity" size={18} />{error}</div>}</section>;
}

function AdminStat({ label, value, detail, icon }) {
  return <div className="admin-stat"><div className="admin-stat-icon"><Icon name={icon} size={26} /></div><div><p>{label}</p><strong>{value}</strong><small>{detail}</small></div></div>;
}

function SystemRow({ label, value, good }) {
  return <div className="system-row"><span><i className={good ? 'system-indicator good' : 'system-indicator'} />{label}</span><strong>{value}</strong></div>;
}

function Settings({ apiOnline, overview, user, onUserUpdate, onToast }) {
  const database = overview?.database ? overview.database.toUpperCase() : '—';
  const [activeTab, setActiveTab] = useState(() => {
    const savedTab = readPreference('drop-settings-tab', 'profile');
    return ['profile', 'security', 'system'].includes(savedTab) ? savedTab : 'profile';
  });
  const [profileForm, setProfileForm] = useState({ name: user?.name || '', email: user?.email || '' });
  const [savingProfile, setSavingProfile] = useState(false);
  const [passwordForm, setPasswordForm] = useState({ currentPassword: '', newPassword: '', confirmPassword: '' });
  const [changingPassword, setChangingPassword] = useState(false);

  useEffect(() => {
    writePreference('drop-settings-tab', activeTab);
  }, [activeTab]);

  useEffect(() => {
    setProfileForm({ name: user?.name || '', email: user?.email || '' });
  }, [user?.id, user?.name, user?.email]);

  async function saveProfile(event) {
    event.preventDefault();
    const name = profileForm.name.trim();
    const email = profileForm.email.trim();
    if (name.length < 2 || name.length > 80) {
      onToast('Der Name muss zwischen 2 und 80 Zeichen haben');
      return;
    }
    if (!/^\S+@\S+\.\S+$/.test(email)) {
      onToast('Bitte eine gültige E-Mail-Adresse eingeben');
      return;
    }
    if (user?.demo) {
      onUserUpdate({ ...user, name, email });
      onToast('Profil gespeichert');
      return;
    }
    setSavingProfile(true);
    try {
      const response = await fetch('/api/account/profile', { method: 'PUT', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name, email }) });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Profil konnte nicht gespeichert werden');
      if (data.user) onUserUpdate(data.user);
      onToast('Profil gespeichert');
    } catch (error) {
      onToast(error.message || 'Profil konnte nicht gespeichert werden');
    } finally {
      setSavingProfile(false);
    }
  }

  async function changePassword(event) {
    event.preventDefault();
    if (passwordForm.newPassword.length < 12 || passwordForm.newPassword.length > 256) {
      onToast('Das neue Passwort muss zwischen 12 und 256 Zeichen haben');
      return;
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      onToast('Die neuen Passwörter stimmen nicht überein');
      return;
    }
    if (user?.demo) {
      onToast('Passwortänderung ist im Demo-Modus nicht verfügbar');
      return;
    }
    setChangingPassword(true);
    try {
      const response = await fetch('/api/account/password', { method: 'PUT', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(passwordForm) });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Passwort konnte nicht geändert werden');
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
      onToast('Passwort geändert. Andere Sitzungen wurden abgemeldet.');
    } catch (error) {
      onToast(error.message || 'Passwort konnte nicht geändert werden');
    } finally {
      setChangingPassword(false);
    }
  }

  const tabs = [
    { id: 'profile', label: 'Profil', icon: 'user' },
    { id: 'security', label: 'Sicherheit', icon: 'shield' },
    { id: 'system', label: 'System', icon: 'database' },
  ];

  return <section className={`settings-page settings-page-${activeTab}`}>
    <div className="settings-card"><div className="settings-icon"><Icon name="settingsHeader" size={22} /></div><div><p className="eyebrow">KONTO & KONFIGURATION</p><h2>Einstellungen</h2><p className="subheading">Verwalte dein Profil, deine Sicherheit und das Erscheinungsbild.</p></div></div>
    <nav className="settings-tabs" aria-label="Einstellungen"><div className="settings-tab-list" role="tablist">{tabs.map((tab) => <button key={tab.id} type="button" role="tab" aria-selected={activeTab === tab.id} className={activeTab === tab.id ? 'settings-tab active' : 'settings-tab'} onClick={() => setActiveTab(tab.id)}><Icon name={tab.icon} size={16} />{tab.label}</button>)}</div></nav>
    <div className="settings-panel-wrap">
      {activeTab === 'profile' && <section className="settings-panel" role="tabpanel"><div className="settings-panel-heading"><div><p className="eyebrow">DEIN KONTO</p><h3>Profil bearbeiten</h3><p className="settings-panel-description">Ändere hier deinen Namen und deine E-Mail-Adresse.</p></div><span className="settings-profile-status" aria-label="Profil" title="Profil"><Icon name="user" size={18} /></span></div><div className="settings-profile-card"><div className="settings-profile-avatar">{initials(profileForm.name)}</div><div><strong>{profileForm.name || 'Benutzer'}</strong><p>{user?.role === 'admin' ? 'Administrator · Vollzugriff' : 'Benutzerkonto'}</p></div></div><form id="profile-form" className="settings-profile-form" onSubmit={saveProfile}><label>Name<input type="text" value={profileForm.name} onChange={(event) => setProfileForm((current) => ({ ...current, name: event.target.value }))} autoComplete="name" maxLength="80" required /></label><label>E-Mail-Adresse<input type="email" value={profileForm.email} onChange={(event) => setProfileForm((current) => ({ ...current, email: event.target.value }))} autoComplete="email" maxLength="254" required /></label></form><div className="settings-info-grid"><div><span>Rolle</span><strong>{user?.role === 'admin' ? 'Administrator' : 'Benutzer'}</strong></div><div><span>Speicher</span><strong>{user?.unlimitedStorage ? 'Unbegrenzt' : 'Limit aktiv'}</strong></div></div><button className="login-submit settings-submit settings-profile-save" form="profile-form" type="submit" disabled={savingProfile}>Änderungen speichern<span>→</span></button></section>}
      {activeTab === 'security' && <section className="settings-panel settings-security-panel" role="tabpanel"><div className="settings-panel-heading"><div><p className="eyebrow">KONTO SCHÜTZEN</p><h3>Passwort ändern</h3><p className="settings-panel-description">Verwende ein neues, einzigartiges Passwort mit mindestens 12 Zeichen.</p></div><Icon name="shield" size={22} /></div><form className="settings-password-form" onSubmit={changePassword}><label>Aktuelles Passwort<input type="password" value={passwordForm.currentPassword} onChange={(event) => setPasswordForm((current) => ({ ...current, currentPassword: event.target.value }))} autoComplete="current-password" required /></label><label>Neues Passwort<input type="password" value={passwordForm.newPassword} onChange={(event) => setPasswordForm((current) => ({ ...current, newPassword: event.target.value }))} minLength="12" autoComplete="new-password" required /><small>Mindestens 12 Zeichen</small></label><label>Neues Passwort wiederholen<input type="password" value={passwordForm.confirmPassword} onChange={(event) => setPasswordForm((current) => ({ ...current, confirmPassword: event.target.value }))} minLength="12" autoComplete="new-password" required /></label><button className="login-submit settings-submit" type="submit" disabled={changingPassword}>{changingPassword ? 'Passwort wird geändert …' : 'Passwort ändern'}<span>→</span></button></form><div className="settings-security-note"><Icon name="activity" size={16} /><span>Nach der Änderung werden alle anderen aktiven Sitzungen abgemeldet.</span></div></section>}
      {activeTab === 'system' && <section className="settings-panel" role="tabpanel"><div className="settings-panel-heading"><div><p className="eyebrow">STATUS & INTEGRATION</p><h3>System</h3><p className="settings-panel-description">Technische Informationen zu deiner Upload-Instanz.</p></div><Icon name="activity" size={22} /></div><div className="settings-system-list"><div className="settings-row"><div><strong>Backend-Status</strong><p>Verbindung zur Go-API</p></div><span className={apiOnline ? 'status-pill good' : 'status-pill'}>{apiOnline ? 'Online' : 'Demo-Modus'}</span></div><div className="settings-row"><div><strong>Datenbank</strong><p>Aktiv verwendeter Datenbanktyp</p></div><span className={overview ? 'status-pill good' : 'status-pill'}>{database}</span></div><div className="settings-row"><div><strong>ShareX-Konfiguration</strong><p>Upload-Vorlage für deinen Client</p></div><span className="status-pill good">Bereit</span></div><div className="settings-row"><div><strong>Benutzerlimits</strong><p>Upload- und Speicherlimits werden vom Admin verwaltet.</p></div><span className="status-pill">Admin</span></div></div></section>}
    </div>
  </section>;
}

createRoot(document.getElementById('root')).render(<App />);

