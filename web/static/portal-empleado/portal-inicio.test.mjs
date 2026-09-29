import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearVistaInicioPortal, resumirBolsasInicio } from "./portal-inicio.js";
import { crearControladorPortal } from "./portal-eventos.js";
import { crearTraductorPortal, MENSAJES_INICIO_RRHH_EN, MENSAJES_PORTAL_ES } from "./portal-i18n.js";

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
    // Un solo repintado previo: la carga pinta su propio resultado, sin montar dos veces.
    assert.equal(repintados, 1);
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
  // Sin región viva por tarjeta: el shell da un único anuncio al terminar.
  assert.match(html, /<span class="estado-proximamente">Comprobando acceso a borradores/);
  assert.doesNotMatch(html, /aria-live/);
  assert.match(html, /<button[^>]+disabled>Comprobando<\/button>/);
  assert.doesNotMatch(html, /data-vista=/);
});

test("un módulo denegado o sin servicio no ocupa una tarjeta vacía", () => {
  for (const acceso of [
    { disponible: false, vista: "", estado: "denegado", etiqueta: "Sin permiso para gestionar borradores" },
    { disponible: false, vista: "", estado: "error", etiqueta: "Servicio de borradores no disponible", reintentar: true },
    { disponible: false, vista: "", estado: "no_disponible" },
    { disponible: false, vista: "" },
  ]) {
    const html = renderizar(acceso);
    assert.doesNotMatch(html, /data-modulo-catalogo=/u, acceso.estado);
    assert.doesNotMatch(html, /Sin permiso|no disponible|reintentar-borradores/u, acceso.estado);
  }
});

