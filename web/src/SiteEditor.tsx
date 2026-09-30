import { useEffect, useState } from 'react';
import { ApiError, Preset, Site, api, fieldErrors } from './api';

function parseLines(text: string): string[] {
  return text
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l.length > 0);
}

function parseEnv(text: string): Record<string, string> {
  const env: Record<string, string> = {};
  for (const line of parseLines(text)) {
    const i = line.indexOf('=');
    if (i > 0) env[line.slice(0, i).trim()] = line.slice(i + 1).trim();
  }
  return env;
}

function formatEnv(env?: Record<string, string>): string {
  return Object.entries(env ?? {})
    .map(([k, v]) => `${k}=${v}`)
    .join('\n');
}

interface Props {
  siteId: string | null; // null = new site
  onChanged: () => void;
}

export default function SiteEditor({ siteId, onChanged }: Props) {
  const isNew = siteId === null;
  const [id, setId] = useState('');
  const [preset, setPreset] = useState<Preset>('static');
  const [domains, setDomains] = useState('');
  const [tls, setTls] = useState<'auto' | 'off'>('auto');
  const [log, setLog] = useState(true);
  const [root, setRoot] = useState('');
  const [service, setService] = useState('');
  const [image, setImage] = useState('');
  const [port, setPort] = useState('3000');
  const [env, setEnv] = useState('');
  const [errors, setErrors] = useState<Map<string, string>>(new Map());
  const [notice, setNotice] = useState('');
  const [snippet, setSnippet] = useState('');

  useEffect(() => {
    if (siteId === null) {
      setId('');
      setPreset('static');
      setDomains('');
      setTls('auto');
      setLog(true);
      setRoot('');
      setService('');
      setImage('');
      setPort('3000');
      setEnv('');
      setSnippet('');
      setErrors(new Map());
      setNotice('');
      return;
    }
    api
      .getSite(siteId)
      .then((s) => {
        setId(s.id);
        setPreset(s.preset);
        setDomains(s.domains.join('\n'));
        setTls(s.tls);
        setLog(s.log);
        setRoot(s.root ?? '');
        setService(s.backend?.service ?? '');
        setImage(s.backend?.image ?? '');
        setPort(String(s.backend?.internalPort ?? 3000));
        setEnv(formatEnv(s.backend?.env));
        setErrors(new Map());
        setNotice('');
      })
      .catch((e: unknown) => setNotice(e instanceof Error ? e.message : 'load failed'));
    api
      .getSnippet(siteId)
      .then((r) => setSnippet(r.snippet))
      .catch(() => setSnippet(''));
  }, [siteId]);

  async function save() {
    setErrors(new Map());
    setNotice('');
    const backend =
      preset === 'proxy' || preset === 'php'
        ? {
            ...(service.trim() ? { service: service.trim() } : {}),
            image: image.trim(),
            ...(preset === 'proxy' ? { internalPort: Number(port) || 0 } : {}),
            ...(env.trim() ? { env: parseEnv(env) } : {}),
          }
        : undefined;
    const site: Site = {
      id: id.trim(),
      version: 1,
      domains: parseLines(domains),
      preset,
      tls,
      log,
      ...(preset === 'static' || preset === 'spa' || preset === 'php' ? { root: root.trim() } : {}),
      ...(backend ? { backend } : {}),
    };
    if (!site.id) {
      setErrors(new Map([['id', 'site id is required']]));
      return;
    }
    try {
      if (isNew) await api.createSite(site);
      else await api.updateSite(siteId as string, site);
      const snip = await api.getSnippet(site.id).catch(() => ({ snippet: '' }));
      setSnippet(snip.snippet);
      setNotice('Saved.');
      onChanged();
    } catch (e: unknown) {
      if (e instanceof ApiError && e.errors.length > 0) setErrors(fieldErrors(e.errors));
      setNotice(e instanceof Error ? e.message : 'save failed');
    }
  }

  async function remove() {
    if (siteId === null) return;
    if (!window.confirm(`Delete site ${siteId}?`)) return;
    try {
      await api.deleteSite(siteId);
      onChanged();
    } catch (e: unknown) {
      setNotice(e instanceof Error ? e.message : 'delete failed');
    }
  }

  const err = (f: string) => (errors.has(f) ? <span className="err">{errors.get(f)}</span> : null);
  const showRoot = preset === 'static' || preset === 'spa' || preset === 'php';
  const showBackend = preset === 'proxy' || preset === 'php';

  return (
    <div className="editor">
      <h2>{isNew ? 'New site' : `Site ${siteId}`}</h2>
      <label>
        Site id {err('id')}
        <input value={id} onChange={(e) => setId(e.target.value)} disabled={!isNew} placeholder="api" />
      </label>
      <label>
        Preset
        <select value={preset} onChange={(e) => setPreset(e.target.value as Preset)}>
          <option value="static">static — plain files</option>
          <option value="spa">spa — single-page fallback</option>
          <option value="proxy">proxy — reverse proxy to image backend</option>
          <option value="php">php — FastCGI to PHP-FPM image</option>
        </select>
      </label>
      <label>
        Domains (one per line) {err('domains')}
        <textarea value={domains} onChange={(e) => setDomains(e.target.value)} rows={3} placeholder={'example.com\nwww.example.com'} />
      </label>
      <div className="row">
        <label>
          TLS {err('tls')}
          <select value={tls} onChange={(e) => setTls(e.target.value as 'auto' | 'off')}>
            <option value="auto">auto</option>
            <option value="off">off</option>
          </select>
        </label>
        <label className="check">
          <input type="checkbox" checked={log} onChange={(e) => setLog(e.target.checked)} /> access log
        </label>
      </div>
      {showRoot && (
        <label>
          Root (container path) {err('root')}
          <input value={root} onChange={(e) => setRoot(e.target.value)} placeholder="/var/www/html" />
        </label>
      )}
      {showBackend && (
        <fieldset>
          <legend>Backend image</legend>
          <label>
            Service name (optional, defaults to site id) {err('backend.service')}
            <input value={service} onChange={(e) => setService(e.target.value)} placeholder={id || 'site id'} />
          </label>
          <label>
            Image {err('backend.image')}
            <input value={image} onChange={(e) => setImage(e.target.value)} placeholder={preset === 'php' ? 'php:8.3-fpm' : 'node:20-alpine'} />
          </label>
          {preset === 'proxy' && (
            <label>
              Internal port {err('backend.internalPort')}
              <input value={port} onChange={(e) => setPort(e.target.value)} inputMode="numeric" placeholder="3000" />
            </label>
          )}
          <label>
            Env (KEY=VALUE per line) {err('backend.env')}
            <textarea value={env} onChange={(e) => setEnv(e.target.value)} rows={3} placeholder={'NODE_ENV=production'} />
          </label>
          <p className="hint">
            {preset === 'proxy'
              ? 'Upstream is derived automatically as service:port — no manual field.'
              : 'FastCGI target is derived automatically as service:9000.'}
          </p>
        </fieldset>
      )}
      <div className="row">
        <button onClick={save}>Save</button>
        {!isNew && (
          <button className="danger" onClick={remove}>
            Delete
          </button>
        )}
      </div>
      {notice && <p className="notice">{notice}</p>}
      {snippet && (
        <div>
          <h3>Caddy snippet</h3>
          <pre>{snippet}</pre>
        </div>
      )}
    </div>
  );
}
