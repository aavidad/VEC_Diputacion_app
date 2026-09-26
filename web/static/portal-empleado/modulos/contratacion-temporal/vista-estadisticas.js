/**
 * Vista de estadísticas de contratación temporal (C18 / G16).
 *
 * Incluye:
 * - Selector de periodo (anual, mensual, semanal) y rango de fechas.
 * - Tabla accesible con las series temporales y totales.
 * - Gráfico SVG sin dependencias externas.
 * - Exportación de datos a CSV generada en el cliente.
 * - Sin uso de la palabra "demo".
 */

import { generarCSVEstadisticas } from "./contrato-estadisticas.js";
import { consultarEstadisticas } from "./cliente-http-estadisticas.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const traducirCT = crearTraductorContratacionTemporal();

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

const textoCT = (clave, variables) => escaparHTML(traducirCT(clave, variables));

function formatearNumero(valor) {
  return new Intl.NumberFormat("es-ES").format(Number(valor || 0));
}

export function renderizarGraficoSVG(series) {
  if (!Array.isArray(series) || series.length === 0) {
    return `<div class="vacio-controlado"><p>${textoCT("ct_txt_no_hay_datos_suficientes_para_generar_el_grafico")}</p></div>`;
  }

  const anchoTotal = 760;
  const altoTotal = 300;
  const margenIzq = 50;
  const margenDer = 20;
  const margenSup = 30;
  const margenInf = 50;

  const anchoGrafico = anchoTotal - margenIzq - margenDer;
  const altoGrafico = altoTotal - margenSup - margenInf;

  // Determinar valor máximo para escala vertical
  let maxValor = 5;
  for (const s of series) {
    const maxSerie = Math.max(s.altas, s.llamamientos, s.formalizaciones, s.cierres, s.incidencias);
    if (maxSerie > maxValor) maxValor = maxSerie;
  }
  // Redondear maxValor hacia arriba
  maxValor = Math.ceil(maxValor * 1.15);

  const colores = {
    altas: "#1d70b8",
    llamamientos: "#00703c",
    formalizaciones: "#4c2c92",
    cierres: "#5694ca",
    incidencias: "#d4351c",
  };

  const metricas = ["altas", "llamamientos", "formalizaciones", "cierres", "incidencias"];
  const nSeries = series.length;
  const anchoGrupo = anchoGrafico / nSeries;
  const anchoBarra = Math.max(4, Math.min(18, (anchoGrupo - 12) / metricas.length));

  // Líneas de guía horizontales (4 niveles)
  const lineasGuia = [];
  for (let i = 0; i <= 4; i++) {
    const yVal = Math.round((maxValor / 4) * i);
    const yPos = margenSup + altoGrafico - (yVal / maxValor) * altoGrafico;
    lineasGuia.push(`
      <line x1="${margenIzq}" y1="${yPos}" x2="${anchoTotal - margenDer}" y2="${yPos}" stroke="#e0e0e0" stroke-width="1" stroke-dasharray="${i === 0 ? 'none' : '4,4'}" />
      <text x="${margenIzq - 8}" y="${yPos + 4}" font-size="11" text-anchor="end" fill="#505a5f">${yVal}</text>
    `);
  }

  // Barras de cada grupo
  const barrasSVG = [];
  series.forEach((s, idx) => {
    const xBase = margenIzq + idx * anchoGrupo + (anchoGrupo - anchoBarra * metricas.length) / 2;
    metricas.forEach((m, mIdx) => {
      const valor = s[m];
      const alturaBarra = (valor / maxValor) * altoGrafico;
      const xBarra = xBase + mIdx * anchoBarra;
      const yBarra = margenSup + altoGrafico - alturaBarra;
      barrasSVG.push(`
        <rect x="${xBarra}" y="${yBarra}" width="${anchoBarra - 1}" height="${alturaBarra}" fill="${colores[m]}" rx="2">
          <title>${s.inicio} — ${m}: ${valor}</title>
        </rect>
      `);
    });

    // Etiqueta del periodo en eje X
    const xCentro = margenIzq + idx * anchoGrupo + anchoGrupo / 2;
    const etiquetaCorta = s.inicio.length > 7 ? s.inicio.slice(5) : s.inicio;
    barrasSVG.push(`
      <text x="${xCentro}" y="${altoTotal - margenInf + 16}" font-size="11" text-anchor="middle" fill="#0b0c0c">${escaparHTML(etiquetaCorta)}</text>
    `);
  });

  // Leyenda accesible
  const leyenda = `
    <g class="leyenda" transform="translate(${margenIzq}, 12)">
      <rect x="0" y="0" width="12" height="12" fill="${colores.altas}" rx="2" />
      <text x="16" y="10" font-size="11" fill="#0b0c0c">${textoCT("ct_txt_altas")}</text>
      <rect x="75" y="0" width="12" height="12" fill="${colores.llamamientos}" rx="2" />
      <text x="91" y="10" font-size="11" fill="#0b0c0c">${textoCT("ct_txt_llamamientos")}</text>
      <rect x="200" y="0" width="12" height="12" fill="${colores.formalizaciones}" rx="2" />
      <text x="216" y="10" font-size="11" fill="#0b0c0c">${textoCT("ct_txt_formalizaciones")}</text>
      <rect x="330" y="0" width="12" height="12" fill="${colores.cierres}" rx="2" />
      <text x="346" y="10" font-size="11" fill="#0b0c0c">${textoCT("ct_txt_cierres")}</text>
      <rect x="420" y="0" width="12" height="12" fill="${colores.incidencias}" rx="2" />
      <text x="436" y="10" font-size="11" fill="#0b0c0c">${textoCT("ct_txt_incidencias")}</text>
    </g>
  `;

  return `
    <div class="grafico-svg-contenedor" style="overflow-x: auto; max-width: 100%;">
      <svg role="img" aria-labelledby="titulo-grafico-ct desc-grafico-ct" viewBox="0 0 ${anchoTotal} ${altoTotal}" style="width: 100%; min-width: 500px; height: auto; font-family: sans-serif;">
        <title id="titulo-grafico-ct">${textoCT("ct_txt_grafico_de_evolucion_temporal_de_contratacion")}</title>
        <desc id="desc-grafico-ct">${textoCT("ct_txt_distribucion_de_altas_llamamientos_formalizacion")}</desc>
        ${leyenda}
        ${lineasGuia.join("")}
        ${barrasSVG.join("")}
      </svg>
    </div>
  `;
}

