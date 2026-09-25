import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { crearCoordinadorModulosPortal } from "./portal-modulos-coordinador.js";
import { traducirPortal } from "./portal-i18n.js";
import { crearVistaInicioPortal } from "./portal-inicio.js";

test("la tarjeta de un módulo que aún carga dice «Comprobando» y queda ocupada", () => {
  const renderizar = crearVistaInicioPortal({
    encabezadoVista: () => "",
    escaparHTML: String,
    obtenerCatalogo: () => [{ clave: "cronos", sigla: "CRO", titulo: "Cronos", texto: "Jornada" }],
    resolverAcceso: () => ({ disponible: false, vista: "", estado: "cargando" }),
  });
  const html = renderizar();
  assert.match(html, /data-modulo-catalogo="cronos"[^>]*aria-busy="true"/);
  const comprobando = traducirPortal("estado_modulo_comprobando");
  assert.equal(html.split(comprobando).length - 1, 2, "estado y botón dicen «Comprobando»");
  assert.ok(!html.includes(traducirPortal("estado_modulo_no_habilitado")));
});

// Arranque del portal: el catálogo se publica en cuanto llega y cada módulo se
// carga en paralelo; uno lento o fallido no retrasa ni oculta a los demás.

function diferido() {
  let resolver;
  let rechazar;
  const promesa = new Promise((si, no) => { resolver = si; rechazar = no; });
  return { promesa, resolver, rechazar };
}

const esperarTurnos = async (turnos = 5) => {
  for (let i = 0; i < turnos; i += 1) await new Promise((seguir) => setImmediate(seguir));
};

const CATALOGO = Object.freeze([
  Object.freeze({ clave: "contratacion_temporal" }),
  Object.freeze({ clave: "cronos" }),
  Object.freeze({ clave: "personal" }),
  Object.freeze({ clave: "dietas" }),
]);

const recursosCronos = () => ({
  saldo: { montarVistaSaldoCronos() {} },
  remoto: { montarVistaRemotoCronos() {} },
  movimientos: { montarVistaMovimientosCronos() {} },
  movimientosPropios: { montarMovimientosPropiosCronos() {} },
  permisosPropios: { montarPermisosPropiosCronos() {} },
  clienteSaldo: { crearClienteSaldoCronosHTTP: () => ({}) },
  clienteRemoto: { crearClienteRemotoCronosHTTP: () => ({}) },
  clienteSolicitudes: { crearClienteSolicitudesCronosHTTP: () => ({}) },
  i18n: { crearTraductorCronos: () => (clave) => clave },
});
const recursosPersonal = () => ({
  contrato: { CAPACIDAD_CONSULTAR_PUESTO: "personal.puesto.read" },
  cliente: { crearClienteHTTPCategoriasPersonal: () => ({}) },
  vista: { montarModuloPersonal: async () => ({ desmontar() {} }) },
});
const recursosDietas = () => ({
  contrato: {},
  clienteBorradores: { crearClienteBorradoresDietasHTTP: () => ({}) },
  clienteAsignacion: { crearClienteAsignacionDietasHTTP: () => ({}) },
  calculador: { crearCalculadorRutasDietasHTTP: () => ({}) },
  mapa: { crearVisorRutaDietas: () => ({}) },
  recorridos: { montarVistaRecorridosDietas() {} },
});

/** Cada cargador espera a que la prueba lo libere: se controla el orden de llegada. */
function coordinadorControlado({ limite = 10_000 } = {}) {
  const pendientes = {
    contratacion_temporal: diferido(), cronos: diferido(), personal: diferido(), dietas: diferido(),
  };
  const iniciados = [];
  const cargador = (clave) => async () => { iniciados.push(clave); return pendientes[clave].promesa; };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    limiteCargaModularMs: limite,
    cargarCatalogoInterno: async () => CATALOGO,
    entorno: { fetch: async () => { throw new Error("sin red"); } },
    cargadoresInternos: {
      contratacion_temporal: cargador("contratacion_temporal"),
      cronos: cargador("cronos"),
      personal: cargador("personal"),
      dietas: cargador("dietas"),
    },
  });
  return { coordinador, pendientes, iniciados };
}

test("los módulos autorizados empiezan a cargar a la vez, sin esperar unos a otros", async () => {
  const { coordinador, pendientes, iniciados } = coordinadorControlado();
  const carga = coordinador.cargarInterno();
  await esperarTurnos();
  // Ningún módulo ha terminado y los cuatro ya se están pidiendo.
  assert.deepEqual([...iniciados].sort(), ["contratacion_temporal", "cronos", "dietas", "personal"]);
  pendientes.cronos.resolver(recursosCronos());
  pendientes.personal.resolver(recursosPersonal());
  pendientes.dietas.resolver(recursosDietas());
  pendientes.contratacion_temporal.rechazar(new Error("sin CT"));
  await carga;
});

