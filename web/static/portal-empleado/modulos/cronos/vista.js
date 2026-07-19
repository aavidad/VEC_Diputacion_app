import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_SOLICITAR_PERMISO,
  exigirContextoActorCronos,
  tieneCapacidadCronos,
  validarCapacidadesCronos,
  validarDatosCronos,
} from "./contrato.js";
import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js";

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function instanteVisible(instante, locale, zonaHoraria) {
  const fecha = new Date(instante);
  if (!Number.isFinite(fecha.getTime())) throw new Error("instante de Cronos no válido");
  return {
    fecha: new Intl.DateTimeFormat(locale, {
      timeZone: zonaHoraria, day: "2-digit", month: "2-digit", year: "numeric",
    }).format(fecha),
    hora: new Intl.DateTimeFormat(locale, {
      timeZone: zonaHoraria, hour: "2-digit", minute: "2-digit", timeZoneName: "short",
    }).format(fecha),
    completo: new Intl.DateTimeFormat(locale, {
      timeZone: zonaHoraria, day: "2-digit", month: "2-digit", year: "numeric",
      hour: "2-digit", minute: "2-digit", timeZoneName: "short",
    }).format(fecha),
  };
}

function fechaCalendarioVisible(fechaISO, locale, zonaHoraria) {
  const [anio, mes, dia] = String(fechaISO).split("-").map(Number);
  const instante = new Date(Date.UTC(anio, mes - 1, dia, 12));
  return new Intl.DateTimeFormat(locale, {
    timeZone: zonaHoraria, day: "2-digit", month: "2-digit", year: "numeric",
  }).format(instante);
}

function cantidadVisible(valor, unidad, t) {
  if (unidad === "dia") return t(Number(valor) === 1 ? "cantidad_dia" : "cantidad_dias", { valor });
  const horas = String(Math.floor(Number(valor) / 60)).padStart(2, "0");
  const minutos = String(Number(valor) % 60).padStart(2, "0");
  return t("cantidad_horas", { horas, minutos });
}

function claseEstado(valor) {
  const estado = String(valor || "");
  if (new Set(["aprobado", "registrado", "revisado", "disfrutado", "calculado"]).has(estado)) return "exito";
  if (new Set(["pendiente_responsable", "preparado_no_registrado", "sin_registrar"]).has(estado)) return "aviso";
  return "neutro";
}

function chip(estado, t) {
  return `<span class="cronos-estado cronos-estado-${claseEstado(estado)}">${escaparHTML(t(`estado_${estado}`))}</span>`;
}

function tabla({ id, titulo, cabeceras, filas, vacio }) {
  return `
    <div class="cronos-tabla-contenedor">
      <table class="cronos-tabla" id="${escaparHTML(id)}">
        <caption>${escaparHTML(titulo)}</caption>
        <thead><tr>${cabeceras.map((cabecera) => `<th scope="col">${escaparHTML(cabecera)}</th>`).join("")}</tr></thead>
        <tbody>${filas.length
    ? filas.map((fila) => `<tr>${fila.map((celda, indice) => `<${indice === 0 ? "th scope=\"row\"" : "td"}>${celda}</${indice === 0 ? "th" : "td"}>`).join("")}</tr>`).join("")
    : `<tr><td colspan="${cabeceras.length}" class="cronos-vacio">${escaparHTML(vacio)}</td></tr>`}</tbody>
      </table>
    </div>`;
}

function indicador(etiqueta, valor, nota, tono = "informacion") {
  return `<article class="cronos-indicador cronos-indicador-${tono}">
    <span>${escaparHTML(etiqueta)}</span>
    <strong>${escaparHTML(valor)}</strong>
    <small>${escaparHTML(nota)}</small>
  </article>`;
}

function accionFichaje(tipo, permitida, demostracion, t) {
  const claves = {
    entrada: "accion_entrada",
    salida: "accion_salida",
    inicio_pausa: "accion_inicio_pausa",
    fin_pausa: "accion_fin_pausa",
  };
  const texto = `${t(claves[tipo])}${demostracion ? " DEMO" : ""}`;
  return `<button type="button" class="${tipo === "entrada" ? "boton-primario" : "boton-secundario"}"
    data-cronos-accion="registrar-fichaje" data-cronos-tipo="${tipo}"
    ${permitida ? "" : `disabled aria-disabled="true" title="${escaparHTML(t("accion_sin_capacidad"))}"`}>${escaparHTML(texto)}</button>`;
}

