import { crearTraductorRecuperacionFirmasV2 } from "./i18n-recuperacion-firmas-v2.js?v=20261004-r5-recuperacion-v1";

let siguientePanel = 0;

function localizacionValida(valor) {
  if (typeof valor !== "string") return false;
  try { return Intl.getCanonicalLocales(valor).length === 1; } catch { return false; }
}

function elemento(doc, etiqueta, clase = "", texto = "") {
  const nodo = doc.createElement(etiqueta);
  if (clase) nodo.className = clase;
  if (texto) nodo.textContent = texto;
  return nodo;
}

function par(doc, lista, etiqueta, valor) {
  if (!valor) return;
  const fila = elemento(doc, "div");
  fila.append(elemento(doc, "dt", "", etiqueta));
  const dato = elemento(doc, "dd");
  dato.append(elemento(doc, "code", "", String(valor)));
  fila.append(dato);
  lista.append(fila);
}

// El selector procede de un consumidor CT confiable. Esta hoja no contiene
// formulario ni reconstruye claves desde el expediente o la identidad.
export function montarVistaRecuperacionFirmasV2({
  raiz, cliente, obtenerSolicitud, locale, zonaHoraria,
  mensajes,
} = {}) {
  if (!raiz || typeof raiz.replaceChildren !== "function"
    || typeof cliente?.recuperar !== "function" || typeof obtenerSolicitud !== "function"
    || !localizacionValida(locale)
    || typeof zonaHoraria !== "string" || zonaHoraria.length === 0) {
    throw new TypeError("vista_recuperacion_firmas_no_disponible");
  }
  const doc = raiz.ownerDocument ?? globalThis.document;
  const t = crearTraductorRecuperacionFirmasV2(mensajes);
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium", timeStyle: "short", timeZone: zonaHoraria,
  });
  let montada = true;
  let selector = obtenerSolicitud;
  let controlador = null;
  let generacion = 0;
  const identificador = "ct-recuperacion-firmas-" + (++siguientePanel);

  const panel = elemento(doc, "section", "panel");
  panel.classList.add("ct-recuperacion-firmas");
  panel.dataset.ctRecuperacionFirmasV2 = "";
  panel.setAttribute("aria-labelledby", identificador);
  const cabecera = elemento(doc, "div", "cabecera-panel");
  const textosCabecera = elemento(doc, "div");
  const titulo = elemento(doc, "h3", "", t("titulo"));
  titulo.id = identificador;
  textosCabecera.append(titulo, elemento(doc, "p", "", t("descripcion")));
  cabecera.append(textosCabecera);
  const cuerpo = elemento(doc, "div", "cuerpo-panel");
  const consultar = elemento(doc, "button", "boton-secundario", t("consultar"));
  consultar.type = "button";
  consultar.dataset.ctRecuperacionAccion = "consultar";
  const cancelar = elemento(doc, "button", "boton-secundario", t("cancelar"));
  cancelar.type = "button";
  cancelar.dataset.ctRecuperacionAccion = "cancelar";
  cancelar.hidden = true;
  const estado = elemento(doc, "p");
  estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite");
  const lista = elemento(doc, "div");
  lista.dataset.ctRecuperacionResultado = "";
  cuerpo.append(consultar, cancelar, estado, lista, elemento(doc, "p", "", t("limite")));
  panel.append(cabecera, cuerpo);
  raiz.replaceChildren(panel);

  function limpiar() {
    lista.replaceChildren();
  }

  function mostrar(resultado) {
    limpiar();
    if (resultado.firmas.length === 0) {
      estado.textContent = t("vacio");
      return;
    }
    estado.textContent = "";
    const filas = elemento(doc, "div");
    for (const [indice, firma] of resultado.firmas.entries()) {
      const fila = elemento(doc, "article", "ct-exp-documento");
      fila.append(elemento(doc, "h4", "", t("firma", { numero: indice + 1 })));
      const resumen = elemento(doc, "dl", "ct-resumen");
      const registrada = elemento(doc, "div");
      registrada.append(elemento(doc, "dt", "", t("fecha")));
      const dd = elemento(doc, "dd");
      const tiempo = elemento(doc, "time", "", fecha.format(new Date(firma.registrada_en)));
      tiempo.dateTime = firma.registrada_en;
      dd.append(tiempo);
      registrada.append(dd);
      resumen.append(registrada);
      fila.append(resumen, elemento(doc, "p", "", t(firma.resultado === "firmado" ? "firmado" : "devuelto")));
      const tecnicos = elemento(doc, "details");
      tecnicos.append(elemento(doc, "summary", "", t("detalle_tecnico")));
      const referencias = elemento(doc, "dl", "ct-resumen");
      par(doc, referencias, t("recibo"), firma.recibo_ref);
      par(doc, referencias, t("paso"), firma.paso_orden);
      par(doc, referencias, t("firma_ref"), firma.firma_ref);
      par(doc, referencias, t("revision_sha256"), firma.revision_sha256);
      par(doc, referencias, t("material_root_sha256"), firma.material_root_sha256);
      par(doc, referencias, t("canon_nominal_sha256"), firma.canon_nominal_sha256);
      par(doc, referencias, t("canon_nominal_ref"), firma.canon_nominal_ref);
      tecnicos.append(referencias);
      fila.append(tecnicos);
      filas.append(fila);
    }
    lista.append(filas);
  }

  function detener(cancelada = true) {
    generacion += 1;
    controlador?.abort();
    controlador = null;
    consultar.disabled = false;
    cancelar.hidden = true;
    limpiar();
    estado.textContent = cancelada ? t("cancelada") : "";
  }

  async function cargar() {
    if (!montada || controlador !== null || selector === null) return null;
    detener(false);
    const turno = ++generacion;
    const activo = new AbortController();
    controlador = activo;
    consultar.disabled = true;
    cancelar.hidden = false;
    estado.textContent = t("cargando");
    try {
      const solicitud = await selector();
      if (!montada || generacion !== turno || activo.signal.aborted) return null;
      const resultado = await cliente.recuperar(solicitud, { signal: activo.signal });
      if (!montada || generacion !== turno || activo.signal.aborted) return null;
      mostrar(resultado);
      return resultado;
    } catch (error) {
      if (!montada || generacion !== turno || activo.signal.aborted) return null;
      limpiar();
      estado.textContent = error?.estado === 401 || error?.estado === 403 ? t("denegada")
        : error instanceof TypeError ? t("no_confiable") : t("no_disponible");
      return null;
    } finally {
      if (montada && generacion === turno) {
        controlador = null;
        consultar.disabled = false;
        cancelar.hidden = true;
      }
    }
  }

  const alConsultar = () => { void cargar(); };
  const alCancelar = () => { detener(); };
  consultar.addEventListener("click", alConsultar);
  cancelar.addEventListener("click", alCancelar);

  return Object.freeze({
    consultar: cargar,
    cancelar: () => detener(),
    actualizarSelector(nuevo) {
      detener(false);
      selector = typeof nuevo === "function" ? nuevo : null;
      if (selector === null) raiz.replaceChildren();
      else raiz.replaceChildren(panel);
      return selector !== null;
    },
    desmontar() {
      if (!montada) return;
      montada = false;
      detener(false);
      consultar.removeEventListener("click", alConsultar);
      cancelar.removeEventListener("click", alCancelar);
      raiz.replaceChildren();
    },
  });
}