export function renderizarTablaEstadisticas(estadisticas) {
  if (!estadisticas || !Array.isArray(estadisticas.series) || estadisticas.series.length === 0) {
    return `<div class="vacio-controlado" role="status"><p>${textoCT("ct_txt_no_hay_datos_estadisticos_para_el_periodo_y_rang")}</p></div>`;
  }

  const filas = estadisticas.series.map((s) => `
    <tr>
      <td><strong>${escaparHTML(s.inicio)}</strong></td>
      <td class="numero">${formatearNumero(s.altas)}</td>
      <td class="numero">${formatearNumero(s.llamamientos)}</td>
      <td class="numero">${formatearNumero(s.formalizaciones)}</td>
      <td class="numero">${formatearNumero(s.cierres)}</td>
      <td class="numero">${formatearNumero(s.incidencias)}</td>
    </tr>
  `).join("");

  const t = estadisticas.totales;
  const pieTotales = `
    <tfoot>
      <tr>
        <th scope="row">${textoCT("ct_txt_totales")}</th>
        <th class="numero">${formatearNumero(t.altas)}</th>
        <th class="numero">${formatearNumero(t.llamamientos)}</th>
        <th class="numero">${formatearNumero(t.formalizaciones)}</th>
        <th class="numero">${formatearNumero(t.cierres)}</th>
        <th class="numero">${formatearNumero(t.incidencias)}</th>
      </tr>
    </tfoot>
  `;

  return `
    <div class="tabla-contenedor tabla-contenedor--estadisticas">
      <table class="tabla-datos tabla-datos--estadisticas">
        <caption>${textoCT("ct_txt_estadisticas_por_periodo", { periodo: estadisticas.periodo })}</caption>
        <thead>
          <tr>
            <th scope="col">${textoCT("ct_txt_periodo")}</th>
            <th scope="col" class="numero">${textoCT("ct_txt_altas")}</th>
            <th scope="col" class="numero">${textoCT("ct_txt_llamamientos")}</th>
            <th scope="col" class="numero">${textoCT("ct_txt_formalizaciones")}</th>
            <th scope="col" class="numero">${textoCT("ct_txt_cierres")}</th>
            <th scope="col" class="numero">${textoCT("ct_txt_incidencias")}</th>
          </tr>
        </thead>
        <tbody>
          ${filas}
        </tbody>
        ${pieTotales}
      </table>
    </div>
  `;
}

