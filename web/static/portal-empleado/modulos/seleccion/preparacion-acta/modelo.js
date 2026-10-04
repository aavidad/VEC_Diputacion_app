import { comprobarClaves, forma } from '../preparacion-bases/modelo.js?v=20261003-s2-consulta-v1';
import { validarReferencia, validarRevision, validarHuella } from '../preparacion-bases/contrato-http.js?v=20261003-s2-consulta-v2';

export const MAXIMO_BYTES = 4 * 1024 * 1024;
export const FIJOS = Object.freeze({ antecedente_tribunal: 'antecedente_no_cotejado', fase_propuesta: 'pertenencia_no_verificada',
  designacion: 'circuito_pendiente', habilitacion: 'circuito_pendiente', sesion_celebrada: 'circuito_pendiente',
  asistencia: 'circuito_pendiente', deliberaciones: 'circuito_pendiente', acuerdos_adoptados: 'circuito_pendiente',
  aprobacion: 'circuito_pendiente', firma: 'circuito_pendiente' });
const AUSENTES = ['sesion_ref', 'fecha_propuesta', 'orden_dia_propuesto', 'acuerdos_propuestos', 'textos_orden_dia', 'textos_acuerdos'];
const cadena = v => typeof v === 'string' && new TextEncoder().encode(v).byteLength <= 65536;
const texto = v => cadena(v) && new TextEncoder().encode(v).byteLength <= 4096
  && !/\p{Cc}/u.test(v) && !/[\uD800-\uDFFF]/u.test(v);
const lista = (validar, maximo = 100) => v => Array.isArray(v) && v.length <= maximo && v.every(validar);
const unico = (v, campo) => new Set(v.map(x => x[campo])).size === v.length;
const fecha = v => typeof v === 'string' && v.length <= 40 && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(v)
  && Number.isFinite(Date.parse(v)) && new Date(`${v.slice(0, 10)}T12:00:00Z`).toISOString().slice(0, 10) === v.slice(0, 10);
const punto = v => forma(v, { punto_ref: validarReferencia, texto_propuesto: texto });
const acuerdo = v => forma(v, { propuesta_ref: validarReferencia, punto_ref: validarReferencia, texto_propuesto: texto });
const antecedente = v => forma(v, { identidad_material: validarReferencia, version_material: validarRevision, huella_aportada_sha256: validarHuella });
const material = v => forma(v, { alcance: x => x === 'preparacion_sintetica', identidad_material: validarReferencia,
  version_material: validarRevision, antecedente_tribunal: antecedente, fase_propuesta: validarReferencia,
  sesion_ref: x => x === '' || validarReferencia(x), orden_dia_propuesto: lista(punto), acuerdos_propuestos: lista(acuerdo) }, ['fecha_propuesta'])
  && (!Object.hasOwn(v, 'fecha_propuesta') || fecha(v.fecha_propuesta))
  && unico(v.orden_dia_propuesto, 'punto_ref') && unico(v.acuerdos_propuestos, 'propuesta_ref')
  && v.acuerdos_propuestos.every(a => v.orden_dia_propuesto.some(p => p.punto_ref === a.punto_ref));
const pendiente = v => forma(v, { campo: x => Object.hasOwn(FIJOS, x) || AUSENTES.includes(x),
  codigo: x => ['antecedente_no_cotejado', 'pertenencia_no_verificada', 'circuito_pendiente', 'material_ausente'].includes(x) });
const mensaje = v => forma(v, { campo: cadena, codigo: cadena, mensaje: cadena });

/** Valida sólo transporte y enlaces locales; nunca acredita una sesión ni sus acuerdos. */
export function leerSalida(bytes) {
  if (!(bytes instanceof Uint8Array) || !bytes.byteLength || bytes.byteLength > MAXIMO_BYTES) throw new TypeError('tamano');
  let dto;
  try { const s = new TextDecoder('utf-8', { fatal: true }).decode(bytes); comprobarClaves(s); dto = JSON.parse(s); }
  catch { throw new TypeError('formato'); }
  const preparacion = v => forma(v, { estado: x => x === 'borrador_propuesto', material_propuesto: material, pendientes: lista(pendiente, 16) });
  if (!forma(dto, { titulo: cadena, preparacion, limite: cadena, mensajes: lista(mensaje, 16) })) throw new TypeError('formato');
  const p = dto.preparacion.pendientes, m = dto.preparacion.material_propuesto;
  const esperados = { ...FIJOS };
  for (const [campo, falta] of Object.entries({ sesion_ref: !m.sesion_ref, fecha_propuesta: !m.fecha_propuesta,
    orden_dia_propuesto: !m.orden_dia_propuesto.length, acuerdos_propuestos: !m.acuerdos_propuestos.length,
    textos_orden_dia: m.orden_dia_propuesto.some(x => !x.texto_propuesto.trim()), textos_acuerdos: m.acuerdos_propuestos.some(x => !x.texto_propuesto.trim()) })) {
    if (falta) esperados[campo] = 'material_ausente';
  }
  if (!unico(p, 'campo') || p.length !== Object.keys(esperados).length || p.length !== dto.mensajes.length
    || p.some((item, i) => !Object.hasOwn(esperados, item.campo) || item.codigo !== esperados[item.campo]
      || dto.mensajes[i].campo !== item.campo || dto.mensajes[i].codigo !== item.codigo)) throw new TypeError('formato');
  return dto;
}
export async function leerArchivo(archivo) {
  if (!archivo || archivo.size <= 0 || archivo.size > MAXIMO_BYTES) throw new TypeError('tamano');
  const bytes = new Uint8Array(await archivo.arrayBuffer());
  return { bytes, dto: leerSalida(bytes) };
}