test("el catálogo se publica al llegar con los módulos en «cargando» y cada uno se actualiza al terminar", async () => {
  const { coordinador, pendientes } = coordinadorControlado();
  const avisos = [];
  const carga = coordinador.cargarInterno({
    alCambiar: (clave) => avisos.push([clave, coordinador.resolverAcceso(clave === "catalogo" ? "cronos" : clave).estado
      ?? (coordinador.resolverAcceso(clave).disponible ? "disponible" : "")]),
  });
  await esperarTurnos();
  assert.deepEqual(avisos, [["catalogo", "cargando"]]);
  assert.equal(coordinador.obtenerCatalogo(), CATALOGO);
  for (const clave of ["contratacion_temporal", "cronos", "personal", "dietas"]) {
    assert.equal(coordinador.resolverAcceso(clave).estado, "cargando", clave);
    assert.equal(coordinador.resolverAcceso(clave).disponible, false, clave);
  }

  // Llegan en orden distinto al de declaración: cada aviso refleja su llegada.
  pendientes.dietas.resolver(recursosDietas());
  await esperarTurnos();
  assert.equal(coordinador.resolverAcceso("dietas").disponible, true);
  assert.equal(coordinador.resolverAcceso("cronos").estado, "cargando");
  pendientes.cronos.resolver(recursosCronos());
  await esperarTurnos();
  pendientes.personal.rechazar(new Error("falla"));
  await esperarTurnos();
  pendientes.contratacion_temporal.rechazar(new Error("falla"));
  await carga;
  assert.deepEqual(avisos.map(([clave]) => clave), ["catalogo", "dietas", "cronos", "personal", "contratacion_temporal"]);
  assert.equal(coordinador.resolverAcceso("cronos").disponible, true);
  assert.equal(coordinador.resolverAcceso("personal").estado, "no_disponible");
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").estado, "no_disponible");
});

test("un módulo que no responde agota su límite sin retrasar a los demás", async () => {
  const { coordinador, pendientes } = coordinadorControlado({ limite: 40 });
  const inicio = Date.now();
  const listos = new Map();
  const carga = coordinador.cargarInterno({ alCambiar: (clave) => listos.set(clave, Date.now() - inicio) });
  await esperarTurnos();
  pendientes.cronos.resolver(recursosCronos());
  pendientes.personal.resolver(recursosPersonal());
  pendientes.contratacion_temporal.rechazar(new Error("sin CT"));
  // Dietas no responde nunca.
  await esperarTurnos();
  assert.equal(coordinador.resolverAcceso("cronos").disponible, true);
  assert.equal(coordinador.resolverAcceso("personal").disponible, true);
  assert.equal(coordinador.resolverAcceso("dietas").estado, "cargando");
  assert.ok(listos.has("cronos") && !listos.has("dietas"), "cronos no espera al límite de dietas");
  await carga;
  assert.equal(coordinador.resolverAcceso("dietas").estado, "no_disponible");
  assert.equal(coordinador.resolverAcceso("cronos").disponible, true);
  assert.ok(listos.get("dietas") >= listos.get("cronos"));
});

test("las tres consultas iniciales de contratación temporal se piden a la vez", async () => {
  const iniciadas = [];
  const pendientes = { cuadro: diferido(), alta: diferido(), analisis: diferido() };
  const consulta = (nombre) => () => { iniciadas.push(nombre); return pendientes[nombre].promesa; };
  const cliente = {
    obtenerCatalogosAlta: consulta("alta"),
    obtenerConfiguracionAnalisis: consulta("analisis"),
    registrarSolicitud: async () => ({}),
  };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => Object.freeze([Object.freeze({ clave: "contratacion_temporal" })]),
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => cliente },
        adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => ({ capacidades: [], listar: consulta("cuadro") }) },
        contrato: { validarCatalogosAlta: (valor) => valor, CAPACIDAD_CREAR_SOLICITUD: "contratacion_temporal.solicitud.crear" },
        presentador: { crearPresentadorExpedientesContratacionTemporal: () => ({}) },
        vista: { montarModuloContratacionTemporal: async () => ({ desmontar() {} }) },
      }),
    },
  });
  const carga = coordinador.cargarInterno();
  await esperarTurnos();
  assert.deepEqual(iniciadas, ["cuadro", "alta", "analisis"]);
  pendientes.analisis.rechazar(new Error("503"));
  pendientes.alta.resolver({ centros: [], categorias: [] });
  pendientes.cuadro.resolver({ expedientes: [] });
  await carga;
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
});

