import assert from "node:assert/strict";
import test from "node:test";
import {
  calcularMetricasCuadro,
  calcularResumenBolsas,
  crearVistaInicioPortal,
  renderizarSeccionBolsasInicio,
  tramitesParaInicio,
} from "./portal-inicio.js";

const moduloBolsa = Object.freeze({
  clave: "bolsa",
  sigla: "BOL",
  titulo: "Bolsas de trabajo",
  texto: "Gobierno de convocatorias y llamamientos.",
});

const escaparHTML = (valor) => String(valor)
  .replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;")
  .replaceAll("'", "&#039;");

function renderizar(acceso) {
  return crearVistaInicioPortal({
    encabezadoVista: () => "<header>Portal</header>",
    escaparHTML,
    obtenerCatalogo: () => [moduloBolsa],
    resolverAcceso: () => acceso,
  })();
}

test("la tarjeta anuncia la comprobación sin ofrecer una ruta prematura", () => {
  const html = renderizar({
    disponible: false,
    vista: "",
    estado: "cargando",
    etiqueta: "Comprobando acceso a borradores",
  });
  assert.match(html, /data-modulo-catalogo="bolsa" tabindex="-1" aria-busy="true"/);
  assert.match(html, /role="status" aria-live="polite">Comprobando acceso a borradores/);
  assert.match(html, /<button[^>]+disabled>Comprobando<\/button>/);
  assert.doesNotMatch(html, /data-vista=/);
});

test("la tarjeta diferencia denegación de error técnico y solo este permite reintentar", () => {
  const denegado = renderizar({
    disponible: false,
    vista: "",
    estado: "denegado",
    etiqueta: "Sin permiso para gestionar borradores",
  });
  assert.match(denegado, /Sin permiso para gestionar borradores/);
  assert.match(denegado, /<button[^>]+disabled>Sin permiso<\/button>/);
  assert.doesNotMatch(denegado, /reintentar-borradores/);

  const error = renderizar({
    disponible: false,
    vista: "",
    estado: "error",
    etiqueta: "Servicio de borradores no disponible",
    reintentar: true,
  });
  assert.match(error, /Servicio de borradores no disponible/);
  assert.match(error, /data-accion="reintentar-borradores">Reintentar<\/button>/);
  assert.doesNotMatch(error, /data-vista=/);
});

test("la capacidad propia abre Elaboración aunque el panel agregado no participe", () => {
  const html = renderizar({
    disponible: true,
    vista: "elaboracion",
    estado: "disponible",
    etiqueta: "Borradores disponibles",
  });
  assert.match(html, /Disponible para el perfil activo/);
  assert.match(html, /data-vista="elaboracion">Entrar<\/button>/);
  assert.doesNotMatch(html, /panel|resumen/u);
});

test("G10: calcularMetricasCuadro extrae correctamente en_tramitacion, con_incidencia y en_llamamiento", async () => {
  const { calcularMetricasCuadro } = await import("./portal-inicio.js");

  // Caso nulo o vacío
  assert.deepEqual(calcularMetricasCuadro(null), {
    en_tramitacion: 0,
    con_incidencia: 0,
    en_llamamiento: 0,
  });
  assert.deepEqual(calcularMetricasCuadro({}), {
    en_tramitacion: 0,
    con_incidencia: 0,
    en_llamamiento: 0,
  });

  // Expedientes variados
  const cuadro = {
    expedientes: [
      { id: "EXP-1", estado_clave: "en_curso", fase_clave: "solicitud" },
      { id: "EXP-2", estado: "En tramitación", fase_clave: "llamamiento" },
      { id: "EXP-3", estado_clave: "incidencia", fase_actual: "Llamamiento a candidatos" },
      { id: "EXP-4", estado: "Con incidencia", fase_clave: "resolucion" },
      { id: "EXP-5", estado_clave: "cerrado", fase_clave: "finalizado" },
    ],
  };

  const metricas = calcularMetricasCuadro(cuadro);
  // EXP-1 (en_curso), EXP-2 (En tramitación) -> 2
  assert.equal(metricas.en_tramitacion, 2);
  // EXP-3 (incidencia), EXP-4 (Con incidencia) -> 2
  assert.equal(metricas.con_incidencia, 2);
  // EXP-2 (llamamiento), EXP-3 (Llamamiento a candidatos) -> 2
  assert.equal(metricas.en_llamamiento, 2);
});

