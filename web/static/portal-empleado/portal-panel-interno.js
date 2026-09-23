/**
 * Presentador del contrato agregado del panel interno de Bolsa y vistas B12 y B5.
 *
 * Conoce `vec.bolsa.panel.interno.v1`, `vec.bolsa.rrhh.bolsas.v1` y `vec.bolsa.rrhh.candidatos.v1`.
 * No adapta el juego sintético de presentación ni completa con valores aparentes los datos que el
 * contrato no proporciona. Recibe las utilidades visuales para mantener este módulo puro y comprobable
 * sin acceder al DOM global.
 */
import { traducirBolsaInterna } from "./portal-i18n.js";
import { renderizarBloqueAvisos } from "./portal-bolsas-avisos.js?v=20260923-pweb17-v1";
import { renderizarOperacionesSituacion } from "./portal-bolsas-operaciones.js?v=20260923-pweb14-v1";
const ESQUEMA_PANEL_INTERNO = "vec.bolsa.panel.interno.v1";
const ESTADOS_BOLSA = Object.freeze(["disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde"]);
export function crearPresentadorPanelInterno(dependencias) {
  const {
    claseEstado,
    encabezadoVista,
    escaparHTML,
    numero,
    obtenerDatosPanel,
    tituloVista,
    obtenerDatosBolsas,
    obtenerDatosCandidatosBolsa,
    obtenerDatosEstadisticas,
    obtenerDatosAvisos,
    obtenerEstadoCandidatos,
    obtenerModalContactos,
    obtenerModalFicha,
    obtenerModalLlamar,
    obtenerModalResultado,
    esLecturaPresentacion = () => false,
  } = dependencias;
  if ([claseEstado, encabezadoVista, escaparHTML, numero, obtenerDatosPanel, tituloVista]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new Error("dependencias del presentador de panel interno no válidas");
  }
  function datosPanel() {
    return obtenerDatosPanel();
  }
  function esActivo() {
    return datosPanel()?.esquema === ESQUEMA_PANEL_INTERNO;
  }
  function etiquetaFuente() {
    return esActivo() ? "Panel interno agregado autorizado" : "";
  }
  function actualizarContextoSesion(elementos) {
    if (!esActivo()) return false;
    const datos = datosPanel();
    elementos.avatar.textContent = "INT";
    elementos.nombre.textContent = "Contexto interno autorizado";
    elementos.perfil.textContent = datos.selector.clase === "unidad_gestion"
      ? "Ámbito: unidad de gestión"
      : "Ámbito: organización";
    if (elementos.avisos) {
      elementos.avisos.textContent = "—";
      elementos.avisos.setAttribute("aria-label", "Avisos no incluidos en el contrato del panel interno");
    }
    return true;
  }
  function tarjetaKPI(sigla, valor, etiqueta) {
    return `
      <article class="tarjeta-kpi">
        <span class="icono-kpi" aria-hidden="true">${escaparHTML(sigla)}</span>
        <div><strong class="valor-kpi">${escaparHTML(valor)}</strong><span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span></div>
      </article>`;
  }
  function etiquetaClave(clave) {
    const texto = String(clave || "").replaceAll(/[._-]+/g, " ").trim();
    return texto ? texto.charAt(0).toLocaleUpperCase("es-ES") + texto.slice(1) : "Sin clave";
  }
  function etiquetaEstadoBolsa(estado) {
    const clave = `bolsa_estado_${estado}`;
    const traducida = traducirBolsaInterna(clave);
    return traducida === clave ? etiquetaClave(estado) : traducida;
  }
  function controlEstadoBolsa(bolsa, estado, clase = "neutro") {
    const total = numero(bolsa.por_estado?.[estado]);
    const etiqueta = etiquetaEstadoBolsa(estado);
    return `<button type="button" class="estado-chip ${escaparHTML(clase)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}" data-estado="${escaparHTML(estado)}" aria-label="Ver ${escaparHTML(total)} candidatos ${escaparHTML(etiqueta.toLocaleLowerCase("es-ES"))} de ${escaparHTML(bolsa.categoria)}">${escaparHTML(total)}</button>`;
  }
  function claseEstadoEstadistica(estado) {
    return ({ disponible: "exito", no_disponible: "peligro", excluido: "peligro", pendiente_incorporacion: "info", disponible_desde: "info" })[estado] || "neutro";
  }
  function etiquetaEstadoEstadistica(estado) { return etiquetaEstadoBolsa(estado); }
  function renderizarEstadisticasBolsa() {
    const estadoEstadisticas = typeof obtenerDatosEstadisticas === "function" ? obtenerDatosEstadisticas() : null;
    const cabecera = encabezadoVista("Gestión interna de Bolsas", "Estadísticas", "Indicadores agregados del ámbito autorizado. Las cifras por bolsa abren la lista B5 con el filtro correspondiente.", '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro</button>');
    if (!estadoEstadisticas || estadoEstadisticas.carga === "cargando") return `${cabecera}<section class="panel"><div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true"><p><strong>Cargando estadísticas de Bolsa…</strong></p><p>Consultando el contrato agregado autorizado.</p></div></section>`;
    if (estadoEstadisticas.carga === "denegado" || estadoEstadisticas.carga === "error") return `${cabecera}<section class="panel"><div class="cuerpo-panel vacio-controlado" role="alert"><p><strong>${estadoEstadisticas.carga === "denegado" ? "Acceso denegado" : "No se pudieron cargar las estadísticas"}</strong></p><p>${escaparHTML(estadoEstadisticas.error || "La sesión no puede consultar esta información.")}</p>${estadoEstadisticas.carga === "error" ? '<button type="button" class="boton-secundario" data-bolsa-accion="reintentar-estadisticas">Reintentar</button>' : ""}</div></section>`;
    const datos = estadoEstadisticas.datos;
    const tarjetas = [["BOL", datos.bolsas.total, "Bolsas"], ["VIG", datos.bolsas.vigentes, "Vigentes"], ["SUS", datos.bolsas.sustituidas, "Sustituidas"], ["PER", datos.personas.total, "Personas"], ["LLA", datos.llamamientos.total, "Llamamientos"]];
    const resumenEstados = Object.entries(datos.personas.por_estado).map(([estado, total]) => `<span class="estado-chip ${claseEstadoEstadistica(estado)}"><strong>${numero(total)}</strong> ${escaparHTML(etiquetaEstadoEstadistica(estado))}</span>`).join("");
    const resumenCanales = Object.entries(datos.llamamientos.por_canal).map(([canal, total]) => `<span class="estado-chip info"><strong>${numero(total)}</strong> ${escaparHTML(etiquetaClave(canal))}</span>`).join("") || '<span class="estado-chip neutro">Sin desglose por canal</span>';
    const resumenResultados = Object.entries(datos.llamamientos.por_resultado).map(([resultado, total]) => `<span class="estado-chip neutro"><strong>${numero(total)}</strong> ${escaparHTML(etiquetaClave(resultado))}</span>`).join("") || '<span class="estado-chip neutro">Sin desglose por resultado</span>';
    const filas = datos.por_bolsa.map((bolsa) => `<tr><td><button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}"><strong>${escaparHTML(bolsa.categoria)}</strong></button><br><small>${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</small></td><td><span class="estado-chip ${bolsa.vigente ? "exito" : "neutro"}">${bolsa.vigente ? "Vigente" : "Sustituida"}</span></td><td><button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}" aria-label="Abrir ${numero(bolsa.total)} personas en B5">${numero(bolsa.total)}</button></td>${ESTADOS_BOLSA.map((estado) => `<td><button type="button" class="estado-chip ${claseEstadoEstadistica(estado)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}" data-estado="${escaparHTML(estado)}" aria-label="Abrir ${numero(bolsa.por_estado[estado])} personas ${escaparHTML(etiquetaEstadoEstadistica(estado).toLocaleLowerCase("es-ES"))} en B5">${numero(bolsa.por_estado[estado])}</button></td>`).join("")}</tr>`).join("");
    return `${cabecera}<div class="rejilla-kpi" aria-label="Totales de Bolsa">${tarjetas.map(([sigla, valor, etiqueta]) => tarjetaKPI(sigla, numero(valor), etiqueta)).join("")}</div><section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Personas y llamamientos</h3><p>Desglose agregado recibido sin completar datos ausentes.</p></div><time datetime="${escaparHTML(datos.generado_en)}">Actualizado ${escaparHTML(instanteVisible(datos.generado_en))}</time></div><div class="cuerpo-panel estadisticas-resumen"><div><h4>Situación de personas</h4><div class="lista-chips">${resumenEstados}</div></div><div><h4>Canal de llamamiento</h4><div class="lista-chips">${resumenCanales}</div></div><div><h4>Resultado de llamamiento</h4><div class="lista-chips">${resumenResultados}</div></div></div></section><section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Desglose por bolsa</h3><p>Seleccione una cifra para abrir B5 con esa bolsa y situación.</p></div><span class="estado-chip info">${numero(datos.por_bolsa.length)} bolsas</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Personas por bolsa y situación</caption><thead><tr><th scope="col">Bolsa</th><th scope="col">Vigencia</th><th scope="col">Total</th>${ESTADOS_BOLSA.map((estado) => `<th scope="col">${escaparHTML(etiquetaEstadoEstadistica(estado))}</th>`).join("")}</tr></thead><tbody>${filas || '<tr><td colspan="10" class="vacio-controlado">La fuente no ha devuelto bolsas para este ámbito.</td></tr>'}</tbody></table></div></section>`;
  }
  function instanteVisible(instante) {
    if (!instante || String(instante).startsWith("0001-01-01")) return "Sin fecha límite";
    const fecha = new Date(instante);
    if (!Number.isFinite(fecha.getTime())) return "Fecha no disponible";
    return new Intl.DateTimeFormat("es-ES", {
      dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid",
    }).format(fecha);
  }
  function fechaCivilVisible(valor) {
    const partes = typeof valor === "string" && /^(\d{4})-(\d{2})-(\d{2})$/u.exec(valor);
    if (!partes) return "Fecha no disponible";
    const [, anioTexto, mesTexto, diaTexto] = partes;
    const anio = Number(anioTexto);
    const mes = Number(mesTexto);
    const dia = Number(diaTexto);
    if (anio < 1) return "Fecha no disponible";
    const fecha = new Date(0);
    fecha.setUTCFullYear(anio, mes - 1, dia);
    fecha.setUTCHours(0, 0, 0, 0);
    if (fecha.getUTCFullYear() !== anio || fecha.getUTCMonth() !== mes - 1 || fecha.getUTCDate() !== dia) {
      return "Fecha no disponible";
    }
    return new Intl.DateTimeFormat("es-ES", { dateStyle: "short", timeZone: "UTC" }).format(fecha);
  }
  function fechaVisible(valor) {
    return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(valor)
      ? fechaCivilVisible(valor)
      : instanteVisible(valor);
  }
  function fechaAvisoSustitucion(valor) {
    const partes = typeof valor === "string" && /^(\d{4})-(\d{2})-(\d{2})$/u.exec(valor);
    if (partes) {
      const [, anio, mes, dia] = partes;
      return `${dia}/${mes}/${anio}`;
    }
    return fechaVisible(valor);
  }
  function fechaMarcada(valor) {
    const texto = fechaVisible(valor);
    return texto === "Fecha no disponible"
      ? escaparHTML(texto)
      : `<time datetime="${escaparHTML(valor)}">${escaparHTML(texto)}</time>`;
  }
  function filasConvocatorias(datos) {
    if (datos.convocatorias.length === 0) {
      return '<tr><td colspan="6" class="vacio-controlado">La fuente autorizada no ha devuelto convocatorias para este ámbito.</td></tr>';
    }
    return datos.convocatorias.map((item) => `
      <tr>
        <td><strong>${escaparHTML(item.convocatoria_ref)}</strong></td>
        <td>${escaparHTML(etiquetaClave(item.categoria_clave))}<br><small>${escaparHTML(item.categoria_clave)}</small></td>
        <td><span class="estado-chip ${claseEstado(item.estado_clave)}">${escaparHTML(etiquetaClave(item.estado_clave))}</span></td>
        <td>${item.plazo_cierra_en ? `<time datetime="${escaparHTML(item.plazo_cierra_en)}">${escaparHTML(instanteVisible(item.plazo_cierra_en))}</time>` : "Sin fecha límite"}</td>
        <td>${numero(item.numero_solicitudes)}</td><td>${numero(item.numero_pendientes)}</td>
      </tr>`).join("");
  }
  function filasActuaciones(datos) {
    if (datos.actuaciones_pendientes.length === 0) {
      return '<tr><td colspan="7" class="vacio-controlado">La fuente autorizada no ha devuelto actuaciones pendientes para este ámbito.</td></tr>';
    }
    return datos.actuaciones_pendientes.map((item) => `
      <tr>
        <td><strong>${escaparHTML(item.actuacion_ref)}</strong></td><td>${escaparHTML(item.recurso_ref)}</td>
        <td>${escaparHTML(etiquetaClave(item.tipo_clave))}<br><small>${escaparHTML(item.tipo_clave)}</small></td>
        <td><span class="estado-chip ${claseEstado(item.estado_clave)}">${escaparHTML(etiquetaClave(item.estado_clave))}</span></td>
        <td><span class="estado-chip ${claseEstado(item.prioridad_clave)}">${escaparHTML(etiquetaClave(item.prioridad_clave))}</span></td>
        <td>${item.fecha_limite ? `<time datetime="${escaparHTML(item.fecha_limite)}">${escaparHTML(instanteVisible(item.fecha_limite))}</time>` : "Sin fecha límite"}</td>
        <td>${numero(item.numero_elementos)}</td>
      </tr>`).join("");
  }
  function renderizarCuadroB12() {
    const estadoBolsas = typeof obtenerDatosBolsas === "function" ? obtenerDatosBolsas() : null;
    if (!estadoBolsas) return "";
    if (estadoBolsas.carga === "cargando") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip neutro">Consultando…</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true">
            <p><strong>Cargando bolsas de trabajo…</strong></p>
            <p>Consultando la disponibilidad y el desglose de aspirantes en el ámbito autorizado.</p>
          </div>
        </section>`;
    }
    if (estadoBolsas.carga === "error") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip peligro">Error de carga</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>No se pudieron cargar las bolsas de trabajo</strong></p>
            <p>${escaparHTML(estadoBolsas.error || "Se ha producido un error al consultar las bolsas.")}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">Reintentar</button>
            </div>
          </div>
        </section>`;
    }
    if (estadoBolsas.carga === "denegado") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip peligro">Acceso denegado</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>Acceso denegado a la consulta de bolsas</strong></p>
            <p>La sesión actual no dispone de permisos suficientes para consultar el cuadro de bolsas de trabajo.</p>
          </div>
        </section>`;
    }
    const bolsas = estadoBolsas.datos?.bolsas || [];
    if (bolsas.length === 0) {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip neutro">0 registros</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="status">
            <p><strong>No hay bolsas de trabajo activas</strong></p>
            <p>El servicio no ha devuelto bolsas de trabajo registradas para este ámbito.</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">Reintentar</button>
            </div>
          </div>
        </section>`;
    }
    const totalAspirantes = bolsas.reduce((total, bolsa) => total + Number(bolsa.total || 0), 0);
    const totalDisponibles = bolsas.reduce((total, bolsa) => total + Number(bolsa.por_estado?.disponible || 0), 0);
    const totalPersonasEnRenuncia = bolsas.reduce((total, bolsa) => total + Number(bolsa.por_estado?.renuncia || 0), 0);
    const filas = bolsas.map((b) => `
      <tr data-bolsa-ref="${escaparHTML(b.bolsa_ref)}">
        <td><button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(b.bolsa_ref)}" aria-label="Abrir candidatos de la bolsa ${escaparHTML(b.categoria)}"><strong>${escaparHTML(b.categoria)}</strong></button><br><small>${escaparHTML(b.categoria_clave)}</small></td>
        <td><span class="estado-chip neutro">${escaparHTML(etiquetaClave(b.tipo_lista))}</span></td>
        <td><small>${fechaMarcada(b.vigente_desde)}${b.vigente_hasta ? ` — ${fechaMarcada(b.vigente_hasta)}` : " (vigente)"}</small></td>
        <td><button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(b.bolsa_ref)}" aria-label="Ver los ${numero(b.total)} candidatos de ${escaparHTML(b.categoria)}"><strong>${numero(b.total)}</strong></button></td>
        <td><span class="estado-chip info">${numero(b.llamamientos_en_curso)}</span></td>
        <td>${controlEstadoBolsa(b, "disponible", "exito")}</td>
        <td>${controlEstadoBolsa(b, "trabajando")}</td>
        <td>${controlEstadoBolsa(b, "no_disponible", "peligro")}</td>
        <td>${controlEstadoBolsa(b, "excluido", "peligro")}</td>
        <td>${controlEstadoBolsa(b, "renuncia")}</td>
        <td>${controlEstadoBolsa(b, "pendiente_incorporacion", "info")}</td>
      </tr>
    `).join("");
    return `
      <section class="panel" aria-labelledby="titulo-cuadro-b12">
        <div class="cabecera-panel">
          <h3 id="titulo-cuadro-b12">Bolsas de trabajo activas (Cuadro B12)</h3>
          <span class="estado-chip info">${numero(bolsas.length)} bolsas</span>
        </div>
        <div class="rejilla-kpi cuadro-b12-kpi" aria-label="Resumen del Cuadro B12">
          <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">▣</span><div><span class="etiqueta-kpi">Bolsas visibles</span><strong class="valor-kpi">${numero(bolsas.length)}</strong></div></article>
          <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">♙</span><div><span class="etiqueta-kpi">Aspirantes</span><strong class="valor-kpi">${numero(totalAspirantes)}</strong></div></article>
          <article class="tarjeta-kpi kpi--exito"><span class="icono-kpi" aria-hidden="true">✓</span><div><span class="etiqueta-kpi">Disponibles</span><strong class="valor-kpi">${numero(totalDisponibles)}</strong></div></article>
          <article class="tarjeta-kpi kpi--advertencia"><span class="icono-kpi" aria-hidden="true">!</span><div><span class="etiqueta-kpi">Personas en renuncia</span><strong class="valor-kpi">${numero(totalPersonasEnRenuncia)}</strong></div></article>
        </div>
        <div class="tabla-contenedor">
          <table class="tabla-datos">
            <caption>Cuadro B12: Bolsas de trabajo y distribución de aspirantes por situación</caption>
            <thead>
              <tr>
                <th scope="col">Bolsa / Categoría</th>
                <th scope="col">Tipo de lista</th>
                <th scope="col">Vigencia</th>
                <th scope="col">Total</th>
                <th scope="col">Llamamientos en curso</th>
                <th scope="col">Disponibles</th>
                <th scope="col">Ocupados / Trabajando</th>
                <th scope="col">No disp.</th>
                <th scope="col">Excluidos</th>
                <th scope="col">Renuncia</th>
                <th scope="col">Pend. incorporación</th>
              </tr>
            </thead>
            <tbody>
              ${filas}
            </tbody>
          </table>
        </div>
      </section>`;
  }
  function renderizarResumen(datos) {
    const i = datos.indicadores;
    const indicadoresConectados = [
      ["BOL", i.bolsas_activas, "Bolsas activas"],
      ["LLA", i.llamamientos_pendientes, "Llamamientos pendientes"],
      ["CUR", i.llamamientos_en_curso, "Llamamientos en curso"],
      ["DOC", i.documentos_pendientes_firma, "Documentos pendientes de firma"],
      ["INC", i.incidencias_abiertas, "Incidencias abiertas"],
    ];
    return `
      ${encabezadoVista("Gestión interna de Bolsas", "Cuadro de mando", "Datos conectados, agregados y autorizados del ámbito interno. El contrato no contiene datos personales ni habilita acciones administrativas.", '<button type="button" class="boton-secundario" data-accion="imprimir">Imprimir resumen</button>')}
      <section class="nota-seguridad" aria-label="Alcance de los datos conectados">Datos conectados en modo de solo lectura. Los indicadores, convocatorias y actuaciones proceden del contrato <code>${ESQUEMA_PANEL_INTERNO}</code>; esta vista no es una presentación ni completa datos ausentes.</section>
      <div class="rejilla-kpi" aria-label="Indicadores operativos de Bolsa">
        ${indicadoresConectados.map(([sigla, valor, etiqueta]) => tarjetaKPI(sigla, numero(valor), etiqueta)).join("")}
      </div>
      <div class="rejilla-cuadro-mando" aria-label="Resumen operativo conectado">
        <div class="columna-cuadro">
          ${renderizarCuadroB12()}
          <section class="panel"><div class="cabecera-panel"><h3>Convocatorias del ámbito autorizado</h3><span class="estado-chip info">${numero(datos.convocatorias.length)} registros</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Convocatorias agregadas devueltas por el panel interno</caption><thead><tr><th scope="col">Referencia</th><th scope="col">Categoría</th><th scope="col">Estado</th><th scope="col">Cierre de plazo</th><th scope="col">Solicitudes</th><th scope="col">Pendientes</th></tr></thead><tbody>${filasConvocatorias(datos)}</tbody></table></div></section>
        </div>
        <aside class="columna-cuadro" aria-label="Actuaciones y prueba de lectura">
          <section class="panel"><div class="cabecera-panel"><h3>Actuaciones pendientes</h3><span class="estado-chip info">${numero(datos.actuaciones_pendientes.length)} registros</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Trabajo administrativo pendiente sin identidad de personas interesadas</caption><thead><tr><th scope="col">Actuación</th><th scope="col">Recurso</th><th scope="col">Tipo</th><th scope="col">Estado</th><th scope="col">Prioridad</th><th scope="col">Fecha límite</th><th scope="col">Elementos</th></tr></thead><tbody>${filasActuaciones(datos)}</tbody></table></div></section>
          <section class="panel"><div class="cabecera-panel"><h3>Prueba de lectura</h3><span class="estado-chip exito">Lectura auditada</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Ámbito</dt><dd>${escaparHTML(etiquetaClave(datos.selector.clase))}</dd></div><div class="fila-resumen"><dt>Revisión de fuente</dt><dd>${escaparHTML(datos.origen.revision)}</dd></div><div class="fila-resumen"><dt>Actualizada</dt><dd><time datetime="${escaparHTML(datos.origen.actualizada_en)}">${escaparHTML(instanteVisible(datos.origen.actualizada_en))}</time></dd></div><div class="fila-resumen"><dt>Lectura</dt><dd>${escaparHTML(datos.prueba_lectura.lectura_ref)}</dd></div><div class="fila-resumen"><dt>Auditoría</dt><dd>${escaparHTML(datos.prueba_lectura.auditoria_ref)} · secuencia ${numero(datos.prueba_lectura.auditoria_secuencia)}</dd></div><div class="fila-resumen"><dt>Confirmada</dt><dd><time datetime="${escaparHTML(datos.prueba_lectura.confirmada_en)}">${escaparHTML(instanteVisible(datos.prueba_lectura.confirmada_en))}</time></dd></div></dl></div></section>
        </aside>
      </div>`;
  }
  function renderizarConvocatorias(datos) {
    const i = datos.indicadores;
    return `
      ${encabezadoVista("Proyección interna autorizada", "Convocatorias", "Consulta de convocatorias agregadas. La edición, las bases y la publicación todavía no están conectadas a esta superficie web.", '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro de mando</button>')}
      <section class="nota-pendiente">Modo de solo lectura: el contrato real no incluye expedientes de elaboración ni concede capacidad para modificarlos.</section>
      <div class="rejilla-kpi">${tarjetaKPI("BOR", numero(i.convocatorias_borrador), "Borrador")}${tarjetaKPI("REV", numero(i.convocatorias_revision), "En revisión")}${tarjetaKPI("FIR", numero(i.convocatorias_pendientes_firma), "Pendientes de firma")}${tarjetaKPI("PUB", numero(i.convocatorias_publicadas), "Publicadas")}</div>
      <section class="panel"><div class="cabecera-panel"><h3>Convocatorias del ámbito autorizado</h3><span class="estado-chip info">${numero(datos.convocatorias.length)} registros</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Convocatorias agregadas en modo de solo lectura</caption><thead><tr><th scope="col">Referencia</th><th scope="col">Categoría</th><th scope="col">Estado</th><th scope="col">Cierre de plazo</th><th scope="col">Solicitudes</th><th scope="col">Pendientes</th></tr></thead><tbody>${filasConvocatorias(datos)}</tbody></table></div></section>`;
  }
  function renderizarNuevoLlamamiento(bolsa, candidatos, flujo) {
    const paso = Math.max(1, Math.min(4, Number(flujo.paso) || 1));
    const nombres = ["Seleccionar bolsa", "Seleccionar candidatos", "Configurar llamamiento", "Revisar y enviar"];
    const rail = `<nav class="pasos" aria-label="Pasos del nuevo llamamiento">${nombres.map((nombre,i)=>`<span class="paso ${i+1<paso?"completado":""}"${i+1===paso?' aria-current="step" tabindex="-1"':""}><span class="paso-numero">${i+1<paso?"✓":i+1}</span><span>${nombre}</span></span>`).join("")}</nav>`;
    const resumen = `<aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Resumen</h3></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Categoría</dt><dd>${escaparHTML(bolsa.categoria)}</dd></div><div class="fila-resumen"><dt>Tipo de lista</dt><dd>${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</dd></div><div class="fila-resumen"><dt>Vigencia</dt><dd>${escaparHTML(fechaVisible(bolsa.vigente_desde))}</dd></div><div class="fila-resumen"><dt>Personas</dt><dd>${numero(bolsa.total)}</dd></div><div class="fila-resumen"><dt>Seleccionadas</dt><dd>${numero(flujo.participaciones?.length||0)}</dd></div></dl></div></section></aside>`;
    let contenido="";
    if(paso===1) contenido=`<form class="panel" data-bolsa-form="b7-paso1"><div class="cabecera-panel"><h3>1. Seleccionar bolsa</h3><span class="estado-chip exito">Bolsa constituida</span></div><div class="cuerpo-panel"><p><strong>${escaparHTML(bolsa.categoria)}</strong></p><p>La bolsa se ha leído de B12 y contiene ${numero(bolsa.total)} personas, ${numero(bolsa.por_estado?.disponible||0)} disponibles.</p><button class="boton-primario" type="submit">Seleccionar esta bolsa</button></div></form>`;
    if (paso === 2) {
      const estados = new Set(flujo.estados || ["disponible"]);
      const visibles = candidatos.filter((candidato) => estados.has(candidato.estado_clave) && Number.isSafeInteger(candidato.orden));
      const pagina = Math.max(0, Math.min(Number(flujo.pagina) || 0, Math.max(0, Math.ceil(visibles.length / 6) - 1)));
      const inicio = pagina * 6;
      const seleccionadas = flujo.participaciones?.length || 0;
      const limiteAlcanzado = Number(flujo.totalElegibles) > 100;
      const filas = visibles.slice(inicio, inicio + 6).map((candidato) => `<tr><td><input type="checkbox" name="participacion" value="${escaparHTML(candidato.participacion_ref)}" ${flujo.participaciones?.includes(candidato.participacion_ref) ? "checked" : ""} aria-label="Seleccionar orden ${numero(candidato.orden)}"></td><td>${numero(candidato.orden)}</td><td>${escaparHTML(candidato.nombre_visible)}</td><td><span class="estado-chip ${claseEstado(candidato.estado_clave)}">${escaparHTML(etiquetaClave(candidato.estado_clave))}</span></td></tr>`).join("");
      contenido = `<form class="panel" data-bolsa-form="b7-paso2" aria-busy="${flujo.consultando ? "true" : "false"}"><div class="cabecera-panel"><h3>2. Seleccionar candidatos</h3><span class="estado-chip info">${numero(visibles.length)} ocupan turno en esta página</span></div><div class="cuerpo-panel"><fieldset><legend>Estados a incluir</legend>${["disponible", "disponible_desde"].map((estado) => `<label><input type="checkbox" name="estado" value="${estado}" ${estados.has(estado) ? "checked" : ""}> ${escaparHTML(etiquetaClave(estado))}</label>`).join(" ")}</fieldset><label><input type="checkbox" checked disabled> Respetar orden de prelación (obligatorio)</label><p><small>Las personas trabajando o en pausa no ocupan turno.</small></p><button type="button" class="boton-secundario" data-bolsa-accion="b7-seleccionar-todas" ${flujo.consultando ? "disabled" : ""}>Seleccionar todas las que cumplen el filtro</button><details><summary aria-label="Ayuda sobre el límite de selección">?</summary><p>Se consulta la bolsa completa con los estados elegidos aquí y se seleccionan por orden B6 hasta 100 candidaturas por envío.</p></details><p role="status" data-b7-seleccion-status><strong>${numero(seleccionadas)} seleccionadas.</strong> ${flujo.consultando ? "Consultando todas las páginas…" : limiteAlcanzado ? `de ${numero(flujo.totalElegibles)} candidaturas que cumplen el filtro.` : flujo.totalElegibles !== null && flujo.totalElegibles !== undefined ? `de ${numero(flujo.totalElegibles)} candidaturas que cumplen el filtro.` : ""}</p>${flujo.error ? `<p role="alert" class="mensaje-error">${escaparHTML(flujo.error)}</p>` : ""}</div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Candidatos por el orden vigente de B6</caption><thead><tr><th>Selección</th><th>Nº orden</th><th>Candidato</th><th>Estado</th></tr></thead><tbody>${filas || '<tr><td colspan="4">No hay candidatos con turno vigente en esos estados.</td></tr>'}</tbody></table></div><div class="cuerpo-panel"><span>Mostrando ${visibles.length ? inicio + 1 : 0} a ${Math.min(inicio + 6, visibles.length)} de ${numero(visibles.length)} en esta página</span> <button type="button" class="boton-secundario" data-bolsa-accion="b7-pagina" data-pagina="${pagina - 1}" ${pagina === 0 ? "disabled" : ""}>Anterior</button> <button type="button" class="boton-secundario" data-bolsa-accion="b7-pagina" data-pagina="${pagina + 1}" ${inicio + 6 >= visibles.length ? "disabled" : ""}>Siguiente</button> <button class="boton-primario" type="submit" ${flujo.consultando ? "disabled" : ""}>Configurar llamamiento</button></div></form>`;
    }
    if(paso===3) contenido=`<form class="panel" data-bolsa-form="b7-paso3"><div class="cabecera-panel"><h3>3. Configurar llamamiento</h3></div><div class="cuerpo-panel rejilla-formulario"><p class="nota-seguridad campo-ancho" role="note"><strong>El asunto y el texto son comunes a todas las personas seleccionadas.</strong> La personalización por persona está pendiente de desarrollo.</p><label>Referencia de necesidad<input name="referencia" required minlength="2" maxlength="160" value="${escaparHTML(flujo.configuracion?.referencia||"")}"></label><label>Categoría<input name="categoria" required value="${escaparHTML(bolsa.categoria)}"></label><label>Centro<input name="centro" required minlength="2" maxlength="200"></label><label>Modalidad<select name="modalidad" required><option>Sustitución</option><option>Vacante</option><option>Programa temporal</option><option>Acumulación de tareas</option></select></label><label>Fecha prevista de inicio<input type="date" name="fecha_inicio" required></label><label>Canal<input value="Correo obligatorio" readonly></label><label>Plazo provisional<input name="plazo" required value="48 horas · pendiente de RRHH, dudas 1–3" readonly></label><label class="campo-ancho">Descripción<textarea name="descripcion" required minlength="2" maxlength="1000"></textarea></label><label>Plantilla<input name="plantilla_version" value="bolsa-llamamiento-v1" readonly></label><label>Asunto<input name="asunto" required value="Llamamiento de bolsa: ${escaparHTML(bolsa.categoria)}"></label><label class="campo-ancho">Texto del correo<textarea name="cuerpo" required>Se comunica una necesidad de cobertura. La referencia, categoría, centro, modalidad y fecha prevista figuran en este llamamiento. Responda por el cauce indicado por RRHH.</textarea></label><p class="nota-integracion campo-ancho"><strong>Pendiente de integración con correo corporativo.</strong> El recorrido usa el relay de desarrollo configurado; no acredita entrega en el buzón corporativo.</p><button type="submit" class="boton-primario">Revisar llamamiento</button></div></form>`;
    if(paso===4){const c=flujo.configuracion||{};contenido=`<section class="panel"><div class="cabecera-panel"><h3>4. Revisar y enviar</h3><span class="estado-chip advertencia">Pendiente de confirmación</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Necesidad</dt><dd>${escaparHTML(c.referencia)}</dd></div><div class="fila-resumen"><dt>Centro / modalidad</dt><dd>${escaparHTML(c.centro)} · ${escaparHTML(c.modalidad)}</dd></div><div class="fila-resumen"><dt>Candidatos</dt><dd>${numero(flujo.participaciones?.length||0)} seleccionados${flujo.totalElegibles > (flujo.participaciones?.length||0) ? ` de ${numero(flujo.totalElegibles)} elegibles` : ""}, en orden de prelación</dd></div><div class="fila-resumen"><dt>Canal</dt><dd>Correo SMTP; teléfono queda como intento manual B3</dd></div><div class="fila-resumen"><dt>Plazo</dt><dd>${escaparHTML(c.plazo)}</dd></div></dl><form data-bolsa-form="b7-paso4" data-cantidad="${numero(flujo.participaciones?.length||0)}"><label><input type="checkbox" name="confirmacion" required> Confirmo la emisión para exactamente ${numero(flujo.participaciones?.length||0)} candidatos y el envío al relay configurado.</label><button type="submit" class="boton-primario" ${flujo.enviando||flujo.recibo?"disabled":""}>${flujo.enviando?"Enviando…":flujo.recibo?"Emitido":"Emitir llamamiento"}</button></form>${flujo.error?`<p role="alert" class="mensaje-error">${escaparHTML(flujo.error)}</p>`:""}${flujo.recibo?`<section class="mensaje-exito" tabindex="-1" data-b7-recibo><strong>Llamamiento ${escaparHTML(flujo.llamamiento_ref)}</strong><br>Estado: emitido, pendiente de respuesta.<br>Recibo <code>${escaparHTML(flujo.recibo)}</code> · <button type="button" class="boton-secundario" data-bolsa-accion="ver-historico-b7">Abrir Histórico</button></section>`:""}</div></section>`}
    return `${encabezadoVista("Gestión interna de Bolsas","Nuevo llamamiento","Recorrido real B7 sobre bolsa constituida, candidatos B5, PostgreSQL y SMTP.",'<button type="button" class="boton-secundario" data-bolsa-accion="cancelar-b7">Cancelar</button>')}${rail}<div class="distribucion-llamamiento"><div>${contenido}</div>${resumen}</div>`;
  }
  function renderizarCandidatosBolsa() {
    const lecturaPresentacion = esLecturaPresentacion() === true;
    const estadoCandidatos = typeof obtenerDatosCandidatosBolsa === "function"
      ? obtenerDatosCandidatosBolsa()
      : null;
    const filtrosActuales = typeof obtenerEstadoCandidatos === "function"
      ? obtenerEstadoCandidatos()
      : { estado: "", texto: "" };
    const pestana = filtrosActuales.pestana === "historico" ? "historico" : "candidatos";
    const accionesEncabezado = '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro</button>';
    if (!estadoCandidatos || estadoCandidatos.carga === "cargando") {
      return `
        ${encabezadoVista("Gestión interna de Bolsas", "Candidatos de la bolsa", "Consulta ordenada de aspirantes y situación de disponibilidad.", accionesEncabezado)}
        <section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
          <div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true">
            <p><strong>Cargando lista de candidatos…</strong></p>
            <p>Obteniendo las candidaturas y llamamientos de la bolsa seleccionada.</p>
          </div>
        </section>`;
    }
    if (estadoCandidatos.carga === "error") {
      return `
        ${encabezadoVista("Gestión interna de Bolsas", "Candidatos de la bolsa", "Consulta ordenada de aspirantes y situación de disponibilidad.", accionesEncabezado)}
        <section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>Error al consultar candidatos</strong></p>
            <p>${escaparHTML(estadoCandidatos.error || "No se pudo cargar la relación de aspirantes.")}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-candidatos">Reintentar</button>
              <button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro</button>
            </div>
          </div>
        </section>`;
    }
    if (estadoCandidatos.carga === "denegado") {
      return `
        ${encabezadoVista("Gestión interna de Bolsas", "Candidatos de la bolsa", "Consulta ordenada de aspirantes y situación de disponibilidad.", accionesEncabezado)}
        <section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>Acceso denegado</strong></p>
            <p>La sesión no dispone de permisos para consultar los candidatos de esta bolsa.</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro</button>
            </div>
          </div>
        </section>`;
    }
    const bolsa = estadoCandidatos.datos?.bolsa;
    const candidatos = estadoCandidatos.datos?.candidatos || [];
    const contactos = estadoCandidatos.datos?.contactos || [];
    const modalFicha = typeof obtenerModalFicha === "function" ? obtenerModalFicha() : null;
    const reciboFichaFuera = modalFicha?.operacionesB8?.recibo && !candidatos.some((item) => item.participacion_ref === modalFicha.candidato?.participacion_ref)
      ? `<p class="mensaje-exito" role="status">Operación registrada. Recibo <code>${escaparHTML(modalFicha.operacionesB8.recibo)}</code>. La participación ya no coincide con el filtro o la página actual; consulte su nueva situación al quitar el filtro.</p>` : "";
    const hayMas = estadoCandidatos.datos?.hay_mas === true;
    const cursorSiguiente = estadoCandidatos.datos?.cursor_siguiente || "";
    if (filtrosActuales.nuevo_llamamiento) return renderizarNuevoLlamamiento(bolsa, candidatos, filtrosActuales.nuevo_llamamiento);
    const tituloBolsa = bolsa ? `Candidatos: ${bolsa.categoria}` : "Candidatos de la bolsa";
    const descripcionBolsa = bolsa
      ? `${bolsa.categoria_clave} · Lista ${bolsa.tipo_lista} · Total: ${numero(bolsa.total)} aspirantes`
      : "Consulta ordenada de aspirantes y situación de disponibilidad.";
    const opcionesEstado = [
      ["", "Todos los estados"],
      ["disponible", "Disponible"],
      ["no_disponible", "No disponible"],
      ["trabajando", "Trabajando"],
      ["pendiente_incorporacion", "Pendiente de incorporación"],
      ["renuncia", "Renuncia"],
      ["excluido", "Excluido"],
      ["disponible_desde", "Disponible desde fecha"],
    ].map(([valor, etiqueta]) => `
      <option value="${escaparHTML(valor)}"${valor === filtrosActuales.estado ? " selected" : ""}>${escaparHTML(etiqueta)}</option>
    `).join("");
    const contadoresEstado = Object.entries(bolsa?.por_estado || {}).map(([estado, total]) => {
      const tono = estado === "disponible" ? "kpi--exito" : ["renuncia", "no_disponible"].includes(estado) ? "kpi--advertencia" : estado === "excluido" ? "kpi--peligro" : "";
      const rotulo = ({ no_disponible: "No disp.", pendiente_incorporacion: "Pend. incorp.", disponible_desde: "Desde fecha" })[estado] || etiquetaClave(estado);
      return `<button type="button" class="tarjeta-kpi kpi-filtro ${tono}" data-bolsa-accion="filtrar-estado" data-estado="${escaparHTML(estado)}" aria-label="Filtrar ${numero(total)} candidatos en situación ${escaparHTML(etiquetaEstadoBolsa(estado))}" aria-pressed="${filtrosActuales.estado === estado}"><span class="icono-kpi" aria-hidden="true">${estado === "disponible" ? "✓" : estado === "excluido" ? "×" : "•"}</span><span><span class="etiqueta-kpi">${escaparHTML(rotulo)}</span><strong class="valor-kpi">${numero(total)}</strong></span></button>`;
    }).join("");
    const formularioFiltros = `
      <form class="barra-filtros-bolsa barra-filtros-estadisticas" data-bolsa-form="filtros" role="search" aria-label="Filtros de candidatos">
        <div class="campo-filtro">
          <label for="filtro-bolsa-estado">Situación:</label>
          <select id="filtro-bolsa-estado" name="estado">${opcionesEstado}</select>
        </div>
        <div class="campo-filtro">
          <label for="filtro-bolsa-texto">Buscar:</label>
          <input type="search" id="filtro-bolsa-texto" name="texto" value="${escaparHTML(filtrosActuales.texto || "")}" placeholder="Nombre o documento…">
        </div>
        <div class="acciones-filtro">
          <button type="submit" class="boton-primario">Filtrar</button>
          <button type="button" class="boton-secundario" data-bolsa-accion="limpiar-filtros">Limpiar</button>
        </div>
      </form>`;
    let cuerpoTabla = "";
    if (candidatos.length === 0) {
      cuerpoTabla = `<tr><td colspan="7" class="vacio-controlado">No se han encontrado aspirantes que coincidan con los criterios seleccionados.</td></tr>`;
    } else {
      cuerpoTabla = candidatos.map((c) => {
        let detalleLlamamiento = '<small class="texto-atenuado">Sin llamamientos</small>';
        if (c.ultimo_llamamiento) {
          const l = c.ultimo_llamamiento;
          detalleLlamamiento = `<span>${escaparHTML(etiquetaClave(l.canal))} · ${escaparHTML(etiquetaClave(l.resultado))}<br><small><time datetime="${escaparHTML(l.comunicado_en)}">${escaparHTML(instanteVisible(l.comunicado_en))}</time></small></span>`;
        }
        const fichaAbierta = modalFicha?.abierto === true
          && modalFicha.candidato?.participacion_ref === c.participacion_ref;
        const fichaId = `ficha-participacion-${c.participacion_ref}`;
        return `
          <tr class="fila-candidato" data-participacion-ref="${escaparHTML(c.participacion_ref)}" data-estado="${escaparHTML(c.estado_clave)}">
            <td><strong>${c.orden === null ? "—" : `#${numero(c.orden)}`}</strong>${c.razon_orden !== "orden_acta" ? `<br><small>${escaparHTML(c.razon_orden === "reposicion_tras_contrato" ? "Reposición tras contrato" : c.razon_orden === "pausa" ? "Pausa" : etiquetaClave(c.razon_orden))}</small>` : ""}</td>
            <td><button type="button" class="enlace-tabla" data-bolsa-accion="abrir-ficha" data-bolsa-control-principal="true" data-participacion-ref="${escaparHTML(c.participacion_ref)}" aria-expanded="${fichaAbierta}" aria-controls="${escaparHTML(fichaId)}" aria-label="Abrir ficha de participación de ${escaparHTML(c.nombre_visible)}"><strong>${escaparHTML(c.nombre_visible)}</strong></button></td>
            <td><code>${escaparHTML(c.documento_enmascarado)}</code></td>
            <td><span class="estado-chip ${claseEstado(c.estado_clave)}">${escaparHTML(etiquetaClave(c.estado_clave))}</span></td>
            <td><small>${escaparHTML(instanteVisible(c.estado_desde))}</small></td>
            <td><small>${c.disponible_desde ? escaparHTML(instanteVisible(c.disponible_desde)) : "—"}</small></td>
            <td>${detalleLlamamiento}</td>
          </tr>
          ${fichaAbierta ? renderizarModalFicha(modalFicha, fichaId) : ""}`;
      }).join("");
    }
    const paginacion = hayMas && cursorSiguiente
      ? `<div class="paginacion-bolsa">
           <button type="button" class="boton-secundario" data-bolsa-accion="pagina-siguiente" data-cursor="${escaparHTML(cursorSiguiente)}">Cargar siguientes aspirantes</button>
         </div>`
      : "";
    const vigenciaBolsa = bolsa
      ? (bolsa.vigente_hasta
        ? `${fechaVisible(bolsa.vigente_desde)} — ${fechaVisible(bolsa.vigente_hasta)}`
        : `${fechaVisible(bolsa.vigente_desde)} — vigente`)
      : "No disponible";
    const accionesBolsa = lecturaPresentacion
      ? `<div class="cuerpo-panel acciones-vista" data-bolsa-c23-pendiente>
            <button type="button" class="boton-secundario boton-ancho" disabled aria-disabled="true">Consultar historial de contactos</button>
            <button type="button" class="boton-secundario boton-ancho" disabled aria-disabled="true">Nuevo llamamiento</button>
            <button type="button" class="boton-secundario boton-ancho" disabled aria-disabled="true">Registrar resultado</button>
            <p><small>Acciones pendientes de composición en la presentación sintética.</small></p>
          </div>`
      : `<div class="cuerpo-panel acciones-vista">
            <button type="button" class="boton-secundario boton-ancho" data-bolsa-accion="cambiar-pestana" data-pestana="historico">Consultar historial de contactos</button>
            <button type="button" class="boton-primario boton-ancho" data-bolsa-accion="iniciar-b7">Nuevo llamamiento</button>
            <button type="button" class="boton-secundario boton-ancho" disabled aria-disabled="true" title="Pendiente de RRHH">Registrar resultado</button>
            <p><small>B7 emite por SMTP y deja recibo; la respuesta y la renuncia siguen pendientes de RRHH.</small></p>
          </div>`;
    const bolsas = typeof obtenerDatosBolsas === "function"
      ? obtenerDatosBolsas()?.datos?.bolsas || []
      : [];
    const bolsaVigente = bolsa?.vigente_hasta
      ? bolsas.find((item) => item.bolsa_ref !== bolsa.bolsa_ref
        && item.categoria_clave === bolsa.categoria_clave && !item.vigente_hasta)
      : null;
    const avisoSustitucion = bolsaVigente ? `
      <section class="nota-pendiente" role="note">
        <p>${escaparHTML(traducirBolsaInterna("bolsa_sustituida_aviso", { fecha: fechaAvisoSustitucion(bolsaVigente.vigente_desde) }))}</p>
        <button type="button" class="boton-secundario" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsaVigente.bolsa_ref)}">${escaparHTML(traducirBolsaInterna("bolsa_abrir_vigente"))}</button>
       </section>` : "";
    const resumenBolsa = bolsa ? `
      <aside class="resumen-lateral" aria-label="Resumen de la bolsa seleccionada">
        <section class="panel">
          <div class="cabecera-panel"><h3>Resumen de la bolsa</h3></div>
          <div class="cuerpo-panel">
            <dl class="resumen-expediente">
              <div class="fila-resumen"><dt>Categoría</dt><dd>${escaparHTML(bolsa.categoria)}</dd></div>
              <div class="fila-resumen"><dt>Tipo de lista</dt><dd>${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</dd></div>
              <div class="fila-resumen"><dt>Vigencia</dt><dd>${escaparHTML(vigenciaBolsa)}</dd></div>
              <div class="fila-resumen"><dt>Personas en bolsa</dt><dd>${numero(bolsa.total)}</dd></div>
            </dl>
          </div>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><h3>Criterios de orden</h3></div>
          <div class="cuerpo-panel"><dl class="resumen-expediente">
            <div class="fila-resumen"><dt>Criterio</dt><dd>Puntuación descendente; desempate estable por nº del acta</dd></div>
            <div class="fila-resumen"><dt>Tipo</dt><dd>${escaparHTML(etiquetaClave(bolsa.politica_orden.tipo_lista))}</dd></div>
            <div class="fila-resumen"><dt>Reposición</dt><dd>${escaparHTML(etiquetaClave(bolsa.politica_orden.reposicion))}</dd></div>
            <div class="fila-resumen"><dt>Versión y vigencia</dt><dd>v${numero(bolsa.politica_orden.version)} · ${escaparHTML(fechaVisible(bolsa.politica_orden.vigente_desde))}</dd></div>
          </dl><p class="nota-pendiente"><strong>${escaparHTML(bolsa.politica_orden.rotulo)}</strong></p></div>
        </section>
        ${avisoSustitucion}
        <section class="panel">
          <div class="cabecera-panel"><h3>Siguientes actuaciones</h3></div>
          ${accionesBolsa}
        </section>
      </aside>` : "";
    const nombres = new Map(candidatos.map((c) => [c.participacion_ref, c]));
    const llamadas = candidatos.filter((candidato) => candidato.ultimo_llamamiento).map((candidato)=>({fecha:candidato.ultimo_llamamiento.comunicado_en,candidato,tipo:traducirBolsaInterna("contacto_llamamiento_tipo"),resultado:etiquetaClave(candidato.ultimo_llamamiento.resultado),actor:"—",contacto:"—"}))
      .concat(contactos.map((contacto)=>({fecha:contacto.instante,candidato:nombres.get(contacto.participacion_ref),tipo:traducirBolsaInterna("contacto_tipo"),resultado:traducirBolsaInterna(`contacto_${contacto.resultado}`),actor:contacto.actor_ref,contacto:`${traducirBolsaInterna(`contacto_${contacto.canal}`)} · ${traducirBolsaInterna(`contacto_${contacto.resultado}`)} · ${contacto.anotacion}`})))
      .sort((a, b) => b.fecha.localeCompare(a.fecha));
    const paginaHistorico = Math.max(0, Number(filtrosActuales.pagina_historico) || 0);
    const inicioHistorico = paginaHistorico * 6;
    const paginaLlamadas = llamadas.slice(inicioHistorico, inicioHistorico + 6);
    const tablaHistorico = paginaLlamadas.length === 0
      ? `<tr><td colspan="6" class="vacio-controlado">${traducirBolsaInterna("contacto_historico_vacio")}</td></tr>`
      : paginaLlamadas.map((evento) => {
        const candidato=evento.candidato;
        return `<tr><td>${fechaMarcada(evento.fecha)}</td><td>${escaparHTML(candidato?.nombre_visible||evento.contacto)}${candidato?`<br><code>${escaparHTML(candidato.documento_enmascarado)}</code>`:""}</td><td>${escaparHTML(evento.tipo)}</td><td>${escaparHTML(evento.resultado)}</td><td>${escaparHTML(evento.actor)}</td><td>${escaparHTML(evento.contacto)}</td></tr>`;
      }).join("");
    const navegacionHistorico = llamadas.length > 6 ? `<div class="acciones-vista" aria-label="Paginación del histórico"><span>Mostrando ${numero(inicioHistorico + 1)} a ${numero(Math.min(inicioHistorico + 6, llamadas.length))} de ${numero(llamadas.length)}</span><button type="button" class="boton-secundario" data-bolsa-accion="pagina-historico" data-pagina="${paginaHistorico - 1}"${paginaHistorico === 0 ? " disabled" : ""}>Anterior</button><button type="button" class="boton-secundario" data-bolsa-accion="pagina-historico" data-pagina="${paginaHistorico + 1}"${inicioHistorico + 6 >= llamadas.length ? " disabled" : ""}>Siguiente</button></div>` : "";
    const pestanas = `<nav class="acciones-vista" role="tablist" aria-label="Vistas de la bolsa"><button type="button" class="boton-secundario" role="tab" aria-selected="${pestana === "candidatos"}" data-bolsa-accion="cambiar-pestana" data-pestana="candidatos">Candidatos</button><button type="button" class="boton-secundario" role="tab" aria-selected="${pestana === "historico"}" data-bolsa-accion="cambiar-pestana" data-pestana="historico">Histórico de llamamientos</button></nav>`;
    const contenidoHistorico = `<section class="panel" data-bolsa-b5-destino="true" tabindex="-1"><div class="cabecera-panel"><h3>${traducirBolsaInterna("contacto_historico_titulo")}</h3><span class="estado-chip info">${numero(llamadas.length)} registros</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>${traducirBolsaInterna("contacto_historico_descripcion")}</caption><thead><tr><th scope="col">Fecha</th><th scope="col">Candidato</th><th scope="col">Tipo</th><th scope="col">Resultado</th><th scope="col">Actor</th><th scope="col">Contacto</th></tr></thead><tbody>${tablaHistorico}</tbody></table></div>${navegacionHistorico}</section>`;
    return `
      ${encabezadoVista("Gestión interna de Bolsas", tituloBolsa, descripcionBolsa, accionesEncabezado)}
      ${lecturaPresentacion ? '<section class="nota-pendiente" role="note"><strong>Presentación sintética de solo lectura.</strong> Los datos visibles no acreditan contacto, envío ni entrega.</section>' : ""}
      ${lecturaPresentacion ? "" : '<section class="nota-seguridad" role="note"><strong>Operaciones conectadas.</strong> Los contactos B3 y la emisión B7 usan autorización nominal, persistencia PostgreSQL y recibos recuperables; las respuestas siguen pendientes de RRHH.</section>'}
      ${reciboFichaFuera}
      <div class="distribucion-llamamiento">
        <div>
          ${pestanas}
          ${pestana === "historico" ? contenidoHistorico : `<section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
            <div class="cabecera-panel">
              <h3>Filtros y ordenación de aspirantes (Vista B5)</h3>
              <span class="estado-chip info">${numero(candidatos.length)} en esta página</span>
            </div>
            <div class="cuerpo-panel"><div class="rejilla-kpi kpi-candidatos" aria-label="Contadores por situación">${contadoresEstado}</div></div>
            <div class="cuerpo-panel">${formularioFiltros}</div>
          </section>`}
          <section class="panel"${pestana === "historico" ? " hidden" : ""}>
            <div class="cabecera-panel"><h3>Relación ordenada de candidatos</h3></div>
            <div class="tabla-contenedor">
              <table class="tabla-datos tabla-datos--candidatos">
                <caption>Aspirantes ordenados por mérito y situación en bolsa</caption>
                <thead><tr><th scope="col">Orden</th><th scope="col">Aspirante</th><th scope="col">DNI</th><th scope="col">Situación</th><th scope="col">Fecha</th><th scope="col">Disponible desde</th><th scope="col">Último llamamiento</th></tr></thead>
                <tbody>${cuerpoTabla}</tbody>
              </table>
            </div>
            ${paginacion}
          </section>
        </div>
        ${resumenBolsa}
      </div>`;
  }
  function renderizarModalFicha(modal, fichaId) {
    if (!modal || !modal.abierto) return "";
    const candidato = modal.candidato;
    const bolsa = modal.bolsa;
    if (!candidato || !bolsa) return "";
    const vigencia = bolsa.vigente_hasta
      ? `${instanteVisible(bolsa.vigente_desde)} — ${instanteVisible(bolsa.vigente_hasta)}`
      : `${instanteVisible(bolsa.vigente_desde)} — vigente`;
    const disponibilidad = candidato.disponible_desde
      ? `<div class="fila-resumen"><dt>Disponible desde</dt><dd>${escaparHTML(instanteVisible(candidato.disponible_desde))}</dd></div>`
      : "";
    const ultimoLlamamiento = candidato.ultimo_llamamiento
      ? `<div class="fila-resumen"><dt>Último llamamiento</dt><dd>${escaparHTML(etiquetaClave(candidato.ultimo_llamamiento.canal))} · ${escaparHTML(etiquetaClave(candidato.ultimo_llamamiento.resultado))}<br><small><time datetime="${escaparHTML(candidato.ultimo_llamamiento.comunicado_en)}">${escaparHTML(instanteVisible(candidato.ultimo_llamamiento.comunicado_en))}</time> · <code>${escaparHTML(candidato.ultimo_llamamiento.llamamiento_ref)}</code></small></dd></div>`
      : `<div class="fila-resumen"><dt>Último llamamiento</dt><dd>Sin llamamientos registrados</dd></div>`;
    const destinos = {
      disponible: ["no_disponible", "pendiente_incorporacion", "renuncia", "excluido"],
      no_disponible: ["disponible", "excluido"],
      pendiente_incorporacion: ["trabajando", "disponible", "renuncia", "excluido"],
      trabajando: ["disponible", "disponible_desde", "excluido"],
      disponible_desde: ["disponible", "excluido"],
      renuncia: ["disponible", "excluido"],
      excluido: [],
    }[candidato.estado_clave] || [];
    const cambio = modal.cambioSituacion ? `
      <form data-bolsa-form="cambio-situacion" data-participacion-ref="${escaparHTML(candidato.participacion_ref)}">
        <p><small>Transiciones provisionales hasta respuesta de RRHH a las dudas 13 y 18.</small></p>
        <label>Destino <select name="situacion" required><option value="">Seleccionar estado</option>${destinos.map((d) => `<option value="${d}">${escaparHTML(etiquetaClave(d))}</option>`).join("")}</select></label>
        <label data-bolsa-fecha-disponible>Fecha de disponibilidad <input type="datetime-local" name="fecha_disponible"></label>
        <label>Motivo <textarea name="motivo" required maxlength="1000"></textarea></label>
        <button type="submit" class="boton-primario"${destinos.length ? "" : " disabled"}>Guardar cambio</button>
        <p class="mensaje-error" role="alert">${escaparHTML(modal.errorCambioSituacion || "")}</p>
      </form>` : "";
    const reciboSituacion = modal.reciboSituacion ? `<p class="mensaje-exito" role="status">Cambio registrado. Recibo <code>${escaparHTML(modal.reciboSituacion)}</code>.</p>` : "";
    const reciboContacto = modal.reciboContacto ? `<p class="mensaje-exito" role="status">${escaparHTML(traducirBolsaInterna("contacto_registrado", { recibo: modal.reciboContacto }))}</p>` : "";
    const opcionLlamamiento = candidato.ultimo_llamamiento ? `<option value="${escaparHTML(candidato.ultimo_llamamiento.llamamiento_ref)}">${escaparHTML(candidato.ultimo_llamamiento.llamamiento_ref)}</option>` : "";
    const t = traducirBolsaInterna;
    const formularioContacto = `<form data-bolsa-form="contacto" data-participacion-ref="${escaparHTML(candidato.participacion_ref)}"><h4>${t("contacto_registrar")}</h4><label>${t("contacto_canal")} <select name="canal" required><option value="telefono">${t("contacto_telefono")}</option><option value="correo">${t("contacto_correo")}</option><option value="sms">${t("contacto_sms")}</option><option value="presencial">${t("contacto_presencial")}</option><option value="otro">${t("contacto_otro")}</option></select></label><label>${t("contacto_resultado")} <select name="resultado" required><option value="contactado">${t("contacto_contactado")}</option><option value="no_contesta">${t("contacto_no_contesta")}</option><option value="buzon">${t("contacto_buzon")}</option><option value="acepta">${t("contacto_acepta")}</option><option value="rechaza">${t("contacto_rechaza")}</option><option value="aplazado">${t("contacto_aplazado")}</option><option value="otro">${t("contacto_otro")}</option></select></label>${opcionLlamamiento?`<label>${t("contacto_llamamiento")} <select name="llamamiento_ref"><option value="">${t("contacto_sin_vincular")}</option>${opcionLlamamiento}</select></label>`:""}<label>${t("contacto_anotacion")} <textarea name="anotacion" required maxlength="1000"></textarea></label><button type="submit" class="boton-primario">${t("contacto_registrar")}</button><p class="mensaje-error" role="alert">${escaparHTML(modal.errorContacto||"")}</p></form>`;
    return `
      <tr class="fila-ficha-participacion" data-ficha-participacion-ref="${escaparHTML(candidato.participacion_ref)}">
        <td colspan="7">
          <section id="${escaparHTML(fichaId)}" class="panel" data-bolsa-ficha-inline="true" tabindex="-1" aria-labelledby="titulo-${escaparHTML(fichaId)}">
            <div class="cabecera-panel">
              <h3 id="titulo-${escaparHTML(fichaId)}">Ficha de participación</h3>
              <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-ficha" aria-label="Cerrar ficha de participación">×</button>
            </div>
            <div class="cuerpo-panel">
              <dl class="resumen-expediente">
                <div class="fila-resumen"><dt>Bolsa</dt><dd>${escaparHTML(bolsa.categoria)}<br><small>${escaparHTML(bolsa.categoria_clave)} · ${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</small></dd></div>
                <div class="fila-resumen"><dt>Vigencia</dt><dd>${escaparHTML(vigencia)}</dd></div>
                <div class="fila-resumen"><dt>Orden del acta</dt><dd>#${numero(candidato.orden_acta)}</dd></div>
                <div class="fila-resumen"><dt>Último cambio de situación</dt><dd>${escaparHTML(instanteVisible(candidato.estado_desde))}</dd></div>
                ${disponibilidad}
                <div class="fila-resumen"><dt>Referencia de participación</dt><dd><code>${escaparHTML(candidato.participacion_ref)}</code></dd></div>
                ${ultimoLlamamiento}
                <div class="fila-resumen"><dt>${traducirBolsaInterna("contacto_contador")}</dt><dd>${numero(candidato.contactos_total)}</dd></div>
              </dl>
              ${reciboSituacion}
              ${reciboContacto}
              ${renderizarOperacionesSituacion({ candidato, estado: modal.operacionesB8 || {}, escaparHTML })}
            </div>
            <div class="acciones-vista">
              <button type="button" class="boton-primario" data-bolsa-accion="abrir-cambio-situacion">Cambiar situación</button>
              <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-ficha">Cerrar</button>
            </div>
          </section>
          ${cambio}
          ${formularioContacto}
        </td>
      </tr>`;
  }
  function renderizarModalContactos(modal) {
    if (!modal || !modal.abierto) return "";
    let contenido = "";
    if (modal.carga === "cargando") {
      contenido = '<p class="vacio-controlado" role="status" aria-busy="true">Cargando historial de contactos…</p>';
    } else if (modal.carga === "error") {
      contenido = `<p class="mensaje-error" role="alert">${escaparHTML(modal.error || "No se pudieron consultar los contactos.")}</p>`;
    } else if (!modal.contactos || modal.contactos.length === 0) {
      contenido = '<p class="vacio-controlado" role="status">No hay contactos previos registrados para este aspirante.</p>';
    } else {
      const filas = modal.contactos.map((ct) => `
        <tr data-contacto-ref="${escaparHTML(ct.contacto_ref)}">
          <td><span class="estado-chip neutro">${escaparHTML(etiquetaClave(ct.canal))}</span></td>
          <td><time datetime="${escaparHTML(ct.realizado_en)}">${escaparHTML(instanteVisible(ct.realizado_en))}</time></td>
          <td><span class="estado-chip ${claseEstado(ct.resultado_clave)}">${escaparHTML(etiquetaClave(ct.resultado_clave))}</span></td>
          <td><small>${escaparHTML(ct.anotacion || "—")}</small></td>
        </tr>
      `).join("");
      contenido = `
        <div class="tabla-contenedor">
          <table class="tabla-datos">
            <caption>Historial de comunicaciones y respuestas del aspirante</caption>
            <thead>
              <tr>
                <th scope="col">Canal</th>
                <th scope="col">Fecha y hora</th>
                <th scope="col">Resultado</th>
                <th scope="col">Anotación</th>
              </tr>
            </thead>
            <tbody>
              ${filas}
            </tbody>
          </table>
        </div>`;
    }
    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-contactos">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-contactos">Historial de contactos: ${escaparHTML(modal.nombreVisible || modal.participacionRef)}</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-contactos" aria-label="Cerrar">×</button>
          </div>
          <div class="cuerpo-panel">
            ${esLecturaPresentacion() === true ? '<p class="nota-pendiente">Historial sintético de solo lectura; no acredita contacto, envío ni entrega.</p>' : ""}
            ${contenido}
          </div>
          <div class="acciones-vista">
            <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-contactos">Cerrar</button>
          </div>
        </div>
      </div>`;
  }
  function renderizarModalLlamar(modal) {
    if (!modal || !modal.abierto) return "";
    const errorHtml = modal.error
      ? `<div class="mensaje-error" role="alert"><p><strong>Error:</strong> ${escaparHTML(modal.error)}</p></div>`
      : "";
    const enviando = modal.carga === "enviando";
    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-llamar">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-llamar">Nuevo llamamiento (B7): ${escaparHTML(modal.nombreVisible || modal.participacionRef)}</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-llamar" aria-label="Cerrar">×</button>
          </div>
          <div class="cuerpo-panel">
            ${errorHtml}
            <p>Aspirante en orden <strong>#${numero(modal.orden)}</strong> de la bolsa. Se registrará la comunicación y el plazo de respuesta.</p>
            <form data-bolsa-form="llamar" data-participacion-ref="${escaparHTML(modal.participacionRef)}">
              <div class="campo-formulario">
                <label for="llamar-canal">Canal de comunicación *</label>
                <select id="llamar-canal" name="canal" required>
                  <option value="correo">Correo electrónico</option>
                  <option value="telefono">Teléfono</option>
                  <option value="sede">Sede electrónica</option>
                </select>
              </div>
              <div class="campo-formulario">
                <label for="llamar-comunicado-en">Fecha y hora de comunicación *</label>
                <input type="datetime-local" id="llamar-comunicado-en" name="comunicado_en" required>
              </div>
              <div class="campo-formulario">
                <label for="llamar-plazo-hasta">Plazo límite de respuesta *</label>
                <input type="datetime-local" id="llamar-plazo-hasta" name="plazo_respuesta_hasta" required>
              </div>
              <div class="campo-formulario">
                <label for="llamar-anotacion">Anotación u observaciones (opcional)</label>
                <textarea id="llamar-anotacion" name="anotacion" rows="3" maxlength="1024" placeholder="Detalle del intento de localización o notas del llamamiento…"></textarea>
              </div>
              <div class="campo-confirmacion">
                <label for="llamar-confirmacion">
                  <input type="checkbox" id="llamar-confirmacion" name="confirmacion" value="true" required>
                  Confirmo el llamamiento formal al aspirante conforme a las normas de gestión de bolsa.
                </label>
              </div>
              <div class="acciones-formulario">
                <button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${enviando ? "Registrando…" : "Registrar llamamiento"}</button>
                <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-llamar">Cancelar</button>
              </div>
            </form>
          </div>
        </div>
      </div>`;
  }
  function renderizarModalResultado(modal) {
    if (!modal || !modal.abierto) return "";
    const errorHtml = modal.error
      ? `<div class="mensaje-error" role="alert"><p><strong>Error:</strong> ${escaparHTML(modal.error)}</p></div>`
      : "";
    const enviando = modal.carga === "enviando";
    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-resultado">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-resultado">Registrar resultado de llamamiento (B3)</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-resultado" aria-label="Cerrar">×</button>
          </div>
          <div class="cuerpo-panel">
            ${errorHtml}
            <p>Aspirante: <strong>${escaparHTML(modal.nombreVisible || modal.participacionRef)}</strong></p>
            <form data-bolsa-form="resultado" data-llamamiento-ref="${escaparHTML(modal.llamamientoRef)}">
              <div class="campo-formulario">
                <label for="resultado-clave">Resultado del llamamiento *</label>
                <select id="resultado-clave" name="resultado_clave" required>
                  <option value="">Seleccione un resultado…</option>
                  <option value="aceptado">Aceptado (pasa a situación Ocupado)</option>
                  <option value="renuncia">Renuncia (pasa a Renuncia pendiente)</option>
                  <option value="sin_respuesta">Sin respuesta (continúa Disponible tras salto)</option>
                </select>
              </div>
              <div class="campo-formulario">
                <label for="resultado-anotacion">Anotación administrativa (opcional)</label>
                <textarea id="resultado-anotacion" name="anotacion" rows="3" maxlength="1024" placeholder="Observaciones sobre la respuesta o justificante aportado…"></textarea>
              </div>
              <div class="campo-confirmacion">
                <label for="resultado-confirmacion">
                  <input type="checkbox" id="resultado-confirmacion" name="confirmacion" value="true" required>
                  Confirmo el resultado del llamamiento y los efectos sobre la posición en bolsa.
                </label>
              </div>
              <div class="acciones-formulario">
                <button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${enviando ? "Guardando…" : "Guardar resultado"}</button>
                <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-resultado">Cancelar</button>
              </div>
            </form>
          </div>
        </div>
      </div>`;
  }
  function renderizarNoConectada(vista) {
    return `
      ${encabezadoVista("Contrato real de alcance mínimo", tituloVista(vista), "Esta sección no está disponible con el contrato interno actualmente conectado.", '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro de mando</button>')}
      <section class="panel"><div class="cuerpo-panel vacio-controlado"><p><strong>Funcionalidad no conectada</strong></p><p>El servidor solo ha acreditado indicadores agregados, convocatorias y actuaciones pendientes. No se muestran valores cero, tablas vacías ni controles aparentes para datos que no han sido proporcionados.</p></div></section>`;
  }
  function renderizarVista(vista) {
    if (!esActivo()) throw new Error("el presentador requiere un panel interno válido");
    const datos = datosPanel();
    if (vista === "resumen") return renderizarResumen(datos);
    if (vista === "elaboracion") return renderizarConvocatorias(datos);
    if (vista === "bolsa-candidatos") return renderizarCandidatosBolsa();
    if (vista === "estadisticas") return renderizarEstadisticasBolsa();
    return renderizarNoConectada(vista);
  }
  // Sin panel agregado compuesto, las bolsas constituidas (B12/B5) se muestran
  // igualmente: tienen su propia API y no dependen de los indicadores.
  function renderizarSoloBolsas(vista) {
    if (vista === "bolsa-candidatos") return renderizarCandidatosBolsa();
    const estadoAvisos = typeof obtenerDatosAvisos === "function" ? obtenerDatosAvisos() : null;
    return `<div class="cuadro-bolsa-solo">
      ${renderizarCuadroB12()}
      ${renderizarBloqueAvisos({
        estado: estadoAvisos?.carga || "cargando",
        datos: estadoAvisos?.datos || null,
        error: estadoAvisos?.error || "",
      })}</div>`;
  }
  return Object.freeze({ actualizarContextoSesion, esActivo, etiquetaFuente, renderizarEstadisticasBolsa, renderizarSoloBolsas, renderizarVista });
}
