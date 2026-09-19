/** Vistas compartidas de revisión de méritos, baremación y alegaciones. */

export function crearVistasBaremacion(u) {
  const { escaparHTML: e, numero, fecha, chip, tabla, kpi, encabezadoVista,
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

  function lista(datos, nombre) {
    return Array.isArray(datos?.[nombre]) ? datos[nombre] : [];
  }

  function estadoLectura(estado, vista) {
    const candidato = estado?.vistasBaremacion?.[vista];
    return candidato && typeof candidato === "object" ? candidato : { carga: "listo" };
  }

  function accionNoOperable(etiqueta, _operacion, _id, clase = "boton-secundario") {
    return `<button type="button" class="${e(clase)}" data-comando="${e(_operacion)}" data-objetivo="${e(_id)}" disabled aria-disabled="true" title="La presentación no crea efectos administrativos.">${e(etiqueta)}</button>`;
  }

  function accionPresentacion(datos, etiqueta, operacion, objetivo, clase = "boton-secundario") {
    return datos?.demostracion === true
      ? botonOperacion(etiqueta, operacion, objetivo, clase)
      : accionNoOperable(etiqueta, operacion, objetivo, clase);
  }

  function estadoNoDisponible(titulo, lectura, vacio) {
    if (lectura.carga === "cargando") {
      return `<section class="panel"><div class="cabecera-panel"><h3>${e(titulo)}</h3><span class="estado-chip neutro">Consultando…</span></div><div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true"><p><strong>Comprobando el acceso y cargando datos…</strong></p><p>La pantalla espera una respuesta autorizada; no muestra datos de presentación mientras se consulta.</p></div></section>`;
    }
    if (lectura.carga === "denegado") {
      return `<section class="panel"><div class="cabecera-panel"><h3>${e(titulo)}</h3><span class="estado-chip peligro">Acceso denegado</span></div><div class="cuerpo-panel vacio-controlado" role="alert"><p><strong>La sesión no dispone de acceso a esta consulta.</strong></p><p>No se muestran méritos, puntuaciones, clasificaciones ni alegaciones fuera del ámbito autorizado.</p></div></section>`;
    }
    if (lectura.carga === "error") {
      return `<section class="panel"><div class="cabecera-panel"><h3>${e(titulo)}</h3><span class="estado-chip peligro">Servicio no disponible</span></div><div class="cuerpo-panel vacio-controlado" role="alert"><p><strong>No se pudo consultar esta información.</strong></p><p>${e(lectura.error || "Revise la conexión con el servicio autorizado e inténtelo de nuevo.")}</p></div></section>`;
    }
    return vacio ? `<section class="panel"><div class="cabecera-panel"><h3>${e(titulo)}</h3><span class="estado-chip neutro">Sin registros</span></div><div class="cuerpo-panel vacio-controlado" role="status"><p><strong>No hay datos para el ámbito consultado.</strong></p><p>La ausencia de registros no autoriza crear decisiones, puntuaciones ni plazos.</p></div></section>` : "";
  }

  function renderizarMeritos(datos, estado = {}) {
    const lectura = estadoLectura(estado, "meritos");
    const meritosRevision = lista(datos, "meritos_revision");
    const bloqueado = estadoNoDisponible("Revisión de méritos", lectura, meritosRevision.length === 0);
    if (bloqueado) return bloqueado;
    const referencia = valorFiltro(estado, "referencia");
    const tipo = valorFiltro(estado, "tipo", "Todos");
    const estadoSeleccionado = valorFiltro(estado, "estado", "Todos");
    const meritos = meritosRevision.filter((item) => {
      const coincideReferencia = !referencia || [item.id, item.persona_ref, item.evidencia]
        .some((valor) => contiene(valor, referencia));
      const coincideTipo = tipo === "Todos" || item.tipo === tipo;
      const coincideEstado = estadoSeleccionado === "Todos" || item.estado === estadoSeleccionado;
      return coincideReferencia && coincideTipo && coincideEstado;
    });
    const filas = meritos.map((item) => [
      `<strong>${e(item.id)}</strong><br><small>${e(item.persona_ref)}</small>`, e(item.tipo), e(item.evidencia),
      e(item.declarado), `<strong>${e(item.puntos)}</strong>`, chip(item.estado),
      `<div class="acciones-fila">${accionPresentacion(datos, "Aceptar", "aceptar-merito", item.id, "boton-terciario")}${accionPresentacion(datos, "Rechazar", "rechazar-merito", item.id, "boton-terciario")}${accionPresentacion(datos, "Revocar", "revocar-merito", item.id, "boton-terciario")}${accionPresentacion(datos, "Rehabilitar", "rehabilitar-merito", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Control técnico", "Revisión de méritos y rectificaciones", "Cada aceptación, rechazo, revocación o rehabilitación exige motivo, actor, versión de criterio y evidencia.", accionPresentacion(datos, "Preparar informe de revisión", "exportar-informe", "DEMO-REV-MERITOS", "boton-secundario"))}
      ${avisoPresentacion("Los datos y decisiones son sintéticos y de solo lectura; esta pantalla no registra valoraciones administrativas.")}
      <div class="rejilla-kpi">${kpi("PEN", numero(meritosRevision.filter((x) => x.estado === "Pendiente").length), "Pendientes DEMO")}${kpi("ACE", numero(meritosRevision.filter((x) => /^Aceptado/.test(x.estado)).length), "Aceptados DEMO")}${kpi("REC", numero(meritosRevision.filter((x) => /Rechazado/.test(x.estado)).length), "Rechazados DEMO")}${kpi("REC", numero(meritosRevision.filter((x) => /revocada/i.test(x.estado)).length), "Rectificados DEMO")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Cola de evidencias</h3><p>Referencias seudonimizadas para reducir exposición durante la revisión masiva.</p></div>${fuentePresentacion()}</div>
        <form class="barra-filtros" aria-label="Filtros de méritos" data-filtro="meritos">${campo("Referencia", `<input type="search" name="referencia" value="${e(referencia)}" placeholder="DEMO-MER-…">`)}${campo("Tipo", `<select name="tipo">${["Todos", "Experiencia profesional", "Formación", "Titulación"].map((valor) => opcion(valor, tipo)).join("")}</select>`)}${campo("Estado", `<select name="estado">${["Todos", "Pendiente", "Aceptado", "Rechazado"].map((valor) => opcion(valor, estadoSeleccionado)).join("")}</select>`)}<button type="submit" class="boton-secundario">Aplicar filtros</button></form>
        <p class="resultado-filtro" role="status" data-total-filtro="meritos" data-total="${meritos.length}">${numero(meritos.length)} méritos encontrados.</p>
        ${tabla({ titulo: "Méritos pendientes y revisados", cabeceras: ["Mérito / persona", "Tipo", "Evidencia", "Declarado", "Puntos", "Estado", "Decisión"], filas })}
      </section>
      <div class="rejilla-dos-columnas panel-separado"><section class="panel"><div class="cabecera-panel"><h3>Motivación de la decisión</h3><span class="estado-chip violeta">Obligatoria</span></div><form class="cuerpo-panel formulario-gobernado" aria-label="Decisión motivada sobre DEMO-MER-001"><fieldset disabled><legend>Fundamento común de la decisión</legend><div class="rejilla-formulario">${campo("Criterio aplicado", '<select name="criterio"><option>DEMO-CRI-001 · experiencia</option><option>DEMO-CRI-003 · formación</option><option>DEMO-CRI-004 · titulación</option></select>')}${campo("Motivo tipificado", '<select name="motivo_tipificado"><option>Documentación suficiente</option><option>No relacionado con la convocatoria</option><option>Periodo no acreditado</option><option>Error material corregido</option></select>')}${campo("Observación técnica", '<textarea name="observacion">Texto sintético de motivación para la presentación.</textarea>')}</div></fieldset><div class="acciones-formulario">${accionPresentacion(datos, "Aceptar", "aceptar-merito", "DEMO-MER-001", "boton-primario")}${accionPresentacion(datos, "Rechazar", "rechazar-merito", "DEMO-MER-001")}${accionPresentacion(datos, "Revocar aceptación", "revocar-merito", "DEMO-MER-001")}${accionPresentacion(datos, "Rehabilitar rechazo", "rehabilitar-merito", "DEMO-MER-001")}</div></form></section><aside class="nota-seguridad"><strong>Rectificación preservada.</strong> Nunca se sobrescribe la decisión anterior: se incorpora un nuevo hecho firmado que indica qué cambia, por qué y desde cuándo.</aside></div>`;
  }

  function renderizarBaremacion(datos, estado = {}) {
    const lectura = estadoLectura(estado, "baremacion");
    const criterios = lista(datos, "criterios_baremo");
    const entradasRanking = lista(datos, "ranking");
    const bloqueado = estadoNoDisponible("Baremación y ranking", lectura, criterios.length === 0 && entradasRanking.length === 0);
    if (bloqueado) return bloqueado;
    const versionAplicada = criterios[0]?.version || "Sin versión disponible";
    const bloquesAplicados = new Set(criterios.map((item) => item.bloque)).size;
    const ranking = entradasRanking.map((item) => [e(item.posicion), e(item.persona_ref), e(item.experiencia), e(item.formacion), e(item.otros), `<strong>${e(item.total)}</strong>`, e(item.desempate), chip(item.estado)]);
    return `
      ${encabezadoVista("Ejecución del cálculo", "Baremación, ranking y listas", "Escenario sintético de consulta: no ejecuta reglas, no asigna puntos y no publica listas.", accionPresentacion(datos, "Calcular ranking DEMO", "calcular-baremo", "DEMO-BOL-014", "boton-primario"))}
      ${avisoPresentacion("Las cifras y el orden son sintéticos; no representan puntos, baremos aprobados ni una lista administrativa.")}
      <div class="rejilla-kpi">${kpi("VER", versionAplicada, "Versión aplicada")}${kpi("CRI", numero(criterios.length), "Criterios ejecutados")}${kpi("BLQ", numero(bloquesAplicados), "Bloques puntuados")}${kpi("ASP", numero(entradasRanking.length), "Aspirantes calculados")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Contexto de ejecución</h3><p>La configuración visible es sintética y no acredita una regla aprobada, cálculo ni huella administrativa.</p></div><span class="estado-chip violeta">DEMO-BOL-014 · ${e(versionAplicada)}</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Convocatoria</dt><dd>DEMO-BOL-014</dd></div><div class="fila-resumen"><dt>Entrada</dt><dd>Datos sintéticos de presentación; sin revisión administrativa acreditada</dd></div><div class="fila-resumen"><dt>Versión de reglas</dt><dd>${e(versionAplicada)} · solo lectura durante el cálculo</dd></div><div class="fila-resumen"><dt>Salida</dt><dd>Orden sintético sin efectos administrativos</dd></div></dl></div></section>
      <section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Clasificación sintética de presentación</h3><p>Los importes y el orden se muestran solo para agrupar el escenario DEMO.</p></div><div class="acciones-vista">${accionPresentacion(datos, "Publicar lista provisional", "publicar-lista-provisional", "DEMO-BOL-014", "boton-primario")}${accionPresentacion(datos, "Exportar", "exportar-informe", "DEMO-RAN-BOL-014")}</div></div>${tabla({ titulo: "Ranking provisional DEMO (sin efectos)", cabeceras: ["Posición", "Persona", "Experiencia", "Formación", "Otros", "Total", "Desempate", "Estado"], filas: ranking })}</section>`;
  }

  function renderizarAlegaciones(datos, estado = {}) {
    const lectura = estadoLectura(estado, "alegaciones");
    const alegaciones = lista(datos, "alegaciones");
    const bloqueado = estadoNoDisponible("Alegaciones y rectificaciones", lectura, alegaciones.length === 0);
    if (bloqueado) return bloqueado;
    const filas = alegaciones.map((item) => [
      `<strong>${e(item.id)}</strong>`, e(item.persona_ref), e(item.objeto), e(fecha(item.registrada)), e(fecha(item.plazo)), e(item.evidencia), chip(item.estado),
      `<div class="acciones-fila">${accionPresentacion(datos, "Estimar", "resolver-alegacion", item.id, "boton-terciario")}${accionPresentacion(datos, "Desestimar", "desestimar-alegacion", item.id, "boton-terciario")}</div>`,
    ]);
    return `
      ${encabezadoVista("Garantías del procedimiento", "Alegaciones y rectificaciones", "Recepción, estudio, resolución motivada, posible recálculo y comunicación a la persona interesada.", accionPresentacion(datos, "Preparar resolución conjunta", "generar-documento", "DEMO-ALE-LOTE-01", "boton-primario"))}
      ${avisoPresentacion("Las referencias y estados son sintéticos. No declaran plazos, registro, notificación ni resolución administrativa.")}
      <div class="rejilla-kpi">${kpi("TOT", numero(alegaciones.length), "Recibidas DEMO")}${kpi("PEN", numero(alegaciones.filter((x) => /pendiente|estudio/i.test(x.estado)).length), "Pendientes DEMO")}${kpi("EST", numero(alegaciones.filter((x) => /Estimada/.test(x.estado)).length), "Estimadas DEMO")}${kpi("DES", numero(alegaciones.filter((x) => /Desestimada/.test(x.estado)).length), "Desestimadas DEMO")}</div>
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
