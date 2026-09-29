import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextosCorreos, crearClienteCorreos, crearSuperficieCorreos, direccionCorreoAdmisible,
  ErrorCorreos, normalizarCodigoCorreo } from "./correos-propios.js";

const RUTA = "/api/vec/usuarios/mis-correos";
const A = "correo:" + "a".repeat(32);
const B = "correo:" + "b".repeat(32);
const respuesta = (json, status = 200) => new Response(JSON.stringify(json), { status, headers: { "Content-Type": "application/json" } });
const textos = await cargarTextosCorreos({ idioma: "es" });
const aleatorio = { randomUUID: () => "123e4567-e89b-12d3-a456-426614174000" };

function correo(ref, direccion, estado, extra = {}) {
  return { correo_ref: ref, direccion, estado, activo: false, creado_utc: "2026-09-29T10:00:00.123456+00:00", ...extra };
}

/** Contenedor mínimo: suficiente para instalar, repintar y enviar formularios. */
function contenedorPrueba() {
  const manejadores = {};
  const raiz = { innerHTML: "", querySelector: () => null, contains: () => true };
  return { raiz, manejadores, addEventListener: (tipo, f) => { manejadores[tipo] = f; }, removeEventListener: () => {},
    querySelector: (selector) => (selector === "[data-correos-raiz]" ? raiz : null) };
}

test("el cliente usa la ruta exacta, sin cookies de otro origen, y valida la lista", async () => {
  const peticiones = [];
  const cliente = crearClienteCorreos({ ruta: RUTA, fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    return respuesta({ data: { version: 2, correos: [correo(A, "ana.reyes@example.org", "verificado", { activo: true, verificado_utc: "2026-09-29T10:01:00Z" })] } });
  } });
  const lista = await cliente.consultar();
  assert.equal(lista.version, 2);
  assert.equal(lista.correos[0].activo, true);
  assert.equal(peticiones[0].ruta, RUTA);
  assert.equal(peticiones[0].opciones.method, "GET");
  assert.equal(peticiones[0].opciones.credentials, "same-origin");
  assert.equal(peticiones[0].opciones.body, undefined);
  for (const mala of [
    { version: 1, correos: [correo(A, "a@example.org", "pendiente", { activo: true })] },
    { version: 1, correos: [correo("correo:uno", "a@example.org", "pendiente")] },
    { version: 1, correos: [correo(A, "a@example.org", "retirado")] },
    { version: -1, correos: [] },
  ]) {
    const otro = crearClienteCorreos({ ruta: RUTA, fetchImpl: async () => respuesta({ data: mala }) });
    await assert.rejects(otro.consultar(), TypeError);
  }
  assert.throws(() => crearClienteCorreos({ ruta: "/api/vec/usuarios/otra", fetchImpl: async () => {} }), TypeError);
});

test("POST envía solo el cuerpo de la operación y traduce los errores sin datos", async () => {
  let cuerpo;
  const cliente = crearClienteCorreos({ ruta: RUTA, fetchImpl: async (_ruta, opciones) => {
    cuerpo = JSON.parse(opciones.body);
    return respuesta({ data: { recibo_ref: "correo_recibo:" + "1".repeat(32), accion: "vec.correos.anadir", correo_ref: A, version: 1, fecha_utc: "2026-09-29T10:00:00Z", replay: false, envio: "aceptado" } }, 201);
  } });
  const recibo = await cliente.operar({ operacion: "anadir", version_esperada: 0, clave_operacion: "web-correo-1234567890ab", direccion: "a@example.org" });
  assert.deepEqual(Object.keys(cuerpo).sort(), ["clave_operacion", "direccion", "operacion", "version_esperada"]);
  assert.equal(recibo.envio, "aceptado");
  const erroneo = crearClienteCorreos({ ruta: RUTA, fetchImpl: async () => respuesta({ error: { codigo: "codigo_incorrecto", clave_i18n: "api.usuarios.correos.error.codigo_incorrecto", intentos_restantes: 2 } }, 422) });
  await assert.rejects(erroneo.operar({ operacion: "verificar", version_esperada: 1, clave_operacion: "web-correo-1234567890ab", correo_ref: A, codigo: "12345678" }),
    (error) => error instanceof ErrorCorreos && error.estado === 422 && error.codigo === "codigo_incorrecto" && error.intentosRestantes === 2);
  const ajeno = crearClienteCorreos({ ruta: RUTA, fetchImpl: async () => respuesta({ error: { codigo: "inventado", clave_i18n: "x" } }, 409) });
  await assert.rejects(ajeno.operar({ operacion: "activar", version_esperada: 1, clave_operacion: "web-correo-1234567890ab", correo_ref: A }),
    (error) => error.codigo === "" && error.estado === 409);
  await assert.rejects(cliente.operar({ operacion: "borrar", version_esperada: 0, clave_operacion: "web-correo-1234567890ab" }), TypeError);
});

