import fs from 'node:fs/promises';
import path from 'node:path';

const MIME = { '.js': 'text/javascript', '.css': 'text/css', '.json': 'application/json' };

export function recursoPermitido(destino, metodo, casos) {
  const u = new URL(destino);
  if (metodo !== 'GET' || u.origin !== casos.origen || u.username || u.password || /%|\/\//u.test(u.pathname)) return null;
  const idioma = u.searchParams.get('lang');
  if (u.pathname === '/fixture' && casos.idiomas.includes(idioma) && u.search === `?lang=${idioma}`) return 'fixture';
  if (u.search && !/^\?v=[A-Za-z0-9-]+$/u.test(u.search)) return null;
  if (/^\/(?:portal-empleado|comun|textos)\/[A-Za-z0-9_./-]+\.(?:js|css|json)$/u.test(u.pathname)
    && !u.pathname.split('/').some(v => v === '..' || v === '.')) return u.pathname.slice(1);
  return null;
}

export async function leerEstatico(raiz, relativo) {
  const ruta = path.resolve(raiz, relativo);
  if (!ruta.startsWith(`${raiz}${path.sep}`)) throw new Error('recurso_fuera_raiz');
  // Ni el fichero ni sus directorios pueden redirigir la lectura fuera de la raíz.
  if (await fs.realpath(ruta) !== ruta) throw new Error('recurso_enlazado');
  const st = await fs.lstat(ruta);
  if (!st.isFile() || st.size > 2 * 1024 * 1024) throw new Error('recurso_invalido');
  return { body: await fs.readFile(ruta), contentType: MIME[path.extname(ruta)] };
}

export async function entregar(route, raiz, html, contadores, casos) {
  const relativo = recursoPermitido(route.request().url(), route.request().method(), casos);
  if (!relativo) { contadores.bloqueadas++; await route.abort(); return; }
  try {
    const recurso = relativo === 'fixture' ? { body: html, contentType: 'text/html' } : await leerEstatico(raiz, relativo);
    await route.fulfill({ status: 200, ...recurso, headers: { 'Cache-Control': 'no-store' } });
    contadores.entregadas++;
  } catch { contadores.bloqueadas++; await route.abort(); }
}

export function guardarAlmacenamiento() {
  window.__almacenamiento = 0;
  const bloquear = () => { window.__almacenamiento++; throw new Error('almacenamiento_offline'); };
  for (const nombre of ['localStorage', 'sessionStorage', 'indexedDB', 'caches']) {
    Object.defineProperty(window, nombre, { get: bloquear });
  }
  Object.defineProperty(document, 'cookie', { get: bloquear, set: bloquear });
}
