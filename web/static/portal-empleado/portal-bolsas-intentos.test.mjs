import test from "node:test";
import assert from "node:assert/strict";
import {
  consultarIntentosContacto,
  crearControladorIntentosContacto,
  registrarContactoIntento,
  renderizarIntentosContacto,
  rutaContactosCandidato,
} from "./portal-bolsas-intentos.js?v=20261010-ct-bolsa-cohorte-v10";
import { MENSAJES_INTENTOS, crearTraductorIntentos } from "./portal-i18n-intentos.js";

const candidato = { participacion_ref: "participacion:1", estado_clave: "disponible", ultimo_llamamiento: { llamamiento_ref: "llamamiento:1" } };
const intentos = {
  configurado: true, llamamiento_ref: "llamamiento:1", sin_contacto: 1, maximo: 4, proceso: 1, intento: 2,
  intentos_por_proceso: 2, procesos: 2, contactado: false, baja_propuesta: false, completo: true,
  ultimo_intento: "2026-09-28T08:00:00Z", siguiente_permitido_desde: "2026-09-28T10:00:00Z", avisos: ["antes_de_separacion"],
  franja: { valor: "09:00-14:00", solo_dias_habiles: true, control: "advertir" },
  reglas: [
    { clave: "b02.intentos_contacto", etiqueta: "Dos intentos", referencia: "vec.bolsa.reglas:1:b02.intentos_contacto", ejemplo: false },
    { clave: "b04.franja_llamadas", etiqueta: "Franja <b>", referencia: "vec.bolsa.reglas:1:b04.franja_llamadas", ejemplo: true },
  ],
};
const respuesta = (status, cuerpo) => ({ ok: status >= 200 && status < 300, status, json: async () => cuerpo });

test("el catálogo i18n de intentos está completo y rechaza claves ajenas", () => {
  const t = crearTraductorIntentos();
  assert.equal(t("registrado"), "Contacto registrado.");
  assert.throws(() => t("inexistente"));
  assert.throws(() => crearTraductorIntentos({ ...MENSAJES_INTENTOS, titulo: "" }));
});

test("la ficha muestra proceso, avisos y reglas sin rotular su procedencia, escapando textos", () => {
  const salida = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: intentos } });
  assert.match(salida, /<details class="intentos-ayuda"><summary[^>]+>\?<\/summary><p>[^<]*plazo de respuesta/u);
  assert.match(salida, /Proceso 1 de 2 · intento 2 de 2/);
  assert.match(salida, /Antes de la separación mínima/);
  assert.match(salida, /1 de 4/);
  assert.match(salida, /<li>Franja &lt;b&gt;/);
  assert.doesNotMatch(salida, /Regla de ejemplo|<code>/);
  assert.match(salida, /data-intentos-form="intento"/);
  assert.match(salida, /data-intentos-form="rebote"/);
  assert.doesNotMatch(salida, /data-b8-accion/);
});

test("con los procesos agotados propone la baja con la exclusión existente y no ofrece otro intento", () => {
  const agotado = { ...intentos, sin_contacto: 4, proceso: 0, intento: 0, baja_propuesta: true, avisos: [] };
  const salida = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: agotado } });
  assert.match(salida, /Baja propuesta/);
  assert.match(salida, /data-b8-accion="seleccionar" data-operacion="excluir"/);
  assert.doesNotMatch(salida, /data-intentos-form="intento"/);
  assert.match(salida, /data-intentos-form="rebote"/);
  const excluido = renderizarIntentosContacto({ candidato: { ...candidato, estado_clave: "excluido" }, estado: { carga: "listo", datos: agotado } });
  assert.doesNotMatch(excluido, /data-b8-accion/);
});

test("sin llamamiento o sin catálogo lo dice sin inventar reglas", () => {
  assert.match(renderizarIntentosContacto({ candidato: { participacion_ref: "p" } }), /Sin llamamiento en curso/);
  const sin = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: { configurado: false } } });
  assert.match(sin, /Sin reglas de intentos en el catálogo/);
  assert.doesNotMatch(sin, /Reglas aplicadas/);
});

