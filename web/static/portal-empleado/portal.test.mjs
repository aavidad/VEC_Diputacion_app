import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import {
  extraerDatosEnvelopeCanonico,
  validarPanelBolsa,
} from "./portal-contrato.js";
import { validarPropuestaLlamamientoPresentacion } from "./portal-llamamientos-contrato.js";
import { obtenerDatosPresentacion, obtenerPropuestaPresentacion } from "./datos-presentacion.js";
import { AYUDA_PORTAL_BOLSA } from "./ayuda-contenido.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";
import { MENSAJES_PORTAL_ES, traducirPortal } from "./portal-i18n.js";
import { accesoBolsaEfectivo } from "./portal-menu-bolsa.js";

const directorio = new URL("./", import.meta.url);
const [html, manifiestoProduccion, javascript, coordinadorModulos, catalogoI18n, eventos, contrato, contratoLlamamientos, apiLlamamientos, flujoLlamamientos, vistaLlamamientos, panelInterno, resumenPresentacion, datos, ayuda, estilosBase, estilosComponentes, estilosFlujos, estilosCapacidades] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("../../produccion.manifest", directorio), "utf8"),
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal-modulos-coordinador.js", directorio), "utf8"),
  readFile(new URL("portal-i18n.js", directorio), "utf8"),
  readFile(new URL("portal-eventos.js", directorio), "utf8"),
  readFile(new URL("portal-contrato.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-contrato.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-api.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-flujo.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-vista.js", directorio), "utf8"),
  readFile(new URL("portal-panel-interno.js", directorio), "utf8"),
  readFile(new URL("portal-resumen-presentacion.js", directorio), "utf8"),
  readFile(new URL("datos-presentacion.js", directorio), "utf8"),
  readFile(new URL("ayuda-contenido.js", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("portal-componentes.css", directorio), "utf8"),
  readFile(new URL("portal-flujos.css", directorio), "utf8"),
  readFile(new URL("portal-capacidades.css", directorio), "utf8"),
]);
const codigo = `${javascript}\n${eventos}\n${contrato}\n${contratoLlamamientos}\n${apiLlamamientos}\n${flujoLlamamientos}\n${vistaLlamamientos}\n${panelInterno}\n${resumenPresentacion}`;
const estilos = `${estilosBase}\n${estilosComponentes}\n${estilosFlujos}\n${estilosCapacidades}`;

function panelInternoReal() {
  return {
    esquema: "vec.bolsa.panel.interno.v1",
    selector: { clase: "organizacion", organizacion_ref: "org_0123456789abcdef" },
    origen: {
      revision: "rev_0123456789abcdef",
      actualizada_en: "2026-07-17T08:59:00Z",
      demostracion: false,
    },
    prueba_lectura: {
      lectura_ref: "lec_0123456789abcdef",
      auditoria_ref: "aud_0123456789abcdef",
      auditoria_secuencia: 17,
      decision_ref: "dec_0123456789abcdef",
      huella_decision_sha256: "a".repeat(64),
      correlacion_ref: "cor_0123456789abcdef",
      confirmada_en: "2026-07-17T09:00:00Z",
    },
    indicadores: {
      convocatorias_borrador: 2,
      convocatorias_revision: 1,
      convocatorias_pendientes_firma: 1,
      convocatorias_publicadas: 4,
      bolsas_activas: 3,
      bolsas_suspendidas: 0,
      bolsas_agotadas: 0,
      llamamientos_pendientes: 5,
      llamamientos_en_curso: 2,
      llamamientos_vencen_hoy: 1,
      documentos_pendientes_firma: 3,
      incidencias_abiertas: 1,
    },
    convocatorias: [{
      convocatoria_ref: "cnv_0123456789abcdef",
      categoria_clave: "auxiliar_administrativo",
      estado_clave: "revision",
      plazo_cierra_en: "2026-07-19T09:00:00Z",
      numero_solicitudes: 120,
      numero_pendientes: 7,
    }],
    actuaciones_pendientes: [{
      actuacion_ref: "act_0123456789abcdef",
      recurso_ref: "cnv_0123456789abcdef",
      tipo_clave: "revisar_bases",
      estado_clave: "pendiente",
      prioridad_clave: "alta",
      fecha_limite: "2026-07-18T09:00:00Z",
      numero_elementos: 1,
    }],
  };
}