test("G10: una página parcial del cuadro no produce cifras", () => {
  assert.equal(calcularMetricasCuadro({ hay_mas: true, expedientes: [{ estado_clave: "en_curso" }] }), null);
  const renderizar = crearVistaInicioPortal({
    encabezadoVista: () => "",
    escaparHTML,
    obtenerCatalogo: () => [],
    resolverAcceso: () => ({ disponible: true, vista: "bolsa" }),
    esPerfilRRHH: () => true,
    obtenerMetricasCuadro: () => null,
  });
  const html = renderizar();
  assert.doesNotMatch(html, /class="rejilla-metricas-rrhh"/);
  assert.match(html, /Los totales se consultan en el cuadro de mando/);
});

test("C17: los totales del servidor prevalecen sobre una página parcial", () => {
  assert.deepEqual(calcularMetricasCuadro({
    hay_mas: true,
    expedientes: [{ estado_clave: "en_curso" }],
    totales: { total: 52, en_tramitacion: 33, con_incidencia: 4, en_llamamiento: 9 },
  }), { total: 52, en_tramitacion: 33, con_incidencia: 4, en_llamamiento: 9 });
});

test("G10: la vista de inicio para RRHH solo renderiza accesos directos y 3 cifras del cuadro sin nada más", () => {
  const renderizarRRHH = crearVistaInicioPortal({
    encabezadoVista: (sup, tit, desc) => `<header><h1>${tit}</h1><p>${sup}</p></header>`,
    escaparHTML,
    obtenerCatalogo: () => [moduloBolsa],
    resolverAcceso: () => ({ disponible: true, vista: "bolsa" }),
    esPerfilRRHH: () => true,
    obtenerMetricasCuadro: () => ({
      en_tramitacion: 5,
      con_incidencia: 2,
      en_llamamiento: 3,
    }),
    numero: (n) => String(n),
  });

  const html = renderizarRRHH();

  // Encabezado y sección RRHH
  assert.match(html, /Gestión de personal/);
  assert.match(html, /Inicio del portal/);
  assert.match(html, /class="portal-rrhh-inicio"/);

  // 3 accesos directos requeridos
  assert.match(html, /data-vista="contratacion-temporal" data-ct-exp-vista="cuadro">Cuadro de mando<\/button>/);
  assert.match(html, /data-vista="contratacion-temporal" data-ct-exp-vista="alta">Nueva petición<\/button>/);
  assert.match(html, /data-accion="ayuda">Ayuda<\/button>/);

  // 3 cifras leídas del cuadro
  assert.match(html, /data-metrica="en_tramitacion"[^>]*>[\s\S]*?<strong class="metrica-valor">5<\/strong>/);
  assert.match(html, /data-metrica="con_incidencia"[^>]*>[\s\S]*?<strong class="metrica-valor">2<\/strong>/);
  assert.match(html, /data-metrica="en_llamamiento"[^>]*>[\s\S]*?<strong class="metrica-valor">3<\/strong>/);

  // "sin nada más" (sin catálogo de módulos ni rejilla-modulos)
  assert.doesNotMatch(html, /rejilla-modulos/);
  assert.doesNotMatch(html, /tarjeta-modulo/);
  assert.doesNotMatch(html, /data-modulo-catalogo/);
});

