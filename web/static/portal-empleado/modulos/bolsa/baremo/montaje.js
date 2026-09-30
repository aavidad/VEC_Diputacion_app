import { cargarTextos } from "/comun/textos.js";
import { crearClienteBaremo } from "../baremo-cliente.js?v=20261001-baremo-editor-v1";
import { crearEditorBaremo, leerReglas, aMicropuntos, MAXIMO_ARCHIVO } from "../baremo-editor.js?v=20261001-baremo-editor-v1";
import { renderizarBaremo } from "../baremo-vista.js?v=20261001-baremo-editor-v1";
const textos = await cargarTextos("baremo-bolsa");
const t = (clave) => textos.traducir(`editor.${clave}`);
document.documentElement.lang = textos.idioma;
document.title = t("titulo");
const raiz = document.getElementById("baremo-contenido");
const cliente = crearClienteBaremo();
let ejemplos = [], ayuda = false, error = "", filtro = "";
let carga = null, lecturaId = 0, focoComparacion = false;
function pintar() {
  const activo = document.activeElement;
  const identidadFoco = raiz.contains(activo) ? { ruta: activo.dataset.ruta, accion: activo.dataset.accion, name: activo.name, submit: activo.type === "submit" } : null;
  raiz.innerHTML = renderizarBaremo({ ...editor.estado(), ayuda, error: error || editor.estado().error }, { textos, ejemplos, filtro });
  filtrar();
  for (const control of raiz.querySelectorAll('[aria-invalid="true"]')) control.setCustomValidity(t("puntos_invalidos"));
  if (Object.keys(editor.estado().invalidos).length || error === "validacion") mostrarErrores();
  if (!editor.estado().trabajando && focoComparacion && document.activeElement === document.body) raiz.querySelector('[type="submit"]')?.focus({ preventScroll: true });
  if (!editor.estado().trabajando) focoComparacion = false;
  if (identidadFoco) {
    const siguiente = [...raiz.querySelectorAll("input,select,button")].find((c) => identidadFoco.ruta ? c.dataset.ruta === identidadFoco.ruta : identidadFoco.accion ? c.dataset.accion === identidadFoco.accion : identidadFoco.submit ? c.type === "submit" : c.name && c.name === identidadFoco.name);
    siguiente?.focus({ preventScroll: true });
  }
}
const editor = crearEditorBaremo({ cliente, alCambiar: pintar });
function filtrar() {
  for (const fila of raiz.querySelectorAll(".baremo-reglas tbody tr:not([data-sin-reglas])")) fila.hidden = !fila.textContent.toLocaleLowerCase(textos.localizacion).includes(filtro.toLocaleLowerCase(textos.localizacion));
  const sinReglas = raiz.querySelector("[data-sin-reglas]");
  if (sinReglas) sinReglas.hidden = [...raiz.querySelectorAll(".baremo-reglas tbody tr:not([data-sin-reglas])")].some((f) => !f.hidden);
}
function mostrarErrores({ enfocar = false } = {}) {
  const invalidos = [...raiz.querySelectorAll("input[data-ruta]")].filter((c) => !c.checkValidity());
  for (const control of raiz.querySelectorAll("input[data-ruta]")) {
    const invalido = invalidos.includes(control), pista = control.closest("label")?.querySelector("[data-error-campo]");
    control.setAttribute("aria-invalid", String(invalido));
    if (pista) { pista.hidden = !invalido; pista.textContent = invalido ? t(control.hasAttribute("data-puntos") ? "puntos_invalidos" : "fecha_invalida") : ""; }
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
raiz.addEventListener("focusout", (evento) => { if (evento.target.dataset?.ruta) mostrarErrores(); });
function descartar() { return !editor.estado().cambiado || window.confirm(t("perder_cambios")); }
async function cargar() {
  carga?.abort(); carga = new AbortController(); error = ""; pintar();
  try { ejemplos = await cliente.ejemplos({ signal: carga.signal }); editor.cargar(ejemplos[0]); }
  catch (fallo) { if (fallo.name !== "AbortError") { error = fallo.codigo ?? "simulacion_fallida"; pintar(); } }
}
raiz.addEventListener("input", (evento) => {
  const control = evento.target;
  if (control.name === "filtro") { filtro = control.value; filtrar(); return; }
  if (!control.dataset.ruta) return;
  lecturaId++;
  try {
    const valor = control.hasAttribute("data-puntos") ? aMicropuntos(control.value) : control.value;
    editor.editar(JSON.parse(control.dataset.ruta), valor); error = ""; control.removeAttribute("aria-invalid"); control.setCustomValidity(""); raiz.querySelector('[data-accion="exportar"]').disabled = Object.keys(editor.estado().invalidos).length > 0;
    // No sustituir el control durante escritura: conserva foco y selección.
    const resultado = raiz.querySelector(".baremo-resultados"); if (resultado) { resultado.textContent = t("pendiente"); resultado.setAttribute("aria-busy", "false"); }
    const boton = raiz.querySelector('[type="submit"]'); boton.disabled = false; boton.textContent = t("comparar");
    raiz.querySelector(".baremo-cabecera [role='status']").textContent = t("sin_guardar");
    if (raiz.querySelector("#baremo-error")?.dataset.validacion) mostrarErrores();
  } catch { editor.registrarInvalido(JSON.parse(control.dataset.ruta), control.value); raiz.querySelector(".baremo-cabecera [role='status']").textContent = t("sin_guardar"); raiz.querySelector('[data-accion="exportar"]').disabled = true; const resultado = raiz.querySelector(".baremo-resultados"); if (resultado) { resultado.textContent = t("pendiente"); resultado.setAttribute("aria-busy", "false"); } const boton = raiz.querySelector('[type="submit"]'); boton.disabled = false; boton.textContent = t("comparar"); control.setAttribute("aria-invalid", "true"); control.setCustomValidity(t("puntos_invalidos")); }
});
raiz.addEventListener("change", async (evento) => {
  const control = evento.target;
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
  if (!mostrarErrores({ enfocar: true })) { error = "validacion"; return; }
  error = ""; focoComparacion = true; void editor.comparar();
});
raiz.addEventListener("click", (evento) => {
  const accion = evento.target.closest("[data-accion]")?.dataset.accion;
  if (accion === "ayuda") { ayuda = !ayuda; pintar(); raiz.querySelector('[data-accion="ayuda"]').focus(); }
  if (accion === "recargar") void cargar();
  if (accion === "restablecer" && descartar()) { lecturaId++; error = ""; editor.cargar(editor.estado().ejemplo); }
  if (accion === "exportar") {
    if (!mostrarErrores({ enfocar: true }) || Object.keys(editor.estado().invalidos).length) return;
    const url = URL.createObjectURL(new Blob([editor.exportar()], { type: "application/json" }));
    const enlace = document.createElement("a"); enlace.href = url; enlace.download = t("archivo_borrador"); enlace.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
});
window.addEventListener("beforeunload", (evento) => { if (editor.estado().cambiado) { evento.preventDefault(); evento.returnValue = ""; } });
window.addEventListener("pagehide", () => { carga?.abort(); editor.desmontar(); });
void cargar();
