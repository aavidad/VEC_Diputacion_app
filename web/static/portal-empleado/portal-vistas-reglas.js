/** Vista de gobierno, versionado y configuración del motor de reglas. */

export function crearVistaReglas(u) {
  const { escaparHTML: e, numero, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, botonOperacion, campo, fuentePresentacion } = u;

  function renderizarFormularioConfiguracion(objetivoBorrador) {
    return `
      <form class="cuerpo-panel formulario-gobernado"
        aria-label="Configuración de una nueva versión de reglas"
        data-comando="guardar-reglas-baremo">
        <fieldset>
          <legend>Puntuación por experiencia</legend>
          <div class="rejilla-formulario">
            ${campo("Unidad de tiempo", '<select name="unidad_tiempo"><option>Mes completo</option><option>Día</option><option>Año</option></select>')}
            ${campo("Puntos por unidad", '<input type="number" name="puntos_unidad" value="0.10" min="0" step="0.01">')}
            ${campo("Fracción de jornada", '<select name="fraccion_jornada"><option>Prorratear por porcentaje</option><option>No prorratear si las bases lo permiten</option></select>')}
            ${campo("Tope del bloque", '<input type="number" name="tope_bloque" value="6.00" min="0" step="0.01">')}
            ${campo("Ámbito de experiencia", '<select name="ambito_experiencia"><option>Administración convocante</option><option>Otra administración</option><option>Sector privado equivalente</option></select>')}
            ${campo("Regla de redondeo", '<select name="redondeo"><option>Sin redondeo intermedio</option><option>Dos decimales al final</option></select>')}
          </div>
        </fieldset>
        <fieldset>
          <legend>Desempates ordenados</legend>
          <div class="rejilla-formulario">
            ${campo("1.º criterio", '<select name="desempate_1"><option>Mayor puntuación en experiencia</option></select>')}
            ${campo("2.º criterio", '<select name="desempate_2"><option>Mayor puntuación en formación</option></select>')}
            ${campo("3.º criterio", '<select name="desempate_3"><option>Fecha y hora de registro</option></select>')}
            ${campo("Último recurso", '<select name="ultimo_recurso"><option>Sorteo trazado</option></select>')}
          </div>
        </fieldset>
        <div class="acciones-formulario">
          ${botonOperacion("Guardar borrador de versión", "guardar-reglas-baremo", objetivoBorrador, "boton-primario")}
        </div>
      </form>`;
  }

  function renderizarReglas(datos) {
    const reglas = Array.isArray(datos.reglas) ? datos.reglas : [];
    const criterios = Array.isArray(datos.criterios_baremo) ? datos.criterios_baremo : [];
    const publicadas = reglas.filter((item) => /publicada/i.test(item.estado)).length;
    const enValidacion = reglas.filter((item) => /validación|revisión/i.test(item.estado)).length;
    const reglaActiva = reglas.find((item) => /publicada/i.test(item.estado)) || reglas[0] || {};
    const criterioActivo = criterios.find((item) => item.version === reglaActiva.version) || criterios[0] || {};
    const objetivoBorrador = criterioActivo.id || reglaActiva.version || "regla-no-seleccionada";
    const contextoCriterios = [reglaActiva.ambito, criterioActivo.version || reglaActiva.version]
      .filter(Boolean).join(" · ") || "Sin versión activa";
    const versiones = reglas.map((item) => [
      `<strong>${e(item.nombre)}</strong>`, e(item.ambito), e(item.version), e(item.vigencia),
      chip(item.estado),
      `<button type="button" class="boton-terciario" data-accion="detalle-regla" data-objetivo="${e(item.version)}">Abrir versión</button>`,
    ]);
    const configuracion = criterios.map((item) => [
      `<strong>${e(item.id)}</strong>`, e(item.bloque), e(item.criterio), e(item.formula),
      e(item.maximo), e(item.version), chip(item.estado),
    ]);

    return `
      ${encabezadoVista("Gobierno del motor", "Reglas, versiones y configuración", "Configuración sin recompilar de criterios derivados de las bases, con vigencia, validación y sustitución sin modificar versiones publicadas.")}
      ${avisoPresentacion("La configuración y las versiones visibles son sintéticas. Guardar genera una actuación DEMO volátil y nunca modifica una regla publicada.")}
      <div class="rejilla-kpi">${kpi("VER", numero(reglas.length), "Versiones")}${kpi("PUB", numero(publicadas), "Publicadas")}${kpi("VAL", numero(enValidacion), "En validación")}${kpi("CRI", numero(criterios.length), "Criterios configurados")}</div>
      <div class="rejilla-dos-columnas rejilla-dos-columnas--tabla-densa">
        <section class="panel">
          <div class="cabecera-panel"><div><h3>Registro de versiones</h3><p>Cada versión conserva su ámbito, vigencia y estado de aprobación.</p></div>${fuentePresentacion()}</div>
          ${tabla({
            titulo: "Versiones gobernadas del motor de reglas",
            cabeceras: ["Regla", "Ámbito", "Versión", "Vigencia", "Estado", "Detalle"],
            clavesColumnas: ["regla", "ambito", "version", "vigencia", "estado", "acciones"],
            prioridadColumnas: "estado-acciones",
            filas: versiones,
          })}
        </section>
        <aside class="resumen-lateral">
          <section class="panel"><div class="cabecera-panel"><h3>Ciclo de gobierno</h3><span class="estado-chip violeta">Controlado</span></div><ol class="lista-comprobacion"><li>Borrador vinculado a bases y fuente jurídica</li><li>Pruebas con casos ordinarios y límite</li><li>Validación técnica y jurídica</li><li>Aprobación, firma y fecha de vigencia</li><li>Sustitución sin borrar el historial</li></ol></section>
          <section class="nota-seguridad"><strong>Inmutabilidad funcional.</strong> Los expedientes y cálculos guardan la versión exacta aplicada; una corrección crea una versión sucesora.</section>
        </aside>
      </div>
      <section class="panel panel-separado">
        <div class="cabecera-panel"><div><h3>Catálogo de criterios de la versión activa</h3><p>Vista de control previa a cualquier cálculo de baremación.</p></div><span class="estado-chip violeta">${e(contextoCriterios)}</span></div>
        ${tabla({
          titulo: "Configuración vigente de criterios",
          cabeceras: ["Criterio", "Bloque", "Descripción", "Fórmula", "Máximo", "Versión", "Estado"],
          clavesColumnas: ["referencia", "bloque", "descripcion", "formula", "maximo", "version", "estado"],
          prioridadColumnas: "estado",
          filas: configuracion,
        })}
      </section>
      <section class="panel panel-separado">
        <div class="cabecera-panel"><div><h3>Preparar la siguiente versión</h3><p>Los cambios quedan en borrador hasta superar el circuito de validación y aprobación.</p></div>${fuentePresentacion()}</div>
        ${renderizarFormularioConfiguracion(objetivoBorrador)}
      </section>`;
  }

  return Object.freeze({ renderizarReglas });
}
