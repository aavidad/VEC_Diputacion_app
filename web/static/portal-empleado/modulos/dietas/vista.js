import { crearPresentadorDietas } from "./presentador.js";
import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";

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

function proyectarPuntosCroquis(geometria) {
  const puntos = geometria.trazado.map(([latitud, longitud]) => ({ latitud, longitud }));
  const latitudes = puntos.map((punto) => punto.latitud);
  const longitudes = puntos.map((punto) => punto.longitud);
  let minLatitud = Math.min(...latitudes);
  let maxLatitud = Math.max(...latitudes);
  let minLongitud = Math.min(...longitudes);
  let maxLongitud = Math.max(...longitudes);
  if (minLatitud === maxLatitud) { minLatitud -= 0.01; maxLatitud += 0.01; }
  if (minLongitud === maxLongitud) { minLongitud -= 0.01; maxLongitud += 0.01; }
  const proyectar = ({ latitud, longitud }) => ({
    x: 42 + ((longitud - minLongitud) / (maxLongitud - minLongitud)) * 556,
    y: 32 + ((maxLatitud - latitud) / (maxLatitud - minLatitud)) * 176,
  });
  return { proyectar, trazado: puntos.map(proyectar) };
}

function renderizarMapaRuta(item, t) {
  if (!item.mapa_ruta) return "";
  const { geometria } = item.mapa_ruta;
  const { proyectar, trazado } = proyectarPuntosCroquis(geometria);
  const rejilla = [1, 2, 3, 4, 5].map((indice) => {
    const x = 42 + indice * (556 / 6);
    const y = 32 + indice * (176 / 6);
    return `<line x1="${x.toFixed(1)}" y1="32" x2="${x.toFixed(1)}" y2="208"></line><line x1="42" y1="${y.toFixed(1)}" x2="598" y2="${y.toFixed(1)}"></line>`;
  }).join("");
  const etiquetasVistas = new Set();
  const paradas = geometria.paradas.map((parada, indice) => {
    const punto = proyectar(parada);
    const clave = `${parada.latitud}:${parada.longitud}:${parada.etiqueta}`;
    const mostrarEtiqueta = !etiquetasVistas.has(clave);
    etiquetasVistas.add(clave);
    return `<g><circle cx="${punto.x.toFixed(1)}" cy="${punto.y.toFixed(1)}" r="7"></circle><text class="dietas-mapa-numero" x="${punto.x.toFixed(1)}" y="${(punto.y + 3.5).toFixed(1)}">${indice + 1}</text>${mostrarEtiqueta ? `<text class="dietas-mapa-etiqueta" x="${Math.min(560, punto.x + 11).toFixed(1)}" y="${Math.max(18, punto.y - 10).toFixed(1)}">${escaparHTML(parada.etiqueta)}</text>` : ""}</g>`;
  }).join("");
  return `<figure class="dietas-mapa" aria-labelledby="dietas-mapa-titulo" aria-describedby="dietas-mapa-nota">
    <figcaption><div><p class="sobrelinea">${escaparHTML(t("mapa_proveedor"))}</p><h4 id="dietas-mapa-titulo">${escaparHTML(t("mapa_titulo"))}</h4></div><span class="estado-chip aviso">${escaparHTML(t("mapa_marca_demo"))}</span></figcaption>
    <div class="dietas-mapa-canvas" data-dietas-mapa-canvas>
      <svg viewBox="0 0 640 240" role="img" aria-labelledby="dietas-croquis-titulo dietas-croquis-descripcion">
        <title id="dietas-croquis-titulo">${escaparHTML(t("mapa_croquis_titulo"))}</title>
        <desc id="dietas-croquis-descripcion">${escaparHTML(t("mapa_croquis_descripcion", { ruta: item.ruta.join(" → ") }))}</desc>
        <g class="dietas-mapa-rejilla">${rejilla}</g>
        <polyline points="${trazado.map((punto) => `${punto.x.toFixed(1)},${punto.y.toFixed(1)}`).join(" ")}"></polyline>
        <g class="dietas-mapa-paradas">${paradas}</g>
      </svg>
    </div>
    <p id="dietas-mapa-nota" class="dietas-mapa-nota" data-dietas-mapa-estado>${escaparHTML(t("mapa_nota_fallback"))}</p>
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
      ${modelo.capacidades.consultarRutas ? renderizarMapaRuta(item, t) : ""}
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

function renderizarFormulario(modelo, t) {
  const inicial = modelo.borradorInicial;
  return `<details class="panel dietas-nueva">
    <summary>${escaparHTML(t("nueva_comision", { demo: modelo.demostracion ? " DEMO" : "" }))}</summary>
    <form data-dietas-formulario>
      <p class="dietas-aviso-formulario">${escaparHTML(t(modelo.demostracion ? "aviso_formulario_demo" : "aviso_formulario_real"))}</p>
      <label>${escaparHTML(t("fecha"))}<input name="fecha" type="date" value="${escaparHTML(inicial.fecha)}" required></label>
      <label class="campo-ancho">${escaparHTML(t("motivo"))}<input name="motivo" maxlength="180" value="${escaparHTML(inicial.motivo)}" required></label>
      <label>${escaparHTML(t("origen"))}<input name="origen" maxlength="80" value="${escaparHTML(inicial.origen)}" required></label>
      <label>${escaparHTML(t("destino"))}<input name="destino" maxlength="80" value="${escaparHTML(inicial.destino)}" required></label>
      <label>${escaparHTML(t("kilometros_ruta"))}<input name="kilometros" type="number" min="0" step="0.1" value="${escaparHTML(inicial.kilometros)}" required></label>
      <label>${escaparHTML(t("manutencion"))}<input name="manutencion_euros" type="number" min="0" step="0.01" value="${escaparHTML(inicial.manutencion_euros)}"></label>
      <label>${escaparHTML(t("alojamiento"))}<input name="alojamiento_euros" type="number" min="0" step="0.01" value="${escaparHTML(inicial.alojamiento_euros)}"></label>
      <label>${escaparHTML(t("otros_gastos"))}<input name="otros_gastos_euros" type="number" min="0" step="0.01" value="${escaparHTML(inicial.otros_gastos_euros)}"></label>
      <p class="campo-ancho dietas-politica"><strong>${escaparHTML(t(modelo.demostracion ? "tarifa_escenario" : "tarifa_aplicable"))}:</strong> ${euros(modelo.politica.tarifa_kilometro_euros)}/${escaparHTML(t("unidad_km"))} · ${escaparHTML(modelo.politica.version)}</p>
      <div class="campo-ancho acciones-vista"><button type="submit" class="boton-primario">${escaparHTML(t("guardar_borrador", { demo: modelo.demostracion ? " DEMO" : "" }))}</button></div>
    </form>
  </details>`;
}

export function renderizarDietas(modelo, {
  descargaDisponible = false, confirmacionDisponible = false, mensajes = MENSAJES_DIETAS_ES,
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
    <form class="dietas-filtros" data-dietas-filtros aria-label="${escaparHTML(t("filtros_etiqueta"))}"><label>${escaparHTML(t("buscar"))}<input name="texto" type="search" value="${escaparHTML(modelo.filtros.texto)}" placeholder="${escaparHTML(t("buscar_placeholder"))}"></label><label>${escaparHTML(t("cab_estado"))}<select name="estado">${opcionesEstado(modelo.filtros.estado, t)}</select></label><button type="submit" class="boton-secundario">${escaparHTML(t("aplicar_filtros"))}</button></form>
    <div class="dietas-espacio-trabajo">${renderizarTabla(modelo, t)}${renderizarDetalle(modelo, descargaDisponible, confirmacionDisponible, t)}</div>
    ${modelo.capacidades.gestionarGastos && modelo.capacidades.gestionarRutas ? renderizarFormulario(modelo, t) : ""}
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
  const presentador = crearPresentadorDietas({
    datos: await adaptador.obtenerDatos(), contextoActor, capacidades, origenComprobacion,
  });
  let activa = true;
  let ocupado = false;
  let visorMontado = null;

  function desmontarVisor() {
    visorMontado?.desmontar?.();
    visorMontado = null;
  }

  function pintar() {
    if (!activa) return;
    desmontarVisor();
    const modelo = presentador.obtenerModelo();
    raiz.innerHTML = renderizarDietas(modelo, {
      descargaDisponible: typeof descargarRecibo === "function",
      confirmacionDisponible: typeof confirmarOperacion === "function",
      mensajes,
    });
    if (visorRuta && modelo.seleccionada?.mapa_ruta) {
      visorMontado = visorRuta.montar({ raiz, descriptor: modelo.seleccionada.mapa_ruta });
    }
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
    marcarOcupado(true);
    try {
      return await tarea();
    } finally {
      marcarOcupado(false);
    }
  }

  async function ejecutarComando(comando, mensaje) {
    await conBloqueo(async () => {
      const siguientesDatos = await adaptador.ejecutar(comando);
      presentador.actualizarDatos(siguientesDatos);
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
        await ejecutarComando(
              { tipo: "crear_borrador", campos: datosFormulario(formulario) },
              t("borrador_demo_creado"),
        );
      } catch (error) {
        anunciar(error.message, "error");
      }
    }
  }

  raiz.addEventListener("click", alClic);
  raiz.addEventListener("submit", alEnviar);
  pintar();

  return Object.freeze({
    obtenerModelo: presentador.obtenerModelo,
    desmontar() {
      if (!activa) return;
      activa = false;
      desmontarVisor();
      raiz.removeEventListener("click", alClic);
      raiz.removeEventListener("submit", alEnviar);
      raiz.replaceChildren();
    },
  });
}
