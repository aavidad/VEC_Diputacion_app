import { validarIdentificador, validarLimites, seleccionarFicheros } from '../modelo.js?v=20261001-preparacion-v1';

// Límite técnico del importador local, sin efecto sobre requisitos o plazos de inscripción.
export const MAXIMO_ARCHIVO = 2 * 1024 * 1024;
const ESQUEMA = 'vec.bolsa.preparacion-local.v1';
const camposRaiz = ['esquema', 'estado', 'convocatoria', 'generada_en', 'lectura_confirmada', 'requisitos', 'plazos', 'documentos_publicos', 'archivos_locales'];
function fallo() { throw new TypeError('material_invalido'); }
function claves(v, permitidas, opcionales = []) {
  if (!v || typeof v !== 'object' || Array.isArray(v) || Object.keys(v).some(k => !permitidas.includes(k)) ||
      permitidas.some(k => !opcionales.includes(k) && !Object.hasOwn(v, k))) fallo();
}
function texto(v, maximo = MAXIMO_ARCHIVO, vacio = false) {
  if (typeof v !== 'string' || v.length > maximo || (!vacio && !v.trim())) fallo();
}
function fecha(v) {
  if (typeof v !== 'string' || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u.test(v) || !Number.isFinite(Date.parse(v)) || new Date(v).toISOString().slice(0, 19) !== v.slice(0, 19)) fallo();
}
function lista(v) { if (!Array.isArray(v)) fallo(); }
function inmovilizar(v) {
  if (v && typeof v === 'object') { Object.values(v).forEach(inmovilizar); Object.freeze(v); }
  return v;
}
export function validarResumen(resumen, limites) {
  claves(resumen, camposRaiz);
  if (resumen.esquema !== ESQUEMA || resumen.estado !== 'sin_presentar' || typeof resumen.lectura_confirmada !== 'boolean') fallo();
  claves(resumen.convocatoria, ['identificador_publico', 'titulo', 'version', 'huella_sha256']);
  if (!validarIdentificador(resumen.convocatoria.identificador_publico) || typeof resumen.convocatoria.huella_sha256 !== 'string' || !/^[a-f0-9]{64}$/u.test(resumen.convocatoria.huella_sha256)) fallo();
  texto(resumen.convocatoria.titulo); texto(resumen.convocatoria.version); fecha(resumen.generada_en);
  for (const k of ['requisitos', 'plazos', 'documentos_publicos', 'archivos_locales']) lista(resumen[k]);
  for (const r of resumen.requisitos) {
    claves(r, ['titulo', 'descripcion', 'obligatorio', 'cumplimiento']); texto(r.titulo); texto(r.descripcion);
    if (typeof r.obligatorio !== 'boolean' || r.cumplimiento !== 'pendiente') fallo();
  }
  for (const p of resumen.plazos) {
    claves(p, ['titulo', 'abre_en', 'cierra_en', 'etiqueta_situacion', 'descripcion'], ['descripcion']);
    texto(p.titulo); fecha(p.abre_en); fecha(p.cierra_en); texto(p.etiqueta_situacion);
    if (Object.hasOwn(p, 'descripcion')) texto(p.descripcion, MAXIMO_ARCHIVO, true);
  }
  for (const d of resumen.documentos_publicos) {
    claves(d, ['titulo', 'url']); texto(d.titulo);
    if (!globalThis.VECBolsaContratoV2.urlDocumentoPublicoValida(d.url)) fallo();
  }
  for (const a of resumen.archivos_locales) claves(a, ['nombre', 'tamano']);
  seleccionarFicheros(resumen.archivos_locales.map(a => ({ name: a.nombre, size: a.tamano })), validarLimites(limites));
  return inmovilizar(resumen);
}
function textoUTF8(bytes) {
  try { return new TextDecoder('utf-8', { fatal: true }).decode(bytes); } catch { fallo(); }
}
// JSON.parse valida la gramática; este recorrido rechaza claves repetidas, incluso escapadas.
function leerJSON(texto) {
  let valor;
  try { valor = JSON.parse(texto); } catch { fallo(); }
  const pila = [];
  for (let i = 0; i < texto.length; i += 1) {
    const c = texto[i];
    if (c === '{' || c === '[') { pila.push(c === '{' ? new Set() : null); if (pila.length > 8) fallo(); }
    else if (c === '}' || c === ']') pila.pop();
    else if (c === '"') {
      const inicio = i; i += 1;
      for (; i < texto.length; i += 1) { if (texto[i] === '\\') i += 1; else if (texto[i] === '"') break; }
      let j = i + 1; while (/\s/u.test(texto[j] ?? '') && j < texto.length) j += 1;
      if (texto[j] === ':') {
        const clave = JSON.parse(texto.slice(inicio, i + 1)); const claves = pila.at(-1);
        if (!claves || claves.has(clave)) fallo(); claves.add(clave);
      }
    }
  }
  return valor;
}

export function recuperarResumen(bytes, limites) {
  if (!(bytes instanceof Uint8Array) || bytes.length === 0 || bytes.length > MAXIMO_ARCHIVO) fallo();
  const original = new Uint8Array(bytes);
  const resumen = validarResumen(leerJSON(textoUTF8(original)), limites);
  return Object.freeze({ resumen, copiarOriginal: () => original.slice() });
}
// Puerto del lector local: no abre rutas, URLs ni archivos mencionados dentro del material.
export async function leerArchivoLocal(archivo, limites) {
  if (!archivo || !Number.isSafeInteger(archivo.size) || archivo.size < 1 || archivo.size > MAXIMO_ARCHIVO || typeof archivo.arrayBuffer !== 'function') fallo();
  const bytes = new Uint8Array(await archivo.arrayBuffer());
  if (bytes.length !== archivo.size) fallo();
  return recuperarResumen(bytes, limites);
}
