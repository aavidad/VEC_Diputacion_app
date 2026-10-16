import { validarSelectoresRecuperacionFirmas } from "./selectores-recuperacion-firmas.js?v=20261016-firma-historica-v2";
import { crearTraductorRecuperacionFirmasV2 } from "./i18n-recuperacion-firmas-v2.js?v=20261016-firma-historica-v2";
import { montarVistaRecuperacionFirmasV2 } from "./vista-recuperacion-firmas-v2.js?v=20261016-firma-historica-v2";

const DOCUMENTOS = new Set([
  "informe_definitivo", "resolucion", "diligencia", "toma_posesion", "notificacion", "comunicacion_centro",
]);
let siguienteResultado = 0;

function nodo(documento, etiqueta, clase = "", texto = "") {
  const elemento = documento.createElement(etiqueta);
  if (clase) elemento.className = clase;
  elemento.textContent = texto;
  return elemento;
}

export function selectoresHistoricosDesdeEstado(estado) {
  const expediente = estado?.expediente;
  if (estado?.vista !== "expediente" || estado.carga !== "listo"
    || expediente?.demostracion !== false || expediente.expediente_ref !== estado.expediente_ref
    || !Number.isSafeInteger(expediente.version) || expediente.version < 1
    || !Object.hasOwn(expediente, "selectores_recuperacion_firmas")) return null;
  try {
    return validarSelectoresRecuperacionFirmas(expediente.selectores_recuperacion_firmas,
      expediente.expediente_ref, expediente.version);
  } catch { return null; }
}

export function crearGestorRecuperacionFirmasHistoricas({
  raiz, cliente, mensajes, errorRecuperacionFirmas = null, mensajeError, locale, zonaHoraria,
} = {}) {
  if (!raiz?.querySelector || !raiz?.ownerDocument?.createElement
    || typeof mensajeError !== "function") throw new TypeError("gestor_recuperacion_no_disponible");
  let panel = null;
  let vista = null;
  let activo = null;
  let ultimoError = null;

  function retirar() {
    vista?.desmontar();
    vista = null;
    activo = null;
    panel?.remove();
    panel = null;
  }

  function montarSiProcede(estado) {
    retirar();
    const expediente = estado?.expediente;
    if (estado?.vista !== "expediente" || estado.carga !== "listo"
      || expediente?.demostracion !== false || expediente.expediente_ref !== estado.expediente_ref
      || !Object.hasOwn(expediente, "selectores_recuperacion_firmas")) return false;
    if (Array.isArray(expediente.selectores_recuperacion_firmas)
      && expediente.selectores_recuperacion_firmas.length === 0) return false;
    const zona = raiz.querySelector(".ct-exp-contenido");
    if (!zona) return false;
    const entradas = selectoresHistoricosDesdeEstado(estado);
    let t = null;
    let errorCatalogo = errorRecuperacionFirmas;
    if (entradas?.length && !errorRecuperacionFirmas) {
      try { t = crearTraductorRecuperacionFirmasV2(mensajes); }
      catch (error) { errorCatalogo = error; }
    }
    const doc = raiz.ownerDocument;
    if (!entradas?.length || t === null || typeof cliente?.recuperar !== "function") {
      const causa = !entradas?.length ? "selectores"
        : errorRecuperacionFirmas || t === null ? "textos" : "cliente";
      if (causa !== ultimoError) {
        console.warn({ origen: "ct.recuperacion_firmas_historicas", causa,
          tipo: typeof errorCatalogo?.name === "string" ? errorCatalogo.name : null });
        ultimoError = causa;
      }
      panel = nodo(doc, "section", "panel ct-recuperacion-firmas-historicas");
      panel.dataset.ctRecuperacionHistorica = "";
      const aviso = nodo(doc, "p", "cuerpo-panel", mensajeError());
      aviso.setAttribute("role", "alert");
      panel.append(aviso);
      zona.append(panel);
      return false;
    }
    ultimoError = null;
    panel = nodo(doc, "section", "panel ct-recuperacion-firmas-historicas");
    panel.dataset.ctRecuperacionHistorica = "";
    const cabecera = nodo(doc, "div", "cabecera-panel");
    cabecera.append(nodo(doc, "h3", "", t("historicas_titulo")));
    const cuerpo = nodo(doc, "div", "cuerpo-panel");
    const grupos = new Map();
    for (const [indice, entrada] of entradas.entries()) {
      const ordinal = indice + 1;
      const documento = DOCUMENTOS.has(entrada.selector.documento)
        ? t(`historicas_documento_${entrada.selector.documento}`) : t("historicas_documento");
      let grupo = grupos.get(entrada.selector.documento);
      if (!grupo) {
        grupo = nodo(doc, "section", "ct-exp-documento");
        grupo.append(nodo(doc, "h4", "", documento));
        grupos.set(entrada.selector.documento, grupo);
        cuerpo.append(grupo);
      }
      const fila = nodo(doc, "div");
      fila.append(nodo(doc, "h5", "", t("firma", { numero: ordinal })));
      fila.append(nodo(doc, "p", "", t("historicas_paso", { numero: entrada.selector.paso_orden })));
      fila.append(nodo(doc, "p", "", t("historicas_version", { numero: entrada.selector.version_expediente })));
      const boton = nodo(doc, "button", "boton-secundario", t("historicas_consultar"));
      boton.type = "button";
      boton.dataset.ctRecuperacionHistoricaConsultar = "";
      boton.setAttribute("aria-label", t("historicas_consultar_firma", {
        documento, ordinal, paso: entrada.selector.paso_orden, version: entrada.selector.version_expediente,
      }));
      const resultado = nodo(doc, "div");
      resultado.dataset.ctRecuperacionHistoricaResultado = "";
      resultado.id = `ct-recuperacion-historica-${++siguienteResultado}`;
      boton.setAttribute("aria-controls", resultado.id);
      boton.setAttribute("aria-expanded", "false");
      boton.addEventListener("click", () => {
        if (panel === null || !panel.isConnected || activo === boton) return;
        vista?.desmontar();
        vista = null;
        if (activo) activo.setAttribute("aria-expanded", "false");
        activo = boton;
        boton.setAttribute("aria-expanded", "true");
        vista = montarVistaRecuperacionFirmasV2({ raiz: resultado, cliente, integrada: true,
          obtenerSolicitud: () => entrada.selector, firmaRef: entrada.firma_ref,
          mensajes, locale, zonaHoraria });
        void vista.consultar();
      });
      fila.append(boton, resultado);
      grupo.append(fila);
    }
    panel.append(cabecera, cuerpo);
    zona.append(panel);
    return true;
  }

  return Object.freeze({ montarSiProcede, retirar });
}
