import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { CLAVES_SIN_ENTRADA_PORTAL, crearCoordinadorModulosPortal } from "./portal-modulos-coordinador.js";
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
    // Estas pruebas miden la carga en paralelo: todos los módulos arrancan a la
    // vez (el portal difiere los que no tienen entrada; ver pruebas propias).
    modulosDiferidos: [],
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
  // Personal, Cronos y Dietas se cargan pero no se ofrecen en menú ni Inicio.
  assert.deepEqual(coordinador.obtenerCatalogo(), CATALOGO.filter(({ clave }) => !CLAVES_SIN_ENTRADA_PORTAL.includes(clave)));
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
  // Los catálogos del alta salen primero: no retrasan el cuadro (ver abajo).
  assert.deepEqual(iniciadas, ["alta", "cuadro", "analisis"]);
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
  assert.match(portal, /if \(estado\.vista === "portal"\) \{ renderizarConservandoFoco\(\);/);
  // Al terminar la carga: Inicio conserva el foco; otra vista solo se monta si no lo estaba.
  assert.match(portal, /if \(estado\.vista === "portal"\) renderizarConservandoFoco\(\);\s*else if \(estado\.vistaMontada !== estado\.vista\) renderizar\(\);/);
  // La vista se monta por su disponibilidad, no por la clave del módulo que avisa.
  assert.match(portal, /estado\.vistaMontada !== estado\.vista\s*&& \(coordinadorModulos\.vistaDisponible\(estado\.vista\)/);
  // Selector del foco escapado y carga sustituida sin marcar error de catálogo.
  assert.match(portal, /CSS\?\.escape/);
  assert.match(portal, /error\?\.codigo === CODIGO_CARGA_SUSTITUIDA/);
  // Repintar no debe abortar la carga en curso.
  assert.doesNotMatch(portal, /coordinadorModulos\.desmontarVistaActual\(\)/);
});

test("dos cargas seguidas: los módulos tardíos de la primera no alteran la composición de la segunda", async () => {
  const catalogos = [diferido(), diferido()];
  const modulosPrimera = { cronos: diferido(), dietas: diferido() };
  let carga = 0;
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    limiteCargaModularMs: 200,
    modulosDiferidos: [],
    cargarCatalogoInterno: async () => catalogos[carga++].promesa,
    entorno: { fetch: async () => { throw new Error("sin red"); } },
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("sin CT"); },
      // La primera carga recibe módulos tardíos; la segunda, fallos inmediatos.
      cronos: () => (carga === 1 ? modulosPrimera.cronos.promesa : Promise.reject(new Error("no"))),
      dietas: () => (carga === 1 ? modulosPrimera.dietas.promesa : Promise.reject(new Error("no"))),
      personal: async () => { throw new Error("no"); },
    },
  });
  const avisos = [];
  const primera = coordinador.cargarInterno({ alCambiar: (clave) => avisos.push(`1:${clave}`) });
  catalogos[0].resolver(Object.freeze([{ clave: "cronos" }, { clave: "dietas" }]));
  await esperarTurnos();
  const segunda = coordinador.cargarInterno({ alCambiar: (clave) => avisos.push(`2:${clave}`) });
  catalogos[1].resolver(Object.freeze([{ clave: "cronos" }, { clave: "dietas" }]));
  await assert.rejects(primera, (error) => error.codigo === "carga_sustituida");
  await segunda;
  modulosPrimera.cronos.resolver(recursosCronos());
  modulosPrimera.dietas.resolver(recursosDietas());
  await esperarTurnos();
  assert.deepEqual(avisos.filter((aviso) => aviso.startsWith("1:")), ["1:catalogo"]);
  assert.equal(coordinador.resolverAcceso("cronos").disponible, false);
  assert.equal(coordinador.resolverAcceso("dietas").disponible, false);
});

test("cancelar sin consultas pendientes invalida igualmente la carga en curso", async () => {
  const { coordinador, pendientes } = coordinadorControlado();
  const avisos = [];
  const carga = coordinador.cargarInterno({ alCambiar: (clave) => avisos.push(clave) });
  await esperarTurnos();
  // Los cargadores de módulo no son consultas de red: no hay controladores.
  coordinador.desmontarVistaActual();
  pendientes.cronos.resolver(recursosCronos());
  pendientes.personal.resolver(recursosPersonal());
  pendientes.dietas.resolver(recursosDietas());
  pendientes.contratacion_temporal.rechazar(new Error("sin CT"));
  await assert.rejects(carga, (error) => error.codigo === "carga_sustituida");
  assert.deepEqual(avisos, ["catalogo"]);
  assert.equal(coordinador.resolverAcceso("cronos").disponible, false);
});