test("la carga inicial usa solo las APIs de Bolsa que están compuestas", () => {
  assert.doesNotMatch(javascript, /\/api\/vec\/bolsa\/panel/);
  assert.doesNotMatch(javascript, /const cargaDisponibilidad = superficieBorradores/);
  assert.match(javascript, /await coordinadorModulos\.cargarInterno\(\)\.catch/);
  assert.match(javascript, /controladorBolsas\.cargarBolsas\(\)/);
  // same-origin es obligatorio para presentar el certificado mTLS (y la
  // autenticación de la demo en el proxy); include enviaría credenciales a otros orígenes.
  assert.doesNotMatch(`${javascript}\n${apiLlamamientos}`, /credentials: "include"/);
  assert.doesNotMatch(javascript, /document\.cookie|localStorage.*(?:token|sesion|auth)/i);
  assert.doesNotMatch(javascript, /PROVEEDOR_BEARER_BORRADORES|globalThis\[[^\]]*BEARER/i);
  assert.doesNotMatch(javascript, /Bearer|Authorization|resolverProveedorBearer|obtenerBearer/i);
  assert.match(contrato, /la API interna no puede responder con datos de demostración/);
  assert.match(javascript, /let DATOS_PANEL = DATOS_VACIOS/);
  assert.match(javascript, /superficieBorradores\.obtenerAcceso\(\)/);
  assert.doesNotMatch(javascript, /resolverAcceso\(clave, estado\.fuenteLista\)/);
  assert.doesNotMatch(codigo, /María Pérez|García López|Auxiliar Administrativo|BOL-2026|CON-2026|DOC-[A-Z]{2}|20\/07\/2026/);
});

test("Bolsa actualiza el indicador del shell al terminar B12 sin cambiar la vista", () => {
  const inicio = javascript.indexOf("function actualizarVistaBolsa(");
  const fin = javascript.indexOf("function renderizar()", inicio);
  assert.ok(inicio > 0 && fin > inicio);
  const pasos = [];
  const raiz = {};
  const actualizar = runInNewContext(`${javascript.slice(inicio, fin)}; actualizarVistaBolsa`, {
    porId: () => raiz,
    VISTAS_INTERNAS_BOLSA: ["resumen"],
    estado: { vista: "resumen" },
    actualizarNavegacionModulos: () => pasos.push("indicador"),
    montarVistaBolsa: () => pasos.push("bolsa"),
    renderizar: () => pasos.push("portal"),
  });
  actualizar();
  assert.deepEqual(pasos, ["indicador", "bolsa"]);
});

test("Bolsa solo figura comprobando mientras existe una consulta activa", () => {
  const inicio = javascript.indexOf("function disponibilidadBolsa()");
  const fin = javascript.indexOf("function resolverAccesoPerfil(", inicio);
  assert.ok(inicio > 0 && fin > inicio);
  const estadoBolsa = { modoPresentacion: false, vista: "portal", datosBolsas: null };
  const disponibilidad = runInNewContext(`${javascript.slice(inicio, fin)}; disponibilidadBolsa`, {
    estado: estadoBolsa,
    accesoBolsaEfectivo,
    superficieBorradores: { obtenerAcceso: () => ({ disponible: false, vista: "", estado: "cargando" }) },
  });
  assert.equal(disponibilidad().estado, "no_disponible");
  estadoBolsa.datosBolsas = { carga: "cargando" };
  assert.equal(disponibilidad().estado, "cargando");
  estadoBolsa.datosBolsas = { carga: "listo", datos: { bolsas: [{ referencia: "bolsa:ejemplo" }] } };
  assert.equal(disponibilidad().disponible, true);
  estadoBolsa.datosBolsas = { carga: "error" };
  assert.equal(disponibilidad().estado, "error");
});

