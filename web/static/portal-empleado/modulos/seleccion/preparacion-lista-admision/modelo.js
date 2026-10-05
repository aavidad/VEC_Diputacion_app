import { comprobarClaves, forma } from '../preparacion-bases/modelo.js?v=20261003-s2-consulta-v1';
import { validarHuella, validarRevision } from '../preparacion-bases/contrato-http.js?v=20261003-s2-consulta-v2';

// La lista admite 2.000 solicitudes; su salida indentada cabe holgadamente aquí.
export const MAXIMO_BYTES = 4 * 1024 * 1024;
const MAXIMO_SOLICITUDES = 2000;
export const PENDIENTES = Object.freeze(['aprobacion_competente', 'identidad_publicacion', 'vencimiento_al_publicar', 'publicacion_oficial', 'catalogo_ejemplo']);
const OBLIGATORIOS = PENDIENTES.slice(0, 4);
const prefijo = 'seleccion.lista_admision.pendiente.';
const id = v => typeof v === 'string' && /^[A-Za-z0-9:._-]{1,256}$/u.test(v);
const codigo = v => typeof v === 'string' && /^[A-Za-z0-9:._-]{1,64}$/u.test(v);
const logico = v => typeof v === 'boolean';
const entero = (min, max) => v => Number.isSafeInteger(v) && v >= min && v <= max;
const lista = (maximo, validar) => v => Array.isArray(v) && v.length <= maximo && v.every(validar);
const antecedente = v => forma(v, { esquema_material: x => x === 'seleccion.admision.material-local.v1',
  preparacion_ref: id, revision: validarRevision, huella_material_sha256: validarHuella });
const motivo = v => forma(v, { codigo, subsanable: logico });
const excluida = v => forma(v, { antecedente, motivos: lista(16, motivo), subsanable: logico })
  && v.motivos.length > 0 && new Set(v.motivos.map(m => m.codigo)).size === v.motivos.length
  && v.subsanable === v.motivos.every(m => m.subsanable);
const UNIDADES = Object.freeze(['dias_habiles', 'dias_naturales', 'meses', 'anios']);

/** Forma de la salida `lista-provisional` del CLI. No verifica huellas ni catálogo. */
export function leerSalida(bytes) {
  if (!(bytes instanceof Uint8Array) || bytes.byteLength === 0 || bytes.byteLength > MAXIMO_BYTES) throw new TypeError('tamano');
  let dto;
  try { const texto = new TextDecoder('utf-8', { fatal: true }).decode(bytes); comprobarClaves(texto); dto = JSON.parse(texto); }
  catch { throw new TypeError('formato'); }
  const valido = forma(dto, {
    esquema: x => x === 'vec.seleccion.lista-admision-provisional.v1', lista_ref: id, revision: validarRevision,
    alcance: x => x === 'preparacion_sintetica', estado: x => x === 'borrador_pendiente_aprobacion',
    bases: x => forma(x, { referencia: id, version: id, huella_sha256: validarHuella }),
    catalogo: x => forma(x, { referencia: id, version: id, paquete_ejemplo: logico }, ['duda_ref']) && (!Object.hasOwn(x, 'duda_ref') || id(x.duda_ref)),
    plazo_subsanacion: x => forma(x, { unidad: u => UNIDADES.includes(u), cantidad: entero(1, 730) }),
    vencimiento_subsanacion: x => x === 'pendiente_publicacion',
    admitidas: lista(MAXIMO_SOLICITUDES, x => forma(x, { antecedente })),
    excluidas: lista(MAXIMO_SOLICITUDES, excluida),
    resumen: x => forma(x, { solicitudes: entero(1, MAXIMO_SOLICITUDES), admitidas: entero(0, MAXIMO_SOLICITUDES),
      excluidas: entero(0, MAXIMO_SOLICITUDES), excluidas_subsanables: entero(0, MAXIMO_SOLICITUDES) }),
    pendientes: lista(PENDIENTES.length, x => PENDIENTES.some(p => x === prefijo + p)),
    aprobada: x => x === false, publicada: x => x === false, persistida: x => x === false,
  });
  if (!valido) throw new TypeError('formato');
  const refs = [...dto.admitidas, ...dto.excluidas].map(x => x.antecedente.preparacion_ref);
  const r = dto.resumen;
  if (new Set(refs).size !== refs.length || r.admitidas !== dto.admitidas.length || r.excluidas !== dto.excluidas.length
    || r.solicitudes !== refs.length || r.excluidas_subsanables !== dto.excluidas.filter(e => e.subsanable).length
    || new Set(dto.pendientes).size !== dto.pendientes.length || OBLIGATORIOS.some(p => !dto.pendientes.includes(prefijo + p))
    || dto.pendientes.includes(prefijo + 'catalogo_ejemplo') !== dto.catalogo.paquete_ejemplo) throw new TypeError('formato');
  return dto;
}

export async function leerArchivo(archivo) {
  if (!archivo || archivo.size <= 0 || archivo.size > MAXIMO_BYTES) throw new TypeError('tamano');
  const bytes = new Uint8Array(await archivo.arrayBuffer());
  return { bytes, dto: leerSalida(bytes) };
}

/** Textos de motivos de un catálogo: código → texto. Lo que no cumpla se ignora entero. */
export function leerTextosMotivos(datos) {
  if (!datos || typeof datos !== 'object' || Array.isArray(datos)) return new Map();
  const entradas = Object.entries(datos);
  if (entradas.length > 64 || entradas.some(([c, t]) => !codigo(c) || typeof t !== 'string' || !t.trim() || t.length > 300)) return new Map();
  return new Map(entradas);
}

/** Ruta relativa de los textos de motivos del catálogo, por idioma. */
export function rutaTextosMotivos(idioma, referencia) {
  if (!/^[a-z]{2,3}(-[a-z0-9]{2,8})*$/u.test(idioma) || !/^[a-z0-9-]{1,64}$/u.test(referencia)) return null;
  return `../../../../textos/${idioma}/motivos-${referencia}.json`;
}
