import { crearClienteCircuitoRRHH } from "./cliente-http-circuito-rrhh.js?v=20261002-rrhh-circuito-v2";
import { cargarTextos } from "../../../comun/textos.js";

const mensajes = await cargarTextos("contratacion-temporal-circuito-rrhh");
const general = mensajes.seccion("general");
const fases = mensajes.seccion("fases");
const actuaciones = mensajes.seccion("actuaciones");
function escapar(valor) {
  return String(valor ?? "").replace(/[&<>"']/gu, (caracter) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  })[caracter]);
}
function texto(clave) { return general[clave] ?? general.no_disponible; }

/** Muestra solo hechos devueltos por la consulta autorizada. */
export function renderizarCircuitoRRHH(datos) {
  const fase = fases[datos.circuito.estado_actual] ?? texto("estado_desconocido");
  const hitos = datos.circuito.hitos;
  return `<div class="cuerpo-panel">
    <p class="estado-linea"><strong>${escapar(texto("fase_actual"))}</strong> ${escapar(fase)}</p>
    ${hitos.length ? `<ol class="lista-documentos">${hitos.map((hito) => `<li>
      <span class="simbolo" aria-hidden="true">•</span>
      <div><strong>${escapar(actuaciones[hito.tipo] ?? texto("actuacion_registrada"))}</strong>
        <time class="texto-secundario" datetime="${escapar(hito.registrado_en)}">${escapar(mensajes.fecha(hito.registrado_en, {
    dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid",
  }))}</time></div>
    </li>`).join("")}</ol>` : `<p>${escapar(texto("sin_hitos"))}</p>`}
  </div>`;
}

export function renderizarConsultaCircuitoRRHH(expediente) {
  return `<section class="panel" data-ct-circuito-rrhh
    data-ct-circuito-expediente="${escapar(expediente.expediente_ref)}"
    data-ct-circuito-version="${escapar(expediente.version)}">
    <div class="cabecera-panel"><h3>${escapar(texto("titulo"))}</h3>
      <button type="button" class="boton-secundario" data-ct-circuito-consultar>${escapar(texto("consultar"))}</button>
    </div><div data-ct-circuito-resultado role="status" aria-live="polite"></div>
  </section>`;
}

/** La consulta se pide al pulsar; no envía actos ni genera otras versiones. */
export function instalarConsultaCircuitoRRHH(documento = globalThis.document, cliente = crearClienteCircuitoRRHH()) {
  if (!documento?.addEventListener) return () => {};
  const pendientes = new Set();
  const manejar = async (evento) => {
    const boton = evento.target?.closest?.("[data-ct-circuito-consultar]");
    if (!boton || boton.getAttribute("aria-busy") === "true") return;
    const bloque = boton.closest("[data-ct-circuito-rrhh]");
    const resultado = bloque?.querySelector("[data-ct-circuito-resultado]");
    const version = Number(bloque?.dataset.ctCircuitoVersion);
    if (!resultado || !Number.isSafeInteger(version) || version < 1) return;
    evento.preventDefault();
    const controlador = new AbortController();
    pendientes.add(controlador);
    boton.setAttribute("aria-busy", "true");
    resultado.textContent = texto("consultando");
    try {
      const respuesta = await cliente.consultar({
        expediente_ref: bloque.dataset.ctCircuitoExpediente, version_observada: version,
      }, { signal: controlador.signal });
      if (!documento.contains(bloque) || controlador.signal.aborted) return;
      if (respuesta.estado === "disponible") resultado.innerHTML = renderizarCircuitoRRHH(respuesta.datos);
      else resultado.textContent = texto(respuesta.estado === "denegado" ? "denegado" : "no_disponible");
    } catch {
      if (documento.contains(bloque) && !controlador.signal.aborted) resultado.textContent = texto("no_disponible");
    } finally {
      pendientes.delete(controlador);
      boton.removeAttribute("aria-busy");
    }
  };
  documento.addEventListener("click", manejar);
  return () => {
    documento.removeEventListener("click", manejar);
    for (const pendiente of pendientes) pendiente.abort();
    pendientes.clear();
  };
}

instalarConsultaCircuitoRRHH();