// E10/P2-4: sin ningún módulo que ofrecer, Inicio del empleado no queda en
// blanco: un estado vacío con texto del catálogo i18n y sin ayuda en pantalla.
test("Inicio del empleado sin módulos disponibles muestra un estado vacío i18n", () => {
  const claves = [];
  const vista = (catalogoFallido) => crearVistaInicioPortal({
    encabezadoVista: () => "<header>Portal</header>",
    escaparHTML,
    obtenerCatalogo: () => [moduloBolsa],
    resolverAcceso: () => ({ disponible: false, vista: "", estado: "denegado" }),
    traducir: (clave) => { claves.push(clave); return `«${clave}»`; },
    catalogoFallido: () => catalogoFallido,
  })();
  const html = vista(false);
  assert.match(html, /role="status" data-inicio-sin-modulos>\s*<p>«inicio_empleado_sin_modulos»<\/p>/u);
  assert.ok(claves.includes("inicio_empleado_sin_modulos"));
  assert.doesNotMatch(html, /data-modulo-catalogo=|data-accion="ayuda"|rejilla-modulos/u);
  assert.match(renderizar({ disponible: false, vista: "", estado: "denegado" }), /No hay módulos disponibles para su perfil\./u);
  // Con el catálogo caído manda su aviso con reintento, no el estado vacío.
  const fallido = vista(true);
  assert.match(fallido, /data-accion="recargar-fuente"/u);
  assert.doesNotMatch(fallido, /data-inicio-sin-modulos/u);
  // Con un módulo ofrecido no hay estado vacío.
  assert.doesNotMatch(renderizar({ disponible: true, vista: "resumen" }), /data-inicio-sin-modulos/u);
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

const fijo = () => new Date("2026-09-29T08:00:00Z");
const expedientesCuadro = [
  { expediente_ref: "exp:a", numero_visible: "2026/CT-000001", centro: "DEPORTES", categoria: "Operario/a",
    fase_clave: "solicitud", estado_clave: "en_curso", plazo: "1 oct 2026", plazo_estado: "en_plazo", plazo_ultimo_dia: "2026-10-01" },
  { expediente_ref: "exp:b", numero_visible: "2026/CT-000002", centro: "CULTURA", categoria: "Técnico/a",
    fase_clave: "subsanacion_unidad", estado_clave: "incidencia", plazo: "29 sept 2026", plazo_estado: "vence_hoy", plazo_ultimo_dia: "2026-09-29" },
  { expediente_ref: "exp:c", numero_visible: "2026/CT-000003", centro: "Centro <libre>", categoria: "Auxiliar & más",
    fase_clave: "analisis", estado_clave: "pendiente", plazo: "25 sept 2026", plazo_estado: "vencido", plazo_ultimo_dia: "2026-09-25" },
  { expediente_ref: "exp:d", numero_visible: "2026/CT-000004", centro: "CULTURA", categoria: "Técnico/a",
    fase_clave: "seguimiento", estado_clave: "completado", plazo: "—" },
];
const bolsasListas = { carga: "listo", datos: { generado_en: "2026-09-28T10:00:00Z", bolsas: [
  { bolsa_ref: "bolsa:1", categoria: "Auxiliar <A>", vigente_desde: "2026-01-01", vigente_hasta: null, llamamientos_en_curso: 2, por_estado: { disponible: 30 } },
  { bolsa_ref: "bolsa:2", categoria: "Técnica", vigente_desde: "2026-01-01", vigente_hasta: "2028-01-14", llamamientos_en_curso: 0, por_estado: { disponible: 12 } },
] } };

function portadaRRHH(opciones = {}) {
  return crearVistaInicioPortal({
    encabezadoVista: (_s, titulo) => `<header><h1>${titulo}</h1></header>`,
    escaparHTML,
    obtenerCatalogo: () => [moduloBolsa],
    resolverAcceso: (clave) => ({ disponible: true, vista: clave === "bolsa" ? "resumen" : "contratacion-temporal" }),
    esPerfilRRHH: () => true,
    obtenerCuadroInicio: () => ({ expedientes: expedientesCuadro, parcial: false, generadoEn: "2026-09-29T07:00:00Z" }),
    obtenerBolsasInicio: () => bolsasListas,
    ahora: fijo,
    ...opciones,
  })();
}

test("la portada de RRHH empieza por los expedientes que piden atención, ordenados por plazo", () => {
  const html = portadaRRHH();
  assert.match(html, /<h3 id="inicio-rrhh-pendientes-titulo">2 expedientes pendientes<\/h3>/u);
  // Vencido (25/09) antes que el que vence hoy (29/09); el que está en plazo no aparece.
  assert.match(html, /2026\/CT-000003[\s\S]*2026\/CT-000002/u);
  assert.doesNotMatch(html.split("tareas-pendientes")[1].split("</ol>")[0], /2026\/CT-000001|2026\/CT-000004/u);
  assert.match(html, /class="fecha-tarea vencido"/u);
  assert.match(html, /class="fecha-tarea hoy"/u);
  assert.match(html, /Fase 2 de 8: Análisis RRHH · Plazo vencido el 25 sept 2026/u);
  assert.match(html, /Fase 4 de 8: Fiscalización · Vence hoy · Con incidencia/u);
  assert.match(html, /data-vista="contratacion-temporal" data-ct-exp-abrir-inicio="exp:c"\s+aria-label="Abrir el expediente 2026\/CT-000003">Abrir expediente/u);
  assert.match(html, /Centro &lt;libre&gt; ·/u);
  assert.match(html, /Auxiliar &amp; más/u);
  assert.doesNotMatch(html, /<libre>|Todos los módulos|rejilla-modulos|data-accion="ayuda"/u);
});

test("los indicadores cuentan igual que la lista y llevan a ella filtrada", () => {
  const html = portadaRRHH();
  assert.match(html, /data-metrica="en_tramite" data-vista="contratacion-temporal" data-ct-exp-vista="cuadro" data-ct-exp-lista-mostrar="en_tramite">[\s\S]*?<strong class="valor-kpi">3<\/strong>/u);
  // Vence esta semana: 29/09 y 01/10 desde el 29/09; el vencido no cuenta.
  assert.match(html, /data-metrica="vencen_semana"[^>]*>[\s\S]*?<strong class="valor-kpi">2<\/strong>/u);
  assert.match(html, /data-metrica="disponibles" data-vista="resumen">[\s\S]*?<strong class="valor-kpi">42<\/strong>[\s\S]*?Personas disponibles en 2 bolsas/u);
  // Ofertas al SAE: sin cifra, explicado.
  assert.match(html, /data-metrica="sae" data-vista="ofertas-sae">[\s\S]*?Pendiente de definir con RRHH/u);
  assert.match(html, /data-ct-exp-lista-fase="analisis_rrhh"[^>]*>2\. Análisis RRHH<\/button><\/th>\s*<td class="numero">1<\/td>/u);
  assert.match(html, /data-ct-exp-lista-fase="seguimiento"[^>]*>8\. Seguimiento<\/button><\/th>\s*<td class="numero">0<\/td>/u);
  assert.match(html, /Auxiliar &lt;A&gt;[\s\S]*?Sin fecha de fin/u);
  assert.match(html, /data-vista="contratacion-temporal" data-ct-exp-vista="alta">Nueva petición de personal/u);
});

test("una consulta con más páginas avisa de que el recuento es parcial", () => {
  const html = portadaRRHH({ obtenerCuadroInicio: () => ({ expedientes: expedientesCuadro, parcial: true, generadoEn: "2026-09-29T07:00:00Z" }) });
  assert.match(html, /role="status">Recuento parcial/u);
  assert.doesNotMatch(portadaRRHH(), /Recuento parcial/u);
});

test("sin nada urgente la portada lo dice y no inventa tareas", () => {
  const html = portadaRRHH({ obtenerCuadroInicio: () => ({ expedientes: [expedientesCuadro[0]], parcial: false, generadoEn: "2026-09-29T07:00:00Z" }) });
  assert.match(html, /No hay expedientes pendientes/u);
  assert.match(html, /Ningún plazo vence hoy ni hay incidencias abiertas/u);
  assert.doesNotMatch(html, /tareas-pendientes/u);
});

test("un fallo del cuadro no se presenta como ausencia de expedientes", () => {
  const html = portadaRRHH({ obtenerCuadroInicio: () => null });
  assert.match(html, /No se pudo consultar el cuadro de expedientes/u);
  assert.doesNotMatch(html, /expedientes pendientes|data-metrica="en_tramite" data-vista/u);
});

test("Bolsa y Contratación denegadas no muestran datos retenidos", () => {
  let lecturaRetenidaConsultada = false;
  const html = portadaRRHH({
    resolverAcceso: () => ({ disponible: false, estado: "denegado", vista: "" }),
    obtenerCuadroInicio: () => ({ expedientes: [{ ...expedientesCuadro[1], categoria: "Dato privado" }], parcial: false }),
    obtenerBolsasInicio: () => { lecturaRetenidaConsultada = true; return bolsasListas; },
  });
  assert.equal(lecturaRetenidaConsultada, false, "el inicio no lee datos de Bolsa sin acceso positivo");
  assert.doesNotMatch(html, /Dato privado|Auxiliar &lt;A&gt;/u);
  assert.match(html, /Sin permiso/u);
  assert.doesNotMatch(html, /data-vista="contratacion-temporal"/u);
});

test("la vigencia de Bolsa usa la fecha de la lectura y no inventa el recuento sin instante", () => {
  const bolsa = { categoria: "Auxiliar", vigente_desde: "2026-01-01", vigente_hasta: "2027-01-01", llamamientos_en_curso: 0 };
  const lectura = { carga: "listo", datos: { generado_en: "2026-09-28T10:00:00Z", bolsas: [bolsa] } };
  assert.equal(resumirBolsasInicio(lectura).vigentes, 1);
  lectura.datos.generado_en = "2027-01-02T10:00:00Z";
  assert.equal(resumirBolsasInicio(lectura).vigentes, 0);
  lectura.datos.generado_en = "";
  assert.equal(resumirBolsasInicio(lectura).vigentes, null, "sin instante verificado no se inventa el recuento");
  // Sin «disponibles» por bolsa no se suma nada: el indicador queda en «—».
  const html = portadaRRHH({ obtenerBolsasInicio: () => lectura });
  assert.match(html, /data-metrica="disponibles">[\s\S]*?<strong class="valor-kpi">—<\/strong>/u);
});

test("las claves de la portada se traducen con el traductor común", () => {
  const traducirEN = crearTraductorPortal({ ...MENSAJES_PORTAL_ES, ...MENSAJES_INICIO_RRHH_EN });
  const variables = { actual: "2", total: "3", contexto: "Cases" };
  for (const clave of Object.keys(MENSAJES_INICIO_RRHH_EN)) {
    const esperado = MENSAJES_INICIO_RRHH_EN[clave].replace(/\{([a-z_]+)\}/gu,
      (_coincidencia, variable) => variables[variable] ?? "");
    assert.equal(traducirEN(clave, variables), esperado);
  }
  const html = portadaRRHH({ traducir: traducirEN, locale: "en-GB" });
  assert.match(html, /2 cases need attention/u);
  assert.match(html, /Stage 2 of 8: HR review · Deadline passed on 25 sept 2026/u);
  assert.match(html, /Requests by stage[\s\S]*?4\. Financial review/u);
  assert.match(html, /To be agreed with HR/u);
  assert.match(html, /Tuesday, 29 September 2026/u);
  assert.doesNotMatch(html, /expedientes pendientes|Análisis RRHH|Peticiones por fase/u);
});

test("la portada agrupa en una fila los expedientes pendientes idénticos y conserva sus números", async () => {
  const { agruparPendientes } = await import("./portal-inicio.js");
  const base = { categoria: "Auxiliar de enfermería", centro: "Residencia", fase_clave: "solicitud", plazo_estado: "vencido", plazo_ultimo_dia: "2026-09-18", plazo: "18/9/26" };
  const grupos = agruparPendientes([
    ...Array.from({ length: 15 }, (_, i) => ({ ...base, numero_visible: `CT-${i}` })),
    { ...base, centro: "Otro centro", numero_visible: "CT-99" },
  ]);
  assert.equal(grupos.length, 2);
  assert.equal(grupos[0].length, 15);
  assert.equal(grupos[1][0].numero_visible, "CT-99");
});