test("consulta con llamamiento_ref y valida el contrato", async () => {
  let url = "";
  const ok = await consultarIntentosContacto("bolsa:1", "participacion:1", "llamamiento:1", {
    fetchImpl: async (u) => { url = u; return respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", contactos: [], intentos } }); },
  });
  assert.equal(url, `${rutaContactosCandidato("bolsa:1", "participacion:1")}?llamamiento_ref=llamamiento%3A1`);
  assert.equal(ok.ok, true);
  const roto = await consultarIntentosContacto("b", "p", "l", { fetchImpl: async () => respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", intentos: { configurado: true } } }) });
  assert.equal(roto.ok, false);
});

test("traduce los rechazos de las reglas al registrar", async () => {
  const res = await registrarContactoIntento("b", "p", {}, "k", { fetchImpl: async () => respuesta(409, { error: { codigo: "intento_antes_de_separacion" } }) });
  assert.equal(res.ok, false);
  assert.equal(res.mensaje, "No ha pasado la separación mínima desde el último intento.");
});

test("cada rechazo del registro tiene su propio mensaje: nota con dato personal, clave repetida o resultado no admitido", async () => {
  const mensaje = async (estado, codigo) => (await registrarContactoIntento("b", "p", {}, "k", { fetchImpl: async () => respuesta(estado, { error: { codigo } }) })).mensaje;
  assert.match(await mensaje(400, "anotacion_con_dato_personal"), /^La nota no puede llevar datos personales/u);
  assert.match(await mensaje(409, "contacto_en_conflicto"), /ya se envió antes con otros datos/u);
  assert.equal(await mensaje(409, "contacto_no_valido"), "No se puede anotar este resultado en este llamamiento. Revise el resultado elegido.");
  assert.doesNotMatch(await mensaje(409, "contacto_no_valido"), /clave/u);
});

test("el justificante muestra el aviso de día no hábil que devolvió el propio registro", () => {
  const contactado = { ...intentos, contactado: true, avisos: [] };
  const salida = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: contactado, recibo: "recibo:contacto:1", avisosRegistro: ["dia_no_habil", "<inventado>"] } });
  assert.match(salida, /Queda registrado con este aviso: <span class="estado-chip aviso">Día no hábil<\/span><\/p>/u);
  assert.doesNotMatch(salida, /inventado/u);
  const sinAviso = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: contactado, recibo: "recibo:contacto:1", avisosRegistro: [] } });
  assert.doesNotMatch(sinAviso, /Queda registrado con este aviso/u);
});

test("el controlador registra un rebote de correo ligado al llamamiento y recarga el estado", async () => {
  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.v = f.valores; } get(c) { return this.v[c]; } };
  try {
    const peticiones = [];
    const fetchImpl = async (url, opciones) => {
      peticiones.push({ url, opciones });
      if (opciones.method === "POST") return respuesta(201, { data: { recibo_ref: "recibo:contacto:1" } });
      return respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", contactos: [], intentos } });
    };
    const modal = { candidato };
    const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: modal };
    const controlador = crearControladorIntentosContacto({ estado, renderizar() {}, fetchImpl });
    const formulario = { dataset: { intentosForm: "rebote" }, valores: { instante: "2026-09-28T10:00", anotacion: "Rebote del aviso" }, closest() { return this; } };
    assert.equal(controlador.manejarSubmit({ target: formulario, preventDefault() {} }), true);
    await new Promise((r) => setTimeout(r, 0));
    await new Promise((r) => setTimeout(r, 0));
    const post = peticiones.find((p) => p.opciones.method === "POST");
    const cuerpo = JSON.parse(post.opciones.body);
    assert.equal(cuerpo.canal, "correo");
    assert.equal(cuerpo.resultado, "no_entregado");
    assert.equal(cuerpo.llamamiento_ref, "llamamiento:1");
    assert.ok(post.opciones.headers["Idempotency-Key"]);
    assert.equal(modal.intentosContacto.recibo, "recibo:contacto:1");
    assert.equal(modal.intentosContacto.carga, "listo");
  } finally {
    globalThis.FormData = anterior;
  }
});