export function renderizarFormularioFiltros({ periodo = "mensual", desde = "", hasta = "" } = {}) {
  const opcionesPeriodo = [
    ["anual", traducirCT("ct_txt_anual")],
    ["mensual", traducirCT("ct_txt_mensual")],
    ["semanal", traducirCT("ct_txt_semanal")],
  ].map(([val, etiqueta]) => `
    <option value="${escaparHTML(val)}"${val === periodo ? " selected" : ""}>${escaparHTML(etiqueta)}</option>
  `).join("");

  return `
    <form class="barra-filtros-estadisticas" data-ct-form="filtros-estadisticas" role="search" aria-label="${textoCT("ct_txt_filtros_de_estadisticas_de_contratacion")}">
      <div class="campo-filtro">
        <label for="filtro-est-periodo">${textoCT("ct_txt_periodo_2")}</label>
        <select id="filtro-est-periodo" name="periodo">${opcionesPeriodo}</select>
      </div>
      <div class="campo-filtro">
        <label for="filtro-est-desde">${textoCT("ct_txt_desde")}</label>
        <input type="date" id="filtro-est-desde" name="desde" value="${escaparHTML(desde)}">
      </div>
      <div class="campo-filtro">
        <label for="filtro-est-hasta">${textoCT("ct_txt_hasta")}</label>
        <input type="date" id="filtro-est-hasta" name="hasta" value="${escaparHTML(hasta)}">
      </div>
      <div class="acciones-filtro">
        <button type="submit" class="boton-primario">${textoCT("ct_txt_consultar")}</button>
        <button type="button" class="boton-secundario" data-ct-accion="exportar-csv">${textoCT("ct_txt_exportar_csv")}</button>
      </div>
    </form>
  `;
}

export function renderizarVistaEstadisticas({ estadoEstadisticas, filtros }) {
  const encabezado = `
    <header class="cabecera-vista">
      <h2>${textoCT("ct_txt_estadisticas_de_contratacion_temporal")}</h2>
      <p>${textoCT("ct_txt_cuadro_de_evolucion_temporal_altas_llamamientos")}</p>
    </header>
  `;

  if (!estadoEstadisticas || estadoEstadisticas.carga === "cargando") {
    return `
      ${encabezado}
      <section class="panel">
        <div class="cabecera-panel"><h3>${textoCT("ct_txt_series_estadisticas")}</h3><span class="estado-chip info">${textoCT("ct_txt_consultando")}</span></div>
        <div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true">
          <p><strong>${textoCT("ct_txt_cargando_estadisticas")}</strong></p>
          <p>${textoCT("ct_txt_obteniendo_series_temporales_agregadas_del_servi")}</p>
        </div>
      </section>
    `;
  }

  if (estadoEstadisticas.carga === "error") {
    return `
      ${encabezado}
      <section class="panel">
        <div class="cabecera-panel"><h3>${textoCT("ct_txt_series_estadisticas")}</h3><span class="estado-chip peligro">${textoCT("ct_txt_consulta_fallida")}</span></div>
        <div class="cuerpo-panel vacio-controlado" role="alert">
          <p><strong>${textoCT("ct_txt_error_al_consultar_estadisticas")}</strong></p>
          <p>${escaparHTML(estadoEstadisticas.error || traducirCT("ct_txt_no_se_pudieron_obtener_las_series_estadisticas"))}</p>
          <div class="acciones-vista">
            <button type="button" class="boton-secundario" data-ct-accion="reintentar-estadisticas">${textoCT("ct_txt_reintentar")}</button>
          </div>
        </div>
      </section>
    `;
  }

  if (estadoEstadisticas.carga === "denegado") {
    return `
      ${encabezado}
      <section class="panel">
        <div class="cabecera-panel"><h3>${textoCT("ct_txt_series_estadisticas")}</h3><span class="estado-chip peligro">${textoCT("ct_txt_acceso_denegado")}</span></div>
        <div class="cuerpo-panel vacio-controlado" role="alert">
          <p><strong>${textoCT("ct_txt_acceso_denegado")}</strong></p>
          <p>${textoCT("ct_txt_la_sesion_no_dispone_de_permisos_suficientes_par")}</p>
        </div>
      </section>
    `;
  }

  const datos = estadoEstadisticas.datos;
  const formularioHtml = renderizarFormularioFiltros(filtros);
  const graficoHtml = datos ? renderizarGraficoSVG(datos.series) : "";
  const tablaHtml = datos ? renderizarTablaEstadisticas(datos) : "";

  return `
    ${encabezado}
    <section class="panel">
      <div class="cabecera-panel">
        <h3>${textoCT("ct_txt_filtros_de_periodo_y_rango_temporal")}</h3>
      </div>
      <div class="cuerpo-panel">
        ${formularioHtml}
      </div>
    </section>
    <section class="panel">
      <div class="cabecera-panel">
        <h3>${textoCT("ct_txt_evolucion_grafica_por_periodo")}</h3>
      </div>
      <div class="cuerpo-panel">
        ${graficoHtml}
      </div>
    </section>
    <section class="panel">
      <div class="cabecera-panel">
        <h3>${textoCT("ct_txt_series_detalladas_y_totales_acumulados")}</h3>
      </div>
      <div class="cuerpo-panel">
        ${tablaHtml}
      </div>
    </section>
  `;
}

