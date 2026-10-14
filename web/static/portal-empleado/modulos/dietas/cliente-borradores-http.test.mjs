import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearClienteBorradoresDietasHTTP, ErrorClienteBorradoresDietas } from "./cliente-borradores-http.js";
const sufijo = "1234567890123456789012";
const solicitud = Object.freeze({ clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000", fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Reunión de coordinación", codigos_ruta: ["granada", "motril"] });
const item = Object.freeze({ comision: { referencia: `dco_${sufijo}`, estado: "borrador", fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Reunión de coordinación", codigos_ruta: ["granada", "motril"], relacion_ref: `rel_${sufijo}` }, recibo: { referencia: `rcd_${sufijo}`, version: 1, registrado_en: "2026-09-20T10:00:00.000000Z", repeticion: false } });
function respuesta(cuerpo, estado = 200, headers = {}) { return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8", ...headers } }); }
test("crea un borrador same-origin sin cookies ni identidad libre y conserva su recibo", async () => { const llamadas = []; const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta(item, 201); } }); const resultado = await cliente.crear(solicitud); assert.equal(resultado.recibo.referencia, item.recibo.referencia); assert.equal(llamadas[0].ruta, "/api/vec/dietas/comisiones"); assert.equal(llamadas[0].opciones.credentials, "same-origin"); assert.equal(llamadas[0].opciones.mode, "same-origin"); assert.deepEqual(JSON.parse(llamadas[0].opciones.body), solicitud); await assert.rejects(() => cliente.crear({ ...solicitud, persona_ref: "per_ajena" }), /solicitud/u); });
test("recupera colección y detalle propios con el contrato de rutas cerrado", async () => { const llamadas = []; const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta(ruta.includes("?") ? { items: [item] } : item); } }); const pagina = await cliente.listar({ limit: 20, cursor: "siguiente" }); const detalle = await cliente.obtener(item.comision.referencia); assert.equal(pagina.items.length, 1); assert.equal(detalle.comision.referencia, item.comision.referencia); assert.equal(llamadas[0].ruta, "/api/vec/dietas/comisiones?limit=20&cursor=siguiente"); assert.equal(llamadas[1].ruta, `/api/vec/dietas/comisiones/${item.comision.referencia}`); });
test("reduce consultas por una relación opaca y no admite identidad ni permisos del navegador", async () => { const llamadas = []; const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta) => { llamadas.push(ruta); return respuesta(ruta.includes("comisiones?") ? { items: [item] } : item); } }); await cliente.listar({ relacion_ref: item.comision.relacion_ref }); await cliente.obtener(item.comision.referencia, { relacion_ref: item.comision.relacion_ref }); assert.equal(llamadas[0], `/api/vec/dietas/comisiones?limit=20&relacion_ref=${item.comision.relacion_ref}`); assert.equal(llamadas[1], `/api/vec/dietas/comisiones/${item.comision.referencia}?relacion_ref=${item.comision.relacion_ref}`); await assert.rejects(() => cliente.listar({ persona_ref: "per_ajena" }), /consulta/u); await assert.rejects(() => cliente.obtener(item.comision.referencia, { actor_ref: "act_ajeno" }), (error) => error.codigo === "opciones_no_validas"); });
test("distingue autenticación, denegación, conflicto, validación y resultado incierto", async () => { for (const [estado, codigo] of [[401, "autenticacion_requerida"], [403, "acceso_denegado"], [409, "conflicto_idempotencia"], [422, "relacion_no_valida"], [503, "resultado_incierto"]]) { const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta({ error: `dietas.error.${codigo}` }, estado) }); await assert.rejects(() => cliente.crear(solicitud), (error) => error instanceof ErrorClienteBorradoresDietas && error.estado === estado && error.codigo === codigo && error.resultadoIndeterminado === (estado === 503)); } });
test("la cancelación se propaga al fetch y no acepta una respuesta tardía", async () => { const controlador = new AbortController(); let recibido; const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: (_ruta, opciones) => new Promise((resolver) => { recibido = opciones.signal; opciones.signal.addEventListener("abort", () => resolver(respuesta(item, 201)), { once: true }); }) }); const pendiente = cliente.crear(solicitud, { signal: controlador.signal }); controlador.abort(); await assert.rejects(() => pendiente, (error) => error instanceof ErrorClienteBorradoresDietas && error.codigo === "operacion_abortada"); assert.equal(recibido.aborted, true); });
test("rechaza cuerpos no incrementales, excesivos y sobres incompatibles", async () => { const clienteSinStream = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => ({ status: 200, ok: true, redirected: false, headers: new Headers({ "content-type": "application/json; charset=utf-8" }), body: null }) }); await assert.rejects(() => clienteSinStream.listar(), (error) => error.codigo === "respuesta_no_incremental"); const clienteExcesivo = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta({ items: [] }, 200, { "content-length": String(128 * 1024 + 1) }) }); await assert.rejects(() => clienteExcesivo.listar(), (error) => error.codigo === "respuesta_excesiva"); const clienteSobre = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta({ items: [{ ...item, actor_ref: "no" }] }) }); await assert.rejects(() => clienteSobre.listar(), /resultado de Dietas incompatible/u); });
test("conserva la clave de reintento si un 503 no trae cuerpo compatible", async () => { const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => new Response("no-json", { status: 503, headers: { "Content-Type": "application/json; charset=utf-8" } }) }); await assert.rejects(() => cliente.crear(solicitud), (error) => error.codigo === "json_no_valido" && error.estado === 503 && error.resultadoIndeterminado); });
test("un POST aceptado sin recibo válido mantiene incierta la operación para repetir su clave", async () => {
  const respuestas = [
    new Response("no-json", { status: 201, headers: { "Content-Type": "application/json; charset=utf-8" } }),
    respuesta({ comision: item.comision, recibo: { ...item.recibo, registrado_en: "sin_fecha" } }, 200),
    new Response(null, { status: 204, headers: { "Content-Type": "application/json; charset=utf-8" } }),
    respuesta({ error: "dietas.error.no_disponible" }, 500),
  ];
  for (const [indice, respuestaHTTP] of respuestas.entries()) {
    const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuestaHTTP });
    await assert.rejects(() => cliente.crear(solicitud), (error) =>
      error instanceof ErrorClienteBorradoresDietas &&
      error.estado === respuestaHTTP.status &&
      error.resultadoIndeterminado === true &&
      error.codigo === ["json_no_valido", "respuesta_incompatible", "respuesta_no_incremental", "respuesta_rechazada"][indice]);
  }
});
test("conserva el orden origen-destino y rechaza un tramo alterado en el recibo", async () => {
  const calculo = { procedencia:"osrm_interno", motor:"OSRM", version_grafo:"grafo-sintetico-v1", version_tarifa:"provisional:rd462:20260923", rotulo:"PROVISIONAL · pendiente de confirmación por RRHH", hora_inicio:"08:00", hora_fin:"18:00", kilometros:"12.0000", eur_por_km:"0.2600", importe_kilometraje_centimos:312, tramos_ruta:[{origen_codigo:"18087",destino_codigo:"18003",kilometros:"12.0000"}], opciones_dieta:[1,2,3].map((grupo)=>({grupo,calculo:{tramos:[],total_maximo_orientativo_centimos:0,version_tarifa_ref:"provisional:rd462:20260923",rotulo:"PROVISIONAL · pendiente de confirmación por RRHH"}})) };
  const guardado={...item,comision:{...item.comision,codigos_ruta:["18087","18003"],calculo}};
  const cliente=crearClienteBorradoresDietasHTTP({fetchImpl:async()=>respuesta(guardado,201)});
  const obtenido=await cliente.crear({...solicitud,codigos_ruta:["18087","18003"],hora_inicio:"08:00",hora_fin:"18:00"});
  assert.equal(obtenido.comision.calculo.importe_kilometraje_centimos,312);
  const adulterado={...guardado,comision:{...guardado.comision,calculo:{...calculo,tramos_ruta:[{...calculo.tramos_ruta[0],destino_codigo:"18087"}]}}};
  await assert.rejects(()=>crearClienteBorradoresDietasHTTP({fetchImpl:async()=>respuesta(adulterado,201)}).crear({...solicitud,codigos_ruta:["18087","18003"]}),
    (error) => error.codigo === "respuesta_incompatible" && error.estado === 201 && error.resultadoIndeterminado);
});

