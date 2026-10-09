/**
 * Canales de aviso de un llamamiento y vista «Seguimiento por teléfono».
 *
 * El llamamiento es uno: el correo sale al emitir y, después, RRHH puede llamar
 * a las personas en el orden de la bolsa y anotar cada llamada. Qué canales
 * están activos y qué resultados cierran el seguimiento lo publica el servidor
 * (`canales_llamamiento`); sin ese dato solo existe el correo.
 */
import { textoPortal, traducirBolsaInterna, traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { etiquetaResultadoTelefono } from "./portal-bolsas-intentos.js?v=20261008-canal-telefono-v2";
import { rutaCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js?v=20261009-ct-resultado-bolsa-v2";

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

/**
 * Filas del seguimiento en el orden de la bolsa. «Siguiente» es la primera
 * persona cuya última llamada no tiene un resultado de cierre.
 */
export function filasSeguimiento({ candidatos = [], contactos = [], llamamientoRef = "", telefono = null }) {
  const cierre = new Set(telefono?.resultados_cierre || []);
  const delLlamamiento = contactos.filter((contacto) => contacto?.llamamiento_ref === llamamientoRef);
  let siguienteMarcado = false;
  return [...candidatos].sort(porOrden).map((candidato) => {
    const propios = delLlamamiento.filter((contacto) => contacto.participacion_ref === candidato.participacion_ref);
    const llamadas = propios.filter((contacto) => contacto.canal === "telefono");
    const correo = propios.filter((contacto) => contacto.canal === "correo").reduce(masReciente, null);
    const ultimaLlamada = llamadas.reduce(masReciente, null);
    const cerrada = llamadas.some((contacto) => cierre.has(contacto.resultado));
    const siguiente = !cerrada && !siguienteMarcado;
    if (siguiente) siguienteMarcado = true;
    return Object.freeze({ candidato, correo, llamadas: llamadas.length, ultimaLlamada, cerrada, siguiente });
  });
}

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
export function enlaceSeguimiento({ bolsaRef, llamamientoRef, escaparHTML, clase, contenido, extra = "" }) {
  const href = rutaSegura(bolsaRef, "", { seguimiento: llamamientoRef });
  if (!href) return "";
  return `<a class="${clase}" href="${escaparHTML(href)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsaRef)}" data-seguimiento="${escaparHTML(llamamientoRef)}"${extra}>${contenido}</a>`;
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
  if (!telefono) return estadoPanel("status", `<p>${textoPortal("panel_seg_no_disponible")}</p>`);
  const filas = filasSeguimiento({
    candidatos: estadoCandidatos.datos.candidatos || [], contactos: estadoCandidatos.datos.contactos || [],
    llamamientoRef: seguimiento.llamamiento_ref, telefono,
  });
  if (filas.length === 0) return estadoPanel("status", `<p>${textoPortal("panel_seg_vacio")}</p>`);
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
        <td><button type="button" class="enlace-tabla" data-bolsa-accion="abrir-ficha" data-participacion-ref="${ref}" aria-label="${textoPortal("txt_aria_abrir_ficha_de", { persona: c.nombre_visible })}"><strong>${nombre}</strong></button>${fila.siguiente ? ` <span class="estado-chip info">${textoPortal("panel_seg_siguiente")}</span>` : ""}</td>
        <td>${correo}</td>
        <td>${llamadas}</td>
        <td><button type="button" class="${fila.siguiente ? "boton-primario" : "boton-secundario"}" data-bolsa-accion="abrir-ficha" data-bolsa-control-principal="true" data-participacion-ref="${ref}" aria-expanded="${fichaAbierta}" aria-controls="${escaparHTML(fichaId)}" aria-label="${textoPortal("panel_seg_llamar_aria", { persona: c.nombre_visible })}">${textoPortal("panel_seg_llamar")}</button></td>
      </tr>${fichaAbierta ? renderizarFicha(modalFicha, fichaId, 5) : ""}`;
  }).join("");
  return `${cabecera}
    <section class="panel" data-bolsa-b5-destino="true" tabindex="-1" aria-labelledby="bolsa-seguimiento-titulo">
      <div class="cabecera-panel"><h3 id="bolsa-seguimiento-titulo">${textoPortal("panel_seg_tabla")}</h3></div>
      <div class="tabla-contenedor" tabindex="0" role="region" aria-labelledby="bolsa-seguimiento-titulo">
        <table class="tabla-datos tabla-datos--candidatos">
          <thead><tr><th scope="col">${textoPortal("panel_seg_col_orden")}</th><th scope="col">${textoPortal("panel_seg_col_persona")}</th><th scope="col">${textoPortal("panel_seg_col_correo")}</th><th scope="col">${textoPortal("panel_seg_col_llamadas")}</th><th scope="col">${textoPortal("panel_seg_col_accion")}</th></tr></thead>
          <tbody>${cuerpo}</tbody>
        </table>
      </div>
    </section>`;
}