export function montarVistaEstadisticas({ raiz, cliente, anunciar, descargarCSVImpl }) {
  let montada = true;
  let generacionConsulta = 0;
  let filtros = {
    periodo: "mensual",
    desde: "",
    hasta: "",
  };

  let estadoEstadisticas = {
    carga: "cargando",
    datos: null,
    error: "",
  };

  const ejecutarConsulta = typeof cliente === "function" ? cliente : consultarEstadisticas;

  function renderizar() {
    raiz.innerHTML = renderizarVistaEstadisticas({ estadoEstadisticas, filtros });
  }

  async function cargar() {
    if (!montada) return;
    const generacionActual = ++generacionConsulta;
    estadoEstadisticas = { carga: "cargando", datos: null, error: "" };
    renderizar();
    let res;
    try {
      res = await ejecutarConsulta({ ...filtros });
    } catch {
      res = { ok: false, mensaje: traducirCT("ct_txt_no_se_pudieron_obtener_las_series_estadisticas") };
    }
    if (!montada || generacionActual !== generacionConsulta) return;
    if (res?.ok) {
      estadoEstadisticas = { carga: "listo", datos: res.datos, error: "" };
    } else if (res?.status === 403) {
      estadoEstadisticas = { carga: "denegado", datos: null, error: res.mensaje };
    } else {
      estadoEstadisticas = { carga: "error", datos: null, error: res?.mensaje };
    }
    renderizar();
    if (res?.ok && montada && generacionActual === generacionConsulta && typeof anunciar === "function") {
      anunciar(traducirCT("ct_txt_estadisticas_actualizadas"));
    }
  }

  function descargarCSV() {
    if (!estadoEstadisticas.datos) return;
    const contenidoCSV = generarCSVEstadisticas(estadoEstadisticas.datos);
    if (typeof descargarCSVImpl === "function") {
      descargarCSVImpl(contenidoCSV);
      return;
    }
    const blob = new Blob([contenidoCSV], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const enlace = document.createElement("a");
    enlace.setAttribute("href", url);
    enlace.setAttribute("download", `estadisticas-contratacion-${filtros.periodo}.csv`);
    document.body.appendChild(enlace);
    enlace.click();
    document.body.removeChild(enlace);
    URL.revokeObjectURL(url);
    if (typeof anunciar === "function") anunciar(traducirCT("ct_txt_archivo_csv_generado_correctamente"));
  }

  function manejarClick(evento) {
    const boton = evento.target?.closest?.("[data-ct-accion]");
    if (!boton) return;
    const accion = boton.dataset.ctAccion;
    if (accion === "exportar-csv") {
      evento.preventDefault();
      descargarCSV();
    } else if (accion === "reintentar-estadisticas") {
      evento.preventDefault();
      void cargar();
    }
  }

  function manejarSubmit(evento) {
    const form = evento.target?.closest?.('[data-ct-form="filtros-estadisticas"]');
    if (!form) return;
    evento.preventDefault();
    const datosForm = new FormData(form);
    filtros = {
      periodo: datosForm.get("periodo") || "mensual",
      desde: datosForm.get("desde") || "",
      hasta: datosForm.get("hasta") || "",
    };
    void cargar();
  }

  raiz.addEventListener("click", manejarClick);
  raiz.addEventListener("submit", manejarSubmit);

  void cargar();

  return {
    desmontar() {
      if (!montada) return;
      montada = false;
      generacionConsulta++;
      raiz.removeEventListener("click", manejarClick);
      raiz.removeEventListener("submit", manejarSubmit);
      raiz.innerHTML = "";
    },
    cargar,
    descargarCSV,
    obtenerFiltros: () => ({ ...filtros }),
    obtenerEstado: () => ({ ...estadoEstadisticas }),
  };
}
