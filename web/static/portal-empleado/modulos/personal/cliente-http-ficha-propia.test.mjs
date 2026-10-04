import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { LIMITE_TEXTO_FICHA_PROPIA, RUTA_FICHA_PROPIA, ACCEPT_FICHA_PROPIA_EXPORTACION, crearFuentesFichaPropia } from "./cliente-http-ficha-propia.js";
import { LIMITE_TEXTO_CAMPO_FICHA } from "./vista-ficha-integral.js";
import { crearTraductorFichaPropia, formatearDiasFichaPropia } from "./i18n-ficha-propia.js";

const FICHA = Object.freeze({
  data: {
    exportacion_servicios_disponible: true,
    ficha: {
      corte: { vigente_en: "2026-09-25", conocido_en: "2026-09-25T08:59:59.000000Z" },
      relaciones: [
        { inicio: "2026-01-01", fin: "", estado: "vigente", regimen: "Funcionario interino", modalidad: "Vacante", unidad: "Servicio de Personal", puesto: "Técnico/a de gestión", situacion: "Servicio activo" },
        { inicio: "2020-03-01", fin: "2020-12-31", estado: "finalizada", regimen: "Laboral temporal", modalidad: "", unidad: "", puesto: "", situacion: "" },
      ],
      servicios: [{ inicio: "2019-01-01", fin: "2019-12-31", clase: "Servicios previos", dias: 1365, estado: "reconocido" }],
    },
    recibo_ref: "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100",
    consultada_en: "2026-09-25T09:00:00.000000Z",
  },
});

function respuesta(cuerpo, estado = 200) {
  return new Response(typeof cuerpo === "string" ? cuerpo : JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

test("una sola consulta same-origin alimenta relaciones y servicios con textos legibles", async () => {
  const llamadas = [];
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta, opciones) => { llamadas.push([ruta, opciones]); return respuesta(FICHA); } }).preparar();
  assert.deepEqual(Object.keys(fuentes), ["relaciones", "servicios"]);
  const relaciones = await fuentes.relaciones.consultarPropios({ signal: new AbortController().signal });
  const servicios = await fuentes.servicios.consultarPropios({});
  assert.equal(llamadas.length, 1, "los dos apartados comparten la misma respuesta");
  const [ruta, opciones] = llamadas[0];
  assert.equal(ruta, RUTA_FICHA_PROPIA);
  assert.deepEqual([opciones.method, opciones.credentials, opciones.mode, opciones.cache, opciones.redirect, opciones.referrerPolicy],
    ["GET", "same-origin", "same-origin", "no-store", "error", "no-referrer"]);
  assert.equal(opciones.body, undefined);
  assert.equal(opciones.headers.Accept, ACCEPT_FICHA_PROPIA_EXPORTACION);
  assert.equal(relaciones.estado, "disponible");
  assert.equal(relaciones.fuente, "Registro de Personal");
  assert.equal(relaciones.actualizado_en, "2026-09-25T09:00:00.000000Z");
  assert.deepEqual(relaciones.items[0], { desde: "2026-01-01", hasta: "Actualidad", regimen: "Funcionario interino · Vacante", puesto: "Técnico/a de gestión", unidad: "Servicio de Personal", estado: "Servicio activo" });
  assert.deepEqual(relaciones.items[1], { desde: "2020-03-01", hasta: "2020-12-31", regimen: "Laboral temporal", puesto: "", unidad: "", estado: "Finalizada" });
  assert.equal(servicios.exportacion_servicios_disponible, true);
  assert.equal(servicios.recibo_ref, FICHA.data.recibo_ref);
  assert.deepEqual(servicios.corte, FICHA.data.ficha.corte);
  assert.ok(Object.isFrozen(servicios.corte));
  assert.equal(typeof fuentes.servicios.exportarPropios, "function");
  assert.equal(fuentes.relaciones.exportarPropios, undefined);
  assert.deepEqual(servicios.items, [{ desde: "2019-01-01", hasta: "2019-12-31", procedencia: "Servicios previos", reconocimiento: "1365 días", estado: "Reconocido" }]);
  assert.ok(!JSON.stringify([relaciones, servicios]).match(/(?:emp|per|rel|srv)_/u), "sin referencias internas");
});

