/**
 * Asistente visual de llamamientos.
 *
 * La propuesta de personas siempre procede del puerto específico del servidor
 * (o de su adaptador sintético aislado). Esta vista solo conoce posiciones,
 * resultado y motivación; nunca recibe identidad, contacto ni documentos.
 */

const CAMPOS_CONFIGURABLES = new Set([
  "destino", "jornada", "duracion", "plazo_respuesta", "canales", "plantilla",
]);

function escaparLocal(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function necesidadSeleccionada(datos, estado) {
  return datos.necesidades_llamamiento.find((item) => item.id === estado.necesidadSeleccionada)
    || datos.necesidades_llamamiento[0]
    || null;
}

function opcionesCanales(datos) {
  const disponibles = Array.isArray(datos.configuracion_llamamiento?.canales)
    ? datos.configuracion_llamamiento.canales.filter((item) => typeof item === "string" && item.trim() !== "")
    : [];
  if (disponibles.length === 0) return [];
  const opciones = [disponibles.join(" + ")];
  if (disponibles.length > 1) opciones.push([disponibles[0], disponibles.at(-1)].join(" + "));
  return [...new Set(opciones)];
}

function opcionesSelect(valores, seleccionada, escaparHTML) {
  return valores.map((valor) => `<option value="${escaparHTML(valor)}"${valor === seleccionada ? " selected" : ""}>${escaparHTML(valor)}</option>`).join("");
}

function errorDeCampo(errores, campo) {
  return errores.find((error) => error.campo === campo)?.mensaje || "";
}

function atributosError(errores, campo) {
  return errorDeCampo(errores, campo) ? ' aria-invalid="true" aria-describedby="errores-configuracion-llamamiento"' : "";
}

export function crearConfiguracionInicialLlamamiento(datos, estado) {
  const necesidad = necesidadSeleccionada(datos, estado);
  if (!necesidad) return null;
  const base = datos.configuracion_llamamiento || {};
  return Object.freeze({
    destino: String(necesidad.destino || base.destino || ""),
    jornada: String(necesidad.jornada || base.jornada || ""),
    duracion: String(necesidad.duracion || base.duracion || ""),
    plazo_respuesta: String(base.plazo_respuesta || datos.catalogos_llamamiento?.plazos_respuesta?.[0] || ""),
    canales: opcionesCanales(datos)[0] || "",
    plantilla: "DEMO-PLT-LLA-v3",
  });
}

export function actualizarConfiguracionLlamamiento(configuracion, campo, valor) {
  if (!CAMPOS_CONFIGURABLES.has(campo) || typeof valor !== "string" || valor.length > 500
    || /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(valor)) {
    throw new TypeError("campo de configuración de llamamiento no válido");
  }
  return Object.freeze({ ...configuracion, [campo]: valor });
}

export function validarConfiguracionLlamamiento(datos, estado) {
  const errores = [];
  const necesidad = necesidadSeleccionada(datos, estado);
  const propuesta = estado.propuestaLlamamiento;
  const configuracion = estado.configuracionLlamamiento;
  if (!necesidad) errores.push({ campo: "necesidad", mensaje: "Seleccione una necesidad de cobertura." });
  if (propuesta?.demostracion !== true || propuesta.necesidad_id !== necesidad?.id) {
    errores.push({ campo: "propuesta", mensaje: "Solicite una propuesta DEMO válida para la necesidad seleccionada." });
  }
  if (!configuracion || typeof configuracion !== "object") {
    errores.push({ campo: "configuracion", mensaje: "La configuración todavía no se ha iniciado." });
    return Object.freeze(errores);
  }
  for (const campo of ["destino", "duracion", "plazo_respuesta", "jornada", "canales", "plantilla"]) {
    const valor = configuracion[campo];
    if (typeof valor !== "string" || valor.trim() === "" || valor.length > 500) {
      errores.push({ campo, mensaje: `Complete correctamente el campo ${campo.replaceAll("_", " ")}.` });
    }
  }
  const catalogos = datos.catalogos_llamamiento || {};
  if (!catalogos.plazos_respuesta?.includes(configuracion.plazo_respuesta)) {
    errores.push({ campo: "plazo_respuesta", mensaje: "Seleccione un plazo de respuesta del catálogo vigente." });
  }
  if (!catalogos.jornadas?.includes(configuracion.jornada)) {
    errores.push({ campo: "jornada", mensaje: "Seleccione una jornada del catálogo vigente." });
  }
  if (!opcionesCanales(datos).includes(configuracion.canales)) {
    errores.push({ campo: "canales", mensaje: "Seleccione una combinación de canales disponible." });
  }
  if (configuracion.plantilla !== "DEMO-PLT-LLA-v3") {
    errores.push({ campo: "plantilla", mensaje: "La plantilla no pertenece al catálogo de la presentación." });
  }
  return Object.freeze(errores.map((error) => Object.freeze(error)));
}

export function prepararOperacionLlamamiento(datos, estado) {
  const errores = validarConfiguracionLlamamiento(datos, estado);
  if (errores.length > 0) return Object.freeze({ ok: false, errores });
  const necesidad = necesidadSeleccionada(datos, estado);
  const propuesta = estado.propuestaLlamamiento;
  const configuracion = estado.configuracionLlamamiento;
  const existente = datos.llamamientos_demo.find((item) => item.necesidad === necesidad.id);
  return Object.freeze({
    ok: true,
    objetivo: existente?.id || `DEMO-LLA-WIZ-${necesidad.id.replace(/^DEMO-NEC-/, "")}`,
    campos: Object.freeze({
      bolsa: String(necesidad.bolsa),
      destino: configuracion.destino.trim(),
      jornada: configuracion.jornada,
      duracion: configuracion.duracion.trim(),
      regla: String(propuesta.version_regla),
      plazo_respuesta: configuracion.plazo_respuesta,
      canales: configuracion.canales,
      plantilla: configuracion.plantilla,
    }),
  });
}

export function renderizarPasosLlamamiento({ modoPresentacion, pasoActual, pasoMaximo = pasoActual }) {
  const nombres = modoPresentacion
    ? ["Seleccionar bolsa", "Seleccionar candidatos · DEMO automático", "Configurar llamamiento", "Revisar y preparar DEMO"]
    : ["Seleccionar bolsa", "Revisar confirmación", "Configurar llamamiento", "Revisar y preparar"];
  const actualSeguro = Math.max(1, Math.min(modoPresentacion ? 4 : 2, pasoActual));
  return `<nav class="pasos" aria-label="Pasos del nuevo llamamiento">${nombres.map((nombre, indice) => {
    const paso = indice + 1;
    const clase = paso < actualSeguro ? "completado" : "";
    const actual = paso === actualSeguro ? ' aria-current="step"' : "";
    const bloqueado = (!modoPresentacion && paso > 2) || paso > pasoMaximo ? " disabled" : "";
    return `<button type="button" class="paso ${clase}" data-accion="ir-paso" data-paso="${paso}"${actual}${bloqueado}><span class="paso-numero">${paso < actualSeguro ? "✓" : paso}</span><span>${escaparLocal(nombre)}</span></button>`;
  }).join("")}</nav>`;
}

export function renderizarConfirmacionCompacta(confirmacion) {
  return `<div class="cuerpo-panel"><dl class="resumen-expediente">
    <div class="fila-resumen"><dt>Propuesta</dt><dd><code>${escaparLocal(confirmacion.propuesta_ref)}</code></dd></div>
    <div class="fila-resumen"><dt>Necesidad</dt><dd><code>${escaparLocal(confirmacion.necesidad.referencia)}</code> · v${escaparLocal(confirmacion.necesidad.version)}</dd></div>
    <div class="fila-resumen"><dt>Bolsa</dt><dd><code>${escaparLocal(confirmacion.bolsa.referencia)}</code> · v${escaparLocal(confirmacion.bolsa.version)}</dd></div>
    <div class="fila-resumen"><dt>Instantánea</dt><dd><code>${escaparLocal(confirmacion.instantanea.referencia)}</code> · v${escaparLocal(confirmacion.instantanea.version)}</dd></div>
    <div class="fila-resumen"><dt>Política</dt><dd><code>${escaparLocal(confirmacion.politica.referencia)}</code> · v${escaparLocal(confirmacion.politica.version)}</dd></div>
    <div class="fila-resumen"><dt>Generada</dt><dd>${escaparLocal(confirmacion.generada_en)}</dd></div>
    <div class="fila-resumen"><dt>Evaluaciones</dt><dd>${escaparLocal(confirmacion.total_evaluaciones)}</dd></div>
    <div class="fila-resumen"><dt>Orden seleccionado</dt><dd>${escaparLocal(confirmacion.orden_seleccionado)}</dd></div>
  </dl><div class="nota-pendiente" role="status"><strong>Detalle no disponible</strong><br>La respuesta real no contiene evaluaciones ni datos personales. Los pasos de configuración permanecen bloqueados.</div></div>`;
}

export function renderizarDetalleLlamamientoBloqueado() {
  return '<section class="panel"><div class="cabecera-panel"><h3>Detalle no disponible</h3><span class="estado-chip">Bloqueado</span></div><div class="vacio-controlado">La confirmación real no contiene detalle y no habilita la configuración del llamamiento.</div><div class="cuerpo-panel"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button></div></section>';
}

export function crearAsistenteLlamamientos(utilidades) {
  const {
    avisoPresentacion, encabezadoVista, escaparHTML, fuentePresentacion, numero, operacionPermitida,
  } = utilidades;
  if ([avisoPresentacion, encabezadoVista, escaparHTML, fuentePresentacion, numero, operacionPermitida]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias del asistente de llamamientos no válidas");
  }

  function reiniciar(estado, necesidadId = "") {
    estado.pasoLlamamiento = 1;
    estado.necesidadSeleccionada = necesidadId;
    estado.propuestaLlamamiento = null;
    estado.confirmacionPropuestaLlamamiento = null;
    estado.configuracionLlamamiento = null;
    estado.erroresConfiguracionLlamamiento = [];
    estado.reciboLlamamiento = null;
  }

  function actualizarCampo(estado, campo, valor) {
    estado.configuracionLlamamiento = actualizarConfiguracionLlamamiento(
      estado.configuracionLlamamiento || {}, campo, valor,
    );
    estado.erroresConfiguracionLlamamiento = [];
    estado.reciboLlamamiento = null;
  }

  function avanzar(datos, estado) {
    if (estado.pasoLlamamiento === 2) {
      if (estado.propuestaLlamamiento?.demostracion !== true
        || estado.propuestaLlamamiento.necesidad_id !== necesidadSeleccionada(datos, estado)?.id
        || Number(estado.propuestaLlamamiento.personas_incluidas) < 1) {
        return Object.freeze({ ok: false, mensaje: "La propuesta DEMO no contiene una selección elegible válida." });
      }
      estado.configuracionLlamamiento ||= crearConfiguracionInicialLlamamiento(datos, estado);
      estado.pasoLlamamiento = 3;
      return Object.freeze({ ok: true, mensaje: "Propuesta DEMO revisada; configure el llamamiento." });
    }
    if (estado.pasoLlamamiento === 3) {
      const errores = validarConfiguracionLlamamiento(datos, estado);
      estado.erroresConfiguracionLlamamiento = errores;
      if (errores.length > 0) return Object.freeze({ ok: false, mensaje: errores[0].mensaje });
      estado.pasoLlamamiento = 4;
      return Object.freeze({ ok: true, mensaje: "Configuración DEMO validada; revise antes de preparar." });
    }
    return Object.freeze({ ok: false, mensaje: "Complete primero el paso actual." });
  }

  function renderizarBandeja(datos) {
    const filas = datos.llamamientos_demo.map((item) => `<tr><td><code>${escaparHTML(item.id)}</code></td><td>${escaparHTML(item.necesidad)}</td><td>${escaparHTML(item.bolsa)}</td><td>${escaparHTML(item.orden)}</td><td>${numero(item.incluidos)}</td><td>${escaparHTML(item.plazo)}</td><td><span class="estado-chip">${escaparHTML(item.estado)}</span></td></tr>`).join("");
    return `<section class="panel panel-separado bandeja-llamamientos"><div class="cabecera-panel"><div><h3>Expedientes de llamamiento</h3><p>Bandeja sintética de seguimiento; la preparación se realiza únicamente desde el asistente.</p></div>${fuentePresentacion()}</div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Llamamientos de la presentación</caption><thead><tr><th scope="col">Llamamiento</th><th scope="col">Necesidad</th><th scope="col">Bolsa</th><th scope="col">Regla</th><th scope="col">Incluidos</th><th scope="col">Plazo</th><th scope="col">Estado</th></tr></thead><tbody>${filas || '<tr><td colspan="7" class="vacio-controlado">No hay llamamientos visibles.</td></tr>'}</tbody></table></div></section>`;
  }

  function renderizarPasoUno(datos, estado) {
    const necesidad = necesidadSeleccionada(datos, estado);
    const puedeSolicitar = estado.modoPresentacion
      || datos.capacidades?.solicitar_propuesta_llamamiento === true;
    const filas = datos.necesidades_llamamiento.map((item) => `<tr aria-selected="${item.id === necesidad?.id}"><td><button type="button" class="boton-secundario" data-accion="seleccionar-necesidad" data-id="${escaparHTML(item.id)}">${item.id === necesidad?.id ? "Seleccionada" : "Seleccionar"}</button></td><td><strong>${escaparHTML(item.bolsa)}</strong><small class="dato-secundario">${escaparHTML(item.puesto)}</small></td><td><code>${escaparHTML(item.referencia)}</code></td><td>${escaparHTML(item.destino)}</td><td>${escaparHTML(item.jornada)}</td><td>${escaparHTML(item.cobertura)}</td><td>${escaparHTML(item.fecha_limite)}</td></tr>`).join("");
    return `<div class="distribucion-llamamiento"><section class="panel"><div class="cabecera-panel"><div><h3>1. Seleccionar bolsa</h3><p>La necesidad autorizada determina la bolsa, el destino y la regla aplicable.</p></div><span class="estado-chip info">${numero(datos.necesidades_llamamiento.length)} necesidades</span></div><div class="tabla-contenedor"><table class="tabla-datos tabla-seleccion-necesidad"><caption>Bolsas con necesidad de cobertura en la presentación</caption><thead><tr><th scope="col">Selección</th><th scope="col">Bolsa y puesto</th><th scope="col">Necesidad</th><th scope="col">Destino</th><th scope="col">Jornada</th><th scope="col">Cobertura</th><th scope="col">Fecha límite</th></tr></thead><tbody>${filas || '<tr><td colspan="7" class="vacio-controlado">No existe una necesidad accesible para iniciar el llamamiento.</td></tr>'}</tbody></table></div></section><aside class="resumen-lateral"><section class="panel resumen-necesidad"><div class="cabecera-panel"><h3>Resumen de selección</h3></div>${necesidad ? `<div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Bolsa</dt><dd>${escaparHTML(necesidad.bolsa)}</dd></div><div class="fila-resumen"><dt>Destino</dt><dd>${escaparHTML(necesidad.destino)}</dd></div><div class="fila-resumen"><dt>Jornada</dt><dd>${escaparHTML(necesidad.jornada)}</dd></div><div class="fila-resumen"><dt>Duración</dt><dd>${escaparHTML(necesidad.duracion)}</dd></div><div class="fila-resumen"><dt>Regla</dt><dd>${escaparHTML(necesidad.regla)}</dd></div></dl><button type="button" class="boton-primario boton-ancho" data-accion="solicitar-propuesta"${puedeSolicitar ? "" : ' disabled aria-disabled="true" title="El servidor no concede la capacidad de solicitar propuesta"'}>${estado.modoPresentacion ? "Solicitar selección automática DEMO" : "Solicitar confirmación al servidor"} →</button></div>` : '<div class="vacio-controlado">Seleccione una necesidad.</div>'}</section><section class="nota-seguridad"><strong>Mínimo privilegio.</strong> La pantalla no consulta una lista global de personas. El puerto de propuestas evalúa la necesidad concreta.</section></aside></div>`;
  }

  function renderizarPasoDos(datos, estado) {
    if (estado.confirmacionPropuestaLlamamiento && !estado.modoPresentacion) {
      return `<section class="panel"><div class="cabecera-panel"><h3>2. Confirmación del servidor</h3><span class="estado-chip">Sin detalle</span></div>${renderizarConfirmacionCompacta(estado.confirmacionPropuestaLlamamiento)}</section>`;
    }
    const propuesta = estado.propuestaLlamamiento;
    if (propuesta?.demostracion !== true) return renderizarDetalleLlamamientoBloqueado();
    const filas = propuesta.evaluaciones.map((item) => `<tr><td><strong>Posición ${escaparHTML(item.orden)}</strong></td><td><span class="estado-chip ${item.resultado === "elegible" ? "exito" : "peligro"}">${item.resultado === "elegible" ? "Elegible" : "No elegible"}</span></td><td>${item.motivos.map((motivo) => escaparHTML(motivo.regla)).join("<br>")}</td><td>${item.motivos.map((motivo) => escaparHTML(motivo.fundamento)).join("<br>")}</td><td>${item.resultado === "elegible" ? '<strong>Incluida por el motor</strong>' : "No incluida"}</td></tr>`).join("");
    return `<div class="distribucion-llamamiento"><section class="panel"><div class="cabecera-panel"><div><h3>2. Seleccionar candidatos</h3><p>Selección automática DEMO según prelación; no permite elegir ni identificar personas manualmente.</p></div><span class="estado-chip violeta">Datos minimizados</span></div><div class="tabla-contenedor"><table class="tabla-datos tabla-propuesta-minimizada"><caption>Evaluaciones sintéticas sin identidad ni contacto</caption><thead><tr><th scope="col">Orden</th><th scope="col">Resultado</th><th scope="col">Regla</th><th scope="col">Fundamento</th><th scope="col">Selección</th></tr></thead><tbody>${filas}</tbody></table></div></section><aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Propuesta DEMO</h3></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Referencia</dt><dd><code>${escaparHTML(propuesta.id)}</code></dd></div><div class="fila-resumen"><dt>Bolsa</dt><dd>${escaparHTML(propuesta.version_bolsa)}</dd></div><div class="fila-resumen"><dt>Regla</dt><dd>${escaparHTML(propuesta.version_regla)}</dd></div><div class="fila-resumen"><dt>Fecha de corte</dt><dd><time datetime="${escaparHTML(propuesta.fecha_corte)}">${escaparHTML(propuesta.fecha_corte)}</time></dd></div><div class="fila-resumen"><dt>Incluidas</dt><dd>${numero(propuesta.personas_incluidas)} posiciones</dd></div></dl><div class="acciones-paso"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button><button type="button" class="boton-primario" data-accion="siguiente-paso">Aceptar propuesta DEMO →</button></div></div></section><section class="nota-seguridad"><strong>Sin identidad.</strong> La referencia de una posición no autoriza a consultar datos personales ni de contacto.</section></aside></div>`;
  }

  function renderizarPasoTres(datos, estado) {
    const necesidad = necesidadSeleccionada(datos, estado);
    const configuracion = estado.configuracionLlamamiento || crearConfiguracionInicialLlamamiento(datos, estado) || {};
    const errores = Array.isArray(estado.erroresConfiguracionLlamamiento) ? estado.erroresConfiguracionLlamamiento : [];
    const resumenErrores = errores.length > 0 ? `<div class="resumen-errores" id="errores-configuracion-llamamiento" role="alert" tabindex="-1"><strong>Revise la configuración</strong><ul>${errores.map((error) => `<li>${escaparHTML(error.mensaje)}</li>`).join("")}</ul></div>` : "";
    return `<div class="distribucion-llamamiento"><section class="panel"><div class="cabecera-panel"><div><h3>3. Configurar llamamiento</h3><p>${escaparHTML(necesidad?.bolsa || "Bolsa no disponible")} · ${numero(estado.propuestaLlamamiento?.personas_incluidas)} posición propuesta</p></div><span class="estado-chip info">Borrador volátil</span></div><form class="cuerpo-panel formulario-llamamiento" id="configuracion-llamamiento" data-formulario-llamamiento aria-describedby="marca-modo-llamamiento" novalidate>${resumenErrores}<label class="campo"><span>Plazo para responder</span><select name="plazo_respuesta" data-llamamiento-campo="plazo_respuesta" required${atributosError(errores, "plazo_respuesta")}>${opcionesSelect(datos.catalogos_llamamiento.plazos_respuesta || [], configuracion.plazo_respuesta, escaparHTML)}</select></label><label class="campo"><span>Centro o destino</span><input name="destino" data-llamamiento-campo="destino" value="${escaparHTML(configuracion.destino)}" maxlength="160" required${atributosError(errores, "destino")}></label><label class="campo"><span>Jornada</span><select name="jornada" data-llamamiento-campo="jornada" required${atributosError(errores, "jornada")}>${opcionesSelect(datos.catalogos_llamamiento.jornadas || [], configuracion.jornada, escaparHTML)}</select></label><label class="campo"><span>Duración prevista</span><input name="duracion" data-llamamiento-campo="duracion" value="${escaparHTML(configuracion.duracion)}" maxlength="80" required${atributosError(errores, "duracion")}></label><label class="campo campo-ancho"><span>Canales DEMO</span><select name="canales" data-llamamiento-campo="canales" required${atributosError(errores, "canales")}>${opcionesSelect(opcionesCanales(datos), configuracion.canales, escaparHTML)}</select><small>No se invoca ningún conector ni se obtiene información de contacto.</small></label><label class="campo campo-ancho"><span>Plantilla versionada</span><input name="plantilla" data-llamamiento-campo="plantilla" value="${escaparHTML(configuracion.plantilla)}" readonly required${atributosError(errores, "plantilla")}></label><div class="acciones-paso campo-ancho"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button><button type="button" class="boton-primario" data-accion="siguiente-paso">Revisar preparación DEMO →</button></div></form></section><aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Condiciones de la necesidad</h3></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Cobertura</dt><dd>${escaparHTML(necesidad?.cobertura || "—")}</dd></div><div class="fila-resumen"><dt>Apertura prevista</dt><dd>${escaparHTML(datos.configuracion_llamamiento.apertura_visible || "—")}</dd></div><div class="fila-resumen"><dt>Regla</dt><dd>${escaparHTML(estado.propuestaLlamamiento?.version_regla || "—")}</dd></div><div class="fila-resumen"><dt>Acuse</dt><dd>No generado en DEMO</dd></div></dl></div></section><section class="nota-pendiente"><strong>Solo presentación.</strong> Estos cambios permanecen en esta instancia del navegador y desaparecen al recargar.</section></aside></div>`;
  }

  function renderizarRecibo(recibo) {
    if (!recibo) return "";
    return `<section class="recibo-llamamiento" role="status" tabindex="-1" data-recibo-llamamiento><div><p class="sobrelinea">Preparación DEMO confirmada</p><h3>Recibo volátil del recorrido</h3></div><dl><div><dt>Recibo</dt><dd><code>${escaparHTML(recibo.referencia)}</code></dd></div><div><dt>Actor resuelto</dt><dd><code>${escaparHTML(recibo.actor)}</code></dd></div><div><dt>Instante</dt><dd><time datetime="${escaparHTML(recibo.instante)}">${escaparHTML(recibo.instante)}</time></dd></div><div><dt>Objetivo</dt><dd><code>${escaparHTML(recibo.objetivo)}</code></dd></div><div><dt>Resultado</dt><dd>${escaparHTML(recibo.resultado)}</dd></div><div><dt>Efectos reales</dt><dd><strong>No</strong></dd></div></dl><p class="nota-seguridad"><strong>No se ha enviado ninguna comunicación.</strong> El recibo demuestra únicamente la simulación y desaparece al recargar la página.</p></section>`;
  }

  function renderizarPasoCuatro(datos, estado) {
    const necesidad = necesidadSeleccionada(datos, estado);
    const configuracion = estado.configuracionLlamamiento || {};
    const permitido = estado.modoPresentacion && estado.propuestaLlamamiento?.demostracion === true
      && operacionPermitida("emitir-llamamiento");
    return `<div class="distribucion-llamamiento"><section class="panel"><div class="cabecera-panel"><div><h3>4. Revisar y preparar DEMO</h3><p>Último control antes de simular la preparación; nunca contacta a una persona.</p></div><span class="estado-chip ${estado.reciboLlamamiento ? "exito" : ""}">${estado.reciboLlamamiento ? "Preparado en DEMO" : "Pendiente de confirmación"}</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Necesidad</dt><dd><code>${escaparHTML(necesidad?.referencia || "—")}</code></dd></div><div class="fila-resumen"><dt>Bolsa</dt><dd>${escaparHTML(necesidad?.bolsa || "—")}</dd></div><div class="fila-resumen"><dt>Selección</dt><dd>${numero(estado.propuestaLlamamiento?.personas_incluidas)} posición propuesta automáticamente, sin identidad</dd></div><div class="fila-resumen"><dt>Versiones</dt><dd>${escaparHTML(estado.propuestaLlamamiento?.version_bolsa || "—")} · ${escaparHTML(estado.propuestaLlamamiento?.version_regla || "—")}</dd></div><div class="fila-resumen"><dt>Destino</dt><dd>${escaparHTML(configuracion.destino || "—")}</dd></div><div class="fila-resumen"><dt>Jornada y duración</dt><dd>${escaparHTML(configuracion.jornada || "—")} · ${escaparHTML(configuracion.duracion || "—")}</dd></div><div class="fila-resumen"><dt>Respuesta</dt><dd>${escaparHTML(configuracion.plazo_respuesta || "—")}</dd></div><div class="fila-resumen"><dt>Canales</dt><dd>${escaparHTML(configuracion.canales || "—")} · desconectados</dd></div><div class="fila-resumen"><dt>Plantilla</dt><dd><code>${escaparHTML(configuracion.plantilla || "—")}</code></dd></div></dl><div class="nota-pendiente"><strong>Confirmación de presentación.</strong> La acción prepara un registro sintético y un recibo en memoria. No persiste, no firma, no registra y no envía.</div><div class="acciones-paso"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button><button type="button" class="boton-primario" data-accion="preparar-llamamiento-demo"${permitido ? "" : ' disabled aria-disabled="true" title="El perfil o el modo activo no permite esta simulación"'}>Preparar DEMO · sin enviar</button></div></div>${renderizarRecibo(estado.reciboLlamamiento)}</section><aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Control previo</h3></div><ul class="lista-comprobacion"><li>Necesidad y bolsa identificadas</li><li>Propuesta ligada a versiones</li><li>Solo posiciones y motivos minimizados</li><li>Configuración validada</li><li>Permiso DEMO revalidado al confirmar</li><li class="pendiente">Sin persistencia ni conectores reales</li></ul></section><section class="nota-seguridad"><strong>Producción.</strong> El servidor deberá revalidar autorización, elegibilidad y versiones dentro de una única operación auditada.</section></aside></div>`;
  }

  function renderizar(datos, estado) {
    const paso = Math.max(1, Math.min(4, Number(estado.pasoLlamamiento) || 1));
    const titulo = paso === 1 ? "Nuevo llamamiento" : `Nuevo llamamiento · Paso ${paso}`;
    const cuerpo = [renderizarPasoUno, renderizarPasoDos, renderizarPasoTres, renderizarPasoCuatro][paso - 1](datos, estado);
    const marcaModo = estado.modoPresentacion
      ? "<strong>MODO DEMO PERSISTENTE</strong> · memoria volátil · ningún efecto real"
      : "<strong>MODO REAL</strong> · capacidades concedidas exclusivamente por el servidor";
    return `${encabezadoVista("Llamamientos según bases y Reglamento", titulo, "Asistente de cuatro pasos con propuesta reproducible, datos minimizados y recibo de presentación.", '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro de mando</button>')}${avisoPresentacion("Las necesidades, posiciones y actuaciones internas son sintéticas. No se accede a identidad ni contacto y no se envía ninguna comunicación.")}${renderizarPasosLlamamiento({ modoPresentacion: estado.modoPresentacion, pasoActual: paso, pasoMaximo: paso })}<section class="estado-asistente" aria-live="polite" id="marca-modo-llamamiento"><span><strong>Paso ${paso} de 4.</strong> ${["Seleccione una bolsa y su necesidad.", "Revise la selección automática minimizada.", "Configure plazo, destino y canales DEMO.", "Confirme la preparación sin envío real."][paso - 1]}</span><span>${marcaModo}</span></section>${cuerpo}${renderizarBandeja(datos)}`;
  }

  return Object.freeze({ actualizarCampo, avanzar, prepararOperacion: prepararOperacionLlamamiento, reiniciar, renderizar });
}
