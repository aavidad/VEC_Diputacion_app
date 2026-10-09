import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import test from "node:test";
import {
  cargarMensajesPortal,
  crearTraductorPortal,
  MENSAJES_PORTAL,
} from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";

import { versionDe } from "./versiones-cache.test-helper.mjs";

const MENSAJES_PORTAL_INGLES = await cargarMensajesPortal("en");

test("la auditoría del expediente tiene textos simétricos ES y EN", async () => {
  const es = crearTraductorPortal();
  const en = crearTraductorPortal(MENSAJES_PORTAL_INGLES);
  assert.equal(es("auditoria_expediente_panel"), "Auditoría del expediente");
  assert.equal(es("auditoria_expediente_accion"), "Consultar auditoría de este expediente");
  assert.equal(en("auditoria_expediente_panel"), "Case audit trail");
  assert.equal(en("auditoria_expediente_accion"), "View this case’s audit trail");
  const vista = await readFile(new URL("modulos/contratacion-temporal/vista-expedientes.js", import.meta.url), "utf8");
  for (const clave of ["auditoria_expediente_panel", "auditoria_expediente_accion"]) {
    assert.match(vista, new RegExp(`traducirPortal\\(\"${clave}\"\\)`, "u"));
  }
});

test("miga, título, navegación y pie de CT usan el catálogo común en ambos idiomas", async () => {
  const es = crearTraductorPortal();
  const en = crearTraductorPortal(MENSAJES_PORTAL_INGLES);
  const textos = [
    ["contratacion_temporal_miga", "Portal del Empleado → Peticiones de personal temporal", "Employee Portal → Temporary staff requests"],
    ["contratacion_temporal_titulo", "Gestión de peticiones de personal temporal", "Manage temporary staff requests"],
    ["plantillas_rrhh_nav", "Plantillas de documentos", "Document templates"],
    ["txt_modulos", "Áreas", "Areas"],
    ["txt_modulos_del_portal", "Áreas del portal", "Portal areas"],
    ["txt_portal_de_recursos_humanos", "Portal de Recursos Humanos", "Human Resources Portal"],
    ["txt_2026_diputacion_de_granada_portal_del_empleado", "© 2026 Diputación de Granada · Portal del Empleado", "© 2026 Diputación de Granada · Employee Portal"],
    ["txt_proteccion_de_datos_accesibilidad_ayuda", "Protección de datos · Accesibilidad · Ayuda", "Data protection · Accessibility · Help"],
  ];
  const [portal, html] = await Promise.all([
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  for (const [clave, textoES, textoEN] of textos) {
    assert.equal(es(clave), textoES, clave);
    assert.equal(en(clave), textoEN, clave);
    if (clave.startsWith("contratacion_temporal_")) assert.ok(portal.includes(`traducirPortal("${clave}")`));
    else assert.ok(html.includes(`data-i18n-portal${clave === "txt_modulos_del_portal" ? "-aria-label" : ""}="${clave}"`), clave);
  }
});

test("el grafo immutable del catálogo de auditoría usa una sola URL nueva", async () => {
  const raiz = new URL("./", import.meta.url);
  const anteriores = ["20260928-ppt-503-v6", "20260928-auditoria-expediente-en-v1", "20260928-auditoria-expediente-en-v2", "20260929-pref-508a-v2", "20260929-firma-506-v1", "20260929-firma-506-v2", "20260929-auditoria-legible-v1", "20260929-sondeo-opcional-507", "20260929-plazas-306-v1", "20260929-i18n-shell-v1"];
  // El shell pasó sus textos a `textos/<idioma>/portal*.json` (integrado con 5.06, 5.07, 3.06 y 4.11): todo su grafo renueva URL.
  const vigente = "20261001-ct-a-i18n-v1";
  const versionBolsaTurno = "20260930-bolsa-turno-v2";
  const versionTemas = "20260930-codexf-temas-v2";
  const entrada = await readFile(new URL("index.html", raiz), "utf8");
  const coordinador = await readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8");
  const recorridosDietas = await readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8");
  const versionesEspeciales = new Map([
    ["modulos/bolsa/rrhh-plazos-api.js", "20261006-reglas-una-lectura-v1"],
    ["modulos/solicitudes/vista-tramites-propios.js", "20261001-g364-reconciliar-v2"],
    ["modulos/contratacion-temporal/formulario-llamamiento-pruebas.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/circuito-firma-acciones.js", "20261002-ct-fin-modalidad-v1"],
    ["portal-modulos-coordinador.js", "20261007-raiz-conjunta-v2"],
    ["portal.js", "20261007-raiz-conjunta-v2"],
    ["portal-eventos.js", "20261002-ct-fin-modalidad-v1"],
    ["portal-bolsas-ofertas.js", "20261002-r3-r4-ofertas-v1"],
    ["portal-bolsas-traza-valores.js", "20261002-r-rrhh18-v2"],
    ["portal-bolsas-historial-ofrecimientos.js", "20261002-r3-r4-historial-v1"],

    ["modulos/cronos/vista-bandeja-permisos.js", "20261001-f-reconciliacion-321-v1"],
    ["modulos/cronos/i18n-resolucion.js", "20261001-cronos-grafo-bandeja-v5"],
    ["modulos/cronos/i18n-permisos.js", "20261001-cronos-grafo-bandeja-v5"],
    ["modulos/cronos/vista-permisos-propios.js", "20261007-u-cronos-dietas-v1"],
    ["modulos/cronos/vista-movimientos-propios.js", "20261001-cronos-calendario-seleccion-v1"],
    ["modulos/cronos/i18n-incidencias.js", "20261001-cronos-grafo-bandeja-v5"],
    ["modulos/cronos/i18n-consulta.js", "20261001-cronos-grafo-bandeja-v5"],
    ["portal-composicion-empleado.js", "20261002-codexe-d7c-web-v1"],
    ["modulos/cronos/vista-saldo-conectado.js", "20261001-cronos-saldo-explicado-v1"],
    ["modulos/cronos/vista-notificaciones-propias.js", "20261001-cronos-c9-historial-v2"],
    ["modulos/cronos/vista-remoto.js", "20261001-cronos-grafo-bandeja-v5"],
    ["modulos/cronos/vista-movimientos-conectado.js", "20261001-cronos-movimientos-consulta-v1"],
    ["modulos/cronos/i18n-fichaje.js", "20261001-cronos-grafo-bandeja-v5"],
    ["modulos/bolsa/rrhh-plazos-ui.js", "20261006-reglas-una-lectura-v1"],
    ["portal-inicio.js", "20261001-g364-reconciliar-v2"],
    // 5.06, segundo corte: el circuito de firma trae el estado de Firmadoc.
    // Reglas vigentes: el detalle de cada regla y sus textos en catálogos renuevan el enlace del panel.
    // Nuevo llamamiento: el campo «Resumen de la preparación» usa la etiqueta encima y el campo a lo ancho.
    ["portal.js", "20261007-raiz-conjunta-v2"],
    ["portal-vistas-utilidades.js", "20261001-ct-a-i18n-v1"],
    ["portal-preferencias-integracion.js", "20261007-p7-http-v1"],
    ["portal-preferencias.js", "20261007-p7-http-v1"],
    ["portal-preferencias-api.js", "20261007-p7-http-v1"],
    ["portal-borrador-llamamiento-ui.js", "20261001-ct-a-i18n-v1"],
    ["portal-panel-interno.js", "20261005-bolsa-usabilidad-v1"],
    ["portal-bolsas-api.js", "20261002-r-rrhh18-v3"],
    ["portal-bolsas-contrato.js", "20261002-r-rrhh18-v3"],
    ["portal-llamamientos-operaciones-api.js", "20261002-r-rrhh18-v3"],
    ["reglas/enlace.js", "20261001-ct-a-i18n-v1"],
    ["portal-modulos-coordinador.js", "20261007-raiz-conjunta-v2"],
    ["modulos/dietas/vista-recorridos.js", "20261002-codexe-d7c-ux-v3"],
    ["modulos/dietas/vista-bandeja-circuito.js", "20261001-ct-a-i18n-v1"],
    ["modulos/contratacion-temporal/vista-expedientes.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/seguimiento-cese.js", vigente],
    ["modulos/contratacion-temporal/circuito-firma.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/formulario-llamamiento.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/renderizado-llamamiento.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/vista-expedientes.js", "20261002-ct-fin-modalidad-v1"],
    ["portal-modulos-coordinador.js", "20261007-raiz-conjunta-v2"],
    ["modulos/bolsa/baremo/montaje.js", "20261001-f-reconciliacion-319-v1"],
    ["modulos/cronos/vista-permisos-propios.js", "20261007-u-cronos-dietas-v1"],
    ["modulos/dietas/vista-borradores-propios.js", "20261002-codexe-d7c-ux-v3"],
    ["modulos/dietas/vista-rectificacion-dietas.js", "20261002-codexe-d7c-ux-v3"],
    ["modulos/cronos/vista-bandeja-permisos.js", "20261001-f-reconciliacion-321-v1"],
    ["portal-bolsas-operaciones.js", "20261002-r-rrhh18-v2"],
    ["portal-bolsas-sanciones.js", "20261002-r-rrhh18-v2"],
    ["portal-bolsas-avisos.js", "20261002-rrhh17-v1"],
    ["modulos/contratacion-temporal/componentes-expedientes.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/consulta-seguimiento.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/documentacion-formalizacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/fase-firma.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/ficha-ginpix.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/formulario-anotacion-administrativa.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/formulario-cierre-administrativo.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/formulario-propuesta-formalizacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/formulario-resolucion-formalizacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/incorporacion-personal-b2.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/informe-tras-subsanacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/renderizado-plazo-llamamiento.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/seguimiento-incorporacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/vista-expedientes-borrador.js", "20261002-ct-fin-modalidad-v1"],

    ["modulos/contratacion-temporal/vista-expedientes-ficha.js", "20261001-f-reconciliacion-324-v1"],
    ["modulos/contratacion-temporal/vista-expedientes-incorporacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/vista-expedientes-render.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/vista-expedientes-tramitacion.js", "20261002-ct-fin-modalidad-v1"],
    ["modulos/contratacion-temporal/recuentos-peticiones.js", "20261001-f-reconciliacion-325-v1"],
    ["modulos/contratacion-temporal/vista-expedientes-lista.js", "20261001-f-reconciliacion-325-v1"],
    ["portal.js", "20261007-raiz-conjunta-v2"],
  ]);
  versionesEspeciales.set("modulos/contratacion-temporal/vista-estadisticas.js", "20261001-ana002-v4");
  versionesEspeciales.set("portal-modulos-coordinador.js", "20261005-ct-llamamiento-fiscalizacion-v1");
  versionesEspeciales.set("portal-composicion-empleado.js", "20261005-b-contacto-v3");
  versionesEspeciales.set("portal.js", "20261005-bolsa-usabilidad-v1");
  versionesEspeciales.set("modulos/bolsa/rrhh-politica-cese-vista.js", "20261006-reglas-no-disponible-v1");
  // Elaboración: el 404 de borradores es «no disponible».
  versionesEspeciales.set("portal-borradores-acceso.js", "20261006-borradores-no-disponible-v1");
  versionesEspeciales.set("portal-borradores-vista.js", "20261006-borradores-no-disponible-v1");
  versionesEspeciales.set("portal-borradores-ui-soporte.js", "20261006-borradores-no-disponible-v1");
  versionesEspeciales.set("portal-borradores-ui.js", "20261006-borradores-no-disponible-v1");
  versionesEspeciales.set("modulos/solicitudes/vista-tramites-propios.js", "20261004-b-tramites-devoluciones-v2");
  versionesEspeciales.set("modulos/solicitudes/fuente-tramites-propios.js", "20261004-b-tramites-devoluciones-v2");
  versionesEspeciales.set("modulos/personal/vista-contacto-propio.js", "20261004-b-contacto-retoma-v2");
  versionesEspeciales.set("modulos/personal/vista-ficha-integral.js", "20261005-b-contacto-v3");
  versionesEspeciales.set("modulos/personal/vista-historia-servicios-propia.js", "20261004-b-revision-valor-v1");
  versionesEspeciales.set("modulos/personal/vista-preparacion-rectificacion-propia.js", "20261004-b-revision-valor-v1");
  versionesEspeciales.set("modulos/personal/preparacion-rectificacion-propia.js", "20261004-b-rectificacion-validacion-v3");
  versionesEspeciales.set("modulos/personal/vista-rpt-publica.js", "20261004-b-rpt-busqueda-v1");
  versionesEspeciales.set("modulos/personal/registro-b2.js", "20261003-personal-comparacion-b2-v3");
  versionesEspeciales.set("modulos/contratacion-temporal/vista-expedientes.js", "20261005-ct-llamamiento-fiscalizacion-v1");
  versionesEspeciales.set("modulos/contratacion-temporal/vista-expedientes-render.js", "20261005-ct-llamamiento-fiscalizacion-v1");
  versionesEspeciales.set("modulos/contratacion-temporal/vista-expedientes-tramitacion.js", "20261005-ct-llamamiento-fiscalizacion-v1");
  // Sólo estos consumidores CT cambiaron en el grafo de firma V2.
  for (const hoja of [
    "circuito-firma.js",
    "circuito-firma-acciones.js",
    "firma-externa-cliente.js",
    "firma-vec-api.js",
    "preflight-firma-api.js",
    "original-firmable-api.js",
    "i18n-circuito-firma.js",
    "formulario-llamamiento-pruebas.js",
  ]) {
    versionesEspeciales.set(`modulos/contratacion-temporal/${hoja}`, "20261003-ct-firma-v2-v1");
  }
  versionesEspeciales.set("modulos/cronos/vista-bandeja-notificaciones.js", "20261001-cronos-c9-recuperacion-v3");
  versionesEspeciales.set("modulos/cronos/i18n-bandeja-notificaciones.js", "20261001-cronos-c9-recuperacion-v3");
  versionesEspeciales.set("modulos/cronos/i18n-notificaciones-historial.js", "20261001-cronos-c9-historial-v2");
  versionesEspeciales.set("modulos/cronos/i18n-permisos-consulta.js", "20261001-cronos-c7-consulta-v2");
  versionesEspeciales.set("portal-bolsas-contratos.js", "20261002-a-recuperar-379-v1");
  // Resumen de la portada (CT-000184): el cliente y el adaptador del cuadro,
  // la portada y el coordinador cambiaron; su cadena de importadores renueva URL.
  for (const ruta of [
    "categorias-rpt/arranque.js",
    "categorias-rpt/cliente.js",
    "modulos/contratacion-temporal/adaptador-http-expedientes.js",
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/cliente-http-cambios-expediente.js",
    "modulos/contratacion-temporal/cliente-http-consultas-rrhh.js",
    "modulos/contratacion-temporal/cliente-http-informe-definitivo.js",
    "modulos/contratacion-temporal/cliente-http.js",
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js",
    "modulos/contratacion-temporal/fase-firma.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-informe-juridico.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/recuentos-peticiones.js",
    "modulos/contratacion-temporal/renderizado-llamamiento.js",
    "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-cambios.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista-expedientes-lista.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "portal-inicio.js",
  ]) versionesEspeciales.set(ruta, "20261006-resumen-inicio-v2");
  for (const ruta of ["portal-catalogo-modulos.js", "portal-inicio.js"])
    versionesEspeciales.set(ruta, "20261007-ct-menu-recuperacion-v1");
  versionesEspeciales.set("portal.js", "20261007-raiz-main-v1");
  versionesEspeciales.set("portal-modulos-coordinador.js", "20261007-raiz-main-v1");
  versionesEspeciales.set("portal-arranque-aviso.js", "20261007-ct-arranque-autonomo-v1");
  for (const ruta of ["portal.js", "portal-modulos-coordinador.js", "portal-menu-bolsa.js", "portal-llamamientos-selector.js"])
    versionesEspeciales.set(ruta, "20261007-bolsa-llamamientos-unico-v1");
  for (const ruta of ["modulos/auditoria/vista.js", "modulos/auditoria/cliente-http.js"])
    versionesEspeciales.set(ruta, "20261007-auditoria-disponibilidad-v1");
  // Todos estos importadores reales cambiaron de bytes o alcanzan una hoja cambiada.
  for (const ruta of [
    "ayuda-contenido.js", "ayudante-tramites.js", "categorias-rpt/arranque.js", "categorias-rpt/cliente.js",
    "modulos/auditoria/vista.js", "modulos/bolsa/rrhh-plazos-api.js", "modulos/bolsa/rrhh-plazos-ui.js", "modulos/bolsa/rrhh-politica-cese-vista.js",
    "modulos/contratacion-temporal/adaptador-http-expedientes.js", "modulos/contratacion-temporal/alta-renderer-puro.js", "modulos/contratacion-temporal/cancelacion-expediente.js", "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js", "modulos/contratacion-temporal/cliente-http-documentacion-formalizacion.js", "modulos/contratacion-temporal/cliente-http-estadisticas.js", "modulos/contratacion-temporal/cliente-http-informe-definitivo.js",
    "modulos/contratacion-temporal/cliente-http.js", "modulos/contratacion-temporal/componentes-expedientes.js", "modulos/contratacion-temporal/consulta-seguimiento.js", "modulos/contratacion-temporal/contrato-estadisticas.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js", "modulos/contratacion-temporal/etiquetas-vias-cobertura.js", "modulos/contratacion-temporal/fase-firma.js", "modulos/contratacion-temporal/fases-expediente.js",
    "modulos/contratacion-temporal/fases-rrhh-datos.js", "modulos/contratacion-temporal/ficha-ginpix.js", "modulos/contratacion-temporal/formulario-analisis.js", "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-asignacion.js", "modulos/contratacion-temporal/formulario-cierre-administrativo.js", "modulos/contratacion-temporal/formulario-cobertura.js", "modulos/contratacion-temporal/formulario-fiscalizacion.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js", "modulos/contratacion-temporal/formulario-informe-juridico.js", "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js", "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js", "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js", "modulos/contratacion-temporal/i18n-analisis-catalogo.js", "modulos/contratacion-temporal/i18n-avisos-via-cobertura.js",
    "modulos/contratacion-temporal/i18n-borradores-publicados.js", "modulos/contratacion-temporal/i18n-cambios-expediente.js", "modulos/contratacion-temporal/i18n-cancelacion.js", "modulos/contratacion-temporal/i18n-catalogos.js",
    "modulos/contratacion-temporal/i18n-circuito-firma.js", "modulos/contratacion-temporal/i18n-expedientes.js", "modulos/contratacion-temporal/i18n-fases-rrhh.js", "modulos/contratacion-temporal/i18n-ficha-lista.js",
    "modulos/contratacion-temporal/i18n-firma-incorporacion-datos.js", "modulos/contratacion-temporal/i18n-firma-remision.js", "modulos/contratacion-temporal/i18n-informe-tras-subsanacion.js", "modulos/contratacion-temporal/i18n-llamamiento.js",
    "modulos/contratacion-temporal/i18n-seguimiento-cese.js", "modulos/contratacion-temporal/i18n-subsanacion-reparos.js", "modulos/contratacion-temporal/i18n-textos-vistas.js", "modulos/contratacion-temporal/i18n-vistas.js",
    "modulos/contratacion-temporal/i18n.js", "modulos/contratacion-temporal/incorporacion-personal-b2.js", "modulos/contratacion-temporal/informe-tras-subsanacion.js", "modulos/contratacion-temporal/presentador-expedientes.js",
    "modulos/contratacion-temporal/recuentos-peticiones.js", "modulos/contratacion-temporal/renderizado-llamamiento.js", "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js", "modulos/contratacion-temporal/rrhh-plantillas-cliente.js",
    "modulos/contratacion-temporal/rrhh-plantillas-vista.js", "modulos/contratacion-temporal/rrhh-reincorporacion-formulario.js", "modulos/contratacion-temporal/rrhh-reincorporacion-i18n.js", "modulos/contratacion-temporal/seguimiento-cese.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js", "modulos/contratacion-temporal/vista-borradores-publicados.js", "modulos/contratacion-temporal/vista-cuadro-ligera.js", "modulos/contratacion-temporal/vista-estadisticas.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js", "modulos/contratacion-temporal/vista-expedientes-cambios.js", "modulos/contratacion-temporal/vista-expedientes-cancelacion.js", "modulos/contratacion-temporal/vista-expedientes-ficha.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js", "modulos/contratacion-temporal/vista-expedientes-incorporacion.js", "modulos/contratacion-temporal/vista-expedientes-lista.js", "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js", "modulos/contratacion-temporal/vista-expedientes.js", "modulos/contratacion-temporal/vista.js", "modulos/cronos/fecha-civil.js",
    "modulos/cronos/vista-movimientos-propios.js", "modulos/cronos/vista-permisos-propios.js", "modulos/dietas/mapa-ruta.js", "modulos/dietas/vista-bandeja-circuito.js",
    "modulos/dietas/vista-borradores-propios.js", "modulos/dietas/vista-recorridos.js", "modulos/dietas/vista-rectificacion-dietas.js", "modulos/documentos/cliente-http.js",
    "modulos/documentos/vista.js", "modulos/personal/i18n.js", "modulos/personal/registro-b2-actos.js", "modulos/personal/registro-b2-catalogos.js",
    "modulos/personal/registro-b2.js", "modulos/personal/vista-contacto-propio.js", "modulos/personal/vista-estructura-organizativa-publica.js", "modulos/personal/vista-ficha-integral.js",
    "modulos/personal/vista.js", "modulos/solicitudes/i18n.js", "modulos/solicitudes/vista-tramites-propios.js", "modulos/solicitudes/vista.js",
    "organizacion/historico.js", "organizacion/organizacion.js", "peticiones-centro/cancelaciones-centro.js", "peticiones-centro/i18n-peticiones-centro.js",
    "peticiones-centro/incorporaciones-centro.js", "peticiones-centro/peticiones-centro.js", "portal-arranque-aviso.js", "portal-bolsas-api.js",
    "portal-bolsas-avisos.js", "portal-bolsas-contacto-origen.js", "portal-bolsas-contacto-registro.js", "portal-bolsas-contrato.js",
    "portal-bolsas-contratos.js", "portal-bolsas-historial-ofrecimientos.js", "portal-bolsas-intentos.js", "portal-bolsas-marcas.js",
    "portal-bolsas-ofertas.js", "portal-bolsas-operaciones.js", "portal-bolsas-reincorporaciones.js", "portal-bolsas-sanciones.js",
    "portal-bolsas-traza-valores.js", "portal-borrador-llamamiento-api.js", "portal-borrador-llamamiento-ui.js", "portal-borradores-acceso.js",
    "portal-borradores-api.js", "portal-borradores-ui-soporte.js", "portal-borradores-ui.js", "portal-borradores-vista.js",
    "portal-catalogo-modulos.js", "portal-eventos.js", "portal-i18n.js", "portal-idioma.js",
    "portal-inicio.js", "portal-llamamientos-api.js", "portal-llamamientos-flujo.js", "portal-llamamientos-operaciones-api.js",
    "portal-llamamientos-selector.js", "portal-menu-bolsa.js", "portal-modulos-coordinador.js", "portal-panel-interno.js",
    "portal-preferencias-api.js", "portal-preferencias-integracion.js", "portal-preferencias.js", "portal-referencias-i18n.js",
    "portal-vistas-convocatorias.js", "portal-vistas-utilidades.js", "portal.js", "reglas/enlace.js",
    "reglas/i18n.js", "reglas/reglas.js",
  ]) versionesEspeciales.set(ruta, "20261007-pantallas-textos-final-v1");
  // Cohorte exacta de importadores y hojas renovadas en carga por pantalla.
  for (const ruta of [
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js",
    "modulos/contratacion-temporal/fase-firma.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/renderizado-llamamiento.js",
    "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-cuadro-ligera.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-lista.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-inicio.js",
    "portal-modulos-coordinador.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261007-carga-pantalla-v1");

  // La corrección de Cronos exige una URL nueva tras ROOT#847.
  versionesEspeciales.set("modulos/cronos/vista-movimientos-propios.js", "20261007-u-dietas-catalogo-v1");

  // Dietas renueva su cadena; el traductor común conserva la URL de ROOT#847.
  for (const ruta of [
    "modulos/dietas/vista-recorridos.js", "modulos/dietas/vista-borradores-propios.js",
    "modulos/dietas/vista-mapa-comision.js", "modulos/dietas/mapa-ruta.js",
    "portal-modulos-coordinador.js", "portal.js",
  ]) versionesEspeciales.set(ruta, "20261007-u-dietas-catalogo-v1");
  for (const ruta of [
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-modulos-coordinador.js", "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-ct-alta-vista-v1");
  versionesEspeciales.set("portal-panel-interno.js", "20261008-bolsa-enlaces-v1");
  for (const ruta of ["portal-ct-ruta-filtro.js", "portal-eventos.js", "portal-inicio.js",
    "modulos/contratacion-temporal/vista-expedientes-lista.js",
    "modulos/contratacion-temporal/vista-cuadro-ligera.js", "portal-modulos-coordinador.js", "portal.js"])
    versionesEspeciales.set(ruta, "20261008-ct-inicio-v1");
  for (const recurso of ["portal-borradores-api.js", "portal-borradores-vista.js",
    "portal-borradores-acceso.js", "portal-borradores-ui-soporte.js", "portal-borradores-ui.js"])
    versionesEspeciales.set(recurso, "20261008-borradores-error-legible-v1");
  // El selector V1 de la lista se comparte con la vista completa: propagar
  // exactamente esa hoja por todos sus importadores evita dos instancias.
  for (const ruta of [
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js",
    "modulos/contratacion-temporal/fase-firma.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/renderizado-llamamiento.js",
    "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
  ]) versionesEspeciales.set(ruta, "20261008-ct-inicio-v1");
  for (const ruta of ["portal-ct-ruta-filtro.js", "portal-inicio.js", "portal.js"])
    versionesEspeciales.set(ruta, "20261008-bolsa-inicio-v2");
  // Alta por circular: una URL para cada módulo y para sus importadores hasta la entrada.
  for (const ruta of [
    "categorias-rpt/arranque.js",
    "categorias-rpt/cliente.js",
    "modulos/contratacion-temporal/adaptador-http-expedientes.js",
    "modulos/contratacion-temporal/alta-renderer-puro.js",
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/cliente-http-alta.js",
    "modulos/contratacion-temporal/cliente-http-estadisticas.js",
    "modulos/contratacion-temporal/cliente-http-informe-definitivo.js",
    "modulos/contratacion-temporal/cliente-http.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/contrato-estadisticas.js",
    "modulos/contratacion-temporal/contrato.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-analisis.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-asignacion.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-cobertura.js",
    "modulos/contratacion-temporal/formulario-fiscalizacion.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-informe-juridico.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/i18n.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/presentador-expedientes.js",
    "modulos/contratacion-temporal/presentador.js",
    "modulos/contratacion-temporal/rrhh-plantillas-cliente.js",
    "modulos/contratacion-temporal/rrhh-plantillas-vista.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-borradores-publicados.js",
    "modulos/contratacion-temporal/vista-estadisticas.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista.js",
    "peticiones-centro/arranque-peticiones-centro.js",
    "peticiones-centro/peticiones-centro.js",
    "portal-modulos-coordinador.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-circular-v3");
  versionesEspeciales.set("portal-inicio.js", "20261008-alta-rpt-circular-v6");
  // RPT recuperable: estas hojas cambiaron o alcanzan la ficha desde la entrada.
  for (const ruta of [
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista.js",
    "modulos/personal/cliente-http-rpt-publica.js",
    "modulos/personal/i18n.js",
    "modulos/personal/registro-b2-actos.js",
    "modulos/personal/registro-b2-catalogos.js",
    "modulos/personal/registro-b2.js",
    "modulos/personal/vista-estructura-organizativa-publica.js",
    "modulos/personal/vista-ficha-integral.js",
    "modulos/personal/vista-rpt-publica.js",
    "modulos/personal/vista.js",
    "organizacion/arranque.js",
    "organizacion/historico.js",
    "organizacion/organizacion.js",
    "portal-composicion-empleado.js",
    "portal-modulos-coordinador.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-rpt-circular-v4");
  // Cohorte v5: las dos hojas corregidas y sus importadores hasta las entradas.
  for (const ruta of [
    "modulos/contratacion-temporal/adaptador-http-expedientes.js",
    "modulos/contratacion-temporal/alta-renderer-puro.js",
    "modulos/contratacion-temporal/cliente-http-estadisticas.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/contrato-estadisticas.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-analisis.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-asignacion.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-cobertura.js",
    "modulos/contratacion-temporal/formulario-fiscalizacion.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-informe-juridico.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/i18n.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/presentador-expedientes.js",
    "modulos/contratacion-temporal/rrhh-plantillas-cliente.js",
    "modulos/contratacion-temporal/rrhh-plantillas-vista.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-borradores-publicados.js",
    "modulos/contratacion-temporal/vista-estadisticas.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista.js",
    "peticiones-centro/arranque-peticiones-centro.js",
    "peticiones-centro/peticiones-centro.js",
    "portal-modulos-coordinador.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-rpt-circular-v6");
  // Canales de aviso y seguimiento por teléfono de Bolsa, y su acceso desde la ficha CT.
  for (const ruta of [
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-bolsas-api.js", "portal-bolsas-contrato.js", "portal-bolsas-intentos.js",
    "portal-llamamientos-operaciones-api.js", "portal-panel-interno.js",
    "portal-modulos-coordinador.js", "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-canal-telefono-v2");
  for (const ruta of [
    "ayuda-contenido.js",
    "ayudante-tramites.js",
    "modulos/contratacion-temporal/alta-renderer-puro.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista.js",
    "peticiones-centro/arranque-peticiones-centro.js",
    "peticiones-centro/peticiones-centro.js",
    "portal-modulos-coordinador.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-etiquetas-ayuda-v2");
  for (const ruta of [
    "modulos/contratacion-temporal/alta-renderer-puro.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista.js",
    "peticiones-centro/arranque-peticiones-centro.js",
    "peticiones-centro/peticiones-centro.js",
    "portal-modulos-coordinador.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-capacidad-v3");
  // La ficha renueva sus consumidores y sus importadores.
  for (const ruta of [
    "categorias-rpt/cliente.js",
    "categorias-rpt/montaje.js",
    "modulos/contratacion-temporal/adaptador-http-expedientes.js",
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/cliente-http-cambios-expediente.js",
    "modulos/contratacion-temporal/cliente-http-consultas-rrhh.js",
    "modulos/contratacion-temporal/cliente-http-informe-definitivo.js",
    "modulos/contratacion-temporal/cliente-http.js",
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js",
    "modulos/contratacion-temporal/fase-firma.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-informe-juridico.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/presentador-expedientes.js",
    "modulos/contratacion-temporal/renderizado-llamamiento.js",
    "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-cambios.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-modulos-coordinador.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-w-ct-borradores-main-v2");

  // Disponibilidad nominal de las secciones de Bolsa y sus importadores.
  for (const ruta of [
    "portal-bolsas-api.js",
    "portal-bolsas-historial-ofrecimientos.js",
    "portal-bolsas-ofertas.js",
    "portal-bolsas-operaciones.js",
    "portal-bolsas-reincorporaciones.js",
    "portal-bolsas-sanciones.js",
    "portal-panel-interno.js",
    "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-w-bolsa-ficha-main-v2");
  for (const ruta of [
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js",
    "modulos/contratacion-temporal/fase-firma.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/renderizado-llamamiento.js",
    "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-modulos-coordinador.js",
    "portal.js"
]) versionesEspeciales.set(ruta, "20261008-ct-sin-bolsa-v2");
  // HZ8 conserva su idioma, pero consume el cliente y los contratos W de main.
  for (const ruta of ["categorias-rpt/arranque.js", "categorias-rpt/vista.js"])
    versionesEspeciales.set(ruta, "20261008-hz8-idioma-v1");
  versionesEspeciales.set("categorias-rpt/montaje.js", "20261008-hz8-idioma-v2");
  for (const ruta of ["portal-modulos-coordinador.js", "portal.js"])
    versionesEspeciales.set(ruta, "20261008-hz8-idioma-v4");
  // HZ8 conserva la vista y el idioma; la lista ligera CT renueva la raíz.
  for (const ruta of ["categorias-rpt/arranque.js", "categorias-rpt/vista.js"])
    versionesEspeciales.set(ruta, "20261008-hz8-idioma-v1");
  versionesEspeciales.set("categorias-rpt/montaje.js", "20261008-hz8-idioma-v2");
  versionesEspeciales.set("modulos/contratacion-temporal/vista-cuadro-ligera.js", "20261008-ct-centros-v1");
  // Documentos: el consumidor renovado debe llegar a la misma URL en todo el grafo.
  for (const ruta of [
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js",
    "modulos/contratacion-temporal/fase-firma.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/renderizado-llamamiento.js",
    "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-borradores-publicados.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-ficha.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-modulos-coordinador.js", "portal.js",
  ]) versionesEspeciales.set(ruta, "20261008-documentos-ficha-v1");
  versionesEspeciales.set("portal-preferencias-integracion.js", "20261008-inicio-idioma-v1");
  for (const ruta of ["portal-modulos-coordinador.js", "portal.js"])
    versionesEspeciales.set(ruta, "20261008-documentos-ficha-v2");
  for (const ruta of [
    "categorias-rpt/cliente.js", "categorias-rpt/montaje.js",
    "modulos/contratacion-temporal/alta-renderer-puro.js",
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/cliente-http-alta.js",
    "modulos/contratacion-temporal/cliente-http-informe-definitivo.js",
    "modulos/contratacion-temporal/cliente-http.js",
    "modulos/contratacion-temporal/contrato.js",
    "modulos/contratacion-temporal/formulario-informe-juridico.js",
    "modulos/contratacion-temporal/presentador.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista.js",
    "peticiones-centro/peticiones-centro.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-rechazo-v2");
  versionesEspeciales.set("modulos/contratacion-temporal/vista-expedientes-incorporacion.js", "20261008-alta-corte-v1");
  for (const ruta of [
    "portal-bolsas-api.js", "portal-bolsas-historial-ofrecimientos.js", "portal-bolsas-ofertas.js",
    "portal-bolsas-operaciones.js", "portal-bolsas-reincorporaciones.js", "portal-bolsas-sanciones.js",
    "portal-ct-ruta-ficha.js", "portal-ct-ruta-filtro.js", "portal-menu-bolsa.js",
    "portal-modulos-coordinador.js", "portal-panel-interno.js", "portal.js",
    "modulos/contratacion-temporal/adaptador-http-expedientes.js",
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/componentes-expedientes.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/documentacion-formalizacion.js",
    "modulos/contratacion-temporal/fase-firma.js",
    "modulos/contratacion-temporal/ficha-ginpix.js",
    "modulos/contratacion-temporal/formulario-anotacion-administrativa.js",
    "modulos/contratacion-temporal/formulario-cierre-administrativo.js",
    "modulos/contratacion-temporal/formulario-incorporacion-ejercicio.js",
    "modulos/contratacion-temporal/formulario-informe-juridico.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/formulario-llamamiento.js",
    "modulos/contratacion-temporal/formulario-propuesta-formalizacion.js",
    "modulos/contratacion-temporal/formulario-resolucion-formalizacion.js",
    "modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/informe-tras-subsanacion.js",
    "modulos/contratacion-temporal/presentador-expedientes.js",
    "modulos/contratacion-temporal/renderizado-llamamiento.js",
    "modulos/contratacion-temporal/renderizado-plazo-llamamiento.js",
    "modulos/contratacion-temporal/seguimiento-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
    "modulos/contratacion-temporal/vista-expedientes-ficha.js",
    "modulos/contratacion-temporal/vista-expedientes-fiscalizacion.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js",
    "modulos/contratacion-temporal/vista-expedientes-render.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
  ]) versionesEspeciales.set(ruta, "20261008-r-fichas-idioma-nav-v1");
  for (const ruta of ["portal-bolsas-ruta-filtros.js", "portal-bolsas-global.js",
    "portal-bolsas-seguimiento.js", "portal-inicio.js"])
    versionesEspeciales.set(ruta, "20261008-bolsa-global-v2");
  for (const ruta of ["portal-bolsas-api.js", "portal-bolsas-historial-ofrecimientos.js",
    "portal-bolsas-ofertas.js", "portal-bolsas-operaciones.js", "portal-bolsas-sanciones.js",
    "portal-bolsas-traza-valores.js", "portal-panel-interno.js", "portal-modulos-coordinador.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/vista-expedientes.js", "portal.js"])
    versionesEspeciales.set(ruta, "20261008-r-traza-idioma-v1");
  for (const ruta of ["portal-bolsas-api.js", "portal-bolsas-historial-ofrecimientos.js",
    "portal-bolsas-ofertas.js", "portal-panel-interno.js", "portal.js"])
    versionesEspeciales.set(ruta, "20261008-r-personas-v1");
  for (const ruta of [
    "categorias-rpt/cliente.js", "categorias-rpt/montaje.js",
    "modulos/contratacion-temporal/alta-renderer-puro.js",
    "modulos/contratacion-temporal/cliente-http-alta.js",
    "modulos/contratacion-temporal/cliente-http-informe-definitivo.js",
    "modulos/contratacion-temporal/cliente-http.js",
    "modulos/contratacion-temporal/contrato.js",
    "modulos/contratacion-temporal/presentador.js",
    "modulos/contratacion-temporal/vista.js",
    "peticiones-centro/arranque-peticiones-centro.js", "peticiones-centro/peticiones-centro.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-analisis-bolsa-v4");
  for (const ruta of [
    "modulos/contratacion-temporal/circuito-firma-acciones.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/consulta-seguimiento.js",
    "modulos/contratacion-temporal/vista-expedientes-borrador.js",
  ]) versionesEspeciales.set(ruta, "20261008-alta-analisis-bolsa-fichas-v5");
  for (const ruta of [
    "modulos/contratacion-temporal/adaptador-http-expedientes.js",
    "modulos/contratacion-temporal/formulario-informe-juridico.js",
    "modulos/contratacion-temporal/presentador-expedientes.js",
  ]) versionesEspeciales.set(ruta, "20261009-ficha-version-recibo-v1");
  // La asignación se abre junto al recibo de cobertura sin recargar la ficha.
  for (const ruta of [
    "modulos/contratacion-temporal/formulario-cobertura.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-modulos-coordinador.js",
  ]) versionesEspeciales.set(ruta, "20261009-asignacion-cobertura-v1");
  // Instantes de Bolsa validados por su forma ISO, no por los patrones de DNI.
  for (const ruta of [
    "portal-bolsas-api.js",
    "portal-bolsas-contrato.js",
    "portal-bolsas-global.js",
    "portal-bolsas-historial-ofrecimientos.js",
    "portal-bolsas-ofertas.js",
    "portal-bolsas-ruta-filtros.js",
    "portal-bolsas-seguimiento.js",
    "portal-inicio.js",
    "portal-llamamientos-operaciones-api.js",
    "portal-panel-interno.js",
  ]) versionesEspeciales.set(ruta, "20261009-instantes-bolsa-v1");
  // Paso a Personal (B2): ruta /api/vec y aviso de preparación pendiente.
  versionesEspeciales.set("modulos/contratacion-temporal/cliente-http-incorporacion-personal-b2.js", "20261009-b2-ruta-vec-v2");
  for (const ruta of ["modulos/contratacion-temporal/incorporacion-personal-b2.js",
    "modulos/contratacion-temporal/vista-expedientes-incorporacion.js", "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/vista-expedientes.js", "portal-modulos-coordinador.js", "portal.js"])
    versionesEspeciales.set(ruta, "20261009-b2-preparacion-v2");
  // La lista de documentos marca los que el expediente aún no permite preparar.
  for (const ruta of [
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/vista-borradores-publicados.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "portal-modulos-coordinador.js", "portal.js",
  ]) versionesEspeciales.set(ruta, "20261009-borradores-fase-v2");
  const archivos = ["index.html"];
  const pendientes = [""];
  while (pendientes.length) {
    const directorio = pendientes.shift();
    for (const entrada of await readdir(new URL(directorio, raiz), { withFileTypes: true })) {
      const nombre = `${directorio}${entrada.name}`;
      if (entrada.isDirectory()) pendientes.push(`${nombre}/`);
      else if (entrada.name.endsWith(".js")) archivos.push(nombre);
    }
  }
  const versiones = new Map();
  const aristas = [];
  const codigo = new Map();
  const patron = /["']((?:\.{1,2}\/|\/portal-empleado\/)[^"']+?\.js)\?v=([\w.-]+)["']/gu;
  for (const archivo of archivos) {
    const fuente = await readFile(new URL(archivo, raiz), "utf8");
    codigo.set(archivo, fuente);
    for (const [, ruta, version] of fuente.matchAll(patron)) {
      const destino = ruta.startsWith("/portal-empleado/")
        ? ruta.slice("/portal-empleado/".length)
        : new URL(ruta, new URL(archivo, raiz)).pathname.split("/portal-empleado/")[1];
      if (destino) aristas.push({ archivo, destino, version });
    }
  }
  // La página peticiones-centro tiene su propia entrada; esta prueba fija
  // las URL del grafo que arranca en index.html del portal RRHH.
  const alcanzables = new Set(["index.html"]);
  let alcanceAnterior;
  do {
    alcanceAnterior = alcanzables.size;
    for (const { archivo, destino } of aristas) if (alcanzables.has(archivo)) alcanzables.add(destino);
  } while (alcanceAnterior !== alcanzables.size);
  const ancestros = new Set(["portal-i18n.js"]);
  let cantidad;
  do {
    cantidad = ancestros.size;
    for (const { archivo, destino } of aristas) if (ancestros.has(destino)) ancestros.add(archivo);
  } while (cantidad !== ancestros.size);
  assert.ok(ancestros.has("index.html"), "el HTML carga el catálogo mediante el grafo real");
  assert.ok(ancestros.has("modulos/contratacion-temporal/vista-expedientes.js"));
  for (const { archivo, destino, version } of aristas) {
    if (!alcanzables.has(archivo) || !ancestros.has(destino)) continue;
    // Fin y MOAD versionan el grafo de Contratación en dos cortes apilados.
    // La unicidad por destino se comprueba abajo para todo el grafo alcanzable.
    const esperada = ["20261002-ct-fin-modalidad-v1", "20261002-ct-fin-moad-v1", "20261002-rrhh-consulta-moad-v1", "20261002-ct-r5-grafo-v1", "20261002-ct-r5-grafo-v2"].includes(version)
      ? version : (versionesEspeciales.get(destino) ?? vigente);
    assert.equal(version, esperada,
      `${archivo} → ${destino}: URL immutable renovada`);
    const urls = versiones.get(destino) ?? new Set();
    urls.add(version);
    versiones.set(destino, urls);
  }
  for (const [destino, urls] of versiones) assert.equal(urls.size, 1, `${destino}: una URL en todos sus importadores`);
  for (const archivo of ancestros) {
    for (const anterior of anteriores) assert.ok(!codigo.get(archivo)?.includes(anterior), `${archivo}: ningún import antiguo`);
  }
});

test("el catálogo i18n cubre los estados nuevos de acceso, navegación y reintento", () => {
  const traducir = crearTraductorPortal();
  for (const clave of Object.keys(MENSAJES_PORTAL)) {
    assert.equal(typeof traducir(clave), "string");
    assert.notEqual(traducir(clave), "");
  }
  assert.match(traducir("acceso_borradores_denegado"), /permiso/);
  assert.match(traducir("accion_reintentar"), /Reintentar/);
  assert.match(traducir("error_catalogo_modulos"), /áreas del portal/u);
  assert.match(traducir("titulo_error_catalogo_modulos"), /áreas/u);
  assert.equal(traducir("personal_catalogo_profesional"), "Catálogo profesional de Personal");
});

test("un catálogo incompleto o una clave no gobernada fallan cerrados", () => {
  assert.throws(() => crearTraductorPortal({}), /incompleto/);
  const traducir = crearTraductorPortal();
  assert.throws(() => traducir("texto_improvisado"), /desconocida/);
});

test("Cronos toma migas y títulos de Jornada y Permisos del catálogo común", async () => {
  const traducir = crearTraductorPortal();
  assert.equal(traducir("cronos_miga"), "Portal del Empleado → Cronos");
  assert.equal(traducir("cronos_jornada_titulo"), "Cronos · jornada y fichajes");
  assert.equal(traducir("cronos_permisos_miga"), "Portal del Empleado → Cronos → Permisos");
  assert.equal(traducir("cronos_permisos_titulo"), "Cronos · permisos y ausencias");
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  for (const clave of ["cronos_miga", "cronos_jornada_titulo", "cronos_permisos_miga", "cronos_permisos_titulo"]) {
    assert.match(portal, new RegExp(`traducirPortal\\("${clave}"\\)`, "u"));
  }
  assert.doesNotMatch(portal, /"Portal del Empleado → Cronos(?: → Permisos)?"|"Cronos · (?:jornada y fichajes|permisos y ausencias)"/u);
});

test("Bolsa interna usa catálogo común y formatos es-ES para textos y valores", async () => {
  const {
    crearTraductorBolsaInterna,
    MENSAJES_BOLSA_INTERNA,
    formatearFechaPortal,
    formatearNumeroPortal,
  } = await import("./portal-i18n.js");
  const traducir = crearTraductorBolsaInterna();
  for (const clave of Object.keys(MENSAJES_BOLSA_INTERNA)) {
    assert.notEqual(traducir(clave), "");
  }
  assert.equal(traducir("numero_convocatorias", { numero: "3" }), "3 convocatorias encontradas.");
  assert.equal(formatearNumeroPortal(12345), "12.345");
  assert.equal(formatearFechaPortal("2026-08-01T09:00"), "1/8/26, 9:00");
  assert.throws(() => traducir("bolsa_texto_improvisado"), /desconocida/);
});

test("B24 traduce desde el catálogo común los eventos del recurso", async () => {
  const { traducirBolsaInterna, MENSAJES_BOLSA_INTERNA } = await import("./portal-i18n.js");
  assert.equal(traducirBolsaInterna("b24_recurso_historial", { total: 2 }), "Historial del recurso (2)");
  assert.equal(traducirBolsaInterna("b24_recurso_evento", { estado: "Interpuesto", fecha: "24 sept 2026", actor: "persona:registro" }),
    "Interpuesto · 24 sept 2026 · Anotado por persona:registro");
  assert.equal(traducirBolsaInterna("b24_recurso_documento", { referencia: "registro:1" }), "Escrito: registro:1");
  for (const clave of ["b24_recurso_historial", "b24_recurso_evento", "b24_recurso_documento"]) {
    assert.equal(typeof MENSAJES_BOLSA_INTERNA[clave], "string");
  }
});


test("Cronos renueva los traductores de permisos y resolución y todos sus padres", async () => {
  const [coordinador, bandeja, avisos, permisos, portal, html] = await Promise.all([
    "portal-modulos-coordinador.js", "modulos/cronos/vista-bandeja-permisos.js",
    "modulos/cronos/vista-avisos-propios.js", "modulos/cronos/vista-permisos-propios.js", "portal.js", "index.html",
  ].map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  const version = versionDe(coordinador, "./modulos/cronos/i18n-resolucion.js");
  assert.notEqual(version, "20260929-i18n-textos-v1");
  for (const fuente of [bandeja, avisos]) {
    assert.equal(versionDe(fuente, "./i18n-resolucion.js"), version);
    assert.doesNotMatch(fuente, /i18n-resolucion\.js["']/u);
  }
  assert.equal(versionDe(permisos, "./i18n-permisos.js"), version);
  assert.equal(versionDe(coordinador, "./modulos/cronos/vista-avisos-propios.js"), "20261001-cronos-avisos-confirmados-v1");
  assert.equal(versionDe(coordinador, "./modulos/cronos/vista-bandeja-permisos.js"), "20261001-f-reconciliacion-321-v1");
  assert.equal(versionDe(coordinador, "./modulos/cronos/vista-permisos-propios.js"), "20261007-pantallas-textos-final-v1");
  const versionPortal = versionDe(portal, "./portal-modulos-coordinador.js");
  assert.notEqual(versionPortal, "20261001-cronos-grafo-bandeja-v5");
  assert.equal(versionDe(html, "/portal-empleado/portal-modulos-coordinador.js"), versionPortal);
  // La entrada puede renovarse sola (sin el coordinador): basta con que no conserve la URL anterior a Cronos.
  assert.notEqual(versionDe(html, "/portal-empleado/portal.js"), "20261001-cronos-grafo-bandeja-v5");
  assert.notEqual(versionPortal, "20261008-r-traza-idioma-v1");
  assert.notEqual(versionPortal, "20261008-r-personas-fichas-v3");

});

test("Dietas usa una sola cadena de caché desde la entrada hasta mapa y borradores", async () => {
  const rutas = ["index.html", "portal.js", "portal-modulos-coordinador.js",
    "modulos/dietas/vista-recorridos.js", "modulos/dietas/vista-borradores-propios.js"];
  const [html, portal, coordinador, recorridos, borradores] = await Promise.all(
    rutas.map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  const version = "20261007-u-dietas-catalogo-v1";
  const versionRaiz = versionDe(html, "/portal-empleado/portal.js");
  const versionCoordinador = versionDe(portal, "./portal-modulos-coordinador.js");
  const comun = "20261007-pantallas-textos-final-v1";
  assert.notEqual(versionRaiz, "20261008-r-traza-idioma-v1");
  assert.notEqual(versionCoordinador, "20261008-r-traza-idioma-v1");
  assert.equal(versionDe(html, "/portal-empleado/portal.js"), versionRaiz);
  assert.equal(versionDe(html, "/portal-empleado/portal-modulos-coordinador.js"), versionCoordinador);
  assert.equal(versionDe(portal, "./portal-modulos-coordinador.js"), versionCoordinador);
  assert.equal(versionDe(coordinador, "./modulos/dietas/vista-recorridos.js"), version);
  assert.equal(versionDe(coordinador, "./modulos/dietas/mapa-ruta.js"), version);
  assert.equal(versionDe(recorridos, "./vista-borradores-propios.js"), version);
  assert.equal(versionDe(borradores, "./vista-mapa-comision.js"), version);
  assert.equal(versionDe(borradores, "./vista-rectificacion-dietas.js"), comun);
  assert.equal(versionDe(portal, "./portal-i18n.js"), comun);
  assert.equal(versionDe(coordinador, "./portal-i18n.js"), comun);
});


test("Borradores renueva todas las entradas que podían conservar errores antiguos", async () => {
  const leer = (nombre) => readFile(new URL(nombre, import.meta.url), "utf8");
  const [html, portal, ui, acceso, soporte, vista, cache] = await Promise.all([
    "index.html", "portal.js", "portal-borradores-ui.js", "portal-borradores-acceso.js",
    "portal-borradores-ui-soporte.js", "portal-borradores-vista.js", "cache-publica-v1.json",
  ].map(leer));
  const anterior = "20261007-pantallas-textos-final-v1";
  const aristas = [
    ["portal-borradores-api.js", [html, ui, acceso, soporte]],
    ["portal-borradores-vista.js", [html, ui]],
    ["portal-borradores-acceso.js", [html, ui]],
    ["portal-borradores-ui-soporte.js", [html, ui, vista]],
    ["portal-borradores-ui.js", [html, portal]],
  ];
  for (const [recurso, importadores] of aristas) {
    const versiones = importadores.map((codigo) => versionDe(codigo, recurso));
    assert.equal(new Set(versiones).size, 1, `${recurso}: todas las entradas coinciden`);
    assert.notEqual(versiones[0], anterior, `${recurso}: descarta la caché publicada`);
  }
  const versionRaiz = versionDe(html, "portal.js");
  assert.notEqual(versionRaiz, "20261008-bolsa-atras-v1");
  assert.equal(versionDe(cache, "portal.js"), versionRaiz);
});
