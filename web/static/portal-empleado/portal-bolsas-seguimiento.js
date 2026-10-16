/**
 * Canales de aviso de un llamamiento y vista «Seguimiento por teléfono».
 *
 * El llamamiento es uno: el correo sale al emitir y, después, RRHH puede llamar
 * a las personas en el orden de la bolsa y anotar cada llamada. Qué canales
 * están activos y qué resultados cierran el seguimiento lo publica el servidor
 * (`canales_llamamiento`); sin ese dato solo existe el correo.
 */
import { textoPortal, traducirBolsaInterna, traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { etiquetaResultadoTelefono } from "./portal-bolsas-intentos.js?v=20261011-ct-bolsa-vinculo-v1";
import { rutaCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js?v=20261016-r1-seleccion-pendiente-v2";
import { origenLlamamientoValido } from "./portal-llamamiento-origen.js";
import { cargarTextos } from "../comun/textos.js";

const TEXTOS_EXCLUSIONES = (await cargarTextos("bolsa-ofertas")).seccion("seguimiento_exclusiones");

export function renderizarMotivosNoOfrecimiento(origen, { escaparHTML, numero, instanteVisible, candidatos = [], exclusionesDesde = 0 }) {
  const tx = (clave, variables = {}) => {
    let valor = TEXTOS_EXCLUSIONES[clave];
    for (const [nombre, reemplazo] of Object.entries(variables)) valor = valor.replaceAll(`{${nombre}}`, String(reemplazo));
    return escaparHTML(valor);
  };
  const contenido = (() => {
    if (!origen) return `<p>${tx("sin_lectura")}</p>`;
    if (origen.estado === "sin_permiso") return `<p>${tx("sin_permiso")}</p>`;
    if (origen.estado === "no_disponible") return `<p role="alert">${tx("no_disponible")}</p>` +
      `<div class="acciones-vista"><button type="button" class="boton-secundario" data-bolsa-accion="reintentar-candidatos">${escaparHTML(textoPortal("txt_reintentar"))}</button></div>`;
    const pagina = origen.pagina;
    if (!pagina?.exclusiones?.length) return `<p>${tx("vacio")}</p>`;
    const nombresPorParticipacion = new Map(candidatos.map((candidato) => [candidato.participacion_ref, candidato.nombre_visible]));
    const cuerpo = pagina.exclusiones.map((fila) => {
      const motivo = Object.hasOwn(TEXTOS_EXCLUSIONES, fila.motivo_clave)
        ? TEXTOS_EXCLUSIONES[fila.motivo_clave] : TEXTOS_EXCLUSIONES.motivo_no_disponible;
      const version = pagina.criterios_ofrecimiento?.version_politica_ofertas;
      const hechos = fila.hechos_relacion.length
        ? `<ul>${fila.hechos_relacion.map((hecho) => `<li>${tx(`hecho_${hecho.tipo}`)}: <time datetime="${escaparHTML(hecho.ocurrido_en)}">${escaparHTML(instanteVisible(hecho.ocurrido_en))}</time></li>`).join("")}</ul>`
        : `<p>${tx("sin_hechos")}</p>`;
      const detalle = `<details><summary>${tx("ver_fundamento", { numero: numero(fila.hechos_relacion.length) })}</summary>` +
        (version ? `<p>${tx("version_regla", { version: numero(version) })}</p>` : "") + hechos + `</details>`;
      const nombre = nombresPorParticipacion.get(fila.participacion_ref) || fila.nombre_visible;
      return `<tr><td>${numero(fila.orden_acta)}</td><td>${nombre
        ? escaparHTML(nombre) : tx("persona_sin_nombre", { posicion: numero(fila.orden_acta) })}</td><td>${escaparHTML(motivo)}${detalle}</td></tr>`;
    }).join("");
    return `<p>${tx("total", { numero: numero(pagina.total_exclusiones) })}</p>` +
      `<div class="tabla-contenedor" tabindex="0" role="region" aria-labelledby="bolsa-motivos-titulo"><table class="tabla-datos"><thead><tr><th scope="col">${tx("col_orden")}</th><th scope="col">${tx("col_persona")}</th><th scope="col">${tx("col_motivo")}</th></tr></thead><tbody>${cuerpo}</tbody></table></div>` +
      `<div class="acciones-vista">${exclusionesDesde > 0 ? `<button type="button" class="boton-secundario" data-bolsa-accion="primera-pagina-exclusiones">${tx("primera_pagina")}</button>` : ""}` +
      `${pagina.hay_mas ? `<button type="button" class="boton-secundario" data-bolsa-accion="pagina-exclusiones">${tx("pagina_siguiente")}</button>` : ""}</div>`;
  })();
  return `<section class="panel" aria-labelledby="bolsa-motivos-titulo"><div class="cabecera-panel"><h3 id="bolsa-motivos-titulo">${tx("titulo")}</h3></div><div class="cuerpo-panel">${contenido}</div></section>`;
}

/** Canales activos: el correo siempre (se envía al emitir); el teléfono si el servidor lo publica. */
export function canalesAviso(datos) {
  const lista = Array.isArray(datos?.canales_llamamiento) ? datos.canales_llamamiento : [];
  const telefono = lista.find((canal) => canal?.canal === "telefono" && canal.seguimiento === true
    && Array.isArray(canal.resultados) && canal.resultados.length > 0) || null;
  return Object.freeze({ correo: true, telefono });
}

function porOrden(izquierda, derecha) {
  const a = Number.isSafeInteger(izquierda.orden) ? izquierda.orden : Number.MAX_SAFE_INTEGER;
  const b = Number.isSafeInteger(derecha.orden) ? derecha.orden : Number.MAX_SAFE_INTEGER;
  return a - b;
}

function masReciente(actual, contacto) {
  return !actual || String(contacto.instante) > String(actual.instante) ? contacto : actual;
}

// Situaciones a las que se llega aceptando un llamamiento.
const SITUACIONES_ACEPTACION = new Set(["pendiente_incorporacion", "trabajando"]);

/**
 * Motivo por el que una persona ya no está pendiente de llamar, o null si lo
 * está: «aceptado» (aceptó por teléfono o en Mi bolsa, o ya se incorpora por
 * este llamamiento), «renuncia» (rechazó o renunció), «no_localizado» (otra
 * llamada de cierre, p. ej. número erróneo) o «no_disponible» (su situación en
 * la bolsa ya no es «disponible»).
 */
function motivoResuelta(candidato, llamadas, cierre, llamamientoRef, respuestaPortal) {
  const ultimo = candidato.ultimo_llamamiento?.llamamiento_ref === llamamientoRef ? candidato.ultimo_llamamiento : null;
  const cierres = llamadas.filter((contacto) => cierre.has(contacto.resultado)).map((contacto) => contacto.resultado);
  if (respuestaPortal === "acepta" || cierres.includes("acepta") || ultimo?.resultado === "aceptado"
    || (ultimo && SITUACIONES_ACEPTACION.has(candidato.estado_clave))) return "aceptado";
  if (respuestaPortal === "renuncia" || respuestaPortal === "renuncia_justificada"
    || cierres.includes("rechaza") || ultimo?.resultado === "renuncia" || candidato.estado_clave === "renuncia") return "renuncia";
  if (cierres.length) return "no_localizado";
  return candidato.estado_clave === "disponible" ? null : "no_disponible";
}

/**
 * Filas del seguimiento en el orden de la bolsa. «Siguiente» es la primera
 * persona que sigue pendiente de llamar.
 */
export function filasSeguimiento({ candidatos = [], contactos = [], respuestasPortal = [], llamamientoRef = "", telefono = null }) {
  const cierre = new Set(telefono?.resultados_cierre || []);
  const respuestaPorParticipacion = new Map(respuestasPortal.map((r) => [r.participacion_ref, r.respuesta]));
  const porParticipacion = new Map();
  for (const contacto of contactos) {
    if (contacto?.llamamiento_ref !== llamamientoRef) continue;
    const propios = porParticipacion.get(contacto.participacion_ref) || [];
    propios.push(contacto);
    porParticipacion.set(contacto.participacion_ref, propios);
  }
  let siguienteMarcado = false;
  const filas = [...candidatos].sort(porOrden).map((candidato) => {
    const propios = porParticipacion.get(candidato.participacion_ref) || [];
    const llamadas = propios.filter((contacto) => contacto.canal === "telefono");
    const correo = propios.filter((contacto) => contacto.canal === "correo").reduce(masReciente, null);
    const ultimaLlamada = llamadas.reduce(masReciente, null);
    const motivo = motivoResuelta(candidato, llamadas, cierre, llamamientoRef, respuestaPorParticipacion.get(candidato.participacion_ref));
    const cerrada = motivo !== null;
    const siguiente = !cerrada && !siguienteMarcado;
    if (siguiente) siguienteMarcado = true;
    return Object.freeze({ candidato, correo, llamadas: llamadas.length, ultimaLlamada, cerrada, motivo, siguiente });
  });
  // Una aceptación cierra el llamamiento completo, incluso si la persona
  // aparece después de otra todavía pendiente en el orden de la bolsa.
  return filas.some((fila) => fila.motivo === "aceptado")
    ? filas.map((fila) => Object.freeze({ ...fila, siguiente: false })) : filas;
}

const CLASE_MOTIVO = Object.freeze({ aceptado: "exito", renuncia: "advertencia", no_localizado: "advertencia", no_disponible: "neutro" });

function etiquetaResultado(contacto) {
  const telefonica = contacto.canal === "telefono" ? etiquetaResultadoTelefono(contacto.resultado) : null;
  if (telefonica) return telefonica;
  try {
    return traducirBolsaInterna(`contacto_${contacto.resultado}`);
  } catch {
    return contacto.resultado;
  }
}

/** Enlace (y botón) de vuelta a la lista de la bolsa, conservando la URL compartible. */
// Una referencia que no cabe en una URL compartible no rompe la pantalla: no se enlaza.
function rutaSegura(...argumentos) {
  try {
    return rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", ...argumentos);
  } catch {
    return "";
  }
}

function enlaceBolsa(bolsaRef, escaparHTML, clase, contenido, extra = "") {
  const href = rutaSegura(bolsaRef);
  if (!href) return "";
  return `<a class="${clase}" href="${escaparHTML(href)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsaRef)}"${extra}>${contenido}</a>`;
}

/** Enlace a la vista de seguimiento de un llamamiento concreto. */
// Con origen (petición de personal), el seguimiento lo conserva para que
// «Llamar al siguiente» abra el asistente con los mismos datos.
export function enlaceSeguimiento({ bolsaRef, llamamientoRef, escaparHTML, clase, contenido, extra = "", origen = null }) {
  const valido = origen ? origenLlamamientoValido(origen) : null;
  const href = rutaSegura(bolsaRef, "", { seguimiento: llamamientoRef, origen: valido });
  if (!href) return "";
  const atributo = (nombre, dato) => (dato ? ` data-origen-${nombre}="${escaparHTML(dato)}"` : "");
  const datosOrigen = valido ? `${atributo("expediente", valido.expediente_ref)}${atributo("referencia", valido.referencia)}${atributo("centro", valido.centro)}${atributo("inicio", valido.fecha_inicio)}` : "";
  return `<a class="${clase}" href="${escaparHTML(href)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsaRef)}" data-seguimiento="${escaparHTML(llamamientoRef)}"${datosOrigen}${extra}>${contenido}</a>`;
}

export function renderizarSeguimientoLlamamiento({
  estadoCandidatos, seguimiento, modalFicha, encabezadoVista, escaparHTML, numero, instanteVisible, renderizarFicha,
}) {
  const bolsaRef = seguimiento.bolsa_ref;
  const bolsa = estadoCandidatos?.datos?.bolsa;
  const titulo = bolsa ? traducirPortal("panel_seg_titulo_bolsa", { categoria: bolsa.categoria }) : traducirPortal("panel_seg_titulo");
  const volver = bolsaRef ? enlaceBolsa(bolsaRef, escaparHTML, "boton-secundario", textoPortal("panel_seg_volver")) : "";
  const cabecera = encabezadoVista("", titulo, "", volver);
  const estadoPanel = (rol, contenido) => `${cabecera}<section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
    <div class="cuerpo-panel vacio-controlado" role="${rol}"${rol === "status" && estadoCandidatos?.carga === "cargando" ? ' aria-busy="true"' : ""}>${contenido}</div></section>`;
  const carga = estadoCandidatos?.carga || "cargando";
  if (carga === "cargando") return estadoPanel("status", `<p><strong>${textoPortal("panel_seg_cargando")}</strong></p>`);
  if (carga === "denegado") {
    return estadoPanel("alert", `<p><strong>${textoPortal("txt_acceso_denegado")}</strong></p><p>${textoPortal("txt_la_sesion_no_dispone_de_permisos_para_consultar")}</p>`);
  }
  if (carga !== "listo") {
    return estadoPanel("alert", `<p><strong>${textoPortal("panel_seg_error_titulo")}</strong></p><p>${escaparHTML(estadoCandidatos.error || traducirPortal("panel_seg_error"))}</p>
      <div class="acciones-vista"><button type="button" class="boton-primario" data-bolsa-accion="reintentar-candidatos">${textoPortal("txt_reintentar")}</button></div>`);
  }
  const { telefono } = canalesAviso(estadoCandidatos.datos);
  const motivos = renderizarMotivosNoOfrecimiento(estadoCandidatos.datos.exclusiones_origen,
    { escaparHTML, numero, instanteVisible, candidatos: estadoCandidatos.datos.candidatos || [],
      exclusionesDesde: estadoCandidatos.exclusionesDesde || 0 });
  if (!telefono) return `${estadoPanel("status", `<p>${textoPortal("panel_seg_no_disponible")}</p>`)}${motivos}`;
  const respuestas = estadoCandidatos.datos.respuestas_portal;
  if (respuestas?.llamamiento_ref !== seguimiento.llamamiento_ref || !Array.isArray(respuestas.respuestas)) {
    return estadoPanel("alert", `<p><strong>${textoPortal("panel_seg_error_titulo")}</strong></p><p>${textoPortal("panel_seg_error")}</p>
      <div class="acciones-vista"><button type="button" class="boton-primario" data-bolsa-accion="reintentar-candidatos">${textoPortal("txt_reintentar")}</button></div>`);
  }
  const filas = filasSeguimiento({
    candidatos: estadoCandidatos.datos.candidatos || [], contactos: estadoCandidatos.datos.contactos || [],
    respuestasPortal: respuestas.respuestas, llamamientoRef: seguimiento.llamamiento_ref, telefono,
  });
  if (filas.length === 0) return `${estadoPanel("status", `<p>${textoPortal("panel_seg_vacio")}</p>`)}${motivos}`;
  // Si alguien aceptó, el llamamiento está cubierto. Si nadie aceptó y nadie
  // queda pendiente, el siguiente paso es otro llamamiento de esta bolsa: el
  // asistente existente («iniciar-b7») con el origen de la petición si se conoce.
  const aceptada = filas.find((fila) => fila.motivo === "aceptado");
  const sinPendientes = !aceptada && !filas.some((fila) => fila.siguiente);
  const chipPendientes = aceptada
    ? `<span class="estado-chip exito">${escaparHTML(traducirPortal("panel_seg_aceptado_por", { persona: aceptada.candidato.nombre_visible }))}</span>`
    : sinPendientes ? `<span class="estado-chip info">${textoPortal("panel_seg_sin_pendientes")}</span>` : "";
  const siguiente = sinPendientes
    ? `<button type="button" class="boton-primario" data-bolsa-accion="iniciar-b7">${textoPortal("panel_seg_llamar_siguiente")}</button>` : "";
  const momento = (contacto) => `<br><small><time datetime="${escaparHTML(contacto.instante)}">${escaparHTML(instanteVisible(contacto.instante))}</time></small>`;
  const cuerpo = filas.map((fila) => {
    const c = fila.candidato;
    const ref = escaparHTML(c.participacion_ref);
    const nombre = escaparHTML(c.nombre_visible);
    const fichaAbierta = modalFicha?.abierto === true && modalFicha.candidato?.participacion_ref === c.participacion_ref;
    const fichaId = `ficha-participacion-${c.participacion_ref}`;
    const correo = fila.correo ? `${escaparHTML(etiquetaResultado(fila.correo))}${momento(fila.correo)}` : `<span class="texto-atenuado">${textoPortal("panel_seg_sin_correo")}</span>`;
    const llamadas = fila.llamadas
      ? `<button type="button" class="enlace-tabla" data-bolsa-accion="abrir-ficha" data-participacion-ref="${ref}" aria-label="${textoPortal("txt_aria_abrir_ficha_de", { persona: c.nombre_visible })}">${textoPortal("panel_seg_llamadas_n", { numero: numero(fila.llamadas) })}</button><br>${escaparHTML(etiquetaResultado(fila.ultimaLlamada))}${momento(fila.ultimaLlamada)}`
      : `<span class="texto-atenuado">${textoPortal("panel_seg_sin_llamadas")}</span>`;
    return `<tr class="fila-candidato" data-participacion-ref="${ref}">
        <td><strong>${c.orden === null ? "—" : `#${numero(c.orden)}`}</strong></td>
        <td><button type="button" class="enlace-tabla" data-bolsa-accion="abrir-ficha" data-participacion-ref="${ref}" aria-label="${textoPortal("txt_aria_abrir_ficha_de", { persona: c.nombre_visible })}"><strong>${nombre}</strong></button>${fila.siguiente ? ` <span class="estado-chip info">${textoPortal("panel_seg_siguiente")}</span>` : ""}${fila.motivo ? ` <span class="estado-chip ${CLASE_MOTIVO[fila.motivo]}">${textoPortal(`panel_seg_motivo_${fila.motivo}`)}</span>` : ""}</td>
        <td>${correo}</td>
        <td>${llamadas}</td>
        <td><button type="button" class="${fila.siguiente ? "boton-primario" : "boton-secundario"}" data-bolsa-accion="abrir-ficha" data-bolsa-control-principal="true" data-participacion-ref="${ref}" aria-expanded="${fichaAbierta}" aria-controls="${escaparHTML(fichaId)}" aria-label="${textoPortal("panel_seg_llamar_aria", { persona: c.nombre_visible })}">${textoPortal("panel_seg_llamar")}</button></td>
      </tr>${fichaAbierta ? renderizarFicha(modalFicha, fichaId, 5) : ""}`;
  }).join("");
  return `${siguiente ? encabezadoVista("", titulo, "", `${volver}${siguiente}`) : cabecera}
    <section class="panel" data-bolsa-b5-destino="true" tabindex="-1" aria-labelledby="bolsa-seguimiento-titulo">
      <div class="cabecera-panel"><h3 id="bolsa-seguimiento-titulo">${textoPortal("panel_seg_tabla")}</h3>${chipPendientes}</div>
      <div class="tabla-contenedor" tabindex="0" role="region" aria-labelledby="bolsa-seguimiento-titulo">
        <table class="tabla-datos tabla-datos--candidatos">
          <thead><tr><th scope="col">${textoPortal("panel_seg_col_orden")}</th><th scope="col">${textoPortal("panel_seg_col_persona")}</th><th scope="col">${textoPortal("panel_seg_col_correo")}</th><th scope="col">${textoPortal("panel_seg_col_llamadas")}</th><th scope="col">${textoPortal("panel_seg_col_accion")}</th></tr></thead>
          <tbody>${cuerpo}</tbody>
        </table>
      </div>
    </section>${motivos}`;
}