test("401, 403 y 404 no ofrecen apartados propios", async () => {
  const casos = [
    () => respuesta({ error: "no_encontrada" }, 404),
    () => respuesta({ error: "sin_empleado" }, 403),
    () => respuesta({ error: "autenticacion_requerida" }, 401),
  ];
  for (const [indice, caso] of casos.entries()) {
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => caso() }).preparar();
    assert.deepEqual(fuentes, {}, `caso ${indice}`);
  }
});

test("fallos de red, servidor o contrato dejan ambos apartados en error y permiten recuperar", async () => {
  const casos = [
    () => respuesta({ error: "no_disponible" }, 503),
    () => respuesta({ data: { ...FICHA.data, persona_ref: "per_x" } }),
    () => respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones: [{ ...FICHA.data.ficha.relaciones[0], relacion_ref: "rel_x" }] } } }),
    () => respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, servicios: [{ ...FICHA.data.ficha.servicios[0], estado: "pendiente" }] } } }),
    () => respuesta("no es JSON"),
    () => new Response(JSON.stringify(FICHA), { status: 200, headers: { "Content-Type": "text/html" } }),
    () => { throw new TypeError("red"); },
  ];
  for (const [indice, caso] of casos.entries()) {
    let llamadas = 0;
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => {
      llamadas += 1; return llamadas === 1 ? caso() : respuesta(FICHA);
    } }).preparar();
    assert.deepEqual(Object.keys(fuentes), ["relaciones", "servicios"], `caso ${indice}`);
    assert.equal(fuentes.relaciones.estadoInicial, "error");
    assert.deepEqual(await fuentes.relaciones.consultarPropios(), { estado: "error" });
    assert.deepEqual(await fuentes.servicios.consultarPropios(), { estado: "error" });
    assert.equal(llamadas, 1, "el fallo se comparte sin duplicar GET");
    fuentes.servicios.actualizar();
    assert.equal((await fuentes.relaciones.consultarPropios()).estado, "disponible");
    assert.equal((await fuentes.servicios.consultarPropios()).estado, "disponible");
    assert.equal(llamadas, 2, "actualizar consulta una sola vez para ambos apartados");
  }
});

test("la columna Estado da el estado de la relación si no está vigente; la abierta llega hasta la actualidad", async () => {
  const relaciones = [
    { inicio: "2026-02-01", fin: "", estado: "suspendida", regimen: "Laboral", modalidad: "", unidad: "", puesto: "", situacion: "Servicio activo" },
    { inicio: "2025-01-01", fin: "2025-12-31", estado: "finalizada", regimen: "Laboral", modalidad: "", unidad: "", puesto: "", situacion: "Excedencia voluntaria" },
    { inicio: "2026-03-01", fin: "", estado: "vigente", regimen: "Laboral", modalidad: "", unidad: "", puesto: "", situacion: "" },
  ];
  const sobre = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones } } };
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(sobre) }).preparar();
  const { items } = await fuentes.relaciones.consultarPropios({});
  assert.deepEqual(items.map((item) => [item.hasta, item.estado]),
    [["Actualidad", "Suspendida"], ["2025-12-31", "Finalizada"], ["Actualidad", "Vigente"]]);
});