test("el inicio de RRHH lista los trámites recientes con incidencias primero y abre cada expediente", () => {
  const cuadro = { expedientes: [
    { expediente_ref: "expediente:ct:a", numero_visible: "2026/CT-000001", centro: "DEPORTES", categoria: "Operario/a", fase_actual: "Solicitud", estado_clave: "en_curso", estado: "En tramitación" },
    { expediente_ref: "expediente:ct:b", numero_visible: "2026/CT-000002", centro: "CULTURA", categoria: "Técnico/a", fase_actual: "Fiscalización", estado_clave: "incidencia", estado: "Con incidencia" },
  ] };
  const tramites = tramitesParaInicio(cuadro);
  assert.equal(tramites[0].expediente_ref, "expediente:ct:b");
  assert.equal(tramitesParaInicio({ expedientes: Array.from({ length: 20 }, (_, i) => ({ expediente_ref: `e${i}` })) }).length, 8);
  const html = crearVistaInicioPortal({
    encabezadoVista: () => "",
    escaparHTML,
    obtenerCatalogo: () => [],
    resolverAcceso: () => ({ disponible: true, vista: "bolsa" }),
    esPerfilRRHH: () => true,
    obtenerMetricasCuadro: () => null,
    obtenerTramitesInicio: () => tramites,
  })();
  assert.match(html, /Trámites recientes/);
  assert.match(html, /data-vista="contratacion-temporal" data-ct-exp-abrir-inicio="expediente:ct:b"/);
  assert.match(html, /class="ct-exp-chip ct-fase-incidencia">Con incidencia</);
  assert.match(html, /2026\/CT-000002[\s\S]*2026\/CT-000001/);
});

test("G21: calcularResumenBolsas calcula totales y extrae top 3 bolsas ordenadas por total", () => {
  // Manejo de valores vacíos o nulos
  assert.equal(calcularResumenBolsas(null), null);
  assert.equal(calcularResumenBolsas([]), null);
  assert.equal(calcularResumenBolsas("invalido"), null);

  const bolsas = [
    { bolsa_ref: "bolsa:ct:1", categoria: "Auxiliar Administrativo", total: 120, por_estado: { disponible: 80 }, estado_clave: "vigente" },
    { bolsa_ref: "bolsa:ct:2", categoria: "Técnico Medio", total: 45, por_estado: { disponible: 30 }, estado_clave: "vigente" },
    { bolsa_ref: "bolsa:ct:3", categoria: "Operario de Servicios", total: 210, por_estado: { disponible: 150 }, estado_clave: "vigente" },
    { bolsa_ref: "bolsa:ct:4", categoria: "Arquitecto/a", total: 15, por_estado: { disponible: 10 }, estado_clave: "vigente" },
    { bolsa_ref: "bolsa:ct:5", categoria: "Conserje", total: 95, por_estado: { disponible: 60 }, estado_clave: "cerrada" },
  ];

  const resumen = calcularResumenBolsas(bolsas);
  assert.ok(resumen);
  assert.equal(resumen.total_bolsas, 4); // 4 vigentes
  assert.equal(resumen.total_aspirantes, 485); // suma total
  assert.equal(resumen.total_disponibles, 330); // suma disponibles

  // Top 3 bolsas ordenadas por total descendente
  assert.equal(resumen.top_bolsas.length, 3);
  assert.equal(resumen.top_bolsas[0].bolsa_ref, "bolsa:ct:3");
  assert.equal(resumen.top_bolsas[0].total, 210);
  assert.equal(resumen.top_bolsas[1].bolsa_ref, "bolsa:ct:1");
  assert.equal(resumen.top_bolsas[1].total, 120);
  assert.equal(resumen.top_bolsas[2].bolsa_ref, "bolsa:ct:5");
  assert.equal(resumen.top_bolsas[2].total, 95);
});