test("Inicio con contratación temporal lenta: estado neutro hasta conocer el perfil", async () => {
  const { coordinador, pendientes } = coordinadorControlado();
  const vista = crearVistaInicioPortal({
    encabezadoVista: (_s, titulo) => `<header><h2>${titulo}</h2></header>`,
    escaparHTML: String,
    obtenerCatalogo: coordinador.obtenerCatalogo,
    resolverAcceso: (clave) => coordinador.resolverAcceso(clave),
    esPerfilRRHH: coordinador.esPerfilRRHH,
    inicioPendiente: coordinador.inicioPendiente,
  });
  assert.equal(coordinador.inicioPendiente(), true, "antes de la primera carga no se conoce el perfil");
  const carga = coordinador.cargarInterno();
  await esperarTurnos();
  // Cronos, Personal y Dietas llegan; contratación temporal sigue cargando.
  pendientes.cronos.resolver(recursosCronos());
  pendientes.personal.resolver(recursosPersonal());
  pendientes.dietas.resolver(recursosDietas());
  await esperarTurnos();
  assert.equal(coordinador.inicioPendiente(), true);
  const neutro = vista();
  assert.match(neutro, /<h2>Inicio<\/h2>/);
  assert.match(neutro, /role="status" data-inicio-pendiente>Comprobando accesos…/);
  assert.doesNotMatch(neutro, /portal-inicio-empleado|portal-rrhh-inicio|Portal del Empleado<\/h2>/);
  assert.equal((neutro.match(/aria-busy="true"/g) || []).length,
    CATALOGO.filter(({ clave }) => !CLAVES_SIN_ENTRADA_PORTAL.includes(clave)).length, "todas las tarjetas ofrecidas «Comprobando»");
  assert.equal(coordinador.vistaPendiente("personal-registro"), true, "el registro RRHH espera al perfil");
  pendientes.contratacion_temporal.rechazar(new Error("sin CT"));
  await carga;
  assert.equal(coordinador.inicioPendiente(), false);
  assert.equal(coordinador.vistaPendiente("personal-registro"), false);
  assert.match(vista(), /data-inicio-sin-modulos/, "sin CT ni módulos ofrecidos: Inicio del empleado sin módulos");
  assert.doesNotMatch(vista(), /nota-seguridad|representa el acceso interno/u);
});

test("empleado sin contratación temporal en el catálogo: su Inicio en cuanto llega el catálogo", async () => {
  const pendiente = diferido();
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    modulosDiferidos: [],
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "cronos" }]),
    cargadoresInternos: { contratacion_temporal: async () => ({}), cronos: () => pendiente.promesa },
  });
  const carga = coordinador.cargarInterno();
  await esperarTurnos();
  assert.equal(coordinador.inicioPendiente(), false);
  assert.equal(coordinador.vistaPendiente("cronos"), true, "la vista pedida espera a su módulo");
  assert.equal(coordinador.vistaPendiente("dietas"), false, "módulo no autorizado: no disponible");
  pendiente.resolver(recursosCronos());
  await carga;
  assert.equal(coordinador.vistaPendiente("cronos"), false);
});

