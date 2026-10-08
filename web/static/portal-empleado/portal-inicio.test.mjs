import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearVistaInicioPortal, resumirBolsasInicio } from "./portal-inicio.js?v=20261008-alta-rpt-circular-v6";
import { leerCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js";
import { crearControladorPortal } from "./portal-eventos.js?v=20261001-ct-a-i18n-v1";
import { cargarMensajesPortal, crearTraductorPortal, MENSAJES_PORTAL } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";

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

test("Inicio ofrece Mi espacio al empleado y a RRHH sin consultar los destinos", () => {
  const anterior = globalThis.fetch;
  globalThis.fetch = () => assert.fail("Inicio no debe consultar datos propios");
  try {
    for (const esPerfilRRHH of [() => false, () => true]) {
      const vista = crearVistaInicioPortal({
        encabezadoVista: () => "<header>Portal</header>", escaparHTML, esPerfilRRHH,
        obtenerCatalogo: () => [], resolverAcceso: () => ({ disponible: false }),
        obtenerAccesosEmpleado: () => ({ personal: { estado: "diferido" }, cronos: { estado: "diferido" }, dietas: { estado: "diferido" } }),
      });
      const html = vista();
      for (const destino of ["personal", "cronos", "dietas"]) assert.match(html, new RegExp(`href="#${destino}" data-vista="${destino}"`));
      assert.doesNotMatch(html, /data-inicio-sin-modulos|cronos-bandeja|personal-registro|datos-presentacion/);
    }
  } finally { globalThis.fetch = anterior; }
});

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
  assert.match(html, /No se han podido cargar las áreas del portal/u);
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

test("Bolsa permite abrir el cuadro para comprobarlo sin afirmar que ya está disponible", () => {
  const html = renderizar({ disponible: true, vista: "resumen", estado: "cargando" });
  assert.match(html, /data-modulo-catalogo="bolsa"[^>]*aria-busy="true"[^>]*data-estado-conexion="comprobando"/u);
  assert.match(html, /<span class="estado-proximamente">Comprobando<\/span>/u);
  assert.match(html, /<button[^>]+data-vista="resumen"/u);
  assert.doesNotMatch(html, /<span class="estado-disponible">/u);
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
  assert.match(renderizar({ disponible: false, vista: "", estado: "denegado" }), /No hay áreas disponibles para su perfil\./u);
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
// Recuentos que el servidor calcula sobre todo el cuadro (CT-000184).
const resumenServidor = Object.freeze({
  en_tramite: 3, con_incidencia: 1, vencidos: 1, vencen_hoy: 1, vencen_semana: 2, sin_calcular: 0,
  por_fase: Object.freeze({ solicitud: 1, subsanacion_unidad: 1, analisis: 1 }),
});
const cuadroInicio = (resumen = resumenServidor) => ({ resumen, generadoEn: "2026-09-29T07:00:00Z" });
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
    obtenerCuadroInicio: () => cuadroInicio(),
    obtenerBolsasInicio: () => bolsasListas,
    ahora: fijo,
    ...opciones,
  })();
}

test("la portada de RRHH enlaza solo el recuento cuyo filtro V1 conserva todo el conjunto", () => {
  const html = portadaRRHH();
  assert.match(html, /<h3 id="inicio-rrhh-pendientes-titulo">Lo pendiente<\/h3>/u);
  assert.match(html, /<div class="tarjeta-kpi kpi--peligro" data-metrica="vencidos">[\s\S]*?<strong class="valor-kpi">1<\/strong>[\s\S]*?Con el plazo vencido/u);
  assert.match(html, /<div class="tarjeta-kpi kpi--advertencia" data-metrica="vencen_hoy">[\s\S]*?<strong class="valor-kpi">1<\/strong>[\s\S]*?Vencen hoy/u);
  assert.match(html, /data-metrica="incidencias"[^>]*data-ct-exp-lista-mostrar="incidencia" aria-label="1 con una incidencia abierta: ver la lista">[\s\S]*?<strong class="valor-kpi">1<\/strong>[\s\S]*?Con una incidencia abierta/u);
  assert.doesNotMatch(html, /data-ct-exp-lista-mostrar="(?:vencidos|vence_hoy|vencen_semana|en_tramite|sin_plazo)"|data-ct-exp-lista-fase=/u);
  // La portada no descarga ni muestra expedientes sueltos.
  assert.doesNotMatch(html, /tareas-pendientes|data-ct-exp-abrir-inicio|Recuento parcial/u);
  assert.doesNotMatch(html, /Todos los módulos|rejilla-modulos|data-accion="ayuda"/u);
});

