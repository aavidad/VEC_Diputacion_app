import { crearPreparacionRectificacionPropia } from "./preparacion-rectificacion-propia.js?v=20261004-personal-rectificacion-v1";
import { traducirPreparacionRectificacion as t } from "./i18n-preparacion-rectificacion-propia.js?v=20261004-personal-rectificacion-v1";
import { formatearFechaHistoriaServicios as fecha, formatearInstanteHistoriaServicios as instante, formatearNumeroHistoriaServicios as numero } from "./i18n-historia-servicios-propia.js?v=20261004-personal-historia-v1";

const nodo = (d, tipo, texto) => { const n = d.createElement(tipo); if (texto !== undefined) n.textContent = texto; return n; };
const valor = (campo, dato) => campo.startsWith("periodo_") ? fecha(dato) : campo === "dias_reconocidos" ? numero(dato) : campo === "estado" ? t(`estados.${dato}`) : dato || t("general.no_consta");

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
  const formulario = nodo(d, "form"); formulario.className = "personal-ficha-corte";
  const entradas = {};
  for (const clave of ["campo", "propuesta", "motivo", "evidencia"]) {
    const grupo = nodo(d, "div"); grupo.className = "personal-ficha-corte-campo";
    const etiqueta = nodo(d, "label", t(`general.${clave}`));
    const input = nodo(d, clave === "campo" ? "select" : "textarea"); input.id = `personal-revision-${clave}`;
    input.dataset.personalRevisionCampo = clave; input.required = true; input.setAttribute("aria-describedby", "personal-revision-error");
    etiqueta.setAttribute("for", input.id);
    if (clave === "campo") for (const campo of preparacion.campos) { const opcion = nodo(d, "option", t(`campos.${campo}`)); opcion.value = campo; input.append(opcion); }
    else { input.rows = 3; input.value = ""; }
    grupo.append(etiqueta, input); formulario.append(grupo); entradas[clave] = input;
  }
  entradas.campo.value = preparacion.campos[0];
  const actual = nodo(d, "p");
  const actualizar = () => { actual.textContent = t("general.actual", { valor: valor(entradas.campo.value, preparacion.valor(entradas.campo.value)) }); };
  actualizar(); entradas.campo.addEventListener("change", actualizar); formulario.append(actual);
  const error = nodo(d, "p"); error.id = "personal-revision-error"; error.setAttribute("role", "alert");
  const revisar = nodo(d, "button", t("general.revisar")); revisar.type = "submit"; revisar.className = "boton-primario"; revisar.dataset.personalRevisionRevisar = "";
  revisar.disabled = typeof reconsultar !== "function";
  const cancelar = nodo(d, "button", t("general.cancelar")); cancelar.type = "button"; cancelar.className = "boton-secundario"; cancelar.dataset.personalRevisionCancelar = "";
  formulario.append(error, revisar, cancelar); cuerpo.append(formulario); panel.append(cabecera, cuerpo); raiz.append(panel);
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
    } catch {
      error.textContent = t("general.invalido");
      for (const input of Object.values(entradas)) input.setAttribute("aria-invalid", "true");
      entradas.propuesta.focus?.(); return;
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