test("validaciones locales de dirección y código", () => {
  assert.equal(direccionCorreoAdmisible("ana.reyes@dipgra.es"), true);
  for (const mala of ["ana", "ana@", "Ana <a@x.es>", "a b@x.es", "a@x"]) assert.equal(direccionCorreoAdmisible(mala), false, mala);
  assert.equal(normalizarCodigoCorreo("1234 5678"), "12345678");
  assert.equal(normalizarCodigoCorreo("1234-5678"), "12345678");
  assert.equal(normalizarCodigoCorreo("1234567"), "");
  assert.equal(normalizarCodigoCorreo("abcd5678"), "");
});

test("la vista presenta estados en palabras, acciones por estado y ningún dato interno", async () => {
  const lista = { version: 3, correos: Object.freeze([
    Object.freeze(correo(A, "ana.reyes@example.org", "verificado", { activo: true })),
    Object.freeze(correo(B, "ana.personal@example.org", "pendiente", { codigo: { vence_utc: "2026-09-29T11:00:00Z", intentos_restantes: 3 } })),
  ]) };
  const vista = crearSuperficieCorreos({ cliente: { consultar: async () => lista }, textos, aleatorio });
  assert.match(vista.renderizar(), /Consultando sus direcciones/u);
  await vista.cargar();
  const html = vista.renderizar();
  assert.match(html, /Recibe los avisos/u);
  assert.match(html, /Sin confirmar/u);
  assert.match(html, /Le quedan 3 intentos/u);
  assert.match(html, /autocomplete="one-time-code"/u);
  assert.match(html, /Enviar otro código/u);
  assert.doesNotMatch(html, /per_|recibo|V3|HTTP|correo_ref/u);
  // La dirección activa no ofrece «Quitar»: solo la pendiente.
  assert.equal((html.match(/data-correos-accion="pedir-quitar"/gu) || []).length, 1);
  assert.equal((html.match(/data-correos-accion="activar"/gu) || []).length, 0);
});