test("los indicadores conservan los recuentos del servidor sin enlaces a predicados parciales", () => {
  const html = portadaRRHH();
  assert.match(html, /<div class="tarjeta-kpi" data-metrica="en_tramite">[\s\S]*?<strong class="valor-kpi">3<\/strong>/u);
  assert.match(html, /data-metrica="vencen_semana"[^>]*>[\s\S]*?<strong class="valor-kpi">2<\/strong>/u);
  assert.match(html, /data-metrica="disponibles" data-vista="resumen">[\s\S]*?<strong class="valor-kpi">42<\/strong>[\s\S]*?Personas disponibles en 2 bolsas/u);
  assert.doesNotMatch(html, /data-metrica="sae"|data-vista="ofertas-sae"|Pendiente de definir con RRHH/u);
  // El reparto pasa de la fase del servidor a las ocho fases de RRHH.
  assert.match(html, /<th scope="row">2\. Análisis RRHH<\/th>\s*<td class="numero">1<\/td>/u);
  assert.match(html, /<th scope="row">4\. [^<]*<\/th>\s*<td class="numero">1<\/td>/u);
  assert.match(html, /<th scope="row">8\. Seguimiento<\/th>\s*<td class="numero">0<\/td>/u);
  assert.match(html, /Auxiliar &lt;A&gt;[\s\S]*?Sin fecha de fin/u);
  assert.match(html, /data-vista="contratacion-temporal" data-ct-exp-vista="alta">Nueva petición de personal/u);
});

test("si algún plazo no se pudo calcular la portada lo dice sin prometer un filtro ausente", () => {
  const html = portadaRRHH({ obtenerCuadroInicio: () => cuadroInicio({ ...resumenServidor, sin_calcular: 2 }) });
  assert.match(html, /role="status">En 2 peticiones no se ha podido calcular el plazo\.<\/p>/u);
  assert.doesNotMatch(html, /data-ct-exp-lista-mostrar="sin_plazo"|Ver cuáles/u);
  assert.match(portadaRRHH({ obtenerCuadroInicio: () => cuadroInicio({ ...resumenServidor, sin_calcular: 1 }) }), /En 1 petición no se ha podido calcular el plazo\./u);
  assert.doesNotMatch(portadaRRHH(), /no se ha podido calcular el plazo/u);
  // Sin nada pendiente pero con plazos sin calcular, no se dice que no vence nada.
  const cero = { ...resumenServidor, vencidos: 0, vencen_hoy: 0, con_incidencia: 0, sin_calcular: 1 };
  assert.doesNotMatch(portadaRRHH({ obtenerCuadroInicio: () => cuadroInicio(cero) }), /Ningún plazo vence hoy/u);
});

test("sin nada urgente la portada lo dice y no inventa tareas", () => {
  const html = portadaRRHH({ obtenerCuadroInicio: () => cuadroInicio({ ...resumenServidor, vencidos: 0, vencen_hoy: 0, con_incidencia: 0 }) });
  assert.match(html, /Ningún plazo vence hoy ni hay incidencias abiertas/u);
  assert.doesNotMatch(html, /data-metrica="vencidos"/u);
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
    obtenerCuadroInicio: () => cuadroInicio({ ...resumenServidor, por_fase: { solicitud: 3 } }),
    obtenerBolsasInicio: () => { lecturaRetenidaConsultada = true; return bolsasListas; },
  });
  assert.equal(lecturaRetenidaConsultada, false, "el inicio no lee datos de Bolsa sin acceso positivo");
  assert.doesNotMatch(html, /Auxiliar &lt;A&gt;|<strong class="valor-kpi">3<\/strong>|Lo pendiente/u);
  assert.match(html, /Sin permiso/u);
  assert.doesNotMatch(html, /data-vista="contratacion-temporal"/u);
});