function botonDescargaRecibo(referencia, disponible, t) {
  return `<button type="button" class="cronos-boton-tabla" data-cronos-accion="descargar-recibo"
    data-cronos-recibo-ref="${escaparHTML(referencia)}"
    ${disponible ? "" : `disabled aria-disabled="true" title="${escaparHTML(t("descarga_sin_puerto"))}"`}>${escaparHTML(t("descargar_recibo"))}</button>`;
}

export function renderizarAreaCronos({
  contextoActor, capacidades = [], datos, recibos = [], mensaje = "",
  mensajes = MENSAJES_CRONOS_ES, descargaRecibosDisponible = false,
  locale = "es-ES", zonaHoraria = "Europe/Madrid",
}) {
  const t = crearTraductorCronos(mensajes);
  const contexto = exigirContextoActorCronos(contextoActor);
  const capacidadesValidadas = validarCapacidadesCronos(capacidades);
  const vista = validarDatosCronos(datos, contexto);
  const puedeConsultarFichajes = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_FICHAJES);
  const puedeConsultarHorario = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_HORARIO);
  const puedeConsultarPermisos = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_CONSULTAR_PERMISOS);
  const puedeFichar = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_REGISTRAR_FICHAJE);
  const puedeSolicitar = tieneCapacidadCronos(capacidadesValidadas, CAPACIDAD_SOLICITAR_PERMISO);
  const puedeConsultarHistorial = puedeConsultarFichajes || puedeConsultarPermisos;
  const perfil = vista.perfil_jornada;
  const resumen = vista.resumen;

  const filasFichajes = puedeConsultarFichajes ? vista.fichajes.slice(0, 12).map((item) => {
    const visible = instanteVisible(item.instante, locale, zonaHoraria);
    return [
      escaparHTML(visible.fecha), escaparHTML(visible.hora), escaparHTML(t(`movimiento_${item.tipo_clave}`)),
      escaparHTML(item.modalidad), escaparHTML(item.canal), chip(item.estado_clave, t),
      `<code>${escaparHTML(item.recibo_ref)}</code>`, botonDescargaRecibo(item.recibo_ref, descargaRecibosDisponible, t),
    ];
  }) : [];
  const filasSaldos = puedeConsultarPermisos ? vista.saldos.map((item) => [
    escaparHTML(item.nombre), escaparHTML(cantidadVisible(item.concedido, item.unidad_clave, t)),
    escaparHTML(cantidadVisible(item.solicitado, item.unidad_clave, t)),
    escaparHTML(cantidadVisible(item.aprobado, item.unidad_clave, t)),
    escaparHTML(cantidadVisible(item.disfrutado, item.unidad_clave, t)),
    `<strong>${escaparHTML(cantidadVisible(item.restante, item.unidad_clave, t))}</strong>`, chip(item.estado_clave, t),
  ]) : [];
  const filasSolicitudes = puedeConsultarPermisos ? vista.solicitudes.map((item) => [
    `<code>${escaparHTML(item.id)}</code>`, escaparHTML(item.tipo),
    `${escaparHTML(fechaCalendarioVisible(item.desde, locale, zonaHoraria))}–${escaparHTML(fechaCalendarioVisible(item.hasta, locale, zonaHoraria))}`,
    escaparHTML(cantidadVisible(item.cantidad_valor, item.unidad_clave, t)),
    chip(item.estado_clave, t), `<code>${escaparHTML(item.recibo_ref)}</code>`, botonDescargaRecibo(item.recibo_ref, descargaRecibosDisponible, t),
  ]) : [];
  const filasHistorial = puedeConsultarHistorial ? vista.historial.map((item) => [
    escaparHTML(instanteVisible(item.instante, locale, zonaHoraria).completo), escaparHTML(item.evento), escaparHTML(item.detalle),
    chip(item.estado_clave, t), `<code>${escaparHTML(item.recibo_ref)}</code>`,
  ]) : [];
  const filasRecibos = recibos.map((item) => [
    `<code>${escaparHTML(item.referencia)}</code>`, escaparHTML(instanteVisible(item.instante, locale, zonaHoraria).completo),
    escaparHTML(item.operacion), chip(item.estado_clave, t), botonDescargaRecibo(item.referencia, descargaRecibosDisponible, t),
  ]);

  return `<section class="cronos-area" aria-labelledby="cronos-titulo">
    <header class="cronos-encabezado" id="cronos-resumen">
      <div>
        <p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p>
        <h2 id="cronos-titulo">${escaparHTML(t("titulo"))}</h2>
        <p>${escaparHTML(t("descripcion"))}</p>
      </div>
      <div class="cronos-encabezado-estado">
        ${vista.demostracion ? `<span class="cronos-distintivo-demo">${escaparHTML(t("entorno_demo"))}</span>` : `<span class="cronos-estado cronos-estado-exito">${escaparHTML(t("servicio_interno"))}</span>`}
        <span>${escaparHTML(t("actualizado", { fecha: instanteVisible(vista.actualizado_en, locale, zonaHoraria).completo }))}</span>
      </div>
    </header>

    <section class="cronos-frontera" aria-label="${escaparHTML(t("ambito_etiqueta"))}">
      <div><strong>${escaparHTML(contexto.actor.nombre_visible)}</strong><span>${escaparHTML(contexto.rol.etiqueta)}</span></div>
      <p><strong>${escaparHTML(t("ambito_texto"))}</strong> ${escaparHTML(t("ambito_descripcion"))}</p>
      <p><strong>${escaparHTML(t("privacidad_texto"))}</strong> ${escaparHTML(t("privacidad_descripcion"))}</p>
    </section>

    <nav class="cronos-navegacion" aria-label="${escaparHTML(t("navegacion_etiqueta"))}">
      <button type="button" data-cronos-destino="cronos-resumen">${escaparHTML(t("navegacion_resumen"))}</button><button type="button" data-cronos-destino="cronos-fichajes">${escaparHTML(t("navegacion_fichajes"))}</button>
      <button type="button" data-cronos-destino="cronos-permisos">${escaparHTML(t("navegacion_permisos"))}</button><button type="button" data-cronos-destino="cronos-historial">${escaparHTML(t("navegacion_historial"))}</button>
    </nav>

    <div class="cronos-indicadores" aria-label="${escaparHTML(t("resumen_etiqueta"))}">
      ${indicador(t("indicador_jornada"), puedeConsultarHorario ? resumen.teoricas_hoy : "—", puedeConsultarHorario ? vista.periodo : t("sin_permiso"))}
      ${indicador(t("indicador_trabajado"), puedeConsultarFichajes ? resumen.trabajadas_hoy : "—", puedeConsultarFichajes ? t("nota_movimientos") : t("sin_permiso"), puedeConsultarFichajes ? "exito" : "informacion")}
      ${indicador(t("indicador_saldo_dia"), puedeConsultarFichajes ? resumen.saldo_hoy : "—", puedeConsultarFichajes ? t("nota_calculo") : t("sin_permiso"), puedeConsultarFichajes ? "exito" : "informacion")}
      ${indicador(t("indicador_saldo_periodo"), puedeConsultarFichajes ? resumen.saldo_periodo : "—", puedeConsultarFichajes ? t("nota_acumulado") : t("sin_permiso"), "informacion")}
      ${indicador(t("indicador_incidencias"), puedeConsultarFichajes ? resumen.incidencias_abiertas : "—", puedeConsultarFichajes ? t("nota_revision") : t("sin_permiso"), puedeConsultarFichajes && resumen.incidencias_abiertas ? "aviso" : "informacion")}
      ${indicador(t("indicador_solicitudes"), puedeConsultarPermisos ? resumen.solicitudes_pendientes : "—", puedeConsultarPermisos ? t("nota_circuito") : t("sin_permiso"), puedeConsultarPermisos && resumen.solicitudes_pendientes ? "aviso" : "informacion")}
    </div>

    <div class="cronos-rejilla-principal">
      <section class="panel cronos-panel" aria-labelledby="cronos-fichajes">
        <div class="cabecera-panel cronos-cabecera-panel"><div><h3 id="cronos-fichajes">${escaparHTML(t("fichajes_titulo"))}</h3><p>${escaparHTML(t("fichajes_descripcion"))}</p></div>
          <div class="cronos-acciones">${accionFichaje("entrada", puedeFichar, vista.demostracion, t)}${accionFichaje("inicio_pausa", puedeFichar, vista.demostracion, t)}${accionFichaje("fin_pausa", puedeFichar, vista.demostracion, t)}${accionFichaje("salida", puedeFichar, vista.demostracion, t)}</div>
        </div>
        ${puedeConsultarFichajes
    ? tabla({ id: "tabla-cronos-fichajes", titulo: t("fichajes_caption"), cabeceras: [t("cab_fecha"), t("cab_hora"), t("cab_movimiento"), t("cab_modalidad"), t("cab_canal"), t("cab_estado"), t("cab_recibo"), t("cab_accion")], filas: filasFichajes, vacio: t("fichajes_vacio") })
    : `<p class="cronos-acceso-denegado" role="status">${escaparHTML(t("fichajes_denegado"))}</p>`}
      </section>

      <aside class="panel cronos-panel" aria-labelledby="cronos-horario-titulo">
        <div class="cabecera-panel"><div><h3 id="cronos-horario-titulo">${escaparHTML(t("horario_titulo"))}</h3><p>${escaparHTML(t("horario_descripcion"))}</p></div></div>
        ${puedeConsultarHorario ? `<dl class="cronos-resumen-datos">
          <div><dt>${escaparHTML(t("perfil"))}</dt><dd>${escaparHTML(perfil.nombre)}</dd></div>
          <div><dt>${escaparHTML(t("jornada_diaria"))}</dt><dd>${escaparHTML(perfil.jornada_diaria)}</dd></div>
          <div><dt>${escaparHTML(t("jornada_semanal"))}</dt><dd>${escaparHTML(perfil.jornada_semanal)}</dd></div>
          <div><dt>${escaparHTML(t("ventana_entrada"))}</dt><dd>${escaparHTML(perfil.ventana_entrada)}</dd></div>
          <div><dt>${escaparHTML(t("tramo_obligatorio"))}</dt><dd>${escaparHTML(perfil.tramo_obligatorio)}</dd></div>
          <div><dt>${escaparHTML(t("teletrabajo"))}</dt><dd>${escaparHTML(perfil.teletrabajo)}</dd></div>
        </dl>` : `<p class="cronos-acceso-denegado" role="status">${escaparHTML(t("horario_denegado"))}</p>`}
      </aside>
    </div>

    <section class="panel cronos-panel" id="cronos-permisos" aria-labelledby="cronos-saldos-titulo">
      <div class="cabecera-panel"><div><h3 id="cronos-saldos-titulo">${escaparHTML(t("permisos_titulo"))}</h3><p>${escaparHTML(t("permisos_descripcion"))}</p></div></div>
      ${puedeConsultarPermisos
    ? tabla({ id: "tabla-cronos-saldos", titulo: t("saldos_caption"), cabeceras: [t("cab_concepto"), t("cab_concedido"), t("cab_solicitado"), t("cab_aprobado"), t("cab_disfrutado"), t("cab_restante"), t("cab_estado")], filas: filasSaldos, vacio: t("saldos_vacio") })
    : `<p class="cronos-acceso-denegado" role="status">${escaparHTML(t("permisos_denegado"))}</p>`}
    </section>

    <div class="cronos-rejilla-secundaria">
      <section class="panel cronos-panel" aria-labelledby="cronos-solicitud-titulo">
        <div class="cabecera-panel"><div><h3 id="cronos-solicitud-titulo">${escaparHTML(t("solicitud_titulo"))}</h3><p>${escaparHTML(t(vista.demostracion ? "solicitud_demo" : "solicitud_real"))}</p></div></div>
        <form class="cronos-formulario" data-cronos-formulario="solicitud-permiso">
          <label><span>${escaparHTML(t("tipo_permiso"))}</span><select name="tipo" required ${puedeSolicitar ? "" : "disabled"}>
            ${puedeSolicitar ? vista.saldos.map((item) => `<option value="${escaparHTML(item.id)}" data-unidad-clave="${escaparHTML(item.unidad_clave)}">${escaparHTML(t("saldo_opcion", { nombre: item.nombre, restante: cantidadVisible(item.restante, item.unidad_clave, t) }))}</option>`).join("") : `<option value="">${escaparHTML(t("no_disponible"))}</option>`}
          </select></label>
          <label><span>${escaparHTML(t("desde"))}</span><input name="desde" type="date" required ${puedeSolicitar ? "" : "disabled"}></label>
          <label><span>${escaparHTML(t("hasta"))}</span><input name="hasta" type="date" required ${puedeSolicitar ? "" : "disabled"}></label>
          <label><span>${escaparHTML(t("cantidad"))}</span><input name="cantidad" type="number" min="1" max="100000" step="1" value="1" required ${puedeSolicitar ? "" : "disabled"}><small>${escaparHTML(t("cantidad_ayuda"))}</small></label>
          <label class="cronos-campo-ancho"><span>${escaparHTML(t("motivo"))}</span><textarea name="motivo" maxlength="500" rows="2" ${puedeSolicitar ? "" : "disabled"}></textarea><small>${escaparHTML(t("motivo_ayuda"))}</small></label>
          <label class="cronos-campo-ancho"><span>${escaparHTML(t("justificante"))}</span><input name="documento_ref" maxlength="120" placeholder="${escaparHTML(t("justificante_placeholder"))}" ${puedeSolicitar ? "" : "disabled"}></label>
          <div class="cronos-formulario-acciones"><button type="submit" class="boton-primario" ${puedeSolicitar ? "" : 'disabled aria-disabled="true"'}>${escaparHTML(t("preparar_solicitud"))}${vista.demostracion ? " DEMO" : ""}</button></div>
        </form>
      </section>

      <section class="panel cronos-panel" aria-labelledby="cronos-solicitudes-titulo">
        <div class="cabecera-panel"><div><h3 id="cronos-solicitudes-titulo">${escaparHTML(t("solicitudes_titulo"))}</h3><p>${escaparHTML(t("solicitudes_descripcion"))}</p></div></div>
        ${tabla({ id: "tabla-cronos-solicitudes", titulo: t("solicitudes_caption"), cabeceras: [t("cab_referencia"), t("cab_tipo"), t("cab_periodo"), t("cab_cantidad"), t("cab_estado"), t("cab_recibo"), t("cab_accion")], filas: filasSolicitudes, vacio: t("solicitudes_vacio") })}
      </section>
    </div>

    <section class="panel cronos-panel" id="cronos-historial" aria-labelledby="cronos-historial-titulo">
      <div class="cabecera-panel"><div><h3 id="cronos-historial-titulo">${escaparHTML(t("historial_titulo"))}</h3><p>${escaparHTML(t("historial_descripcion"))}</p></div></div>
      ${puedeConsultarHistorial ? tabla({ id: "tabla-cronos-historial", titulo: t("historial_caption"), cabeceras: [t("cab_fecha"), t("cab_evento"), t("cab_detalle"), t("cab_estado"), t("cab_recibo")], filas: filasHistorial, vacio: t("historial_vacio") }) : `<p class="cronos-acceso-denegado" role="status">${escaparHTML(t("historial_denegado"))}</p>`}
      ${filasRecibos.length ? `<div class="cronos-recibos"><h4>${escaparHTML(t("recibos_titulo"))}</h4>${tabla({ id: "tabla-cronos-recibos", titulo: t(vista.demostracion ? "recibos_demo_caption" : "recibos_caption"), cabeceras: [t("cab_referencia"), t("cab_fecha"), t("cab_operacion"), t("cab_estado"), t("cab_accion")], filas: filasRecibos, vacio: t("recibos_vacio") })}</div>` : ""}
      <p class="cronos-mensaje" role="status" aria-live="polite">${escaparHTML(mensaje)}</p>
    </section>
  </section>`;
}
