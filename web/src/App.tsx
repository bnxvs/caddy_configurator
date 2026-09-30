import { useCallback, useEffect, useState } from 'react';
import { ProjectInfo, PreviewResult, StatusResult, api } from './api';
import SiteEditor from './SiteEditor';

type View = { kind: 'site'; id: string } | { kind: 'new' } | { kind: 'preview' };

export default function App() {
  const [project, setProject] = useState<ProjectInfo | null>(null);
  const [view, setView] = useState<View>({ kind: 'preview' });
  const [preview, setPreview] = useState<PreviewResult | null>(null);
  const [status, setStatus] = useState<StatusResult | null>(null);
  const [email, setEmail] = useState('');
  const [message, setMessage] = useState('');
  const [loadError, setLoadError] = useState('');

  const reloadProject = useCallback(async () => {
    const p = await api.getProject();
    setProject(p);
    setEmail(p.email);
  }, []);

  useEffect(() => {
    reloadProject().catch((e: unknown) => setLoadError(e instanceof Error ? e.message : 'load failed'));
  }, [reloadProject]);

  useEffect(() => {
    const t = setInterval(() => {
      api
        .getStatus()
        .then(setStatus)
        .catch(() => undefined);
    }, 5000);
    api
      .getStatus()
      .then(setStatus)
      .catch(() => undefined);
    return () => clearInterval(t);
  }, []);

  async function showPreview() {
    setMessage('');
    try {
      setPreview(await api.getPreview());
      setView({ kind: 'preview' });
    } catch (e: unknown) {
      setMessage(e instanceof Error ? e.message : 'preview failed');
    }
  }

  async function generate() {
    setMessage('');
    try {
      const res = await api.generate();
      setMessage(`Wrote ${res.caddyfilePath} + ${res.composePath}${res.formatted ? ' (formatted)' : ''}.`);
      await reloadProject();
      await showPreview();
    } catch (e: unknown) {
      setMessage(e instanceof Error ? e.message : 'generate failed');
    }
  }

  async function saveEmail() {
    setMessage('');
    try {
      await api.updateProject({ email });
      setMessage('Global email saved.');
      await reloadProject();
    } catch (e: unknown) {
      setMessage(e instanceof Error ? e.message : 'save failed');
    }
  }

  async function acknowledgeDisk() {
    await api.refreshStatus();
    const st = await api.getStatus();
    setStatus(st);
    await reloadProject();
  }

  if (loadError) return <main className="pad">Failed to load project: {loadError}</main>;
  if (!project) return <main className="pad">Loading…</main>;

  return (
    <>
      <header>
        <div>
          <strong>{project.name}</strong> <span className="muted">{project.root}</span>{' '}
          <span className={project.caddyFound ? 'badge ok' : 'badge'}>caddy {project.caddyFound ? 'found' : 'missing'}</span>
          {status?.generatedStale && <span className="badge warn">generated stale</span>}
        </div>
        <div className="row">
          <button onClick={showPreview}>Preview</button>
          <button onClick={generate}>Generate</button>
        </div>
      </header>
      {status && status.changedOnDisk.length > 0 && (
        <div className="banner">
          Changed on disk: {status.changedOnDisk.join(', ')}.{' '}
          <button onClick={acknowledgeDisk}>Reload</button>
        </div>
      )}
      {message && <div className="banner info">{message}</div>}
      <div className="layout">
        <aside>
          <div className="row between">
            <h3>Sites</h3>
            <button onClick={() => setView({ kind: 'new' })}>+ New</button>
          </div>
          <ul>
            {project.sites.map((s) => (
              <li key={s.id}>
                <button
                  className={view.kind === 'site' && view.id === s.id ? 'active' : ''}
                  onClick={() => setView({ kind: 'site', id: s.id })}
                >
                  <strong>{s.id}</strong> <span className="muted">{s.preset}</span>
                  <br />
                  <span className="muted">{s.domains.join(', ')}</span>
                </button>
              </li>
            ))}
          </ul>
          <h3>Global</h3>
          <label>
            ACME email
            <input value={email} onChange={(e) => setEmail(e.target.value)} placeholder="admin@example.com" />
          </label>
          <button onClick={saveEmail}>Save email</button>
        </aside>
        <section>
          {view.kind === 'preview' &&
            (preview ? (
              <div>
                <h2>Preview {preview.formatted && <span className="badge ok">formatted</span>}</h2>
                {preview.warnings.length > 0 && (
                  <ul>
                    {preview.warnings.map((w, i) => (
                      <li key={i}>
                        {w.file}: {w.field} — {w.reason}
                      </li>
                    ))}
                  </ul>
                )}
                <h3>Caddyfile</h3>
                <pre>{preview.caddyfile}</pre>
                <h3>compose.yaml</h3>
                <pre>{preview.compose}</pre>
              </div>
            ) : (
              <p className="muted">Press Preview to render the current sources.</p>
            ))}
          {view.kind === 'site' && (
            <SiteEditor
              key={view.id}
              siteId={view.id}
              onChanged={async () => {
                await reloadProject();
              }}
            />
          )}
          {view.kind === 'new' && (
            <SiteEditor
              key="new"
              siteId={null}
              onChanged={async () => {
                await reloadProject();
                await showPreview();
              }}
            />
          )}
        </section>
      </div>
    </>
  );
}