test("cambiar de vista o repintar Inicio durante la carga no cancela los módulos pendientes", async () => {
  const { coordinador, pendientes } = coordinadorControlado();
  const carga = coordinador.cargarInterno();
  await esperarTurnos();
  pendientes.cronos.resolver(recursosCronos());
  await esperarTurnos();
  coordinador.retirarVistaMontada();
  pendientes.dietas.resolver(recursosDietas());
  pendientes.personal.resolver(recursosPersonal());
  pendientes.contratacion_temporal.rechazar(new Error("sin CT"));
  await carga;
  assert.equal(coordinador.resolverAcceso("dietas").disponible, true);
  assert.equal(coordinador.resolverAcceso("personal").disponible, true);
});

test("el shell repinta Inicio con cada módulo y no vuelve a montar una vista ya montada", async () => {
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  assert.match(portal, /coordinadorModulos\.cargarInterno\(\{ alCambiar: alCambiarModulos \}\)/);
  assert.match(portal, /if \(estado\.vista === "portal"\) \{ renderizarConservandoFoco\(\); return; \}/);
  assert.match(portal, /if \(estado\.vistaMontada !== estado\.vista\) renderizar\(\);/);
  // Repintar no debe abortar la carga en curso.
  assert.doesNotMatch(portal, /coordinadorModulos\.desmontarVistaActual\(\)/);
});

// --- Grafo de módulos: sin URL duplicadas y precarga exacta del grafo estático ---

const raizWeb = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const EXP_ESTATICO = /(?:^|[;\n}])\s*(?:import|export)\s[^'"]*?from\s*["']([^"']+)["']|(?:^|[;\n])\s*import\s*["']([^"']+)["']/g;
const EXP_DINAMICO = /import\(\s*["']([^"']+)["']\s*\)/g;

async function recorrerGrafo(entrada, { dinamicos }) {
  const inicio = path.join(raizWeb, entrada);
  const vistos = new Set([inicio]);
  const cola = [inicio];
  const urls = new Set();
  while (cola.length > 0) {
    const fichero = cola.shift();
    const codigo = await readFile(fichero, "utf8");
    const especificadores = [...codigo.matchAll(EXP_ESTATICO)].map((m) => m[1] || m[2]);
    if (dinamicos) especificadores.push(...[...codigo.matchAll(EXP_DINAMICO)].map((m) => m[1]));
    for (const especificador of especificadores) {
      if (!especificador.startsWith(".")) continue;
      const [ruta, consulta] = especificador.split("?");
      const absoluta = path.resolve(path.dirname(fichero), ruta);
      urls.add(`/${path.relative(raizWeb, absoluta).split(path.sep).join("/")}${consulta ? `?${consulta}` : ""}`);
      if (!vistos.has(absoluta)) { vistos.add(absoluta); cola.push(absoluta); }
    }
  }
  return urls;
}

test("ningún módulo del portal se pide con dos URL distintas (una sola descarga y una sola instancia)", async () => {
  const urls = await recorrerGrafo("portal-empleado/portal.js", { dinamicos: true });
  const porFichero = new Map();
  for (const url of urls) {
    const fichero = url.split("?")[0];
    porFichero.set(fichero, [...(porFichero.get(fichero) || []), url]);
  }
  const duplicados = [...porFichero.values()].filter((lista) => lista.length > 1);
  assert.deepEqual(duplicados, []);
});

test("index.html precarga exactamente el grafo estático de portal.js", async () => {
  const html = await readFile(new URL("index.html", import.meta.url), "utf8");
  const precargas = [...html.matchAll(/<link rel="modulepreload" href="([^"]+)">/g)].map((m) => m[1]);
  assert.equal(new Set(precargas).size, precargas.length, "sin precargas repetidas");
  const estatico = await recorrerGrafo("portal-empleado/portal.js", { dinamicos: false });
  const sobrantes = precargas.filter((url) => !estatico.has(url));
  const faltantes = [...estatico].filter((url) => !precargas.includes(url));
  // Una precarga con otra ?v= descargaría dos veces el mismo módulo.
  assert.deepEqual(sobrantes, [], "precargas que el código ya no importa con esa URL");
  assert.deepEqual(faltantes, [], `añadir a index.html: ${faltantes.map((u) => `<link rel="modulepreload" href="${u}">`).join(" ")}`);
  // La cadena que carga catálogo y módulos va primero.
  assert.match(precargas[0], /^\/portal-empleado\/portal-modulos-coordinador\.js\?v=/);
  const entrada = html.indexOf('<script type="module" src="/portal-empleado/portal.js?v=');
  assert.ok(entrada > html.lastIndexOf('rel="modulepreload"'), "las precargas preceden a la entrada");
});
