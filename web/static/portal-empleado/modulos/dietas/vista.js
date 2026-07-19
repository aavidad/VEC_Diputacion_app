import { crearPresentadorDietas } from "./presentador.js";
import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { CODIGO_ERROR_SERVICIO_RUTAS_DIETAS } from "./contrato.js";

const ESTADOS_FILTRO = Object.freeze([
  ["todos", "estado_todos"],
  ["borrador", "estado_borrador"],
  ["pendiente_jefatura", "estado_pendiente_jefatura"],
  ["aprobada", "estado_aprobada"],
  ["enviada_rrhh", "estado_enviada_rrhh"],
  ["enviada_nomina", "estado_enviada_nomina"],
  ["pagada", "estado_pagada"],
]);

const CLAVES_ESTADO = Object.freeze(Object.fromEntries(ESTADOS_FILTRO.slice(1)));
const CLAVES_ETAPA = Object.freeze({
  borrador: "etapa_borrador", jefatura: "etapa_jefatura", aprobada: "etapa_aprobada",
  rrhh: "etapa_rrhh", nomina: "etapa_nomina", pagada: "etapa_pagada",
});
const CLAVES_SIGUIENTE = Object.freeze({
  remision_rrhh: "siguiente_remision_rrhh",
  completar_enviar_validacion: "siguiente_completar_enviar_validacion",
  expediente_finalizado: "siguiente_expediente_finalizado",
  inclusion_nomina: "siguiente_inclusion_nomina",
  revision_jefatura: "siguiente_revision_jefatura",
});
const CLAVES_RESULTADO = Object.freeze({
  borrador_creado_demo: "resultado_borrador_creado_demo",
  envio_jefatura_demo: "resultado_envio_jefatura_demo",
});
const CLAVES_ETIQUETA_ALTERNATIVA = Object.freeze({
  ruta_alternativa_osrm_1: "ruta_etiqueta_osrm_1",
  ruta_alternativa_osrm_2: "ruta_etiqueta_osrm_2",
  ruta_alternativa_osrm_3: "ruta_etiqueta_osrm_3",
});

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function euros(valor) {
  return new Intl.NumberFormat("es-ES", { style: "currency", currency: "EUR" }).format(Number(valor || 0));
}

function numero(valor, decimales = 0) {
  return new Intl.NumberFormat("es-ES", { minimumFractionDigits: decimales, maximumFractionDigits: decimales }).format(Number(valor || 0));
}

function fechaCorta(valor) {
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.getTime()) ? new Intl.DateTimeFormat("es-ES", { timeZone: "UTC" }).format(fecha) : "-";
}

function fechaHora(valor) {
  const fecha = new Date(valor);
  return Number.isFinite(fecha.getTime())
    ? new Intl.DateTimeFormat("es-ES", { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }).format(fecha)
    : "-";
}

function claseEstado(estado) {
  if (estado === "pagada" || estado === "aprobada") return "exito";
  if (estado === "borrador") return "neutro";
  if (estado === "pendiente_jefatura") return "aviso";
  return "info";
}

function traducirCodigo(t, mapa, codigo, respaldo = "sin_capacidad") {
  return t(mapa[codigo] || respaldo);
}

function opcionesEstado(seleccionado, t) {
  return ESTADOS_FILTRO
    .map(([estado, clave]) => `<option value="${escaparHTML(estado)}" ${estado === seleccionado ? "selected" : ""}>${escaparHTML(t(clave))}</option>`)
    .join("");
}

function renderizarEtapas(modelo, comision, t) {
  return `<ol class="dietas-etapas" aria-label="${escaparHTML(t("circuito_aprobacion"))}">
    ${modelo.etapas.map((etapa, indice) => {
      const clase = indice < comision.etapa_actual ? "completada" : (indice === comision.etapa_actual ? "actual" : "pendiente");
      const claveEstado = indice < comision.etapa_actual ? "etapa_completada" : (indice === comision.etapa_actual ? "etapa_actual" : "etapa_pendiente");
      return `<li class="${clase}"><span aria-hidden="true">${indice + 1}</span><strong>${escaparHTML(traducirCodigo(t, CLAVES_ETAPA, etapa))}</strong><small>${escaparHTML(t(claveEstado))}</small></li>`;
    }).join("")}
  </ol>`;
}

function identificadorHTML(valor) {
  return String(valor ?? "mapa").replace(/[^A-Za-z0-9_-]/g, "-").slice(0, 90) || "mapa";
}

function renderizarMapaRuta(item, t) {
  if (!item.mapa_ruta) return "";
  const referenciaMapa = item.mapa_ruta.vista_ref || "mapa-dietas";
  const sufijo = identificadorHTML(referenciaMapa);
  const { geometria } = item.mapa_ruta;
  const geometriaOSRM = geometria.origen === "osrm_interno";
  return `<figure class="dietas-mapa" data-dietas-mapa-ref="${escaparHTML(referenciaMapa)}" aria-labelledby="dietas-mapa-titulo-${sufijo}" aria-describedby="dietas-mapa-nota-${sufijo}">
    <figcaption><div><p class="sobrelinea">${escaparHTML(t("mapa_proveedor"))}</p><h4 id="dietas-mapa-titulo-${sufijo}">${escaparHTML(t("mapa_titulo"))}</h4></div><span class="estado-chip ${geometriaOSRM ? "info" : "aviso"}">${escaparHTML(t(geometriaOSRM ? "mapa_marca_osrm" : "mapa_marca_demo"))}</span></figcaption>
    <div class="dietas-mapa-canvas" data-dietas-mapa-canvas role="region" aria-label="${escaparHTML(t("mapa_region_accesible"))}"><p class="dietas-mapa-espera" role="status">${escaparHTML(t("mapa_cargando_interno"))}</p></div>
    <p id="dietas-mapa-nota-${sufijo}" class="dietas-mapa-nota" data-dietas-mapa-estado role="status" aria-live="polite" aria-atomic="true">${escaparHTML(t(geometriaOSRM ? "mapa_nota_osrm_local" : "mapa_nota_fallback"))}</p>
    <small data-dietas-mapa-atribucion hidden>${escaparHTML(item.mapa_ruta.atribucion)}</small>
  </figure>`;
}