test("límite de texto único: régimen y modalidad largos se recortan y la vista los admite", async () => {
  assert.equal(LIMITE_TEXTO_FICHA_PROPIA, 300);
  assert.equal(LIMITE_TEXTO_CAMPO_FICHA, LIMITE_TEXTO_FICHA_PROPIA, "cliente y vista comparten límite");
  const largo = "R".repeat(300); const emoji = "😀".repeat(150);
  const relaciones = [
    { inicio: "2026-01-01", fin: "", estado: "vigente", regimen: largo, modalidad: largo, unidad: largo, puesto: largo, situacion: largo },
    { inicio: "2026-01-01", fin: "", estado: "vigente", regimen: "A", modalidad: emoji, unidad: "", puesto: "", situacion: "" },
  ];
  const sobre = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones } } };
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(sobre) }).preparar();
  const { items } = await fuentes.relaciones.consultarPropios({});
  assert.equal(items[0].regimen.length, 300);
  assert.ok(items[0].regimen.endsWith("…"));
  assert.ok(items.every((item) => Object.values(item).every((valor) => valor.length <= LIMITE_TEXTO_CAMPO_FICHA)));
  assert.doesNotMatch(items[1].regimen, /[\ud800-\udbff](?![\udc00-\udfff])/u, "sin pares sustitutos partidos");
  const excedido = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones: [{ ...relaciones[0], unidad: `${largo}x` }] } } };
  const fuentesInvalidas = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(excedido) }).preparar();
  assert.deepEqual(await fuentesInvalidas.relaciones.consultarPropios(), { estado: "error" });
});

test("más filas de las que se muestran: los apartados se ofrecen con estado propio, sin volver a consultar", async () => {
  const exceso = [
    () => respuesta({ error: "excede_limite" }, 422),
    () => respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, servicios: Array.from({ length: 201 }, () => FICHA.data.ficha.servicios[0]) } } }),
  ];
  for (const [indice, caso] of exceso.entries()) {
    let llamadas = 0;
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => { llamadas += 1; return caso(); } }).preparar();
    assert.deepEqual(Object.keys(fuentes), ["relaciones", "servicios"], `caso ${indice}`);
    assert.deepEqual(await fuentes.relaciones.consultarPropios({}), { estado: "excede_limite" });
    assert.deepEqual(await fuentes.servicios.consultarPropios({}), { estado: "excede_limite" });
    assert.equal(llamadas, 1, `caso ${indice}`);
  }
  // Un 422 con otro código no es un exceso de filas.
  const otras = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta({ error: "otra_cosa" }, 422) }).preparar();
  assert.deepEqual(await otras.relaciones.consultarPropios(), { estado: "error" });
});

test("una ficha sin registros deja los apartados vacíos, no en cero inventado", async () => {
  const vacia = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones: [], servicios: [] } } };
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(vacia) }).preparar();
  assert.deepEqual(await fuentes.servicios.consultarPropios({}), { estado: "vacio", fuente: "Registro de Personal", actualizado_en: "2026-09-25T09:00:00.000000Z", items: [], fecha_referencia: "2026-09-25", exportacion_servicios_disponible: true, historia_servicios_disponible: false, recibo_ref: FICHA.data.recibo_ref, corte: FICHA.data.ficha.corte });
});

test("actualizar borra la ficha anterior y no permite que una respuesta tardía repueble la caché", async () => {
  let resolverAntigua;
  let llamadas = 0;
  const cliente = crearFuentesFichaPropia({ fetchImpl: async () => {
    llamadas += 1;
    if (llamadas === 1) return respuesta(FICHA);
    if (llamadas === 2) return new Promise((resolver) => { resolverAntigua = resolver; });
    if (llamadas === 3) return respuesta({ error: "no_disponible" }, 503);
    return respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones: [], servicios: [] } } });
  } });
  const fuentes = await cliente.preparar();
  assert.equal((await fuentes.relaciones.consultarPropios()).estado, "disponible");
  fuentes.relaciones.actualizar();
  const antigua = fuentes.relaciones.consultarPropios();
  fuentes.servicios.actualizar();
  assert.deepEqual(await fuentes.servicios.consultarPropios(), { estado: "error" });
  resolverAntigua(respuesta(FICHA));
  await antigua;
  assert.deepEqual(await fuentes.relaciones.consultarPropios(), { estado: "error" }, "una respuesta anterior no sustituye el fallo reciente");
  fuentes.relaciones.actualizar();
  assert.equal((await fuentes.relaciones.consultarPropios()).estado, "vacio");
  assert.equal(llamadas, 4);
});

