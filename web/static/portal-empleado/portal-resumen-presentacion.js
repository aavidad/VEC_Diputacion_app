/**
 * Renderizador del cuadro de mando sintético de presentación.
 *
 * La ruta productiva usa `portal-panel-interno.js`; esta unidad se mantiene
 * aislada para que los datos de presentación no condicionen el portal real.
 */
export function crearVistaResumenPresentacion({
  escaparHTML,
  encabezadoVista,
  etiquetaFuentePanel,
  numero,
  obtenerDatosPanel,
  porcentajeSeguro,
} = {}) {
  if ([escaparHTML, encabezadoVista, etiquetaFuentePanel, numero, obtenerDatosPanel,
    porcentajeSeguro].some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias de la vista resumen de presentación no válidas");
  }

  function tarjetaKPI(sigla, valor, etiqueta, destino) {
    return `
      <article class="tarjeta-kpi">
        <span class="icono-kpi" aria-hidden="true">${escaparHTML(sigla)}</span>
        <div>
          <strong class="valor-kpi">${escaparHTML(valor)}</strong>
          <span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span>
          <button type="button" class="enlace-kpi" data-vista="${escaparHTML(destino)}">Ver detalle →</button>
        </div>
      </article>`;
  }

  return function renderizarResumenPresentacion() {
    const datosPanel = obtenerDatosPanel();
    const indicadores = datosPanel.indicadores;
    const distribucion = datosPanel.distribucion_global;
    const totalDistribucion = Object.values(distribucion)
      .reduce((total, valor) => total + (Number(valor) || 0), 0);
    const porcentajes = [distribucion.disponibles, distribucion.en_llamamiento, distribucion.contratados]
      .map((valor) => totalDistribucion > 0
        ? porcentajeSeguro((Number(valor) || 0) * 100 / totalDistribucion)
        : 0);
    const contratosMes = Array.isArray(datosPanel.series.contratos_mes)
      ? datosPanel.series.contratos_mes
      : [];
    const bolsasTabla = datosPanel.bolsas.slice(0, 5).map((bolsa) => `
      <tr>
        <td><button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-id="${escaparHTML(bolsa.id)}">${escaparHTML(bolsa.nombre)}</button></td>
        <td>${escaparHTML(bolsa.categoria)}</td>
        <td>${numero(bolsa.integrantes)}</td>
        <td>${numero(bolsa.llamamiento)}</td>
        <td><span class="barra-cobertura" role="meter" aria-label="Cobertura ${porcentajeSeguro(bolsa.cobertura)}%" aria-valuenow="${porcentajeSeguro(bolsa.cobertura)}" aria-valuemin="0" aria-valuemax="100"><span data-ancho="${porcentajeSeguro(bolsa.cobertura)}"></span></span>${porcentajeSeguro(bolsa.cobertura)}%</td>
        <td><button type="button" class="boton-terciario" data-accion="ver-bolsa" data-id="${escaparHTML(bolsa.id)}">Abrir</button></td>
      </tr>`).join("");
    return `
      ${encabezadoVista("Gestión interna de Bolsas", "Cuadro de mando", "Situación operativa, próximos llamamientos, cobertura y actividad trazada.", '<button type="button" class="boton-secundario" data-accion="imprimir">Imprimir resumen</button><button type="button" class="boton-primario" data-accion="nuevo-llamamiento" data-requiere-vista="llamamientos">Nuevo llamamiento</button>')}
      <div class="rejilla-kpi" aria-label="Indicadores de Bolsa">
        ${tarjetaKPI("BOL", numero(indicadores.bolsas_activas), "Bolsas activas", "elaboracion")}
        ${tarjetaKPI("DIS", numero(indicadores.candidatos_disponibles), "Candidatos disponibles", "elaboracion")}
        ${tarjetaKPI("LLA", numero(indicadores.llamamientos_pendientes), "Llamamientos pendientes", "llamamientos")}
        ${tarjetaKPI("CON", numero(indicadores.contratos_activos), "Contratos activos", "contratos")}
        ${tarjetaKPI("COB", `${numero(indicadores.cobertura_media, 1)} %`, "Cobertura media", "estadisticas")}
      </div>
      <div class="rejilla-cuadro-mando">
        <div class="columna-cuadro">
          <section class="panel">
            <div class="cabecera-panel"><h3>Bolsas destacadas</h3><button type="button" class="boton-terciario" data-vista="elaboracion">Ver todas</button></div>
            <div class="tabla-contenedor">
              <table class="tabla-datos">
                <caption>Bolsas destacadas con disponibilidad y cobertura</caption>
                <thead><tr><th scope="col">Bolsa</th><th scope="col">Categoría</th><th scope="col">Integrantes</th><th scope="col">En llamamiento</th><th scope="col">Cobertura</th><th scope="col">Acción</th></tr></thead>
                <tbody>${bolsasTabla}</tbody>
              </table>
            </div>
          </section>
          <section class="panel">
            <div class="cabecera-panel"><h3>Indicadores clave</h3><button type="button" class="boton-terciario" data-vista="estadisticas">Abrir informe</button></div>
            <div class="graficos-resumen">
              <article class="grafico-mini"><h4>Candidatos por estado</h4><div class="grafico-anillo" data-anillo-a="${porcentajes[0]}" data-anillo-b="${porcentajes[1]}" data-anillo-c="${porcentajes[2]}" role="img" aria-label="${numero(distribucion.disponibles)} disponibles, ${numero(distribucion.en_llamamiento)} en llamamiento, ${numero(distribucion.contratados)} contratados y ${numero(distribucion.no_disponibles)} no disponibles"></div><p class="texto-grafico">${numero(distribucion.disponibles)} disponibles · total ${numero(totalDistribucion)}</p></article>
              <article class="grafico-mini"><h4>Cobertura por bolsa</h4><div class="barras-mini" role="img" aria-label="Cobertura por bolsa">${datosPanel.bolsas.slice(0, 5).map((bolsa) => `<span data-altura="${porcentajeSeguro(bolsa.cobertura)}"></span>`).join("")}</div><p class="texto-grafico">Media de las bolsas destacadas</p></article>
              <article class="grafico-mini"><h4>Contratos por mes</h4><div class="linea-mini" role="img" aria-label="Evolución mensual de contratos">${contratosMes.map((valor) => `<span data-altura="${porcentajeSeguro(valor)}"></span>`).join("")}</div><p class="texto-grafico">${escaparHTML(datosPanel.series.periodo_contratos)}</p></article>
            </div>
          </section>
        </div>
        <div class="columna-cuadro">
          <section class="panel">
            <div class="cabecera-panel"><h3>Llamamientos próximos (7 días)</h3><button type="button" class="boton-terciario" data-vista="llamamientos">Ver todos</button></div>
            <ol class="lista-proximos">
              ${datosPanel.proximos.map((item) => `<li class="elemento-proximo"><span class="fecha-proximo"><span><strong>${escaparHTML(item.dia)}</strong>${escaparHTML(item.mes)}</span></span><span class="detalle-lista"><strong>${escaparHTML(item.bolsa)}</strong><span>Llamamiento n.º ${escaparHTML(item.numero)} · ${escaparHTML(item.fecha)}</span></span><span class="insignia">${escaparHTML(item.estado)}</span></li>`).join("")}
            </ol>
          </section>
          <section class="panel">
            <div class="cabecera-panel"><h3>Actividad reciente</h3><button type="button" class="boton-terciario" data-vista="auditoria">Ver trazabilidad</button></div>
            <ol class="lista-actividad">
              ${datosPanel.actividad.map((item) => `<li class="elemento-actividad"><span class="marca-actividad" aria-hidden="true"></span><span class="detalle-lista"><strong>${escaparHTML(item.accion)}: ${escaparHTML(item.objeto)}</strong><span>Por ${escaparHTML(item.actor)} · ${escaparHTML(item.fecha)}</span></span></li>`).join("")}
            </ol>
          </section>
        </div>
      </div>
      <section class="panel">
        <div class="cabecera-panel"><h3>Accesos rápidos</h3><span class="estado-chip info">${escaparHTML(etiquetaFuentePanel())}</span></div>
        <div class="accesos-rapidos">
          <button type="button" class="acceso-rapido" data-accion="nuevo-llamamiento" data-requiere-vista="llamamientos">Nuevo llamamiento</button>
          <button type="button" class="acceso-rapido" data-vista="elaboracion">Elaborar una bolsa</button>
          <button type="button" class="acceso-rapido" data-vista="contratos">Registrar contrato</button>
          <button type="button" class="acceso-rapido" data-vista="contratos">Registrar cese</button>
          <button type="button" class="acceso-rapido" data-vista="documentos">Generar documento</button>
          <button type="button" class="acceso-rapido" data-vista="comunicaciones">Preparar comunicación</button>
        </div>
      </section>`;
  };
}