function renderizarDetalle(modelo, descargaDisponible, confirmacionDisponible, t) {
  const item = modelo.seleccionada;
  if (!item) return `<section class="panel dietas-detalle"><div class="cuerpo-panel vacio-controlado"><p><strong>${escaparHTML(t("sin_expediente"))}</strong></p><p>${escaparHTML(t("sin_expediente_ayuda"))}</p></div></section>`;
  const historial = item.historial.map((evento) => `<li>
    <span class="dietas-marca-historial" aria-hidden="true"></span>
    <div><strong>${escaparHTML(traducirCodigo(t, CLAVES_ESTADO, evento.estado))}</strong><span>${escaparHTML(fechaHora(evento.instante))} · ${escaparHTML(evento.actor_ref)}</span><small>${escaparHTML(t("recibo_referencia", { referencia: evento.recibo }))}</small></div>
  </li>`).join("");
  return `<section class="panel dietas-detalle" aria-labelledby="dietas-titulo-detalle">
    <div class="cabecera-panel"><div><p class="sobrelinea">${escaparHTML(t("expediente_seleccionado"))}</p><h3 id="dietas-titulo-detalle" tabindex="-1">${escaparHTML(item.referencia)}</h3></div><span class="estado-chip ${claseEstado(item.estado)}">${escaparHTML(traducirCodigo(t, CLAVES_ESTADO, item.estado))}</span></div>
    <div class="cuerpo-panel dietas-detalle-cuerpo">
      <dl class="dietas-datos-clave">
        <div><dt>${escaparHTML(t("fecha"))}</dt><dd>${escaparHTML(fechaCorta(item.fecha))}</dd></div>
        <div><dt>${escaparHTML(t("motivo"))}</dt><dd>${escaparHTML(item.motivo)}</dd></div>
        <div><dt>${escaparHTML(t("ruta"))}</dt><dd>${modelo.capacidades.consultarRutas ? escaparHTML(item.ruta.join(" → ")) : escaparHTML(t("sin_capacidad"))}</dd></div>
        <div><dt>${escaparHTML(t("distancia"))}</dt><dd>${modelo.capacidades.consultarRutas ? `${numero(item.kilometros, 1)} ${escaparHTML(t("unidad_km"))}` : escaparHTML(t("sin_capacidad"))}</dd></div>
        <div><dt>${escaparHTML(t("justificantes"))}</dt><dd>${numero(item.justificantes)}</dd></div>
        <div><dt>${escaparHTML(t("siguiente_actuacion"))}</dt><dd>${escaparHTML(traducirCodigo(t, CLAVES_SIGUIENTE, item.siguiente_actuacion))}</dd></div>
      </dl>
      ${renderizarEtapas(modelo, item, t)}
      <section aria-labelledby="dietas-titulo-desglose"><h4 id="dietas-titulo-desglose">${escaparHTML(t("desglose_gastos"))}</h4>
        <dl class="dietas-desglose">
          <div><dt>${escaparHTML(t("kilometraje"))}</dt><dd>${modelo.capacidades.consultarRutas ? euros(item.kilometraje_euros) : escaparHTML(t("sin_capacidad"))}</dd></div>
          <div><dt>${escaparHTML(t("manutencion"))}</dt><dd>${euros(item.manutencion_euros)}</dd></div>
          <div><dt>${escaparHTML(t("alojamiento"))}</dt><dd>${euros(item.alojamiento_euros)}</dd></div>
          <div><dt>${escaparHTML(t("otros_gastos"))}</dt><dd>${euros(item.otros_gastos_euros)}</dd></div>
          <div class="total"><dt>${escaparHTML(t("total"))}</dt><dd>${euros(item.total_euros)}</dd></div>
        </dl>
      </section>
      ${item.nomina ? `<p class="dietas-pago"><strong>${escaparHTML(t("pago_asociado", { demo: modelo.demostracion ? " DEMO" : "" }))}</strong> ${escaparHTML(item.nomina)} · ${escaparHTML(item.referencia_pago)}</p>` : ""}
      ${modelo.capacidades.consultarHistorialPropio
    ? `<section aria-labelledby="dietas-titulo-historial"><h4 id="dietas-titulo-historial">${escaparHTML(t("historial_trazabilidad"))}</h4><ol class="dietas-historial">${historial}</ol></section>`
    : `<p role="status">${escaparHTML(t("auditoria_denegada"))}</p>`}
      <div class="acciones-vista">
        ${modelo.capacidades.consultarHistorialPropio ? `<button type="button" class="boton-secundario" data-dietas-descargar-recibo="${escaparHTML(item.historial.at(-1)?.recibo || "")}" ${descargaDisponible ? "" : "disabled"}>${escaparHTML(t(descargaDisponible ? "descargar_recibo" : "descarga_no_conectada"))}</button>` : ""}
        ${item.estado === "borrador" && modelo.capacidades.gestionarGastos ? `<button type="button" class="boton-primario" data-dietas-enviar="${escaparHTML(item.referencia)}" ${confirmacionDisponible ? "" : "disabled"}>${escaparHTML(confirmacionDisponible ? t("enviar_validacion", { demo: modelo.demostracion ? " DEMO" : "" }) : t("confirmacion_no_conectada"))}</button>` : ""}
      </div>
    </div>
  </section>`;
}

function renderizarTabla(modelo, t) {
  const filas = modelo.comisiones.map((item) => `<tr>
    <th scope="row"><button type="button" class="enlace-tabla" data-dietas-seleccionar="${escaparHTML(item.referencia)}" ${item.referencia === modelo.seleccionada?.referencia ? 'aria-current="true"' : ""}>${escaparHTML(item.referencia)}</button></th>
    <td>${escaparHTML(fechaCorta(item.fecha))}</td>
    <td>${modelo.capacidades.consultarRutas ? escaparHTML(item.ruta.join(" → ")) : escaparHTML(t("sin_capacidad"))}</td>
    <td class="numero">${modelo.capacidades.consultarRutas ? `${numero(item.kilometros, 1)} ${escaparHTML(t("unidad_km"))}` : escaparHTML(t("sin_capacidad"))}</td>
    <td class="numero">${euros(item.total_euros)}</td>
    <td><span class="estado-chip ${claseEstado(item.estado)}">${escaparHTML(traducirCodigo(t, CLAVES_ESTADO, item.estado))}</span></td>
  </tr>`).join("");
  return `<section class="panel dietas-listado" aria-labelledby="dietas-titulo-listado">
    <div class="cabecera-panel"><h3 id="dietas-titulo-listado">${escaparHTML(t("mis_comisiones"))}</h3><span class="estado-chip info">${escaparHTML(t("contador_resultados", { visibles: numero(modelo.comisiones.length), total: numero(modelo.totalSinFiltrar) }))}</span></div>
    <div class="tabla-contenedor"><table class="tabla-datos"><caption>${escaparHTML(t("caption_comisiones"))}</caption>
      <thead><tr><th scope="col">${escaparHTML(t("cab_expediente"))}</th><th scope="col">${escaparHTML(t("cab_fecha"))}</th><th scope="col">${escaparHTML(t("cab_ruta"))}</th><th scope="col">${escaparHTML(t("cab_kilometros"))}</th><th scope="col">${escaparHTML(t("cab_total"))}</th><th scope="col">${escaparHTML(t("cab_estado"))}</th></tr></thead>
      <tbody>${filas || `<tr><td colspan="6" class="vacio-controlado">${escaparHTML(t("sin_resultados"))}</td></tr>`}</tbody>
    </table></div>
  </section>`;
}