test("la carga inicial no consulta servicios de Bolsa ausentes", () => {
  const inicio = javascript.indexOf("async function cargarFuenteDatos()");
  const fin = javascript.indexOf("function necesidadLlamamientoSeleccionada()", inicio);
  const cargaInicial = javascript.slice(inicio, fin);
  assert.ok(inicio > 0 && fin > inicio);
  assert.doesNotMatch(cargaInicial, /fetch\(/);
  assert.doesNotMatch(cargaInicial, /comprobarDisponibilidad/);
  assert.doesNotMatch(cargaInicial, /API_PANEL_BOLSA/);
  assert.match(cargaInicial, /requiereLecturaBolsas\(estado\.vista\)/);
  const vistasSinLectura = javascript.match(/const VISTAS_BOLSA_SIN_LECTURA = new Set\(\[([\s\S]*?)\]\);/)?.[1] || "";
  for (const vista of ["contratos", "seleccion-inscripciones", "seleccion-pruebas", "seleccion-comunicaciones"])
    assert.match(vistasSinLectura, new RegExp(`"${vista}"`));
});

test("el arranque desconocido normaliza a portal sin sondear Bolsa", () => {
  const inicio = javascript.indexOf("async function inicializar()");
  const arranque = javascript.slice(inicio);
  assert.match(arranque, /estado\.vista = vistaDesdeHash\(\)/);
  assert.doesNotMatch(arranque, /controladorBolsas\.cargarBolsas\(\)/);
  assert.match(javascript, /if \(Object\.hasOwn\(TITULOS, candidata\)\) return candidata/);
  assert.match(javascript, /history\.replaceState\(null, "", "#portal"\);\s+return "portal"/);
});

test("solo Elaboración compone el destructor de borradores con el de B5 y B12", () => {
  assert.match(javascript, /if \(vista === "elaboracion"\) superficieBorradoresActiva\(\)\?\.desmontar\(\)/);
  assert.match(javascript, /controladorBolsas\.cancelarPeticiones\(\)/);
  assert.match(javascript, /estado\.vista === "elaboracion"\) actualizarVistaBolsa\(\)/);
  assert.doesNotMatch(javascript, /estado\.vista === "elaboracion"\) renderizar\(\)/);
});

test("el contrato real exige envelope canónico y rechaza una raíz raw", () => {
  const panel = panelInternoReal();
  assert.throws(() => extraerDatosEnvelopeCanonico(panel), /envelope canónico/);
  assert.deepEqual(extraerDatosEnvelopeCanonico({ data: panel }), panel);
  const validado = validarPanelBolsa(extraerDatosEnvelopeCanonico({ data: panel }));
  assert.equal(validado.esquema, "vec.bolsa.panel.interno.v1");
  assert.equal(validado.convocatorias[0].numero_pendientes, 7);
  assert.equal(validado.actuaciones_pendientes[0].tipo_clave, "revisar_bases");
});

test("el panel global prohíbe candidatos y la propuesta es un contrato separado", () => {
  const panel = { ...panelInternoReal(), candidatos: [] };
  assert.throws(() => validarPanelBolsa(panel), /no admite listados/);
  assert.doesNotMatch(datos, /\bcandidatos\s*:/);
  assert.doesNotMatch(datos, /\bdni\s*:/i);
  assert.doesNotMatch(codigo, /data-candidato|Nombre o DNI parcial|filtros-candidatos/);
  const propuesta = validarPropuestaLlamamientoPresentacion(obtenerPropuestaPresentacion("DEMO-NEC-0045"));
  assert.deepEqual(Object.keys(propuesta.evaluaciones[0]), ["orden", "resultado", "motivos"]);
  assert.throws(() => validarPropuestaLlamamientoPresentacion({
    ...obtenerPropuestaPresentacion("DEMO-NEC-0045"),
    nombre: "dato no permitido",
  }), /contrato cerrado/);
});

