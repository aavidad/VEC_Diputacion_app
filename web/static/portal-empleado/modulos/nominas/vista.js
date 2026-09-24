import { crearTraductorNominas } from "./i18n.js?v=20260924-f2-web2";

const ESTADOS = new Set(["no_configurado", "cargando", "disponible", "vacio", "denegado", "error"]);
const FORMATO_PERIODO = /^\d{4}-(0[1-9]|1[0-2])$/;
const NOMBRE_PDF = /^[A-Za-z0-9][A-Za-z0-9_.-]{0,119}\.pdf$/i;
const fechaES = new Intl.DateTimeFormat("es-ES", { day: "2-digit", month: "2-digit", year: "numeric" });
let siguienteDetalleId = 0;

function elemento(doc, etiqueta, texto, clase) {
  const nodo = doc.createElement(etiqueta);
  if (texto !== undefined) nodo.textContent = texto;
  if (clase) nodo.className = clase;
  return nodo;
}

function panel(doc, titulo, subtitulo, clase) {
  const seccion = elemento(doc, "section", undefined, `panel ${clase}`);
  const cabecera = elemento(doc, "header", undefined, "cabecera-panel");
  const texto = elemento(doc, "div");
  texto.append(elemento(doc, "h3", titulo), elemento(doc, "p", subtitulo));
  cabecera.append(texto);
  const cuerpo = elemento(doc, "div", undefined, "cuerpo-panel");
  seccion.append(cabecera, cuerpo);
  return { seccion, cuerpo };
}

function validarRespuesta(respuesta) {
  if (!respuesta || typeof respuesta !== "object" || !["disponible", "vacio", "denegado"].includes(respuesta.estado)) throw new TypeError("respuesta de nóminas inválida");
  if (respuesta.estado === "denegado") return { estado: "denegado", recibos: [] };
  if (typeof respuesta.origen !== "string" || !respuesta.origen.trim() || respuesta.origen.length > 120) throw new TypeError("fuente de nóminas inválida");
  if (!Array.isArray(respuesta.recibos) || respuesta.recibos.length > 100) throw new TypeError("historial de nóminas inválido");
  const recibos = respuesta.recibos.map((r) => {
    if (!r || typeof r !== "object" || typeof r.referencia !== "string" || !r.referencia.trim() || r.referencia.length > 120 || !FORMATO_PERIODO.test(r.periodo) || typeof r.tipo !== "string" || !r.tipo.trim() || r.tipo.length > 80 || !Number.isSafeInteger(r.version) || r.version < 1) throw new TypeError("recibo de nómina inválido");
    const fecha = r.fecha_emision ? new Date(r.fecha_emision) : null;
    if (fecha && Number.isNaN(fecha.getTime())) throw new TypeError("fecha de nómina inválida");
    return { referencia: r.referencia, periodo: r.periodo, tipo: r.tipo, version: r.version, fecha, descargable: r.descargable === true };
  });
  if (new Set(recibos.map((r) => r.referencia)).size !== recibos.length) throw new TypeError("referencias de nómina duplicadas");
  const actualizada = respuesta.actualizado_en ? new Date(respuesta.actualizado_en) : null;
  if (actualizada && Number.isNaN(actualizada.getTime())) throw new TypeError("actualización de nómina inválida");
  return { estado: recibos.length ? "disponible" : "vacio", recibos, origen: respuesta.origen.trim(), actualizada };
}

async function validarDocumento(respuesta, ventana) {
  if (!respuesta || !(respuesta.contenido instanceof ventana.Blob) || respuesta.contenido.size === 0 || respuesta.contenido.size > 50 * 1024 * 1024 || respuesta.contenido.type !== "application/pdf" || typeof respuesta.nombre !== "string" || !NOMBRE_PDF.test(respuesta.nombre)) {
    throw new TypeError("documento de nómina inválido");
  }
  if (await respuesta.contenido.slice(0, 5).text() !== "%PDF-") throw new TypeError("documento de nómina inválido");
  return respuesta;
}

