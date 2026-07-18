/** Vistas compartidas de gobierno de convocatorias y admisión. */

export function crearVistasConvocatorias(u) {
  const { escaparHTML: e, numero, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, botonOperacion, campo, fuentePresentacion } = u;

  function renderizarConvocatorias(datos, estado) {
    const seleccionada = datos.elaboraciones.find((item) => item.id === estado.elaboracionSeleccionada)
      || datos.elaboraciones[0];
    const filas = datos.elaboraciones.map((item) => [
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
          <form class="barra-filtros" aria-label="Filtros de convocatorias">
            ${campo("Buscar", '<input type="search" value="" placeholder="Referencia o categoría">')}
            ${campo("Estado", '<select><option>Todos</option><option>Borrador</option><option>En revisión</option><option>Publicada</option></select>')}
            ${campo("Unidad responsable", '<select><option>Todas</option><option>Unidad DEMO de Selección</option></select>')}
            ${botonOperacion("Aplicar filtros", "exportar-informe", "DEMO-FILTRO-CONVOCATORIAS")}
          </form>
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
        <form class="cuerpo-panel formulario-gobernado" aria-label="Configuración de bases y calendario">
          <fieldset><legend>Identificación</legend><div class="rejilla-formulario">${campo("Denominación pública", `<input value="${e(seleccionada?.nombre || "")}">`)}${campo("Categoría profesional", '<select><option>Auxiliar Administrativo</option><option>Trabajador Social</option><option>Operario Servicios Múltiples</option></select>')}${campo("Código de expediente", `<input value="${e(seleccionada?.expediente || "")}" readonly>`)}${campo("Tipo de proceso", '<select><option>Bolsa de trabajo</option><option>Proceso selectivo</option></select>')}</div></fieldset>
          <fieldset><legend>Calendario gobernado</legend><div class="rejilla-formulario">${campo("Apertura de solicitudes", '<input type="datetime-local" value="2026-08-01T09:00">')}${campo("Cierre de solicitudes", '<input type="datetime-local" value="2026-08-20T23:59">')}${campo("Subsanación desde", '<input type="date" value="2026-08-25">')}${campo("Subsanación hasta", '<input type="date" value="2026-09-05">')}</div></fieldset>
          <fieldset><legend>Documentación y publicación</legend><div class="rejilla-formulario">${campo("Versión de bases", '<input value="v3">')}${campo("Medio de publicación", '<select><option>BOP + sede + portal</option><option>Sede + portal</option></select>')}${campo("Plantilla documental", '<select><option>DEMO-PLT-BASES-v4</option></select>')}${campo("Circuito de firma", '<select><option>FIR-DEMO-CONVOCATORIA-v2</option></select>')}</div></fieldset>
          <div class="acciones-formulario">${botonOperacion("Validar y guardar versión", "guardar-bases", seleccionada?.id || "DEMO-BOL-SIN-SELECCION", "boton-primario")}</div>
        </form>
      </section>`;
  }

  function renderizarSolicitudes(datos) {
    const pendientes = datos.solicitudes.filter((item) => /pendiente/i.test(item.estado)).length;
    const filas = datos.solicitudes.map((item) => [
      `<strong>${e(item.id)}</strong>`, e(item.persona_ref), e(item.convocatoria), e(item.registrada),
      e(item.requisitos), e(item.subsanacion), chip(item.estado),
      `<div class="acciones-fila">${botonOperacion("Admitir", "admitir-solicitud", item.id, "boton-terciario")}${botonOperacion("Excluir", "excluir-solicitud", item.id, "boton-terciario")}${botonOperacion("Subsanar", "registrar-subsanacion", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Bandeja de tramitación", "Solicitudes, admisión y subsanación", "Revisión de requisitos, listas provisionales y subsanaciones con motivación y trazabilidad.", botonOperacion("Publicar lista provisional", "publicar-lista-provisional", "DEMO-BOL-014", "boton-primario"))}
      ${avisoPresentacion()}
      <div class="rejilla-kpi">${kpi("REG", numero(datos.solicitudes.length), "Registradas")}${kpi("PEN", numero(pendientes), "Pendientes")}${kpi("SUB", numero(datos.solicitudes.filter((x) => /subsan/i.test(x.estado)).length), "En subsanación")}${kpi("ADM", numero(datos.solicitudes.filter((x) => /admitida/i.test(x.estado)).length), "Admitidas")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Bandeja de solicitudes</h3><p>No se muestran nombres, documentos de identidad ni datos de contacto en el listado.</p></div>${fuentePresentacion()}</div>
        <form class="barra-filtros" aria-label="Filtros de solicitudes">${campo("Buscar referencia", '<input type="search" placeholder="DEMO-SOL-…">')}${campo("Convocatoria", '<select><option>Todas</option><option>DEMO-BOL-014</option><option>DEMO-BOL-021</option></select>')}${campo("Estado", '<select><option>Todos</option><option>Pendiente de revisión</option><option>Pendiente de subsanación</option><option>Admitida provisional</option></select>')}${botonOperacion("Aplicar filtros", "exportar-informe", "DEMO-FILTRO-SOLICITUDES")}</form>
        ${tabla({ titulo: "Solicitudes presentadas", cabeceras: ["Solicitud", "Persona", "Convocatoria", "Registro", "Requisitos", "Subsanación", "Estado", "Acciones"], filas })}
      </section>
      <section class="nota-pendiente">Las exclusiones requieren causa tipificada y texto motivado. La demo muestra el cambio de estado, pero no genera resolución, asiento registral ni notificación fehaciente.</section>`;
  }

  return Object.freeze({ renderizarConvocatorias, renderizarSolicitudes });
}