function renderizarHistorialMensual(modelo, descargaDisponible, t) {
  const anual = modelo.resumenAnual;
  return `<section class="panel dietas-resumen-anual" aria-labelledby="dietas-titulo-resumen-anual">
    <div class="cabecera-panel"><div><h3 id="dietas-titulo-resumen-anual">${escaparHTML(t("resumen_mes"))}</h3><span>${escaparHTML(t(modelo.demostracion ? "escenario_demo" : "datos_sesion"))}</span></div>
      ${anual ? `<button type="button" class="boton-secundario dietas-exportar-anual" data-dietas-descargar-anual="${anual.anio}" ${descargaDisponible && modelo.demostracion ? "" : "disabled"}>${escaparHTML(t(descargaDisponible && modelo.demostracion ? "descargar_resumen_anual" : "resumen_anual_no_conectado", { anio: anual.anio }))}</button>` : ""}
    </div>
    <div class="tabla-contenedor"><table class="tabla-datos"><caption>${escaparHTML(t("caption_historico"))}</caption>
      <thead><tr><th scope="col">${escaparHTML(t("cab_mes"))}</th><th scope="col">${escaparHTML(t("cab_expedientes"))}</th><th scope="col">${escaparHTML(t("cab_kilometros"))}</th><th scope="col">${escaparHTML(t("cab_devengado"))}</th><th scope="col">${escaparHTML(t("cab_pagado"))}</th></tr></thead>
      <tbody>${modelo.historialMensual.map((item) => `<tr><th scope="row">${escaparHTML(item.mes)}</th><td class="numero">${numero(item.expedientes)}</td><td class="numero">${modelo.capacidades.consultarRutas ? `${numero(item.kilometros, 1)} ${escaparHTML(t("unidad_km"))}` : escaparHTML(t("sin_capacidad"))}</td><td class="numero">${euros(item.total_euros)}</td><td class="numero">${euros(item.pagado_euros)}</td></tr>`).join("")}</tbody>
    </table></div>
  </section>`;
}

function opcionesCatalogoRuta(puntos, seleccionado, t) {
  return `<option value="">${escaparHTML(t("ruta_seleccionar_localidad"))}</option>${puntos.map((punto) => {
    const etiqueta = punto.tipo === "nucleo" && punto.municipio_nombre !== punto.nombre
      ? `${punto.nombre} (${punto.municipio_nombre})` : punto.nombre;
    return `<option value="${escaparHTML(punto.codigo)}" ${punto.codigo === seleccionado ? "selected" : ""}>${escaparHTML(etiqueta)}</option>`;
  }).join("")}`;
}

