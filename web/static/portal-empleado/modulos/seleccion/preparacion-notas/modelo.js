import { comprobarClaves, forma } from '../preparacion-bases/modelo.js?v=20261003-s2-consulta-v1';
import { validarConfiguracion, validarResultado } from '../configuracion.js?v=20261004-codexa-s6-notas-v1';

export const MAXIMO_BYTES = 1024 * 1024;
const pendientes = ['revision_competente', 'acto', 'firma', 'publicacion'];
const referencia = v => typeof v === 'string' && /^[a-z0-9_]{1,64}$/u.test(v);
const puntos = v => v === null || Number.isSafeInteger(v) && v >= 0 && v <= 1000000000;
const clave = n => `${n.solicitud_ref}\0${n.fase_ref}`;
const lista = validar => v => Array.isArray(v) && v.length <= 128 && v.every(validar);
const nota = v => forma(v, { solicitud_ref: referencia, fase_ref: referencia, puntos_micropuntos: puntos });
const cambio = v => forma(v, { solicitud_ref: referencia, fase_ref: referencia, anterior_micropuntos: puntos, propuesta_micropuntos: puntos });
const unica = v => new Set(v.map(clave)).size === v.length;

/** Valida estructura y coherencia local; el archivo no acredita origen ni aprobación. */
export function leerSalida(bytes) {
  if (!(bytes instanceof Uint8Array) || !bytes.byteLength || bytes.byteLength > MAXIMO_BYTES) throw new TypeError('tamano');
  let dto;
  try { const s = new TextDecoder('utf-8', { fatal: true }).decode(bytes); comprobarClaves(s); dto = JSON.parse(s); }
  catch { throw new TypeError('formato'); }
  if (!forma(dto, {
    esquema: v => v === 'seleccion.revision_notas.v1', alcance: v => v === 'preparacion_sintetica', ejemplo_ref: referencia,
    configuracion: v => validarConfiguracion(v).length === 0, notas_antecedente: lista(nota), notas_propuestas: lista(nota),
    antecedente: v => v !== null && typeof v === 'object', propuesta: v => v !== null && typeof v === 'object',
    huella_antecedente_sha256: v => typeof v === 'string' && /^[a-f0-9]{64}$/u.test(v), cambios: lista(cambio),
    pendientes: v => Array.isArray(v) && v.length === pendientes.length && v.every((p, i) => p === pendientes[i]),
  }) || !unica(dto.notas_antecedente) || !unica(dto.notas_propuestas) || !unica(dto.cambios)) throw new TypeError('formato');
  const datos = { configuracion: dto.configuracion, convocatoria_ref: dto.antecedente.convocatoria_ref, bases_version: dto.antecedente.bases_version };
  try { validarResultado(dto.antecedente, datos); validarResultado(dto.propuesta, datos); }
  catch { throw new TypeError('formato'); }
  if (dto.antecedente.solicitudes.length > 128 || dto.antecedente.solicitudes.length !== dto.propuesta.solicitudes.length
    || dto.antecedente.solicitudes.some(a => !dto.propuesta.solicitudes.some(p => p.referencia === a.referencia && p.nombre === a.nombre))) throw new TypeError('formato');
  const existe = n => dto.antecedente.solicitudes.some(s => s.referencia === n.solicitud_ref)
    && dto.configuracion.fases.some(f => f.referencia === n.fase_ref && f.tipo === 'prueba');
  const rango = n => n.puntos_micropuntos === null || dto.configuracion.fases.some(f => f.referencia === n.fase_ref && n.puntos_micropuntos <= f.maximo_micropuntos);
  if ([...dto.notas_antecedente, ...dto.notas_propuestas].some(n => !existe(n) || !rango(n))
    || dto.cambios.some(n => !existe(n) || n.anterior_micropuntos === n.propuesta_micropuntos
      || !dto.notas_propuestas.some(p => clave(p) === clave(n) && p.puntos_micropuntos === n.propuesta_micropuntos))) throw new TypeError('formato');
  return dto;
}
export async function leerArchivo(archivo) {
  if (!archivo || archivo.size <= 0 || archivo.size > MAXIMO_BYTES) throw new TypeError('tamano');
  const bytes = new Uint8Array(await archivo.arrayBuffer());
  return { bytes, dto: leerSalida(bytes) };
}