// Recorrido en Chrome del 25/09/2026 (fallo 6): los catálogos del alta de
// Contratación (~0,5 s) retrasaban Inicio. Ahora el cuadro y la configuración
// deciden el Inicio; los nombres de centro y categoría llegan después con un
// aviso, y abrir Contratación espera a los catálogos para montar el alta.
test("los catálogos del alta no retrasan Inicio y abrir Contratación los espera", async () => {
  const pendientes = { cuadro: diferido(), alta: diferido(), analisis: diferido() };
  const consulta = (nombre) => () => pendientes[nombre].promesa;
  let altaMontada = "sin montar";
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => Object.freeze([Object.freeze({ clave: "contratacion_temporal" })]),
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => ({
          obtenerCatalogosAlta: consulta("alta"), obtenerConfiguracionAnalisis: consulta("analisis"),
          registrarSolicitud: async () => ({}), registrarAnalisis: async () => ({}),
        }) },
        adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => ({ capacidades: [], listar: consulta("cuadro") }) },
        contrato: { validarCatalogosAlta: (valor) => valor, CAPACIDAD_CREAR_SOLICITUD: "contratacion_temporal.solicitud.crear" },
        presentador: { crearPresentadorExpedientesContratacionTemporal: () => ({}) },
        vista: { montarModuloContratacionTemporal: async ({ alta }) => { altaMontada = alta; return { desmontar() {} }; } },
      }),
    },
  });
  const avisos = [];
  const carga = coordinador.cargarInterno({ alCambiar: (clave) => avisos.push(clave) });
  await esperarTurnos();
  pendientes.cuadro.resolver({ expedientes: [{ expediente_ref: "e:1", numero_visible: "2026/CT-1", centro: "centro:1",
    categoria: "categoria:1", fase_clave: "solicitud", estado_clave: "en_curso", version: 1 }] });
  pendientes.analisis.resolver({ subsanacion_disponible: false, modalidades: [], categorias: [], causas: [],
    entradas_rc: [], motivos_rectificacion: [], artefacto_ref: "artefacto:1" });
  await carga;
  // Sin catálogos todavía: Contratación ya está disponible y el perfil es RRHH.
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
  assert.equal(coordinador.esPerfilRRHH(), true);
  assert.deepEqual(avisos, ["catalogo", "contratacion_temporal"]);
  const raiz = { innerHTML: "", replaceChildren() {} };
  const montaje = coordinador.montarVista("contratacion-temporal", raiz);
  await esperarTurnos();
  assert.equal(altaMontada, "sin montar", "abrir Contratación espera a los catálogos del alta");
  pendientes.alta.resolver({ centros: [{ referencia: "centro:1", etiqueta: "DEPORTES" }],
    categorias: [{ referencia: "categoria:1", etiqueta: "Auxiliar" }] });
  assert.equal(await montaje, true);
  assert.equal(altaMontada.catalogos.centros[0].etiqueta, "DEPORTES");
  // Al llegar se avisa para repintar Inicio con los nombres.
  assert.deepEqual(avisos, ["catalogo", "contratacion_temporal", "contratacion_temporal"]);
  assert.equal(coordinador.obtenerTramitesInicio()[0].centro, "DEPORTES");
});

test("sin cuadro, el perfil sigue esperando a los catálogos del alta", async () => {
  const alta = diferido();
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => Object.freeze([Object.freeze({ clave: "contratacion_temporal" })]),
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => ({
          obtenerCatalogosAlta: () => alta.promesa, obtenerConfiguracionAnalisis: async () => { throw new Error("403"); },
          registrarSolicitud: async () => ({}),
        }) },
        adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => ({ capacidades: [], listar: async () => { throw new Error("503"); } }) },
        contrato: { validarCatalogosAlta: (valor) => valor, CAPACIDAD_CREAR_SOLICITUD: "contratacion_temporal.solicitud.crear" },
        presentador: { crearPresentadorExpedientesContratacionTemporal: () => ({}) },
        vista: { montarModuloContratacionTemporal: async () => ({ desmontar() {} }) },
      }),
    },
  });
  const carga = coordinador.cargarInterno();
  await esperarTurnos();
  assert.equal(coordinador.inicioPendiente(), true, "sin cuadro ni análisis el perfil depende del alta");
  alta.resolver({ centros: [], categorias: [] });
  await carga;
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
});

// Recorrido en Chrome del 25/09/2026 (fallo 1): Personal no tiene entrada en el
// portal y aun así se descargaba su código en cada Inicio (y un fichero ausente
// del paquete daba 404). Ahora los módulos sin entrada se cargan al pedir su vista.
test("Personal, Cronos y Dietas no se cargan al arrancar; su vista directa los carga", async () => {
  const iniciados = [];
  const personal = diferido();
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => CATALOGO,
    entorno: { fetch: async () => { throw new Error("sin red"); } },
    cargadoresInternos: {
      contratacion_temporal: async () => { iniciados.push("contratacion_temporal"); throw new Error("sin CT"); },
      cronos: async () => { iniciados.push("cronos"); return recursosCronos(); },
      personal: async () => { iniciados.push("personal"); return personal.promesa; },
      personal_catalogos_publicos: async () => { iniciados.push("personal_catalogos_publicos"); throw new Error("no"); },
      dietas: async () => { iniciados.push("dietas"); return recursosDietas(); },
    },
  });
  await coordinador.cargarInterno();
  assert.deepEqual(iniciados, ["contratacion_temporal"], "solo el módulo con entrada");
  assert.equal(coordinador.resolverAcceso("personal").estado, "diferido");
  assert.equal(coordinador.vistaPendiente("personal"), true, "su URL directa dice «Comprobando», no «no disponible»");
  const carga = coordinador.prepararVista("personal");
  assert.ok(carga instanceof Promise);
  assert.equal(coordinador.prepararVista("personal"), null, "una sola carga");
  assert.equal(coordinador.prepararVista("resumen"), null, "Bolsa no se difiere");
  assert.equal(coordinador.resolverAcceso("personal").estado, "cargando");
  personal.resolver(recursosPersonal());
  await carga;
  assert.equal(coordinador.vistaDisponible("personal"), true);
  assert.deepEqual(iniciados.filter((clave) => clave !== "personal_catalogos_publicos"),
    ["contratacion_temporal", "personal"], "Cronos y Dietas siguen sin cargarse");
  assert.equal(coordinador.vistaPendiente("cronos"), true);
});