function renderizarHerramientaRutas(rutas, t, errorRuta = "") {
  if (!rutas) return `<section class="dietas-ruta-no-disponible" role="status"><strong>${escaparHTML(t("ruta_puerto_no_disponible"))}</strong></section>`;
  const puntos = rutas.catalogo.puntos;
  const puntosPorCodigo = new Map(puntos.map((punto) => [punto.codigo, punto]));
  const paradas = rutas.paradas.map((codigo, indice) => {
    const clave = indice === 0 ? "ruta_salida" : (indice === rutas.paradas.length - 1 ? "ruta_destino_final" : "ruta_parada_intermedia");
    const etiqueta = t(clave, { numero: indice });
    const nombreParada = puntosPorCodigo.get(codigo)?.nombre || etiqueta;
    return `<div class="dietas-ruta-parada">
      <label><span>${escaparHTML(etiqueta)}</span><select data-dietas-ruta-parada="${indice}" required>${opcionesCatalogoRuta(puntos, codigo, t)}</select></label>
      <button type="button" class="boton-terciario" data-dietas-ruta-quitar="${indice}" aria-label="${escaparHTML(t("ruta_quitar_parada_accesible", { parada: nombreParada, numero: indice + 1 }))}" ${rutas.paradas.length <= 2 ? "disabled" : ""}>${escaparHTML(t("ruta_quitar_parada"))}</button>
    </div>`;
  }).join("");
  const alternativas = rutas.alternativas.map((alternativa) => `<article class="dietas-ruta-alternativa ${alternativa.seleccionada ? "seleccionada" : ""}">
    <div><strong>${escaparHTML(CLAVES_ETIQUETA_ALTERNATIVA[alternativa.etiqueta]
    ? t(CLAVES_ETIQUETA_ALTERNATIVA[alternativa.etiqueta]) : alternativa.etiqueta)}</strong><span>${numero(alternativa.kilometros, 1)} km · ${numero(alternativa.duracion_minutos)} min</span></div>
    ${alternativa.recomendada
    ? `<button type="button" class="boton-secundario" data-dietas-ruta-alternativa="${escaparHTML(alternativa.referencia)}" aria-pressed="${alternativa.seleccionada}">${escaparHTML(t("ruta_usar_recomendada"))}</button>`
    : `<label>${escaparHTML(t("ruta_motivo_alternativa"))}<input data-dietas-ruta-motivo-alternativa="${escaparHTML(alternativa.referencia)}" maxlength="500" placeholder="${escaparHTML(t("ruta_motivo_alternativa_ejemplo"))}" value="${alternativa.seleccionada ? escaparHTML(rutas.motivo_alternativa) : ""}"></label><button type="button" class="boton-secundario" data-dietas-ruta-alternativa="${escaparHTML(alternativa.referencia)}" aria-pressed="${alternativa.seleccionada}">${escaparHTML(t("ruta_usar_alternativa"))}</button>`}
  </article>`).join("");
  const tramos = rutas.tramos.map((tramo) => `<tr data-dietas-tramo="${tramo.indice}">
    <th scope="row"><span class="dietas-tramo-color" aria-hidden="true"></span>${escaparHTML(tramo.origen_nombre)} → ${escaparHTML(tramo.destino_nombre)}</th>
    <td class="numero">${numero(tramo.kilometros, 1)} km</td><td class="numero">${numero(tramo.duracion_minutos)} min</td>
    <td><label class="solo-etiqueta">${escaparHTML(t("ruta_ajuste_km"))}<input data-dietas-ruta-ajuste-km="${tramo.indice}" type="number" min="0" max="1000" step="0.1" value="${escaparHTML(tramo.ajuste_kilometros)}"></label></td>
    <td><label class="solo-etiqueta">${escaparHTML(t("ruta_motivo_ajuste"))}<input data-dietas-ruta-ajuste-motivo="${tramo.indice}" maxlength="500" value="${escaparHTML(tramo.motivo_ajuste)}" placeholder="${escaparHTML(t("ruta_motivo_ajuste_ejemplo"))}"></label></td>
    <td><button type="button" class="boton-terciario" data-dietas-ruta-aplicar-ajuste="${tramo.indice}">${escaparHTML(t("ruta_aplicar_ajuste"))}</button></td>
  </tr>`).join("");
  const resultado = rutas.calculado ? `<section class="dietas-ruta-resultado" aria-labelledby="dietas-ruta-resultado-titulo">
    <div class="cabecera-panel"><div><h4 id="dietas-ruta-resultado-titulo">${escaparHTML(t("ruta_resultado"))}</h4><span>${escaparHTML(t("ruta_motor_version", { motor: rutas.motor, version: rutas.version_grafo }))}</span></div><span class="estado-chip aviso">${escaparHTML(t("ruta_no_liquidable"))}</span></div>
    <dl class="dietas-ruta-totales"><div><dt>${escaparHTML(t("ruta_km_base"))}</dt><dd>${numero(rutas.kilometros_base, 1)} km</dd></div><div><dt>${escaparHTML(t("ruta_km_ajuste"))}</dt><dd>${numero(rutas.kilometros_ajuste, 1)} km</dd></div><div><dt>${escaparHTML(t("ruta_km_total"))}</dt><dd>${numero(rutas.kilometros_total, 1)} km</dd></div><div><dt>${escaparHTML(t("ruta_duracion"))}</dt><dd>${numero(rutas.duracion_minutos)} min</dd></div></dl>
    <div class="dietas-ruta-alternativas" aria-label="${escaparHTML(t("ruta_alternativas"))}">${alternativas}</div>
    <div class="tabla-contenedor"><table class="tabla-datos dietas-ruta-tramos"><caption>${escaparHTML(t("ruta_caption_tramos"))}</caption><thead><tr><th scope="col">${escaparHTML(t("ruta_tramo"))}</th><th scope="col">${escaparHTML(t("ruta_distancia"))}</th><th scope="col">${escaparHTML(t("ruta_tiempo"))}</th><th scope="col">${escaparHTML(t("ruta_ajuste_km"))}</th><th scope="col">${escaparHTML(t("ruta_motivo_ajuste"))}</th><th scope="col">${escaparHTML(t("ruta_accion"))}</th></tr></thead><tbody>${tramos}</tbody></table></div>
    ${rutas.mapa_ruta ? renderizarMapaRuta({ ruta: rutas.ruta, mapa_ruta: rutas.mapa_ruta }, t) : ""}
  </section>` : `<p class="dietas-ruta-pendiente" role="status">${escaparHTML(t("ruta_pendiente_calculo"))}</p>`;
  return `<fieldset class="dietas-ruta-herramienta campo-ancho"><legend>${escaparHTML(t("ruta_del_dia"))}</legend>
    <header class="dietas-ruta-catalogo"><div><strong>${escaparHTML(t("ruta_catalogo_provincial"))}</strong><span>${escaparHTML(t("ruta_catalogo_resumen", { total: puntos.length, version: rutas.catalogo.version }))}</span></div><span class="estado-chip ${rutas.catalogo.completo ? "exito" : "aviso"}">${escaparHTML(t(rutas.catalogo.completo ? "ruta_catalogo_completo" : "ruta_catalogo_parcial"))}</span></header>
    <div class="dietas-ruta-paradas">${paradas}</div>
    <div class="acciones-vista dietas-ruta-acciones"><button type="button" class="boton-secundario" data-dietas-ruta-anadir>${escaparHTML(t("ruta_anadir_parada"))}</button><button type="button" class="boton-primario" data-dietas-ruta-calcular>${escaparHTML(t("ruta_calcular_osrm"))}</button></div>
    <p class="dietas-ruta-ayuda">${escaparHTML(t("ruta_ayuda"))}</p>
    ${errorRuta ? `<p class="dietas-ruta-error" data-dietas-ruta-error role="alert">${escaparHTML(errorRuta)}</p>` : ""}${resultado}
  </fieldset>`;
}