const registroTelefono = {
  esquema: "vec.bolsa.registro_telefono.v1", instante_servidor: true, anotacion_opcional: true,
  resultados: ["no_contesta", "comunica", "numero_erroneo", "acepta", "rechaza", "aplazado"],
};
const datosServidor = { ...intentos, registro_telefono: registroTelefono };
const lecturaTelefono = async (datos = datosServidor) => consultarIntentosContacto("bolsa:1", "participacion:1", "llamamiento:1", {
  fetchImpl: async () => respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", intentos: datos } }),
});

function acuseTelefono(comando) {
  return { ...comando, participacion_ref: "participacion:1", instante: "2026-10-08T09:12:45Z", recibo_ref: "recibo:contacto:1", reutilizado: false };
}

test("el modo de servidor muestra los resultados publicados y nota opcional sólo para teléfono", async () => {
  assert.equal((await lecturaTelefono()).ok, true);
  const salida = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: datosServidor } });
  const telefono = salida.match(/<form data-intentos-form="intento">([\s\S]*?)<\/form>/)[1];
  const rebote = salida.match(/<form data-intentos-form="rebote">([\s\S]*?)<\/form>/)[1];
  assert.match(telefono, /Comunica/);
  assert.match(telefono, /Pide pensarlo/);
  assert.match(telefono, /Nota \(opcional\)/);
  assert.match(telefono, /La fecha y la hora se registran al confirmar la llamada/u);
  assert.doesNotMatch(telefono, /name="instante"|<textarea[^>]*required/);
  assert.match(rebote, /name="instante" required/);
  assert.match(rebote, /<textarea[^>]*required/);
  assert.match(telefono, /Indica que acepta/u);
  assert.match(telefono, /Indica que rechaza/u);
  assert.doesNotMatch(salida, /value="sms"|value="telegram"/u);
});

test("sin política de intentos conserva el registro telefónico anunciado por el servidor", async () => {
  const sinPolitica = { configurado: false, llamamiento_ref: "llamamiento:1",
    registro_telefono: registroTelefono };
  assert.equal((await lecturaTelefono(sinPolitica)).ok, true);
  const salida = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: sinPolitica } });
  const telefono = salida.match(/<form data-intentos-form="intento">([\s\S]*?)<\/form>/u)?.[1];
  assert.ok(telefono);
  assert.match(telefono, /Indica que acepta/u);
  assert.match(telefono, /Nota \(opcional\)/u);
  assert.doesNotMatch(telefono, /name="instante"|<textarea[^>]*required/u);
});

test("metadata incompleta, duplicada o ajena no habilita resultados ni degrada a legado", async () => {
  for (const registro of [null, { ...registroTelefono, esquema: "otra" },
    { ...registroTelefono, instante_servidor: false }, { ...registroTelefono, resultados: ["no_contesta", "no_contesta"] },
    { ...registroTelefono, resultados: ["<img>"] }, { ...registroTelefono, resultados: ["resuelto"] }]) {
    assert.equal((await lecturaTelefono({ ...intentos, registro_telefono: registro })).ok, false);
  }
});

test("un acuse telefónico de otro contacto o sin instante confirmado no muestra éxito", async () => {
  const comando = { canal: "telefono", llamamiento_ref: "llamamiento:1", resultado: "comunica", anotacion: "" };
  for (const cambio of [{ participacion_ref: "otra" }, { llamamiento_ref: "otro" }, { resultado: "acepta" },
    { anotacion: "cambiada" }, { instante: "fecha inválida" }]) {
    const res = await registrarContactoIntento("bolsa:1", "participacion:1", comando, "k", {
      fetchImpl: async () => respuesta(201, { data: { ...acuseTelefono(comando), ...cambio } }),
    });
    assert.equal(res.ok, false);
  }
  const pendiente = await registrarContactoIntento("bolsa:1", "participacion:1", comando, "k", {
    fetchImpl: async () => respuesta(202, { data: acuseTelefono(comando) }),
  });
  assert.equal(pendiente.ok, false, "202 no acredita un registro confirmado");
});