test("el contrato real falla cerrado y no completa datos ausentes con ceros o listas", () => {
  const sinIndicador = panelInternoReal();
  delete sinIndicador.indicadores.llamamientos_en_curso;
  assert.throws(() => validarPanelBolsa(sinIndicador), /indicadores no respeta el contrato cerrado/);

  const sinActuaciones = panelInternoReal();
  delete sinActuaciones.actuaciones_pendientes;
  assert.throws(() => validarPanelBolsa(sinActuaciones), /panel interno no respeta el contrato cerrado/);

  const demostracion = panelInternoReal();
  demostracion.origen.demostracion = true;
  assert.throws(() => validarPanelBolsa(demostracion), /no puede responder con datos de demostración/);

  const sinFechasOpcionales = panelInternoReal();
  delete sinFechasOpcionales.convocatorias[0].plazo_cierra_en;
  delete sinFechasOpcionales.actuaciones_pendientes[0].fecha_limite;
  const validado = validarPanelBolsa(sinFechasOpcionales);
  assert.equal(Object.hasOwn(validado.convocatorias[0], "plazo_cierra_en"), false);
  assert.equal(Object.hasOwn(validado.actuaciones_pendientes[0], "fecha_limite"), false);
});

test("el modo real renderiza solo indicadores, convocatorias y actuaciones acreditadas", () => {
  let fuente = validarPanelBolsa(panelInternoReal());
  const escapar = (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "neutro",
    encabezadoVista: (_sobrelinea, titulo, descripcion) => `<header><h2>${escapar(titulo)}</h2><p>${escapar(descripcion)}</p></header>`,
    escaparHTML: escapar,
    numero: (valor) => String(valor),
    obtenerDatosPanel: () => fuente,
    tituloVista: (vista) => `Vista ${vista}`,
  });
  assert.equal(presentador.esActivo(), true);
  const resumen = presentador.renderizarVista("resumen");
  assert.match(resumen, /cnv_0123456789abcdef/);
  assert.match(resumen, /120/);
  assert.match(resumen, /act_0123456789abcdef/);
  assert.match(resumen, /Prueba de lectura/);
  assert.match(resumen, /Datos conectados en modo de solo lectura/);
  assert.match(resumen, /class="rejilla-cuadro-mando"/);
  assert.ok(resumen.indexOf("Cuadro B12") < resumen.indexOf("Convocatorias del ámbito autorizado"));
  assert.ok(resumen.indexOf("Convocatorias del ámbito autorizado") < resumen.indexOf("Actuaciones pendientes"));
  assert.ok(resumen.indexOf("Actuaciones pendientes") < resumen.indexOf("Prueba de lectura"));
  for (const etiqueta of [
    "Bolsas activas", "Llamamientos pendientes", "Llamamientos en curso",
    "Documentos pendientes de firma", "Incidencias abiertas",
  ]) assert.match(resumen, new RegExp(etiqueta));
  assert.doesNotMatch(resumen, /Bolsas suspendidas|Bolsas agotadas|Convocatorias en borrador/);
  assert.doesNotMatch(resumen, /Nuevo llamamiento|actividad|gráfico/i);
  assert.match(presentador.renderizarVista("contratos"), /Funcionalidad no conectada/);

  fuente = validarPanelBolsa(obtenerDatosPresentacion(), true);
  assert.equal(presentador.esActivo(), false);
  assert.throws(() => presentador.renderizarVista("resumen"), /requiere un panel interno válido/);

  assert.match(javascript, /crearPresentadorPanelInterno/);
  assert.match(javascript, /portal-panel-interno\.js\?v=20260924-integracion-b7-v1/);
  for (const indicador of [
    "bolsas_activas", "llamamientos_pendientes", "llamamientos_en_curso",
    "documentos_pendientes_firma", "incidencias_abiertas",
  ]) assert.match(panelInterno, new RegExp(`i\\.${indicador}`));
  assert.match(panelInterno, /No se muestran valores cero, tablas vacías ni controles aparentes/);
  assert.match(javascript, /estado\.modoPresentacion && datos/);
  assert.match(javascript, /datos-presentacion\.js/);
});