test("la consulta se cancela con la vista y respeta el plazo", async () => {
  const controlador = new AbortController();
  const fuentes = crearFuentesFichaPropia({ fetchImpl: (_ruta, { signal }) => new Promise((_resolver, rechazar) => signal.addEventListener("abort", () => rechazar(new Error("abortada")))) });
  const pendiente = fuentes.preparar({ signal: controlador.signal });
  controlador.abort();
  await assert.rejects(pendiente, (causa) => causa.codigo === "operacion_abortada");
  const conPlazo = crearFuentesFichaPropia({ plazoMs: 5, fetchImpl: (_ruta, { signal }) => new Promise((_resolver, rechazar) => signal.addEventListener("abort", () => rechazar(new Error("plazo")))) });
  const conError = await conPlazo.preparar();
  assert.deepEqual(await conError.relaciones.consultarPropios(), { estado: "error" });
  assert.throws(() => crearFuentesFichaPropia({ fetchImpl: null }), TypeError);
});

test("i18n de la ficha propia: plural de días y estados cerrados", () => {
  const t = crearTraductorFichaPropia();
  assert.equal(formatearDiasFichaPropia(1, t), "1 día");
  assert.equal(formatearDiasFichaPropia(0, t), "0 días");
  assert.equal(formatearDiasFichaPropia(12345, t), "12.345 días");
  assert.throws(() => t("estado_relacion_activa"), /desconocida/u);
  assert.throws(() => formatearDiasFichaPropia(-1, t), TypeError);
});

test("el cliente no usa almacenamiento del navegador ni credenciales entre orígenes", async () => {
  const fuente = await readFile(new URL("cliente-http-ficha-propia.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|credentials:\s*"include"|Authorization/u);
  assert.match(fuente, /from "\.\/i18n-ficha-propia\.js\?v=[\w.-]+"/u);
});


test("solo Servicios admite fecha civil y Relaciones conserva la consulta actual", async () => {
  const llamadas = [];
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta) => {
    llamadas.push(ruta);
    const corte = ruta.includes("?") ? "2020-02-29" : "2026-09-25";
    return respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, corte: { ...FICHA.data.ficha.corte, vigente_en: corte } } } });
  } }).preparar();
  const antiguo = await fuentes.servicios.consultarPropios({ fechaReferencia: "2020-02-29" });
  assert.equal(antiguo.fecha_referencia, "2020-02-29");
  assert.equal(fuentes.servicios.fechaReferencia, "2026-09-25");
  assert.equal((await fuentes.relaciones.consultarPropios()).items[0].hasta, "Actualidad");
  assert.deepEqual(llamadas, [RUTA_FICHA_PROPIA, `${RUTA_FICHA_PROPIA}?fecha_referencia=2020-02-29`]);
  for (const invalida of ["2025-02-29", "0000-01-01", "2020-02-29&empleado=emp_x", "2020-02-29T00:00:00Z"]) {
    await assert.rejects(fuentes.servicios.consultarPropios({ fechaReferencia: invalida }), (e) => e.codigo === "fecha_no_valida");
  }
  await assert.rejects(fuentes.relaciones.consultarPropios({ fechaReferencia: "2020-02-29" }), (e) => e.codigo === "fecha_no_admitida");
  assert.equal(llamadas.length, 2);
});

