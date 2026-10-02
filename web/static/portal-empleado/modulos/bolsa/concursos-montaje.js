import { crearEditorConcursos, MAXIMO_CONFIGURACION } from "./concursos-editor.js?v=20261001-concursos-v6";
import { aMicropuntos } from "./baremo-editor.js?v=20261002-a-restos-recuperacion-v1";

export function montarConcursos({ raiz, cliente, textos, alCambiar, alComparar = () => {}, activo }) {
  const t = (clave) => textos.traducir(`concursos.${clave}`);
  let ejemplos = [], ayuda = false, filtro = "", carga = null, lectura = 0;
  const editor = crearEditorConcursos({ cliente, alCambiar });
  function pintar() { if (activo()) alCambiar(); }
  function filtrar() {
    const filas = [...raiz.querySelectorAll("[data-concurso-regla]")];
    for (const fila of filas) fila.hidden = !fila.textContent.toLocaleLowerCase(textos.localizacion).includes(filtro.toLocaleLowerCase(textos.localizacion));
    const vacio = raiz.querySelector("[data-concurso-sin-reglas]");
    if (vacio) vacio.hidden = filas.some((f) => !f.hidden);
  }
  function enfocarCampo(c) {
    for (let p = c.parentElement; p && p !== raiz; p = p.parentElement) {
      if (p.tagName === "DETAILS") p.open = true;
      if (p.hasAttribute("data-concurso-regla") && p.hidden) {
        filtro = ""; const buscador = raiz.querySelector('[name="concurso-filtro"]');
        if (buscador) buscador.value = ""; filtrar();
      }
    }
    c.focus();
  }
  function validar({ enfocar = false } = {}) {
    const campos = [...raiz.querySelectorAll("[data-concurso-ruta]")];
    const invalidos = campos.filter((c) => !c.checkValidity());
    for (const c of campos) {
      const invalido = invalidos.includes(c), pista = c.closest("label").querySelector("[data-error-concurso]");
      c.setAttribute("aria-invalid", String(invalido)); pista.hidden = !invalido;
      if (invalido) pista.textContent = t(c.dataset.concursoTipo === "puntos" ? "puntos_invalidos" : c.dataset.concursoTipo === "fecha" ? "fecha_invalida" : "nivel_invalido");
    }
    const resumen = raiz.querySelector("#concursos-error");
    if (invalidos.length) {
      resumen.replaceChildren(); resumen.hidden = false; resumen.dataset.validacion = "true";
      const p = document.createElement("p"); p.textContent = t("validacion"); resumen.append(p);
      const lista = document.createElement("ul");
      for (const c of invalidos) {
        const li = document.createElement("li"), a = document.createElement("a");
        a.href = `#${c.id}`; a.textContent = c.getAttribute("aria-label");
        a.addEventListener("click", (e) => { e.preventDefault(); enfocarCampo(c); }); li.append(a); lista.append(li);
      }
      resumen.append(lista); if (enfocar) enfocarCampo(invalidos[0]);
    } else if (resumen?.dataset.validacion) { resumen.hidden = true; delete resumen.dataset.validacion; }
    return invalidos.length === 0;
  }
  async function cargar() {
    carga?.abort(); carga = new AbortController(); const turno = ++lectura; errorCarga = ""; pintar();
    try {
      const nuevos = await cliente.ejemplos({ signal: carga.signal });
      if (turno !== lectura) return;
      ejemplos = nuevos; editor.cargar(ejemplos[0]);
    } catch (error) {
      if (error.name !== "AbortError" && turno === lectura) { errorCarga = error.codigo ?? "simulacion_fallida"; pintar(); }
    }
  }
  let errorCarga = "", errorArchivo = "";
  const descartar = () => !editor.estado().cambiado || globalThis.confirm(t("perder_cambios"));
  raiz.addEventListener("input", (e) => {
    if (!activo()) return;
    const c = e.target;
    if (c.name === "concurso-filtro") { filtro = c.value; filtrar(); return; }
    if (!c.dataset.concursoRuta) return;
    lectura++; errorArchivo = ""; c.setCustomValidity("");
    try {
      let valor = c.value;
      if (c.dataset.concursoTipo === "puntos") valor = aMicropuntos(c.value);
      if (c.dataset.concursoTipo === "nivel") {
        if (!/^-?(0|[1-9][0-9]{0,2})$/u.test(c.value) || !c.checkValidity()) throw new Error();
        valor = Number(c.value);
      }
      if (c.dataset.concursoTipo === "fecha" && (!c.value || !c.checkValidity())) throw new Error();
      editor.editar(JSON.parse(c.dataset.concursoRuta), valor);
    } catch {
      editor.editar(JSON.parse(c.dataset.concursoRuta), c.value, { invalido: true });
      c.setCustomValidity(t(c.dataset.concursoTipo === "puntos" ? "puntos_invalidos" : c.dataset.concursoTipo === "fecha" ? "fecha_invalida" : "nivel_invalido"));
    }
    const r = raiz.querySelector(".concursos-resultados"); r.textContent = t("pendiente"); r.setAttribute("aria-busy", "false");
    raiz.querySelector("[data-concurso-estado]").textContent = t("sin_guardar");
    const boton = raiz.querySelector("#concursos-formulario [type='submit']"); boton.disabled = false; boton.textContent = t("comparar");
    raiz.querySelector('[data-concurso-accion="exportar"]').disabled = Object.keys(editor.estado().invalidos).length > 0;
    if (raiz.querySelector("#concursos-error").dataset.validacion) validar();
  });
  raiz.addEventListener("focusout", (e) => { if (activo() && e.target.dataset?.concursoRuta) validar(); });
  raiz.addEventListener("change", async (e) => {
    if (!activo()) return;
    const c = e.target;
    if (c.name === "concurso-ejemplo") {
      if (descartar()) { lectura++; errorArchivo = ""; editor.cargar(ejemplos.find((v) => v.referencia === c.value)); }
      else c.value = editor.estado().ejemplo.referencia;
    }
    if (c.name === "archivo-concurso" && c.files[0]) {
      const turno = ++lectura, contexto = editor.estado().ejemplo?.referencia;
      try {
        const archivo = c.files[0];
        if (archivo.size > MAXIMO_CONFIGURACION) throw new Error();
        const texto = await archivo.text();
        if (turno !== lectura || contexto !== editor.estado().ejemplo?.referencia || !activo()) return;
        if (descartar()) { errorArchivo = ""; await editor.importar(texto); }
      } catch { if (turno === lectura) { errorArchivo = "archivo_invalido"; pintar(); } }
    }
  });
  raiz.addEventListener("submit", (e) => {
    if (!activo() || e.target.id !== "concursos-formulario") return;
    e.preventDefault(); if (!validar({ enfocar: true })) return;
    errorArchivo = ""; alComparar(); void editor.comparar();
  });
  raiz.addEventListener("click", (e) => {
    if (!activo()) return;
    const accion = e.target.closest("[data-concurso-accion]")?.dataset.concursoAccion;
    if (accion === "ayuda") { ayuda = !ayuda; pintar(); }
    if (accion === "recargar") { errorCarga = ""; void cargar(); }
    if (accion === "restablecer" && descartar()) { lectura++; errorArchivo = ""; editor.cargar(editor.estado().ejemplo); }
    if (accion === "exportar" && validar({ enfocar: true }) && !Object.keys(editor.estado().invalidos).length) {
      const url = URL.createObjectURL(new Blob([editor.exportar()], { type: "application/json" }));
      const a = document.createElement("a"); a.href = url; a.download = t("archivo_borrador"); a.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
    }
  });
  return Object.freeze({
    estado: () => ({ ...editor.estado(), error: errorArchivo || errorCarga || editor.estado().error }),
    opciones: () => ({ ejemplos, ayuda, filtro, textos }),
    cargar, filtrar,
    alPintar() {
      filtrar();
      for (const c of raiz.querySelectorAll('[data-concurso-ruta][aria-invalid="true"]')) c.setCustomValidity(t(c.dataset.concursoTipo === "puntos" ? "puntos_invalidos" : c.dataset.concursoTipo === "fecha" ? "fecha_invalida" : "nivel_invalido"));
    },
    cancelar() { lectura++; carga?.abort(); editor.cancelar(); },
    desmontar() { lectura++; carga?.abort(); editor.desmontar(); },
  });
}
