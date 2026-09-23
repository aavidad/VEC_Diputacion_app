const OPCIONES = Object.freeze(["institucional", "granate"]);

function nodo(documento, etiqueta, texto = "", clase = "") {
  const elemento = documento.createElement(etiqueta);
  if (texto) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}

/** Vista previa efímera: el controlador común es la única autoridad que cambia el tema. */
export function montarVistaApariencia({ raiz, anunciar = () => {}, t, cargarControlador = () => import("../../../comun/tema-vec.js") } = {}) {
  if (!raiz?.replaceChildren || !raiz.ownerDocument?.createElement || typeof t !== "function" || typeof anunciar !== "function") {
    throw new TypeError("vista de Apariencia no disponible");
  }
  const documento = raiz.ownerDocument;
  let activa = true;
  let controlador = null;
  let seleccion = OPCIONES.includes(documento.documentElement?.getAttribute?.("data-tema"))
    ? documento.documentElement.getAttribute("data-tema") : "institucional";

  const cabecera = () => {
    const header = nodo(documento, "header", "", "administracion-apariencia-cabecera");
    const titulo = nodo(documento, "h3", t("apariencia_titulo"));
    const ayuda = nodo(documento, "button", "?", "administracion-apariencia-ayuda");
    ayuda.type = "button";
    ayuda.setAttribute("aria-label", t("apariencia_ayuda_abrir"));
    ayuda.setAttribute("aria-expanded", "false");
    const textoAyuda = nodo(documento, "p", t("apariencia_ayuda"), "administracion-apariencia-ayuda-texto");
    textoAyuda.hidden = true;
    ayuda.addEventListener("click", () => {
      textoAyuda.hidden = !textoAyuda.hidden;
      ayuda.setAttribute("aria-expanded", String(!textoAyuda.hidden));
      ayuda.setAttribute("aria-label", t(textoAyuda.hidden ? "apariencia_ayuda_abrir" : "apariencia_ayuda_cerrar"));
    });
    header.append(titulo, ayuda, textoAyuda);
    return header;
  };

  const pintarEspera = (clave, reintentar = false) => {
    const panel = nodo(documento, "section", "", "panel administracion-apariencia");
    panel.dataset.aparienciaEstado = reintentar ? "error" : "cargando";
    const mensaje = nodo(documento, "p", t(clave), "administracion-apariencia-estado");
    mensaje.setAttribute("role", "status");
    panel.append(cabecera(), mensaje);
    if (reintentar) {
      const boton = nodo(documento, "button", t("apariencia_reintentar"), "boton-secundario");
      boton.type = "button";
      boton.addEventListener("click", cargar);
      panel.append(boton);
    }
    raiz.replaceChildren(panel);
  };

  const pintarLista = () => {
    const panel = nodo(documento, "section", "", "panel administracion-apariencia");
    panel.dataset.aparienciaEstado = "disponible";
    const form = nodo(documento, "form", "", "administracion-apariencia-formulario");
    const opciones = nodo(documento, "fieldset", "", "administracion-apariencia-opciones");
    opciones.append(nodo(documento, "legend", t("apariencia_elegir")));
    for (const id of OPCIONES) {
      const etiqueta = nodo(documento, "label", "", "administracion-apariencia-opcion");
      const radio = nodo(documento, "input");
      radio.type = "radio";
      radio.name = "tema-vec-previa";
      radio.value = id;
      radio.checked = seleccion === id;
      radio.addEventListener("change", () => { seleccion = id; });
      etiqueta.append(radio, nodo(documento, "span", t(`apariencia_tema_${id}`)), nodo(documento, "small", t("apariencia_revision")));
      opciones.append(etiqueta);
    }
    const estado = nodo(documento, "p", "", "administracion-apariencia-estado");
    estado.setAttribute("role", "status");
    const acciones = nodo(documento, "div", "", "administracion-apariencia-acciones");
    const previsualizar = nodo(documento, "button", t("apariencia_previsualizar"), "boton-primario");
    previsualizar.type = "submit";
    const cancelar = nodo(documento, "button", t("apariencia_cancelar"), "boton-secundario");
    cancelar.type = "button";
    const publicar = nodo(documento, "button", t("apariencia_publicar"), "administracion-accion");
    publicar.type = "button";
    publicar.disabled = true;
    publicar.setAttribute("aria-disabled", "true");
    publicar.title = t("apariencia_sin_autoridad");
    const actualizarEstado = () => {
      const previa = controlador.leerEstado().previsualizacion;
      panel.dataset.aparienciaPrevia = String(previa);
      estado.textContent = t(previa ? "apariencia_estado_previa" : "apariencia_estado_base");
      cancelar.disabled = !previa;
    };
    form.addEventListener("submit", (evento) => {
      evento.preventDefault();
      try {
        controlador.previsualizar({ tema_id: seleccion, revision: 1 });
        actualizarEstado();
        anunciar(t("apariencia_estado_previa"), "info");
      } catch {
        estado.textContent = t("apariencia_error_aplicar");
        anunciar(estado.textContent, "error");
      }
    });
    cancelar.addEventListener("click", () => {
      controlador.cancelarPrevisualizacion();
      actualizarEstado();
      anunciar(t("apariencia_estado_base"), "info");
    });
    acciones.append(previsualizar, cancelar, publicar);
    form.append(opciones, acciones);
    panel.append(cabecera(), form, estado);
    raiz.replaceChildren(panel);
    actualizarEstado();
  };

  const cargar = async () => {
    pintarEspera("apariencia_cargando");
    try {
      const modulo = await cargarControlador();
      if (!activa) return;
      if (typeof modulo?.crearControladorTema !== "function") throw new TypeError("controlador de tema no disponible");
      controlador = modulo.crearControladorTema({ documento });
      pintarLista();
    } catch {
      if (activa) { pintarEspera("apariencia_error_carga", true); anunciar(t("apariencia_error_carga"), "error"); }
    }
  };
  void cargar();
  return Object.freeze({ desmontar() {
    if (!activa) return;
    activa = false;
    controlador?.cancelarPrevisualizacion();
    raiz.replaceChildren();
  } });
}
