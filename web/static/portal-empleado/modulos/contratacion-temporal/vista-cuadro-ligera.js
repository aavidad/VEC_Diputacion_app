/** Entrada de solo lectura a la lista CT. El detalle conserva su montaje propio. */
import { localizacionDe } from "../../../comun/idioma.js";
import { FASE_RRHH_DE_ORIGEN } from "./fases-rrhh-datos.js?v=20261007-pantallas-textos-final-v1";
import { FILTRO_LISTA_INICIAL, filtroListaValido } from "./recuentos-peticiones.js?v=20261007-pantallas-textos-final-v1";
import { renderizarListaPeticiones } from "./vista-expedientes-lista.js?v=20261007-pantallas-textos-final-v1";
import { crearTraductorCuadroCT, prepararTextosContratacionVista } from "./i18n-vistas.js?v=20261007-pantallas-textos-final-v1";

const SOLICITUD_INICIAL = Object.freeze({
  filtros: Object.freeze({ texto: "", estado_clave: "", fase_clave: "" }),
  paginacion: Object.freeze({ limite: 100, cursor: "" }),
});
const escapar = (valor) => String(valor ?? "").replace(/[&<>"']/gu,
  (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[caracter]);

function proyectarPagina(pagina, fases, locale) {
  if (!Array.isArray(pagina?.expedientes) || typeof pagina.generada_en !== "string"
    || typeof pagina.hay_mas !== "boolean") throw new TypeError("cuadro CT no válido");
  const fecha = new Intl.DateTimeFormat(locale, { dateStyle: "short", timeZone: "Europe/Madrid" });
  const expedientes = pagina.expedientes.map((entrada) => {
    const fase = FASE_RRHH_DE_ORIGEN[entrada.fase_clave];
    const estadoClave = entrada.estado_clave === "espera_externa" ? "espera" : entrada.estado_clave;
    const plazo = entrada.plazo_fase;
    return Object.freeze({
      expediente_ref: entrada.expediente_ref,
      numero_visible: entrada.numero_visible,
      centro: entrada.centro_ref,
      categoria: entrada.categoria_ref,
      fase_clave: entrada.fase_clave,
      fase_actual: fases[`fase_${fase}`] ?? entrada.fase_clave,
      estado_clave: estadoClave,
      estado: fases[`estado_${estadoClave}`] ?? entrada.estado_clave,
      plazo_estado: plazo?.estado ?? "no_calculado",
      plazo_ultimo_dia: plazo?.ultimo_dia ?? "",
      plazo: plazo?.ultimo_dia ? fecha.format(new Date(`${plazo.ultimo_dia}T00:00:00Z`)) : "",
      fecha_solicitud: entrada.creado_en,
      version: entrada.version,
      ...(entrada.urgente === true ? { urgente: true } : {}),
    });
  });
  return Object.freeze({ generado_en: pagina.generada_en, expedientes: Object.freeze(expedientes),
    paginacion: Object.freeze({ cursor_siguiente: pagina.hay_mas ? pagina.cursor_siguiente : "" }) });
}

/**
 * Requiere el cliente CT ya autorizado. Consulta una página con el contrato
 * existente; el callback abrirDetalle importa el montaje completo solo al pulsar.
 */
export async function montarCuadroContratacionLigero({
  raiz, cliente, idioma, abrirDetalle, abrirAlta = null, mostrarError,
  filtroLista = null, signal = null,
  nombreCentro = (referencia) => referencia, nombreCategoria = (referencia) => referencia,
} = {}) {
  if (!raiz?.addEventListener || !raiz?.querySelector
    || typeof cliente?.consultarCuadroRRHH !== "function"
    || typeof abrirDetalle !== "function" || typeof mostrarError !== "function"
    || (signal !== null && (typeof signal?.addEventListener !== "function"
      || typeof signal.aborted !== "boolean"))) {
    throw new TypeError("dependencias de cuadro CT incompletas");
  }
  const montajeAbortado = Object.freeze({ recargar: async () => {}, desmontar() {} });
  if (signal?.aborted) return montajeAbortado;
  let vigente = true;
  let controlador = null;
  let preparado;
  let cuadro;
  let paginaIndice = 0;
  const cursores = [""];
  let filtro = FILTRO_LISTA_INICIAL;
  let filtroRuta = filtroLista;
  let filtroServidorActual = null;
  let temporizadorBusqueda = null;
  let filtroConsultado = null;

  function filtroServidor() {
    if (filtroServidorActual) return filtroServidorActual;
    if (filtroRuta === null) return SOLICITUD_INICIAL.filtros;
    if (!filtroRuta || typeof filtroRuta !== "object" || Array.isArray(filtroRuta)) {
      throw new TypeError("filtro CT de ruta no válido");
    }
    const validado = filtroListaValido({ mostrar: "todas", ...filtroRuta });
    if (Object.keys(filtroRuta).some((clave) => !Object.hasOwn(FILTRO_LISTA_INICIAL, clave))
      || Object.entries(filtroRuta).some(([clave, valor]) => valor !== undefined && valor !== null
        && String(valor).trim() !== validado[clave])
      || validado.fase || validado.centro || validado.categoria
      || !["incidencia", "espera", "todas"].includes(validado.mostrar)) {
      throw Object.assign(new Error("filtro CT sin alcance completo en el servidor"),
        { codigo: "filtro_servidor_no_disponible" });
    }
    filtro = validado;
    filtroServidorActual = Object.freeze({ texto: validado.texto,
      estado_clave: validado.mostrar === "incidencia" ? "incidencia"
        : (validado.mostrar === "espera" ? "espera_externa" : ""),
      fase_clave: "" });
    return filtroServidorActual;
  }
  const ayudas = Object.freeze({
    numeroVisible: (numero) => numero,
    centroVisible: (referencia) => Object.freeze({ etiqueta: nombreCentro(referencia), referencia }),
  });

  const cuadroVisible = () => ({ ...cuadro,
    expedientes: cuadro.expedientes.map((entrada) => ({ ...entrada, categoria: nombreCategoria(entrada.categoria) })) });

  const filtroResultados = () => ({ ...filtro,
    ...(filtroServidorActual?.texto ? { texto: "" } : {}),
    ...(filtroServidorActual?.estado_clave ? { mostrar: "todas" } : {}),
  });

  function paginacion(t) {
    const anterior = paginaIndice > 0
      ? `<button type="button" data-ct-pagina="anterior">${escapar(t("paginacion_marco_anterior"))}</button>` : "";
    const siguiente = cuadro.paginacion.cursor_siguiente
      ? `<button type="button" data-ct-pagina="siguiente">${escapar(t("paginacion_marco_siguiente"))}</button>` : "";
    return anterior || siguiente ? `<nav aria-label="${escapar(t("paginacion_marco_etiqueta"))}">${anterior}${siguiente}</nav>` : "";
  }

  function pintar() {
    if (!vigente || !cuadro) return;
    raiz.lang = preparado.idioma;
    const t = crearTraductorCuadroCT(preparado);
    raiz.innerHTML = `${preparado.reintentar
      ? `<p class="ct-exp-aviso-textos" role="status">${escapar(t("lista_textos_respaldo"))}</p>` : ""}`
      + renderizarListaPeticiones({ cuadro: cuadroVisible(), filtros: {} },
      t, filtro, ayudas, paginacion(t), { altaDisponible: typeof abrirAlta === "function",
        actualizarDisponible: true, filtroResultados: filtroResultados() });
  }

  function pintarConFocoDeFiltro() {
    const activo = raiz.ownerDocument?.activeElement;
    const nombre = activo?.name;
    const conservaFoco = ["texto", "fase", "centro", "categoria", "mostrar"].includes(nombre)
      && raiz.contains?.(activo) && activo.closest?.("[data-ct-exp-filtros-locales]");
    const valor = conservaFoco ? activo.value : null;
    const inicio = nombre === "texto" ? activo.selectionStart : null;
    const fin = nombre === "texto" ? activo.selectionEnd : null;
    const direccion = nombre === "texto" ? activo.selectionDirection : null;
    pintar();
    if (!conservaFoco) return;
    const reemplazo = raiz.querySelector(`[data-ct-exp-filtros-locales] [name="${nombre}"]`);
    if (!reemplazo || reemplazo.value !== valor) return;
    reemplazo.focus?.({ preventScroll: true });
    if (nombre === "texto" && Number.isInteger(inicio) && Number.isInteger(fin)) {
      reemplazo.setSelectionRange?.(inicio, fin, direccion ?? "none");
    }
  }

  function restaurarFoco(selector) {
    if (!selector) return;
    const control = raiz.querySelector(selector) ?? raiz.querySelector("#ct-exp-lista-titulo-panel");
    if (!control) return;
    if (!control.matches?.(selector)) control.setAttribute?.("tabindex", "-1");
    control.focus?.();
  }

  async function cargar({ reintentar = false, soloResultados = false, foco = "" } = {}) {
    if (!vigente || signal?.aborted) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = controlador;
    try {
      preparado = await prepararTextosContratacionVista("cuadro", { idioma, reintentar });
      if (!vigente || signal?.aborted || actual.signal.aborted) return;
      const solicitud = { filtros: filtroServidor(),
        paginacion: { limite: SOLICITUD_INICIAL.paginacion.limite, cursor: cursores[paginaIndice] } };
      filtroConsultado = JSON.stringify(filtro);
      const pagina = await cliente.consultarCuadroRRHH(solicitud, { signal: actual.signal });
      if (!vigente || actual !== controlador) return;
      cuadro = proyectarPagina(pagina, preparado.secciones["portal.fases_rrhh"], localizacionDe(preparado.idioma));
      if (soloResultados) pintarConFocoDeFiltro();
      else pintar();
      restaurarFoco(foco);
    } catch (error) {
      if (vigente && actual === controlador && !actual.signal.aborted) {
        const filtroNoDisponible = error?.codigo === "filtro_servidor_no_disponible";
        const mensaje = filtroNoDisponible ? crearTraductorCuadroCT(preparado)("lista_filtro_no_disponible") : undefined;
        mostrarError(raiz, { error, mensaje, reintentar: () => {
          if (filtroNoDisponible) {
            filtroRuta = null;
            filtroServidorActual = null;
            filtro = FILTRO_LISTA_INICIAL;
            paginaIndice = 0;
            cursores.length = 1;
          }
          return cargar({ reintentar: true });
        } });
      }
    }
  }

  async function alPulsar(evento) {
    const boton = evento.target?.closest?.("[data-ct-exp-abrir], [data-ct-exp-vista], [data-ct-pagina], [data-ct-exp-quitar-filtro], [data-ct-exp-recargar]");
    if (!boton || !vigente || signal?.aborted) return;
    if (boton.dataset.ctExpAbrir) {
      const expediente = cuadro?.expedientes.find(({ expediente_ref: ref }) => ref === boton.dataset.ctExpAbrir);
      if (!expediente) return;
      try {
        const textos = await prepararTextosContratacionVista("expediente", { idioma: preparado.idioma });
        if (vigente && !signal?.aborted) await abrirDetalle({ expedienteRef: expediente.expediente_ref,
          version: expediente.version, textos, idioma: textos.idioma });
      } catch (error) { if (vigente) mostrarError(raiz, { error, reintentar: () => alPulsar(evento) }); }
    } else if (boton.dataset.ctExpVista === "alta" && typeof abrirAlta === "function") {
      try {
        const textos = await prepararTextosContratacionVista("alta", { idioma: preparado.idioma });
        if (vigente && !signal?.aborted) await abrirAlta({ textos, idioma: textos.idioma });
      } catch (error) { if (vigente) mostrarError(raiz, { error, reintentar: () => alPulsar(evento) }); }
    } else if (boton.hasAttribute?.("data-ct-exp-recargar")) {
      await cargar({ reintentar: true, foco: "[data-ct-exp-recargar]" });
    } else if (boton.dataset.ctPagina === "siguiente" && cuadro?.paginacion.cursor_siguiente) {
      cursores.push(cuadro.paginacion.cursor_siguiente);
      paginaIndice++;
      await cargar({ foco: '[data-ct-pagina="siguiente"]' });
    } else if (boton.dataset.ctPagina === "anterior" && paginaIndice > 0) {
      paginaIndice--;
      await cargar({ foco: '[data-ct-pagina="anterior"]' });
    } else if (boton.dataset.ctExpQuitarFiltro) {
      filtro = boton.dataset.ctExpQuitarFiltro === "todos" ? FILTRO_LISTA_INICIAL
        : filtroListaValido({ ...filtro, [boton.dataset.ctExpQuitarFiltro]: "" });
      filtroRuta = { ...filtro, mostrar: filtro.mostrar === "en_tramite" ? "todas" : filtro.mostrar };
      filtroServidorActual = null;
      paginaIndice = 0;
      cursores.length = 1;
      await cargar({ foco: "#ct-exp-lista-titulo-panel" });
    }
  }

  function alFiltrar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-exp-filtros-locales]");
    if (!formulario || !cuadro || !vigente || signal?.aborted) return;
    const datos = Object.fromEntries(new FormData(formulario).entries());
    if (evento.type === "input" && evento.target?.name === "texto" && datos.mostrar === "en_tramite") {
      datos.mostrar = "todas";
      const selector = formulario.elements?.namedItem?.("mostrar");
      if (selector) selector.value = "todas";
    }
    const nuevoFiltro = filtroListaValido(datos);
    const claveNueva = JSON.stringify(nuevoFiltro);
    if (claveNueva === filtroConsultado && temporizadorBusqueda === null) return;
    if (claveNueva !== JSON.stringify(filtro)) {
      filtro = nuevoFiltro;
      filtroRuta = { ...datos };
      filtroServidorActual = null;
      filtroConsultado = null;
      controlador?.abort();
    }
    if (temporizadorBusqueda !== null) clearTimeout(temporizadorBusqueda);
    const consultarFiltro = () => {
      temporizadorBusqueda = null;
      paginaIndice = 0;
      cursores.length = 1;
      void cargar({ soloResultados: true });
    };
    if (evento.type === "input" && evento.target?.name === "texto") {
      temporizadorBusqueda = setTimeout(consultarFiltro, 250);
    } else {
      consultarFiltro();
    }
  }

  function alEnviarFiltro(evento) {
    if (!evento.target?.matches?.("[data-ct-exp-filtros-locales]")) return;
    evento.preventDefault();
    alFiltrar({ target: evento.target, type: "change" });
  }

  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("input", alFiltrar);
  raiz.addEventListener("change", alFiltrar);
  raiz.addEventListener("submit", alEnviarFiltro);
  const alAbortar = () => desmontar();
  signal?.addEventListener("abort", alAbortar, { once: true });
  if (signal?.aborted) desmontar();
  await cargar();
  function desmontar() {
      vigente = false;
      controlador?.abort();
      if (temporizadorBusqueda !== null) clearTimeout(temporizadorBusqueda);
      signal?.removeEventListener?.("abort", alAbortar);
      raiz.removeEventListener?.("click", alPulsar);
      raiz.removeEventListener?.("input", alFiltrar);
      raiz.removeEventListener?.("change", alFiltrar);
      raiz.removeEventListener?.("submit", alEnviarFiltro);
  }
  return Object.freeze({ recargar: () => cargar({ reintentar: true }), desmontar });
}
