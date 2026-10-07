import { cargarTextos } from "/comun/textos.js";
import { leerRecursoJSON } from "/comun/idioma.js";
import { crearClienteBaremo } from "../baremo-cliente.js?v=20261001-concursos-v6";
import { crearEditorBaremo, leerReglas, aMicropuntos, MAXIMO_ARCHIVO, comprobarCatalogoJornada, normalizarMinimoFormacion, comprobarCatalogoRestos, errorRestosRegla } from "../baremo-editor.js?v=20261002-a-restos-recuperacion-v1";
import { renderizarPanelesBaremo } from "../baremo-vista.js?v=20261002-a-restos-recuperacion-v1";
import { crearClienteConcursos } from "../concursos-cliente.js?v=20261001-g-concursos-topes-v2";
import { renderizarConcursos } from "../concursos-vista.js?v=20261002-a-restos-recuperacion-v1";
import { montarConcursos } from "../concursos-montaje.js?v=20261002-a-restos-recuperacion-v1";
const [textos, textosConcursos, datosJornada, datosRestos] = await Promise.all([cargarTextos("baremo-bolsa"), cargarTextos("baremo-concursos"),
  leerRecursoJSON(new URL("/catalogos/baremo-jornada-v1.json", import.meta.url)).catch(() => null),
  leerRecursoJSON(new URL("/catalogos/baremo-restos-v1.json", import.meta.url)).catch(() => null)]);
let catalogoJornada = null;
try {
  const candidato = comprobarCatalogoJornada(datosJornada);
  for (const opcion of candidato.opciones) {
    textos.traducir(`editor.${opcion.etiqueta}`);
    if (!opcion.disponible) textos.traducir(`editor.${opcion.motivo}`);
  }
  catalogoJornada = candidato;
} catch { /* Conserva las demás reglas; no habilita edición de jornada. */ }
let catalogoRestos = null;
try {
  const candidato = comprobarCatalogoRestos(datosRestos);
  for (const opcion of candidato.opciones) {
    textos.traducir(`editor.${opcion.etiqueta}`);
    textos.traducir(`editor.${opcion.explicacion}`);
  }
  catalogoRestos = candidato;
} catch { /* Sin catálogo positivo se conserva el consumidor y se cierra la nueva edición. */ }
const t = (clave) => textos.traducir(`editor.${clave}`);
document.documentElement.lang = textos.idioma;
document.title = t("titulo");
const raiz = document.getElementById("baremo-contenido");
const cliente = crearClienteBaremo();
let ejemplos = [], ayuda = false, error = "", filtro = "", panel = "bolsa";
let carga = null, lecturaId = 0, focoComparacion = false;
function pintar() {
  const activo = document.activeElement;
  const identidadFoco = raiz.contains(activo) ? { ruta: activo.dataset.ruta, accion: activo.dataset.accion, name: activo.name, panel: activo.dataset.panel, concursoRuta: activo.dataset.concursoRuta, concursoAccion: activo.dataset.concursoAccion, submit: activo.type === "submit" } : null;
  document.title = t(panel === "concursos" ? "concursos" : "titulo");
  const concursosHTML = panel === "concursos" ? renderizarConcursos(concursos.estado(), concursos.opciones()) : "";
  raiz.innerHTML = renderizarPanelesBaremo({ ...editor.estado(), ayuda, error: error || editor.estado().error }, { textos, ejemplos, filtro, panel, concursosHTML });
  if (panel === "concursos") concursos.alPintar();
  filtrar();
  for (const control of raiz.querySelectorAll('#baremo-panel-bolsa [aria-invalid="true"]')) control.setCustomValidity(t(claveErrorCampo(control)));
  if (panel === "bolsa" && (Object.values(editor.estado().invalidos).some((valor) => valor !== "") || error === "validacion")) mostrarErrores();
  const trabajandoActivo = panel === "concursos" ? concursos.estado().trabajando : editor.estado().trabajando;
  let siguiente = !trabajandoActivo && focoComparacion && document.activeElement === document.body ? raiz.querySelector(`#baremo-panel-${panel} [type="submit"]`) : null;
  if (!trabajandoActivo) focoComparacion = false;
  if (identidadFoco) {
    siguiente = [...raiz.querySelectorAll(`#baremo-panel-${panel} input, #baremo-panel-${panel} select, #baremo-panel-${panel} button, #baremo-panel-${panel} summary, .baremo-navegacion button`)].find((c) => identidadFoco.panel ? c.dataset.panel === identidadFoco.panel : identidadFoco.concursoRuta ? c.dataset.concursoRuta === identidadFoco.concursoRuta : identidadFoco.concursoAccion ? c.dataset.concursoAccion === identidadFoco.concursoAccion : identidadFoco.ruta ? c.dataset.ruta === identidadFoco.ruta : identidadFoco.accion ? c.dataset.accion === identidadFoco.accion : identidadFoco.submit ? c.type === "submit" : c.name && c.name === identidadFoco.name);
  }
  if (siguiente) {
    const caja = siguiente.getBoundingClientRect();
    const x = (caja.left + caja.right) / 2, y = (caja.top + caja.bottom) / 2;
    const dx = Math.min(1, (caja.right - caja.left) / 4), dy = Math.min(1, (caja.bottom - caja.top) / 4);
    const puntos = [[x, y], [caja.left + dx, y], [caja.right - dx, y], [x, caja.top + dy], [x, caja.bottom - dy]];
    const visible = caja.top >= 0 && caja.left >= 0 && caja.bottom <= window.innerHeight && caja.right <= window.innerWidth && puntos.every(([px, py]) => {
      const encima = document.elementFromPoint(px, py);
      return encima === siguiente || siguiente.contains(encima);
    });
    siguiente.focus({ preventScroll: visible });
  }
}
const editor = crearEditorBaremo({ cliente, catalogoJornada, catalogoRestos, alCambiar: pintar });
const concursos = montarConcursos({ raiz, cliente: crearClienteConcursos(), textos: textosConcursos,
  alCambiar: pintar, alComparar: () => { focoComparacion = true; }, activo: () => panel === "concursos" });
