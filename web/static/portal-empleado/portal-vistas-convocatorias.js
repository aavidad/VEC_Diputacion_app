import { traducirBolsaInterna } from "./portal-i18n.js";
import { traducirConvocatoriasS1 } from "./portal-i18n-convocatorias.js";
/** Vistas compartidas de gobierno de convocatorias y admisión. */

export function crearVistasConvocatorias(u) {
  const { escaparHTML: e, numero, fecha, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, botonOperacion, campo, fuentePresentacion } = u;

  function valorFiltro(estado, grupo, nombre, porDefecto = "") {
    return String(estado?.filtros?.[grupo]?.[nombre] ?? porDefecto);
  }

  function opcion(valor, seleccionada, etiqueta = valor) {
    return `<option value="${e(valor)}"${valor === seleccionada ? " selected" : ""}>${e(etiqueta)}</option>`;
  }

  function contiene(valor, busqueda) {
    return String(valor || "").toLocaleLowerCase("es").includes(String(busqueda || "").trim().toLocaleLowerCase("es"));
  }

  function urlDocumentoPublico(valor) {
    const url = String(valor || "");
    return /^\/bolsa\/documentos\/bases-(?:auxiliar|gestion|operario)-demo\.(?:pdf|html)$/.test(url) ? url : "";
  }

  function enlacesDocumentosPublicos(elaboracion) {
    const documentos = (elaboracion?.documentos_publicos || []).map((documento) => {
      const url = urlDocumentoPublico(documento.url);
      if (!url) return "";
      return `<a class="boton-secundario" href="${e(url)}" target="_blank" rel="noopener noreferrer">${e(documento.titulo)} <span aria-hidden="true">↗</span></a>`;
    }).filter(Boolean);
    return documentos.length > 0
      ? documentos.join("")
      : '<p class="nota-pendiente">No hay una reproducción documental asociada a este escenario.</p>';
  }

  function estadoFuente(estado, titulo) {
    const error = String(estado?.errorFuente || "").trim();
    if (error) return `<section class="panel"><div class="cuerpo-panel vacio-controlado" role="alert"><p><strong>Error al cargar ${e(titulo)}</strong></p><p>${e(error)}</p></div></section>`;
    if (estado?.datosBolsas?.carga === "denegado") return `<section class="panel"><div class="cuerpo-panel vacio-controlado" role="alert"><p><strong>Acceso denegado</strong></p><p>La sesión no dispone de permiso para consultar ${e(titulo)}.</p></div></section>`;
    if (estado?.fuenteLista === false || estado?.datosBolsas?.carga === "cargando") return `<section class="panel"><div class="cuerpo-panel vacio-controlado" role="status" aria-live="polite"><p><strong>Cargando ${e(titulo)}…</strong></p><p>Se está comprobando la sesión y el ámbito de acceso con la API interna.</p></div></section>`;
    return "";
  }

  function renderizarConvocatorias(datos, estado = {}) {
    const bloqueoFuente = estadoFuente(estado, "convocatorias");
    if (bloqueoFuente) return `${encabezadoVista("Expediente electrónico de selección", "Convocatorias, bases y calendario", "La consulta interna permanece cerrada hasta recibir una fuente autorizada.")}${bloqueoFuente}`;
    const seleccionada = datos.elaboraciones.find((item) => item.id === estado.elaboracionSeleccionada)
      || datos.elaboraciones[0];
    const texto = valorFiltro(estado, "convocatorias", "texto");
    const estadoSeleccionado = valorFiltro(estado, "convocatorias", "estado", "Todos");
    const unidad = valorFiltro(estado, "convocatorias", "unidad", "Todas");
    const elaboraciones = datos.elaboraciones.filter((item) => {
      const coincideBusqueda = !texto || [item.id, item.nombre, item.expediente, item.fase]
        .some((valor) => contiene(valor, texto));
      const coincideEstado = estadoSeleccionado === "Todos" || item.estado === estadoSeleccionado;
      const coincideUnidad = unidad === "Todas" || item.responsable === unidad;
      return coincideBusqueda && coincideEstado && coincideUnidad;
    });
    const filas = elaboraciones.map((item) => [
      `<button type="button" class="enlace-tabla" data-accion="seleccionar-elaboracion" data-id="${e(item.id)}">${e(item.id)}</button>`,
      `<strong>${e(item.nombre)}</strong><br><small>${e(item.expediente)}</small>`,
      `<strong>${e(item.cve_bop || "Sin referencia pública")}</strong><br><small>${e(item.publicacion_bop ? `Publicado ${item.publicacion_bop}` : "Sin fecha pública")}</small>`,
      e(item.fase), e(item.reglas), chip(item.estado),
    ]);
    return `
      ${encabezadoVista("Expediente electrónico de selección", "Convocatorias, bases y calendario", "Configuración completa y versionada de una Bolsa antes de firma, publicación y apertura.", botonOperacion("Nueva convocatoria", "crear-convocatoria", "DEMO-BOL-NUEVA", "boton-primario"))}
      ${avisoPresentacion("Los formularios y estados son los definitivos; durante la presentación solo cambia una copia volátil y cada actuación genera un recibo DEMO.")}
      <div class="rejilla-kpi">${kpi("BOR", numero(datos.elaboraciones.filter((x) => x.estado === "Borrador").length), "Borradores")}${kpi("REV", numero(datos.elaboraciones.filter((x) => /revisión/i.test(x.estado)).length), "En revisión")}${kpi("FIR", "2", "Circuitos pendientes")}${kpi("PUB", numero(datos.elaboraciones.filter((x) => /publicada/i.test(x.estado)).length), "Publicadas")}</div>
      <div class="rejilla-elaboracion">
        <section class="panel">
          <div class="cabecera-panel"><div><h3>Expedientes de convocatoria</h3><p>Seleccione uno para editar su configuración.</p></div>${fuentePresentacion()}</div>
          <form class="barra-filtros" aria-label="Filtros de convocatorias" data-filtro="convocatorias">
            ${campo("Buscar", `<input type="search" name="texto" value="${e(texto)}" placeholder="Referencia o categoría">`)}
            ${campo("Estado", `<select name="estado">${["Todos", "Borrador", "En revisión", "Publicada"].map((valor) => opcion(valor, estadoSeleccionado)).join("")}</select>`)}
            ${campo("Unidad responsable", `<select name="unidad">${opcion("Todas", unidad)}${opcion("Unidad DEMO de Selección", unidad)}</select>`)}
            <button type="submit" class="boton-secundario">Aplicar filtros</button>
          </form>
          <p class="resultado-filtro" role="status" data-total-filtro="convocatorias" data-total="${elaboraciones.length}">${traducirBolsaInterna("numero_convocatorias", { numero: numero(elaboraciones.length) })}</p>
          ${tabla({
            titulo: "Convocatorias de Bolsa",
            cabeceras: ["Referencia DEMO", "Convocatoria pública / expediente DEMO", "BOP / publicación", "Fase DEMO", "Baremo DEMO", "Estado DEMO"],
            clavesColumnas: ["referencia", "convocatoria", "publicacion", "fase", "baremo", "estado"],
            prioridadColumnas: "estado",
            filas,
          })}
        </section>
        <aside class="resumen-lateral">
          <section class="panel"><div class="cabecera-panel"><div><h3>${e(seleccionada?.id || "Sin selección")}</h3><p>${e(seleccionada?.nombre || "No hay expedientes")}</p></div>${seleccionada ? chip(seleccionada.estado) : ""}</div>
            <div class="cuerpo-panel"><dl class="resumen-expediente">
              <div class="fila-resumen"><dt>Expediente</dt><dd>${e(seleccionada?.expediente || "—")}</dd></div>
              <div class="fila-resumen"><dt>Bases</dt><dd>${e(seleccionada?.version_bases || "—")}</dd></div>
              <div class="fila-resumen"><dt>Publicación</dt><dd>${e(seleccionada?.publicacion_bop || "—")}</dd></div>
              <div class="fila-resumen"><dt>Referencia pública</dt><dd>${e(seleccionada?.identificador_publico || "—")}</dd></div>
              <div class="fila-resumen"><dt>Calendario</dt><dd>${e(seleccionada?.calendario || "—")}</dd></div>
              <div class="fila-resumen"><dt>Firmantes</dt><dd>${e(seleccionada?.firmantes || "—")}</dd></div>
              <div class="fila-resumen"><dt>Responsable</dt><dd>${e(seleccionada?.responsable || "—")}</dd></div>
            </dl>
            <p class="nota-informativa"><strong>Fuente pública real:</strong> título, fecha y CVE/BOP. Los PDF y HTML son reproducciones adaptadas para la presentación, sin validez administrativa. El expediente y su tramitación son sintéticos.</p>
            <div class="acciones-verticales" aria-label="Bases públicas adaptadas">
              ${enlacesDocumentosPublicos(seleccionada)}
            </div>
            <div class="acciones-verticales">
              ${botonOperacion("Guardar bases y baremo", "guardar-bases", seleccionada?.id || "DEMO-BOL-SIN-SELECCION", "boton-primario")}
              ${botonOperacion("Enviar al circuito de firma", "enviar-firma-convocatoria", seleccionada?.id || "DEMO-BOL-SIN-SELECCION")}
              ${botonOperacion("Publicar convocatoria", "publicar-convocatoria", seleccionada?.id || "DEMO-BOL-SIN-SELECCION")}
            </div></div>
          </section>
        </aside>
      </div>
      <section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Formulario de bases y calendario</h3><p>Escenario editable DEMO. En producción los valores procederán de las bases aprobadas y nunca quedarán fijados en código.</p></div><span class="estado-chip violeta">Versión de trabajo DEMO</span></div>
        <form class="cuerpo-panel formulario-gobernado" aria-label="Configuración de bases y calendario" data-comando="guardar-bases">
          <fieldset><legend>Identificación</legend><div class="rejilla-formulario">${campo("Denominación pública", `<input name="denominacion" value="${e(seleccionada?.nombre || "")}">`)}${campo("Categoría o proceso", `<select name="categoria">${datos.elaboraciones.map((item) => opcion(item.nombre, seleccionada?.nombre)).join("")}</select>`)}${campo("Código de expediente DEMO", `<input name="expediente" value="${e(seleccionada?.expediente || "")}" readonly>`)}${campo("Tipo de proceso", `<select name="tipo_proceso">${["Bolsa de trabajo", "Proceso selectivo"].map((valor) => opcion(valor, seleccionada?.tipo_publico)).join("")}</select>`)}</div></fieldset>
          <fieldset><legend>Calendario gobernado</legend><div class="rejilla-formulario">${campo("Apertura de solicitudes", '<input type="datetime-local" name="apertura" value="2026-08-01T09:00">')}${campo("Cierre de solicitudes", '<input type="datetime-local" name="cierre" value="2026-08-20T23:59">')}${campo("Subsanación desde", '<input type="date" name="subsanacion_desde" value="2026-08-25">')}${campo("Subsanación hasta", '<input type="date" name="subsanacion_hasta" value="2026-09-05">')}</div></fieldset>
          <fieldset><legend>Documentación y publicación</legend><div class="rejilla-formulario">${campo("Versión de bases", '<input name="version_bases" value="v3">')}${campo("Medio de publicación", '<select name="medio_publicacion"><option>BOP + sede + portal</option><option>Sede + portal</option></select>')}${campo("Plantilla documental", '<select name="plantilla"><option>DEMO-PLT-BASES-v4</option></select>')}${campo("Circuito de firma", '<select name="circuito_firma"><option>DEMO-FIR-CONVOCATORIA-v2</option></select>')}</div></fieldset>
          <div class="acciones-formulario">${botonOperacion("Validar y guardar versión", "guardar-bases", seleccionada?.id || "DEMO-BOL-SIN-SELECCION", "boton-primario")}</div>
        </form>
      </section>`;
  }

  function renderizarSolicitudes(datos, estado = {}) {
    const bloqueoFuente = estadoFuente(estado, "solicitudes");
    if (bloqueoFuente) return `${encabezadoVista("Bandeja de tramitación", "Solicitudes, admisión y subsanación", "La consulta interna permanece cerrada hasta recibir una fuente autorizada.")}${bloqueoFuente}`;
    const pendientes = datos.solicitudes.filter((item) => /pendiente/i.test(item.estado)).length;
    const referencia = valorFiltro(estado, "solicitudes", "referencia");
    const convocatoria = valorFiltro(estado, "solicitudes", "convocatoria", "Todas");
    const estadoSeleccionado = valorFiltro(estado, "solicitudes", "estado", "Todos");
    const solicitudes = datos.solicitudes.filter((item) => {
      const coincideReferencia = !referencia || [item.id, item.persona_ref].some((valor) => contiene(valor, referencia));
      const coincideConvocatoria = convocatoria === "Todas" || item.convocatoria === convocatoria;
      const coincideEstado = estadoSeleccionado === "Todos" || item.estado === estadoSeleccionado;
      return coincideReferencia && coincideConvocatoria && coincideEstado;
    });
    const filas = solicitudes.map((item) => [
      `<strong>${e(item.id)}</strong>`, e(item.persona_ref), e(item.convocatoria), e(fecha(item.registrada)),
      e(item.requisitos), e(fecha(item.subsanacion)), chip(item.estado),
      `<div class="acciones-fila">${botonOperacion("Admitir", "admitir-solicitud", item.id, "boton-terciario")}${botonOperacion("Excluir", "excluir-solicitud", item.id, "boton-terciario")}${botonOperacion("Subsanar", "registrar-subsanacion", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Bandeja de tramitación", "Solicitudes, admisión y subsanación", "Revisión de requisitos, listas provisionales y subsanaciones con motivación y trazabilidad.", botonOperacion("Publicar lista provisional", "publicar-lista-provisional", "DEMO-BOL-014", "boton-primario"))}
      ${avisoPresentacion()}
      <div class="rejilla-kpi">${kpi("REG", numero(datos.solicitudes.length), "Registradas")}${kpi("PEN", numero(pendientes), "Pendientes")}${kpi("SUB", numero(datos.solicitudes.filter((x) => /subsan/i.test(x.estado)).length), "En subsanación")}${kpi("ADM", numero(datos.solicitudes.filter((x) => /admitida/i.test(x.estado)).length), "Admitidas")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Bandeja de solicitudes</h3><p>No se muestran nombres, documentos de identidad ni datos de contacto en el listado.</p></div>${fuentePresentacion()}</div>
        <form class="barra-filtros" aria-label="Filtros de solicitudes" data-filtro="solicitudes">${campo("Buscar referencia", `<input type="search" name="referencia" value="${e(referencia)}" placeholder="DEMO-SOL-…">`)}${campo("Convocatoria", `<select name="convocatoria">${["Todas", "DEMO-BOL-014", "DEMO-BOL-021"].map((valor) => opcion(valor, convocatoria)).join("")}</select>`)}${campo("Estado", `<select name="estado">${["Todos", "Pendiente de revisión", "Pendiente de subsanación", "Admitida provisional"].map((valor) => opcion(valor, estadoSeleccionado)).join("")}</select>`)}<button type="submit" class="boton-secundario">Aplicar filtros</button></form>
        <p class="resultado-filtro" role="status" data-total-filtro="solicitudes" data-total="${solicitudes.length}">${traducirBolsaInterna("numero_solicitudes", { numero: numero(solicitudes.length) })}</p>
        ${tabla({
          titulo: "Solicitudes presentadas",
          cabeceras: ["Solicitud", "Persona", "Convocatoria", "Registro", "Requisitos", "Subsanación", "Estado", "Acciones"],
          clavesColumnas: ["referencia", "persona", "convocatoria", "registro", "requisitos", "subsanacion", "estado", "acciones"],
          prioridadColumnas: "estado-acciones",
          filas,
        })}
      </section>
      <section class="nota-pendiente">Las exclusiones requieren causa tipificada y texto motivado. La demo muestra el cambio de estado, pero no genera resolución, asiento registral ni notificación fehaciente.</section>`;
  }

  return Object.freeze({ renderizarConvocatorias, renderizarSolicitudes });
}

/**
 * Consulta S1 de solo lectura. El montaje aporta funciones autorizadas; esta
 * vista no conoce rutas HTTP, credenciales, permisos ni datos de presentación.
 *
 * consultarLista({ signal }) -> [{ referencia, titulo, categoria, estado,
 *   version_actual, cierre_plazo }]
 * consultarDetalle(referencia, { signal }) -> { referencia, titulo, resumen,
 *   categoria, estado, identificador_publico, version_actual,
 *   versiones: [{ codigo, estado, publicada_en }],
 *   bases: { codigo, resumen, requisitos: [{ referencia, titulo, descripcion,
 *     obligatorio, hito_exigibilidad }], hitos: [{ titulo, fecha }],
 *     documentos: [{ titulo, referencia }] } }
 * La capa que consulta valida el contrato y la autorización antes de entregar
 * la proyección. Aquí se exige una forma mínima y se escapa cada dato visible.
 */
export function crearSuperficieConvocatoriasS1({
  contenedor, consultarLista, consultarDetalle, escaparHTML = escaparHTMLConvocatorias,
  traducir = traducirConvocatoriasS1,
} = {}) {
  const e = (valor) => escaparHTML(String(valor ?? ""));
  const t = (clave, variables) => traducir(clave, variables);
  let activo = false;
  let generacion = 0;
  let controlador = null;
  let estado = { carga: "no_configurado", convocatorias: [], detalle: null, seleccionada: "" };

  function fecha(valor) {
    if (!valor) return t("sin_fecha");
    if (!/^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:\d{2}))?$/.test(String(valor))) return t("sin_fecha");
    const instante = new Date(valor);
    if (Number.isNaN(instante.getTime())) return t("sin_fecha");
    return new Intl.DateTimeFormat("es-ES", {
      dateStyle: "medium", ...(String(valor).includes("T") ? { timeStyle: "short" } : {}),
      timeZone: "Europe/Madrid",
    }).format(instante);
  }

  function texto(valor) { return valor ? e(valor) : e(t("sin_dato")); }
  function lista(valor) { return Array.isArray(valor) ? valor : []; }
  function validaLista(valor) {
    return Array.isArray(valor) && valor.length <= 100
      && new Set(valor.map((item) => item?.referencia)).size === valor.length
      && valor.every((item) => item
      && typeof item.referencia === "string" && item.referencia.length > 0
      && typeof item.titulo === "string" && item.titulo.length > 0);
  }
  function validaDetalle(valor, referencia) {
    return valor && typeof valor === "object" && valor.referencia === referencia
      && typeof valor.titulo === "string" && valor.titulo.length > 0
      && valor.bases && typeof valor.bases === "object"
      && Array.isArray(valor.versiones) && Array.isArray(valor.bases.requisitos)
      && Array.isArray(valor.bases.hitos) && Array.isArray(valor.bases.documentos)
      && valor.versiones.length <= 100 && valor.bases.requisitos.length <= 256
      && valor.bases.hitos.length <= 100 && valor.bases.documentos.length <= 100
      && valor.versiones.every((item) => item && typeof item === "object")
      && valor.bases.requisitos.every((item) => item && typeof item === "object")
      && valor.bases.hitos.every((item) => item && typeof item === "object")
      && valor.bases.documentos.every((item) => item && typeof item === "object");
  }
  function aviso(carga) {
    const alerta = carga === "denegado" || carga === "error" || carga === "error_detalle";
    return `<section class="panel s1-estado" ${alerta ? 'role="alert"' : 'role="status"'} aria-live="polite" tabindex="-1"><div class="cabecera-panel"><h3>${e(t(`estado_${carga}_titulo`))}</h3></div><div class="cuerpo-panel"><p>${e(t(`estado_${carga}_detalle`))}</p></div></section>`;
  }
  function controlNoConectado(clave) {
    return `<button type="button" class="boton-secundario" disabled aria-disabled="true" title="${e(t("accion_no_conectada"))}">${e(t(clave))}</button>`;
  }
  function ficha(item) {
    const seleccionada = item.referencia === estado.seleccionada;
    return `<li class="s1-elemento${seleccionada ? " s1-elemento-activo" : ""}"><button type="button" data-s1-convocatoria="${e(item.referencia)}" aria-current="${seleccionada ? "true" : "false"}"><strong>${e(item.titulo)}</strong><span>${texto(item.categoria)}</span><span class="s1-meta">${e(t("version"))}: ${texto(item.version_actual)} · ${e(t("cierre"))}: ${e(fecha(item.cierre_plazo))}</span><span class="estado-chip info">${texto(item.estado)}</span></button></li>`;
  }
  const apartados = Object.freeze([
    ["resumen", "resumen"], ["bases", "bases_apartado"], ["versiones", "navegacion_versiones"],
    ["requisitos", "navegacion_requisitos"], ["hitos", "navegacion_hitos"], ["documentos", "navegacion_documentos"],
  ]);
  function navegacionDetalle() {
    return `<nav class="s1-navegacion" aria-label="${e(t("navegacion_detalle"))}">${apartados.map(([id, clave]) => `<button type="button" data-s1-seccion="${id}" aria-controls="s1-seccion-${id}">${e(t(clave))}</button>`).join("")}</nav>`;
  }
  function grupo(id, titulo, contenido, vacio) {
    return `<section class="panel" id="s1-seccion-${id}" tabindex="-1"><div class="cabecera-panel"><h3>${e(t(titulo))}</h3></div><div class="cuerpo-panel">${contenido || `<p class="s1-vacio">${e(t(vacio))}</p>`}</div></section>`;
  }
  function detalleHTML(detalle) {
    if (!detalle) return aviso("sin_seleccion");
    const bases = detalle.bases;
    const versiones = lista(detalle.versiones).map((item) => `<li class="s1-fila"><strong>${texto(item.codigo)}</strong><span>${texto(item.estado)}</span><time>${e(fecha(item.publicada_en))}</time></li>`).join("");
    const requisitos = lista(bases.requisitos).map((item) => `<li class="s1-fila s1-requisito"><div><strong>${texto(item.titulo)}</strong><p>${texto(item.descripcion)}</p></div><div><span class="estado-chip ${item.obligatorio === true ? "aviso" : "info"}">${e(t(item.obligatorio === true ? "obligatorio" : "no_obligatorio"))}</span><small>${e(t("hito_exigibilidad"))}: ${texto(item.hito_exigibilidad)}</small></div></li>`).join("");
    const hitos = lista(bases.hitos).map((item) => `<li class="s1-fila"><strong>${texto(item.titulo)}</strong><time>${e(fecha(item.fecha))}</time></li>`).join("");
    const documentos = lista(bases.documentos).map((item) => `<li class="s1-fila"><strong>${texto(item.titulo)}</strong><span>${texto(item.referencia)}</span></li>`).join("");
    return `<div class="s1-detalle">
      ${navegacionDetalle()}
      <section class="panel s1-resumen" id="s1-seccion-resumen" tabindex="-1"><div class="cabecera-panel"><div><h3>${e(detalle.titulo)}</h3><p>${texto(detalle.categoria)}</p></div><span class="estado-chip info">${texto(detalle.estado)}</span></div><div class="cuerpo-panel"><p>${texto(detalle.resumen)}</p><dl class="resumen-expediente"><div class="fila-resumen"><dt>${e(t("version_actual"))}</dt><dd>${texto(detalle.version_actual)}</dd></div><div class="fila-resumen"><dt>${e(t("identificador_publico"))}</dt><dd>${texto(detalle.identificador_publico)}</dd></div></dl></div></section>
      ${grupo("bases", "bases_apartado", `<dl class="s1-bases-datos"><div><dt>${e(t("version"))}</dt><dd>${texto(bases.codigo)}</dd></div></dl>${bases.resumen ? `<p>${texto(bases.resumen)}</p>` : `<p class="s1-vacio">${e(t("bases_vacias"))}</p>`}`, "bases_vacias")}
      ${grupo("versiones", "versiones", versiones ? `<ul class="s1-lista">${versiones}</ul>` : "", "versiones_vacias")}
      ${grupo("requisitos", "requisitos", requisitos ? `<ul class="s1-lista">${requisitos}</ul>` : "", "requisitos_vacios")}
      ${grupo("hitos", "hitos", hitos ? `<ul class="s1-lista">${hitos}</ul>` : "", "hitos_vacios")}
      ${grupo("documentos", "documentos", documentos ? `<ul class="s1-lista">${documentos}</ul>` : "", "documentos_vacios")}
      <section class="panel"><div class="cabecera-panel"><h3>${e(t("acciones"))}</h3></div><div class="cuerpo-panel s1-acciones">${controlNoConectado("editar_bases")}${controlNoConectado("enviar_firma")}${controlNoConectado("publicar")}</div></section>
    </div>`;
  }
  function renderizar() {
    const cabecera = `<header class="s1-cabecera"><div><p class="sobrelinea">${e(t("sobrelinea"))}</p><h2>${e(t("titulo"))}</h2><p>${e(t("descripcion"))}</p></div><details class="s1-ayuda"><summary aria-label="${e(t("ayuda_aria"))}">?</summary><p>${e(t("ayuda_detalle"))}</p></details></header>`;
    if (estado.carga !== "disponible") return `<div class="consulta-convocatorias-s1">${cabecera}${aviso(estado.carga)}</div>`;
    if (estado.convocatorias.length === 0) return `<div class="consulta-convocatorias-s1">${cabecera}${aviso("vacio")}</div>`;
    return `<div class="consulta-convocatorias-s1">${cabecera}<div class="s1-rejilla"><section class="panel s1-listado"><div class="cabecera-panel"><div><h3>${e(t("listado"))}</h3><p>${e(t("cantidad", { numero: new Intl.NumberFormat("es-ES").format(estado.convocatorias.length) }))}</p></div><span class="estado-chip info">${e(t("solo_lectura"))}</span></div><ul class="s1-lista">${estado.convocatorias.map(ficha).join("")}</ul></section>${estado.cargaDetalle === "cargando" ? aviso("cargando_detalle") : estado.cargaDetalle === "no_configurado" ? aviso("no_configurado") : estado.cargaDetalle === "denegado" ? aviso("denegado") : estado.cargaDetalle === "error" ? aviso("error_detalle") : detalleHTML(estado.detalle)}</div></div>`;
  }
  function pintar() { if (activo && contenedor) contenedor.innerHTML = renderizar(); }
  async function cargarDetalle(referencia, { enfocarDetalle = false } = {}) {
    if (!activo || !estado.convocatorias.some((item) => item.referencia === referencia)) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++generacion;
    estado = { ...estado, seleccionada: referencia, detalle: null, cargaDetalle: "cargando" };
    pintar();
    if (enfocarDetalle) contenedor.querySelector?.(".s1-rejilla > .s1-estado")?.focus();
    if (typeof consultarDetalle !== "function") {
      estado = { ...estado, cargaDetalle: "no_configurado" };
      pintar();
      if (enfocarDetalle) contenedor.querySelector?.(".s1-rejilla > .s1-estado")?.focus();
      return;
    }
    try {
      const detalle = await consultarDetalle(referencia, { signal: controlador.signal });
      if (!activo || actual !== generacion) return;
      if (!validaDetalle(detalle, referencia)) throw new Error("contrato de detalle inválido");
      estado = { ...estado, detalle, cargaDetalle: "disponible" };
    } catch (error) {
      if (!activo || actual !== generacion) return;
      estado = { ...estado, detalle: null, cargaDetalle: error?.status === 401 || error?.status === 403 ? "denegado" : "error" };
    }
    pintar();
    if (enfocarDetalle) {
      contenedor.querySelector?.(estado.cargaDetalle === "disponible" ? "#s1-seccion-resumen" : ".s1-rejilla > .s1-estado")?.focus();
    }
  }
  function seleccionar(evento) {
    const boton = evento.target?.closest?.("[data-s1-convocatoria]");
    if (boton && contenedor?.contains(boton)) {
      void cargarDetalle(boton.dataset.s1Convocatoria, { enfocarDetalle: true });
      return;
    }
    const enlace = evento.target?.closest?.("[data-s1-seccion]");
    if (!enlace || !contenedor?.contains(enlace)) return;
    const seccion = apartados.find(([id]) => id === enlace.dataset.s1Seccion);
    if (!seccion) return;
    const destino = contenedor.querySelector?.(`#s1-seccion-${seccion[0]}`);
    destino?.scrollIntoView?.({ block: "start", behavior: "instant" });
    destino?.focus?.({ preventScroll: true });
  }
  async function montar() {
    if (!contenedor || typeof contenedor.addEventListener !== "function") throw new TypeError("contenedor S1 inválido");
    if (activo) return;
    activo = true;
    contenedor.addEventListener("click", seleccionar);
    if (typeof consultarLista !== "function") { pintar(); return; }
    estado = { carga: "cargando", convocatorias: [], detalle: null, seleccionada: "" };
    pintar();
    controlador = new AbortController();
    const actual = ++generacion;
    try {
      const convocatorias = await consultarLista({ signal: controlador.signal });
      if (!activo || actual !== generacion) return;
      if (!validaLista(convocatorias)) throw new Error("contrato de lista inválido");
      estado = { carga: "disponible", convocatorias, detalle: null, seleccionada: "" };
      pintar();
      if (convocatorias.length > 0) await cargarDetalle(convocatorias[0].referencia);
    } catch (error) {
      if (!activo || actual !== generacion) return;
      estado = { carga: error?.status === 401 || error?.status === 403 ? "denegado" : "error", convocatorias: [], detalle: null, seleccionada: "" };
      pintar();
    }
  }
  function desmontar() {
    if (!activo) return;
    activo = false;
    ++generacion;
    controlador?.abort();
    contenedor.removeEventListener("click", seleccionar);
    contenedor.replaceChildren();
  }
  return Object.freeze({ montar, desmontar, renderizar, cargarDetalle });
}

function escaparHTMLConvocatorias(valor) {
  return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
