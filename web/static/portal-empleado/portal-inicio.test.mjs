import assert from "node:assert/strict";
import test from "node:test";
import { calcularMetricasCuadro, crearVistaInicioPortal, tramitesParaInicio } from "./portal-inicio.js";

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
