/** Vistas compartidas de revisión de méritos, baremación y alegaciones. */

export function crearVistasBaremacion(u) {
  const { escaparHTML: e, numero, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, botonOperacion, campo, fuentePresentacion } = u;

  function renderizarMeritos(datos) {
    const filas = datos.meritos_revision.map((item) => [
      `<strong>${e(item.id)}</strong><br><small>${e(item.persona_ref)}</small>`, e(item.tipo), e(item.evidencia),
      e(item.declarado), `<strong>${e(item.puntos)}</strong>`, chip(item.estado),
      `<div class="acciones-fila">${botonOperacion("Aceptar", "aceptar-merito", item.id, "boton-terciario")}${botonOperacion("Rechazar", "rechazar-merito", item.id, "boton-terciario")}${botonOperacion("Revocar", "revocar-merito", item.id, "boton-terciario")}${botonOperacion("Rehabilitar", "rehabilitar-merito", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Control técnico", "Revisión de méritos y rectificaciones", "Cada aceptación, rechazo, revocación o rehabilitación exige motivo, actor, versión de criterio y evidencia.", botonOperacion("Preparar informe de revisión", "exportar-informe", "REV-DEMO-MERITOS", "boton-secundario"))}
      ${avisoPresentacion("Las cuatro decisiones se pueden recorrer. En producción deberán firmarse cuando la política del procedimiento lo exija.")}
      <div class="rejilla-kpi">${kpi("PEN", numero(datos.meritos_revision.filter((x) => x.estado === "Pendiente").length), "Pendientes")}${kpi("ACE", numero(datos.meritos_revision.filter((x) => /^Aceptado/.test(x.estado)).length), "Aceptados")}${kpi("REC", numero(datos.meritos_revision.filter((x) => /Rechazado/.test(x.estado)).length), "Rechazados")}${kpi("REC", numero(datos.meritos_revision.filter((x) => /revocada/i.test(x.estado)).length), "Rectificados")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Cola de evidencias</h3><p>Referencias seudonimizadas para reducir exposición durante la revisión masiva.</p></div>${fuentePresentacion()}</div>
        <form class="barra-filtros" aria-label="Filtros de méritos">${campo("Referencia", '<input type="search" placeholder="DEMO-MER-…">')}${campo("Tipo", '<select><option>Todos</option><option>Experiencia profesional</option><option>Formación</option><option>Titulación</option></select>')}${campo("Estado", '<select><option>Todos</option><option>Pendiente</option><option>Aceptado</option><option>Rechazado</option></select>')}${botonOperacion("Aplicar filtros", "exportar-informe", "DEMO-FILTRO-MERITOS")}</form>
        ${tabla({ titulo: "Méritos pendientes y revisados", cabeceras: ["Mérito / persona", "Tipo", "Evidencia", "Declarado", "Puntos", "Estado", "Decisión"], filas })}
      </section>
      <div class="rejilla-dos-columnas panel-separado"><section class="panel"><div class="cabecera-panel"><h3>Motivación de la decisión</h3><span class="estado-chip violeta">Obligatoria</span></div><form class="cuerpo-panel formulario-gobernado"><fieldset><legend>Fundamento</legend><div class="rejilla-formulario">${campo("Criterio aplicado", '<select><option>DEMO-CRI-001 · experiencia</option><option>DEMO-CRI-003 · formación</option><option>DEMO-CRI-004 · titulación</option></select>')}${campo("Resultado", '<select><option>Aceptar</option><option>Rechazar</option><option>Revocar aceptación</option><option>Rehabilitar rechazo</option></select>')}${campo("Motivo tipificado", '<select><option>Documentación suficiente</option><option>No relacionado con la convocatoria</option><option>Periodo no acreditado</option><option>Error material corregido</option></select>')}${campo("Observación técnica", '<textarea>Texto sintético de motivación para la presentación.</textarea>')}</div></fieldset>${botonOperacion("Firmar decisión DEMO", "aceptar-merito", "DEMO-MER-001", "boton-primario")}</form></section><aside class="nota-seguridad"><strong>Rectificación preservada.</strong> Nunca se sobrescribe la decisión anterior: se incorpora un nuevo hecho firmado que indica qué cambia, por qué y desde cuándo.</aside></div>`;
  }

  function renderizarBaremacion(datos) {
    const criterios = datos.criterios_baremo.map((item) => [e(item.id), e(item.bloque), e(item.criterio), e(item.formula), e(item.maximo), e(item.version), chip(item.estado)]);
    const ranking = datos.ranking.map((item) => [e(item.posicion), e(item.persona_ref), e(item.experiencia), e(item.formacion), e(item.otros), `<strong>${e(item.total)}</strong>`, e(item.desempate), chip(item.estado)]);
    return `
      ${encabezadoVista("Motor configurable", "Baremación, ranking, desempates y listas", "Las reglas se definen desde las bases, se versionan y se aplican a la autobaremación revisada.", botonOperacion("Calcular ranking DEMO", "calcular-baremo", "DEMO-BOL-014", "boton-primario"))}
      ${avisoPresentacion("El cálculo visible es sintético y explicable. No sustituye el cálculo reproducible y firmado del núcleo.")}
      <section class="panel"><div class="cabecera-panel"><div><h3>Criterios de la convocatoria</h3><p>Coeficientes, jornada, topes, redondeo, fuentes y vigencia configurables.</p></div><span class="estado-chip violeta">DEMO-BOL-014 · reglas v3</span></div>${tabla({ titulo: "Criterios de baremación", cabeceras: ["Criterio", "Bloque", "Descripción", "Fórmula", "Máximo", "Versión", "Estado"], filas: criterios })}</section>
      <section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Configurar criterio</h3><p>Formulario operativo que se conectará al catálogo gobernado de reglas.</p></div>${fuentePresentacion()}</div><form class="cuerpo-panel formulario-gobernado"><fieldset><legend>Puntuación por experiencia</legend><div class="rejilla-formulario">${campo("Unidad de tiempo", '<select><option>Mes completo</option><option>Día</option><option>Año</option></select>')}${campo("Puntos por unidad", '<input type="number" value="0.10" min="0" step="0.01">')}${campo("Fracción de jornada", '<select><option>Prorratear por porcentaje</option><option>No prorratear si las bases lo permiten</option></select>')}${campo("Tope del bloque", '<input type="number" value="6.00" min="0" step="0.01">')}${campo("Ámbito de experiencia", '<select><option>Administración convocante</option><option>Otra administración</option><option>Sector privado equivalente</option></select>')}${campo("Regla de redondeo", '<select><option>Sin redondeo intermedio</option><option>Dos decimales al final</option></select>')}</div></fieldset><fieldset><legend>Desempates ordenados</legend><div class="rejilla-formulario">${campo("1.º criterio", '<select><option>Mayor puntuación en experiencia</option></select>')}${campo("2.º criterio", '<select><option>Mayor puntuación en formación</option></select>')}${campo("3.º criterio", '<select><option>Fecha y hora de registro</option></select>')}${campo("Último recurso", '<select><option>Sorteo trazado</option></select>')}</div></fieldset>${botonOperacion("Guardar versión de reglas", "guardar-bases", "DEMO-BOL-014", "boton-primario")}</form></section>
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
      <section class="panel"><div class="cabecera-panel"><div><h3>Bandeja de alegaciones</h3><p>El acceso al escrito y sus documentos requiere finalidad y expediente asignado.</p></div>${fuentePresentacion()}</div>${tabla({ titulo: "Alegaciones registradas", cabeceras: ["Alegación", "Persona", "Objeto", "Registro", "Plazo", "Evidencia", "Estado", "Resolución"], filas })}</section>
      <section class="nota-pendiente">Una estimación que afecte al baremo desencadena un nuevo cálculo versionado; nunca altera silenciosamente la lista ya publicada.</section>`;
  }

  return Object.freeze({ renderizarAlegaciones, renderizarBaremacion, renderizarMeritos });
}
