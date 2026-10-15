import { crearTraductorRecuperacionFirmasV2 } from "./i18n-recuperacion-firmas-v2.js?v=20261016-firma-historica-v1";

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

// El selector procede de un consumidor CT confiable. Esta hoja no contiene
// formulario ni reconstruye claves desde el expediente o la identidad.
export function montarVistaRecuperacionFirmasV2({
  raiz, cliente, obtenerSolicitud, firmaRef = null, integrada = false, locale, zonaHoraria,
  mensajes,
} = {}) {
  if (!raiz || typeof raiz.replaceChildren !== "function"
    || typeof cliente?.recuperar !== "function" || typeof obtenerSolicitud !== "function"
    || (firmaRef !== null && (typeof firmaRef !== "string" || firmaRef.length === 0))
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

  const panel = elemento(doc, integrada ? "div" : "section", integrada ? "" : "panel");
  panel.classList.add("ct-recuperacion-firmas");
  panel.dataset.ctRecuperacionFirmasV2 = "";
  panel.setAttribute("aria-labelledby", identificador);
  const cabecera = elemento(doc, "div", integrada ? "" : "cabecera-panel");
  const textosCabecera = elemento(doc, "div");
  const titulo = elemento(doc, integrada ? "h5" : "h3", "", t("titulo"));
  titulo.id = identificador;
  textosCabecera.append(titulo);
  cabecera.append(textosCabecera);
  const cuerpo = elemento(doc, "div", integrada ? "" : "cuerpo-panel");
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
  cuerpo.append(consultar, cancelar, estado, lista);
  panel.append(cabecera, cuerpo);
  raiz.replaceChildren(panel);

  function limpiar() {
    lista.replaceChildren();
  }

  function mostrar(resultado) {
    limpiar();
    const firmas = firmaRef === null ? resultado.firmas : resultado.firmas.filter((firma) => firma.firma_ref === firmaRef);
    if (firmas.length === 0) {
      estado.textContent = t("vacio");
      return;
    }
    estado.textContent = "";
    const filas = elemento(doc, "div");
    for (const firma of firmas) {
      const fila = elemento(doc, "article", "ct-exp-documento");
      fila.append(elemento(doc, "h4", "", t("firma", { numero: firma.paso_orden })));
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
      const recibo = elemento(doc, "p", "", t("recibo_con_valor", { recibo: firma.recibo_ref }));
      fila.append(recibo);
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
      if (firmaRef !== null && !resultado.firmas.some((firma) => firma.firma_ref === firmaRef)) {
        throw new TypeError("firma_recuperada_ajena");
      }
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
