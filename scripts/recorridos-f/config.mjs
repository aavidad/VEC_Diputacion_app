import fs from 'node:fs';
import path from 'node:path';
import { createHash } from 'node:crypto';

export const CUADRO = '/api/vec/contratacion-temporal/cuadro/consultas';
export const DETALLE = '/api/vec/contratacion-temporal/expedientes/consultas';
export const BORRADORES = '/api/vec/contratacion-temporal/expedientes/borradores/disponibles';
export const FIRMAS = '/api/vec/contratacion-temporal/firmas-documento/consultas';
export const huella = value => createHash('sha256').update(value).digest('hex');
export const idiomas = JSON.parse(fs.readFileSync(new URL('idiomas.json', import.meta.url)));
const referencia = /^[A-Za-z0-9][A-Za-z0-9:._-]{2,159}$/;
function exigir(ok) { if (!ok) throw new Error('config'); }

// No se leen rutas privadas hasta descartar enlaces, Git y tipos especiales.
export function rutaExterna(value, { fichero = true, privado = false } = {}) {
  exigir(typeof value === 'string' && path.isAbsolute(value) && !value.split('/').includes('..'));
  let actual = '/';
  const partes = value.split('/').filter(Boolean);
  for (const [i, parte] of partes.entries()) {
    actual = path.join(actual, parte);
    const st = fs.lstatSync(actual);
    exigir(!st.isSymbolicLink());
    if (i < partes.length - 1 || !fichero) {
      exigir(st.isDirectory());
      exigir(!fs.existsSync(path.join(actual, '.git')));
      exigir(!(fs.existsSync(path.join(actual, 'HEAD')) && fs.existsSync(path.join(actual, 'objects'))));
    } else {
      exigir(st.isFile() && st.size > 0 && st.size <= 2 * 1024 * 1024 && st.nlink === 1);
    }
    if (i === partes.length - 1 && privado) exigir(st.uid === process.getuid() && (st.mode & 0o077) === 0);
  }
  return value;
}

export function validarOrigen(value) {
  const u = new URL(value);
  exigir(u.protocol === 'https:' && ['127.0.0.1', '[::1]'].includes(u.hostname));
  exigir(!u.username && !u.password && u.pathname === '/' && !u.search && !u.hash);
  return u.origin;
}

export function cargarConfig(file) {
  rutaExterna(file, { privado: true });
  const c = JSON.parse(fs.readFileSync(file, 'utf8'));
  exigir(c.sintetico === true && c.entorno_controlado === true && c.version === 1);
  exigir(c.origenes && typeof c.origenes === 'object');
  c.origenes.interno = validarOrigen(c.origenes.interno);
  c.origenes.externo = validarOrigen(c.origenes.externo);
  exigir(c.origenes.interno !== c.origenes.externo);
  exigir(typeof c.commit_servido === 'string' && /^[0-9a-f]{40}$/.test(c.commit_servido));
  exigir(typeof c.bolsa_ref === 'string' && referencia.test(c.bolsa_ref));
  exigir(typeof c.expediente_ref === 'string' && referencia.test(c.expediente_ref));
  exigir(Object.hasOwn(idiomas.disponibles, c.idioma));
  const certificados = [], claves = [];
  for (const nombre of ['rrhh', 'candidato']) {
    const identidad = c.identidades?.[nombre];
    exigir(identidad && typeof identidad === 'object');
    certificados.push(huella(fs.readFileSync(rutaExterna(identidad.certificado))));
    claves.push(huella(fs.readFileSync(rutaExterna(identidad.clave, { privado: true }))));
  }
  exigir(certificados[0] !== certificados[1] && claves[0] !== claves[1]);
  // Destino explícito, sin Chromium descargado ni ejecutable elegido en el JSON.
  exigir(fs.statSync('/usr/bin/google-chrome').isFile());
  fs.accessSync('/usr/bin/google-chrome', fs.constants.X_OK);
  return c;
}

export function solicitudPermitida(request, origen) {
  try {
    const u = new URL(request.url());
    if (u.origin !== origen || u.username || u.password || !['GET', 'POST'].includes(request.method())) return false;
    if (request.method() === 'GET') return true;
    if (u.search || ![CUADRO, DETALLE, BORRADORES, FIRMAS].includes(u.pathname)) return false;
    const raw = request.postData();
    if (typeof raw !== 'string' || Buffer.byteLength(raw) > 4096) return false;
    const body = JSON.parse(raw);
    const exacto = (v, keys) => v && typeof v === 'object' && !Array.isArray(v)
      && Object.keys(v).length === keys.length && keys.every(k => Object.hasOwn(v, k));
    if (u.pathname === FIRMAS) return exacto(body, ['expediente_ref']) && typeof body.expediente_ref === 'string' && referencia.test(body.expediente_ref);
    if ([DETALLE, BORRADORES].includes(u.pathname)) return exacto(body, ['expediente_ref', 'version_observada'])
      && typeof body.expediente_ref === 'string' && referencia.test(body.expediente_ref)
      && Number.isSafeInteger(body.version_observada) && body.version_observada >= (u.pathname === BORRADORES ? 1 : 0);
    return exacto(body, ['filtros', 'paginacion'])
      && exacto(body.filtros, ['texto', 'estado_clave', 'fase_clave'])
      && Object.values(body.filtros).every(v => typeof v === 'string' && v.length <= 80)
      && exacto(body.paginacion, ['limite', 'cursor'])
      && Number.isSafeInteger(body.paginacion.limite) && body.paginacion.limite >= 1 && body.paginacion.limite <= 100
      && typeof body.paginacion.cursor === 'string' && body.paginacion.cursor.length <= 43;
  } catch { return false; }
}

export function prepararSalida(destino) {
  exigir(typeof destino === 'string' && path.isAbsolute(destino) && !fs.existsSync(destino));
  exigir(!destino.split('/').includes('..'));
  rutaExterna(path.dirname(destino), { fichero: false, privado: true });
  fs.mkdirSync(destino, { mode: 0o700 });
  return destino;
}