test("cambiar de fecha retira caché y un corte incorrecto o un 403 no reutiliza servicios anteriores", async () => {
  let pendiente; let llamadas = 0;
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => {
    llamadas += 1;
    if (llamadas === 1) return respuesta(FICHA);
    if (llamadas === 2) return new Promise((resolver) => { pendiente = resolver; });
    if (llamadas === 3) return respuesta({ error: "acceso_denegado" }, 403);
    return respuesta(FICHA);
  } }).preparar();
  fuentes.servicios.actualizar();
  const anterior = fuentes.servicios.consultarPropios({ fechaReferencia: "2020-02-29" });
  fuentes.servicios.actualizar();
  assert.deepEqual(await fuentes.servicios.consultarPropios({ fechaReferencia: "2021-01-01" }), { estado: "denegado" });
  pendiente(respuesta(FICHA)); await anterior;
  assert.deepEqual(await fuentes.servicios.consultarPropios({ fechaReferencia: "2021-01-01" }), { estado: "denegado" });
  fuentes.servicios.actualizar();
  assert.deepEqual(await fuentes.servicios.consultarPropios({ fechaReferencia: "2020-02-29" }), { estado: "error" }, "un resultado con otra fecha no se muestra");
});


test("el catálogo dedicado ofrece las mismas claves y variables en ambos idiomas", async () => {
  const raiz = new URL("../../../textos/", import.meta.url);
  const [castellano, ingles] = await Promise.all(["es", "en"].map(async (idioma) => JSON.parse(await readFile(new URL(`${idioma}/personal-corte-propio.json`, raiz), "utf8"))));
  assert.deepEqual(Object.keys(castellano.general), Object.keys(ingles.general));
  for (const clave of Object.keys(castellano.general)) {
    assert.ok(castellano.general[clave] && ingles.general[clave]);
    assert.deepEqual(castellano.general[clave].match(/\{[a-z_]+\}/gu), ingles.general[clave].match(/\{[a-z_]+\}/gu));
  }
});


test("exportar Servicios conserva la consulta original y no vuelve a hacer GET", async () => {
  const { createHash } = await import("node:crypto");
  const csv = new TextEncoder().encode("Inicio,Fin\n");
  const peticiones = [];
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    if (opciones.method === "GET") return respuesta(FICHA);
    return new Response(csv, { headers: { "Content-Type": "text/csv; charset=utf-8", "Content-Length": String(csv.length), "Content-Disposition": "attachment; filename=servicios.csv", "X-Content-SHA256": createHash("sha256").update(csv).digest("hex"), "X-Recibo-Ref": FICHA.data.recibo_ref } });
  } }).preparar();
  const resultado = await fuentes.servicios.consultarPropios();
  const archivo = await fuentes.servicios.exportarPropios({ reciboRef: resultado.recibo_ref, corte: resultado.corte });
  assert.deepEqual(archivo.bytes, csv);
  assert.deepEqual(peticiones.map(({ opciones }) => opciones.method), ["GET", "POST"]);
  assert.deepEqual(JSON.parse(peticiones[1].opciones.body).corte, FICHA.data.ficha.corte);
  assert.equal(JSON.parse(peticiones[1].opciones.body).recibo_ref, FICHA.data.recibo_ref);
});


test("servidor anterior o exportación no montada conserva la consulta y declara disponibilidad falsa", async () => {
  for (const valor of [undefined, false, true]) {
    const data = { ...FICHA.data }; delete data.exportacion_servicios_disponible;
    if (valor !== undefined) data.exportacion_servicios_disponible = valor;
    const llamadas = [];
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta, opciones) => { llamadas.push([ruta, opciones]); return respuesta({ data }); } }).preparar();
    const servicios = await fuentes.servicios.consultarPropios();
    assert.equal(servicios.estado, "disponible");
    assert.equal(servicios.exportacion_servicios_disponible, valor === true);
    assert.equal(servicios.recibo_ref, FICHA.data.recibo_ref); assert.deepEqual(servicios.corte, FICHA.data.ficha.corte);
    assert.equal(llamadas.length, 1); assert.equal(llamadas[0][1].headers.Accept, ACCEPT_FICHA_PROPIA_EXPORTACION);
  }
});

