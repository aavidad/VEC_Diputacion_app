import fs from 'node:fs';
import path from 'node:path';
import http from 'node:http';
import { once } from 'node:events';

const datos = new Set([
  '/data/demo/dietas/informes.json',
  '/data/demo/dietas/catalogo-rrhh.json',
  '/data/catalogos/dietas/informes-ejemplo-v1.json',
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
  if (!url.startsWith('/web/static/')) return null;
  const ext = path.extname(url).toLowerCase();
  return extensiones.has(ext) ? url.slice(1) : null;
}

export function comprobarFuente(fuente, casos) {
  if (!path.isAbsolute(fuente) || fuente.split(path.sep).includes('Emilio')) throw new Error('fuente_no_admitida');
  const stat = fs.lstatSync(fuente);
  if (!stat.isDirectory() || stat.isSymbolicLink()) throw new Error('fuente_no_admitida');
  if (fs.realpathSync(fuente).split(path.sep).includes('Emilio')) throw new Error('fuente_no_admitida');
  const requeridos = [...Object.values(casos.paginas), ...casos.datos];
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
    let url;
    try { url = new URL(req.url, 'http://127.0.0.1'); } catch { url = null; }
    const consultaIdioma = url && /^\?lang=(?:es|en)$/u.test(url.search)
      && ['/web/static/portal-empleado/modulos/dietas/informes/index.html', '/web/static/portal-empleado/modulos/dietas/catalogo/index.html'].includes(url.pathname);
    const version = url && url.pathname.startsWith('/web/static/') && /^\?v=[A-Za-z0-9._-]{1,80}$/u.test(url.search);
    const relativo = req.method === 'GET' && url && (!url.search || consultaIdioma || version) && !url.hash
      ? ficheroPermitido(url.pathname) : null;
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