function renderizarFormulario(modelo, t, errorRuta = "") {
  const inicial = modelo.borradorInicial;
  return `<details class="panel dietas-nueva" open>
    <summary>${escaparHTML(t("nueva_comision", { demo: modelo.demostracion ? " DEMO" : "" }))}</summary>
    <form data-dietas-formulario>
      <p class="dietas-aviso-formulario">${escaparHTML(t(modelo.demostracion ? "aviso_formulario_demo" : "aviso_formulario_real"))}</p>
      <label>${escaparHTML(t("fecha_inicio"))}<input name="fecha" type="date" value="${escaparHTML(inicial.fecha)}" required></label>
      <label>${escaparHTML(t("hora_inicio"))}<input name="hora_inicio" type="time" value="08:00" required></label>
      <label>${escaparHTML(t("fecha_fin"))}<input name="fecha_fin" type="date" value="${escaparHTML(inicial.fecha)}" required></label>
      <label>${escaparHTML(t("hora_fin"))}<input name="hora_fin" type="time" value="15:00" required></label>
      <label class="campo-ancho">${escaparHTML(t("motivo"))}<input name="motivo" maxlength="180" value="${escaparHTML(inicial.motivo)}" required></label>
      ${renderizarHerramientaRutas(modelo.herramientaRutas, t, errorRuta)}
      <label class="dietas-vehiculo-propio"><span>${escaparHTML(t("vehiculo"))}</span><span><input name="vehiculo_propio" type="checkbox" checked> ${escaparHTML(t("vehiculo_propio"))}</span></label>
      <label>${escaparHTML(t("tarifa_kilometro"))}<input name="tarifa_kilometro_euros" type="number" value="${escaparHTML(modelo.politica.tarifa_kilometro_euros)}" step="0.01" readonly></label>
      <label>${escaparHTML(t("manutencion"))}<input name="manutencion_euros" type="number" min="0" step="0.01" value="${escaparHTML(inicial.manutencion_euros)}"></label>
      <label>${escaparHTML(t("alojamiento"))}<input name="alojamiento_euros" type="number" min="0" step="0.01" value="${escaparHTML(inicial.alojamiento_euros)}"></label>
      <label>${escaparHTML(t("otros_gastos"))}<input name="otros_gastos_euros" type="number" min="0" step="0.01" value="${escaparHTML(inicial.otros_gastos_euros)}"></label>
      <p class="campo-ancho dietas-politica"><strong>${escaparHTML(t(modelo.demostracion ? "tarifa_escenario" : "tarifa_aplicable"))}:</strong> ${euros(modelo.politica.tarifa_kilometro_euros)}/${escaparHTML(t("unidad_km"))} · ${escaparHTML(modelo.politica.version)}</p>
      <dl class="campo-ancho dietas-borrador-resumen" aria-live="polite"><div><dt>${escaparHTML(t("ruta_km_total"))}</dt><dd data-dietas-resumen-borrador="kilometros">0,0 km</dd></div><div><dt>${escaparHTML(t("kilometraje"))}</dt><dd data-dietas-resumen-borrador="kilometraje">0,00 €</dd></div><div><dt>${escaparHTML(t("dietas_gastos"))}</dt><dd data-dietas-resumen-borrador="gastos">0,00 €</dd></div><div><dt>${escaparHTML(t("total"))}</dt><dd data-dietas-resumen-borrador="total">0,00 €</dd></div></dl>
      <div class="campo-ancho acciones-vista"><button type="submit" class="boton-primario" ${modelo.herramientaRutas?.lista_para_borrador ? "" : "disabled"}>${escaparHTML(t(modelo.herramientaRutas?.lista_para_borrador ? "guardar_borrador" : "ruta_calcular_antes_guardar", { demo: modelo.demostracion ? " DEMO" : "" }))}</button></div>
    </form>
  </details>`;
}

export function renderizarDietas(modelo, {
  descargaDisponible = false, confirmacionDisponible = false, mensajes = MENSAJES_DIETAS_ES,
  errorRuta = "",
} = {}) {
  const t = crearTraductorDietas(mensajes);
  const recibo = modelo.ultimoRecibo;
  const cabecera = `<header class="encabezado-vista"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2>${escaparHTML(t("titulo"))}</h2><p>${escaparHTML(t("descripcion"))}</p></div><span class="estado-chip ${modelo.demostracion ? "aviso" : "exito"}">${escaparHTML(t(modelo.demostracion ? "presentacion_sintetica" : "sesion_autorizada"))}</span></header>`;
  const identidad = `<section class="nota-seguridad dietas-identidad" aria-label="${escaparHTML(t("identidad_alcance"))}"><div><strong>${escaparHTML(modelo.identidad.actor.nombre_visible)}</strong><span>${escaparHTML(modelo.identidad.rol.etiqueta)}</span><small>${escaparHTML(t("contexto_compartido", { actor: modelo.identidad.actor.actor_ref }))}</small></div><p>${escaparHTML(modelo.politica.advertencia)}</p></section>`;
  if (!modelo.capacidades.consultarGastos) {
    return `<div class="modulo-dietas" data-modulo="dietas">${cabecera}${identidad}<section class="panel vacio-controlado" role="status"><h3>${escaparHTML(t("acceso_denegado_titulo"))}</h3><p>${escaparHTML(t("acceso_denegado_texto"))}</p></section></div>`;
  }
  return `<div class="modulo-dietas" data-modulo="dietas">
    ${cabecera}${identidad}
    <div class="rejilla-kpi dietas-kpi" aria-label="${escaparHTML(t("resumen_dietas"))}">
      <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">EXP</span><div><strong class="valor-kpi">${numero(modelo.resumen.expedientes)}</strong><span class="etiqueta-kpi">${escaparHTML(t("indicador_expedientes"))}</span></div></article>
      <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">PEN</span><div><strong class="valor-kpi">${numero(modelo.resumen.pendientes)}</strong><span class="etiqueta-kpi">${escaparHTML(t("indicador_pendientes"))}</span></div></article>
      <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">KM</span><div><strong class="valor-kpi">${modelo.capacidades.consultarRutas ? numero(modelo.resumen.kilometros, 1) : escaparHTML(t("sin_capacidad"))}</strong><span class="etiqueta-kpi">${escaparHTML(t("indicador_kilometros"))}</span></div></article>
      <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">DEV</span><div><strong class="valor-kpi">${euros(modelo.resumen.total_euros)}</strong><span class="etiqueta-kpi">${escaparHTML(t("indicador_devengado", { demo: modelo.demostracion ? " DEMO" : "" }))}</span></div></article>
      <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">PAG</span><div><strong class="valor-kpi">${euros(modelo.resumen.pagado_euros)}</strong><span class="etiqueta-kpi">${escaparHTML(t("indicador_pagado", { demo: modelo.demostracion ? " DEMO" : "" }))}</span></div></article>
    </div>
    ${recibo ? `<section class="dietas-recibo" role="status" tabindex="-1" data-dietas-recibo><strong>${escaparHTML(traducirCodigo(t, CLAVES_RESULTADO, recibo.resultado))}</strong><span>${escaparHTML(t("recibo_estado", { referencia: recibo.referencia, actor: recibo.actor_ref, efectos: recibo.efectos_reales === false ? t("sin_efectos") : "" }))}</span></section>` : '<div class="dietas-anuncio" role="status" aria-live="polite" data-dietas-anuncio></div>'}
    ${modelo.capacidades.gestionarGastos && modelo.capacidades.gestionarRutas ? renderizarFormulario(modelo, t, errorRuta) : ""}
    <form class="dietas-filtros" data-dietas-filtros aria-label="${escaparHTML(t("filtros_etiqueta"))}"><label>${escaparHTML(t("buscar"))}<input name="texto" type="search" value="${escaparHTML(modelo.filtros.texto)}" placeholder="${escaparHTML(t("buscar_placeholder"))}"></label><label>${escaparHTML(t("cab_estado"))}<select name="estado">${opcionesEstado(modelo.filtros.estado, t)}</select></label><button type="submit" class="boton-secundario">${escaparHTML(t("aplicar_filtros"))}</button></form>
    <div class="dietas-espacio-trabajo">${renderizarTabla(modelo, t)}${renderizarDetalle(modelo, descargaDisponible, confirmacionDisponible, t)}</div>
    ${renderizarHistorialMensual(modelo, descargaDisponible, t)}
  </div>`;
}

