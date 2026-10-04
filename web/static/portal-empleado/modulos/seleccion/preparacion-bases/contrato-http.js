import { forma, contenido, pendiente, ref, comprobarClaves, CAMPOS_REFERENCIA, MAXIMO_BYTES } from './modelo.js?v=20261003-s2-consulta-v1';

export class ErrorConsultaBases extends Error {
  constructor(codigo) { super(codigo); this.codigo = codigo; }
}
const fallo = () => { throw new ErrorConsultaBases('respuesta_incompatible'); };
export const validarReferencia = v => typeof v === 'string' && /^[a-zA-Z0-9:_./-]{1,180}$/u.test(v);
export const validarHuella = v => typeof v === 'string' && /^[a-f0-9]{64}$/u.test(v);
export const validarRevision = v => Number.isSafeInteger(v) && v >= 1 && v <= 1000000;
const id = validarReferencia, huella = validarHuella, revision = validarRevision;
const instante = v => typeof v === 'string' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(v) && Number.isFinite(Date.parse(v)) && !v.startsWith('0001-');

export function validarSelector(entrada) {
  if (!forma(entrada, { modo: v => ['actual', 'exacta'].includes(v), preparacion_ref: id,
    revision: v => entrada.modo === 'actual' ? v === 0 : revision(v),
    huella_material_sha256: v => entrada.modo === 'actual' ? v === '' : huella(v) })) throw new ErrorConsultaBases('selector_invalido');
  return { ...entrada };
}

/** Sólo forma de transporte; la autoridad de material y pendientes permanece en Bolsa. */
export function leerConsulta(bytes, entrada) {
  const selector = validarSelector(entrada);
  if (!(bytes instanceof Uint8Array) || !bytes.byteLength || bytes.byteLength > MAXIMO_BYTES) fallo();
  let dto;
  try { const texto = new TextDecoder('utf-8', { fatal: true }).decode(bytes); comprobarClaves(texto); dto = JSON.parse(texto); } catch { fallo(); }
  const estado = v => forma(v, { preparacion_ref: id, revision, huella_material_sha256: huella });
  const referencia = v => forma(v, { campo: c => CAMPOS_REFERENCIA.includes(c), referencia: ref });
  const material = v => forma(v, { contenido, referencias: r => Array.isArray(r) && r.length <= 10 && r.every(referencia) && new Set(r.map(x => x.campo)).size === r.length });
  const ambito = v => forma(v, { organizacion_ref: x => typeof x === 'string' && /^org_[a-z0-9]{16,80}$/u.test(x) }, ['unidad_gestion_ref'])
    && (!Object.hasOwn(v, 'unidad_gestion_ref') || typeof v.unidad_gestion_ref === 'string' && /^uni_[a-z0-9]{16,80}$/u.test(v.unidad_gestion_ref));
  const recibo = v => forma(v, { recibo_ref: id, historia_ref: id, auditoria_ref: id, evento_ref: id, huella_intencion_sha256: huella, confirmada_en: instante });
  const acceso = v => forma(v, { decision_ref: id, consumo_huella_sha256: huella, auditoria_ref: id, recibo_ref: id, correlacion_ref: id, accedida_en: instante });
  if (!forma(dto, { estado: v => v === 'obtenida', preparacion: v => forma(v, { ambito, estado, material }),
    pendientes: v => Array.isArray(v) && v.length <= 64 && v.every(pendiente), recibo, acceso })) fallo();
  const e = dto.preparacion.estado, pendientes = dto.pendientes;
  if (e.preparacion_ref !== selector.preparacion_ref || selector.modo === 'exacta' && (e.revision !== selector.revision || e.huella_material_sha256 !== selector.huella_material_sha256)
    || new Set(pendientes.map(p => p.campo)).size !== pendientes.length
    || CAMPOS_REFERENCIA.some(c => !pendientes.some(p => p.campo === c && p.codigo.startsWith('referencia_')))
    || ['documentos_admitidos', 'firma_y_custodia', 'acto_aprobacion', 'publicacion_oficial'].some(c => !pendientes.some(p => p.campo === c && p.codigo === 'circuito_pendiente'))) fallo();
  return dto;
}