test("G21: renderizarSeccionBolsasInicio muestra fallback exacto cuando no hay datos", () => {
  const htmlNull = renderizarSeccionBolsasInicio(null, escaparHTML, (v) => String(v));
  assert.match(htmlNull, /Bolsas de trabajo: no disponible/);
  assert.match(htmlNull, /class="portal-rrhh-bolsas"/);

  const htmlVacio = renderizarSeccionBolsasInicio({ top_bolsas: null }, escaparHTML, (v) => String(v));
  assert.match(htmlVacio, /Bolsas de trabajo: no disponible/);
});

test("G21: renderizarSeccionBolsasInicio muestra tarjetas métricas, navegación a B12 y acciones ver-bolsa sin datos personales", () => {
  const resumen = {
    total_bolsas: 12,
    total_aspirantes: 840,
    total_disponibles: 520,
    top_bolsas: [
      { bolsa_ref: "bolsa:ct:operario", categoria: "Operario/a", total: 300, disponibles: 200 },
      { bolsa_ref: "bolsa:ct:admin", categoria: "Administrativo/a C1", total: 250, disponibles: 150 },
      { bolsa_ref: "bolsa:ct:tecnico", categoria: "Técnico/a A2", total: 120, disponibles: 80 },
    ],
  };

  const html = renderizarSeccionBolsasInicio(resumen, escaparHTML, (v) => String(v));

  // Cabecera y botón B12
  assert.match(html, /<h3>Bolsas de trabajo<\/h3>/);
  assert.match(html, /data-vista="resumen">Ver cuadro B12<\/button>/);

  // Tarjetas métricas
  assert.match(html, /data-metrica="bolsas_vigentes" data-vista="resumen"[\s\S]*?<strong class="metrica-valor">12<\/strong>/);
  assert.match(html, /data-metrica="total_aspirantes" data-vista="resumen"[\s\S]*?<strong class="metrica-valor">840<\/strong>/);
  assert.match(html, /data-metrica="total_disponibles" data-vista="resumen"[\s\S]*?<strong class="metrica-valor">520<\/strong>/);

  // Top 3 bolsas con botón Ver candidatos
  assert.match(html, /Operario\/a/);
  assert.match(html, /data-accion="ver-bolsa" data-bolsa-ref="bolsa:ct:operario">Ver candidatos<\/button>/);
  assert.match(html, /Administrativo\/a C1/);
  assert.match(html, /data-accion="ver-bolsa" data-bolsa-ref="bolsa:ct:admin">Ver candidatos<\/button>/);
  assert.match(html, /Técnico\/a A2/);
  assert.match(html, /data-accion="ver-bolsa" data-bolsa-ref="bolsa:ct:tecnico">Ver candidatos<\/button>/);

  // Cero datos personales (sin DNI ni nombres de personas)
  assert.doesNotMatch(html, /\d{8}[A-Z]/);
  assert.doesNotMatch(html, /\*{3}\d{4}\*{2}/);
  assert.doesNotMatch(html, /candidato_ref/);
});

test("G21: crearVistaInicioPortal integra la sección de bolsas en la portada de RRHH", () => {
  const resumenBolsas = {
    total_bolsas: 12,
    total_aspirantes: 500,
    total_disponibles: 300,
    top_bolsas: [
      { bolsa_ref: "bolsa:ct:aux", categoria: "Auxiliar", total: 200, disponibles: 120 },
    ],
  };

  const renderizar = crearVistaInicioPortal({
    encabezadoVista: () => "<header>Inicio</header>",
    escaparHTML,
    obtenerCatalogo: () => [],
    resolverAcceso: () => ({ disponible: true, vista: "bolsa" }),
    esPerfilRRHH: () => true,
    obtenerMetricasCuadro: () => ({ en_tramitacion: 10, con_incidencia: 1, en_llamamiento: 2 }),
    obtenerResumenBolsas: () => resumenBolsas,
  });

  const html = renderizar();
  assert.match(html, /class="portal-rrhh-bolsas"/);
  assert.match(html, /data-metrica="bolsas_vigentes"/);
  assert.match(html, /data-accion="ver-bolsa" data-bolsa-ref="bolsa:ct:aux"/);
});