/**
 * Punto de montaje para el shell del portal.
 *
 * Ejemplo de composicion desde portal.js:
 *   await montarModuloDietas({ raiz, contextoActor, capacidades, adaptador,
 *     descargarRecibo, confirmarOperacion })
 *
 * En produccion se sustituye `crearDatos` por un adaptador autenticado; la
 * identidad sigue llegando del nucleo y nunca se resuelve dentro de Dietas.
 */
export async function montarModuloDietas({
  raiz,
  contextoActor,
  capacidades = [],
  adaptador,
  calculadorRuta,
  descargarRecibo,
  visorRuta,
  confirmarOperacion,
  origenComprobacion = globalThis.location?.origin || "",
  mensajes = MENSAJES_DIETAS_ES,
  anunciar = () => {},
} = {}) {
  if (!raiz || typeof raiz.replaceChildren !== "function") throw new Error("raiz de Dietas no valida");
  if (!adaptador || typeof adaptador.obtenerDatos !== "function" || typeof adaptador.ejecutar !== "function") {
    throw new Error("adaptador de Dietas no valido");
  }
  if (calculadorRuta !== undefined && (!calculadorRuta
    || typeof calculadorRuta.obtenerCatalogo !== "function" || typeof calculadorRuta.calcular !== "function")) {
    throw new Error("puerto de calculo de rutas de Dietas no valido");
  }
  if (descargarRecibo !== undefined && typeof descargarRecibo !== "function") {
    throw new Error("puerto descargarRecibo no valido");
  }
  if (confirmarOperacion !== undefined && typeof confirmarOperacion !== "function") {
    throw new Error("puerto confirmarOperacion no válido");
  }
  if (visorRuta !== undefined && (!visorRuta || typeof visorRuta.montar !== "function")) {
    throw new Error("puerto de mapa de Dietas no válido");
  }
  const t = crearTraductorDietas(mensajes);
  let catalogoRutas = null;
  if (calculadorRuta) {
    try {
      catalogoRutas = await calculadorRuta.obtenerCatalogo();
    } catch {
      anunciar(t("ruta_puerto_no_disponible"), "error");
    }
  }
  const presentador = crearPresentadorDietas({
    datos: await adaptador.obtenerDatos(), contextoActor, capacidades, origenComprobacion, catalogoRutas,
  });
  let activa = true;
  let ocupado = false;
  let visoresMontados = [];
  let borradorEdicion = null;
  let focoPendiente = null;
  let errorRuta = "";

  const atributosFoco = Object.freeze([
    "data-dietas-ruta-anadir",
    "data-dietas-ruta-quitar",
    "data-dietas-ruta-calcular",
    "data-dietas-ruta-parada",
    "data-dietas-ruta-alternativa",
    "data-dietas-ruta-motivo-alternativa",
    "data-dietas-ruta-aplicar-ajuste",
    "data-dietas-ruta-ajuste-km",
    "data-dietas-ruta-ajuste-motivo",
  ]);

  function capturarFoco() {
    const control = raiz.ownerDocument?.activeElement;
    if (!control || (typeof raiz.contains === "function" && !raiz.contains(control))) return null;
    for (const atributo of atributosFoco) {
      if (control.hasAttribute?.(atributo)) {
        return Object.freeze({ atributo, valor: control.getAttribute(atributo) || "" });
      }
    }
    return null;
  }

  function restaurarFoco(descriptor) {
    if (!descriptor) return;
    const candidatos = [...raiz.querySelectorAll(`[${descriptor.atributo}]`)];
    let control = candidatos.find((item) => !item.disabled
      && (item.getAttribute(descriptor.atributo) || "") === descriptor.valor);
    if (!control && descriptor.atributo === "data-dietas-ruta-quitar") {
      const indiceEliminado = Number(descriptor.valor);
      control = [...candidatos].reverse().find((item) => !item.disabled
        && Number(item.getAttribute(descriptor.atributo)) < indiceEliminado)
        || raiz.querySelector("[data-dietas-ruta-anadir]");
    }
    if (!control?.focus) return;
    const enfocar = () => {
      if (typeof raiz.contains !== "function" || raiz.contains(control)) {
        control.focus({ preventScroll: true });
      }
    };
    enfocar();
    // En un clic síncrono el navegador puede aplicar su foco por defecto
    // después de los escuchadores, cuando el botón original ya no existe.
    // Repetir en microtarea deja el foco en el control equivalente nuevo.
    const ventana = raiz.ownerDocument?.defaultView;
    const programar = typeof ventana?.queueMicrotask === "function"
      ? ventana.queueMicrotask.bind(ventana) : globalThis.queueMicrotask;
    programar?.(enfocar);
  }

  function capturarBorradorEdicion() {
    const formulario = raiz.querySelector("[data-dietas-formulario]");
    if (!formulario) return;
    borradorEdicion = Object.fromEntries([...formulario.querySelectorAll("[name]")].map((control) => [
      control.name,
      control.type === "checkbox" ? control.checked : control.value,
    ]));
  }

  function restaurarBorradorEdicion() {
    if (!borradorEdicion) return;
    const formulario = raiz.querySelector("[data-dietas-formulario]");
    if (!formulario) return;
    Object.entries(borradorEdicion).forEach(([nombre, valor]) => {
      const control = formulario.querySelector(`[name='${nombre}']`);
      if (!control) return;
      if (control.type === "checkbox") control.checked = valor === true;
      else control.value = String(valor);
    });
  }

  function desmontarVisor() {
    visoresMontados.forEach((visor) => visor?.desmontar?.());
    visoresMontados = [];
  }

  function pintar() {
    if (!activa) return;
    const foco = capturarFoco() || focoPendiente;
    desmontarVisor();
    const modelo = presentador.obtenerModelo();
    raiz.innerHTML = renderizarDietas(modelo, {
      descargaDisponible: typeof descargarRecibo === "function",
      confirmacionDisponible: typeof confirmarOperacion === "function",
      mensajes,
      errorRuta,
    });
    restaurarBorradorEdicion();
    if (visorRuta) {
      // La geometría histórica del expediente permanece en el modelo para
      // consulta y descarga, pero el espacio de trabajo muestra un único mapa:
      // el planificador activo de «Ruta del día». Evita dos visores Leaflet
      // simultáneos y recupera la jerarquía de la pantalla original de Dietas.
      const descriptores = [modelo.herramientaRutas?.mapa_ruta].filter(Boolean);
      const figuras = [...raiz.querySelectorAll("[data-dietas-mapa-ref]")];
      descriptores.forEach((descriptor) => {
        const figura = figuras.find((item) => item.dataset.dietasMapaRef === descriptor.vista_ref);
        if (figura) visoresMontados.push(visorRuta.montar({ raiz: figura, descriptor }));
      });
    }
    actualizarResumenBorrador();
    restaurarFoco(foco);
    if (foco === focoPendiente) focoPendiente = null;
  }

  function actualizarResumenBorrador() {
    const formulario = raiz.querySelector("[data-dietas-formulario]");
    if (!formulario) return;
    const modelo = presentador.obtenerModelo();
    const kilometrosRuta = Number(modelo.herramientaRutas?.kilometros_total || 0);
    const vehiculoPropio = formulario.querySelector("[name='vehiculo_propio']")?.checked === true;
    const kilometros = vehiculoPropio ? kilometrosRuta : 0;
    const valor = (nombre) => Number(formulario.querySelector(`[name='${nombre}']`)?.value || 0);
    const kilometraje = kilometros * Number(modelo.politica.tarifa_kilometro_euros || 0);
    const gastos = valor("manutencion_euros") + valor("alojamiento_euros") + valor("otros_gastos_euros");
    const valores = {
      kilometros: `${numero(kilometros, 1)} km`, kilometraje: euros(kilometraje),
      gastos: euros(gastos), total: euros(kilometraje + gastos),
    };
    Object.entries(valores).forEach(([clave, contenido]) => {
      const salida = formulario.querySelector(`[data-dietas-resumen-borrador='${clave}']`);
      if (salida) salida.textContent = contenido;
    });
  }

  function datosFormulario(formulario) {
    const datos = new FormData(formulario);
    return Object.fromEntries(datos.entries());
  }

  function marcarOcupado(valor) {
    ocupado = valor;
    if (valor) raiz.setAttribute("aria-busy", "true");
    else raiz.removeAttribute("aria-busy");
    raiz.querySelectorAll([
      "[data-dietas-enviar]", "[data-dietas-descargar-recibo]",
      "[data-dietas-descargar-anual]",
      "[data-dietas-ruta-calcular]", "[data-dietas-ruta-alternativa]", "[data-dietas-ruta-aplicar-ajuste]",
      "[data-dietas-formulario] button[type='submit']",
    ].join(",")).forEach((control) => {
      if (valor && !control.disabled) {
        control.disabled = true;
        control.dataset.dietasBloqueoTemporal = "true";
      } else if (!valor && control.dataset.dietasBloqueoTemporal === "true") {
        control.disabled = false;
        delete control.dataset.dietasBloqueoTemporal;
      }
    });
  }

  async function conBloqueo(tarea) {
    if (ocupado) throw new Error(t("operacion_en_curso"));
    focoPendiente = capturarFoco();
    marcarOcupado(true);
    try {
      return await tarea();
    } finally {
      marcarOcupado(false);
      if (focoPendiente) {
        restaurarFoco(focoPendiente);
        focoPendiente = null;
      }
    }
  }

  async function ejecutarComando(comando, mensaje, { reiniciarBorrador = false } = {}) {
    await conBloqueo(async () => {
      const siguientesDatos = await adaptador.ejecutar(comando);
      presentador.actualizarDatos(siguientesDatos);
      if (reiniciarBorrador) {
        borradorEdicion = null;
        presentador.rutas?.reiniciar();
      }
      pintar();
      raiz.querySelector("[data-dietas-recibo]")?.focus?.({ preventScroll: true });
      anunciar(mensaje);
    });
  }

  async function alClic(evento) {
    if (ocupado) {
      evento.preventDefault();
      anunciar(t("espere_operacion"));
      return;
    }
    const seleccionar = evento.target.closest?.("[data-dietas-seleccionar]");
    if (seleccionar) {
      presentador.seleccionar(seleccionar.dataset.dietasSeleccionar);
      pintar();
      raiz.querySelector("#dietas-titulo-detalle")?.focus?.({ preventScroll: true });
      return;
    }
    const anadirParada = evento.target.closest?.("[data-dietas-ruta-anadir]");
    if (anadirParada && presentador.rutas) {
      try {
        errorRuta = "";
        capturarBorradorEdicion();
        presentador.rutas.agregarParada();
        pintar();
        anunciar(t("ruta_parada_anadida"));
      } catch (error) { anunciar(error.message, "error"); }
      return;
    }
    const quitarParada = evento.target.closest?.("[data-dietas-ruta-quitar]");
    if (quitarParada && presentador.rutas) {
      try {
        errorRuta = "";
        capturarBorradorEdicion();
        presentador.rutas.eliminarParada(Number(quitarParada.dataset.dietasRutaQuitar));
        pintar();
        anunciar(t("ruta_parada_eliminada"));
      } catch (error) { anunciar(error.message, "error"); }
      return;
    }
    const calcularRuta = evento.target.closest?.("[data-dietas-ruta-calcular]");
    if (calcularRuta && presentador.rutas) {
      try {
        if (!calculadorRuta) throw new Error(t("ruta_puerto_no_disponible"));
        errorRuta = "";
        capturarBorradorEdicion();
        await conBloqueo(async () => {
          const solicitud = presentador.rutas.prepararSolicitudCalculo();
          const calculo = await calculadorRuta.calcular(solicitud);
          presentador.rutas.registrarCalculo(calculo);
          pintar();
          anunciar(t("ruta_calculo_completado"));
        });
      } catch (error) {
        errorRuta = t(error?.codigo === CODIGO_ERROR_SERVICIO_RUTAS_DIETAS
          ? "ruta_error_servicio" : "ruta_error_operacion");
        pintar();
        anunciar(errorRuta, "error");
      }
      return;
    }
    const alternativa = evento.target.closest?.("[data-dietas-ruta-alternativa]");
    if (alternativa && presentador.rutas) {
      try {
        capturarBorradorEdicion();
        const tarjeta = alternativa.closest(".dietas-ruta-alternativa");
        const motivo = tarjeta?.querySelector("[data-dietas-ruta-motivo-alternativa]")?.value || "";
        presentador.rutas.seleccionarAlternativa(alternativa.dataset.dietasRutaAlternativa, motivo);
        pintar();
        anunciar(t("ruta_alternativa_aplicada"));
      } catch (error) { anunciar(error.message, "error"); }
      return;
    }
    const aplicarAjuste = evento.target.closest?.("[data-dietas-ruta-aplicar-ajuste]");
    if (aplicarAjuste && presentador.rutas) {
      try {
        capturarBorradorEdicion();
        const indice = Number(aplicarAjuste.dataset.dietasRutaAplicarAjuste);
        const fila = aplicarAjuste.closest("[data-dietas-tramo]");
        presentador.rutas.ajustarTramo(
          indice,
          fila?.querySelector(`[data-dietas-ruta-ajuste-km="${indice}"]`)?.value || 0,
          fila?.querySelector(`[data-dietas-ruta-ajuste-motivo="${indice}"]`)?.value || "",
        );
        pintar();
        anunciar(t("ruta_ajuste_aplicado"));
      } catch (error) { anunciar(error.message, "error"); }
      return;
    }
    const enviar = evento.target.closest?.("[data-dietas-enviar]");
    if (enviar) {
      try {
        if (typeof confirmarOperacion !== "function") throw new Error(t("confirmacion_no_conectada"));
        const modelo = presentador.obtenerModelo();
        const item = modelo.comisiones.find((comision) => comision.referencia === enviar.dataset.dietasEnviar);
        const confirmado = await conBloqueo(() => confirmarOperacion(Object.freeze({
          tipo: "enviar_validacion",
          referencia: item?.referencia || enviar.dataset.dietasEnviar,
          titulo: t("confirmacion_titulo"),
          advertencia: t("confirmacion_advertencia"),
          estado: item?.estado || "",
          importe_euros: item?.total_euros ?? null,
          demostracion: modelo.demostracion,
        })));
        if (confirmado !== true) {
          anunciar(t("confirmacion_cancelada"));
          return;
        }
        await ejecutarComando(
          { tipo: "enviar_validacion", referencia: enviar.dataset.dietasEnviar },
          t("envio_demo_registrado"),
        );
      } catch (error) {
        anunciar(error.message, "error");
      }
      return;
    }
    const descargar = evento.target.closest?.("[data-dietas-descargar-recibo]");
    if (descargar && typeof descargarRecibo === "function") {
      try {
        await conBloqueo(async () => {
          const descriptor = presentador.prepararDescriptorRecibo(descargar.dataset.dietasDescargarRecibo, t);
          await descargarRecibo(descriptor);
          anunciar(t("recibo_generado", { referencia: descriptor.referencia }));
        });
      } catch (error) {
        anunciar(error.message, "error");
      }
      return;
    }
    const descargarAnual = evento.target.closest?.("[data-dietas-descargar-anual]");
    if (descargarAnual && typeof descargarRecibo === "function") {
      try {
        await conBloqueo(async () => {
          const descriptor = presentador.prepararDescriptorResumenAnual(
            Number(descargarAnual.dataset.dietasDescargarAnual), t,
          );
          await descargarRecibo(descriptor);
          anunciar(t("resumen_anual_generado", { anio: descriptor.periodo }));
        });
      } catch (error) {
        anunciar(error.message, "error");
      }
    }
  }

  async function alEnviar(evento) {
    if (ocupado) {
      evento.preventDefault();
      anunciar(t("espere_operacion"));
      return;
    }
    const filtros = evento.target.closest?.("[data-dietas-filtros]");
    if (filtros) {
      evento.preventDefault();
      const datos = datosFormulario(filtros);
      presentador.filtrar({ estado: datos.estado, texto: datos.texto });
      pintar();
      return;
    }
    const formulario = evento.target.closest?.("[data-dietas-formulario]");
    if (formulario) {
      evento.preventDefault();
      try {
        const rutaBorrador = presentador.rutas.prepararRutaBorrador();
        const vehiculoPropio = formulario.querySelector("[name='vehiculo_propio']")?.checked === true;
        await ejecutarComando(
              { tipo: "crear_borrador", campos: {
                ...datosFormulario(formulario),
                ...rutaBorrador,
                kilometros: vehiculoPropio ? rutaBorrador.kilometros : 0,
                vehiculo_propio: vehiculoPropio,
              } },
              t("borrador_demo_creado"),
              { reiniciarBorrador: true },
        );
      } catch (error) {
        anunciar(error.message, "error");
      }
    }
  }

  function alCambiar(evento) {
    const parada = evento.target.closest?.("[data-dietas-ruta-parada]");
    if (!parada || !presentador.rutas) {
      capturarBorradorEdicion();
      actualizarResumenBorrador();
      return;
    }
    try {
      errorRuta = "";
      capturarBorradorEdicion();
      presentador.rutas.establecerParada(Number(parada.dataset.dietasRutaParada), parada.value);
      pintar();
    } catch (error) { anunciar(error.message, "error"); }
  }

  function alEntrada(evento) {
    if (evento.target.closest?.("[data-dietas-formulario]")) {
      capturarBorradorEdicion();
      actualizarResumenBorrador();
    }
  }

  raiz.addEventListener("click", alClic);
  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("change", alCambiar);
  raiz.addEventListener("input", alEntrada);
  pintar();

  return Object.freeze({
    obtenerModelo: presentador.obtenerModelo,
    desmontar() {
      if (!activa) return;
      activa = false;
      desmontarVisor();
      raiz.removeEventListener("click", alClic);
      raiz.removeEventListener("submit", alEnviar);
      raiz.removeEventListener("change", alCambiar);
      raiz.removeEventListener("input", alEntrada);
      raiz.replaceChildren();
    },
  });
}
