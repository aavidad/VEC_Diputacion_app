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

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function formatearNumero(valor) {
  return new Intl.NumberFormat("es-ES").format(Number(valor || 0));
}

export function renderizarGraficoSVG(series) {
  if (!Array.isArray(series) || series.length === 0) {
    return `<div class="vacio-controlado"><p>No hay datos suficientes para generar el gráfico temporal.</p></div>`;
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
      <text x="16" y="10" font-size="11" fill="#0b0c0c">Altas</text>
      <rect x="75" y="0" width="12" height="12" fill="${colores.llamamientos}" rx="2" />
      <text x="91" y="10" font-size="11" fill="#0b0c0c">Llamamientos</text>
      <rect x="200" y="0" width="12" height="12" fill="${colores.formalizaciones}" rx="2" />
      <text x="216" y="10" font-size="11" fill="#0b0c0c">Formalizaciones</text>
      <rect x="330" y="0" width="12" height="12" fill="${colores.cierres}" rx="2" />
      <text x="346" y="10" font-size="11" fill="#0b0c0c">Cierres</text>
      <rect x="420" y="0" width="12" height="12" fill="${colores.incidencias}" rx="2" />
      <text x="436" y="10" font-size="11" fill="#0b0c0c">Incidencias</text>
    </g>
  `;

  return `
    <div class="grafico-svg-contenedor" style="overflow-x: auto; max-width: 100%;">
      <svg role="img" aria-labelledby="titulo-grafico-ct desc-grafico-ct" viewBox="0 0 ${anchoTotal} ${altoTotal}" style="width: 100%; min-width: 500px; height: auto; font-family: sans-serif;">
        <title id="titulo-grafico-ct">Gráfico de evolución temporal de contratación</title>
        <desc id="desc-grafico-ct">Distribución de altas, llamamientos, formalizaciones, cierres e incidencias por periodo.</desc>
        ${leyenda}
        ${lineasGuia.join("")}
        ${barrasSVG.join("")}
      </svg>
    </div>
  `;
}

export function renderizarTablaEstadisticas(estadisticas) {
  if (!estadisticas || !Array.isArray(estadisticas.series) || estadisticas.series.length === 0) {
    return `<div class="vacio-controlado" role="status"><p>No hay datos estadísticos para el periodo y rango seleccionados.</p></div>`;
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
        <th scope="row">TOTALES</th>
        <th class="numero">${formatearNumero(t.altas)}</th>
        <th class="numero">${formatearNumero(t.llamamientos)}</th>
        <th class="numero">${formatearNumero(t.formalizaciones)}</th>
        <th class="numero">${formatearNumero(t.cierres)}</th>
        <th class="numero">${formatearNumero(t.incidencias)}</th>
      </tr>
    </tfoot>
  `;

  return `
    <div class="tabla-contenedor">
      <table class="tabla-datos">
        <caption>Estadísticas agregadas de contratación temporal por periodo (${escaparHTML(estadisticas.periodo)})</caption>
        <thead>
          <tr>
            <th scope="col">Periodo</th>
            <th scope="col" class="numero">Altas</th>
            <th scope="col" class="numero">Llamamientos</th>
            <th scope="col" class="numero">Formalizaciones</th>
            <th scope="col" class="numero">Cierres</th>
            <th scope="col" class="numero">Incidencias</th>
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
    ["anual", "Anual"],
    ["mensual", "Mensual"],
    ["semanal", "Semanal"],
  ].map(([val, etiqueta]) => `
    <option value="${escaparHTML(val)}"${val === periodo ? " selected" : ""}>${escaparHTML(etiqueta)}</option>
  `).join("");

  return `
    <form class="barra-filtros-estadisticas" data-ct-form="filtros-estadisticas" role="search" aria-label="Filtros de estadísticas de contratación">
      <div class="campo-filtro">
        <label for="filtro-est-periodo">Periodo:</label>
        <select id="filtro-est-periodo" name="periodo">${opcionesPeriodo}</select>
      </div>
      <div class="campo-filtro">
        <label for="filtro-est-desde">Desde:</label>
        <input type="date" id="filtro-est-desde" name="desde" value="${escaparHTML(desde)}">
      </div>
      <div class="campo-filtro">
        <label for="filtro-est-hasta">Hasta:</label>
        <input type="date" id="filtro-est-hasta" name="hasta" value="${escaparHTML(hasta)}">
      </div>
      <div class="acciones-filtro">
        <button type="submit" class="boton-primario">Consultar</button>
        <button type="button" class="boton-secundario" data-ct-accion="exportar-csv">Exportar CSV</button>
      </div>
    </form>
  `;
}

export function renderizarVistaEstadisticas({ estadoEstadisticas, filtros }) {
  const encabezado = `
    <header class="cabecera-vista">
      <h2>Estadísticas de contratación temporal</h2>
      <p>Cuadro de evolución temporal: altas, llamamientos, formalizaciones, cierres e incidencias.</p>
    </header>
  `;

  if (!estadoEstadisticas || estadoEstadisticas.carga === "cargando") {
    return `
      ${encabezado}
      <section class="panel">
        <div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true">
          <p><strong>Cargando estadísticas…</strong></p>
          <p>Obteniendo series temporales agregadas del servidor.</p>
        </div>
      </section>
    `;
  }

  if (estadoEstadisticas.carga === "error") {
    return `
      ${encabezado}
      <section class="panel">
        <div class="cuerpo-panel vacio-controlado" role="alert">
          <p><strong>Error al consultar estadísticas</strong></p>
          <p>${escaparHTML(estadoEstadisticas.error || "No se pudieron obtener las series estadísticas.")}</p>
          <div class="acciones-vista">
            <button type="button" class="boton-secundario" data-ct-accion="reintentar-estadisticas">Reintentar</button>
          </div>
        </div>
      </section>
    `;
  }

  if (estadoEstadisticas.carga === "denegado") {
    return `
      ${encabezado}
      <section class="panel">
        <div class="cuerpo-panel vacio-controlado" role="alert">
          <p><strong>Acceso denegado</strong></p>
          <p>La sesión no dispone de permisos suficientes para consultar las estadísticas de contratación temporal.</p>
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
        <h3>Filtros de periodo y rango temporal</h3>
      </div>
      <div class="cuerpo-panel">
        ${formularioHtml}
      </div>
    </section>
    <section class="panel">
      <div class="cabecera-panel">
        <h3>Evolución gráfica por periodo</h3>
      </div>
      <div class="cuerpo-panel">
        ${graficoHtml}
      </div>
    </section>
    <section class="panel">
      <div class="cabecera-panel">
        <h3>Series detalladas y totales acumulados</h3>
      </div>
      <div class="cuerpo-panel">
        ${tablaHtml}
      </div>
    </section>
  `;
}

export function montarVistaEstadisticas({ raiz, cliente, anunciar, descargarCSVImpl }) {
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
    estadoEstadisticas = { carga: "cargando", datos: null, error: "" };
    renderizar();
    const res = await ejecutarConsulta(filtros);
    if (res.ok) {
      estadoEstadisticas = { carga: "listo", datos: res.datos, error: "" };
      if (typeof anunciar === "function") anunciar("Estadísticas actualizadas");
    } else if (res.status === 403) {
      estadoEstadisticas = { carga: "denegado", datos: null, error: res.mensaje };
    } else {
      estadoEstadisticas = { carga: "error", datos: null, error: res.mensaje };
    }
    renderizar();
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
    if (typeof anunciar === "function") anunciar("Archivo CSV generado correctamente");
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
