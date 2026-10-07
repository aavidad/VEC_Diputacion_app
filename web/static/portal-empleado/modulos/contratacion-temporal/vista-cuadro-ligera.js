/** Entrada de solo lectura a la lista CT. El detalle conserva su montaje propio. */
import { localizacionDe } from "../../../comun/idioma.js";
import { FASE_RRHH_DE_ORIGEN } from "./fases-rrhh-datos.js";
import { FILTRO_LISTA_INICIAL, filtroListaValido } from "./recuentos-peticiones.js?v=20261006-resumen-inicio-v2";
import { renderizarListaPeticiones, renderizarResultadosLista } from "./vista-expedientes-lista.js?v=20261006-resumen-inicio-v2";
import { crearTraductorCuadroCT, prepararTextosContratacionVista } from "./i18n-vistas.js";

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
  nombreCentro = (referencia) => referencia, nombreCategoria = (referencia) => referencia,
} = {}) {
  if (!raiz?.addEventListener || !raiz?.querySelector
    || typeof cliente?.consultarCuadroRRHH !== "function"
    || typeof abrirDetalle !== "function" || typeof mostrarError !== "function") {
    throw new TypeError("dependencias de cuadro CT incompletas");
  }
  let vigente = true;
  let controlador = null;
  let preparado;
  let cuadro;
  let paginaIndice = 0;
  const cursores = [""];
  let filtro = FILTRO_LISTA_INICIAL;
  const ayudas = Object.freeze({
    numeroVisible: (numero) => numero,
    centroVisible: (referencia) => Object.freeze({ etiqueta: nombreCentro(referencia), referencia }),
  });

  const cuadroVisible = () => ({ ...cuadro,
    expedientes: cuadro.expedientes.map((entrada) => ({ ...entrada, categoria: nombreCategoria(entrada.categoria) })) });

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
      t, filtro, ayudas, paginacion(t), { altaDisponible: typeof abrirAlta === "function", actualizarDisponible: true });
  }

  async function cargar({ reintentar = false } = {}) {
    controlador?.abort();
    controlador = new AbortController();
    const actual = controlador;
    try {
      preparado = await prepararTextosContratacionVista("cuadro", { idioma, reintentar });
      const solicitud = { filtros: SOLICITUD_INICIAL.filtros,
        paginacion: { limite: SOLICITUD_INICIAL.paginacion.limite, cursor: cursores[paginaIndice] } };
      const pagina = await cliente.consultarCuadroRRHH(solicitud, { signal: actual.signal });
      if (!vigente || actual !== controlador) return;
      cuadro = proyectarPagina(pagina, preparado.secciones["portal.fases_rrhh"], localizacionDe(preparado.idioma));
      pintar();
    } catch (error) {
      if (vigente && actual === controlador && !actual.signal.aborted) {
        mostrarError(raiz, { error, reintentar: () => cargar({ reintentar: true }) });
      }
    }
  }

  async function alPulsar(evento) {
    const boton = evento.target?.closest?.("[data-ct-exp-abrir], [data-ct-exp-vista], [data-ct-pagina], [data-ct-exp-quitar-filtro], [data-ct-exp-recargar]");
    if (!boton || !vigente) return;
    if (boton.dataset.ctExpAbrir) {
      const expediente = cuadro?.expedientes.find(({ expediente_ref: ref }) => ref === boton.dataset.ctExpAbrir);
      if (!expediente) return;
      try {
        const textos = await prepararTextosContratacionVista("expediente", { idioma: preparado.idioma });
        if (vigente) await abrirDetalle({ expedienteRef: expediente.expediente_ref,
          version: expediente.version, textos, idioma: textos.idioma });
      } catch (error) { if (vigente) mostrarError(raiz, { error, reintentar: () => alPulsar(evento) }); }
    } else if (boton.dataset.ctExpVista === "alta" && typeof abrirAlta === "function") {
      try {
        const textos = await prepararTextosContratacionVista("alta", { idioma: preparado.idioma });
        if (vigente) await abrirAlta({ textos, idioma: textos.idioma });
      } catch (error) { if (vigente) mostrarError(raiz, { error, reintentar: () => alPulsar(evento) }); }
    } else if (boton.hasAttribute?.("data-ct-exp-recargar")) {
      await cargar({ reintentar: true });
    } else if (boton.dataset.ctPagina === "siguiente" && cuadro?.paginacion.cursor_siguiente) {
      cursores.push(cuadro.paginacion.cursor_siguiente);
      paginaIndice++;
      await cargar();
    } else if (boton.dataset.ctPagina === "anterior" && paginaIndice > 0) {
      paginaIndice--;
      await cargar();
    } else if (boton.dataset.ctExpQuitarFiltro) {
      filtro = boton.dataset.ctExpQuitarFiltro === "todos" ? FILTRO_LISTA_INICIAL
        : filtroListaValido({ ...filtro, [boton.dataset.ctExpQuitarFiltro]: "" });
      pintar();
    }
  }

  function alFiltrar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-exp-filtros-locales]");
    if (!formulario || !cuadro) return;
    const datos = Object.fromEntries(new FormData(formulario).entries());
    filtro = filtroListaValido(datos);
    const resultado = raiz.querySelector("[data-ct-exp-resultados]");
    if (resultado) resultado.outerHTML = renderizarResultadosLista({ cuadro: cuadroVisible(), filtros: {} },
      crearTraductorCuadroCT(preparado), filtro, ayudas);
  }

  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("input", alFiltrar);
  raiz.addEventListener("change", alFiltrar);
  await cargar();
  return Object.freeze({
    recargar: () => cargar({ reintentar: true }),
    desmontar() {
      vigente = false;
      controlador?.abort();
      raiz.removeEventListener?.("click", alPulsar);
      raiz.removeEventListener?.("input", alFiltrar);
      raiz.removeEventListener?.("change", alFiltrar);
    },
  });
}
