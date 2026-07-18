/** Vistas compartidas de estadísticas, auditoría, configuración y roles. */

export function crearVistasGobierno(u) {
  const { escaparHTML: e, numero, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, botonOperacion, botonBloqueado, campo, fuentePresentacion } = u;

  function renderizarEstadisticas(datos) {
    const indicadores = datos.indicadores;
    const filas = datos.bolsas.map((item) => [e(item.nombre), e(item.categoria), numero(item.integrantes), numero(item.disponibles), numero(item.llamamiento), `${numero(item.cobertura, 1)} %`, chip(item.estado)]);
    return `
      ${encabezadoVista("Información agregada", "Estadísticas y exportación", "Indicadores operativos sin datos personales y exportaciones sujetas a permiso y finalidad.", botonOperacion("Preparar informe", "exportar-informe", "DEMO-INF-ESTADISTICAS", "boton-primario"))}
      ${avisoPresentacion("La acción genera un recibo, no un archivo. Los renderizadores de formato se conectarán sin modificar esta pantalla.")}
      <div class="rejilla-kpi">${kpi("COB", `${numero(indicadores.cobertura_media, 1)} %`, "Cobertura media")}${kpi("TME", indicadores.tiempo_medio_cobertura, "Tiempo de cobertura")}${kpi("REN", `${numero(indicadores.renuncias_porcentaje, 1)} %`, "Renuncias")}${kpi("RES", `${numero(indicadores.respuesta_mediana_horas)} h`, "Respuesta mediana")}${kpi("BOL", numero(indicadores.bolsas_activas), "Bolsas activas")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Cobertura por Bolsa</h3><p>Tabla accesible equivalente a cualquier visualización gráfica.</p></div>${fuentePresentacion()}</div>${tabla({ titulo: "Cobertura por Bolsa", cabeceras: ["Bolsa", "Categoría", "Integrantes", "Disponibles", "En llamamiento", "Cobertura", "Estado"], filas })}</section>
      <section class="panel panel-separado"><div class="cabecera-panel"><h3>Definir exportación</h3><span class="estado-chip violeta">Finalidad obligatoria</span></div><form class="cuerpo-panel formulario-gobernado" data-comando="exportar-informe"><fieldset><legend>Contenido y formato</legend><div class="rejilla-formulario">${campo("Informe", '<select name="informe"><option>Cobertura y tiempos</option><option>Actividad de llamamientos</option><option>Estado de convocatorias</option></select>')}${campo("Ámbito", '<select name="ambito"><option>Datos agregados</option><option>Expedientes autorizados</option></select>')}${campo("Formato", '<select name="formato"><option>PDF/A</option><option>CSV</option><option>ODS</option><option>JSON</option></select>')}${campo("Finalidad", '<select name="finalidad"><option>Seguimiento de gestión</option><option>Transparencia</option><option>Auditoría</option></select>')}</div></fieldset>${botonOperacion("Preparar exportación DEMO", "exportar-informe", "DEMO-INF-ESTADISTICAS", "boton-primario")}</form></section>`;
  }

  function renderizarAuditoria(datos) {
    const filas = datos.auditoria_eventos.map((item) => [e(item.referencia), e(item.instante), e(item.actor), e(item.operacion), e(item.objetivo), chip(item.resultado), item.efectos_reales === false ? '<span class="estado-chip info">Sin efectos reales</span>' : '<span class="estado-chip peligro">No permitido</span>']);
    return `
      ${encabezadoVista("Hechos reconstruibles", "Auditoría y trazabilidad", "Actor, instante, expediente, operación, decisión, regla, evidencia y resultado de cada actuación.", botonOperacion("Preparar paquete de auditoría", "exportar-informe", "DEMO-AUD-PAQUETE", "boton-primario"))}
      ${avisoPresentacion("Los recibos nuevos aparecen al ejecutar cualquier operación. Se pierden íntegramente al recargar la página.")}
      <div class="rejilla-kpi">${kpi("EVT", numero(datos.auditoria_eventos.length), "Eventos visibles")}${kpi("ACT", numero(new Set(datos.auditoria_eventos.map((x) => x.actor)).size), "Actores sintéticos")}${kpi("REA", "0", "Efectos reales")}${kpi("ERR", "0", "Eventos sin recibo")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Registro de actuaciones DEMO</h3><p>Orden inverso de incorporación; referencias inequívocamente sintéticas.</p></div>${fuentePresentacion()}</div>${tabla({ titulo: "Eventos de auditoría", cabeceras: ["Recibo", "Instante UTC", "Actor", "Operación", "Objetivo", "Resultado", "Alcance"], filas })}</section>
      <section class="nota-seguridad"><strong>Producción.</strong> El navegador no será la fuente de verdad de auditoría. La API devolverá recibos firmados o protegidos contra alteración y registrará también lecturas y exportaciones.</section>`;
  }

  function renderizarConfiguracion(datos) {
    const roles = datos.roles_demo.map((item) => [e(item.id), e(item.nombre), e(item.ambito), e(item.permisos), e(item.segregacion), chip(item.estado), botonOperacion("Revisar", "revisar-permisos", item.id, "boton-terciario")]);
    const configuraciones = datos.configuraciones_demo.map((item) => [e(item.id), e(item.parametro), e(item.valor), e(item.version), chip(item.estado), botonOperacion("Nueva versión", "guardar-configuracion", item.id, "boton-terciario")]);
    return `
      ${encabezadoVista("Mínimo privilegio", "Configuración, roles y permisos", "Catálogos, calendarios, reglas, ámbitos y autorización configurables sin recompilar el núcleo.", botonOperacion("Proponer rol", "crear-rol", "DEMO-ROL-NUEVO", "boton-primario"))}
      ${avisoPresentacion("No se modifica ninguna identidad ni autorización. Los roles visibles son perfiles sintéticos de la matriz prevista.")}
      <section class="nota-seguridad"><strong>Política por defecto: denegar.</strong> Un permiso solo existe si está concedido explícitamente para acción, recurso, ámbito y finalidad.</section>
      <section class="panel"><div class="cabecera-panel"><div><h3>Matriz de roles</h3><p>Separación entre preparación, revisión, firma, publicación y auditoría.</p></div>${fuentePresentacion()}</div>${tabla({ titulo: "Roles de gestión de Bolsa", cabeceras: ["Rol", "Nombre", "Ámbito", "Permisos", "Segregación", "Estado", "Acción"], filas: roles })}</section>
      <section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Configuración gobernada</h3><p>Cada cambio crea una versión con vigencia, motivo y prueba.</p></div>${botonOperacion("Guardar configuración", "guardar-configuracion", "DEMO-CFG-NUEVA", "boton-primario")}</div>${tabla({ titulo: "Configuraciones de Bolsa", cabeceras: ["Referencia", "Parámetro", "Valor", "Versión", "Estado", "Acción"], filas: configuraciones })}</section>
      <div class="rejilla-dos-columnas panel-separado"><section class="panel"><div class="cabecera-panel"><h3>Propuesta de rol</h3><span class="estado-chip violeta">Sin activar</span></div><form class="cuerpo-panel formulario-gobernado" data-comando="crear-rol"><fieldset><legend>Alcance mínimo</legend><div class="rejilla-formulario">${campo("Nombre", '<input name="nombre" value="Revisor DEMO de admisión">')}${campo("Unidad", '<select name="unidad"><option>Unidad DEMO de Selección</option></select>')}${campo("Recursos", '<select name="recursos"><option>Convocatorias asignadas</option><option>Expedientes asignados</option></select>')}${campo("Acciones", '<select name="acciones"><option>Consultar y proponer</option><option>Solo consultar</option></select>')}${campo("Vigencia desde", '<input type="date" name="vigencia_desde" value="2026-07-21">')}${campo("Revisión obligatoria", '<select name="revision"><option>90 días</option><option>180 días</option></select>')}</div></fieldset>${botonOperacion("Guardar propuesta DEMO", "crear-rol", "DEMO-ROL-NUEVO", "boton-primario")}</form></section><aside class="nota-pendiente">${botonBloqueado("Activar rol en directorio", "La demo no dispone de identidad corporativa, Kerberos ni aprobación de la persona responsable de seguridad.")}</aside></div>`;
  }

  return Object.freeze({ renderizarAuditoria, renderizarConfiguracion, renderizarEstadisticas });
}
