import { comprobarClaves, forma } from '../preparacion-bases/modelo.js?v=20261003-s2-consulta-v1';
import { validarHuella, validarRevision } from '../preparacion-bases/contrato-http.js?v=20261003-s2-consulta-v2';

// 2.000 solicitudes con motivos y referencias largos ocupan algo más de 5 MiB
// indentados; 8 MiB cubren ese caso extremo sin abrir archivos arbitrarios.
export const MAXIMO_BYTES = 8 * 1024 * 1024;
const MAXIMO_SOLICITUDES = 2000;
export const ESQUEMA_PROVISIONAL = 'vec.seleccion.lista-admision-provisional.v1';
export const ESQUEMA_DEFINITIVA = 'vec.seleccion.lista-admision-definitiva.v1';
const PENDIENTE_EJEMPLO = 'seleccion.lista_admision.pendiente.catalogo_ejemplo';
// Pendientes obligatorios de cada lista, como los fija el dominio Go.
const OBLIGATORIOS = Object.freeze({
  [ESQUEMA_PROVISIONAL]: ['aprobacion_competente', 'identidad_publicacion', 'vencimiento_al_publicar', 'publicacion_oficial']
    .map(p => `seleccion.lista_admision.pendiente.${p}`),
  [ESQUEMA_DEFINITIVA]: ['aprobacion_competente', 'pie_recursos', 'ultima_revision', 'registro_escritos', 'identidad_publicacion', 'publicacion_oficial']
    .map(p => `seleccion.lista_definitiva.pendiente.${p}`),
});
const id = v => typeof v === 'string' && /^[A-Za-z0-9:._-]{1,256}$/u.test(v);
const codigo = v => typeof v === 'string' && /^[A-Za-z0-9:._-]{1,64}$/u.test(v);
const logico = v => typeof v === 'boolean';
const entero = (min, max) => v => Number.isSafeInteger(v) && v >= min && v <= max;
const lista = (maximo, validar) => v => Array.isArray(v) && v.length <= maximo && v.every(validar);
const cuenta = entero(0, MAXIMO_SOLICITUDES);
const antecedente = v => forma(v, { esquema_material: x => x === 'seleccion.admision.material-local.v1',
  preparacion_ref: id, revision: validarRevision, huella_material_sha256: validarHuella });
const motivo = v => forma(v, { codigo, subsanable: logico });
const motivos = v => lista(16, motivo)(v) && v.length > 0 && new Set(v.map(m => m.codigo)).size === v.length;
const excluidaProvisional = v => forma(v, { antecedente, motivos, subsanable: logico }) && v.subsanable === v.motivos.every(m => m.subsanable);
const VIAS = Object.freeze(['subsanacion', 'reclamacion']);
const excluidaDefinitiva = v => forma(v, { antecedente, motivos, resolucion: x => x === 'desestimada' || x === 'no_presentada' }, ['via'])
  && (v.resolucion === 'desestimada' ? VIAS.includes(v.via) : !Object.hasOwn(v, 'via'));
const admitidaDefinitiva = v => forma(v, { antecedente, origen: x => x === 'provisional' || VIAS.includes(x) });
// Máximos por unidad de Calendarios (calendarios/domain/plazo.go).
const MAXIMO_POR_UNIDAD = Object.freeze({ dias_habiles: 250, dias_naturales: 730, meses: 60, anios: 5 });

