/**
 * Presentador del contrato agregado del panel interno de Bolsa, del cuadro de bolsas
 * y de la lista de candidatos.
 *
 * Conoce `vec.bolsa.panel.interno.v1`, `vec.bolsa.rrhh.bolsas.v1` y `vec.bolsa.rrhh.candidatos.v1/v2`.
 * Los datos proceden de los contratos conectados; una fuente no configurada se presenta como tal.
 * Recibe las utilidades visuales para mantener este módulo puro y comprobable
 * sin acceder al DOM global.
 */
import { finVigenciaBolsaPortal, LOCALIZACION_PORTAL, textoPortal, traducirBolsaInterna, traducirPortal, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { renderizarBloqueAvisos } from "./portal-bolsas-avisos.js?v=20261007-pantallas-textos-final-v1";
import { renderizarChipsMarcas, renderizarMarcasFicha, seleccionableEnLlamamiento, traducirMarcasBolsa } from "./portal-bolsas-marcas.js?v=20261007-pantallas-textos-final-v1";
import { renderizarOperacionesSituacion } from "./portal-bolsas-operaciones.js?v=20261008-w-bolsa-ficha-nominal-v1";
import { destinosSituacion, fechaDisponiblePropuesta, renderizarCamposReposicion } from "./portal-bolsas-reglas-situacion.js?v=20260930-portales-i18n-integracion-v1";
import { renderizarIntentosContacto } from "./portal-bolsas-intentos.js?v=20261008-canal-telefono-v2";
import { canalesAviso, enlaceSeguimiento, renderizarSeguimientoLlamamiento } from "./portal-bolsas-seguimiento.js";
import { renderizarContratosParticipacion } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
import { renderizarReincorporacionesTitular } from "./portal-bolsas-reincorporaciones.js?v=20261008-w-bolsa-ficha-nominal-v1";
import { renderizarSanciones } from "./portal-bolsas-sanciones.js?v=20261008-w-bolsa-ficha-nominal-v1";
import { renderizarAvisosContactoEmision, renderizarOrigenContacto } from "./portal-bolsas-contacto-origen.js?v=20261007-pantallas-textos-final-v1";
import { renderizarRegistroContacto } from "./portal-bolsas-contacto-registro.js?v=20261007-pantallas-textos-final-v1";
import { traducirAvisoPanelInterno } from "./portal-panel-interno-i18n.js?v=20260930-portales-i18n-integracion-v1";
import { traducirEnlacesBolsa } from "./portal-enlaces-i18n.js?v=20260930-portales-i18n-integracion-v1";
import { icono } from "../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { actorTraducido, justificanteTraducido, referenciaCopiableTraducida } from "./portal-justificante.js";
import { tieneTextoReferencia, traducirReferencia } from "./portal-referencias-i18n.js?v=20261007-pantallas-textos-final-v1";


const REPOSICIONES_CONOCIDAS = new Set(["misma_posicion", "fin_lista", "no_disponible_hasta_fecha"]);
import { RUTA_PANTALLA_REGLAS } from "./reglas/enlace.js?v=20261007-pantallas-textos-final-v1";
import { renderizarMarcadoresCorreo, renderizarVistaPreviaCorreo } from "./portal-bolsas-correo.js?v=20260930-portales-i18n-integracion-v1";
import { rutaCandidatosBolsaCompartible, rutaResumenBolsasCompartible } from "./portal-bolsas-ruta-filtros.js";
const ESQUEMA_PANEL_INTERNO = "vec.bolsa.panel.interno.v1";
const RUTA_PETICIONES_PERSONAL_TEMPORAL = "/portal-empleado/#contratacion-temporal"; // la aceptación o renuncia se registra en su expediente, no en Bolsa
const ESTADOS_BOLSA = Object.freeze(["disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision"]);
export function crearPresentadorPanelInterno(dependencias) {
  const {
    claseEstado,
    encabezadoVista,
    escaparHTML,
    numero,
    obtenerDatosPanel,
    tituloVista,
    obtenerDatosBolsas,
    obtenerDatosCandidatosBolsa,
    obtenerDatosEstadisticas,
    obtenerDatosAvisos,
    obtenerEstadoCandidatos,
    obtenerModalContactos,
    obtenerModalFicha,
    obtenerModalResultado,
  } = dependencias;
  if ([claseEstado, encabezadoVista, escaparHTML, numero, obtenerDatosPanel, tituloVista]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new Error("dependencias del presentador de panel interno no válidas");
  }
  function datosPanel() {
    return obtenerDatosPanel();
  }
  function esActivo() {
    return datosPanel()?.esquema === ESQUEMA_PANEL_INTERNO;
  }
  function etiquetaFuente() {
    return esActivo() ? traducirPortal("txt_panel_interno_agregado_autorizado") : "";
  }
  // Con «destino», el recuadro es un botón que abre la lista correspondiente.
  function tarjetaKPI(nombreIcono, valor, etiqueta, destino = null) {
    const contenido = `
        <span class="icono-kpi" aria-hidden="true">${icono(nombreIcono)}</span>
        <div><strong class="valor-kpi">${escaparHTML(valor)}</strong><span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span>`;
    if (!destino) return `
      <article class="tarjeta-kpi">${contenido}</div>
      </article>`;
    return `
      <button type="button" class="tarjeta-kpi" ${destino.atributos} aria-label="${escaparHTML(destino.aria)}">${contenido}
        <span class="metrica-enlace" aria-hidden="true">${escaparHTML(destino.enlace)}</span></div>
      </button>`;
  }
  // El rótulo de la política llega del servidor; si marca una regla aún provisional
  // no se muestra en la pantalla de trabajo: su origen consta en «Reglas vigentes».
  function rotuloPoliticaOrden(rotulo) {
    const texto = String(rotulo || "").trim();
    if (!texto) return "";
    if (/pendiente|provisional/iu.test(texto)) return "";
    return `<p><strong>${escaparHTML(texto)}</strong></p>`;
  }
  function etiquetaClave(clave) {
    const texto = String(clave || "").replaceAll(/[._-]+/g, " ").trim();
    return texto ? texto.charAt(0).toLocaleUpperCase(LOCALIZACION_PORTAL) + texto.slice(1) : traducirPortal("txt_sin_clave");
  }
  function etiquetaIntentoTurno(clave, tipo) {
    const deContacto = tipo === "canal"
      ? ["telefono", "correo", "sms", "presencial", "otro"]
      : ["contactado", "no_contesta", "buzon", "acepta", "rechaza", "aplazado", "otro", "enviado", "no_enviado"];
    return deContacto.includes(clave)
      ? traducirBolsaInterna(`contacto_${clave}`)
      : traducirPortal(`bolsa_turno_${tipo}_${clave}`);
  }
  function etiquetaReposicion(clave) {
    return REPOSICIONES_CONOCIDAS.has(clave) ? traducirBolsaInterna(`bolsa_reposicion_${clave}`) : etiquetaClave(clave);
  }
  function etiquetaEstadoBolsa(estado) {
    if (estado === "en_revision") return traducirPortal("txt_b8_en_revision");
    const clave = `bolsa_estado_${estado}`;
    const traducida = traducirBolsaInterna(clave);
    return traducida === clave ? etiquetaClave(estado) : traducida;
  }
  function controlEstadoBolsa(bolsa, estado, clase = "neutro") {
    const total = numero(bolsa.por_estado?.[estado]);
    const etiqueta = etiquetaEstadoBolsa(estado);
    const href = rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", bolsa.bolsa_ref, estado);
    return `<a class="estado-chip ${escaparHTML(clase)}" href="${escaparHTML(href)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}" data-estado="${escaparHTML(estado)}" aria-label="${textoPortal("txt_aria_ver_candidatos_estado", { total, estado: etiqueta.toLocaleLowerCase(LOCALIZACION_PORTAL), categoria: bolsa.categoria })}">${escaparHTML(total)}</a>`;
  }
  function claseEstadoEstadistica(estado) {
    return ({ disponible: "exito", no_disponible: "peligro", excluido: "peligro", pendiente_incorporacion: "info", disponible_desde: "info", en_revision: "advertencia" })[estado] || "neutro";
  }
  function etiquetaEstadoEstadistica(estado) { return etiquetaEstadoBolsa(estado); }
  function renderizarEstadisticasBolsa() {
    const estadoEstadisticas = typeof obtenerDatosEstadisticas === "function" ? obtenerDatosEstadisticas() : null;
    const cabecera = encabezadoVista("", traducirPortal("txt_estadisticas"), "", '<button type="button" class="boton-secundario" data-vista="resumen">' + textoPortal("txt_volver_al_cuadro") + '</button>');
    if (!estadoEstadisticas || estadoEstadisticas.carga === "cargando") return `${cabecera}<section class="panel"><div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true"><p><strong>${textoPortal("txt_cargando_estadisticas_de_bolsa")}</strong></p></div></section>`;
    if (estadoEstadisticas.carga === "denegado" || estadoEstadisticas.carga === "error") return `${cabecera}<section class="panel"><div class="cuerpo-panel vacio-controlado" role="alert"><p><strong>${estadoEstadisticas.carga === "denegado" ? traducirPortal("txt_acceso_denegado") : traducirPortal("txt_no_se_pudieron_cargar_las_estadisticas")}</strong></p><p>${escaparHTML(estadoEstadisticas.error || traducirPortal("txt_la_sesion_no_puede_consultar_esta_informacion"))}</p>${estadoEstadisticas.carga === "error" ? '<button type="button" class="boton-secundario" data-bolsa-accion="reintentar-estadisticas">' + textoPortal("txt_reintentar") + '</button>' : ""}</div></section>`;
    const datos = estadoEstadisticas.datos;
    const lecturaBolsas = typeof obtenerDatosBolsas === "function" ? obtenerDatosBolsas() : null;
    const conteosCurso = lecturaBolsas?.carga === "listo" && Array.isArray(lecturaBolsas.datos?.bolsas)
      ? lecturaBolsas.datos.bolsas.map((bolsa) => bolsa.llamamientos_en_curso) : null;
    const totalEnCurso = conteosCurso?.every((total) => Number.isSafeInteger(total) && total >= 0)
      ? conteosCurso.reduce((suma, total) => suma + total, 0) : null;
    const enCurso = Number.isSafeInteger(totalEnCurso) ? numero(totalEnCurso) : traducirPortal("txt_no_disponible");
    const tarjetas = [["bolsas", numero(datos.bolsas.total), traducirPortal("txt_bolsas")], ["correcto", numero(datos.bolsas.vigentes), traducirPortal("txt_vigentes")], ["en_curso", numero(datos.bolsas.sustituidas), traducirPortal("txt_sustituidas")], ["personas", numero(datos.personas.total), traducirPortal("txt_personas")], ["llamamiento", enCurso, traducirPortal("txt_llamamientos_en_curso")]];
    const resumenEstados = Object.entries(datos.personas.por_estado).map(([estado, total]) => `<span class="estado-chip ${claseEstadoEstadistica(estado)}"><strong>${numero(total)}</strong> ${escaparHTML(etiquetaEstadoEstadistica(estado))}</span>`).join("");
    // El contrato v1 no declara si cargó el histórico: su cero no acredita ausencia.
    const historico = `<div><h4>${textoPortal("txt_historico_de_llamamientos")}</h4><p class="estado-chip neutro" role="status">${textoPortal("txt_no_disponible")}</p></div>`;
    const filas = datos.por_bolsa.map((bolsa) => {
      const href = escaparHTML(rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", bolsa.bolsa_ref));
      return `<tr><td><a class="enlace-tabla" href="${href}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}"><strong>${escaparHTML(bolsa.categoria)}</strong></a><br><small>${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</small></td><td><span class="estado-chip ${bolsa.vigente ? "exito" : "neutro"}">${bolsa.vigente ? traducirPortal("txt_vigente") : traducirPortal("txt_sustituida")}</span></td><td><a class="enlace-tabla" href="${href}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}" aria-label="${textoPortal("txt_aria_abrir_personas_bolsa", { total: numero(bolsa.total), categoria: bolsa.categoria })}">${numero(bolsa.total)}</a></td>${ESTADOS_BOLSA.map((estado) => `<td><a class="estado-chip ${claseEstadoEstadistica(estado)}" href="${escaparHTML(rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", bolsa.bolsa_ref, estado))}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}" data-estado="${escaparHTML(estado)}" aria-label="${textoPortal("txt_aria_abrir_personas_estado", { total: numero(bolsa.por_estado[estado]), estado: etiquetaEstadoEstadistica(estado).toLocaleLowerCase(LOCALIZACION_PORTAL), categoria: bolsa.categoria })}">${numero(bolsa.por_estado[estado])}</a></td>`).join("")}</tr>`;
    }).join("");
    return `${cabecera}<div class="rejilla-kpi" aria-label="${textoPortal("txt_totales_de_bolsa")}">${tarjetas.map(([sigla, valor, etiqueta]) => tarjetaKPI(sigla, valor, etiqueta, sigla === "bolsas" ? {
      atributos: 'data-vista="resumen"',
      aria: traducirEnlacesBolsa("kpi_bolsas_aria", { total: valor }),
      enlace: traducirEnlacesBolsa("kpi_ver_cuadro"),
    } : null)).join("")}</div><section class="panel panel-separado"><div class="cabecera-panel"><h3>${textoPortal("txt_personas_y_llamamientos")}</h3><time datetime="${escaparHTML(datos.generado_en)}">${textoPortal("txt_actualizado_en", { instante: instanteVisible(datos.generado_en) })}</time></div><div class="cuerpo-panel estadisticas-resumen"><div><h4>${textoPortal("txt_situacion_de_personas")}</h4><div class="lista-chips">${resumenEstados}</div></div>${historico}</div></section><section class="panel panel-separado"><div class="cabecera-panel"><h3>${textoPortal("txt_desglose_por_bolsa")}</h3></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>${textoPortal("txt_personas_por_bolsa_y_situacion")}</caption><thead><tr><th scope="col">${textoPortal("txt_bolsa")}</th><th scope="col">${textoPortal("txt_vigencia")}</th><th scope="col">${textoPortal("txt_total")}</th>${ESTADOS_BOLSA.map((estado) => `<th scope="col">${escaparHTML(etiquetaEstadoEstadistica(estado))}</th>`).join("")}</tr></thead><tbody>${filas || '<tr><td colspan="11" class="vacio-controlado">' + textoPortal("txt_la_fuente_no_ha_devuelto_bolsas_para_este_ambito") + '</td></tr>'}</tbody></table></div></section>`;
  }
  function instanteVisible(instante) {
    if (!instante || String(instante).startsWith("0001-01-01")) return traducirPortal("txt_sin_fecha_limite");
    const fecha = new Date(instante);
    if (!Number.isFinite(fecha.getTime())) return traducirPortal("txt_fecha_no_disponible");
    return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, {
      dateStyle: "short", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL,
    }).format(fecha);
  }
  function fechaCivilVisible(valor) {
    const partes = typeof valor === "string" && /^(\d{4})-(\d{2})-(\d{2})$/u.exec(valor);
    if (!partes) return traducirPortal("txt_fecha_no_disponible");
    const [, anioTexto, mesTexto, diaTexto] = partes;
    const anio = Number(anioTexto);
    const mes = Number(mesTexto);
    const dia = Number(diaTexto);
    if (anio < 1) return traducirPortal("txt_fecha_no_disponible");
    const fecha = new Date(0);
    fecha.setUTCFullYear(anio, mes - 1, dia);
    fecha.setUTCHours(0, 0, 0, 0);
    if (fecha.getUTCFullYear() !== anio || fecha.getUTCMonth() !== mes - 1 || fecha.getUTCDate() !== dia) {
      return traducirPortal("txt_fecha_no_disponible");
    }
    return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "short", timeZone: "UTC" }).format(fecha);
  }
  function fechaVisible(valor) {
    return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(valor)
      ? fechaCivilVisible(valor)
      : instanteVisible(valor);
  }
  function fechaAvisoSustitucion(valor) {
    const partes = typeof valor === "string" && /^(\d{4})-(\d{2})-(\d{2})$/u.exec(valor);
    if (partes) {
      const [, anio, mes, dia] = partes;
      return `${dia}/${mes}/${anio}`;
    }
    return fechaVisible(valor);
  }
  function fechaMarcada(valor) {
    const texto = fechaVisible(valor);
    return texto === traducirPortal("txt_fecha_no_disponible")
      ? escaparHTML(texto)
      : `<time datetime="${escaparHTML(valor)}">${escaparHTML(texto)}</time>`;
  }
  const tp = traducirReferencia;
  // Tipo de actuación por catálogo; una clave que el catálogo aún no nombra se
  // presenta legible a partir de la propia clave, nunca como referencia.
  function etiquetaTipoActuacion(clave) {
    const entrada = `actuacion_tipo_${clave}`;
    return tieneTextoReferencia(entrada) ? tp(entrada) : etiquetaClave(clave);
  }
  function celdaFechaLimite(valor) {
    return valor ? `<time datetime="${escaparHTML(valor)}">${escaparHTML(instanteVisible(valor))}</time>` : escaparHTML(tp("sin_fecha_limite"));
  }
  // Las referencias de convocatoria, actuación y recurso son opacas: la fila
  // se identifica por su número de orden y sus datos, y la referencia viaja en
  // un atributo y en el botón «Copiar referencia» para el seguimiento.
  function filasConvocatorias(datos) {
    if (datos.convocatorias.length === 0) {
      return `<tr><td colspan="7" class="vacio-controlado">${escaparHTML(tp("convocatorias_vacio"))}</td></tr>`;
    }
    return datos.convocatorias.map((item, indice) => {
      const categoria = etiquetaClave(item.categoria_clave);
      return `
      <tr data-convocatoria-ref="${escaparHTML(item.convocatoria_ref)}">
        <th scope="row">${numero(indice + 1)}</th>
        <td><strong>${escaparHTML(categoria)}</strong></td>
        <td><span class="estado-chip ${claseEstado(item.estado_clave)}">${escaparHTML(etiquetaClave(item.estado_clave))}</span></td>
        <td>${celdaFechaLimite(item.plazo_cierra_en)}</td>
        <td>${numero(item.numero_solicitudes)}</td><td>${numero(item.numero_pendientes)}</td>
        <td>${referenciaCopiableTraducida(item.convocatoria_ref, escaparHTML, traducirReferencia, tp("convocatoria_copiar_aria", { numero: numero(indice + 1), categoria }))}</td>
      </tr>`;
    }).join("");
  }
  function filasActuaciones(datos) {
    if (datos.actuaciones_pendientes.length === 0) {
      return `<tr><td colspan="7" class="vacio-controlado">${escaparHTML(tp("actuaciones_vacio"))}</td></tr>`;
    }
    return datos.actuaciones_pendientes.map((item, indice) => {
      const tipo = etiquetaTipoActuacion(item.tipo_clave);
      return `
      <tr data-actuacion-ref="${escaparHTML(item.actuacion_ref)}" data-recurso-ref="${escaparHTML(item.recurso_ref)}">
        <th scope="row">${numero(indice + 1)}</th>
        <td><strong>${escaparHTML(tipo)}</strong></td>
        <td><span class="estado-chip ${claseEstado(item.estado_clave)}">${escaparHTML(etiquetaClave(item.estado_clave))}</span></td>
        <td><span class="estado-chip ${claseEstado(item.prioridad_clave)}">${escaparHTML(etiquetaClave(item.prioridad_clave))}</span></td>
        <td>${celdaFechaLimite(item.fecha_limite)}</td>
        <td>${numero(item.numero_elementos)}</td>
        <td>${referenciaCopiableTraducida(item.actuacion_ref, escaparHTML, traducirReferencia, tp("actuacion_copiar_aria", { numero: numero(indice + 1), tipo }))}</td>
      </tr>`;
    }).join("");
  }
  function celdaActor(valor) {
    return actorTraducido(valor, escaparHTML, traducirReferencia);
  }
  function tablaConvocatorias(datos) {
    const columnas = ["col_numero", "convocatoria_col_categoria", "convocatoria_col_estado", "convocatoria_col_cierre",
      "convocatoria_col_solicitudes", "convocatoria_col_pendientes", "col_referencia"];
    return `<section class="panel"><div class="cabecera-panel"><h3>${escaparHTML(tp("convocatorias_titulo"))}</h3><span class="estado-chip info">${escaparHTML(tp("registros", { cantidad: numero(datos.convocatorias.length) }))}</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>${escaparHTML(tp("convocatorias_caption"))}</caption><thead><tr>${columnas.map((clave) => `<th scope="col">${escaparHTML(tp(clave))}</th>`).join("")}</tr></thead><tbody>${filasConvocatorias(datos)}</tbody></table></div></section>`;
  }
  function tablaActuaciones(datos) {
    const columnas = ["col_numero", "actuacion_col_tipo", "actuacion_col_estado", "actuacion_col_prioridad",
      "actuacion_col_fecha_limite", "actuacion_col_elementos", "col_referencia"];
    return `<section class="panel"><div class="cabecera-panel"><h3>${escaparHTML(tp("actuaciones_titulo"))}</h3><span class="estado-chip info">${escaparHTML(tp("registros", { cantidad: numero(datos.actuaciones_pendientes.length) }))}</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>${escaparHTML(tp("actuaciones_caption"))}</caption><thead><tr>${columnas.map((clave) => `<th scope="col">${escaparHTML(tp(clave))}</th>`).join("")}</tr></thead><tbody>${filasActuaciones(datos)}</tbody></table></div></section>`;
  }
  function renderizarCuadroB12() {
    const estadoBolsas = typeof obtenerDatosBolsas === "function" ? obtenerDatosBolsas() : null;
    if (!estadoBolsas) return "";
    if (estadoBolsas.carga === "cargando") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12" aria-busy="true">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">${textoPortal("txt_bolsas_de_trabajo")}</h3>
            <span class="estado-chip neutro">${textoPortal("txt_consultando")}</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="status">
            <p><strong>${textoPortal("txt_cargando_bolsas_de_trabajo")}</strong></p>
          </div>
        </section>`;
    }
    if (estadoBolsas.carga === "error") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">${textoPortal("txt_bolsas_de_trabajo")}</h3>
            <span class="estado-chip peligro">${textoPortal("txt_error_de_carga")}</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>${textoPortal("txt_no_se_pudieron_cargar_las_bolsas_de_trabajo")}</strong></p>
            <p>${escaparHTML(estadoBolsas.error || traducirPortal("txt_se_ha_producido_un_error_al_consultar_las_bolsas"))}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">${textoPortal("txt_reintentar")}</button>
            </div>
          </div>
        </section>`;
    }
    if (estadoBolsas.carga === "denegado") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">${textoPortal("txt_bolsas_de_trabajo")}</h3>
            <span class="estado-chip peligro">${textoPortal("txt_acceso_denegado")}</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>${textoPortal("txt_acceso_denegado_a_la_consulta_de_bolsas")}</strong></p>
            <p>${textoPortal("txt_la_sesion_actual_no_dispone_de_permisos_suficien")}</p>
          </div>
        </section>`;
    }
    const bolsas = estadoBolsas.datos?.bolsas || [];
    if (bolsas.length === 0) {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">${textoPortal("txt_bolsas_de_trabajo")}</h3>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="status">
            <p><strong>${textoPortal("txt_no_hay_bolsas_de_trabajo_activas")}</strong></p>
            <p>${textoPortal("txt_el_servicio_no_ha_devuelto_bolsas_de_trabajo_reg")}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">${textoPortal("txt_reintentar")}</button>
            </div>
          </div>
        </section>`;
    }
    const totalAspirantes = bolsas.reduce((total, bolsa) => total + Number(bolsa.total || 0), 0);
    const totalDisponibles = bolsas.reduce((total, bolsa) => total + Number(bolsa.por_estado?.disponible || 0), 0);
    const totalPersonasEnRenuncia = bolsas.reduce((total, bolsa) => total + Number(bolsa.por_estado?.renuncia || 0), 0);
    const filas = bolsas.map((b) => `
      <tr data-bolsa-ref="${escaparHTML(b.bolsa_ref)}">
        <td><a class="enlace-tabla" href="${escaparHTML(rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", b.bolsa_ref))}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(b.bolsa_ref)}" aria-label="${textoPortal("txt_aria_abrir_candidatos_bolsa", { categoria: b.categoria })}"><strong>${escaparHTML(b.categoria)}</strong></a></td>
        <td><span class="estado-chip neutro">${escaparHTML(etiquetaClave(b.tipo_lista))}</span></td>
        <td>${escaparHTML(finVigenciaBolsaPortal(b.vigente_hasta))}</td>
        <td><a class="enlace-tabla" href="${escaparHTML(rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", b.bolsa_ref))}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(b.bolsa_ref)}" aria-label="${textoPortal("txt_aria_ver_candidatos_bolsa", { total: numero(b.total), categoria: b.categoria })}"><strong>${numero(b.total)}</strong></a></td>
        <td><span class="estado-chip info">${numero(b.llamamientos_en_curso)}</span></td>
        <td>${controlEstadoBolsa(b, "disponible", "exito")}</td>
        <td>${controlEstadoBolsa(b, "trabajando")}</td>
        <td>${controlEstadoBolsa(b, "no_disponible", "peligro")}</td>
        <td>${controlEstadoBolsa(b, "excluido", "peligro")}</td>
        <td>${controlEstadoBolsa(b, "renuncia")}</td>
        <td>${controlEstadoBolsa(b, "pendiente_incorporacion", "info")}</td>
        <td>${controlEstadoBolsa(b, "en_revision", "advertencia")}</td>
      </tr>
    `).join("");
    return `
      <section class="panel" aria-labelledby="titulo-cuadro-b12">
        <div class="cabecera-panel">
          <h3 id="titulo-cuadro-b12">${textoPortal("txt_bolsas_de_trabajo_activas")}</h3>
        </div>
        <div class="rejilla-kpi cuadro-b12-kpi" aria-label="${textoPortal("txt_resumen_de_las_bolsas_de_trabajo")}">
          <a class="tarjeta-kpi" href="${escaparHTML(rutaResumenBolsasCompartible(globalThis.location?.search ?? ""))}" data-vista="resumen"><span class="icono-kpi" aria-hidden="true">${icono("bolsas")}</span><div><span class="etiqueta-kpi">${textoPortal("txt_bolsas_visibles")}</span><strong class="valor-kpi">${numero(bolsas.length)}</strong></div></a>
          <article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">${icono("personas")}</span><div><span class="etiqueta-kpi">${textoPortal("txt_aspirantes")}</span><strong class="valor-kpi">${numero(totalAspirantes)}</strong></div></article>
          <article class="tarjeta-kpi kpi--exito"><span class="icono-kpi" aria-hidden="true">${icono("persona_ok")}</span><div><span class="etiqueta-kpi">${textoPortal("txt_disponibles")}</span><strong class="valor-kpi">${numero(totalDisponibles)}</strong></div></article>
          <article class="tarjeta-kpi kpi--advertencia"><span class="icono-kpi" aria-hidden="true">${icono("persona_baja")}</span><div><span class="etiqueta-kpi">${textoPortal("txt_personas_en_renuncia")}</span><strong class="valor-kpi">${numero(totalPersonasEnRenuncia)}</strong></div></article>
        </div>
        <div class="tabla-contenedor">
          <table class="tabla-datos">
            <caption>${textoPortal("txt_bolsas_de_trabajo_y_distribucion_de_aspirantes_p")}</caption>
            <thead>
              <tr>
                <th scope="col">${textoPortal("txt_bolsa_categoria")}</th>
                <th scope="col">${textoPortal("txt_tipo_de_lista")}</th>
                <th scope="col">${textoPortal("inicio_rrhh_col_vigente_hasta")}</th>
                <th scope="col">${textoPortal("txt_total")}</th>
                <th scope="col">${textoPortal("txt_llamamientos_en_curso")}</th>
                <th scope="col">${textoPortal("txt_disponibles")}</th>
                <th scope="col">${textoPortal("txt_ocupados_trabajando")}</th>
                <th scope="col">${textoPortal("txt_no_disp")}</th>
                <th scope="col">${textoPortal("txt_excluidos")}</th>
                <th scope="col">${textoPortal("txt_renuncia")}</th>
                <th scope="col">${textoPortal("txt_pend_incorporacion")}</th>
                <th scope="col">${textoPortal("txt_b8_en_revision")}</th>
              </tr>
            </thead>
            <tbody>
              ${filas}
            </tbody>
          </table>
        </div>
      </section>`;
  }
  function renderizarResumen(datos) {
    const i = datos.indicadores;
    const indicadoresConectados = [
      ["bolsas", i.bolsas_activas, traducirPortal("txt_bolsas_activas")],
      ["llamamiento", i.llamamientos_pendientes, traducirPortal("txt_llamamientos_pendientes")],
      ["en_curso", i.llamamientos_en_curso, traducirPortal("txt_llamamientos_en_curso")],
      ["documento", i.documentos_pendientes_firma, traducirPortal("txt_documentos_pendientes_de_firma")],
      ["alerta", i.incidencias_abiertas, traducirPortal("txt_incidencias_abiertas")],
    ];
    return `
      ${encabezadoVista("", traducirPortal("txt_bolsas_de_trabajo"), "", `<a class="boton-secundario" href="${RUTA_PANTALLA_REGLAS}" target="_blank" rel="noopener">${textoPortal("txt_reglas_vigentes")}</a><button type="button" class="boton-secundario" data-accion="imprimir">${textoPortal("txt_imprimir_resumen")}</button>`)}
      <div class="rejilla-kpi" aria-label="${textoPortal("txt_indicadores_operativos_de_bolsa")}">
        ${indicadoresConectados.map(([sigla, valor, etiqueta]) => tarjetaKPI(sigla, numero(valor), etiqueta)).join("")}
      </div>
      <div class="rejilla-cuadro-mando" aria-label="${textoPortal("txt_resumen_operativo")}">
        <div class="columna-cuadro">
          ${renderizarCuadroB12()}
          ${tablaConvocatorias(datos)}
        </div>
        <aside class="columna-cuadro" aria-label="${textoPortal("txt_actuaciones_y_prueba_de_lectura")}">
          ${tablaActuaciones(datos)}
          <section class="panel"><div class="cabecera-panel"><h3>${textoPortal("txt_prueba_de_lectura")}</h3><span class="estado-chip exito">${textoPortal("txt_lectura_auditada")}</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>${textoPortal("txt_ambito")}</dt><dd>${escaparHTML(etiquetaClave(datos.selector.clase))}</dd></div><div class="fila-resumen"><dt>${textoPortal("txt_revision_de_fuente")}</dt><dd>${referenciaCopiableTraducida(datos.origen.revision, escaparHTML, traducirReferencia, tp("revision_fuente_copiar_aria"))}</dd></div><div class="fila-resumen"><dt>${textoPortal("txt_actualizada")}</dt><dd><time datetime="${escaparHTML(datos.origen.actualizada_en)}">${escaparHTML(instanteVisible(datos.origen.actualizada_en))}</time></dd></div><div class="fila-resumen"><dt>${textoPortal("txt_confirmada")}</dt><dd><time datetime="${escaparHTML(datos.prueba_lectura.confirmada_en)}">${escaparHTML(instanteVisible(datos.prueba_lectura.confirmada_en))}</time></dd></div></dl></div></section>
        </aside>
      </div>`;
  }
  function renderizarConvocatorias(datos) {
    const i = datos.indicadores;
    return `
      ${encabezadoVista("", traducirPortal("txt_convocatorias"), "", '<button type="button" class="boton-secundario" data-vista="resumen">' + textoPortal("txt_volver_al_cuadro_de_mando") + '</button>')}
      <div class="rejilla-kpi">${tarjetaKPI("documento", numero(i.convocatorias_borrador), traducirPortal("txt_borrador"))}${tarjetaKPI("en_curso", numero(i.convocatorias_revision), traducirPortal("txt_en_revision"))}${tarjetaKPI("contrato", numero(i.convocatorias_pendientes_firma), traducirPortal("txt_pendientes_de_firma"))}${tarjetaKPI("correcto", numero(i.convocatorias_publicadas), traducirPortal("txt_publicadas"))}</div>
      ${tablaConvocatorias(datos)}`;
  }
  function renderizarNuevoLlamamiento(bolsa, candidatos, flujo, fuente = {}) {
    const paso = Math.max(1, Math.min(4, Number(flujo.paso) || 1));
    const plazoAnterior = String(flujo.configuracion?.plazo || "");
    const reglaPlazo = flujo.reglaPlazo || null;
    const plazoEditable = (/\bpendiente\b/i.test(plazoAnterior) ? "" : plazoAnterior) || (flujo.configuracion ? "" : reglaPlazo?.texto || "");
    const t = (clave, variables) => escaparHTML(traducirPortal(clave, variables));
    const etiquetaEstadoB7 = (estado) => estado === "disponible" || estado === "disponible_desde"
      ? t(estado === "disponible" ? "panel_b7_estado_disponible" : "panel_b7_estado_disponible_desde") : escaparHTML(etiquetaEstadoBolsa(estado));
    const nombres = ["panel_b7_paso_bolsa", "panel_b7_paso_candidatos", "panel_b7_paso_configurar", "panel_b7_paso_revisar"];
    const rail = `<nav class="pasos" aria-label="${t("panel_b7_pasos_aria")}">${nombres.map((nombre,i)=>`<span class="paso ${i+1<paso?"completado":""}"${i+1===paso?' aria-current="step" tabindex="-1"':""}><span class="paso-numero">${i+1<paso?"✓":i+1}</span><span>${t(nombre)}</span></span>`).join("")}</nav>`;
    const resumen = `<aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>${t("panel_b7_resumen")}</h3></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>${t("panel_b7_categoria")}</dt><dd>${escaparHTML(bolsa.categoria)}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_tipo_lista")}</dt><dd>${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_vigencia")}</dt><dd>${escaparHTML(fechaVisible(bolsa.vigente_desde))}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_personas")}</dt><dd>${numero(bolsa.total)}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_seleccionadas")}</dt><dd>${numero(flujo.participaciones?.length||0)}</dd></div></dl></div></section></aside>`;
    let contenido="";
    if(paso===1) contenido=`<form class="panel" data-bolsa-form="b7-paso1"><div class="cabecera-panel"><h3>1. ${t("panel_b7_paso_bolsa")}</h3><span class="estado-chip exito">${t("panel_b7_bolsa_constituida")}</span></div><div class="cuerpo-panel"><p><strong>${escaparHTML(bolsa.categoria)}</strong></p><p>${t("panel_b7_bolsa_datos", { total: numero(bolsa.total), disponibles: numero(bolsa.por_estado?.disponible||0) })}</p><button class="boton-primario" type="submit">${t("panel_b7_seleccionar_bolsa")}</button></div></form>`;
    if (paso === 2) {
      const estados = new Set(flujo.estados || ["disponible"]);
      const visibles = candidatos.filter((candidato) => estados.has(candidato.estado_clave) && Number.isSafeInteger(candidato.orden));
      const pagina = Math.max(0, Math.min(Number(flujo.pagina) || 0, Math.max(0, Math.ceil(visibles.length / 6) - 1)));
      const inicio = pagina * 6;
      const seleccionadas = flujo.participaciones?.length || 0, botonPaginaB7 = (texto, accion) => `<button type="button" class="boton-secundario" ${accion || "disabled"}>${texto}</button>`; // una sola paginación: al agotar las filas cargadas pide a la bolsa el tramo contiguo
      // Quien ya presta servicios con el catálogo en «excluir» se ve, pero no se puede marcar.
      const filas = visibles.slice(inicio, inicio + 6).map((candidato) => { const seleccionable = !["en_revision", "trabajando", "excluido"].includes(candidato.estado_clave) && seleccionableEnLlamamiento(candidato); return `<tr><td><input type="checkbox" name="participacion" value="${escaparHTML(candidato.participacion_ref)}" ${seleccionable && flujo.participaciones?.includes(candidato.participacion_ref) ? "checked" : ""}${seleccionable ? "" : " disabled"} aria-label="${seleccionable ? t("panel_b7_seleccionar_orden_aria", { orden: numero(candidato.orden) }) : candidato.estado_clave === "en_revision" ? t("txt_b8_en_revision") : ["trabajando", "excluido"].includes(candidato.estado_clave) ? escaparHTML(etiquetaEstadoBolsa(candidato.estado_clave)) : escaparHTML(traducirMarcasBolsa("no_seleccionable_aria"))}"></td><td>${numero(candidato.orden)}</td><td>${escaparHTML(candidato.nombre_visible)}</td><td><span class="estado-chip ${claseEstado(candidato.estado_clave)}">${etiquetaEstadoB7(candidato.estado_clave)}</span>${renderizarChipsMarcas(candidato, escaparHTML)}</td></tr>`; }).join("");
      contenido = `<form class="panel" data-bolsa-form="b7-paso2" aria-busy="${flujo.consultando ? "true" : "false"}"><div class="cabecera-panel"><h3>2. ${t("panel_b7_paso_candidatos")}</h3><span class="estado-chip info">${t("panel_b7_turno_pagina", { cantidad: numero(visibles.length) })}</span></div><div class="cuerpo-panel"><fieldset><legend>${t("panel_b7_estados_incluir")}</legend>${["disponible", "disponible_desde"].map((estado) => `<label><input type="checkbox" name="estado" value="${estado}" ${estados.has(estado) ? "checked" : ""}> ${etiquetaEstadoB7(estado)}</label>`).join(" ")}</fieldset><p>${t("panel_b7_orden_obligatorio")}</p><button type="button" class="boton-secundario" data-bolsa-accion="b7-seleccionar-todas" ${flujo.consultando ? "disabled" : ""}>${t("panel_b7_seleccionar_todas")}</button><p role="status" tabindex="-1" data-b7-seleccion-status><strong>${t("panel_b7_seleccionadas_estado", { cantidad: numero(seleccionadas) })}</strong>${flujo.consultando ? t("panel_b7_consultando_paginas") : flujo.totalElegibles !== null && flujo.totalElegibles !== undefined ? t("panel_b7_elegibles_estado", { cantidad: numero(flujo.totalElegibles) }) : t("panel_b7_punto")}</p>${flujo.error ? `<p role="alert" tabindex="-1" data-b7-seleccion-error class="mensaje-error">${escaparHTML(flujo.error)}</p>` : ""}</div><div class="tabla-contenedor"><table class="tabla-datos"><caption>${t("panel_b7_tabla_orden")}</caption><thead><tr><th scope="col">${t("panel_b7_col_seleccion")}</th><th scope="col">${t("panel_b7_col_orden")}</th><th scope="col">${t("panel_b7_col_candidato")}</th><th scope="col">${t("panel_b7_col_estado")}</th></tr></thead><tbody>${filas || `<tr><td colspan="4">${t("panel_b7_sin_turno")}</td></tr>`}</tbody></table></div><div class="cuerpo-panel"><span>${t("panel_b7_pagina_estado", { inicio: numero(visibles.length ? inicio + 1 : 0), fin: numero(Math.min(inicio + 6, visibles.length)), total: numero(visibles.length) })}</span> ${botonPaginaB7(t("panel_b7_anterior"), pagina > 0 ? `data-bolsa-accion="b7-pagina" data-pagina="${pagina - 1}"` : flujo.cursoresPagina?.length > 1 && !flujo.consultando ? 'data-bolsa-accion="b7-fuente-anterior"' : "")} ${botonPaginaB7(t("panel_b7_siguiente"), inicio + 6 < visibles.length ? `data-bolsa-accion="b7-pagina" data-pagina="${pagina + 1}"` : fuente.hay_mas && !flujo.consultando ? 'data-bolsa-accion="b7-fuente-siguiente"' : "")} <button type="button" class="boton-secundario" data-bolsa-accion="b7-limpiar-seleccion" ${!seleccionadas || flujo.consultando ? "disabled" : ""}>${t("panel_b7_borrar_seleccion")}</button> <button class="boton-primario" type="submit" ${flujo.consultando ? "disabled" : ""}>${t("panel_b7_configurar")}</button></div></form>`;
    }
    if(paso===3){
      const campoB7=(etiqueta,control,ancho=false)=>`<label class="campo${ancho?" campo-ancho":""}"><span>${/\srequired[\s>]/.test(control)?t("panel_b7_campo_obligatorio",{campo:"{campo}"}).replace("{campo}",etiqueta):etiqueta}</span>${control}</label>`;
      const modalidades=[["Sustitución","panel_b7_modalidad_sustitucion"],["Vacante","panel_b7_modalidad_vacante"],["Programa temporal","panel_b7_modalidad_programa"],["Acumulación de tareas","panel_b7_modalidad_acumulacion"]];
      const c=flujo.configuracion||{};
      // Con origen en una petición, referencia, centro y fecha vienen de ella y no se editan (readonly sí se envía).
      const origen=flujo.origen||null;
      const fijo=(campo)=>origen?.[campo]?" readonly":"";
      const valorOrigen=(campo)=>escaparHTML(origen?.[campo]||c[campo]||"");
      const {telefono}=canalesAviso(fuente);
      const canales=`<fieldset class="campo campo-ancho"><legend>${t("panel_b7_canal")}</legend><label><input type="checkbox" checked disabled> ${t("panel_b7_canal_correo_emitir")}</label>${telefono?`<label><input type="checkbox" name="seguimiento_telefono" value="1"${flujo.canales?.telefono===false?"":" checked"}> ${t("panel_b7_canal_telefono")}</label>`:""}</fieldset>`;
      contenido=`<form class="panel" data-bolsa-form="b7-paso3"><div class="cabecera-panel"><h3>3. ${t("panel_b7_paso_configurar")}</h3></div><div class="cuerpo-panel rejilla-formulario rejilla-llamamiento">${campoB7(t("panel_b7_referencia"),`<input name="referencia" required minlength="2" maxlength="160" value="${valorOrigen("referencia")}"${fijo("referencia")}>`)}${campoB7(t("panel_b7_categoria"),`<input name="categoria" required value="${escaparHTML(c.categoria||bolsa.categoria)}">`)}${campoB7(t("panel_b7_centro"),`<input name="centro" required minlength="2" maxlength="200" value="${valorOrigen("centro")}"${fijo("centro")}>`)}${campoB7(t("panel_b7_modalidad"),`<select name="modalidad" required>${modalidades.map(([modalidad,clave])=>`<option value="${escaparHTML(modalidad)}" ${c.modalidad===modalidad?"selected":""}>${t(clave)}</option>`).join("")}</select>`)}${campoB7(t("panel_b7_fecha_inicio"),`<input type="date" name="fecha_inicio" required value="${valorOrigen("fecha_inicio")}"${fijo("fecha_inicio")}>`)}${canales}${campoB7(t("panel_b7_plazo_indicado"),`<input name="plazo" required minlength="2" maxlength="160" value="${escaparHTML(plazoEditable)}">`,true)}${campoB7(t("panel_b7_descripcion_campo"),`<textarea name="descripcion" required minlength="2" maxlength="1000">${escaparHTML(c.descripcion||"")}</textarea>`,true)}<input type="hidden" name="plantilla_version" value="${escaparHTML(flujo.plantilla_version||"bolsa-llamamiento-v1")}">${campoB7(t("panel_b7_asunto"),`<input name="asunto" required value="${escaparHTML(c.asunto||traducirPortal("panel_b7_asunto_defecto", { categoria: bolsa.categoria }))}">`,true)}${campoB7(t("panel_b7_texto_correo"),`<textarea name="cuerpo" required maxlength="4000">${escaparHTML(flujo.cuerpoBorrador||traducirPortal("panel_b7_cuerpo_defecto"))}</textarea>`,true)}${renderizarMarcadoresCorreo(flujo)}${flujo.error?`<p role="alert" class="mensaje-error campo-ancho">${escaparHTML(flujo.error)}</p>`:""}<div class="campo-ancho acciones-formulario"><button type="submit" class="boton-primario">${t("panel_b7_revisar")}</button></div></div></form>`;
    }
    if(paso===4){const c=flujo.configuracion||{};const revisionRequerida=flujo.revision_obligatoria===true;contenido=`<section class="panel"><div class="cabecera-panel"><h3>4. ${t("panel_b7_paso_revisar")}</h3><span class="estado-chip advertencia">${t("panel_b7_confirmacion_pendiente")}</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>${t("panel_b7_necesidad")}</dt><dd>${escaparHTML(c.referencia)}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_centro_modalidad")}</dt><dd>${escaparHTML(c.centro)} · ${escaparHTML(c.modalidad)}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_candidatos")}</dt><dd>${t("panel_b7_seleccionados", { cantidad: numero(flujo.participaciones?.length||0) })}${flujo.totalElegibles > (flujo.participaciones?.length||0) ? `${t("panel_b7_de_elegibles", { cantidad: numero(flujo.totalElegibles) })}` : ""}${t("panel_b7_orden_preferencia")}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_canal")}</dt><dd>${t(flujo.canales?.telefono?"panel_b7_canal_detalle_telefono":"panel_b7_canal_detalle")}</dd></div><div class="fila-resumen"><dt>${t("panel_b7_plazo")}</dt><dd>${escaparHTML(c.plazo)}</dd></div></dl>${renderizarVistaPreviaCorreo(flujo, candidatos)}<form class="confirmacion-llamamiento" data-bolsa-form="b7-paso4" data-cantidad="${numero(flujo.participaciones?.length||0)}"><label class="casilla-confirmacion"><input type="checkbox" name="confirmacion" required> <span>${t("panel_b7_confirmar", { cantidad: numero(flujo.participaciones?.length||0) })}</span></label><div class="acciones-formulario"><button type="submit" class="boton-primario" ${flujo.enviando||flujo.recibo||revisionRequerida?"disabled":""}${revisionRequerida?` aria-describedby="b7-motivo-revision" aria-label="${t("panel_b7_revision_requerida")}"`:""}>${flujo.enviando?t("panel_b7_enviando"):flujo.recibo?t("panel_b7_emitido"):revisionRequerida?t("panel_b7_revision_pendiente"):t("panel_b7_emitir")}</button></div></form>${revisionRequerida?`<p id="b7-motivo-revision" class="nota-pendiente" role="status">${t("panel_b7_revision_requerida")}</p>`:""}${flujo.error?`<p role="alert" tabindex="-1" data-b7-emision-error class="mensaje-error">${escaparHTML(flujo.error)}</p>`:""}${flujo.error_422?`<div><button type="button" class="boton-secundario" data-bolsa-accion="b7-revisar-configuracion">${t("panel_b7_revisar_configuracion")}</button> <button type="button" class="boton-secundario" data-bolsa-accion="b7-volver-seleccion">${t("panel_b7_actualizar_seleccion")}</button></div>`:""}${flujo.recibo?`<section class="mensaje-exito" tabindex="-1" data-b7-recibo><p><strong>${t("panel_b7_llamamiento_emitido")}</strong></p><p>${t("panel_b7_estado_emitido")}</p><p>${justificanteTraducido(flujo.recibo, escaparHTML, (clave) => traducirPortal(`panel_${clave}`))}</p><div class="acciones-vista">${flujo.canales?.telefono&&flujo.llamamiento_ref?enlaceSeguimiento({bolsaRef:bolsa.bolsa_ref,llamamientoRef:flujo.llamamiento_ref,escaparHTML,clase:"boton-primario",contenido:t("panel_b7_empezar_llamadas")}):""}<button type="button" class="boton-secundario" data-bolsa-accion="ver-historico-b7">${t("panel_b7_abrir_historico")}</button></div></section>${renderizarAvisosContactoEmision({ avisos: flujo.avisos_contacto, candidatos, escaparHTML })}`:""}</div></section>`}
    return `${encabezadoVista("",traducirPortal("panel_b7_titulo"),"",`<button type="button" class="boton-secundario" data-bolsa-accion="cancelar-b7" ${flujo.enviando?"disabled":""}>${flujo.enviando?t("panel_b7_envio_curso"):t("panel_b7_cancelar")}</button>`)}${rail}<div class="distribucion-llamamiento"><div>${contenido}</div>${resumen}</div>`;
  }
  function renderizarCandidatosBolsa() {
    const estadoCandidatos = typeof obtenerDatosCandidatosBolsa === "function"
      ? obtenerDatosCandidatosBolsa()
      : null;
    const filtrosActuales = typeof obtenerEstadoCandidatos === "function"
      ? obtenerEstadoCandidatos()
      : { estado: "", texto: "" };
    if (filtrosActuales.seguimiento?.llamamiento_ref && !filtrosActuales.nuevo_llamamiento) {
      return renderizarSeguimientoLlamamiento({ estadoCandidatos, seguimiento: filtrosActuales.seguimiento,
        modalFicha: typeof obtenerModalFicha === "function" ? obtenerModalFicha() : null,
        encabezadoVista, escaparHTML, numero, instanteVisible, renderizarFicha: renderizarModalFicha });
    }
    const pestana = filtrosActuales.pestana === "historico" ? "historico" : "candidatos";
    const accionesEncabezado = '<button type="button" class="boton-secundario" data-vista="resumen">' + textoPortal("txt_volver_al_cuadro") + '</button>';
    if (!estadoCandidatos) {
      return `
        ${encabezadoVista("", traducirPortal("txt_candidatos_de_la_bolsa"), "", accionesEncabezado)}
        <section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>${escaparHTML(traducirPortal("panel_fuente_no_configurada_titulo"))}</strong></p>
            <p>${escaparHTML(traducirPortal("panel_fuente_no_configurada"))}</p>
          </div>
        </section>`;
    }
    if (estadoCandidatos.carga === "cargando") {
      return `
        ${encabezadoVista("", traducirPortal("txt_candidatos_de_la_bolsa"), "", accionesEncabezado)}
        <section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
          <div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true">
            <p><strong>${textoPortal("txt_cargando_lista_de_candidatos")}</strong></p>
          </div>
        </section>`;
    }
    if (estadoCandidatos.carga === "error") {
      return `
        ${encabezadoVista("", traducirPortal("txt_candidatos_de_la_bolsa"), "", accionesEncabezado)}
        <section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>${textoPortal("txt_error_al_consultar_candidatos")}</strong></p>
            <p>${escaparHTML(estadoCandidatos.error || traducirPortal("txt_no_se_pudo_cargar_la_relacion_de_aspirantes"))}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-candidatos">${textoPortal("txt_reintentar")}</button>
              <button type="button" class="boton-secundario" data-vista="resumen">${textoPortal("txt_volver_al_cuadro")}</button>
            </div>
          </div>
        </section>`;
    }
    if (estadoCandidatos.carga === "denegado") {
      return `
        ${encabezadoVista("", traducirPortal("txt_candidatos_de_la_bolsa"), "", accionesEncabezado)}
        <section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>${textoPortal("txt_acceso_denegado")}</strong></p>
            <p>${textoPortal("txt_la_sesion_no_dispone_de_permisos_para_consultar")}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-vista="resumen">${textoPortal("txt_volver_al_cuadro")}</button>
            </div>
          </div>
        </section>`;
    }
    const bolsa = estadoCandidatos.datos?.bolsa;
    const candidatos = estadoCandidatos.datos?.candidatos || [];
    const contactos = estadoCandidatos.datos?.contactos || [];
    const modalFicha = typeof obtenerModalFicha === "function" ? obtenerModalFicha() : null;
    const reciboFichaFuera = modalFicha?.operacionesB8?.recibo && !candidatos.some((item) => item.participacion_ref === modalFicha.candidato?.participacion_ref)
      ? `<p class="mensaje-exito" role="status">${textoPortal("txt_operacion_registrada")} ${justificanteTraducido(modalFicha.operacionesB8.recibo, escaparHTML, (clave) => traducirPortal(`panel_${clave}`))}. ${textoPortal("txt_participacion_fuera_de_filtro")}</p>` : "";
    const hayMas = estadoCandidatos.datos?.hay_mas === true;
    const cursorSiguiente = estadoCandidatos.datos?.cursor_siguiente || "";
    const turno = estadoCandidatos.datos?.turno;
    if (filtrosActuales.nuevo_llamamiento) return renderizarNuevoLlamamiento(bolsa, candidatos, filtrosActuales.nuevo_llamamiento, estadoCandidatos.datos);
    const tituloBolsa = bolsa ? traducirPortal("txt_candidatos_de_categoria", { categoria: bolsa.categoria }) : traducirPortal("txt_candidatos_de_la_bolsa");
    const opcionesEstado = [
      ["", traducirPortal("txt_todos_los_estados")],
      ["disponible", traducirPortal("txt_disponible")],
      ["no_disponible", traducirPortal("txt_no_disponible")],
      ["trabajando", traducirPortal("txt_trabajando")],
      ["pendiente_incorporacion", traducirPortal("txt_pendiente_de_incorporacion")],
      ["renuncia", traducirPortal("txt_renuncia")],
      ["excluido", traducirPortal("txt_excluido")],
      ["disponible_desde", traducirPortal("txt_disponible_desde_fecha")],
      ["en_revision", traducirPortal("txt_b8_en_revision")],
    ].map(([valor, etiqueta]) => `
      <option value="${escaparHTML(valor)}"${valor === filtrosActuales.estado ? " selected" : ""}>${escaparHTML(etiqueta)}</option>
    `).join("");
    const contadoresEstado = Object.entries(bolsa?.por_estado || {}).map(([estado, total]) => {
      const tono = estado === "disponible" ? "kpi--exito" : ["renuncia", "no_disponible", "en_revision"].includes(estado) ? "kpi--advertencia" : estado === "excluido" ? "kpi--peligro" : "";
      const rotulo = ({ no_disponible: traducirPortal("txt_no_disp"), pendiente_incorporacion: traducirPortal("txt_pend_incorp"), disponible_desde: traducirPortal("txt_desde_fecha") })[estado] || etiquetaEstadoBolsa(estado);
      return `<button type="button" class="tarjeta-kpi kpi-filtro ${tono}" data-bolsa-accion="filtrar-estado" data-estado="${escaparHTML(estado)}" aria-label="${textoPortal("txt_aria_filtrar_situacion", { total: numero(total), situacion: etiquetaEstadoBolsa(estado) })}" aria-pressed="${filtrosActuales.estado === estado}"><span class="icono-kpi" aria-hidden="true">${estado === "disponible" ? "✓" : estado === "excluido" ? "×" : "•"}</span><span><span class="etiqueta-kpi">${escaparHTML(rotulo)}</span><strong class="valor-kpi">${numero(total)}</strong></span></button>`;
    }).join("");
    const formularioFiltros = `
      <form class="barra-filtros-bolsa barra-filtros-estadisticas" data-bolsa-form="filtros" role="search" aria-label="${textoPortal("txt_filtros_de_candidatos")}">
        <div class="campo-filtro">
          <label for="filtro-bolsa-estado">${textoPortal("txt_situacion")}</label>
          <select id="filtro-bolsa-estado" name="estado">${opcionesEstado}</select>
        </div>
        <div class="campo-filtro">
          <label for="filtro-bolsa-texto">${textoPortal("txt_buscar")}</label>
          <input type="search" id="filtro-bolsa-texto" name="texto" value="${escaparHTML(filtrosActuales.texto || "")}" placeholder="${textoPortal("txt_nombre_o_documento")}">
        </div>
        <div class="acciones-filtro">
          <button type="submit" class="boton-primario">${textoPortal("txt_filtrar")}</button>
          <button type="button" class="boton-secundario" data-bolsa-accion="limpiar-filtros">${textoPortal("txt_limpiar")}</button>
        </div>
      </form>`;
    let cuerpoTabla = "";
    const telefonoActivo = Boolean(canalesAviso(estadoCandidatos.datos).telefono);
    if (candidatos.length === 0) {
      cuerpoTabla = `<tr><td colspan="7" class="vacio-controlado">${textoPortal("txt_no_se_han_encontrado_aspirantes_que_coincidan_co")}</td></tr>`;
    } else {
      cuerpoTabla = candidatos.map((c) => {
        let detalleLlamamiento = '<small class="texto-atenuado">' + textoPortal("txt_sin_llamamientos") + '</small>';
        if (c.ultimo_llamamiento) {
          const l = c.ultimo_llamamiento;
          detalleLlamamiento = `<span>${escaparHTML(etiquetaClave(l.canal))} · ${escaparHTML(etiquetaClave(l.resultado))}<br><small><time datetime="${escaparHTML(l.comunicado_en)}">${escaparHTML(instanteVisible(l.comunicado_en))}</time></small></span>`;
          if (telefonoActivo && bolsa?.bolsa_ref) detalleLlamamiento += `<br>${enlaceSeguimiento({ bolsaRef: bolsa.bolsa_ref, llamamientoRef: l.llamamiento_ref, escaparHTML, clase: "boton-secundario",
            contenido: textoPortal("panel_seg_boton"), extra: ` aria-label="${textoPortal("panel_seg_boton_aria", { persona: c.nombre_visible })}"` })}`;
        }
        const fichaAbierta = modalFicha?.abierto === true
          && modalFicha.candidato?.participacion_ref === c.participacion_ref;
        const fichaId = `ficha-participacion-${c.participacion_ref}`;
        return `
          <tr class="fila-candidato" data-participacion-ref="${escaparHTML(c.participacion_ref)}" data-estado="${escaparHTML(c.estado_clave)}">
            <td><strong>${c.orden === null ? "—" : `#${numero(c.orden)}`}</strong>${c.razon_orden !== "orden_acta" ? `<br><small>${escaparHTML(c.razon_orden === "restriccion_cese"
              ? traducirPortal("bolsa_razon_restriccion_cese", { fecha: instanteVisible(c.disponible_desde) })
              : c.razon_orden === "retorno_tras_cese" ? traducirPortal("bolsa_razon_retorno_tras_cese")
                : c.razon_orden === "reposicion_tras_contrato" ? traducirPortal("txt_reposicion_tras_contrato")
                  : c.razon_orden === "pausa" ? traducirPortal("txt_pausa") : etiquetaClave(c.razon_orden))}</small>` : ""}</td>
            <td><button type="button" class="enlace-tabla" data-bolsa-accion="abrir-ficha" data-bolsa-control-principal="true" data-participacion-ref="${escaparHTML(c.participacion_ref)}" aria-expanded="${fichaAbierta}" aria-controls="${escaparHTML(fichaId)}" aria-label="${textoPortal("txt_aria_abrir_ficha_de", { persona: c.nombre_visible })}"><strong>${escaparHTML(c.nombre_visible)}</strong></button></td>
            <td><code>${escaparHTML(c.documento_enmascarado)}</code></td>
            <td><span class="estado-chip ${claseEstado(c.estado_clave)}">${escaparHTML(etiquetaEstadoBolsa(c.estado_clave))}</span>${renderizarChipsMarcas(c, escaparHTML)}</td>
            <td><small>${escaparHTML(instanteVisible(c.estado_desde))}</small></td>
            <td><small>${c.disponible_desde ? escaparHTML(instanteVisible(c.disponible_desde)) : "—"}</small></td>
            <td>${detalleLlamamiento}</td>
          </tr>
          ${fichaAbierta ? renderizarModalFicha(modalFicha, fichaId) : ""}`;
      }).join("");
    }
    const paginacion = hayMas && cursorSiguiente
      ? `<div class="paginacion-bolsa">
           <button type="button" class="boton-secundario" data-bolsa-accion="pagina-siguiente" data-cursor="${escaparHTML(cursorSiguiente)}">${textoPortal("txt_cargar_siguientes_aspirantes")}</button>
         </div>`
      : "";
    const vigenciaBolsa = bolsa
      ? (bolsa.vigente_hasta
        ? `${fechaVisible(bolsa.vigente_desde)} — ${fechaVisible(bolsa.vigente_hasta)}`
        : traducirPortal("txt_vigencia_abierta", { desde: fechaVisible(bolsa.vigente_desde) }))
      : traducirPortal("txt_no_disponible");
    const accionesBolsa = `<div class="cuerpo-panel acciones-vista">
            <button type="button" class="boton-secundario boton-ancho" data-bolsa-accion="cambiar-pestana" data-pestana="historico">${textoPortal("txt_consultar_historial_de_contactos")}</button>
            <button type="button" class="boton-primario boton-ancho" data-bolsa-accion="iniciar-b7">${textoPortal("txt_nuevo_llamamiento")}</button>
            <a class="boton-secundario boton-ancho" href="${RUTA_PETICIONES_PERSONAL_TEMPORAL}">${textoPortal("txt_registrar_resultado_en_peticiones")}</a>
          </div>`;
    const bolsas = typeof obtenerDatosBolsas === "function"
      ? obtenerDatosBolsas()?.datos?.bolsas || []
      : [];
    const bolsaVigente = bolsa?.vigente_hasta
      ? bolsas.find((item) => item.bolsa_ref !== bolsa.bolsa_ref
        && item.categoria_clave === bolsa.categoria_clave && !item.vigente_hasta)
      : null;
    const avisoSustitucion = bolsaVigente ? `
      <section class="nota-pendiente" role="note">
        <p>${escaparHTML(traducirBolsaInterna("bolsa_sustituida_aviso", { fecha: fechaAvisoSustitucion(bolsaVigente.vigente_desde) }))}</p>
        <button type="button" class="boton-secundario" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsaVigente.bolsa_ref)}">${escaparHTML(traducirBolsaInterna("bolsa_abrir_vigente"))}</button>
       </section>` : "";
    const resumenBolsa = bolsa ? `
      <aside class="resumen-lateral" aria-label="${textoPortal("txt_resumen_de_la_bolsa_seleccionada")}">
        <section class="panel">
          <div class="cabecera-panel"><h3>${textoPortal("txt_resumen_de_la_bolsa")}</h3></div>
          <div class="cuerpo-panel">
            <dl class="resumen-expediente">
              <div class="fila-resumen"><dt>${textoPortal("txt_categoria")}</dt><dd>${escaparHTML(bolsa.categoria)}</dd></div>
              <div class="fila-resumen"><dt>${textoPortal("txt_tipo_de_lista")}</dt><dd>${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</dd></div>
              <div class="fila-resumen"><dt>${textoPortal("txt_vigencia")}</dt><dd>${escaparHTML(vigenciaBolsa)}</dd></div>
              <div class="fila-resumen"><dt>${textoPortal("txt_personas_en_bolsa")}</dt><dd>${numero(bolsa.total)}</dd></div>
            </dl>
          </div>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><h3>${textoPortal("txt_criterios_de_orden")}</h3></div>
          <div class="cuerpo-panel"><dl class="resumen-expediente">
            <div class="fila-resumen"><dt>${textoPortal("txt_criterio")}</dt><dd>${textoPortal("txt_puntuacion_descendente_desempate_estable_por_n_d")}</dd></div>
            <div class="fila-resumen"><dt>${textoPortal("txt_tipo")}</dt><dd>${escaparHTML(etiquetaClave(bolsa.politica_orden.tipo_lista))}</dd></div>
            <div class="fila-resumen"><dt>${textoPortal("txt_reposicion")}</dt><dd>${escaparHTML(etiquetaReposicion(bolsa.politica_orden.reposicion))}</dd></div>
            <div class="fila-resumen"><dt>${textoPortal("txt_version_y_vigencia")}</dt><dd>${textoPortal("txt_version_n", { version: numero(bolsa.politica_orden.version) })} · ${escaparHTML(fechaVisible(bolsa.politica_orden.vigente_desde))}</dd></div>
          </dl>${rotuloPoliticaOrden(bolsa.politica_orden.rotulo)}</div>
        </section>
        ${avisoSustitucion}
        <section class="panel">
          <div class="cabecera-panel"><h3>${textoPortal("txt_siguientes_actuaciones")}</h3></div>
          ${accionesBolsa}
        </section>
      </aside>` : "";
    const nombres = new Map(candidatos.map((c) => [c.participacion_ref, c]));
    const llamadas = candidatos.filter((candidato) => candidato.ultimo_llamamiento).map((candidato)=>({fecha:candidato.ultimo_llamamiento.comunicado_en,candidato,tipo:traducirBolsaInterna("contacto_llamamiento_tipo"),resultado:etiquetaClave(candidato.ultimo_llamamiento.resultado),actor:"—",contacto:"—"}))
      .concat(contactos.map((contacto)=>({fecha:contacto.instante,candidato:nombres.get(contacto.participacion_ref),tipo:traducirBolsaInterna("contacto_tipo"),resultado:traducirBolsaInterna(`contacto_${contacto.resultado}`),actor:contacto.actor_ref,contacto:`${traducirBolsaInterna(`contacto_${contacto.canal}`)} · ${traducirBolsaInterna(`contacto_${contacto.resultado}`)} · ${contacto.anotacion}`})))
      .sort((a, b) => b.fecha.localeCompare(a.fecha));
    const paginaHistorico = Math.max(0, Number(filtrosActuales.pagina_historico) || 0);
    const inicioHistorico = paginaHistorico * 6;
    const paginaLlamadas = llamadas.slice(inicioHistorico, inicioHistorico + 6);
    const tablaHistorico = paginaLlamadas.length === 0
      ? `<tr><td colspan="6" class="vacio-controlado">${traducirBolsaInterna("contacto_historico_vacio")}</td></tr>`
      : paginaLlamadas.map((evento) => {
        const candidato=evento.candidato;
        const nombre = candidato
          ? `<button type="button" class="enlace-tabla" data-bolsa-accion="abrir-ficha-historico" data-participacion-ref="${escaparHTML(candidato.participacion_ref)}" aria-label="${escaparHTML(traducirEnlacesBolsa("historico_abrir_ficha_aria", { nombre: candidato.nombre_visible }))}">${escaparHTML(candidato.nombre_visible)}</button>`
          : escaparHTML(evento.contacto);
        return `<tr><td>${fechaMarcada(evento.fecha)}</td><td>${nombre}${candidato?`<br><code>${escaparHTML(candidato.documento_enmascarado)}</code>`:""}</td><td>${escaparHTML(evento.tipo)}</td><td>${escaparHTML(evento.resultado)}</td><td>${celdaActor(evento.actor)}</td><td>${escaparHTML(evento.contacto)}</td></tr>`;
      }).join("");
    const navegacionHistorico = llamadas.length > 6 ? `<div class="acciones-vista" aria-label="${textoPortal("txt_paginacion_del_historico")}"><span>${textoPortal("txt_mostrando_desde_hasta_total", { desde: numero(inicioHistorico + 1), hasta: numero(Math.min(inicioHistorico + 6, llamadas.length)), total: numero(llamadas.length) })}</span><button type="button" class="boton-secundario" data-bolsa-accion="pagina-historico" data-pagina="${paginaHistorico - 1}"${paginaHistorico === 0 ? " disabled" : ""}>${textoPortal("txt_anterior")}</button><button type="button" class="boton-secundario" data-bolsa-accion="pagina-historico" data-pagina="${paginaHistorico + 1}"${inicioHistorico + 6 >= llamadas.length ? " disabled" : ""}>${textoPortal("txt_siguiente")}</button></div>` : "";
    const pestanas = `<nav class="acciones-vista" role="tablist" aria-label="${textoPortal("txt_vistas_de_la_bolsa")}"><button type="button" class="boton-secundario" role="tab" aria-selected="${pestana === "candidatos"}" data-bolsa-accion="cambiar-pestana" data-pestana="candidatos">${textoPortal("txt_candidatos")}</button><button type="button" class="boton-secundario" role="tab" aria-selected="${pestana === "historico"}" data-bolsa-accion="cambiar-pestana" data-pestana="historico">${textoPortal("txt_historico_de_llamamientos")}</button></nav>`;
    const ultimoTurno = turno?.ultimo_llamado;
    const siguienteTurno = turno?.siguiente;
    const panelTurno = turno && pestana !== "historico" ? `<section class="panel panel-separado" aria-labelledby="bolsa-turno-titulo">
      <div class="cabecera-panel"><h3 id="bolsa-turno-titulo">${textoPortal("bolsa_turno_titulo")}</h3><span class="estado-chip info">${textoPortal(turno.provisional ? "bolsa_turno_regla_provisional" : "bolsa_turno_regla_versionada", { version: numero(turno.politica_version) })}</span></div>
      <div class="cuerpo-panel"><dl class="resumen-expediente">
        <div class="fila-resumen"><dt>${textoPortal("txt_criterios_de_orden")}</dt><dd>${textoPortal("txt_puntuacion_descendente_desempate_estable_por_n_d")} · ${textoPortal(`bolsa_turno_tipo_${bolsa.politica_orden.tipo_lista}`)} · ${textoPortal("bolsa_turno_reposicion", { reposicion: etiquetaReposicion(bolsa.politica_orden.reposicion) })}</dd></div>
        <div class="fila-resumen"><dt>${textoPortal("bolsa_turno_ultimo")}</dt><dd>${ultimoTurno
          ? `<strong>${escaparHTML(ultimoTurno.nombre_visible)}</strong> · ${ultimoTurno.orden === null ? textoPortal("bolsa_turno_sin_puesto") : textoPortal("bolsa_turno_puesto", { orden: numero(ultimoTurno.orden) })}<br><small><time datetime="${escaparHTML(ultimoTurno.comunicado_en)}">${escaparHTML(instanteVisible(ultimoTurno.comunicado_en))}</time> · ${escaparHTML(etiquetaIntentoTurno(ultimoTurno.canal, "canal"))} · ${escaparHTML(etiquetaIntentoTurno(ultimoTurno.resultado, "resultado"))}</small>`
          : textoPortal(bolsa?.total === 0 ? "bolsa_turno_vacio" : "bolsa_turno_sin_contacto")}</dd></div>
        <div class="fila-resumen"><dt>${textoPortal("bolsa_turno_siguiente")}</dt><dd>${siguienteTurno
          ? `<strong>${escaparHTML(siguienteTurno.nombre_visible)}</strong> · ${textoPortal("bolsa_turno_puesto", { orden: numero(siguienteTurno.orden) })}`
          : textoPortal("bolsa_turno_sin_disponibles")}</dd></div>
      </dl>${siguienteTurno ? `<p class="nota-pendiente" role="note">${textoPortal("bolsa_turno_aviso")}</p>` : ""}</div>
    </section>` : "";
    const contenidoHistorico = `<section class="panel" data-bolsa-b5-destino="true" tabindex="-1"><div class="cabecera-panel"><h3>${traducirBolsaInterna("contacto_historico_titulo")}</h3><span class="estado-chip info">${textoPortal("txt_n_registros", { numero: numero(llamadas.length) })}</span></div><div class="tabla-contenedor" tabindex="0" role="region" aria-label="${traducirBolsaInterna("contacto_historico_descripcion")}"><table class="tabla-datos"><caption>${traducirBolsaInterna("contacto_historico_descripcion")}</caption><thead><tr><th scope="col">${textoPortal("txt_fecha")}</th><th scope="col">${textoPortal("txt_candidato")}</th><th scope="col">${textoPortal("txt_tipo")}</th><th scope="col">${textoPortal("txt_resultado")}</th><th scope="col">${textoPortal("txt_actor")}</th><th scope="col">${textoPortal("txt_contacto")}</th></tr></thead><tbody>${tablaHistorico}</tbody></table></div>${navegacionHistorico}</section>`;
    return `
      ${encabezadoVista("", tituloBolsa, "", accionesEncabezado)}
      ${reciboFichaFuera}
      <div class="distribucion-llamamiento">
        <div>
          ${pestanas}
          ${panelTurno}
          ${pestana === "historico" ? contenidoHistorico : `<section class="panel" data-bolsa-b5-destino="true" tabindex="-1">
            <div class="cabecera-panel">
              <h3>${textoPortal("txt_situacion_de_los_candidatos")}</h3>
            </div>
            <div class="cuerpo-panel"><div class="rejilla-kpi kpi-candidatos" aria-label="${textoPortal("txt_contadores_por_situacion")}">${contadoresEstado}</div></div>
            <div class="cuerpo-panel">${formularioFiltros}</div>
          </section>`}
          <section class="panel"${pestana === "historico" ? " hidden" : ""}>
            <div class="cabecera-panel"><h3>${textoPortal("txt_relacion_ordenada_de_candidatos")}</h3></div>
            <div class="tabla-contenedor" tabindex="0" role="region" aria-label="${textoPortal("txt_aspirantes_ordenados_por_merito_y_situacion_en_b")}">
              <table class="tabla-datos tabla-datos--candidatos">
                <caption>${textoPortal("txt_aspirantes_ordenados_por_merito_y_situacion_en_b")}</caption>
                <thead><tr><th scope="col">${textoPortal("txt_orden")}</th><th scope="col">${textoPortal("txt_aspirante")}</th><th scope="col">${textoPortal("txt_dni")}</th><th scope="col">${textoPortal("txt_situacion")}</th><th scope="col">${textoPortal("txt_fecha")}</th><th scope="col">${textoPortal("txt_disponible_desde")}</th><th scope="col">${textoPortal("txt_ultimo_llamamiento")}</th></tr></thead>
                <tbody>${cuerpoTabla}</tbody>
              </table>
            </div>
            ${paginacion}
          </section>
        </div>
        ${resumenBolsa}
      </div>`;
  }
  function renderizarModalFicha(modal, fichaId, columnas = 7) {
    if (!modal || !modal.abierto) return "";
    const candidato = modal.candidato;
    const bolsa = modal.bolsa;
    if (!candidato || !bolsa) return "";
    const vigencia = bolsa.vigente_hasta
      ? `${instanteVisible(bolsa.vigente_desde)} — ${instanteVisible(bolsa.vigente_hasta)}`
      : traducirPortal("txt_vigencia_abierta", { desde: instanteVisible(bolsa.vigente_desde) });
    const disponibilidad = candidato.disponible_desde
      ? `<div class="fila-resumen"><dt>${textoPortal(candidato.estado_clave === "disponible_desde" ? "rrhh_no_disponible_hasta" : "txt_disponible_desde")}</dt><dd>${escaparHTML(instanteVisible(candidato.disponible_desde))}</dd></div>`
      : "";
    const ultimoLlamamiento = candidato.ultimo_llamamiento
      ? `<div class="fila-resumen"><dt>${textoPortal("txt_ultimo_llamamiento")}</dt><dd>${escaparHTML(etiquetaClave(candidato.ultimo_llamamiento.canal))} · ${escaparHTML(etiquetaClave(candidato.ultimo_llamamiento.resultado))}<br><small><time datetime="${escaparHTML(candidato.ultimo_llamamiento.comunicado_en)}">${escaparHTML(instanteVisible(candidato.ultimo_llamamiento.comunicado_en))}</time></small></dd></div>`
      : `<div class="fila-resumen"><dt>${textoPortal("txt_ultimo_llamamiento")}</dt><dd>${textoPortal("txt_sin_llamamientos_registrados")}</dd></div>`;
    const destinos = destinosSituacion(modal.reglasSituacion, candidato.estado_clave);
    const fechaPropuesta = fechaDisponiblePropuesta(modal.reglasSituacion, modal.reposicion);
    const cambio = modal.cambioSituacion ? `
      <form data-bolsa-form="cambio-situacion" data-participacion-ref="${escaparHTML(candidato.participacion_ref)}">
        <label>${textoPortal("txt_destino")} <select name="situacion" required><option value="">${textoPortal("txt_elija_nueva_situacion")}</option>${destinos.map((d) => `<option value="${d}">${escaparHTML(etiquetaClave(d))}</option>`).join("")}</select></label>
        ${renderizarCamposReposicion({ reglas: modal.reglasSituacion, candidato, estadoReposicion: modal.reposicion, escaparHTML })}
        <label data-bolsa-fecha-disponible>${textoPortal("txt_fecha_de_disponibilidad")} <input type="datetime-local" name="fecha_disponible"${fechaPropuesta ? ` value="${escaparHTML(fechaPropuesta)}"` : ""}></label>
        <label>${textoPortal("txt_motivo")} <textarea name="motivo" required maxlength="1000"></textarea></label>
        <button type="submit" class="boton-primario"${destinos.length ? "" : " disabled"}>${textoPortal("txt_guardar_cambio")}</button>
        <p class="mensaje-error" role="alert">${escaparHTML(modal.errorCambioSituacion || "")}</p>
      </form>` : "";
    const reciboSituacion = modal.reciboSituacion ? `<p class="mensaje-exito" role="status">${textoPortal("txt_cambio_registrado")} ${justificanteTraducido(modal.reciboSituacion, escaparHTML, (clave) => traducirPortal(`panel_${clave}`))}</p>` : "";
    const reciboContacto = modal.reciboContacto ? `<p class="mensaje-exito" role="status">${escaparHTML(traducirBolsaInterna("contacto_registrado"))} ${justificanteTraducido(modal.reciboContacto, escaparHTML, (clave) => traducirPortal(`panel_${clave}`))}</p>` : "";
    const t = traducirBolsaInterna;
    const opcionLlamamiento = candidato.ultimo_llamamiento
      ? `<option value="${escaparHTML(candidato.ultimo_llamamiento.llamamiento_ref)}">${escaparHTML(t("contacto_ultimo_llamamiento"))}</option>` : "";
    const formularioContacto = `<form data-bolsa-form="contacto" data-participacion-ref="${escaparHTML(candidato.participacion_ref)}" aria-describedby="bolsa-contacto-no-respuesta">
      <h4>${t("contacto_registrar")}</h4>
      <p id="bolsa-contacto-no-respuesta" class="nota-pendiente">${escaparHTML(traducirAvisoPanelInterno("panel_contacto_no_respuesta"))}</p>
      <label>${t("contacto_canal")} <select name="canal" required><option value="telefono">${t("contacto_telefono")}</option><option value="correo">${t("contacto_correo")}</option><option value="sms">${t("contacto_sms")}</option><option value="presencial">${t("contacto_presencial")}</option><option value="otro">${t("contacto_otro")}</option></select></label>
      <label>${t("contacto_resultado")} <select name="resultado" required><option value="contactado">${t("contacto_contactado")}</option><option value="no_contesta">${t("contacto_no_contesta")}</option><option value="buzon">${t("contacto_buzon")}</option><option value="aplazado">${t("contacto_aplazado")}</option><option value="otro">${t("contacto_otro")}</option></select></label>
      ${opcionLlamamiento ? `<label>${t("contacto_llamamiento")} <select name="llamamiento_ref"><option value="">${t("contacto_sin_vincular")}</option>${opcionLlamamiento}</select></label>` : ""}
      <label>${t("contacto_anotacion")} <textarea name="anotacion" required maxlength="1000"></textarea></label>
      <button type="submit" class="boton-primario">${t("contacto_registrar")}</button>
      <p class="mensaje-error" role="alert">${escaparHTML(modal.errorContacto || "")}</p>
    </form>`;
    return `
      <tr class="fila-ficha-participacion" data-ficha-participacion-ref="${escaparHTML(candidato.participacion_ref)}">
        <td colspan="${columnas}">
          <section id="${escaparHTML(fichaId)}" class="panel" data-bolsa-ficha-inline="true" tabindex="-1" aria-labelledby="titulo-${escaparHTML(fichaId)}">
            <div class="cabecera-panel">
              <h3 id="titulo-${escaparHTML(fichaId)}">${textoPortal("txt_ficha_de_persona", { nombre: candidato.nombre_visible })}</h3>
              <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-ficha" aria-label="${textoPortal("txt_cerrar_ficha_de_persona", { nombre: candidato.nombre_visible })}">×</button>
            </div>
            <div class="cuerpo-panel">
              <dl class="resumen-expediente">
                <div class="fila-resumen"><dt>${textoPortal("txt_bolsa")}</dt><dd>${escaparHTML(bolsa.categoria)}<br><small>${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</small></dd></div>
                <div class="fila-resumen"><dt>${textoPortal("txt_vigencia")}</dt><dd>${escaparHTML(vigencia)}</dd></div>
                <div class="fila-resumen"><dt>${textoPortal("txt_orden_del_acta")}</dt><dd>#${numero(candidato.orden_acta)}</dd></div>
                <div class="fila-resumen"><dt>${textoPortal("txt_situacion")}</dt><dd><span class="estado-chip ${candidato.estado_clave === "en_revision" ? "advertencia" : claseEstado(candidato.estado_clave)}">${escaparHTML(etiquetaEstadoBolsa(candidato.estado_clave))}</span></dd></div>
                <div class="fila-resumen"><dt>${textoPortal("txt_ultimo_cambio_de_situacion")}</dt><dd>${escaparHTML(instanteVisible(candidato.estado_desde))}</dd></div>
                ${renderizarMarcasFicha(candidato, escaparHTML)}
                ${disponibilidad}
                ${ultimoLlamamiento}
                <div class="fila-resumen"><dt>${traducirBolsaInterna("contacto_contador")}</dt><dd>${numero(candidato.contactos_total)}</dd></div>
                ${renderizarOrigenContacto({ estado: modal.contactoOrigen || {}, escaparHTML })}
              </dl>
              ${renderizarRegistroContacto({ estado: modal.registroContacto || {}, escaparHTML })}
              ${reciboSituacion}
              ${reciboContacto}
              ${renderizarOperacionesSituacion({ candidato, estado: modal.operacionesB8 || {}, escaparHTML })}
              ${renderizarIntentosContacto({ candidato, estado: modal.intentosContacto || {}, escaparHTML, llamamientoRef: modal.llamamientoSeguimiento || "" })}
              ${renderizarContratosParticipacion({ estado: modal.contratosB13 || {}, escaparHTML, categoria: bolsa.categoria })}
              ${renderizarReincorporacionesTitular({ estado: modal.reincorporacionesTitular || {}, escaparHTML })}
              ${renderizarSanciones({ estado: modal.sancionesB24 || {}, escaparHTML })}
            </div>
            <div class="acciones-vista">
              <button type="button" class="boton-primario" data-bolsa-accion="abrir-cambio-situacion">${textoPortal("txt_cambiar_situacion")}</button>
              <button type="button" class="boton-secundario" data-bolsa-auditoria="${escaparHTML(candidato.participacion_ref)}">${textoPortal("auditoria_participacion_accion")}</button>
              <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-ficha">${textoPortal("txt_cerrar")}</button>
            </div>
          </section>
          ${cambio}
          ${formularioContacto}
        </td>
      </tr>`;
  }
  function renderizarModalContactos(modal) {
    if (!modal || !modal.abierto) return "";
    let contenido = "";
    if (modal.carga === "cargando") {
      contenido = '<p class="vacio-controlado" role="status" aria-busy="true">' + textoPortal("txt_cargando_historial_de_contactos") + '</p>';
    } else if (modal.carga === "error") {
      contenido = `<p class="mensaje-error" role="alert">${escaparHTML(modal.error || traducirPortal("txt_no_se_pudieron_consultar_los_contactos"))}</p>`;
    } else if (!modal.contactos || modal.contactos.length === 0) {
      contenido = '<p class="vacio-controlado" role="status">' + textoPortal("txt_no_hay_contactos_previos_registrados_para_este_a") + '</p>';
    } else {
      const filas = modal.contactos.map((ct) => `
        <tr data-contacto-ref="${escaparHTML(ct.contacto_ref)}">
          <td><span class="estado-chip neutro">${escaparHTML(etiquetaClave(ct.canal))}</span></td>
          <td><time datetime="${escaparHTML(ct.realizado_en)}">${escaparHTML(instanteVisible(ct.realizado_en))}</time></td>
          <td><span class="estado-chip ${claseEstado(ct.resultado_clave)}">${escaparHTML(etiquetaClave(ct.resultado_clave))}</span></td>
          <td><small>${escaparHTML(ct.anotacion || "—")}</small></td>
        </tr>
      `).join("");
      contenido = `
        <div class="tabla-contenedor" tabindex="0" role="region" aria-label="${textoPortal("txt_historial_de_comunicaciones_y_respuestas_del_asp")}">
          <table class="tabla-datos">
            <caption>${textoPortal("txt_historial_de_comunicaciones_y_respuestas_del_asp")}</caption>
            <thead>
              <tr>
                <th scope="col">${textoPortal("txt_canal")}</th>
                <th scope="col">${textoPortal("txt_fecha_y_hora")}</th>
                <th scope="col">${textoPortal("txt_resultado")}</th>
                <th scope="col">${textoPortal("txt_anotacion")}</th>
              </tr>
            </thead>
            <tbody>
              ${filas}
            </tbody>
          </table>
        </div>`;
    }
    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-contactos">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-contactos">${textoPortal("txt_historial_de_contactos_de", { persona: modal.nombreVisible || modal.participacionRef })}</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-contactos" aria-label="${textoPortal("txt_cerrar")}">×</button>
          </div>
          <div class="cuerpo-panel">
            ${contenido}
          </div>
          <div class="acciones-vista">
            <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-contactos">${textoPortal("txt_cerrar")}</button>
          </div>
        </div>
      </div>`;
  }
  function renderizarModalResultado(modal) {
    if (!modal || !modal.abierto) return "";
    const errorHtml = modal.error
      ? `<div class="mensaje-error" role="alert"><p><strong>${textoPortal("txt_error")}</strong> ${escaparHTML(modal.error)}</p></div>`
      : "";
    const enviando = modal.carga === "enviando";
    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-resultado">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-resultado">${textoPortal("txt_registrar_resultado_de_llamamiento")}</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-resultado" aria-label="${textoPortal("txt_cerrar")}">×</button>
          </div>
          <div class="cuerpo-panel">
            ${errorHtml}
            <p>${textoPortal("txt_aspirante_2")} <strong>${escaparHTML(modal.nombreVisible || modal.participacionRef)}</strong></p>
            <form data-bolsa-form="resultado" data-llamamiento-ref="${escaparHTML(modal.llamamientoRef)}">
              <div class="campo-formulario">
                <label for="resultado-clave">${textoPortal("txt_resultado_del_llamamiento")}</label>
                <select id="resultado-clave" name="resultado_clave" required>
                  <option value="">${textoPortal("txt_seleccione_un_resultado")}</option>
                  <option value="aceptado">${textoPortal("txt_aceptado_pasa_a_situacion_ocupado")}</option>
                  <option value="renuncia">${textoPortal("txt_renuncia_pasa_a_renuncia_pendiente")}</option>
                  <option value="sin_respuesta">${textoPortal("txt_sin_respuesta_continua_disponible_tras_salto")}</option>
                </select>
              </div>
              <div class="campo-formulario">
                <label for="resultado-anotacion">${textoPortal("txt_anotacion_administrativa_opcional")}</label>
                <textarea id="resultado-anotacion" name="anotacion" rows="3" maxlength="1024" placeholder="${textoPortal("txt_observaciones_sobre_la_respuesta_o_justificante")}"></textarea>
              </div>
              <div class="campo-confirmacion">
                <label for="resultado-confirmacion">
                  <input type="checkbox" id="resultado-confirmacion" name="confirmacion" value="true" required>
                  ${textoPortal("txt_confirmo_el_resultado_del_llamamiento_y_los_efec")}
                </label>
              </div>
              <div class="acciones-formulario">
                <button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${enviando ? traducirPortal("txt_guardando") : traducirPortal("txt_guardar_resultado")}</button>
                <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-resultado">${textoPortal("txt_cancelar")}</button>
              </div>
            </form>
          </div>
        </div>
      </div>`;
  }
  function renderizarNoConectada(vista) {
    return `
      ${encabezadoVista("", tituloVista(vista), "", '<button type="button" class="boton-secundario" data-vista="resumen">' + textoPortal("txt_volver_al_cuadro_de_mando") + '</button>')}
      <section class="panel"><div class="cuerpo-panel vacio-controlado"><p><strong>${textoPortal("txt_seccion_todavia_no_disponible")}</strong></p></div></section>`;
  }
  function renderizarVista(vista) {
    if (!esActivo()) throw new Error("el presentador requiere un panel interno válido");
    const datos = datosPanel();
    if (vista === "resumen") return renderizarResumen(datos);
    if (vista === "elaboracion") return renderizarConvocatorias(datos);
    if (vista === "bolsa-candidatos") return renderizarCandidatosBolsa();
    if (vista === "estadisticas") return renderizarEstadisticasBolsa();
    return renderizarNoConectada(vista);
  }
  // Sin panel agregado compuesto, el cuadro de bolsas y la lista de candidatos se muestran
  // igualmente: tienen su propia API y no dependen de los indicadores.
  function renderizarSoloBolsas(vista) {
    if (vista === "bolsa-candidatos") return renderizarCandidatosBolsa();
    const estadoAvisos = typeof obtenerDatosAvisos === "function" ? obtenerDatosAvisos() : null;
    return `<div class="cuadro-bolsa-solo">
      ${renderizarCuadroB12()}
      ${renderizarBloqueAvisos({
        estado: estadoAvisos?.carga || "cargando",
        datos: estadoAvisos?.datos || null,
        error: estadoAvisos?.error || "",
      })}</div>`;
  }
  return Object.freeze({ esActivo, etiquetaFuente, renderizarEstadisticasBolsa, renderizarSoloBolsas, renderizarVista });
}
