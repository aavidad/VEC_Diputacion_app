import { crearPreparacionRectificacionPropia } from "./preparacion-rectificacion-propia.js?v=20261004-b-rectificacion-validacion-v3";
import { traducirPreparacionRectificacion as t } from "./i18n-preparacion-rectificacion-propia.js?v=20261004-personal-rectificacion-v2";
import { formatearFechaHistoriaServicios as fecha, formatearInstanteHistoriaServicios as instante, formatearNumeroHistoriaServicios as numero } from "./i18n-historia-servicios-propia.js?v=20261004-personal-historia-v1";

const nodo = (d, tipo, texto) => { const n = d.createElement(tipo); if (texto !== undefined) n.textContent = texto; return n; };
const valor = (campo, dato) => campo.startsWith("periodo_") ? fecha(dato) : campo === "dias_reconocidos" ? numero(dato) : campo === "estado" ? t(`estados.${dato}`) : dato || t("general.no_consta");
const ESTADOS_SERVICIO = Object.freeze(["declarado", "comprobado", "reconocido"]);

/** Se monta desde una fila autorizada. Ningún borrador sale del navegador. */
export function montarPreparacionRectificacionPropia({ raiz, datos, seleccion, reconsultar, alInvalidar = () => {}, alCancelar = () => {} }) {
  const d = raiz.ownerDocument, preparacion = crearPreparacionRectificacionPropia(datos, seleccion);
  let activa = true, controlador;
  const panel = nodo(d, "section"); panel.className = "panel personal-ficha-panel";
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-panel";
  cabecera.append(nodo(d, "h3", t("general.titulo")));
  const ayuda = nodo(d, "details"); ayuda.className = "ayuda-contextual";
  const abrir = nodo(d, "summary", "?"); abrir.setAttribute("aria-label", t("general.ayuda"));
  ayuda.append(abrir, nodo(d, "p", t("general.ayuda_texto"))); cabecera.append(ayuda);
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel";
  const estado = nodo(d, "p", t("general.estado")); estado.setAttribute("role", "status"); cuerpo.append(estado);
  const formulario = nodo(d, "form"); formulario.className = "personal-ficha-corte"; formulario.noValidate = true;
  const entradas = {}, erroresCampo = {};
  let erroresVisibles = {};
  const error = nodo(d, "div"); error.id = "personal-revision-error"; error.setAttribute("role", "alert");
  cuerpo.append(error);
  let grupoPropuesta, etiquetaPropuesta;
  for (const clave of ["campo", "propuesta", "motivo", "evidencia"]) {
    const grupo = nodo(d, "div"); grupo.className = "personal-ficha-corte-campo";
    const etiqueta = nodo(d, "label", t(`general.${clave}`));
    const input = nodo(d, clave === "campo" ? "select" : clave === "propuesta" ? "input" : "textarea"); input.id = `personal-revision-${clave}`;
    input.dataset.personalRevisionCampo = clave; input.required = true; input.setAttribute("aria-describedby", `personal-revision-error-${clave}`);
    etiqueta.setAttribute("for", input.id);
    if (clave === "campo") for (const campo of preparacion.campos) { const opcion = nodo(d, "option", t(`campos.${campo}`)); opcion.value = campo; input.append(opcion); }
    else { if (clave !== "propuesta") input.rows = 3; input.value = ""; }
    const aviso = nodo(d, "p"); aviso.id = `personal-revision-error-${clave}`; erroresCampo[clave] = aviso;
    grupo.append(etiqueta, input, aviso); formulario.append(grupo); entradas[clave] = input;
    if (clave === "propuesta") { grupoPropuesta = grupo; etiquetaPropuesta = etiqueta; }
  }
  entradas.campo.value = preparacion.campos[0];
  const actual = nodo(d, "p");
  const prepararControlPropuesta = () => {
    const campo = entradas.campo.value;
    const input = nodo(d, campo === "estado" ? "select" : "input");
    input.id = "personal-revision-propuesta"; input.dataset.personalRevisionCampo = "propuesta";
    input.required = true; input.setAttribute("aria-describedby", "personal-revision-error-propuesta"); input.value = "";
    if (campo === "estado") {
      const vacia = nodo(d, "option", t("general.elegir_estado")); vacia.value = ""; input.append(vacia);
      for (const estado of ESTADOS_SERVICIO) { const opcion = nodo(d, "option", t(`estados.${estado}`)); opcion.value = estado; input.append(opcion); }
    } else if (campo.startsWith("periodo_")) input.type = "date";
    else if (campo === "dias_reconocidos") { input.type = "number"; input.min = "0"; input.step = "1"; }
    else { input.type = "text"; input.maxLength = 300; }
    grupoPropuesta.replaceChildren(etiquetaPropuesta, actual, input, erroresCampo.propuesta); entradas.propuesta = input;
    input.addEventListener("blur", () => validarCampo("propuesta"));
  };
  prepararControlPropuesta();
  const actualizar = () => { actual.textContent = t("general.actual", { valor: valor(entradas.campo.value, preparacion.valor(entradas.campo.value)) }); };
  actualizar(); entradas.campo.addEventListener("change", () => {
    prepararControlPropuesta(); actualizar(); error.textContent = ""; error.replaceChildren();
    erroresVisibles = {};
    for (const input of Object.values(entradas)) input.setAttribute("aria-invalid", "false");
    for (const aviso of Object.values(erroresCampo)) aviso.textContent = "";
  });
  const mostrarErrores = (errores, enfocar = true) => {
    erroresVisibles = { ...errores };
    error.textContent = ""; error.replaceChildren();
    if (Object.keys(errores).length) error.append(nodo(d, "p", t("general.invalido")));
    const lista = nodo(d, "ul");
    let primero;
    for (const clave of ["campo", "propuesta", "motivo", "evidencia"]) {
      const codigo = errores[clave];
      entradas[clave].setAttribute("aria-invalid", codigo ? "true" : "false");
      erroresCampo[clave].textContent = codigo ? t(`errores.${codigo}`) : "";
      if (!codigo) continue;
      const elemento = nodo(d, "li"), enlace = nodo(d, "a", `${t(`general.${clave}`)}: ${t(`errores.${codigo}`)}`);
      enlace.href = `#${entradas[clave].id}`;
      enlace.addEventListener("click", (evento) => { evento.preventDefault(); entradas[clave].focus?.(); });
      elemento.append(enlace); lista.append(elemento);
      primero ??= entradas[clave];
    }
    if (Object.keys(errores).length) error.append(lista);
    if (enfocar) (primero || entradas.propuesta).focus?.();
  };
  const validarCampo = (clave) => {
    if (!activa || controlador) return;
    let codigo;
    try { preparacion.preparar(Object.fromEntries(Object.entries(entradas).map(([k, input]) => [k, input.value]))); }
    catch (causa) {
      if (causa?.codigo !== "borrador_invalido") return;
      codigo = causa.errores?.[clave];
    }
    const errores = { ...erroresVisibles };
    if (codigo) errores[clave] = codigo; else delete errores[clave];
    mostrarErrores(errores, false);
  };
  for (const clave of ["motivo", "evidencia"]) entradas[clave].addEventListener("blur", () => validarCampo(clave));
  const revisar = nodo(d, "button", t("general.revisar")); revisar.type = "submit"; revisar.className = "boton-primario"; revisar.dataset.personalRevisionRevisar = "";
  revisar.disabled = typeof reconsultar !== "function";
  const cancelar = nodo(d, "button", t("general.cancelar")); cancelar.type = "button"; cancelar.className = "boton-secundario"; cancelar.dataset.personalRevisionCancelar = "";
  formulario.append(revisar, cancelar); cuerpo.append(formulario); panel.append(cabecera, cuerpo); raiz.append(panel);
  const limpiar = () => {
    if (!activa) return; activa = false; controlador?.abort(); preparacion.limpiar();
    for (const input of Object.values(entradas)) input.value = "";
    panel.replaceChildren(); panel.remove?.();
  };
  cancelar.addEventListener("click", () => { limpiar(); alCancelar(); });
  formulario.addEventListener("submit", async (evento) => {
    evento.preventDefault(); if (!activa || controlador || typeof reconsultar !== "function") return;
    try {
      preparacion.preparar(Object.fromEntries(Object.entries(entradas).map(([k, input]) => [k, input.value])));
    } catch (causa) {
      if (causa?.codigo !== "borrador_invalido") { limpiar(); alInvalidar(causa); return; }
      mostrarErrores(causa.errores || {}); return;
    }
    controlador = new AbortController(); const signal = controlador.signal;
    for (const input of Object.values(entradas)) input.value = "";
    cuerpo.replaceChildren(nodo(d, "p", t("general.estado")), nodo(d, "p", t("general.comprobando")), cancelar);
    try {
      const respuesta = await reconsultar({ signal });
      if (!activa || signal.aborted) return;
      const borrador = preparacion.revisar(respuesta);
      cuerpo.replaceChildren(nodo(d, "p", t("general.estado")), nodo(d, "p", t("general.sin_presentar")));
      const lista = nodo(d, "dl");
      for (const [clave, contenido] of [["campo", t(`campos.${borrador.campo}`)], ["actual_etiqueta", valor(borrador.campo, borrador.valor_actual)], ["propuesta", borrador.propuesta], ["motivo", borrador.motivo], ["evidencia", borrador.evidencia]]) lista.append(nodo(d, "dt", t(`general.${clave}`)), nodo(d, "dd", contenido));
      const procedencia = nodo(d, "details"), detalle = nodo(d, "dl"); procedencia.append(nodo(d, "summary", t("general.procedencia")));
      for (const [clave, contenido] of [["servicio", borrador.servicio_ref], ["revision", numero(borrador.traza.version)], ["fuente", borrador.traza.fuente_ref], ["version_fuente", numero(borrador.traza.fuente_version)], ["acto", borrador.traza.acto_ref], ["corte_desde", fecha(borrador.corte.efectos_desde)], ["corte_hasta", fecha(borrador.corte.efectos_hasta)], ["conocido_en", instante(borrador.corte.conocido_en)], ["recibo", borrador.recibo_ref]]) detalle.append(nodo(d, "dt", t(`general.${clave}`)), nodo(d, "dd", contenido));
      procedencia.append(detalle); cuerpo.append(lista, procedencia, cancelar); lista.setAttribute("tabindex", "-1"); lista.focus?.(); preparacion.limpiar();
    } catch (causa) {
      if (!activa || signal.aborted) return;
      limpiar(); alInvalidar(causa);
    }
  });
  revisar.focus?.(); return Object.freeze({ desmontar: limpiar });
}
