import test from "node:test";
import assert from "node:assert/strict";
import { validarCircuitoRRHH, validarConsultaCircuitoRRHH } from "./contrato-circuito-rrhh.js";
import { crearClienteCircuitoRRHH, RUTA_CONSULTA_CIRCUITO_RRHH } from "./cliente-http-circuito-rrhh.js";
import { instalarConsultaCircuitoRRHH, marcarRailDesconocido, pasosRailCircuitoRRHH, renderizarCircuitoRRHH } from "./vista-circuito-rrhh.js";
import { insertarConsultaCircuitoRRHH } from "./vista-expedientes.js?v=20261008-alta-rpt-circular-v4";

const consulta = { expediente_ref: "expediente:prueba:rrhh", version_observada: 1 };
const flujo = { definicion_ref: "flujo:prueba:rrhh", version: 2, huella_sha256: "a".repeat(64) };
function datos() {
  return { flujo, circuito: { definicion: flujo, estado_actual: "solicitud", hitos: [] },
    version_expediente: 1, transiciones_permitidas: [] };
}
test("la consulta no admite identidad ni permisos enviados por la pantalla", () => {
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, actor_ref: "actor:ajeno" }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, perfil_ref: "perfil:direccion" }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, version_observada: 0 }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, expediente_ref: undefined }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, expediente_ref: null }));
});
test("la respuesta debe corresponder al flujo y no exponer evidencias personales", () => {
  const cambiado = datos(); cambiado.circuito.definicion = { ...flujo, version: 3 };
  assert.throws(() => validarCircuitoRRHH(cambiado, consulta));
  const personal = datos(); personal.circuito.hitos = [{ actor_ref: "persona:ajena" }];
  assert.throws(() => validarCircuitoRRHH(personal, consulta));
  assert.throws(() => validarCircuitoRRHH(datos(), { ...consulta, version_observada: 2 }));
  const adelantado = datos(); adelantado.version_expediente = 2;
  assert.throws(() => validarCircuitoRRHH(adelantado, consulta));
});
test("alta sin hitos se presenta pendiente, sin afirmar firmas ni trámites hechos", () => {
  const html = renderizarCircuitoRRHH(validarCircuitoRRHH(datos(), consulta));
  assert.match(html, /Todavía no constan actuaciones/u);
  assert.match(html, /consulte las tareas del expediente/u);
  assert.doesNotMatch(html, /✓|<li>/u);
});
test("el carril usa el estado acreditado del circuito tras alta y primer análisis", () => {
  const claves = ["circuito_solicitud", "circuito_autorizacion_rrhh", "circuito_credito", "circuito_oferta"];
  assert.deepEqual(pasosRailCircuitoRRHH(datos(), claves), ["ahora", "falta", "falta", "falta"]);
  const trasAnalisis = datos();
  trasAnalisis.circuito.estado_actual = "credito";
  trasAnalisis.circuito.hitos = [
    { origen: "solicitud", destino: "autorizacion_rrhh" },
    { origen: "autorizacion_rrhh", destino: "credito" },
  ];
  assert.deepEqual(pasosRailCircuitoRRHH(trasAnalisis, claves), ["hecho", "hecho", "ahora", "falta"]);
  trasAnalisis.circuito.estado_actual = "subsanacion_servicio";
  assert.equal(pasosRailCircuitoRRHH(trasAnalisis, claves), null);
});
test("el panel se inserta solo en fichas del flujo nuevo", () => {
  const inserciones = [];
  let consultas = 0;
  const { bloque, panel, elemento } = crearRailPrueba();
  bloque.querySelector = () => ({ click: () => { consultas += 1; } });
  const raiz = { querySelector: (selector) => selector === "[data-ct-exp-ancla-firma]"
    ? { insertAdjacentHTML: (...args) => inserciones.push(args), previousElementSibling: bloque } : null };
  assert.equal(insertarConsultaCircuitoRRHH(raiz, { expediente_ref: consulta.expediente_ref,
    version: 1, fases: [{ fase_ref: "fase:ct:solicitud" }] }), false);
  assert.equal(insertarConsultaCircuitoRRHH(raiz, { expediente_ref: consulta.expediente_ref,
    version: 1, fases: [{ fase_ref: "fase:ct:circuito_solicitud" }] }), true);
  assert.equal(inserciones.length, 1);
  assert.equal(panel.hidden, false);
  assert.equal(elemento.className, "desconocido");
  assert.equal(consultas, 1);
  assert.equal(inserciones[0][0], "beforebegin");
  assert.match(inserciones[0][1], /data-ct-circuito-expediente="expediente:prueba:rrhh"/u);
});

