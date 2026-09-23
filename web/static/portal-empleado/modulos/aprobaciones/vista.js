import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { crearTraductorAprobaciones } from "./i18n.js?v=20260924-f2-web2";

const ESTADOS_CONSULTA = new Set(["disponible", "vacio", "denegado", "error"]);
const ESTADOS_PENDIENTE = new Set(["pendiente", "vencida", "bloqueada", "subsanacion", "reasignada"]);
const PRIORIDADES = new Set(["alta", "normal", "baja"]);
const LIMITE_PENDIENTES = 200;

function texto(valor, campo, { obligatorio = true, maximo = 240 } = {}) {
  if (!obligatorio && (valor === undefined || valor === null || valor === "")) return "";
  if (typeof valor !== "string" || valor.trim().length === 0 || valor.length > maximo) {
    throw new TypeError("consulta de Aprobaciones inválida: " + campo);
  }
  return valor.trim();
}

function fecha(valor, campo) {
  if (valor === undefined || valor === null || valor === "") return "";
  const original = texto(valor, campo, { maximo: 40 });
  if (!/^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2}))?$/.test(original)) {
    throw new TypeError("consulta de Aprobaciones inválida: " + campo);
  }
  const fechaLeida = new Date(original);
  if (Number.isNaN(fechaLeida.getTime())) throw new TypeError("consulta de Aprobaciones inválida: " + campo);
  if (!original.includes("T") && fechaLeida.toISOString().slice(0, 10) !== original) throw new TypeError("consulta de Aprobaciones inválida: " + campo);
  return new Intl.DateTimeFormat("es-ES", { day: "2-digit", month: "2-digit", year: "numeric", ...(original.includes("T") ? { hour: "2-digit", minute: "2-digit" } : {}) }).format(fechaLeida);
}

function normalizarPendiente(entrada) {
  if (!entrada || typeof entrada !== "object" || Array.isArray(entrada)) throw new TypeError("pendiente de Aprobaciones inválido");
  const estado = texto(entrada.estado, "pendiente.estado");
  const prioridad = texto(entrada.prioridad, "pendiente.prioridad");
  if (!ESTADOS_PENDIENTE.has(estado) || !PRIORIDADES.has(prioridad)) throw new TypeError("estado o prioridad de Aprobaciones inválido");
  if (entrada.evidencias !== undefined && (!Array.isArray(entrada.evidencias) || entrada.evidencias.length > 10)) {
    throw new TypeError("evidencias de Aprobaciones inválidas");
  }
  return Object.freeze({
    referencia: texto(entrada.referencia, "pendiente.referencia", { maximo: 120 }),
    modulo: texto(entrada.modulo, "pendiente.modulo", { maximo: 80 }),
    tipo: texto(entrada.tipo, "pendiente.tipo", { maximo: 120 }),
    solicitante: texto(entrada.solicitante, "pendiente.solicitante", { maximo: 120 }),
    estado, prioridad,
    resumen: texto(entrada.resumen, "pendiente.resumen", { obligatorio: false, maximo: 500 }),
    unidad: texto(entrada.unidad, "pendiente.unidad", { obligatorio: false }),
    responsable: texto(entrada.responsable, "pendiente.responsable", { obligatorio: false }),
    impacto: texto(entrada.impacto, "pendiente.impacto", { obligatorio: false }),
    recibido: fecha(entrada.recibido, "pendiente.recibido"),
    plazo: fecha(entrada.plazo, "pendiente.plazo"),
    evidencias: Object.freeze((entrada.evidencias ?? []).map((item) => texto(item, "pendiente.evidencias", { maximo: 160 }))),
  });
}

