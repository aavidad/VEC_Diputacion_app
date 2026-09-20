/** Contrato mínimo del catálogo profesional consultable de Personal. */
export const CAPACIDAD_CONSULTAR_PUESTO = "personal.puesto.read";
export const RUTA_CATEGORIAS_PROFESIONALES = "/api/vec/personal/categories";

const PATRON_CLAVE = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/u;
const PATRON_AREA = /^[a-z][a-z0-9_]{0,79}$/u;
const PATRON_ID = /^[a-z][a-z0-9._-]{0,127}$/u;
const PATRON_HUELLA = /^[a-f0-9]{64}$/u;
const PATRON_REVISION = /^[a-z0-9][a-z0-9._-]{0,79}$/u;

function registro(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor) && Object.getPrototypeOf(valor) === Object.prototype; }
function camposExactos(valor, campos) { return registro(valor) && Object.keys(valor).length === campos.length && Object.keys(valor).every((clave) => campos.includes(clave)); }
function texto(valor, maximo) { return typeof valor === "string" && valor.length > 0 && valor.length <= maximo && valor === valor.trim() && !/[\u0000-\u001F\u007F-\u009F]/u.test(valor); }

export function validarConsultaCategorias(consulta = {}) {
  if (!camposExactos(consulta, ["q", "area", "limit", "offset"])
    || typeof consulta.q !== "string" || consulta.q.length > 100 || consulta.q !== consulta.q.trim()
    || typeof consulta.area !== "string" || !PATRON_AREA.test(consulta.area) && consulta.area !== ""
    || !Number.isSafeInteger(consulta.limit) || consulta.limit < 1 || consulta.limit > 500
    || !Number.isSafeInteger(consulta.offset) || consulta.offset < 0) throw new TypeError("consulta de categorías profesionales no válida");
  return Object.freeze({ ...consulta });
}

export function validarPaginaCategorias(datos, consulta) {
  const itemsRespuesta = datos?.items === null ? [] : datos?.items;
  const esperados = Number.isSafeInteger(datos?.total) && Number.isSafeInteger(datos?.offset)
    ? Math.min(datos.limit, Math.max(0, datos.total - datos.offset)) : -1;
  if (!camposExactos(datos, ["items", "total", "limit", "offset", "catalogo", "fuente"])
    || !Array.isArray(itemsRespuesta) || (datos.items === null && esperados !== 0)
    || !Number.isSafeInteger(datos.total) || datos.total < 0
    || datos.limit !== consulta.limit || datos.offset !== consulta.offset || !camposExactos(datos.catalogo, ["catalogo_id", "catalogo_version", "catalogo_huella_sha256"])
    || !PATRON_ID.test(datos.catalogo.catalogo_id) || !Number.isSafeInteger(datos.catalogo.catalogo_version) || datos.catalogo.catalogo_version < 1 || !PATRON_HUELLA.test(datos.catalogo.catalogo_huella_sha256)
    || !camposExactos(datos.fuente, ["revision", "actualizada_en", "demostracion", "aviso"]) || !PATRON_REVISION.test(datos.fuente.revision)
    || typeof datos.fuente.actualizada_en !== "string" || Number.isNaN(Date.parse(datos.fuente.actualizada_en)) || typeof datos.fuente.demostracion !== "boolean" || typeof datos.fuente.aviso !== "string" || datos.fuente.aviso.length > 4096
    || itemsRespuesta.length !== esperados) throw new TypeError("página de categorías profesionales incompatible");
  const claves = new Set();
  const items = itemsRespuesta.map((item) => {
    const campos = ["catalog", "clave", "slug", "etiqueta", "name", "orden", "area", "area_etiqueta", "source", "module_key", "state", "usage"];
    const tieneDescripcion = Object.hasOwn(item ?? {}, "descripcion");
    if (!(camposExactos(item, tieneDescripcion ? [...campos, "descripcion"] : campos)
      && item.catalog === "categoria_profesional" && PATRON_CLAVE.test(item.clave) && item.slug === item.clave && texto(item.etiqueta, 512) && item.name === item.etiqueta && (!tieneDescripcion || typeof item.descripcion === "string" && item.descripcion.length <= 8192) && Number.isSafeInteger(item.orden) && item.orden >= 1 && PATRON_AREA.test(item.area) && texto(item.area_etiqueta, 512) && item.source === "catalogo_gobernado_vec" && item.module_key === "vec.module.personal" && texto(item.state, 512) && texto(item.usage, 512) && !claves.has(item.clave))) throw new TypeError("categoría profesional incompatible");
    claves.add(item.clave); return Object.freeze({ ...item, descripcion: item.descripcion ?? "" });
  });
  return Object.freeze({ items: Object.freeze(items), total: datos.total, limit: datos.limit, offset: datos.offset, catalogo: Object.freeze({ ...datos.catalogo }), fuente: Object.freeze({ ...datos.fuente }) });
}
