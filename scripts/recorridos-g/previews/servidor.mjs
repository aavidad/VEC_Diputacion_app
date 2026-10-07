import fs from 'node:fs';
import path from 'node:path';
import http from 'node:http';
import { once } from 'node:events';

const casos = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url), 'utf8'));
const idiomas = JSON.parse(fs.readFileSync(new URL('idiomas.json', import.meta.url), 'utf8'));
const datos = new Set(casos.datos);
const scriptsPreview = new Set([
  '/scripts/recorridos-g/previews/paginas/dietas/catalogo/demo.js',
  '/scripts/recorridos-g/previews/paginas/dietas/informes/demo.js',
]);
const paginasPreview = new Set([
  '/scripts/recorridos-g/previews/paginas/dietas/catalogo/index.html',
  '/scripts/recorridos-g/previews/paginas/dietas/informes/index.html',
]);
const extensiones = new Map([
  ['.html', 'text/html; charset=utf-8'], ['.js', 'text/javascript; charset=utf-8'],
  ['.css', 'text/css; charset=utf-8'], ['.json', 'application/json; charset=utf-8'],
  ['.svg', 'image/svg+xml'], ['.png', 'image/png'], ['.jpg', 'image/jpeg'],
  ['.jpeg', 'image/jpeg'], ['.webp', 'image/webp'], ['.woff', 'font/woff'],
  ['.woff2', 'font/woff2'],
]);

export function ficheroPermitido(url) {
  if (typeof url !== 'string' || !url.startsWith('/') || url.includes('?') || url.includes('#') || url.includes('%')) return null;
  const partes = url.slice(1).split('/');
  if (partes.some(p => !p || p === '.' || p === '..' || p.startsWith('.'))) return null;
  if (datos.has(url)) return url.slice(1);
  if (paginasPreview.has(url) || scriptsPreview.has(url)) return url.slice(1);
  if (!url.startsWith('/web/static/')) return null;
  const ext = path.extname(url).toLowerCase();
  return extensiones.has(ext) ? url.slice(1) : null;
}

export function peticionPermitida(raw, metodo, origen) {
  let url;
  try { url = new URL(raw, origen); } catch { return false; }
  if (url.origin !== origen || metodo !== 'GET' || url.hash || !ficheroPermitido(url.pathname)) return false;
  if (!url.search) return true;
  if (paginasPreview.has(url.pathname) && Object.values(casos.paginas).includes(url.pathname)) {
    const valor = url.searchParams.get('lang');
    return url.searchParams.size === 1 && Object.hasOwn(idiomas.disponibles, valor)
      && url.search === `?lang=${encodeURIComponent(valor)}`;
  }
  return (url.pathname.startsWith('/web/static/') || scriptsPreview.has(url.pathname))
    && /^\?v=[A-Za-z0-9._-]{1,80}$/u.test(url.search);
}

export function comprobarFuente(fuente, casos) {
  const paginasDeclaradas = Object.values(casos.paginas);
  if (paginasDeclaradas.length !== paginasPreview.size || paginasDeclaradas.some(ruta => !paginasPreview.has(ruta)))
    throw new Error('ruta_no_admitida');
  if (!path.isAbsolute(fuente) || fuente.split(path.sep).includes('Emilio')) throw new Error('fuente_no_admitida');
  const stat = fs.lstatSync(fuente);
  if (!stat.isDirectory() || stat.isSymbolicLink()) throw new Error('fuente_no_admitida');
  if (fs.realpathSync(fuente).split(path.sep).includes('Emilio')) throw new Error('fuente_no_admitida');
  const requeridos = [...paginasDeclaradas, ...scriptsPreview, ...casos.datos];
  const ausentes = [];
  for (const url of requeridos) {
    const relativo = ficheroPermitido(url);
    if (!relativo) throw new Error('ruta_no_admitida');
    const absoluto = path.join(fuente, relativo);
    try {
      let actual = fuente;
      for (const parte of relativo.split('/').slice(0, -1)) {
        actual = path.join(actual, parte);
        const directorio = fs.lstatSync(actual);
        if (!directorio.isDirectory() || directorio.isSymbolicLink()) throw new Error('enlace');
      }
      const st = fs.lstatSync(absoluto);
      if (!st.isFile() || st.isSymbolicLink() || st.size === 0 || st.size > 2 * 1024 * 1024) ausentes.push(url);
    } catch { ausentes.push(url); }
  }
  return ausentes;
}

export function crearServidor(fuente) {
  let peticiones = 0;
  let denegadas = 0;
  const servidor = http.createServer((req, res) => {
    const origen = `http://127.0.0.1:${servidor.address().port}`;
    const relativo = peticionPermitida(req.url, req.method, origen)
      ? ficheroPermitido(new URL(req.url, origen).pathname) : null;
    if (!relativo) { denegadas++; res.writeHead(403, { 'Cache-Control': 'no-store' }); res.end(); return; }
    const absoluto = path.join(fuente, relativo);
    try {
      const st = fs.lstatSync(absoluto);
      if (!st.isFile() || st.isSymbolicLink() || st.size > 2 * 1024 * 1024) throw new Error('recurso');
      // Todo el árbol servido debe ser una copia sintética controlada, sin enlaces.
      let actual = fuente;
      for (const parte of relativo.split('/').slice(0, -1)) {
        actual = path.join(actual, parte);
        if (fs.lstatSync(actual).isSymbolicLink()) throw new Error('enlace');
      }
      peticiones++;
      res.writeHead(200, {
        'Content-Type': extensiones.get(path.extname(absoluto).toLowerCase()),
        'Cache-Control': 'no-store',
        'Content-Security-Policy': "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; form-action 'none'; base-uri 'none'",
        'X-Content-Type-Options': 'nosniff',
      });
      fs.createReadStream(absoluto).pipe(res);
    } catch { denegadas++; res.writeHead(404, { 'Cache-Control': 'no-store' }); res.end(); }
  });
  return {
    servidor,
    escuchar: async () => { servidor.listen(0, '127.0.0.1'); await once(servidor, 'listening'); return `http://127.0.0.1:${servidor.address().port}`; },
    cerrar: async () => { servidor.closeAllConnections(); await new Promise(resolve => servidor.close(resolve)); },
    contadores: () => ({ peticiones, denegadas }),
  };
}