/** Normaliza un resultado de lectura. La autorización pertenece a la fuente inyectada. */
export function normalizarConsultaAprobaciones(respuesta) {
  if (!respuesta || typeof respuesta !== "object" || Array.isArray(respuesta) || !ESTADOS_CONSULTA.has(respuesta.estado)) {
    throw new TypeError("consulta de Aprobaciones inválida");
  }
  if (respuesta.estado !== "disponible") {
    if (respuesta.pendientes !== undefined && (!Array.isArray(respuesta.pendientes) || respuesta.pendientes.length !== 0)) {
      throw new TypeError("consulta de Aprobaciones inválida: pendientes");
    }
    return Object.freeze({ estado: respuesta.estado, pendientes: Object.freeze([]) });
  }
  if (!Array.isArray(respuesta.pendientes) || respuesta.pendientes.length > LIMITE_PENDIENTES) {
    throw new TypeError("consulta de Aprobaciones inválida: pendientes");
  }
  const pendientes = respuesta.pendientes.map(normalizarPendiente);
  if (new Set(pendientes.map((item) => item.referencia)).size !== pendientes.length) {
    throw new TypeError("consulta de Aprobaciones inválida: referencias duplicadas");
  }
  return Object.freeze({ estado: pendientes.length ? "disponible" : "vacio", pendientes: Object.freeze(pendientes) });
}

function crear(documento, etiqueta, textoVisible = "", clase = "") {
  const nodo = documento.createElement(etiqueta);
  if (textoVisible) nodo.textContent = textoVisible;
  if (clase) nodo.className = clase;
  return nodo;
}

function cabeceraPanel(documento, titulo, ayuda) {
  const cabecera = crear(documento, "header", "", "cabecera-panel aprobaciones-panel-cabecera");
  cabecera.append(crear(documento, "h3", titulo));
  if (ayuda) {
    const desplegable = crear(documento, "details", "", "aprobaciones-ayuda");
    const resumen = crear(documento, "summary", "?");
    resumen.setAttribute("aria-label", titulo + ": " + ayuda);
    desplegable.append(resumen, crear(documento, "p", ayuda));
    cabecera.append(desplegable);
  }
  return cabecera;
}

function botonInactivo(documento, textoVisible, motivo) {
  const bloque = crear(documento, "div", "", "aprobaciones-accion-bloqueada");
  const boton = crear(documento, "button", textoVisible, "aprobaciones-accion");
  boton.type = "button";
  boton.disabled = true;
  boton.setAttribute("aria-disabled", "true");
  bloque.append(boton, crear(documento, "small", motivo));
  return bloque;
}

function tabla(documento, titulo, cabeceras, filas) {
  const region = crear(documento, "div", "", "aprobaciones-tabla");
  region.tabIndex = 0;
  region.setAttribute("role", "region");
  region.setAttribute("aria-label", titulo);
  const elemento = crear(documento, "table");
  elemento.append(crear(documento, "caption", titulo));
  const cabecera = crear(documento, "thead");
  const cabeceraFila = crear(documento, "tr");
  cabeceras.forEach((nombre) => { const th = crear(documento, "th", nombre); th.scope = "col"; cabeceraFila.append(th); });
  cabecera.append(cabeceraFila);
  const cuerpo = crear(documento, "tbody");
  filas.forEach((fila) => cuerpo.append(fila));
  elemento.append(cabecera, cuerpo);
  region.append(elemento);
  return region;
}

function claseEstado(estado) {
  return estado === "vencida" || estado === "bloqueada" ? "peligro" : estado === "pendiente" ? "violeta" : "neutro";
}

/**
 * Monta una bandeja de lectura. fuentePendientes.consultarPendientes({signal})
 * aporta solo resultados previamente autorizados para el actor y el ámbito.
 * Esta vista nunca decide, firma, descarga, envía ni concede autorización.
 */