const comunes = {
  lista_ref: id, revision: validarRevision,
  alcance: x => x === 'preparacion_sintetica', estado: x => x === 'borrador_pendiente_aprobacion',
  bases: x => forma(x, { referencia: id, version: id, huella_sha256: validarHuella }),
  catalogo: x => forma(x, { referencia: id, version: id, paquete_ejemplo: logico }, ['duda_ref'])
    && (Object.hasOwn(x, 'duda_ref') ? id(x.duda_ref) : !x.paquete_ejemplo),
  pendientes: lista(8, x => typeof x === 'string'),
  aprobada: x => x === false, publicada: x => x === false, persistida: x => x === false,
};
const PROVISIONAL = {
  ...comunes, esquema: x => x === ESQUEMA_PROVISIONAL,
  plazo_subsanacion: x => forma(x, { unidad: u => Object.hasOwn(MAXIMO_POR_UNIDAD, u), cantidad: entero(1, 730) })
    && x.cantidad <= MAXIMO_POR_UNIDAD[x.unidad],
  vencimiento_subsanacion: x => x === 'pendiente_publicacion',
  admitidas: lista(MAXIMO_SOLICITUDES, x => forma(x, { antecedente })),
  excluidas: lista(MAXIMO_SOLICITUDES, excluidaProvisional),
  resumen: x => forma(x, { solicitudes: entero(1, MAXIMO_SOLICITUDES), admitidas: cuenta, excluidas: cuenta, excluidas_subsanables: cuenta }),
};
const DEFINITIVA = {
  ...comunes, esquema: x => x === ESQUEMA_DEFINITIVA,
  provisional: x => forma(x, { esquema: e => e === 'seleccion.admision.lista-provisional.v1', lista_ref: id, revision: validarRevision, huella_sha256: validarHuella }),
  admitidas: lista(MAXIMO_SOLICITUDES, admitidaDefinitiva),
  excluidas: lista(MAXIMO_SOLICITUDES, excluidaDefinitiva),
  resumen: x => forma(x, { solicitudes: entero(1, MAXIMO_SOLICITUDES), admitidas: cuenta, admitidas_tras_escrito: cuenta, excluidas: cuenta, excluidas_sin_escrito: cuenta }),
};

/** Recuentos que el dominio calcula a partir de las propias listas. */
function resumenCoherente(dto) {
  const r = dto.resumen;
  const base = r.admitidas === dto.admitidas.length && r.excluidas === dto.excluidas.length && r.solicitudes === r.admitidas + r.excluidas;
  if (dto.esquema === ESQUEMA_PROVISIONAL) return base && r.excluidas_subsanables === dto.excluidas.filter(e => e.subsanable).length;
  return base && dto.provisional.lista_ref !== dto.lista_ref
    && r.admitidas_tras_escrito === dto.admitidas.filter(a => a.origen !== 'provisional').length
    && r.excluidas_sin_escrito === dto.excluidas.filter(e => e.resolucion === 'no_presentada').length;
}

/** Forma de las salidas `lista-provisional` y `lista-definitiva` del CLI. No verifica huellas ni catálogo. */
export function leerSalida(bytes) {
  if (!(bytes instanceof Uint8Array) || bytes.byteLength === 0 || bytes.byteLength > MAXIMO_BYTES) throw new TypeError('tamano');
  let dto;
  try { const texto = new TextDecoder('utf-8', { fatal: true }).decode(bytes); comprobarClaves(texto); dto = JSON.parse(texto); }
  catch { throw new TypeError('formato'); }
  const campos = dto?.esquema === ESQUEMA_DEFINITIVA ? DEFINITIVA : PROVISIONAL;
  if (!forma(dto, campos) || !resumenCoherente(dto)) throw new TypeError('formato');
  const refs = [...dto.admitidas, ...dto.excluidas].map(x => x.antecedente.preparacion_ref);
  const esperados = [...OBLIGATORIOS[dto.esquema], ...(dto.catalogo.paquete_ejemplo ? [PENDIENTE_EJEMPLO] : [])];
  if (new Set(refs).size !== refs.length || dto.pendientes.length !== esperados.length
    || esperados.some(p => !dto.pendientes.includes(p))) throw new TypeError('formato');
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

/** Catálogo de textos de los motivos de una lista (`motivos-<referencia>`), o null si la referencia no es simple. */
export function moduloTextosMotivos(referencia) {
  return /^[a-z0-9]+(?:-[a-z0-9]+)*$/u.test(referencia) && referencia.length <= 64 ? `motivos-${referencia}` : null;
}