test("el coordinador respeta DEC-051 y carga el presentador con versión de caché", () => {
  // Dirección elevó el tope de DEC-051 para el coordinador el 23/09/2026: el
  // montaje mínimo de cada ruta real vive aquí y trocearlo antes de la
  // presentación no aporta. Se congela el tamaño actual para que no crezca sin
  // decisión expresa.
  assert.ok(javascript.split(/\r?\n/).length - 1 <= 950, "portal.js debe mantenerse por debajo de 950 líneas");
  assert.match(html, /portal\.js\?v=20260924-web-paradas-periodos-v1/);
  assert.match(javascript, /portal-modulos-coordinador\.js\?v=20260924-web-paradas-periodos-v1/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v5/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-web-c-ayuda-v5/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v4/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-web-c-ayuda-v4/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-v3/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-web-c-v3/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-dietas-consulta-v2/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-f2-dietas-consulta-v2/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cronos-permisos-v2/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-f2-cronos-permisos-v2/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cronos-permisos-v1/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-f2-cronos-permisos-v1/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-personal-estados-v4/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-f2-personal-estados-v4/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-p1-personal-interno-v2/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-p1-personal-interno-v2/);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cache-v3/);
  assert.doesNotMatch(javascript, /portal-modulos-coordinador\.js\?v=20260924-f2-cache-v3/);
  assert.match(javascript, /portal-inicio\.js\?v=20260924-f2-cronos-permisos-v2/);
  assert.match(javascript, /portal-i18n\.js\?v=20260924-f2-cronos-permisos-v2/);
  assert.match(javascript, /traducirPortal\("error_catalogo_modulos"\)/);
  assert.match(javascript, /portal-bolsas-api\.js\?v=20260924-integracion-b7-v1/);
  assert.match(javascript, /portal-borradores-ui\.js\?v=20260924-f2-cronos-permisos-v2/);
  assert.match(javascript, /portal-eventos\.js\?v=20260924-f2-cronos-permisos-v2/);
  assert.match(javascript, /import\("\.\/portal-resumen-presentacion\.js\?v=20260721-acceso-real-v2"\)/);
  assert.doesNotMatch(javascript, /^import .*portal-resumen-presentacion/m);
  assert.doesNotMatch(manifiestoProduccion, /portal-resumen-presentacion\.js/);
  assert.match(manifiestoProduccion, /portal-i18n\.js/);
  assert.match(javascript, /if \(acceso\.disponible !== true \|\| vistaPermitida\(acceso\.vista\)\) return acceso/);
  assert.match(eventos, /case "reintentar-borradores"/);
  assert.match(eventos, /comprobarDisponibilidadBorradores\(\{ forzar: true \}\)/);
  assert.match(panelInterno, /export function crearPresentadorPanelInterno/);
});

test("el montaje de Elaboración conserva la referencia de navegación hasta la superficie", () => {
  assert.match(javascript, /montar: \(\{ vista, raiz, opciones \}\)/);
  assert.match(javascript, /function montarVistaBolsa\(vista, contenedor, opciones = \{\}, \{ activar = true \} = \{\}\)/);
  assert.match(javascript, /superficie\.activar\(\{ referencia: opciones\.referencia \}\)/);
  assert.match(javascript, /if \(activar\) void superficie\.activar/);
  assert.match(javascript, /alCambiar: \(\) => \{ if \(estado\.vista === "elaboracion"\) actualizarVistaBolsa\(\); \}/);
  assert.match(javascript, /function actualizarVistaBolsa\(\{ activar = false \} = \{\}\)/);
  assert.match(javascript, /superficieBorradoresActiva\(\)\?\.desmontar\(\)/);
});