test("alta: envía la dirección, recarga y anuncia el código sin repetir la operación", async () => {
  const operaciones = [];
  let lista = { version: 0, correos: [] };
  const cliente = {
    consultar: async () => lista,
    operar: async (cuerpo) => {
      operaciones.push(cuerpo);
      lista = { version: 1, correos: [correo(A, "ana.reyes@example.org", "pendiente", { codigo: { vence_utc: "2026-09-29T11:00:00Z", intentos_restantes: 5 } })] };
      return { recibo_ref: "r", correo_ref: A, version: 1, replay: false, envio: "aceptado" };
    },
  };
  const vista = crearSuperficieCorreos({ cliente, textos, aleatorio });
  const contenedor = contenedorPrueba();
  vista.instalar(contenedor);
  await vista.cargar();
  assert.match(contenedor.raiz.innerHTML, /Aún no tiene ninguna dirección/u);
  const formulario = { matches: (s) => s === "[data-correos-anadir]", elements: {}, dataset: {} };
  globalThis.FormData ??= class {};
  const original = globalThis.FormData;
  globalThis.FormData = class { constructor() {} get(nombre) { return nombre === "direccion" ? "  ana.reyes@example.org " : null; } };
  try {
    contenedor.manejadores.submit({ target: formulario, preventDefault() {} });
    await new Promise((r) => setTimeout(r, 0));
    await new Promise((r) => setTimeout(r, 0));
  } finally { globalThis.FormData = original; }
  assert.equal(operaciones.length, 1);
  assert.deepEqual(operaciones[0], { operacion: "anadir", version_esperada: 0, clave_operacion: "web-correo-123e4567-e89b-12d3-a456-426614174000", direccion: "ana.reyes@example.org" });
  assert.match(contenedor.raiz.innerHTML, /Le hemos enviado un código a ana\.reyes@example\.org/u);
  assert.match(contenedor.raiz.innerHTML, /Código de 8 números/u);
});

test("código incorrecto, dirección en uso y fallo incierto dan mensajes útiles", async () => {
  const lista = { version: 1, correos: [correo(A, "ana.reyes@example.org", "verificado", { activo: true }), correo(B, "otra@example.org", "pendiente", { codigo: { vence_utc: "2026-09-29T11:00:00Z", intentos_restantes: 5 } })] };
  const errores = [new ErrorCorreos(422, "codigo_incorrecto", 4), new ErrorCorreos(503, "no_disponible")];
  const enviados = [];
  const cliente = { consultar: async () => lista, operar: async (cuerpo) => { enviados.push(cuerpo); throw errores.shift(); } };
  const vista = crearSuperficieCorreos({ cliente, textos, aleatorio });
  const contenedor = contenedorPrueba();
  vista.instalar(contenedor);
  await vista.cargar();
  const original = globalThis.FormData;
  globalThis.FormData = class { get(nombre) { return nombre === "codigo" ? "1111 2222" : null; } };
  try {
    contenedor.manejadores.submit({ target: { matches: (s) => s === "[data-correos-verificar]", dataset: { correosVerificar: B } }, preventDefault() {} });
    await new Promise((r) => setTimeout(r, 5));
  } finally { globalThis.FormData = original; }
  assert.equal(enviados[0].codigo, "11112222");
  assert.match(contenedor.raiz.innerHTML, /El código no es correcto\. Le quedan 4 intentos/u);
  assert.match(contenedor.raiz.innerHTML, /aria-invalid="true"/u);
  contenedor.manejadores.click({ target: { closest: () => ({ dataset: { correosAccion: "activar", correosRef: B } }) } });
  await new Promise((r) => setTimeout(r, 5));
  assert.match(contenedor.raiz.innerHTML, /No sabemos si se ha completado/u);
  assert.match(contenedor.raiz.innerHTML, /data-correos-accion="reintentar"/u);
  // Reintentar repite la MISMA operación con la misma clave.
  errores.push(new ErrorCorreos(409, "en_uso"));
  contenedor.manejadores.click({ target: { closest: () => ({ dataset: { correosAccion: "reintentar" } }) } });
  await new Promise((r) => setTimeout(r, 5));
  assert.equal(enviados.length, 3);
  assert.deepEqual(enviados[2], enviados[1]);
  assert.match(contenedor.raiz.innerHTML, /Elija antes otra dirección para los avisos/u);
});

test("los textos de «Mis correos» existen en castellano e inglés con la misma forma", async () => {
  const en = await cargarTextosCorreos({ idioma: "en" });
  assert.deepEqual(en.faltantes, []);
  assert.deepEqual(Object.keys(en.seccion("correos")).sort(), Object.keys(textos.seccion("correos")).sort());
  assert.equal(en.traducir("correos.titulo"), "My email addresses");
});