test("disponibilidad con tipo erróneo o claves extra rechaza el DTO", async () => {
  for (const data of [{ ...FICHA.data, exportacion_servicios_disponible: "true" }, { ...FICHA.data, exportacion_servicios_disponible: 1 }, { ...FICHA.data, permiso_exportar: true }]) {
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta({ data }) }).preparar();
    assert.deepEqual(await fuentes.servicios.consultarPropios(), { estado: "error" });
  }
});

test("sesión caducada al exportar retira toda la ficha cacheada sin GET hasta actualizar", async () => {
  const peticiones = [];
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta, opciones) => {
    peticiones.push(opciones.method);
    if (opciones.method === "GET") return respuesta(FICHA);
    const cuerpo = JSON.stringify({ error: "autenticacion_requerida" });
    return new Response(cuerpo, { status: 401, headers: { "Content-Type": "application/json; charset=utf-8", "Content-Length": String(Buffer.byteLength(cuerpo)) } });
  } }).preparar();
  const servicios = await fuentes.servicios.consultarPropios();
  await assert.rejects(fuentes.servicios.exportarPropios({ reciboRef: servicios.recibo_ref, corte: servicios.corte }), { estado: 401 });
  for (const fuente of [fuentes.servicios, fuentes.relaciones]) {
    assert.deepEqual(await fuente.consultarPropios(), { estado: "denegado", aviso_exportacion: "sesion_caducada" });
  }
  assert.deepEqual(peticiones, ["GET", "POST"]);
  fuentes.servicios.actualizar();
  assert.equal((await fuentes.servicios.consultarPropios()).estado, "disponible");
  assert.deepEqual(peticiones, ["GET", "POST", "GET"]);
});

test("denegar exportación conserva consulta y bloqueo al reabrir, sin reutilizar permiso de consulta", async () => {
  for (const estado of [403, 404]) {
    const peticiones = [];
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta, opciones) => {
      peticiones.push(opciones.method);
      if (opciones.method === "GET") return respuesta(FICHA);
      const cuerpo = JSON.stringify({ error: estado === 403 ? "acceso_denegado" : "no_encontrada" });
      return new Response(cuerpo, { status: estado, headers: { "Content-Type": "application/json; charset=utf-8", "Content-Length": String(Buffer.byteLength(cuerpo)) } });
    } }).preparar();
    const consulta = await fuentes.servicios.consultarPropios();
    const entrada = { reciboRef: consulta.recibo_ref, corte: consulta.corte };
    await assert.rejects(fuentes.servicios.exportarPropios(entrada), { codigo: "denegado" });
    const conservada = await fuentes.servicios.consultarPropios();
    assert.deepEqual(conservada.items, consulta.items);
    assert.deepEqual(conservada.corte, consulta.corte);
    assert.equal(conservada.exportacion_servicios_disponible, false);
    await assert.rejects(fuentes.servicios.exportarPropios(entrada), { codigo: "denegado" });
    assert.deepEqual(peticiones, ["GET", "POST"]);
  }
});

test("historia sólo se ofrece con disponibilidad explícita y conserva exportación del servidor anterior", async () => {
  for (const valor of [undefined,false,true,"true"]) {
    const data={...FICHA.data};if(valor!==undefined)data.historia_servicios_disponible=valor;
    let peticion;
    const fuentes=await crearFuentesFichaPropia({fetchImpl:async(_,opciones)=>{peticion=opciones;return respuesta({data});}}).preparar();
    const servicios=await fuentes.servicios.consultarPropios();
    if(typeof valor==="string"){assert.equal(servicios.estado,"error");continue;}
    assert.equal(servicios.historia_servicios_disponible,valor===true);
    assert.equal(servicios.exportacion_servicios_disponible,true);
    assert.equal(peticion.headers.Prefer,"vec-personal-historia-servicios-v1, vec-personal-historia-relaciones-v1");
  }
});