test("el hash directo de CT falla cerrado con retorno seguro y sin mensajes de Bolsa", () => {
  assert.match(javascript, /"contratacion-temporal": \[/);
  assert.match(javascript, /if \(Object\.hasOwn\(TITULOS, candidata\)\) return candidata/);
  const decision = javascript.indexOf('contenedor.innerHTML = estado.vista === "contratacion-temporal"');
  const montaje = javascript.indexOf("void coordinadorModulos.montarVista", decision);
  assert.ok(decision > 0 && montaje > decision, "la indisponibilidad debe resolverse antes del montaje");

  const inicio = javascript.indexOf("function renderizarContratacionTemporalNoDisponible()");
  const fin = javascript.indexOf("function encabezadoVista", inicio);
  const vistaCerrada = javascript.slice(inicio, fin);
  for (const clave of [
    "contratacion_temporal_encabezado",
    "estado_modulo_no_disponible_titulo",
    "contratacion_temporal_descripcion_no_disponible",
    "contratacion_temporal_aviso_no_disponible",
    "accion_volver_portal",
  ]) {
    assert.match(vistaCerrada, new RegExp(`traducirPortal\\("${clave}"\\)`));
    assert.match(catalogoI18n, new RegExp(`\\b${clave}:`));
  }
  assert.match(javascript, /traducirPortal\("contratacion_temporal_miga"\)/);
  assert.match(javascript, /traducirPortal\("contratacion_temporal_titulo"\)/);
  assert.match(coordinadorModulos, /textoEstado: traducir\("estado_modulo_no_disponible_titulo"\)/);
  for (const clave of [
    "contratacion_temporal_encabezado",
    "contratacion_temporal_miga",
    "contratacion_temporal_titulo",
    "estado_modulo_no_disponible_titulo",
    "contratacion_temporal_descripcion_no_disponible",
    "contratacion_temporal_aviso_no_disponible",
    "accion_volver_portal",
  ]) {
    const literal = MENSAJES_PORTAL_ES[clave];
    assert.equal(`${javascript}\n${coordinadorModulos}`.includes(literal), false,
      `el literal visible de CT debe residir solo en el catálogo i18n: ${literal}`);
    assert.equal(catalogoI18n.includes(literal), true);
  }
  assert.doesNotMatch(vistaCerrada, /Gestión de Bolsas|No se han cargado datos de Bolsa|recargar-fuente/);
});

test("la propuesta real usa el cliente cerrado y no habilita un detalle inexistente", () => {
  assert.match(apiLlamamientos, /const RUTA_PROPUESTAS_LLAMAMIENTO = "\/api\/vec\/bolsa\/propuestas-llamamiento"/);
  assert.match(apiLlamamientos, /if \(capacidad !== true\)/);
  assert.match(apiLlamamientos, /esquema: "vec\.bolsa\.propuesta-llamamiento\.solicitud\.v1"/);
  assert.doesNotMatch(`${javascript}\n${apiLlamamientos}`, /Idempotency-Key|randomUUID|claveIdempotenciaPropuesta/);
  assert.match(datos, /solicitar_propuesta_llamamiento: false/);
  assert.match(flujoLlamamientos, /conoce el cliente HTTP/);
  assert.match(javascript, /import\("\.\/portal-presentacion-adaptador\.js/);
  assert.doesNotMatch(javascript, /^import .*portal-presentacion-adaptador/m);
  assert.match(`${flujoLlamamientos}\n${vistaLlamamientos}`, /Detalle no disponible/);
  assert.match(eventos, /if \(resultado\.avanzar === true\) estado\.pasoLlamamiento = 2/);
  // Ninguna clave de puntuación fabricada para candidatos; el nombre de la columna
  // «Puntuación» en las incidencias de importación es un texto, no una puntuación.
  assert.doesNotMatch(datos, /puntuaci[oó]n[a-z_]*\s*:/i);
  assert.doesNotMatch(contratoLlamamientos, /evaluaciones.*confirmacion|camposEvaluacion/i);
});

test("los datos de presentación están aislados y se activan de forma explícita", () => {
  const presentacion = validarPanelBolsa(obtenerDatosPresentacion(), true);
  assert.equal(presentacion.esquema, "vec.bolsa.panel.presentacion.v1");
  assert.equal(presentacion.demostracion, true);
  assert.ok(presentacion.bolsas.length > 0);
  assert.match(javascript, /getAll\("presentacion"\)/);
  assert.match(javascript, /import\("\.\/datos-presentacion\.js/);
  assert.match(datos, /ADAPTADOR EXCLUSIVO DE PRESENTACIÓN RRHH/);
  assert.match(datos, /esquema: "vec\.bolsa\.panel\.presentacion\.v1"/);
  assert.match(datos, /demostracion: true/);
  assert.match(html, /class="aviso-presentacion" role="status" hidden/);
  assert.match(html, /Referencias públicas de convocatoria y BOP reales/);
  assert.match(html, /Personas, expedientes y actuaciones internas sintéticos/);
  assert.doesNotMatch(datos, /\b\d{8}[A-Z]\b/);
});

test("la presentación RRHH usa referencias públicas reales y bases adaptadas locales", async () => {
  const elaboraciones = obtenerDatosPresentacion().elaboraciones;
  assert.deepEqual(elaboraciones.map((item) => item.cve_bop), [
    "BOP-GRA-2025-125002",
    "BOP-GRA-2024-244002",
    "BOP-GRA-2026-043004",
  ]);
  assert.deepEqual(elaboraciones.map((item) => item.publicacion_bop), [
    "04/07/2025",
    "19/12/2024",
    "05/03/2026",
  ]);
  assert.match(elaboraciones[0].nombre, /Auxiliar de Servicios Generales/);
  assert.equal(elaboraciones[1].nombre, "Ingreso en la Subescala de Gestión de Administración General");
  assert.equal(elaboraciones[1].identificador_publico, "gestion-administracion-general-2024");
  assert.match(elaboraciones[2].nombre, /Bolsa de empleo de Operario/);
  for (const elaboracion of elaboraciones) {
    assert.match(elaboracion.expediente, /^DEMO-/);
    assert.match(elaboracion.fase, /DEMO/);
    assert.equal(elaboracion.documentos_publicos.length, 2);
    for (const documento of elaboracion.documentos_publicos) {
      assert.match(documento.url, /^\/bolsa\/documentos\/bases-(?:auxiliar|gestion|operario)-demo\.(?:pdf|html)$/);
      const ruta = new URL(`../${documento.url.slice(1)}`, directorio);
      assert.ok((await stat(ruta)).size > 1_000, `${documento.url} debe ser un documento real de la presentación`);
    }
  }
});

test("el selector de perfil es cerrado y la navegación aplica mínimo privilegio", () => {
  assert.match(javascript, /getAll\("presentacion"\)/);
  assert.match(javascript, /getAll\("perfil"\)/);
  assert.match(javascript, /valores\.length !== 1[\s\S]{0,100}return null/);
  assert.doesNotMatch(javascript, /valores\.length === 0\) return "administrador"/);
  assert.match(javascript, /\["administrador", "tecnico", "funcionario"\]\.includes/);
  assert.match(javascript, /perfilPresentacionSolicitado\(\) === null\) return vista === "portal"/);
  assert.match(javascript, /adaptador\.obtenerDatosPresentacion\(perfil\)/);
  assert.match(javascript, /function vistaPermitida\(vista\)/);
  assert.match(javascript, /history\.replaceState\(null, "", hashSeguro\)/);
  assert.match(javascript, /control\.disabled = true/);
  assert.match(javascript, /querySelectorAll\("\[data-vista\], \[data-requiere-vista\]"\)/);
  assert.equal((`${javascript}\n${resumenPresentacion}`.match(/data-requiere-vista="llamamientos"/g) || []).length, 2);
  assert.match(eventos, /navegar\(vista, \{ enfocar: false \}\)/);
});

test("el portal interno no usa cookies ni almacenamiento del navegador", () => {
  assert.doesNotMatch(codigo, /localStorage|sessionStorage|document\.cookie/);
  assert.match(eventos, /document\.body\.dataset\.textoGrande/);
  assert.match(eventos, /document\.documentElement\.dataset\.textoGrande/);
  assert.match(estilosBase, /html\[data-texto-grande="true"\][\s\S]{0,80}font-size: 125%/);
  assert.match(eventos, /document\.body\.dataset\.contraste/);
});

test("texto ampliado y contraste siguen disponibles en resoluciones compactas", () => {
  assert.match(html, /id="boton-texto"[^>]+aria-label="Aumentar o restablecer el tamaño del texto"/);
  assert.match(html, /id="boton-contraste"[^>]+aria-label="Activar o desactivar el alto contraste"/);
  assert.doesNotMatch(estilosFlujos, /boton-cabecera:not\(\.boton-avisos\)[^{]*\{[^}]*display:\s*none/);
  assert.match(estilosCapacidades, /body\.portal-empleado-app\s*\{[^}]*font-size:\s*1rem/);
});

test("el portal conserva el shell rico y delega el catálogo sin fijar módulos en la plantilla", () => {
  assert.match(html, /id="navegacion-modulos-dinamica"/);
  assert.match(javascript, /crearCoordinadorModulosPortal/);
  assert.match(javascript, /VISTAS_MODULOS_PERSONALES/);
  assert.match(html, /modulos\/cronos\/cronos\.css/);
  assert.match(html, /modulos\/dietas\/dietas\.css/);
  assert.doesNotMatch(html, /Bolsas de trabajo[\s\S]{0,180}etiqueta-menu/);
  for (const vista of ["resumen", "elaboracion", "convocatorias", "solicitudes", "meritos",
    "baremacion", "alegaciones", "importacion", "llamamientos", "estadisticas", "auditoria", "configuracion"])
    assert.match(html, new RegExp(`data-vista="${vista}"`));
  assert.match(html, /data-categoria-bolsa="contratos" data-vista="contratos"/);
  assert.match(html, /data-categoria-bolsa="documentos" data-vista="contratacion-temporal"/);
  assert.match(javascript, /function renderizarLlamamientoSinBolsa\(\)/u);
  assert.match(javascript, /Elija una bolsa para iniciar un llamamiento\./u);
});

test("la interfaz es semántica, adaptable y no contiene CSS inline", () => {
  assert.doesNotMatch(html.toLowerCase(), /<style\b|\sstyle=/);
  assert.match(html, /Saltar al contenido principal/);
  assert.match(html, /aria-live="polite"/);
  assert.match(estilos, /@media \(max-width: 1040px\)/);
  assert.match(estilos, /@media \(max-width: 780px\)/);
  assert.match(estilos, /@media \(max-width: 520px\)/);
  assert.match(estilos, /prefers-reduced-motion/);
});

test("la ayuda configurable incluye audio local, FAQ y transcripción accesible", async () => {
  assert.equal(AYUDA_PORTAL_BOLSA.esquema, "vec.portal.ayuda.v1");
  assert.ok(AYUDA_PORTAL_BOLSA.pasos.length >= 4);
  assert.ok(AYUDA_PORTAL_BOLSA.preguntas.length >= 3);
  assert.match(javascript, /<audio controls preload="metadata" aria-describedby="transcripcion-ayuda">/);
  assert.match(javascript, /Transcripción del audio/);
  assert.doesNotMatch(AYUDA_PORTAL_BOLSA.audio.src, /^https?:/);
  const rutaAudio = new URL(`.${AYUDA_PORTAL_BOLSA.audio.src.replace("/portal-empleado", "")}`, directorio);
  assert.ok((await stat(rutaAudio)).size > 10_000, "el audio local debe ser reproducible, no un marcador vacío");
  assert.match(ayuda, /Contenido de ayuda sustituible por catálogo o conector/);
});

test("la cabecera usa el logo institucional local, dimensionado y sin hotlink", async () => {
  assert.match(html, /data-identidad-institucional="diputacion-granada"/);
  assert.match(html, /src="\/assets\/logo-diputacion-granada\.svg" width="250" height="84" alt="Diputación de Granada"/);
  assert.match(estilosBase, /\.logo-institucional[\s\S]{0,260}width: min\(100%, 214px\)[\s\S]{0,160}height: auto/);
  assert.doesNotMatch(html, /<img[^>]+src="https?:/i);
  const rutaLogo = new URL("../assets/logo-diputacion-granada.svg", directorio);
  assert.ok((await stat(rutaLogo)).size > 10_000);
  assert.doesNotMatch(await readFile(rutaLogo, "utf8"), /<script\b|<foreignObject\b|\sonload=/i);
});
