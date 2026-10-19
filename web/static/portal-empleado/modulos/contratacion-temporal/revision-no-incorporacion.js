import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js?v=VEC_WEB_VERSION";
import { validarReferenciaExpedienteSeguimiento } from "./contrato-seguimiento-incorporacion.js?v=VEC_WEB_VERSION";
import { crearTraductorSeguimientoCese } from "./i18n-seguimiento-cese.js?v=VEC_WEB_VERSION";
import { montarPanelSeguimientoCese } from "./seguimiento-cese.js?v=VEC_WEB_VERSION";
import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "../../../comun/idioma.js?v=VEC_WEB_VERSION";

/** La URL localiza el expediente; la versión y el permiso proceden de la consulta. */
export function montarRevisionNoIncorporacion({ documento, localizacion, ventana, cliente, mensajes = {} } = {}) {
  const panel = documento?.querySelector?.("[data-revision-panel]");
  const estado = documento?.querySelector?.("[data-revision-estado]");
  const numero = documento?.querySelector?.("[data-revision-numero]");
  if (!panel || !estado || !numero || typeof cliente?.consultarSeguimientoCese !== "function") throw new TypeError("revisión no disponible");
  const t = crearTraductorSeguimientoCese(mensajes);
  for (const nodo of documento.querySelectorAll("[data-revision-texto]")) nodo.textContent = t(nodo.dataset.revisionTexto);
  for (const nodo of documento.querySelectorAll("[data-revision-alt]")) nodo.alt = t(nodo.dataset.revisionAlt);
  if (documento.documentElement) documento.documentElement.lang = IDIOMA_ACTUAL;
  documento.title = t("revision_titulo");
  let expedienteRef;
  try {
    const parametros = new URL(localizacion.href).searchParams;
    if (parametros.getAll("expediente_ref").length !== 1) throw new TypeError("referencia ausente");
    expedienteRef = validarReferenciaExpedienteSeguimiento(parametros.get("expediente_ref"));
  } catch {
    estado.textContent = t("revision_enlace_invalido");
    return () => {};
  }
  estado.textContent = "";
  const desmontar = montarPanelSeguimientoCese({ contenedor: panel, cliente, contexto: { expediente_ref: expedienteRef },
    mensajes, locale: LOCALIZACION_ACTUAL, soloNoIncorporacion: true,
    confirmarOperacion: ({ titulo, advertencia }) => ventana.confirm(`${titulo}. ${advertencia}`),
    alConsultar: ({ estado: actual }) => { numero.textContent = t("revision_expediente", { numero: actual.numero_expediente }); },
  });
  ventana.addEventListener("pagehide", desmontar, { once: true });
  return () => { ventana.removeEventListener("pagehide", desmontar); desmontar(); };
}

if (globalThis.document) montarRevisionNoIncorporacion({ documento: globalThis.document, localizacion: globalThis.location,
  ventana: globalThis, cliente: crearClienteHTTPContratacionTemporal().seguimientoCese });
