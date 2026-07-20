/** Vistas compartidas de revisión de méritos, baremación y alegaciones. */

export function crearVistasBaremacion(u) {
  const { escaparHTML: e, numero, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, botonOperacion, campo, fuentePresentacion } = u;

  function valorFiltro(estado, nombre, porDefecto = "") {
    return String(estado?.filtros?.meritos?.[nombre] ?? porDefecto);
  }

  function opcion(valor, seleccionada) {
    return `<option value="${e(valor)}"${valor === seleccionada ? " selected" : ""}>${e(valor)}</option>`;
  }

  function contiene(valor, busqueda) {
    return String(valor || "").toLocaleLowerCase("es").includes(String(busqueda || "").trim().toLocaleLowerCase("es"));
  }

  function renderizarMeritos(datos, estado = {}) {
    const referencia = valorFiltro(estado, "referencia");
    const tipo = valorFiltro(estado, "tipo", "Todos");
    const estadoSeleccionado = valorFiltro(estado, "estado", "Todos");
    const meritos = datos.meritos_revision.filter((item) => {
      const coincideReferencia = !referencia || [item.id, item.persona_ref, item.evidencia]
        .some((valor) => contiene(valor, referencia));
      const coincideTipo = tipo === "Todos" || item.tipo === tipo;
      const coincideEstado = estadoSeleccionado === "Todos" || item.estado === estadoSeleccionado;
      return coincideReferencia && coincideTipo && coincideEstado;
    });
    const filas = meritos.map((item) => [
      `<strong>${e(item.id)}</strong><br><small>${e(item.persona_ref)}</small>`, e(item.tipo), e(item.evidencia),
      e(item.declarado), `<strong>${e(item.puntos)}</strong>`, chip(item.estado),
      `<div class="acciones-fila">${botonOperacion("Aceptar", "aceptar-merito", item.id, "boton-terciario")}${botonOperacion("Rechazar", "rechazar-merito", item.id, "boton-terciario")}${botonOperacion("Revocar", "revocar-merito", item.id, "boton-terciario")}${botonOperacion("Rehabilitar", "rehabilitar-merito", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Control técnico", "Revisión de méritos y rectificaciones", "Cada aceptación, rechazo, revocación o rehabilitación exige motivo, actor, versión de criterio y evidencia.", botonOperacion("Preparar informe de revisión", "exportar-informe", "DEMO-REV-MERITOS", "boton-secundario"))}
      ${avisoPresentacion("Las cuatro decisiones se pueden recorrer. En producción deberán firmarse cuando la política del procedimiento lo exija.")}
      <div class="rejilla-kpi">${kpi("PEN", numero(datos.meritos_revision.filter((x) => x.estado === "Pendiente").length), "Pendientes")}${kpi("ACE", numero(datos.meritos_revision.filter((x) => /^Aceptado/.test(x.estado)).length), "Aceptados")}${kpi("REC", numero(datos.meritos_revision.filter((x) => /Rechazado/.test(x.estado)).length), "Rechazados")}${kpi("REC", numero(datos.meritos_revision.filter((x) => /revocada/i.test(x.estado)).length), "Rectificados")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Cola de evidencias</h3><p>Referencias seudonimizadas para reducir exposición durante la revisión masiva.</p></div>${fuentePresentacion()}</div>
        <form class="barra-filtros" aria-label="Filtros de méritos" data-filtro="meritos">${campo("Referencia", `<input type="search" name="referencia" value="${e(referencia)}" placeholder="DEMO-MER-…">`)}${campo("Tipo", `<select name="tipo">${["Todos", "Experiencia profesional", "Formación", "Titulación"].map((valor) => opcion(valor, tipo)).join("")}</select>`)}${campo("Estado", `<select name="estado">${["Todos", "Pendiente", "Aceptado", "Rechazado"].map((valor) => opcion(valor, estadoSeleccionado)).join("")}</select>`)}<button type="submit" class="boton-secundario">Aplicar filtros</button></form>
        <p class="resultado-filtro" role="status" data-total-filtro="meritos" data-total="${meritos.length}">${numero(meritos.length)} méritos encontrados.</p>
        ${tabla({ titulo: "Méritos pendientes y revisados", cabeceras: ["Mérito / persona", "Tipo", "Evidencia", "Declarado", "Puntos", "Estado", "Decisión"], filas })}
      </section>
      <div class="rejilla-dos-columnas panel-separado"><section class="panel"><div class="cabecera-panel"><h3>Motivación de la decisión</h3><span class="estado-chip violeta">Obligatoria</span></div><form class="cuerpo-panel formulario-gobernado" aria-label="Decisión motivada sobre DEMO-MER-001"><fieldset><legend>Fundamento común de la decisión</legend><div class="rejilla-formulario">${campo("Criterio aplicado", '<select name="criterio"><option>DEMO-CRI-001 · experiencia</option><option>DEMO-CRI-003 · formación</option><option>DEMO-CRI-004 · titulación</option></select>')}${campo("Motivo tipificado", '<select name="motivo_tipificado"><option>Documentación suficiente</option><option>No relacionado con la convocatoria</option><option>Periodo no acreditado</option><option>Error material corregido</option></select>')}${campo("Observación técnica", '<textarea name="observacion">Texto sintético de motivación para la presentación.</textarea>')}</div></fieldset><div class="acciones-formulario">${botonOperacion("Aceptar", "aceptar-merito", "DEMO-MER-001", "boton-primario")}${botonOperacion("Rechazar", "rechazar-merito", "DEMO-MER-001")}${botonOperacion("Revocar aceptación", "revocar-merito", "DEMO-MER-001")}${botonOperacion("Rehabilitar rechazo", "rehabilitar-merito", "DEMO-MER-001")}</div></form></section><aside class="nota-seguridad"><strong>Rectificación preservada.</strong> Nunca se sobrescribe la decisión anterior: se incorpora un nuevo hecho firmado que indica qué cambia, por qué y desde cuándo.</aside></div>`;
  }

  function renderizarBaremacion(datos) {
    const criterios = datos.criterios_baremo;
    const versionAplicada = criterios[0]?.version || "Sin versión disponible";
    const bloquesAplicados = new Set(criterios.map((item) => item.bloque)).size;
    const ranking = datos.ranking.map((item) => [e(item.posicion), e(item.persona_ref), e(item.experiencia), e(item.formacion), e(item.otros), `<strong>${e(item.total)}</strong>`, e(item.desempate), chip(item.estado)]);
    return `
      ${encabezadoVista("Ejecución del cálculo", "Baremación, ranking y listas", "Aplicación reproducible de una versión aprobada de reglas sobre los méritos revisados, con desglose y desempates explicables.", botonOperacion("Calcular ranking DEMO", "calcular-baremo", "DEMO-BOL-014", "boton-primario"))}
      ${avisoPresentacion("El cálculo visible es sintético y explicable. No sustituye el cálculo reproducible y firmado del núcleo.")}
      <div class="rejilla-kpi">${kpi("VER", versionAplicada, "Versión aplicada")}${kpi("CRI", numero(criterios.length), "Criterios ejecutados")}${kpi("BLQ", numero(bloquesAplicados), "Bloques puntuados")}${kpi("ASP", numero(datos.ranking.length), "Aspirantes calculados")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Contexto de ejecución</h3><p>La configuración se administra en «Reglas y versiones»; aquí solo se aplica una versión aprobada y se conserva su huella con el resultado.</p></div><span class="estado-chip violeta">DEMO-BOL-014 · ${e(versionAplicada)}</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Convocatoria</dt><dd>DEMO-BOL-014</dd></div><div class="fila-resumen"><dt>Entrada</dt><dd>Autobaremaciones con revisión técnica completada</dd></div><div class="fila-resumen"><dt>Versión de reglas</dt><dd>${e(versionAplicada)} · solo lectura durante el cálculo</dd></div><div class="fila-resumen"><dt>Salida</dt><dd>Clasificación provisional versionada y explicable</dd></div></dl></div></section>
      <section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Clasificación provisional explicable</h3><p>La puntuación total procede de cada bloque y conserva la regla exacta aplicada.</p></div><div class="acciones-vista">${botonOperacion("Publicar lista provisional", "publicar-lista-provisional", "DEMO-BOL-014", "boton-primario")}${botonOperacion("Exportar", "exportar-informe", "DEMO-RAN-BOL-014")}</div></div>${tabla({ titulo: "Ranking provisional", cabeceras: ["Posición", "Persona", "Experiencia", "Formación", "Otros", "Total", "Desempate", "Estado"], filas: ranking })}</section>`;
  }

  function renderizarAlegaciones(datos) {
    const filas = datos.alegaciones.map((item) => [
      `<strong>${e(item.id)}</strong>`, e(item.persona_ref), e(item.objeto), e(item.registrada), e(item.plazo), e(item.evidencia), chip(item.estado),
      `<div class="acciones-fila">${botonOperacion("Estimar", "resolver-alegacion", item.id, "boton-terciario")}${botonOperacion("Desestimar", "desestimar-alegacion", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Garantías del procedimiento", "Alegaciones y rectificaciones", "Recepción, estudio, resolución motivada, posible recálculo y comunicación a la persona interesada.", botonOperacion("Preparar resolución conjunta", "generar-documento", "DEMO-ALE-LOTE-01", "boton-primario"))}
      ${avisoPresentacion()}
      <div class="rejilla-kpi">${kpi("TOT", numero(datos.alegaciones.length), "Recibidas")}${kpi("PEN", numero(datos.alegaciones.filter((x) => /pendiente|estudio/i.test(x.estado)).length), "Pendientes")}${kpi("EST", numero(datos.alegaciones.filter((x) => /Estimada/.test(x.estado)).length), "Estimadas")}${kpi("DES", numero(datos.alegaciones.filter((x) => /Desestimada/.test(x.estado)).length), "Desestimadas")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Bandeja de alegaciones</h3><p>El acceso al escrito y sus documentos requiere finalidad y expediente asignado.</p></div>${fuentePresentacion()}</div>${tabla({
        titulo: "Alegaciones registradas",
        cabeceras: ["Alegación", "Persona", "Objeto", "Registro", "Plazo", "Evidencia", "Estado", "Resolución"],
        clavesColumnas: ["referencia", "persona", "objeto", "registro", "plazo", "evidencia", "estado", "acciones"],
        prioridadColumnas: "estado-acciones",
        filas,
      })}</section>
      <section class="nota-pendiente">Una estimación que afecte al baremo desencadena un nuevo cálculo versionado; nunca altera silenciosamente la lista ya publicada.</section>`;
  }

  return Object.freeze({ renderizarAlegaciones, renderizarBaremacion, renderizarMeritos });
}
