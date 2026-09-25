/**
 * Petición RRHH p.4: sección «Cambios de datos» del detalle del expediente.
 * Se consulta solo a petición de la persona usuaria, porque cada consulta
 * consume una autorización y deja su auditoría. Los textos libres y los datos
 * personales llegan como la marca «*protegido»: solo se sabe que cambiaron.
 */
import { crearClienteHTTPCambiosExpediente } from "./cliente-http-cambios-expediente.js";
import { MENSAJES_CAMBIOS_EXPEDIENTE_ES } from "./i18n-cambios-expediente.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

const PROTEGIDO = "*protegido";
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

export function crearTraductorCambiosExpediente(mensajes = {}) {
  return crearTraductorExpedientesContratacion({ ...MENSAJES_CAMBIOS_EXPEDIENTE_ES, ...mensajes });
}

const formatoFecha = new Intl.DateTimeFormat("es-ES", { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" });

function fecha(valor) {
  const instante = new Date(valor);
  return Number.isNaN(instante.getTime()) ? String(valor) : formatoFecha.format(instante);
}

export function etiquetaRutaCambio(ruta, t) {
  return ruta.split(".").map((segmento) => {
    const [nombre, ...indices] = segmento.split("[");
    const clave = `cambios_segmento_${nombre}`;
    let texto;
    try { texto = t(clave); } catch { texto = nombre.replaceAll("_", " "); }
    return indices.length ? `${texto} ${indices.map((i) => Number.parseInt(i, 10) + 1).join(" ")}` : texto;
  }).join(" · ");
}

export function valorVisibleCambio(valor, t) {
  if (valor === null) return t("cambios_sin_valor");
  if (valor === PROTEGIDO) return t("cambios_texto_protegido");
  if (INSTANTE.test(valor)) return fecha(valor);
  return valor;
}

/** Marcado estático del apartado; el delegado del documento lo activa. */
export function renderizarCambiosExpediente(expediente, mensajes = {}) {
  if (expediente?.demostracion !== false || typeof expediente.expediente_ref !== "string"
    || !Number.isSafeInteger(expediente.version)) return "";
  const t = crearTraductorCambiosExpediente(mensajes);
  return `<details class="ct-exp-detalle-tecnico ct-exp-cambios" data-ct-cambios data-expediente-ref="${escapar(expediente.expediente_ref)}" data-version="${expediente.version}">
      <summary>${escapar(t("cambios_titulo"))}</summary>
      <button type="button" class="boton-secundario" data-ct-cambios-consultar>${escapar(t("cambios_consultar"))}</button>
      <div data-ct-cambios-resultado aria-live="polite"></div>
    </details>`;
}

export function renderizarTablaCambios(resultado, t) {
  if (!resultado.cambios.length) return `<p class="vacio-controlado" role="status">${escapar(t("cambios_vacio"))}</p>`;
  const cabecera = ["cambios_col_version", "cambios_col_fecha", "cambios_col_campo", "cambios_col_anterior", "cambios_col_nuevo"]
    .map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("");
  const filas = resultado.cambios.map((c) => `<tr><td>${c.version_expediente}</td><td><time datetime="${escapar(c.registrada_en)}">${escapar(fecha(c.registrada_en))}</time></td><th scope="row">${escapar(etiquetaRutaCambio(c.ruta, t))}</th><td>${escapar(valorVisibleCambio(c.valor_anterior, t))}</td><td>${escapar(valorVisibleCambio(c.valor_nuevo, t))}</td></tr>`).join("");
  const limite = resultado.recortado ? `<p role="note">${escapar(t("cambios_limite", { total: resultado.cambios.length }))}</p>` : "";
  return `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${escapar(t("cambios_titulo"))}"><table class="tabla-datos ct-exp-tabla-panel"><caption>${escapar(t("cambios_leyenda"))}</caption><thead><tr>${cabecera}</tr></thead><tbody>${filas}</tbody></table></div>${limite}`;
}

const consultasEnCurso = new WeakMap();

/** Consulta los cambios del apartado y pinta la tabla en su destino. */
export async function consultarCambiosDesde(boton, { cliente = crearClienteHTTPCambiosExpediente(), mensajes = {} } = {}) {
  const apartado = boton?.closest?.("[data-ct-cambios]");
  const destino = apartado?.querySelector?.("[data-ct-cambios-resultado]");
  if (!apartado || !destino) return false;
  const t = crearTraductorCambiosExpediente(mensajes);
  consultasEnCurso.get(apartado)?.abort();
  const controlador = new AbortController();
  consultasEnCurso.set(apartado, controlador);
  boton.disabled = true;
  destino.setAttribute("aria-busy", "true");
  destino.innerHTML = `<p role="status">${escapar(t("cambios_cargando"))}</p>`;
  try {
    const resultado = await cliente.consultarCambios({
      expediente_ref: apartado.dataset.expedienteRef,
      version_observada: Number.parseInt(apartado.dataset.version, 10),
    }, { signal: controlador.signal });
    if (!controlador.signal.aborted) destino.innerHTML = renderizarTablaCambios(resultado, t);
  } catch {
    if (!controlador.signal.aborted) destino.innerHTML = `<p class="mensaje-error" role="alert">${escapar(t("cambios_error"))}</p>`;
  } finally {
    if (!controlador.signal.aborted) {
      boton.disabled = false;
      destino.removeAttribute("aria-busy");
      consultasEnCurso.delete(apartado);
    }
  }
  return true;
}

// Delegado en el documento, como los atajos de incidencia, para no crecer la
// vista principal. Solo lee; no ejecuta ninguna actuación administrativa.
let instalado = false;
export function instalarCambiosExpediente(documento = globalThis.document, opciones = {}) {
  if (instalado || !documento?.addEventListener) return;
  instalado = true;
  documento.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.("[data-ct-cambios-consultar]");
    if (!boton) return;
    evento.preventDefault();
    void consultarCambiosDesde(boton, opciones);
  });
}

instalarCambiosExpediente();
