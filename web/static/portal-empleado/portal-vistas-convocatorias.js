/** Vistas compartidas de gobierno de convocatorias y admisión. */

export function crearVistasConvocatorias(u) {
  const { escaparHTML: e, numero, chip, tabla, kpi, encabezadoVista,
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

  function renderizarConvocatorias(datos, estado) {
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
      e(item.fase), e(item.reglas), e(item.plazo), chip(item.estado),
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
          <p class="resultado-filtro" role="status" data-total-filtro="convocatorias" data-total="${elaboraciones.length}">${numero(elaboraciones.length)} convocatorias encontradas.</p>
          ${tabla({ titulo: "Convocatorias de Bolsa", cabeceras: ["Referencia", "Categoría / expediente", "Fase", "Baremo", "Plazo", "Estado"], filas })}
        </section>
        <aside class="resumen-lateral">
          <section class="panel"><div class="cabecera-panel"><div><h3>${e(seleccionada?.id || "Sin selección")}</h3><p>${e(seleccionada?.nombre || "No hay expedientes")}</p></div>${seleccionada ? chip(seleccionada.estado) : ""}</div>
            <div class="cuerpo-panel"><dl class="resumen-expediente">
              <div class="fila-resumen"><dt>Expediente</dt><dd>${e(seleccionada?.expediente || "—")}</dd></div>
              <div class="fila-resumen"><dt>Bases</dt><dd>${e(seleccionada?.version_bases || "—")}</dd></div>
              <div class="fila-resumen"><dt>Calendario</dt><dd>${e(seleccionada?.calendario || "—")}</dd></div>
              <div class="fila-resumen"><dt>Firmantes</dt><dd>${e(seleccionada?.firmantes || "—")}</dd></div>
              <div class="fila-resumen"><dt>Responsable</dt><dd>${e(seleccionada?.responsable || "—")}</dd></div>
            </dl><div class="acciones-verticales">
              ${botonOperacion("Guardar bases y baremo", "guardar-bases", seleccionada?.id || "DEMO-BOL-SIN-SELECCION", "boton-primario")}
              ${botonOperacion("Enviar al circuito de firma", "enviar-firma-convocatoria", seleccionada?.id || "DEMO-BOL-SIN-SELECCION")}
              ${botonOperacion("Publicar convocatoria", "publicar-convocatoria", seleccionada?.id || "DEMO-BOL-SIN-SELECCION")}
            </div></div>
          </section>
        </aside>
      </div>
      <section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Formulario de bases y calendario</h3><p>Los valores proceden de las bases aprobadas; nunca quedan fijados en código.</p></div><span class="estado-chip violeta">Versión v3 DEMO</span></div>
        <form class="cuerpo-panel formulario-gobernado" aria-label="Configuración de bases y calendario" data-comando="guardar-bases">
          <fieldset><legend>Identificación</legend><div class="rejilla-formulario">${campo("Denominación pública", `<input name="denominacion" value="${e(seleccionada?.nombre || "")}">`)}${campo("Categoría profesional", '<select name="categoria"><option>Auxiliar Administrativo</option><option>Trabajador Social</option><option>Operario Servicios Múltiples</option></select>')}${campo("Código de expediente", `<input name="expediente" value="${e(seleccionada?.expediente || "")}" readonly>`)}${campo("Tipo de proceso", '<select name="tipo_proceso"><option>Bolsa de trabajo</option><option>Proceso selectivo</option></select>')}</div></fieldset>
          <fieldset><legend>Calendario gobernado</legend><div class="rejilla-formulario">${campo("Apertura de solicitudes", '<input type="datetime-local" name="apertura" value="2026-08-01T09:00">')}${campo("Cierre de solicitudes", '<input type="datetime-local" name="cierre" value="2026-08-20T23:59">')}${campo("Subsanación desde", '<input type="date" name="subsanacion_desde" value="2026-08-25">')}${campo("Subsanación hasta", '<input type="date" name="subsanacion_hasta" value="2026-09-05">')}</div></fieldset>
          <fieldset><legend>Documentación y publicación</legend><div class="rejilla-formulario">${campo("Versión de bases", '<input name="version_bases" value="v3">')}${campo("Medio de publicación", '<select name="medio_publicacion"><option>BOP + sede + portal</option><option>Sede + portal</option></select>')}${campo("Plantilla documental", '<select name="plantilla"><option>DEMO-PLT-BASES-v4</option></select>')}${campo("Circuito de firma", '<select name="circuito_firma"><option>DEMO-FIR-CONVOCATORIA-v2</option></select>')}</div></fieldset>
          <div class="acciones-formulario">${botonOperacion("Validar y guardar versión", "guardar-bases", seleccionada?.id || "DEMO-BOL-SIN-SELECCION", "boton-primario")}</div>
        </form>
      </section>`;
  }

  function renderizarSolicitudes(datos, estado = {}) {
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
      `<strong>${e(item.id)}</strong>`, e(item.persona_ref), e(item.convocatoria), e(item.registrada),
      e(item.requisitos), e(item.subsanacion), chip(item.estado),
      `<div class="acciones-fila">${botonOperacion("Admitir", "admitir-solicitud", item.id, "boton-terciario")}${botonOperacion("Excluir", "excluir-solicitud", item.id, "boton-terciario")}${botonOperacion("Subsanar", "registrar-subsanacion", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Bandeja de tramitación", "Solicitudes, admisión y subsanación", "Revisión de requisitos, listas provisionales y subsanaciones con motivación y trazabilidad.", botonOperacion("Publicar lista provisional", "publicar-lista-provisional", "DEMO-BOL-014", "boton-primario"))}
      ${avisoPresentacion()}
      <div class="rejilla-kpi">${kpi("REG", numero(datos.solicitudes.length), "Registradas")}${kpi("PEN", numero(pendientes), "Pendientes")}${kpi("SUB", numero(datos.solicitudes.filter((x) => /subsan/i.test(x.estado)).length), "En subsanación")}${kpi("ADM", numero(datos.solicitudes.filter((x) => /admitida/i.test(x.estado)).length), "Admitidas")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Bandeja de solicitudes</h3><p>No se muestran nombres, documentos de identidad ni datos de contacto en el listado.</p></div>${fuentePresentacion()}</div>
        <form class="barra-filtros" aria-label="Filtros de solicitudes" data-filtro="solicitudes">${campo("Buscar referencia", `<input type="search" name="referencia" value="${e(referencia)}" placeholder="DEMO-SOL-…">`)}${campo("Convocatoria", `<select name="convocatoria">${["Todas", "DEMO-BOL-014", "DEMO-BOL-021"].map((valor) => opcion(valor, convocatoria)).join("")}</select>`)}${campo("Estado", `<select name="estado">${["Todos", "Pendiente de revisión", "Pendiente de subsanación", "Admitida provisional"].map((valor) => opcion(valor, estadoSeleccionado)).join("")}</select>`)}<button type="submit" class="boton-secundario">Aplicar filtros</button></form>
        <p class="resultado-filtro" role="status" data-total-filtro="solicitudes" data-total="${solicitudes.length}">${numero(solicitudes.length)} solicitudes encontradas.</p>
        ${tabla({ titulo: "Solicitudes presentadas", cabeceras: ["Solicitud", "Persona", "Convocatoria", "Registro", "Requisitos", "Subsanación", "Estado", "Acciones"], filas })}
      </section>
      <section class="nota-pendiente">Las exclusiones requieren causa tipificada y texto motivado. La demo muestra el cambio de estado, pero no genera resolución, asiento registral ni notificación fehaciente.</section>`;
  }

  return Object.freeze({ renderizarConvocatorias, renderizarSolicitudes });
}
