import { CAPACIDAD_CONSULTAR_EMPLEADO, validarEstadoConsultaPersonal } from "./contrato.js";
import { crearDatosPersonalPresentacion } from "./datos-presentacion.js";
import { crearTraductorPersonal } from "./i18n.js";

const ESTADOS = Object.freeze({ cargando: ["Comprobando la consulta de Personal", "No se muestra ningún dato hasta recibir una capacidad positiva y una respuesta del servidor."], denegado: ["Consulta de Personal denegada", "El menú y el manifiesto no conceden acceso a datos de empleado."], error: ["Consulta de Personal no disponible", "Un error de composición no habilita datos ni operaciones."], no_habilitada: ["Consulta de Personal aún no habilitada", "El manifiesto declara la capacidad, pero el servidor no compone todavía una ruta de consulta para Personal."] });
const PRESENTACIONES_VALIDAS = new WeakSet();

function escaparHTML(valor) { return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;"); }
function resumen(filas) { return '<dl class="resumen-expediente modulo-personal-resumen">' + filas.map((fila) => '<div><dt>' + escaparHTML(fila[0]) + '</dt><dd>' + escaparHTML(fila[1]) + '</dd></div>').join("") + '</dl>'; }
function tabla(titulo, cabeceras, filas) { return '<section class="panel" aria-labelledby="' + titulo.id + '"><div class="cabecera-panel"><h3 id="' + titulo.id + '">' + escaparHTML(titulo.texto) + '</h3><p>' + escaparHTML(titulo.ayuda) + '</p></div><div class="tabla-contenedor" tabindex="0" role="region" aria-label="' + escaparHTML(titulo.texto) + '"><table class="tabla-datos"><caption>' + escaparHTML(titulo.texto) + '</caption><thead><tr>' + cabeceras.map((x) => '<th scope="col">' + escaparHTML(x) + '</th>').join("") + '</tr></thead><tbody>' + filas.map((fila) => '<tr><th scope="row">' + escaparHTML(fila[0]) + '</th>' + fila.slice(1).map((x) => '<td>' + escaparHTML(x) + '</td>').join("") + '</tr>').join("") + '</tbody></table></div></section>'; }

export function crearPresentacionPersonalDemo() {
  const presentacion = Object.freeze({ esquema: "vec.personal.presentacion.v1", demostracion: true, datos: crearDatosPersonalPresentacion(), traducir: crearTraductorPersonal() });
  PRESENTACIONES_VALIDAS.add(presentacion);
  return presentacion;
}

function vistaDemostracion(presentacion) {
  const datos = presentacion.datos;
  const t = presentacion.traducir;
  const relacion = datos.relacion_actual;
  return '<section class="modulo-personal" aria-labelledby="titulo-personal"><header class="cabecera-vista"><p class="sobrelinea">' + escaparHTML(t("sobrelinea")) + '</p><h2 id="titulo-personal">' + escaparHTML(t("titulo")) + '</h2><p>' + escaparHTML(t("presentacion_demo")) + '</p></header><section class="nota-pendiente" role="note"><strong>' + escaparHTML(t("aviso_demo")) + '</strong> ' + escaparHTML(datos.origen.aviso) + '</section><div class="rejilla-dos-columnas"><section class="panel" aria-labelledby="personal-relacion"><div class="cabecera-panel"><h3 id="personal-relacion">' + escaparHTML(t("relacion_titulo")) + '</h3><p>' + escaparHTML(t("relacion_ayuda")) + '</p></div>' + resumen([[t("cab_referencia"), relacion.referencia], ["Situación", relacion.situacion], ["Puesto", relacion.puesto], ["Unidad", relacion.unidad], ["Grupo", relacion.grupo], [t("cab_observacion"), relacion.observacion]]) + '</section><section class="panel" aria-labelledby="personal-economico"><div class="cabecera-panel"><h3 id="personal-economico">' + escaparHTML(t("nominas_titulo")) + '</h3><p>' + escaparHTML(t("nominas_ayuda")) + '</p></div>' + resumen([["Nómina", t("sin_importes")], ["Bases", t("sin_importes")], ["Dietas", t("sin_pago")], ["Operaciones", "No disponibles en presentación"]]) + '</section></div>' + tabla({ id: "personal-servicios", texto: t("servicios_titulo"), ayuda: t("servicios_ayuda") }, [t("cab_referencia"), t("cab_desde"), t("cab_hasta"), t("cab_descripcion"), t("cab_observacion")], datos.servicios.map((x) => [x.referencia, x.desde, x.hasta, x.descripcion, x.observacion])) + tabla({ id: "personal-formacion", texto: t("formacion_titulo"), ayuda: t("formacion_ayuda") }, [t("cab_referencia"), "Actividad", t("cab_periodo"), t("cab_estado"), "Acreditación"], datos.formacion.map((x) => [x.referencia, x.actividad, x.periodo, x.estado, x.acreditacion])) + tabla({ id: "personal-nominas", texto: t("nominas_titulo"), ayuda: t("nominas_ayuda") }, [t("cab_referencia"), t("cab_periodo"), t("cab_estado"), t("cab_observacion")], datos.nominas.map((x) => [x.referencia, x.periodo, x.estado, x.observacion])) + tabla({ id: "personal-dietas", texto: t("dietas_titulo"), ayuda: t("dietas_ayuda") }, [t("cab_referencia"), t("cab_periodo"), t("cab_estado"), t("cab_observacion")], datos.dietas_cobradas.map((x) => [x.referencia, x.periodo, x.estado, x.observacion])) + '<section class="nota-seguridad" aria-label="Límites de la presentación"><strong>Sin cálculo de trienios.</strong> Los periodos son referencias sintéticas y no se suman, verifican ni producen derechos. La consulta efectiva requiere una decisión positiva del servidor para cada dato y finalidad.</section></section>';
}

function vistaCerrada(estado) {
  const [titulo, detalle] = ESTADOS[estado];
  return '<section class="modulo-personal" aria-labelledby="titulo-personal"><header class="cabecera-vista"><p class="sobrelinea">Módulo Personal</p><h2 id="titulo-personal">' + escaparHTML(titulo) + '</h2><p>' + escaparHTML(detalle) + '</p></header><section class="panel" aria-label="Estado de la capacidad de consulta"><div class="cuerpo-panel"><p class="nota-pendiente" role="status" aria-live="polite"><strong>Sin datos de empleado.</strong> Esta pantalla no solicita, conserva ni presenta identidades, puestos, situaciones, nóminas o servicios prestados.</p><dl class="resumen-expediente modulo-personal-resumen"><div><dt>Capacidad declarada</dt><dd><code>' + CAPACIDAD_CONSULTAR_EMPLEADO + '</code></dd></div><div><dt>Autorización efectiva</dt><dd>No comprobada</dd></div><div><dt>Ruta de consulta</dt><dd>No compuesta</dd></div><div><dt>Operaciones</dt><dd>No disponibles</dd></div></dl><p>Cuando exista una ruta compuesta, el servidor deberá decidir cada consulta. Esta entrada no deduce permisos de la navegación, del manifiesto ni del navegador.</p></div></section></section>';
}

export function renderizarModuloPersonal({ estado = "no_habilitada", presentacion } = {}) {
  validarEstadoConsultaPersonal(estado);
  return PRESENTACIONES_VALIDAS.has(presentacion) && presentacion.demostracion === true ? vistaDemostracion(presentacion) : vistaCerrada(estado);
}
export async function montarModuloPersonal({ raiz, estado = "no_habilitada", presentacion } = {}) {
  if (!raiz || typeof raiz.replaceChildren !== "function") throw new TypeError("raíz de Personal no válida");
  raiz.innerHTML = renderizarModuloPersonal({ estado, presentacion });
  return Object.freeze({ desmontar() { raiz.replaceChildren(); } });
}

