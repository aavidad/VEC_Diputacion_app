import { comprobarClaves, forma } from '../preparacion-bases/modelo.js?v=20261003-s2-consulta-v1';
import { validarHuella, validarRevision } from '../preparacion-bases/contrato-http.js?v=20261003-s2-consulta-v2';

export const MAXIMO_BYTES = 4 * 1024 * 1024;
export const CAUSAS = Object.freeze(['fuentes_no_verificadas', 'texto_libre', 'regla_ausente', 'evaluador_pendiente',
  'rechazo_aplicabilidad_pendiente', 'vigencia_aplicabilidad_pendiente', 'dato_no_aportado', 'acreditacion_pendiente']);
export const PENDIENTES = Object.freeze(['bases_universo', 'correspondencia_solicitud', 'lectura_autorizada',
  'identidad_registro_documentos', 'subsanacion_bases', 'aprobacion_listas']);
const prefijo = 'seleccion.admision.';
const id = v => typeof v === 'string' && /^[A-Za-z0-9:._-]{1,256}$/u.test(v);
const titulo = v => typeof v === 'string' && v.trim() === v && v.length > 0
  && new TextEncoder().encode(v).byteLength <= 256 && !/\p{Cc}/u.test(v);
const unico = (v, campo) => new Set(v.map(x => campo ? x[campo] : x)).size === v.length;
const lista = (maximo, validar) => v => Array.isArray(v) && v.length <= maximo && v.every(validar);
const fecha = v => typeof v === 'string' && /^\d{4}-\d{2}-\d{2}$/u.test(v) && !v.startsWith('0000-')
  && Number.isFinite(Date.parse(v)) && new Date(v).toISOString().slice(0, 10) === v;
const versionHecho = v => Number.isSafeInteger(v) && v > 0;
const hecho = v => forma(v, { referencia: id, version: versionHecho });
const requisito = v => forma(v, { referencia: id, version: id, titulo_propuesto: titulo,
  representacion: x => ['texto_libre', 'estructurada'].includes(x), hito_ref: id, hito_fecha: fecha,
  hechos_esperados: lista(32, hecho) }, ['regla_ref', 'regla_version'])
  && unico(v.hechos_esperados, 'referencia')
  && Object.hasOwn(v, 'regla_ref') === Object.hasOwn(v, 'regla_version')
  && (!Object.hasOwn(v, 'regla_ref') || v.representacion === 'estructurada' && id(v.regla_ref) && id(v.regla_version));
const soporte = v => forma(v, { referencia: id, version: versionHecho, fuente_ref: id, fuente_version: id,
  estado_aportado: x => ['declarado', 'pendiente', 'acreditado', 'rechazado'].includes(x),
  evidencias_aportadas: x => Number.isSafeInteger(x) && x >= 0 && x <= 32, desde: fecha }, ['hasta'])
  && (!Object.hasOwn(v, 'hasta') || fecha(v.hasta) && v.hasta >= v.desde);
const revisionRequisito = v => forma(v, { requisito, estado: x => x === 'pendiente',
  causas: lista(CAUSAS.length, x => CAUSAS.some(c => x === `${prefijo}causa.${c}`)),
  soportes: lista(32, soporte), accion_propuesta: x => ['preparar_revision', 'preparar_aportacion'].some(a => x === `${prefijo}accion.${a}`) })
  && v.causas.includes(`${prefijo}causa.fuentes_no_verificadas`) && unico(v.causas)
  && unico(v.soportes, 'referencia') && v.soportes.every(s => v.requisito.hechos_esperados.some(h => h.referencia === s.referencia && h.version === s.version));
const bases = v => forma(v, { referencia: id, version: id, huella_sha256: validarHuella });
const solicitud = v => forma(v, { huella_archivo_original_sha256: validarHuella, identificador_publico: id,
  version: id, huella_sha256: validarHuella, estado: x => x === 'sin_presentar' });

/** Forma del transporte CLI. No evalúa requisitos ni verifica las fuentes. */
export function leerSalida(bytes) {
  if (!(bytes instanceof Uint8Array) || bytes.byteLength === 0 || bytes.byteLength > MAXIMO_BYTES) throw new TypeError('tamano');
  let dto;
  try { const texto = new TextDecoder('utf-8', { fatal: true }).decode(bytes); comprobarClaves(texto); dto = JSON.parse(texto); }
  catch { throw new TypeError('formato'); }
  if (!forma(dto, {
    esquema: x => x === 'vec.seleccion.admision-preparacion.v1', preparacion_ref: id, revision: validarRevision,
    alcance: x => x === 'preparacion_sintetica', estado: x => x === 'pendiente_revision_competente',
    universo_requisitos: x => x === 'propuesto_no_cotejado', bases,
    requisitos: lista(64, revisionRequisito), pendientes: lista(PENDIENTES.length, x => PENDIENTES.some(p => x === `${prefijo}pendiente.${p}`)),
    persistido: x => x === false, admision_oficial: x => x === false,
  }, ['solicitud_contexto', 'version_paquete_hechos'])
    || Object.hasOwn(dto, 'version_paquete_hechos') && !id(dto.version_paquete_hechos)
    || Object.hasOwn(dto, 'solicitud_contexto') && (!solicitud(dto.solicitud_contexto)
      || dto.solicitud_contexto.version !== dto.bases.version || dto.solicitud_contexto.huella_sha256 !== dto.bases.huella_sha256)
    || !unico(dto.requisitos.map(x => x.requisito), 'referencia') || !unico(dto.pendientes)
    || PENDIENTES.some(p => !dto.pendientes.includes(`${prefijo}pendiente.${p}`))) throw new TypeError('formato');
  const hechos = new Map();
  for (const r of dto.requisitos) for (const h of r.requisito.hechos_esperados) {
    if (hechos.has(h.referencia) && hechos.get(h.referencia) !== h.version) throw new TypeError('formato');
    hechos.set(h.referencia, h.version);
  }
  return dto;
}

export async function leerArchivo(archivo) {
  if (!archivo || archivo.size <= 0 || archivo.size > MAXIMO_BYTES) throw new TypeError('tamano');
  const bytes = new Uint8Array(await archivo.arrayBuffer());
  return { bytes, dto: leerSalida(bytes) };
}