test("acepta alta, lectura y edición con tarifa publicada y conserva el documento provisional histórico", async () => {
  const vectores = JSON.parse(await readFile(new URL("../../../../../internal/modules/dietas/adapters/httpinterno/testdata/documento_propio_v2.json", import.meta.url), "utf8"));
  const historico = vectores.find((vector) => vector.nombre === "en_correccion").resultado;
  const publicada = structuredClone(historico); const calculo = publicada.comision.calculo;
  const version = "tarifa:dipgra:2026:art30:v1"; const rotulo = "dietas_tarifa_publicada";
  calculo.version_tarifa = version; calculo.rotulo = rotulo; calculo.regla_ref = "regla:dipgra:2026:devengo:v1";
  for (const opcion of calculo.opciones_dieta) {
    opcion.calculo.version_tarifa_ref = version; opcion.calculo.rotulo = rotulo; opcion.calculo.regla_ref = calculo.regla_ref;
    for (const tramo of opcion.calculo.tramos) { tramo.version_tarifa_ref = version; tramo.rotulo = rotulo; }
  }
  publicada.comision.documento.version_tarifa_aceptada = version;
  for (const linea of publicada.comision.documento.lineas) {
    if (linea.tipo === "dieta" || linea.tipo === "kilometraje") { linea.version_tarifa_ref = version; linea.rotulo = rotulo; }
  }
  const crearCliente = (resultado, estado = 200) => crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta(resultado, estado) });
  assert.equal((await crearCliente(historico).obtener(historico.comision.referencia)).comision.calculo.version_tarifa,
    "provisional:rd462:20260923");
  const provisionalConClavePublicada = structuredClone(historico);
  provisionalConClavePublicada.comision.calculo.rotulo = "dietas_tarifa_publicada";
  await assert.rejects(() => crearCliente(provisionalConClavePublicada, 201).crear(solicitud),
    (error) => error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado);
  assert.equal((await crearCliente(publicada, 201).crear(solicitud)).comision.calculo.version_tarifa, version);
  assert.equal((await crearCliente(publicada).obtener(publicada.comision.referencia)).comision.documento.version_tarifa_aceptada, version);
  const entradaEditar = { clave_idempotencia: solicitud.clave_idempotencia, version_esperada: 4,
    relacion_ref: publicada.comision.relacion_ref, fecha_inicio: publicada.comision.fecha_inicio,
    fecha_fin: publicada.comision.fecha_fin, motivo: publicada.comision.motivo,
    codigos_ruta: publicada.comision.codigos_ruta, vehiculo_propio: true,
    rutas: publicada.comision.rutas, otros: [], tramos_aceptados: publicada.comision.documento.tramos_aceptados,
    version_tarifa_aceptada: version };
  assert.equal((await crearCliente(publicada).editar(publicada.comision.referencia, entradaEditar)).recibo.version, 5);
  await assert.rejects(() => crearCliente(publicada).editar(publicada.comision.referencia,
    { ...entradaEditar, version_tarifa_aceptada: "tarifa:Provisional:v1" }), TypeError);

  for (const [caso, alterar] of [
    ["prefijo libre", (r) => { r.comision.calculo.version_tarifa = "libre:dipgra:2026"; }],
    ["provisional encubierta", (r) => { r.comision.calculo.version_tarifa = "tarifa:Provisional:v1"; }],
    ["rótulo provisional", (r) => { r.comision.calculo.rotulo = "PROVISIONAL · pendiente de confirmación por RRHH"; }],
    ["regla ajena", (r) => { r.comision.calculo.regla_ref = "provisional:regla:nacional-ordinaria:20260923"; }],
    ["huella corta", (r) => { r.comision.calculo.regla_huella_sha256 = "abc"; }],
    ["opción mezclada", (r) => { r.comision.calculo.opciones_dieta[0].calculo.version_tarifa_ref = "provisional:rd462:20260923"; }],
    ["tramo mezclado", (r) => { r.comision.calculo.opciones_dieta[1].calculo.tramos[0].version_tarifa_ref = "provisional:rd462:20260923"; }],
    ["documento mezclado", (r) => { r.comision.documento.version_tarifa_aceptada = "provisional:rd462:20260923"; }],
    ["línea de dieta mezclada", (r) => { r.comision.documento.lineas.find((l) => l.tipo === "dieta").version_tarifa_ref = "provisional:rd462:20260923"; }],
    ["línea de kilometraje mezclada", (r) => { r.comision.documento.lineas.find((l) => l.tipo === "kilometraje").version_tarifa_ref = "provisional:rd462:20260923"; }],
  ]) {
    const alterada = structuredClone(publicada); alterar(alterada);
    await assert.rejects(() => crearCliente(alterada, 201).crear(solicitud),
      (error) => error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado, caso);
  }
});