test("el shell arranca la carga diferida al pintar o al llegar el catálogo", async () => {
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  assert.match(portal, /if \(clave === "catalogo"\) coordinadorModulos\.prepararVista\(estado\.vista\);/u);
  assert.match(portal, /coordinadorModulos\.prepararVista\(estado\.vista\);\s+const pendiente =/u);
});

// Recorrido en Chrome del 25/09/2026 (fallo 5): en 1 de 4 cargas a 390 px
// Contratación no llegó a cargarse. El código de los módulos tenía un límite de
// 2 s, y por una red lenta (túnel y pasarela serializada) las importaciones se
// encolaban tras /bolsas y la sonda de borradores: el temporizador ganaba la
// carrera y el módulo quedaba «no disponible» para toda la sesión.
test("el límite de carga de un módulo admite una red lenta", async () => {
  const { LIMITE_CARGA_MODULAR_MS } = await import("./portal-modulos-carga.js");
  assert.ok(LIMITE_CARGA_MODULAR_MS >= 10_000, "no puede agotarse en una carga normal por una red lenta");
  const plazos = [];
  const reloj = {
    setTimeout: (_fn, ms) => { plazos.push(ms); return plazos.length; },
    clearTimeout: () => {},
  };
  const lento = diferido();
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    temporizadores: reloj,
    cargarCatalogoInterno: async () => Object.freeze([Object.freeze({ clave: "contratacion_temporal" })]),
    cargadoresInternos: { contratacion_temporal: () => lento.promesa },
  });
  const carga = coordinador.cargarInterno();
  await esperarTurnos();
  assert.ok(plazos.length > 0 && plazos.every((ms) => ms === LIMITE_CARGA_MODULAR_MS));
  lento.rechazar(new Error("fin"));
  await carga;
});

// Recorrido en Chrome del 25/09/2026 (fallo 4): recargar (F5) en #bolsa/resumen
// dejaba «Gestión de Bolsas no disponible · Comprobando acceso…». La vista se
// montaba antes del catálogo y la carga del catálogo la desmontaba, lo que
// cancelaba la lectura del cuadro de bolsas y nadie la volvía a pedir.
test("cargar el catálogo no desmonta una vista de Bolsa ya montada", async () => {
  const desmontajes = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "bolsa" }]),
    montajeBolsa: { disponible: () => true, montar: ({ vista }) => ({ desmontar: () => desmontajes.push(vista) }) },
  });
  const raiz = { innerHTML: "", replaceChildren() {} };
  // Como al arrancar tras F5: la carga del catálogo empieza mientras la vista
  // de Bolsa aún se está montando (montarVista espera un turno).
  const montaje = coordinador.montarVista("resumen", raiz);
  const carga = coordinador.cargarInterno();
  assert.equal(await montaje, true, "el montaje en curso no se sustituye");
  await carga;
  assert.deepEqual(desmontajes, [], "la vista de Bolsa y sus lecturas siguen vivas");
  await coordinador.cargarInterno();
  assert.deepEqual(desmontajes, []);
  // Otra vista sí se retira (depende de la composición que se recarga).
  coordinador.desmontarVistaActual();
  assert.deepEqual(desmontajes, ["resumen"]);
});

test("F5 en el cuadro de Bolsa: se pide el cuadro al montar y la carga no lo repite", async () => {
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  const montaje = portal.slice(portal.indexOf("function montarVistaBolsa("), portal.indexOf("function renderizarLlamamientoSinBolsa("));
  // Sin lectura del cuadro, la vista la pide (en vez de pintar «no disponible»).
  assert.match(montaje, /if \(vistaBolsas && estado\.datosBolsas === null\) \{\s+(?:\/\/[^\n]*\s+)*void controladorBolsas\.cargarBolsas\(\);\s+return;/u);
  assert.ok(montaje.indexOf("estado.datosBolsas === null") < montaje.indexOf("if (!estado.fuenteLista)"));
  const carga = portal.slice(portal.indexOf("async function cargarFuenteDatos()"), portal.indexOf("function necesidadLlamamientoSeleccionada()"));
  // Una vista de Bolsa montada se conserva y no se vuelve a montar.
  assert.match(carga, /if \(moduloDeVistaPortal\(estado\.vistaMontada \|\| ""\) !== "bolsa"\) estado\.vistaMontada = "";/u);
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
