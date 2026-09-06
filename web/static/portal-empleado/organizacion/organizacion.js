export const API_ORGANIZACION = "/api/vec/contratacion-temporal/organizacion";
export const FUENTE_RPT = "https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/a-informacion-institucional-y-organizativa/a3-personal/relacion-de-Puestos-de-trabajo/";
export const ESQUEMA_ORGANIZACION = "personal.estructura_organizativa.v1";
export const LIMITE_UNIDADES = 1000;
export const LIMITE_RESPUESTA = 512 * 1024;

export const TEXTOS = Object.freeze({
  eyebrow: "Organización de referencia · Preparación",
  title: "Organización de referencia en preparación",
  intro: "Consulta de la estructura organizativa de referencia para la gestión de Recursos Humanos.",
  back: "Volver al Portal del Empleado", provenanceDetail: "Procedencia y alcance de los datos",
  traceability: "Versión y trazabilidad del catálogo",
  noticeTitle: "Procedencia del catálogo",
  notice: "No acredita ocupantes, dependencia funcional ni permisos de ratificación. La edición desde la pantalla aún no está disponible.",
  catalogId: "Catálogo", version: "Versión", fingerprint: "Huella SHA-256", status: "Estado",
  tableTitle: "Unidades organizativas", filterText: "Filtrar por texto", filterType: "Filtrar por tipo",
  allTypes: "Todos los tipos", delegacion: "Agrupación", centro: "Centro", puesto: "Puesto de responsabilidad",
  tableCaption: "Unidades de la versión consultada", type: "Tipo", code: "Código / clave",
  name: "Denominación", adscription: "Adscripción", page: "Página PDF", sourceLabel: "Fuente declarada:",
  sourceLink: "Consultar transparencia", note: "Se muestra el código de la fuente cuando existe; en su defecto, la clave técnica del catálogo. No identifica personas ni individualiza puestos.",
  loading: "Cargando organización…", error: "No se pudo cargar la organización.", retry: "Reintentar",
  empty: "No hay unidades que coincidan con los filtros.", count: "{visible} de {total} unidades", draft: "Borrador",
});
const TYPES = new Set(["delegacion", "centro", "puesto_responsabilidad"]);
const text = (value) => String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
export function validarOrganizacion(payload, bytes = 0) {
  if (bytes > LIMITE_RESPUESTA) throw new Error("respuesta demasiado grande");
  const d = payload?.data;
  if (!d || d.esquema !== ESQUEMA_ORGANIZACION || d.estado !== "borrador" || !Number.isInteger(d.catalogo_version) || d.catalogo_version < 1 || typeof d.catalogo_id !== "string" || typeof d.catalogo_huella_sha256 !== "string" || !/^[a-f0-9]{64}$/u.test(d.catalogo_huella_sha256) || typeof d.fuente_ref !== "string" || typeof d.descripcion !== "string" || !Array.isArray(d.unidades) || d.unidades.length > LIMITE_UNIDADES) throw new Error("respuesta de organización no válida");
  return { ...d, unidades: d.unidades.map((u) => { if (!u || typeof u.clave !== "string" || typeof u.etiqueta !== "string" || !TYPES.has(u.tipo) || (u.adscripcion_clave !== undefined && typeof u.adscripcion_clave !== "string") || (u.codigo_fuente !== undefined && typeof u.codigo_fuente !== "string") || (u.pagina_fuente !== undefined && (!Number.isInteger(u.pagina_fuente) || u.pagina_fuente < 1))) throw new Error("unidad organizativa no válida"); return u; }) };
}
export function crearCliente(fetchImpl = globalThis.fetch, timeoutMs = 10000) { return { async obtener() { const controller = new AbortController(); const timer = setTimeout(() => controller.abort(), timeoutMs); try { const response = await fetchImpl(API_ORGANIZACION, { credentials:"same-origin", redirect:"error", cache:"no-store", signal:controller.signal }); if (!response.ok) throw new Error(`HTTP ${response.status}`); if (!response.body?.getReader) throw new Error("respuesta sin flujo legible"); const reader = response.body.getReader(); const chunks = []; let total = 0; const decoder = new TextDecoder(); while (true) { const parte = await reader.read(); if (parte.done) break; total += parte.value.byteLength; if (total > LIMITE_RESPUESTA) { await reader.cancel(); throw new Error("respuesta demasiado grande"); } chunks.push(parte.value); } const bytes = new Uint8Array(total); let offset = 0; for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length; } return validarOrganizacion(JSON.parse(decoder.decode(bytes)), total); } finally { clearTimeout(timer); } } }; }
const traducirTipo = (tipo) => ({ delegacion:TEXTOS.delegacion, centro:TEXTOS.centro, puesto_responsabilidad:TEXTOS.puesto })[tipo];
const normalizar = (value) => String(value ?? "").normalize("NFD").replace(/[\u0300-\u036f]/gu, "").toLocaleLowerCase("es");
export function filtrarUnidades(unidades, filtro, tipo) { const padres = new Map(unidades.map((u) => [u.clave, u.etiqueta])); const needle = normalizar(filtro.trim()); return unidades.filter((u) => (!tipo || u.tipo === tipo) && (!needle || [u.clave,u.etiqueta,u.adscripcion_clave,u.codigo_fuente,padres.get(u.adscripcion_clave)].some((v) => normalizar(v).includes(needle)))); }
function mostrarTextos() { document.querySelectorAll("[data-i18n]").forEach((el) => { el.textContent = TEXTOS[el.dataset.i18n] ?? ""; }); }
function render(unidades) { const padres = new Map(unidades.map((u) => [u.clave, u.etiqueta])); const visibles = filtrarUnidades(unidades, document.querySelector("#filter-text").value, document.querySelector("#filter-type").value); document.querySelector("#result-count").textContent = TEXTOS.count.replace("{visible}", visibles.length).replace("{total}", unidades.length); document.querySelector("#rows").innerHTML = visibles.map((u) => `<tr><td>${text(traducirTipo(u.tipo))}</td><td>${text(u.codigo_fuente ?? u.clave)}</td><td>${text(u.etiqueta)}</td><td>${text(padres.get(u.adscripcion_clave) ?? "—")}</td><td>${text(u.pagina_fuente ?? "—")}</td></tr>`).join(""); document.querySelector("#table-wrap").hidden = false; document.querySelector("#state").hidden = visibles.length > 0; document.querySelector("#state").textContent = visibles.length ? "" : TEXTOS.empty; }
function main() { mostrarTextos(); document.querySelector("#source-link").href = FUENTE_RPT; const state = document.querySelector("#state"); const cargar = async () => { state.hidden = false; state.className = "org-state"; state.textContent = TEXTOS.loading; document.querySelector("#table-wrap").hidden = true; try { const data = await crearCliente().obtener(); document.querySelector("#catalog-id").textContent = data.catalogo_id; document.querySelector("#catalog-version").textContent = data.catalogo_version; document.querySelector("#catalog-hash").textContent = data.catalogo_huella_sha256; document.querySelector("#catalog-status").textContent = TEXTOS.draft; document.querySelector("#provenance").textContent = data.descripcion; document.querySelector("#source-ref").textContent = data.fuente_ref; render(data.unidades); document.querySelector("#filter-text").oninput = () => render(data.unidades); document.querySelector("#filter-type").onchange = () => render(data.unidades); } catch (error) { state.className = "org-state error"; state.innerHTML = `${text(TEXTOS.error)} <button type="button" id="retry">${text(TEXTOS.retry)}</button>`; document.querySelector("#retry").onclick = cargar; } }; cargar(); }
if (typeof document !== "undefined") main();