test("Inicio RRHH explica la caída de CT anunciado y permite reintentar sin abrir su módulo", () => {
  const accesoCT = () => ({ disponible: false, estado: "no_disponible", vista: "" });
  const conCT = portadaRRHH({
    obtenerCatalogo: () => [moduloBolsa, { clave: "contratacion_temporal" }],
    resolverAcceso: (clave) => clave === "contratacion_temporal" ? accesoCT() : { disponible: true, vista: "resumen" },
  });
  assert.match(conCT, /id="inicio-ct-no-disponible"/u);
  assert.match(conCT, /La gestión de peticiones de personal temporal no está disponible ahora/u);
  assert.match(conCT, /data-accion="recargar-fuente">Reintentar<\/button>/u);
  assert.doesNotMatch(conCT, /data-vista="contratacion-temporal"/u);
  assert.ok(conCT.indexOf("inicio-ct-no-disponible") < conCT.indexOf("rejilla-kpi"));
  const sinCT = portadaRRHH({ resolverAcceso: (clave) => clave === "contratacion_temporal" ? accesoCT() : { disponible: true, vista: "resumen" } });
  assert.doesNotMatch(sinCT, /inicio-ct-no-disponible/u);
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

test("Inicio enlaza cada bolsa del GET a su lista exacta sin enlazar totales globales", () => {
  const html = portadaRRHH();
  assert.match(html, /href="\?bolsa_ref=bolsa%3A1#bolsa\/bolsa-candidatos" data-accion="ver-bolsa" data-bolsa-ref="bolsa:1"/u);
  assert.match(html, /href="\?bolsa_ref=bolsa%3A2#bolsa\/bolsa-candidatos" data-accion="ver-bolsa" data-bolsa-ref="bolsa:2"/u);
  assert.equal((html.match(/data-accion="ver-bolsa"/gu) ?? []).length, 4, "nombre y disponibles enlazan las dos bolsas leídas");
  assert.match(html, /<td class="numero"><a class="enlace-tabla" href="\?bolsa_ref=bolsa%3A1&amp;estado=disponible#bolsa\/bolsa-candidatos" data-accion="ver-bolsa" data-bolsa-ref="bolsa:1" data-estado="disponible" aria-label="Ver 30 candidatos disponibles de Auxiliar &lt;A&gt;">30<\/a><\/td>/u);
  const destino = html.match(/href="(\?bolsa_ref=bolsa%3A1&amp;estado=disponible#bolsa\/bolsa-candidatos)"/u)?.[1].replaceAll("&amp;", "&");
  assert.deepEqual(leerCandidatosBolsaCompartible(new URL(destino, "https://vec.example/portal-empleado/").search, bolsasListas.datos.bolsas),
    { bolsaRef: "bolsa:1", estado: "disponible" });
});

test("Inicio enlaza cero disponibles y deja sin acción el dato ausente o la referencia inválida", () => {
  const bolsas = [
    { ...bolsasListas.datos.bolsas[0], por_estado: { disponible: 0 } },
    { ...bolsasListas.datos.bolsas[1], por_estado: {} },
    { ...bolsasListas.datos.bolsas[1], bolsa_ref: "bolsa/ajena", categoria: "Sin referencia válida", por_estado: { disponible: 4 } },
  ];
  const previo = globalThis.fetch;
  globalThis.fetch = () => assert.fail("pintar Inicio no consulta candidatos");
  try {
    const html = portadaRRHH({ obtenerBolsasInicio: () => ({ carga: "listo", datos: { ...bolsasListas.datos, bolsas } }) });
    assert.match(html, /<td class="numero"><a class="enlace-tabla" href="\?bolsa_ref=bolsa%3A1&amp;estado=disponible#bolsa\/bolsa-candidatos"[^>]*data-estado="disponible"[^>]*>0<\/a><\/td>/u);
    assert.match(html, /<th scope="row"><a[^>]*>Técnica<\/a><\/th>\s*<td class="numero">—<\/td>/u);
    assert.match(html, /<th scope="row">Sin referencia válida<\/th>\s*<td class="numero">4<\/td>/u);
    assert.equal((html.match(/data-estado="disponible"/gu) ?? []).length, 1);
  } finally { globalThis.fetch = previo; }
});

test("las claves de la portada se traducen con el traductor común", async () => {
  const ingles = await cargarMensajesPortal("en");
  const traducirEN = crearTraductorPortal(ingles);
  const variables = { actual: "2", total: "3", contexto: "Cases" };
  for (const clave of Object.keys(MENSAJES_PORTAL).filter((nombre) => nombre.startsWith("inicio_rrhh_"))) {
    const esperado = ingles[clave].replace(/\{([a-z_]+)\}/gu,
      (_coincidencia, variable) => variables[variable] ?? "");
    assert.equal(traducirEN(clave, variables), esperado);
  }
  const html = portadaRRHH({ traducir: traducirEN, locale: "en-GB" });
  assert.match(html, /<h3 id="inicio-rrhh-pendientes-titulo">Needs attention<\/h3>/u);
  assert.match(html, /Deadline passed/u);
  assert.match(html, /Requests by stage[\s\S]*?4\. Financial review/u);
  assert.doesNotMatch(html, /To be agreed with HR|data-vista="ofertas-sae"/u);
  assert.match(html, /Tuesday, 29 September 2026/u);
  assert.match(html, /aria-label="View 30 candidates in Auxiliar &lt;A&gt; with status: available"/u);
  assert.doesNotMatch(html, /Lo pendiente|Análisis RRHH|Peticiones por fase/u);
});