export function montarVistaAprobaciones({ raiz, anunciar = () => {}, registrarDesmontar, fuentePendientes } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de Aprobaciones no disponible");
  }
  if (fuentePendientes !== undefined && typeof fuentePendientes?.consultarPendientes !== "function") {
    throw new TypeError("fuente de Aprobaciones no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Aprobaciones no disponible");
  const t = crearTraductorAprobaciones();
  const numero = new Intl.NumberFormat("es-ES");
  let activa = true;
  let estado = fuentePendientes ? "cargando" : "no_configurado";
  let pendientes = [];
  let seleccion = null;
  let modulo = "todos";
  let consulta = "";
  let secuencia = 0;
  let controlador = null;

  const contenedor = crear(documento, "section", "", "modulo-aprobaciones");
  contenedor.dataset.aprobaciones = "";
  const cabecera = crear(documento, "header", "", "aprobaciones-cabecera");
  const introduccion = crear(documento, "div", "", "aprobaciones-intro");
  introduccion.append(crear(documento, "p", t("sobrelinea"), "sobrelinea"), crear(documento, "h2", t("titulo")), crear(documento, "p", t("descripcion")));
  const estadoChip = crear(documento, "span", "", "estado-chip neutro");
  cabecera.append(introduccion, estadoChip);
  const aviso = crear(documento, "p", "", "aprobaciones-aviso");

  const filtros = crear(documento, "form", "", "panel aprobaciones-filtros");
  const filtrosCuerpo = crear(documento, "div", "", "cuerpo-panel aprobaciones-filtros-cuerpo");
  const filtroModulo = crear(documento, "label", t("circuito"));
  const selector = crear(documento, "select");
  filtroModulo.append(selector);
  const filtroTexto = crear(documento, "label", t("buscar"));
  const campo = crear(documento, "input");
  campo.type = "search";
  campo.placeholder = t("buscar_placeholder");
  filtroTexto.append(campo);
  const aplicar = crear(documento, "button", t("aplicar"), "boton-primario aprobaciones-filtrar");
  aplicar.type = "submit";
  filtrosCuerpo.append(filtroModulo, filtroTexto, aplicar);
  filtros.append(cabeceraPanel(documento, t("filtros"), t("ayuda_filtros")), filtrosCuerpo);

  const indicadores = crear(documento, "div", "", "rejilla-kpi aprobaciones-indicadores");
  const principal = crear(documento, "div", "", "aprobaciones-principal");
  const bandeja = crear(documento, "section", "", "panel aprobaciones-bandeja");
  const detalle = crear(documento, "section", "", "panel aprobaciones-detalle");
  const suplencias = crear(documento, "section", "", "panel aprobaciones-suplencias");
  const pie = crear(documento, "div", "", "aprobaciones-limite");

  function visibles() {
    if (estado !== "disponible") return [];
    const termino = consulta.trim().toLocaleLowerCase("es");
    return pendientes.filter((item) => (modulo === "todos" || item.modulo === modulo)
      && (!termino || [item.referencia, item.modulo, item.tipo, item.solicitante, item.resumen, item.unidad].join(" ").toLocaleLowerCase("es").includes(termino)));
  }

  function pintarFiltros() {
    const opciones = [["todos", t("todos_circuitos")], ...[...new Set(pendientes.map((item) => item.modulo))].sort((a, b) => a.localeCompare(b, "es")).map((valor) => [valor, valor])];
    selector.replaceChildren(...opciones.map(([valor, etiqueta]) => { const opcion = crear(documento, "option", etiqueta); opcion.value = valor; return opcion; }));
    selector.value = modulo;
    const habilitado = estado === "disponible";
    selector.disabled = !habilitado;
    campo.disabled = !habilitado;
    aplicar.disabled = !habilitado;
  }

  function pintarIndicadores() {
    const lista = visibles();
    const resumen = [
      ["▤", t("pendientes_visibles"), numero.format(lista.length), estado === "disponible" ? t("consulta_lectura") : t("sin_fuente")],
      ["!", t("alta_prioridad"), numero.format(lista.filter((item) => item.prioridad === "alta").length), t("segun_fuente")],
      ["◫", t("circuitos"), numero.format(new Set(lista.map((item) => item.modulo)).size), t("consulta_lectura")],
      ["◇", t("firma_multiple"), t("firma_no_verificada"), t("portafirmas_pendiente")],
    ];
    indicadores.replaceChildren(...resumen.map(([icono, nombre, valor, nota], indice) => {
      const tarjeta = crear(documento, "article", "", "tarjeta-kpi aprobaciones-kpi aprobaciones-kpi--" + indice);
      const simbolo = crear(documento, "span", icono, "icono-kpi");
      simbolo.setAttribute("aria-hidden", "true");
      tarjeta.append(simbolo, crear(documento, "span", nombre), crear(documento, "strong", valor, "valor-kpi"), crear(documento, "small", nota));
      return tarjeta;
    }));
  }

  function pintarBandeja() {
    const cuerpo = crear(documento, "div", "", "cuerpo-panel");
    const lista = visibles();
    if (lista.length) {
      const filas = lista.map((item) => {
        const tr = crear(documento, "tr");
        if (seleccion?.referencia === item.referencia) tr.dataset.seleccionada = "true";
        [item.referencia, item.modulo, item.solicitante, t("prioridad_" + item.prioridad)].forEach((valor, indice) => {
          const celda = crear(documento, "td");
          celda.append(crear(documento, indice === 0 ? "strong" : "span", valor));
          tr.append(celda);
        });
        const estadoCelda = crear(documento, "td");
        estadoCelda.append(crear(documento, "span", t("estado_item_" + item.estado), "estado-chip aprobaciones-etiqueta " + claseEstado(item.estado)));
        tr.append(estadoCelda);
        const accion = crear(documento, "td");
        const ver = crear(documento, "button", t("ver_detalle"), "aprobaciones-enlace");
        ver.type = "button";
        ver.addEventListener("click", () => {
          seleccion = item;
          bandeja.querySelectorAll("tbody tr").forEach((fila) => { if (fila === tr) fila.dataset.seleccionada = "true"; else delete fila.dataset.seleccionada; });
          pintarDetalle();
          const titulo = detalle.querySelector(".cabecera-panel h3");
          titulo.tabIndex = -1;
          titulo.focus();
          anunciar(t("detalle_anunciado", { referencia: item.referencia }), "informacion");
        });
        accion.append(ver);
        tr.append(accion);
        return tr;
      });
      cuerpo.append(tabla(documento, t("tabla_pendientes"), ["referencia", "circuito", "solicitante", "prioridad", "estado", "acciones"].map(t), filas));
    } else {
      const clave = estado === "disponible" ? "sin_resultados" : "mensaje_" + estado;
      const mensaje = crear(documento, "p", t(clave), "aprobaciones-vacio");
      mensaje.setAttribute("role", estado === "error" || estado === "denegado" ? "alert" : "status");
      cuerpo.append(mensaje);
      if (estado === "error" && fuentePendientes) {
        const reintentar = crear(documento, "button", t("reintentar"), "aprobaciones-enlace aprobaciones-reintentar");
        reintentar.type = "button";
        reintentar.addEventListener("click", () => { void cargar(); const titulo = introduccion.querySelector("h2"); titulo.tabIndex = -1; titulo.focus(); });
        cuerpo.append(reintentar);
      }
    }
    bandeja.replaceChildren(cabeceraPanel(documento, t("bandeja"), t("ayuda_bandeja")), cuerpo);
  }

  function pintarDetalle() {
    const cuerpo = crear(documento, "div", "", "cuerpo-panel aprobaciones-detalle-cuerpo");
    const item = seleccion;
    if (estado === "disponible" && item) {
      cuerpo.append(crear(documento, "p", t("expediente_seleccionado"), "sobrelinea"), crear(documento, "h4", item.resumen || item.tipo), crear(documento, "span", t("estado_item_" + item.estado), "estado-chip aprobaciones-etiqueta " + claseEstado(item.estado)));
      const resumen = crear(documento, "dl", "", "aprobaciones-resumen");
      const campos = [["referencia", item.referencia], ["circuito", item.modulo], ["tipo", item.tipo], ["solicitante", item.solicitante], ["unidad", item.unidad], ["responsable", item.responsable], ["recibido", item.recibido], ["plazo", item.plazo], ["impacto", item.impacto]];
      campos.forEach(([clave, valor]) => { const fila = crear(documento, "div"); fila.append(crear(documento, "dt", t(clave)), crear(documento, "dd", valor || t("sin_dato"))); resumen.append(fila); });
      cuerpo.append(resumen, crear(documento, "h4", t("evidencias")));
      const evidencias = crear(documento, "ul", "", "aprobaciones-documentos");
      if (item.evidencias.length) item.evidencias.forEach((nombre) => { const li = crear(documento, "li"); li.append(crear(documento, "strong", nombre), crear(documento, "span", t("original_pendiente"))); evidencias.append(li); });
      else { const li = crear(documento, "li", t("sin_evidencias")); evidencias.append(li); }
      cuerpo.append(evidencias);
    } else cuerpo.append(crear(documento, "p", t("detalle_vacio"), "aprobaciones-vacio"));
    cuerpo.append(crear(documento, "p", t("separacion_funciones"), "aprobaciones-separacion"), crear(documento, "h4", t("acciones_pendientes")));
    const acciones = crear(documento, "div", "", "aprobaciones-acciones");
    [["aprobar", "motivo_aprobar"], ["devolver", "motivo_devolver"], ["rechazar", "motivo_rechazar"], ["firmar", "motivo_firmar"], ["remitir", "motivo_remitir"], ["descargar", "motivo_descargar"], ["delegar", "motivo_delegar"]].forEach(([accion, motivo]) => acciones.append(botonInactivo(documento, t(accion), t(motivo))));
    cuerpo.append(acciones);
    detalle.replaceChildren(cabeceraPanel(documento, t("detalle"), t("ayuda_detalle")), cuerpo);
  }

  function pintarPie() {
    const lectura = estado === "disponible" || estado === "vacio";
    pie.innerHTML = renderizarEstadoEntrega({
      estado: "visual_pendiente_backend",
      resumen: t(lectura ? "resumen_lectura" : "resumen_entrega"),
      pendientes: ["pendiente_identidad", "pendiente_matriz", "pendiente_firma", "pendiente_operacion", "pendiente_origen"].map(t),
      fuente: { etiqueta: t(lectura ? "fuente_lectura" : fuentePendientes ? "fuente_no_confirmada" : "sin_fuente") },
      conexion: t("conexion"),
    });
  }

  function pintar() {
    if (!activa) return;
    contenedor.dataset.estadoVista = estado;
    contenedor.dataset.estadoEntrega = "visual_pendiente_backend";
    estadoChip.textContent = t("estado_" + estado);
    aviso.textContent = t(estado === "no_configurado" ? "aviso_sin_fuente" : estado === "disponible" || estado === "vacio" ? "aviso_lectura" : "aviso_sin_operaciones");
    pintarFiltros();
    pintarIndicadores();
    pintarBandeja();
    pintarDetalle();
    pintarPie();
  }

  async function cargar() {
    if (!activa || !fuentePendientes) return;
    const actual = ++secuencia;
    controlador?.abort();
    controlador = new AbortController();
    estado = "cargando";
    pendientes = [];
    seleccion = null;
    modulo = "todos";
    consulta = "";
    campo.value = "";
    pintar();
    try {
      const respuesta = await fuentePendientes.consultarPendientes({ signal: controlador.signal });
      const normalizada = normalizarConsultaAprobaciones(respuesta);
      if (!activa || actual !== secuencia) return;
      estado = normalizada.estado;
      pendientes = normalizada.pendientes;
      seleccion = pendientes[0] ?? null;
    } catch {
      if (!activa || actual !== secuencia) return;
      estado = "error";
      pendientes = [];
      seleccion = null;
    }
    pintar();
    anunciar(t("estado_" + estado), estado === "error" || estado === "denegado" ? "error" : "informacion");
  }

  filtros.addEventListener("submit", (evento) => {
    evento.preventDefault();
    if (estado !== "disponible") return;
    modulo = selector.value;
    consulta = campo.value;
    const lista = visibles();
    if (!lista.some((item) => item.referencia === seleccion?.referencia)) seleccion = lista[0] ?? null;
    pintar();
    anunciar(t("visibles_anunciado", { cantidad: numero.format(lista.length) }), "informacion");
  });

  const suplenciasCuerpo = crear(documento, "div", "", "cuerpo-panel");
  suplenciasCuerpo.append(crear(documento, "p", t("suplencias_sin_fuente"), "aprobaciones-vacio"));
  suplencias.replaceChildren(cabeceraPanel(documento, t("suplencias"), t("suplencias_ayuda")), suplenciasCuerpo);
  principal.append(bandeja, detalle);
  contenedor.append(cabecera, aviso, filtros, indicadores, principal, suplencias, pie);
  raiz.append(contenedor);
  pintar();
  if (fuentePendientes) void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    ++secuencia;
    controlador?.abort();
    contenedor.remove();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, refrescar: () => cargar() });
}
