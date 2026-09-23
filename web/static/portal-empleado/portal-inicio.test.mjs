import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { calcularMetricasCuadro, crearVistaInicioPortal, tramitesParaInicio } from "./portal-inicio.js";
import { crearControladorPortal } from "./portal-eventos.js";

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

test("la portada sin catálogo ofrece reintento y el clic activa la recarga existente", async () => {
  const estado = { errorFuente: "error anterior", fuenteLista: true };
  const vista = crearVistaInicioPortal({
    encabezadoVista: () => "<header>Portal</header>",
    escaparHTML,
    obtenerCatalogo: () => [],
    resolverAcceso: () => { throw new Error("sin módulos"); },
    catalogoFallido: () => estado.errorFuente !== "",
  });
  let html = vista();
  assert.match(html, /role="alert" aria-labelledby="error-catalogo-modulos-titulo"/u);
  assert.match(html, /El catálogo interno de módulos no está disponible/u);
  assert.match(html, /<button[^>]*data-accion="recargar-fuente"[^>]*>Reintentar<\/button>/u);

  const documentoAnterior = globalThis.document;
  const ventanaAnterior = globalThis.window;
  const escuchas = new Map();
  const control = { addEventListener() {} };
  let recargas = 0; let repintados = 0;
  globalThis.document = { addEventListener: (tipo, escuchar) => escuchas.set(tipo, escuchar) };
  globalThis.window = { addEventListener() {} };
  try {
    const controlador = crearControladorPortal({
      porId: () => control,
      estado,
      obtenerDatosPanel: () => ({}),
      renderizar: () => { repintados += 1; html = vista(); },
      cargarFuenteDatos: async () => { recargas += 1; },
    });
    controlador.instalar();
    const boton = { dataset: { accion: "recargar-fuente" } };
    escuchas.get("click")({ target: { closest: (selector) => selector === "[data-accion]" ? boton : null } });
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(recargas, 1);
    assert.equal(estado.errorFuente, "");
    assert.equal(estado.fuenteLista, false);
    assert.equal(repintados, 2);
    assert.doesNotMatch(html, /data-accion="recargar-fuente"/u);
  } finally {
    globalThis.document = documentoAnterior;
    globalThis.window = ventanaAnterior;
  }
});

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
  assert.match(html, /Borradores disponibles/);
  assert.match(html, /data-vista="elaboracion">Entrar<\/button>/);
  assert.doesNotMatch(html, /panel|resumen/u);
});

test("una ruta de presentación conserva su etiqueta escapada y no se anuncia como backend conectado", () => {
  const html = renderizar({
    disponible: true,
    vista: "dietas",
    estado: "presentacion",
    presentacion: true,
    etiqueta: "Recorrido visual <pendiente> de backend",
    accion_etiqueta: "Ver recorrido",
  });
  assert.match(html, /tarjeta-modulo-presentacion/);
  assert.match(html, /data-estado-conexion="recorrido-visual"/);
  assert.match(html, /estado-presentacion[^>]*>Recorrido visual &lt;pendiente&gt; de backend/);
  assert.match(html, /class="boton-secundario" data-vista="dietas">Ver recorrido<\/button>/);
  assert.doesNotMatch(html, /Disponible para el perfil activo/);
  assert.doesNotMatch(html, /<pendiente>/);
});

test("el acceso conectado mantiene el texto productivo si la composición no aporta etiqueta", () => {
  const html = renderizar({ disponible: true, vista: "elaboracion", estado: "disponible" });
  assert.match(html, /data-estado-conexion="conectado"/);
  assert.match(html, /estado-disponible[^>]*>Disponible para el perfil activo/);
  assert.match(html, /class="boton-primario" data-vista="elaboracion">Entrar<\/button>/);
  assert.doesNotMatch(html, /tarjeta-modulo-presentacion/);
});

test("la identidad visual cubre los trece módulos del catálogo y conserva salida móvil y contraste forzado", async () => {
  const catalogo = [
    "bolsa", "contratacion_temporal", "personal", "nominas", "cronos", "dietas", "solicitudes",
    "meritos", "comunicaciones", "documentos", "aprobaciones", "auditoria", "administracion",
  ].map((clave) => ({ clave, sigla: clave.slice(0, 3).toUpperCase(), titulo: clave, texto: "Datos sintéticos" }));
  const html = crearVistaInicioPortal({
    encabezadoVista: () => "",
    escaparHTML,
    obtenerCatalogo: () => catalogo,
    resolverAcceso: (clave) => ({ disponible: true, vista: clave, presentacion: true, etiqueta: "Recorrido visual · pendiente de backend", accion_etiqueta: "Ver recorrido" }),
  })();
  for (const { clave } of catalogo) {
    assert.match(html, new RegExp(`data-modulo-catalogo="${clave}"`));
  }
  const css = await readFile(new URL("./portal-modulos.css", import.meta.url), "utf8");
  for (const { clave } of catalogo) {
    assert.match(css, new RegExp(`data-modulo-catalogo="${clave}"`));
  }
  assert.match(css, /@media \(max-width: 720px\)/);
  assert.match(css, /@media \(forced-colors: active\)/);
  assert.match(css, /\.estado-presentacion/);
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

test("G10: la vista de inicio para RRHH conserva cuadro y accesos, y expone el catálogo completo", () => {
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

  assert.match(html, /Todos los módulos de Recursos Humanos/);
  assert.match(html, /class="rejilla-modulos" aria-label="Todos los módulos de Recursos Humanos"/);
  assert.match(html, /data-modulo-catalogo="bolsa"/);
});

test("el inicio de RRHH muestra los trece módulos y conserva la advertencia ámbar de presentación", () => {
  const claves = [
    "bolsa", "contratacion_temporal", "personal", "nominas", "cronos", "dietas", "solicitudes",
    "meritos", "comunicaciones", "documentos", "aprobaciones", "auditoria", "administracion",
  ];
  const html = crearVistaInicioPortal({
    encabezadoVista: () => "",
    escaparHTML,
    obtenerCatalogo: () => claves.map((clave) => ({ clave, sigla: clave.slice(0, 3), titulo: clave, texto: "Datos sintéticos" })),
    resolverAcceso: (clave) => ({ disponible: true, vista: clave, estado: "presentacion", presentacion: true, etiqueta: "Recorrido visual · pendiente de backend", accion_etiqueta: "Ver recorrido" }),
    esPerfilRRHH: () => true,
  })();
  assert.match(html, /Todos los módulos de Recursos Humanos/);
  for (const clave of claves) {
    assert.match(html, new RegExp(`data-modulo-catalogo="${clave}"`));
  }
  assert.equal((html.match(/tarjeta-modulo-presentacion/g) || []).length, claves.length);
  assert.equal((html.match(/>Ver recorrido<\/button>/g) || []).length, claves.length);
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
