import { crearClienteCircuitoRRHH } from "./cliente-http-circuito-rrhh.js?v=20261002-rrhh-circuito-v2";
import { cargarTextos } from "../../../comun/textos.js";

const mensajes = await cargarTextos("contratacion-temporal-circuito-rrhh");
const textosPortal = await cargarTextos("portal");
const general = mensajes.seccion("general");
const fases = mensajes.seccion("fases");
const actuaciones = mensajes.seccion("actuaciones");
const textosFases = textosPortal.seccion("fases_rrhh");
function escapar(valor) {
  return String(valor ?? "").replace(/[&<>"']/gu, (caracter) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  })[caracter]);
}
function texto(clave) { return general[clave] ?? general.no_disponible; }
function plantilla(clave, variables) {
  return Object.entries(variables).reduce((valor, [nombre, dato]) =>
    valor.replaceAll(`{${nombre}}`, String(dato)), texto(clave));
}

// El carril se actualiza con el estado acreditado del circuito, no con la fase
// administrativa: el análisis puede dejar esta última todavía en solicitud.
export function pasosRailCircuitoRRHH(datos, claves) {
  const indices = new Map(claves.map((clave, indice) => [clave, indice]));
  const actual = indices.get(`circuito_${datos.circuito.estado_actual}`);
  if (actual === undefined) return null;
  const hechos = new Set();
  for (const hito of datos.circuito.hitos) {
    const origen = indices.get(`circuito_${hito.origen}`);
    const destino = indices.get(`circuito_${hito.destino}`);
    if (destino !== undefined && origen !== undefined && destino < origen) {
      for (const indice of hechos) if (indice >= destino) hechos.delete(indice);
    } else if (origen !== undefined && origen !== destino) {
      hechos.add(origen);
    }
  }
  return claves.map((_, indice) => indice === actual ? "ahora" : hechos.has(indice) ? "hecho" : "falta");
}

/** Conserva la navegación de la ficha sin atribuirle un avance no consultado. */
export function marcarRailDesconocido(bloque) {
  const rail = bloque?.closest?.(".ct-expedientes")?.querySelector?.("[data-ct-exp-rail]");
  const elementos = [...(rail?.querySelectorAll?.(":scope > li") ?? [])];
  if (!elementos.length) return false;
  for (const elemento of elementos) {
    elemento.className = "desconocido";
    elemento.removeAttribute("aria-current");
    const boton = elemento.querySelector?.("[data-ct-exp-fase-ver]");
    if (!boton) continue;
    const nombre = boton.querySelector(".nombre")?.textContent ?? "";
    const estado = texto("avance_no_disponible");
    const rotulo = boton.querySelector("small");
    const marca = boton.querySelector(".marca");
    if (rotulo) rotulo.textContent = estado;
    if (marca) marca.textContent = elemento.dataset.ctExpOrden;
    boton.setAttribute("aria-label", plantilla("ver_fase_estado", { fase: nombre, estado }));
  }
  const panel = rail.closest("nav");
  const resumen = panel?.querySelector(".cabecera-panel .texto-secundario");
  if (resumen) resumen.textContent = texto("resumen_desconocido");
  if (panel) panel.hidden = false;
  return true;
}

export function actualizarRailCircuitoRRHH(bloque, datos) {
  const rail = bloque?.closest?.(".ct-expedientes")?.querySelector?.("[data-ct-exp-rail]");
  const elementos = [...(rail?.querySelectorAll?.(":scope > li") ?? [])];
  if (!elementos.length) return false;
  const claves = elementos.map((elemento) => elemento.querySelector?.("[data-ct-exp-fase-ver]")?.dataset.ctExpFaseVer ?? "");
  const pasos = pasosRailCircuitoRRHH(datos, claves);
  if (!pasos) return marcarRailDesconocido(bloque);
  for (const [indice, elemento] of elementos.entries()) {
    const paso = pasos[indice];
    elemento.className = paso;
    if (paso === "ahora") elemento.setAttribute("aria-current", "step");
    else elemento.removeAttribute("aria-current");
    const boton = elemento.querySelector("[data-ct-exp-fase-ver]");
    const estado = textosFases[`linea_${paso}`];
    if (!boton || !estado) continue;
    const nombre = boton.querySelector(".nombre")?.textContent ?? "";
    const textoEstado = boton.querySelector("small");
    const marca = boton.querySelector(".marca");
    if (textoEstado) textoEstado.textContent = estado;
    if (marca) marca.textContent = paso === "hecho" ? "✓" : elemento.dataset.ctExpOrden;
    boton.setAttribute("aria-label", plantilla("ver_fase_estado", { fase: nombre, estado }));
  }
  const resumen = rail.closest("nav")?.querySelector(".cabecera-panel .texto-secundario");
  if (resumen) resumen.textContent = plantilla("resumen_fases", {
    hechas: pasos.filter((paso) => paso === "hecho").length,
    ahora: pasos.filter((paso) => paso === "ahora").length,
    faltan: pasos.filter((paso) => paso === "falta").length,
  });
  const panel = rail.closest("nav");
  if (panel) panel.hidden = false;
  return true;
}

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
    <p>${escapar(texto("continuar"))}</p>
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
    marcarRailDesconocido(bloque);
    resultado.textContent = texto("consultando");
    try {
      const respuesta = await cliente.consultar({
        expediente_ref: bloque.dataset.ctCircuitoExpediente, version_observada: version,
      }, { signal: controlador.signal });
      if (!documento.contains(bloque) || controlador.signal.aborted) return;
      if (respuesta.estado === "disponible") {
        resultado.innerHTML = renderizarCircuitoRRHH(respuesta.datos);
        actualizarRailCircuitoRRHH(bloque, respuesta.datos);
      }
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