function crearRailPrueba() {
  const resumen = { textContent: "1 hecha · 0 actuales · 0 pendientes" };
  const panel = { hidden: true, querySelector: () => resumen };
  const nombre = { textContent: "Firma de la petición" };
  const rotulo = { textContent: "Hecho" };
  const marca = { textContent: "✓" };
  const atributosBoton = new Map();
  const botonFase = {
    querySelector: (selector) => ({ ".nombre": nombre, small: rotulo, ".marca": marca })[selector],
    setAttribute: (nombre, valor) => atributosBoton.set(nombre, valor),
  };
  const atributosElemento = new Map([["aria-current", "step"]]);
  const elemento = {
    className: "hecho", dataset: { ctExpOrden: "1" },
    querySelector: () => botonFase,
    removeAttribute: (nombre) => atributosElemento.delete(nombre),
  };
  const rail = { querySelectorAll: () => [elemento], closest: () => panel };
  const contenedor = { querySelector: () => rail };
  const resultado = { textContent: "", innerHTML: "" };
  const bloque = {
    dataset: { ctCircuitoVersion: "1", ctCircuitoExpediente: consulta.expediente_ref },
    closest: () => contenedor,
    querySelector: () => resultado,
  };
  const atributosConsulta = new Map();
  const botonConsulta = {
    closest: () => bloque,
    getAttribute: (nombre) => atributosConsulta.get(nombre),
    setAttribute: (nombre, valor) => atributosConsulta.set(nombre, valor),
    removeAttribute: (nombre) => atributosConsulta.delete(nombre),
  };
  return { bloque, panel, elemento, resumen, rotulo, marca, atributosElemento, atributosBoton,
    resultado, botonConsulta };
}

test("403, 404, 503 y fallo de red conservan las fases navegables con avance desconocido", async () => {
  for (const estado of [403, 404, 503, "red"]) {
    const prueba = crearRailPrueba();
    let manejar;
    const documento = {
      addEventListener: (_tipo, callback) => { manejar = callback; },
      removeEventListener: () => {}, contains: () => true,
    };
    const cliente = crearClienteCircuitoRRHH({ fetchImpl: async () => {
      if (estado === "red") throw new TypeError("sin conexión");
      return new Response("{}", { status: estado });
    } });
    const retirar = instalarConsultaCircuitoRRHH(documento, cliente);
    await manejar({ target: { closest: () => prueba.botonConsulta }, preventDefault: () => {} });
    assert.equal(prueba.panel.hidden, false, String(estado));
    assert.equal(prueba.elemento.className, "desconocido", String(estado));
    assert.equal(prueba.atributosElemento.has("aria-current"), false, String(estado));
    assert.equal(prueba.marca.textContent, "1", String(estado));
    assert.equal(prueba.rotulo.textContent, "Avance no disponible", String(estado));
    assert.match(prueba.resumen.textContent, /Avance no disponible/u);
    assert.match(prueba.atributosBoton.get("aria-label"), /Firma de la petición: Avance no disponible/u);
    assert.match(prueba.resultado.textContent, estado === 403 ? /No dispone de permiso/u : /No se puede mostrar/u);
    assert.equal(prueba.resultado.innerHTML, "", String(estado));
    retirar();
  }
  assert.equal(marcarRailDesconocido(null), false);
});
test("el cliente consulta por POST fijo sin identidad libre ni persistencia", async () => {
  let peticion;
  const cliente = crearClienteCircuitoRRHH({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return new Response(JSON.stringify({ data: datos() }), {
      status: 200, headers: { "Content-Type": "application/json" },
    });
  } });
  const resultado = await cliente.consultar(consulta);
  assert.equal(resultado.estado, "disponible");
  assert.equal(peticion.ruta, RUTA_CONSULTA_CIRCUITO_RRHH);
  assert.equal(peticion.opciones.method, "POST");
  assert.equal(peticion.opciones.credentials, "same-origin");
  assert.equal(peticion.opciones.cache, "no-store");
  assert.equal(peticion.opciones.redirect, "error");
  assert.equal(peticion.opciones.referrerPolicy, "no-referrer");
  assert.deepEqual(JSON.parse(peticion.opciones.body), consulta);
});
test("una denegación o una respuesta excesiva no presenta ningún hito", async () => {
  const denegado = crearClienteCircuitoRRHH({ fetchImpl: async () => new Response("{}", { status: 403 }) });
  assert.deepEqual(await denegado.consultar(consulta), { estado: "denegado" });
  const excesivo = crearClienteCircuitoRRHH({ fetchImpl: async () => new Response("x".repeat(128 * 1024 + 1), {
    status: 200, headers: { "Content-Type": "application/json" },
  }) });
  assert.deepEqual(await excesivo.consultar(consulta), { estado: "no_disponible" });
});
