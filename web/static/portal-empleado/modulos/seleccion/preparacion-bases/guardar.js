import { ErrorConsultaBases, validarReferencia, validarHuella, validarRevision, leerConsulta } from './contrato-http.js?v=VEC_WEB_VERSION';
import { contenido } from './modelo.js?v=VEC_WEB_VERSION';
import { leerBytes, RUTA_GUARDAR_BASES } from './cliente-http.js?v=VEC_WEB_VERSION';

const fallo = codigo => new ErrorConsultaBases(codigo);
export const nuevoMaterial = (titulo, modalidad) => ({ contenido: {
  identificador_publico: '', tipo: '', modalidad_inscripcion: modalidad,
  catalogo_categorias: { catalogo_id: '', catalogo_version: 0, catalogo_huella_sha256: '' },
  categorias: [], titulo, resumen: '', descripcion: '', plazos: [], requisitos: [], documentos: [], ayuda: [],
}, referencias: [] });

export function prepararGuardado(base, titulo, modalidad, uuid = () => globalThis.crypto.randomUUID()) {
  if (typeof titulo !== 'string' || !titulo.trim() || titulo.length > 180 || !['separada', 'conjunta'].includes(modalidad)) throw fallo('solicitud_invalida');
  const esperada = base?.preparacion.estado ?? { preparacion_ref: uuid(), revision: 0, huella_material_sha256: '' };
  if (!validarReferencia(esperada.preparacion_ref) || !(esperada.revision === 0 && esperada.huella_material_sha256 === '' || validarRevision(esperada.revision) && validarHuella(esperada.huella_material_sha256))) throw fallo('solicitud_invalida');
  const material = base ? structuredClone(base.preparacion.material) : nuevoMaterial(titulo.trim(), modalidad);
  material.contenido.titulo = titulo.trim(); material.contenido.modalidad_inscripcion = modalidad;
  if (!contenido(material.contenido)) throw fallo('solicitud_invalida');
  const clave_operacion = uuid();
  if (!validarReferencia(clave_operacion)) throw fallo('solicitud_invalida');
  return Object.freeze({ esperada, material, clave_operacion });
}

export function crearEscritorBasesHTTP({ fetchImpl = globalThis.fetch, plazoMs = 25000 } = {}) {
  if (typeof fetchImpl !== 'function' || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30000) throw fallo('configuracion_invalida');
  return Object.freeze({ async guardar(solicitud, { signal } = {}) {
    if (signal?.aborted) throw fallo('operacion_abortada');
    const controlador = new AbortController(), abortar = () => controlador.abort();
    signal?.addEventListener('abort', abortar, { once: true }); const limite = setTimeout(abortar, plazoMs);
    try {
      const respuesta = await fetchImpl(RUTA_GUARDAR_BASES, { method: 'POST', headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
        body: JSON.stringify(solicitud), mode: 'same-origin', credentials: 'same-origin', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer', signal: controlador.signal });
      if (controlador.signal.aborted) throw fallo('operacion_abortada');
      if (!respuesta || respuesta.redirected) throw fallo('respuesta_incompatible');
      if ([401, 403].includes(respuesta.status)) throw fallo('acceso_denegado');
      if (respuesta.status === 409) throw fallo('version_en_conflicto');
      if (![200, 201].includes(respuesta.status) || !respuesta.ok) throw fallo('servicio_no_disponible');
      if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get('content-type') || '')) throw fallo('respuesta_incompatible');
      const bytes = await leerBytes(respuesta);
      if (controlador.signal.aborted) throw fallo('operacion_abortada');
      const dto = leerConsulta(bytes, solicitud, true);
      if ((respuesta.status === 201) !== (dto.estado === 'guardada')) throw fallo('respuesta_incompatible');
      return dto;
    } catch (error) {
      if (signal?.aborted) throw fallo('operacion_abortada');
      if (controlador.signal.aborted) throw fallo('servicio_no_disponible');
      if (error instanceof ErrorConsultaBases) throw error;
      throw fallo('servicio_no_disponible');
    } finally { clearTimeout(limite); signal?.removeEventListener('abort', abortar); }
  } });
}