test("el doble envío no duplica el POST y el recibo precede al refresco fallido", async () => {
  await lecturaTelefono();
  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.v = f.valores; } get(c) { return this.v[c]; } };
  try {
    let resolverPost; let resolverGet;
    const posts = [];
    const modal = { candidato, intentosContacto: { carga: "listo", datos: datosServidor } };
    const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: modal };
    const pintados = [];
    const controlador = crearControladorIntentosContacto({ estado,
      renderizar: () => pintados.push(renderizarIntentosContacto({ candidato, estado: modal.intentosContacto })),
      fetchImpl: async (_url, opciones) => {
        if (opciones.method === "POST") {
          posts.push(opciones);
          return new Promise((resolver) => { resolverPost = resolver; });
        }
        return new Promise((resolver) => { resolverGet = resolver; });
      } });
    const formulario = { dataset: { intentosForm: "intento" },
      valores: { resultado: "comunica", anotacion: "" }, closest() { return this; } };
    const evento = { target: formulario, preventDefault() {} };
    assert.equal(controlador.manejarSubmit(evento), true);
    assert.equal(controlador.manejarSubmit(evento), true);
    assert.equal(posts.length, 1);
    const comando = JSON.parse(posts[0].body);
    assert.equal(Object.hasOwn(comando, "instante"), false);
    resolverPost(respuesta(201, { data: acuseTelefono(comando) }));
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(modal.intentosContacto.recibo, "recibo:contacto:1");
    assert.ok(pintados.some((html) => /recibo:contacto:1/u.test(html)),
      "el recibo se ve mientras se actualiza el historial");
    resolverGet(respuesta(503, { error: { codigo: "servicio_no_disponible" } }));
    await new Promise((resolver) => setImmediate(resolver));
    assert.match(renderizarIntentosContacto({ candidato, estado: modal.intentosContacto }), /recibo:contacto:1/u);
    assert.equal(posts.length, 1);
  } finally { globalThis.FormData = anterior; }
});

test("el registro con reloj servidor conserva nota y clave en un reintento y no envía fecha", async () => {
  await lecturaTelefono();
  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.v = f.valores; } get(c) { return this.v[c]; } };
  try {
    const posts = [];
    const modal = { candidato, intentosContacto: { carga: "listo", datos: datosServidor } };
    const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: modal };
    const controlador = crearControladorIntentosContacto({ estado, renderizar() {}, fetchImpl: async (_url, opciones) => {
      if (opciones.method !== "POST") return respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", intentos: datosServidor } });
      posts.push(opciones);
      return posts.length === 1 ? respuesta(503, { error: { codigo: "servicio_no_disponible" } })
        : respuesta(200, { data: acuseTelefono(JSON.parse(opciones.body)) });
    } });
    const formulario = { dataset: { intentosForm: "intento" }, valores: { resultado: "comunica", anotacion: "Esperar <sin perder>" }, closest() { return this; } };
    const enviar = async () => {
      assert.equal(controlador.manejarSubmit({ target: formulario, preventDefault() {} }), true);
      await new Promise((r) => setTimeout(r, 0));
    };
    await enviar();
    const pintado = renderizarIntentosContacto({ candidato, estado: modal.intentosContacto });
    assert.match(pintado, /Esperar &lt;sin perder&gt;/);
    assert.match(pintado, /value="comunica" selected/);
    await enviar();
    assert.equal(posts.length, 2);
    assert.equal(posts[0].body, posts[1].body);
    assert.equal(posts[0].headers["Idempotency-Key"], posts[1].headers["Idempotency-Key"]);
    assert.equal(Object.hasOwn(JSON.parse(posts[0].body), "instante"), false);
    assert.equal(modal.intentosContacto.registradoEn, "2026-10-08T09:12:45Z");
    assert.equal(modal.intentosContacto.recibo, "recibo:contacto:1");
    assert.equal(estado.modalFicha, modal);
  } finally { globalThis.FormData = anterior; }
});

test("una consulta pendiente se comparte y una respuesta tardía no cambia otra ficha", async () => {
  let resolver;
  let consultas = 0;
  const modal = { candidato };
  const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: modal };
  const controlador = crearControladorIntentosContacto({ estado, renderizar() {}, fetchImpl: async () => {
    consultas++;
    return new Promise((r) => { resolver = r; });
  } });
  const pendiente = controlador.cargar(modal);
  await controlador.cargar(modal);
  assert.equal(consultas, 1);
  const otro = { candidato: { ...candidato, participacion_ref: "participacion:2" } };
  estado.modalFicha = otro;
  resolver(respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", intentos } }));
  await pendiente;
  assert.equal(otro.intentosContacto, undefined);
});
