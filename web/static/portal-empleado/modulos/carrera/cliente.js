const VIAS = new Set(['grado', 'progresion', 'promocion']);
const ESTADOS = new Set(['disponible', 'pendiente', 'incompatible']);
const texto = v => typeof v === 'string' && v.length > 0 && v.length <= 1024;
const datoTexto = v => typeof v === 'string' && v.length <= 1024;
const version = v => texto(v) || Number.isSafeInteger(v) && v > 0;
const versionDato = v => datoTexto(v) || Number.isSafeInteger(v) && v > 0;
const clave = (v, grupo) => texto(v) && new RegExp(`^carrera\\.${grupo}\\.[a-z_]+$`).test(v);
const fuente = v => v && datoTexto(v.referencia) && versionDato(v.version);
function invalido() { throw new TypeError('carrera.dto_invalido'); }
export function validarEscenario(dto) {
  if (!dto || dto.alcance !== 'preparacion_sintetica' || !version(dto.version)
    || !fuente(dto.fuente) || !Array.isArray(dto.casos) || dto.casos.length > 100
    || !Array.isArray(dto.pendientes) || dto.pendientes.length > 100 || !dto.pendientes.every(v => clave(v, 'pendiente'))) invalido();
  const refs = new Set();
  for (const c of dto.casos) {
    if (!c || !texto(c.referencia) || refs.has(c.referencia) || !VIAS.has(c.via)
      || c.estado_global !== 'pendiente' || !datoTexto(c.persona_nombre) || !datoTexto(c.regimen)
      || ![c.nivel_puesto, c.grado_personal].every(v => v === null || Number.isSafeInteger(v))
      || !Array.isArray(c.fuentes) || c.fuentes.length > 100 || !c.fuentes.every(fuente)
      || !Array.isArray(c.comprobaciones) || c.comprobaciones.length > 100
      || !c.comprobaciones.every(x => x && clave(x.clave, 'comprobacion') && ESTADOS.has(x.estado) && clave(x.motivo_clave, 'motivo'))
      || !Array.isArray(c.pendientes) || c.pendientes.length > 100 || !c.pendientes.every(v => clave(v, 'pendiente'))) invalido();
    refs.add(c.referencia);
  }
  return dto;
}
export function crearClienteCarrera({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({ async listar({ signal } = {}) {
    const respuesta = await fetchImpl(new URL('./escenario.json?v=20261001-carrera-preparacion-v1', import.meta.url), {
      method: 'GET', credentials: 'omit', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer', signal,
      headers: { Accept: 'application/json' },
    });
    if (!respuesta.ok) throw new Error('carrera.carga_fallida');
    const bytes = await respuesta.text();
    if (bytes.length > 1024 * 1024) throw new Error('carrera.limite');
    return validarEscenario(JSON.parse(bytes));
  } });
}
export function crearBorrador(dto) {
  validarEscenario(dto);
  return JSON.stringify(dto, null, 2);
}
export async function leerErrorCatalogo({ fetchImpl = globalThis.fetch } = {}) {
  const r = await fetchImpl(new URL('./error-catalogo.json?v=20261001-carrera-preparacion-v1', import.meta.url), {
    credentials: 'omit', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer',
  });
  if (!r.ok) throw new Error('carrera.catalogo');
  const bytes = await r.text();
  if (bytes.length > 4096) throw new Error('carrera.catalogo');
  const datos = JSON.parse(bytes);
  if (!texto(datos.titulo) || !texto(datos.mensaje) || !/^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/.test(datos.idioma)) throw new Error('carrera.catalogo');
  return datos;
}