function filtrar() {
  if (panel !== "bolsa") return;
  for (const fila of raiz.querySelectorAll("#baremo-panel-bolsa .baremo-reglas tbody tr:not([data-sin-reglas])")) fila.hidden = !fila.textContent.toLocaleLowerCase(textos.localizacion).includes(filtro.toLocaleLowerCase(textos.localizacion));
  const sinReglas = raiz.querySelector("[data-sin-reglas]");
  if (sinReglas) sinReglas.hidden = [...raiz.querySelectorAll("#baremo-panel-bolsa .baremo-reglas tbody tr:not([data-sin-reglas])")].some((f) => !f.hidden);
}
function falloRestosBorrador() {
  const estado = editor.estado();
  if (!estado.catalogoRestos) return "";
  return (estado.borrador?.reglas_experiencia ?? []).map((regla, indice) => {
    const ruta = JSON.stringify(["reglas_experiencia", indice, "restos", "modo"]);
    return errorRestosRegla(regla, estado.catalogoRestos, estado.invalidos[ruta] ?? regla.restos?.modo);
  }).find(Boolean) ?? "";
}
function exportacionBloqueada() { return Object.keys(editor.estado().invalidos).length > 0 || Boolean(falloRestosBorrador()); }
function claveErrorCampo(control) {
  if (control.hasAttribute("data-restos-modo")) return control.dataset.restosError || "restos_no_disponibles";
  return control.hasAttribute("data-minimo-formacion") ? "minimo_formacion_invalido"
    : control.hasAttribute("data-umbral") ? "umbral_invalido"
    : control.hasAttribute("data-puntos") ? "puntos_invalidos" : "fecha_invalida";
}
function mostrarErrores({ enfocar = false } = {}) {
  for (const control of raiz.querySelectorAll("input[data-minimo-formacion]")) {
    try { normalizarMinimoFormacion(control.value); control.setCustomValidity(""); }
    catch { control.setCustomValidity(t("minimo_formacion_invalido")); }
  }
  const estado = editor.estado();
  for (const control of raiz.querySelectorAll("select[data-restos-modo]:not([disabled])")) {
    const regla = estado.borrador?.reglas_experiencia?.[Number(control.dataset.indice)];
    const fallo = errorRestosRegla(regla, estado.catalogoRestos, control.value);
    control.dataset.restosError = fallo; control.setCustomValidity(fallo ? t(fallo) : "");
  }
  const controles = [...raiz.querySelectorAll("input[data-ruta], select[data-restos-modo]:not([disabled])")];
  const invalidos = controles.filter((c) => !c.checkValidity());
  for (const control of controles) {
    const invalido = invalidos.includes(control), pista = control.closest("label")?.querySelector("[data-error-campo]");
    control.setAttribute("aria-invalid", String(invalido));
    if (pista) { pista.hidden = !invalido; pista.textContent = invalido ? t(claveErrorCampo(control)) : ""; }
  }
  const resumen = raiz.querySelector("#baremo-error");
  if (invalidos.length && resumen) {
    resumen.replaceChildren(); resumen.hidden = false; resumen.dataset.validacion = "true";
    const titulo = document.createElement("p"); titulo.textContent = t("validacion"); resumen.append(titulo);
    const lista = document.createElement("ul");
    for (const control of invalidos) {
      const li = document.createElement("li"), enlace = document.createElement("a");
      enlace.href = `#${control.id}`; enlace.textContent = control.getAttribute("aria-label");
      enlace.addEventListener("click", (e) => { e.preventDefault(); control.focus(); });
      li.append(enlace); lista.append(li);
    }
    resumen.append(lista); if (enfocar) invalidos[0].focus();
  } else if (resumen?.dataset.validacion === "true") { resumen.hidden = true; delete resumen.dataset.validacion; }
  return invalidos.length === 0;
}
raiz.addEventListener("focusout", (evento) => { if (panel === "bolsa" && raiz.contains(evento.target) && evento.target.dataset?.ruta) mostrarErrores(); });
function descartar() { return !editor.estado().cambiado || window.confirm(t("perder_cambios")); }
async function cargar() {
  carga?.abort(); carga = new AbortController(); error = ""; pintar();
  try { ejemplos = await cliente.ejemplos({ signal: carga.signal }); editor.cargar(ejemplos[0]); }
  catch (fallo) { if (fallo.name !== "AbortError") { error = fallo.codigo ?? "simulacion_fallida"; pintar(); } }
}
raiz.addEventListener("input", (evento) => {
  if (panel !== "bolsa") return;
  const control = evento.target;
  if (control.name === "filtro") { filtro = control.value; filtrar(); return; }
  if (!control.dataset.ruta || control.hasAttribute("data-jornada-modo") || control.hasAttribute("data-restos-modo")) return;
  lecturaId++;
  try {
    const valor = control.hasAttribute("data-puntos") ? aMicropuntos(control.value) : control.value;
    if (control.hasAttribute("data-umbral")) editor.editarUmbralJornada(Number(control.dataset.indice), valor);
    else if (control.hasAttribute("data-minimo-formacion")) editor.editarMinimoFormacion(Number(control.dataset.indice), valor);
    else editor.editar(JSON.parse(control.dataset.ruta), valor);
    error = ""; control.removeAttribute("aria-invalid"); control.setCustomValidity(""); raiz.querySelector('[data-accion="exportar"]').disabled = exportacionBloqueada();
    // No sustituir el control durante escritura: conserva foco y selección.
    const resultado = raiz.querySelector(".baremo-resultados"); if (resultado) { resultado.textContent = t("pendiente"); resultado.setAttribute("aria-busy", "false"); }
    const boton = raiz.querySelector('[type="submit"]'); boton.disabled = false; boton.textContent = t("comparar");
    raiz.querySelector(".baremo-cabecera [role='status']").textContent = t("sin_guardar");
    if (raiz.querySelector("#baremo-error")?.dataset.validacion) mostrarErrores();
  } catch { editor.registrarInvalido(JSON.parse(control.dataset.ruta), control.value); raiz.querySelector(".baremo-cabecera [role='status']").textContent = t("sin_guardar"); raiz.querySelector('[data-accion="exportar"]').disabled = true; const resultado = raiz.querySelector(".baremo-resultados"); if (resultado) { resultado.textContent = t("pendiente"); resultado.setAttribute("aria-busy", "false"); } const boton = raiz.querySelector('[type="submit"]'); boton.disabled = false; boton.textContent = t("comparar"); control.setAttribute("aria-invalid", "true"); control.setCustomValidity(t(claveErrorCampo(control))); }
});
raiz.addEventListener("change", async (evento) => {
  if (panel !== "bolsa") return;
  const control = evento.target;
  if (control.hasAttribute("data-restos-modo")) {
    lecturaId++; focoComparacion = false;
    try { error = ""; editor.editarRestos(Number(control.dataset.indice), control.value); }
    catch { error = falloRestosBorrador() || "restos_no_disponibles"; pintar(); }
    return;
  }
  if (control.hasAttribute("data-jornada-modo")) {
    lecturaId++;
    try { error = ""; editor.editarPoliticaJornada(Number(control.dataset.indice), control.value); }
    catch { error = "jornada_no_disponible"; pintar(); }
    return;
  }
  if (control.name === "ejemplo") {
    if (descartar()) { lecturaId++; error = ""; editor.cargar(ejemplos.find((e) => e.referencia === control.value)); }
    else control.value = editor.estado().ejemplo.referencia;
  }
  if (control.name === "archivo" && control.files[0]) {
    const turno = ++lecturaId, contexto = editor.estado().ejemplo.referencia;
    try {
      if (control.files[0].size > MAXIMO_ARCHIVO) throw new Error();
      const reglas = leerReglas(await control.files[0].text());
      if (turno !== lecturaId || contexto !== editor.estado().ejemplo?.referencia) return;
      const actual = editor.estado().ejemplo;
      if (reglas.esquema !== actual.reglas.esquema) throw new Error();
      if (descartar()) { error = ""; editor.cargar(actual, reglas); }
    } catch { if (turno === lecturaId) { error = "archivo_invalido"; pintar(); } }
  }
});
raiz.addEventListener("submit", (evento) => {
  evento.preventDefault();
  if (panel !== "bolsa") return;
  if (!mostrarErrores({ enfocar: true })) { error = "validacion"; return; }
  error = ""; focoComparacion = true; void editor.comparar();
});
raiz.addEventListener("click", (evento) => {
  const destino = evento.target.closest("[data-panel]")?.dataset.panel;
  if (["bolsa", "concursos"].includes(destino)) {
    if (destino !== panel) {
      editor.cancelarSimulacion(); concursos.cancelar(); lecturaId++; focoComparacion = false; panel = destino;
      if (panel === "concursos" && !concursos.estado().borrador) void concursos.cargar();
      pintar(); raiz.querySelector(`[data-panel="${panel}"]`).focus({ preventScroll: true });
    }
    return;
  }
  if (panel !== "bolsa") return;
  const accion = evento.target.closest("[data-accion]")?.dataset.accion;
  if (accion === "ayuda") { ayuda = !ayuda; pintar(); raiz.querySelector('[data-accion="ayuda"]').focus(); }
  if (accion === "recargar") void cargar();
  if (accion === "restablecer" && descartar()) { lecturaId++; error = ""; editor.cargar(editor.estado().ejemplo); }
  if (accion === "exportar") {
    if (!mostrarErrores({ enfocar: true }) || exportacionBloqueada()) return;
    try {
      const url = URL.createObjectURL(new Blob([editor.exportar()], { type: "application/json" }));
      const enlace = document.createElement("a"); enlace.href = url; enlace.download = t("archivo_borrador"); enlace.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch { error = falloRestosBorrador() || "validacion"; pintar(); mostrarErrores({ enfocar: true }); }
  }
});
window.addEventListener("beforeunload", (evento) => { if (editor.estado().cambiado || concursos.estado().cambiado) { evento.preventDefault(); evento.returnValue = ""; } });
window.addEventListener("pagehide", () => { carga?.abort(); editor.desmontar(); concursos.desmontar(); });
void cargar();