/**
 * Montaje: montarVistaNominas({ raiz, anunciar?, registrarDesmontar?, fuente? }).
 * fuente.consultar({ signal }) devuelve { estado, origen, actualizado_en, recibos }.
 * fuente.descargar(referencia, { signal }) devuelve { contenido: Blob PDF, nombre }.
 * El conector autorizado obtiene el original; la vista valida el archivo y dispara
 * la descarga local sin guardar el contenido en almacenamiento web.
 * Sin fuente no se consulta ni se muestra el atlas sintético de presentación.
 */
export function montarVistaNominas({ raiz, anunciar = () => {}, registrarDesmontar, fuente } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") || (fuente !== undefined && typeof fuente?.consultar !== "function")) throw new TypeError("vista de Nóminas no disponible");
  const doc = raiz.ownerDocument;
  const t = crearTraductorNominas();
  const contenedor = elemento(doc, "section", undefined, "modulo-nominas");
  contenedor.dataset.nominas = "";
  const cabecera = elemento(doc, "header", undefined, "nominas-cabecera");
  cabecera.append(elemento(doc, "h2", t("titulo")), elemento(doc, "p", t("descripcion")));
  const ayuda = elemento(doc, "details", undefined, "nominas-ayuda");
  ayuda.append(elemento(doc, "summary", t("ayuda")), elemento(doc, "p", t("ayuda_texto")));
  cabecera.append(ayuda);
  const estadoVisible = elemento(doc, "div", undefined, "nominas-estado");
  estadoVisible.setAttribute("role", "status");
  estadoVisible.setAttribute("aria-live", "polite");
  const metadatos = elemento(doc, "p", undefined, "nominas-metadatos");
  const principal = elemento(doc, "div", undefined, "nominas-principal");
  const historial = panel(doc, t("historial"), t("historial_subtitulo"), "nominas-historial");
  const lateral = elemento(doc, "div", undefined, "nominas-lateral");
  lateral.tabIndex = 0;
  lateral.setAttribute("role", "region");
  lateral.setAttribute("aria-label", t("lateral"));
  const detalle = panel(doc, t("detalle"), t("detalle_subtitulo"), "nominas-detalle");
  detalle.seccion.id = `nominas-detalle-${++siguienteDetalleId}`;
  detalle.seccion.hidden = true;
  const certificados = panel(doc, t("certificados"), t("certificados_subtitulo"), "nominas-certificados");
  certificados.cuerpo.append(elemento(doc, "p", t("certificados_pendientes")));
  const aclaraciones = panel(doc, t("aclaraciones"), t("aclaraciones_subtitulo"), "nominas-aclaraciones");
  aclaraciones.cuerpo.append(elemento(doc, "p", t("aclaraciones_pendientes")));
  const botonAclaracion = elemento(doc, "button", t("solicitar_aclaracion"), "boton boton-secundario nominas-accion-bloqueada");
  botonAclaracion.type = "button";
  botonAclaracion.disabled = true;
  botonAclaracion.title = t("aclaraciones_pendientes");
  aclaraciones.cuerpo.append(botonAclaracion);
  lateral.append(detalle.seccion, certificados.seccion, aclaraciones.seccion);
  principal.append(historial.seccion, lateral);
  contenedor.append(cabecera, estadoVisible, metadatos, principal);
  raiz.append(contenedor);

  let activa = true;
  let controlador = null;
  let estado = "no_configurado";
  let recibos = [];
  let origen = "";
  let actualizada = null;
  let periodo = "";
  let seleccion = "";
  let secuencia = 0;
  let descargando = false;
  const urlsTemporales = new Set();
  const temporizadoresURL = new Set();

  function pintarEstado() {
    estadoVisible.dataset.estado = estado;
    estadoVisible.replaceChildren(
      elemento(doc, "strong", t(`estado_${estado}`)),
      elemento(doc, "span", t(`explicacion_${estado}`)),
    );
    metadatos.replaceChildren();
    if (origen && ["disponible", "vacio"].includes(estado)) {
      metadatos.append(elemento(doc, "span", `${t("origen")}: ${origen}`));
      if (actualizada) metadatos.append(elemento(doc, "span", `${t("actualizado")}: ${fechaES.format(actualizada)}`));
    }
  }

  function pintarDetalle() {
    const recibo = estado === "disponible" ? recibos.find((r) => r.referencia === seleccion) : null;
    detalle.seccion.hidden = !recibo;
    detalle.cuerpo.replaceChildren();
    if (!recibo) return;
    const datos = elemento(doc, "dl", undefined, "nominas-ficha");
    for (const [clave, valor] of [
      ["periodo", recibo.periodo],
      ["tipo", recibo.tipo],
      ["version", String(recibo.version)],
      ["fecha", recibo.fecha ? fechaES.format(recibo.fecha) : t("dato_no_disponible")],
      ["referencia", recibo.referencia],
    ]) {
      datos.append(elemento(doc, "dt", t(clave)), elemento(doc, "dd", valor));
    }
    const descarga = elemento(doc, "button", t("descargar"), "boton boton-secundario nominas-descarga");
    descarga.type = "button";
    descarga.disabled = descargando || !recibo.descargable || typeof fuente?.descargar !== "function";
    if (descarga.disabled) descarga.title = t(descargando ? "descarga_en_curso" : "descarga_no_disponible");
    else descarga.addEventListener("click", async () => {
      if (descargando) return;
      descargando = true;
      const peticion = secuencia;
      const senal = controlador?.signal;
      const focoPrevio = doc.activeElement === descarga;
      descarga.disabled = true;
      descarga.setAttribute("aria-busy", "true");
      try {
        const archivo = await validarDocumento(await fuente.descargar(recibo.referencia, { signal: senal }), doc.defaultView);
        if (!activa || peticion !== secuencia || senal?.aborted || seleccion !== recibo.referencia) return;
        const url = doc.defaultView.URL.createObjectURL(archivo.contenido);
        urlsTemporales.add(url);
        const enlace = elemento(doc, "a");
        enlace.href = url;
        enlace.download = archivo.nombre;
        contenedor.append(enlace);
        try { enlace.click(); } catch (error) {
          doc.defaultView.URL.revokeObjectURL(url);
          urlsTemporales.delete(url);
          throw error;
        } finally { enlace.remove(); }
        const temporizador = doc.defaultView.setTimeout(() => {
          doc.defaultView.URL.revokeObjectURL(url);
          urlsTemporales.delete(url);
          temporizadoresURL.delete(temporizador);
        }, 30000);
        temporizadoresURL.add(temporizador);
        anunciar(t("descarga_iniciada"), "informacion");
      } catch {
        if (activa && peticion === secuencia && !senal?.aborted) anunciar(t("descarga_error"), "error");
      } finally {
        descargando = false;
        if (activa && descarga.isConnected) {
          descarga.disabled = false;
          descarga.removeAttribute("aria-busy");
          if (focoPrevio) descarga.focus({ preventScroll: true });
        } else if (activa) pintarDetalle();
      }
    });
    detalle.cuerpo.append(datos, descarga);
  }

  function pintarHistorial() {
    historial.cuerpo.replaceChildren();
    const barra = elemento(doc, "div", undefined, "nominas-barra");
    const etiqueta = elemento(doc, "label", t("periodo"));
    const selector = elemento(doc, "select");
    const todos = elemento(doc, "option", t("todos"));
    todos.value = "";
    selector.append(todos);
    const periodos = [...new Set(recibos.map((r) => r.periodo))].sort().reverse();
    for (const valor of periodos) {
      const opcion = elemento(doc, "option", valor);
      opcion.value = valor;
      selector.append(opcion);
    }
    selector.value = periodo;
    selector.disabled = estado !== "disponible";
    selector.addEventListener("change", () => {
      periodo = selector.value;
      const visibles = recibos.filter((r) => !periodo || r.periodo === periodo);
      if (!visibles.some((r) => r.referencia === seleccion)) seleccion = visibles[0]?.referencia ?? "";
      pintarHistorial();
      pintarDetalle();
      historial.cuerpo.querySelector("select")?.focus({ preventScroll: true });
    });
    etiqueta.append(selector);
    barra.append(etiqueta);
    if (estado === "error" && fuente) {
      const reintentar = elemento(doc, "button", t("reintentar"), "boton boton-secundario");
      reintentar.type = "button";
      reintentar.addEventListener("click", () => { void consultar(true); });
      barra.append(reintentar);
    }
    historial.cuerpo.append(barra);
    const visibles = recibos.filter((r) => !periodo || r.periodo === periodo);
    if (estado !== "disponible" || !visibles.length) {
      historial.cuerpo.append(elemento(doc, "p", t("sin_recibos"), "nominas-vacio"));
      return;
    }
    const region = elemento(doc, "div", undefined, "nominas-tabla");
    historial.cuerpo.append(elemento(doc, "p", t("tabla_desplazable"), "nominas-ayuda-tabla"));
    region.tabIndex = 0;
    region.setAttribute("role", "region");
    region.setAttribute("aria-label", t("historial"));
    const tabla = elemento(doc, "table");
    const thead = elemento(doc, "thead");
    const encabezado = elemento(doc, "tr");
    for (const clave of ["periodo", "tipo", "version", "fecha", "acciones"]) {
      const th = elemento(doc, "th", t(clave));
      th.scope = "col";
      encabezado.append(th);
    }
    thead.append(encabezado);
    const tbody = elemento(doc, "tbody");
    for (const recibo of visibles) {
      const fila = elemento(doc, "tr");
      for (const valor of [recibo.periodo, recibo.tipo, String(recibo.version), recibo.fecha ? fechaES.format(recibo.fecha) : t("dato_no_disponible")]) fila.append(elemento(doc, "td", valor));
      const celda = elemento(doc, "td");
      const ver = elemento(doc, "button", t("ver_detalle"), "boton boton-secundario nominas-ver");
      ver.type = "button";
      ver.setAttribute("aria-controls", detalle.seccion.id);
      ver.setAttribute("aria-pressed", String(recibo.referencia === seleccion));
      if (recibo.referencia === seleccion) fila.dataset.seleccionada = "true";
      ver.addEventListener("click", () => {
        seleccion = recibo.referencia;
        for (const otra of tbody.querySelectorAll("tr")) {
          const boton = otra.querySelector("button[aria-controls]");
          const elegido = boton === ver;
          boton?.setAttribute("aria-pressed", String(elegido));
          if (elegido) otra.dataset.seleccionada = "true";
          else delete otra.dataset.seleccionada;
        }
        pintarDetalle();
        anunciar(t("seleccionado", recibo), "informacion");
      });
      celda.append(ver);
      fila.append(celda);
      tbody.append(fila);
    }
    tabla.append(thead, tbody);
    region.append(tabla);
    historial.cuerpo.append(region);
  }

  function pintar() {
    if (!activa || !ESTADOS.has(estado)) return;
    pintarEstado();
    pintarHistorial();
    pintarDetalle();
  }

  async function consultar(enfocar = false) {
    if (!activa || !fuente) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++secuencia;
    estado = "cargando";
    recibos = [];
    origen = "";
    actualizada = null;
    periodo = "";
    seleccion = "";
    pintar();
    if (enfocar) { estadoVisible.tabIndex = -1; estadoVisible.focus({ preventScroll: true }); }
    try {
      const respuesta = validarRespuesta(await fuente.consultar({ signal: controlador.signal }));
      if (!activa || actual !== secuencia || controlador.signal.aborted) return;
      ({ estado, recibos, origen = "", actualizada = null } = respuesta);
      seleccion = recibos[0]?.referencia ?? "";
    } catch {
      if (!activa || actual !== secuencia || controlador.signal.aborted) return;
      estado = "error";
    }
    pintar();
  }

  function desmontar() {
    if (!activa) return;
    activa = false;
    ++secuencia;
    controlador?.abort();
    for (const temporizador of temporizadoresURL) doc.defaultView.clearTimeout(temporizador);
    for (const url of urlsTemporales) doc.defaultView.URL.revokeObjectURL(url);
    temporizadoresURL.clear();
    urlsTemporales.clear();
    contenedor.remove();
  }
  registrarDesmontar?.(desmontar);
  pintar();
  if (fuente) void consultar();
  return Object.freeze({ desmontar, consultar });
}
