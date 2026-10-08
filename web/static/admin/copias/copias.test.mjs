import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { crearClienteCopias, normalizarCopia } from "./cliente-http.js";
import { normalizarConfiguracion, normalizarPropuesta, normalizarMetadatosRevision } from "./contratos.js";
import { resumenRevisable } from "./vista-recuperacion.js";
import { cargarTextos } from "../../comun/textos.js";

const fecha = "2026-10-01T12:00:00Z";
const copia = { copia_ref: "copia:sintetica", version: 8, tipo: "completa", estado: "valida", iniciada_en: fecha, compatibilidad: { estado: "compatible" } };
const politica = { formato: 1, referencia: "politica:sintetica", destino: "destino:sintetico", zona_horaria: "Europe/Madrid", fecha_inicial: "2026-10-01", cada_dias: 7, dias_semana: [1], ventana: { inicio: "02:00", fin: "03:00" }, retencion: { conservar_minimo: 2, edad_maxima_dias: 60, protegidas: ["copia:protegida"], borrado_permitido: false } };
const json = datos => new Response(JSON.stringify(datos), { headers: { "Content-Type": "application/json" } });
function metadatosSinteticos(p) {
  const version = { release_ref: "release:ensayo", app_version: "1.2.3", postgresql_version: "18.4", esquema_ref: "esquema:ensayo", descriptor_huella_sha256: "e".repeat(64) };
  return { propuesta_ref: p.propuesta_ref, propuesta_huella_sha256: p.huella_sha256, conjunto_ref: p.conjunto_ref,
    conjunto_huella_sha256: p.conjunto_huella_sha256, destino_ref: p.destino_ref, preimagen_sha256: p.preimagen_sha256,
    fecha_copia: fecha, perdida_desde: fecha, observada_en: fecha, actual: version, resultante: { ...version, app_version: "1.2.2" }, compatibilidad: { estado: "compatible", razones: ["api.admin.copias.compatibilidad.comprobacion_conjunto_compatible"] } };
}


test("transporte fixed-origin no-store y cuerpos sin identidad ni extras", async () => {
  const llamadas = [];
  const c = crearClienteCopias({ fetchImpl: async (url, options) => {
    llamadas.push({ url, options });
    return url.endsWith("/lanzamientos") ? json({ recibo: { operacion_ref: "operacion:sintetica", recibo_ref: "recibo:sintetico", recurso_ref: copia.copia_ref, version: 8, estado: "solicitada", registrado_en: fecha } }) : json({ version: 8, copias: [copia] });
  } });
  assert.equal((await c.listar()).version, 8);
  await c.lanzar({ operacion_ref: "operacion:sintetica", version_esperada: 8, actor: "no_debe_enviarse", tipo: "completa" });
  for (const { url, options } of llamadas) {
    assert.ok(url.startsWith("/api/admin/copias/v1"));
    for (const [k, v] of Object.entries({ credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" })) assert.equal(options[k], v);
    assert.deepEqual(Object.keys(options.headers).filter(k => /identity|authorization|cookie/i.test(k)), []);
  }
  assert.deepEqual(JSON.parse(llamadas[1].options.body), { operacion_ref: "operacion:sintetica", version_esperada: 8, tipo: "completa" });
});
test("referencias/rutas alteradas e incremental no llegan a red", async () => {
  let count = 0; const c = crearClienteCopias({ fetchImpl: async () => { count++; return json({}); } });
  for (const ref of ["../secret", "https://externo/", "copia:?cookie", "copia:\nheader"]) await assert.rejects(c.detalle(ref));
  await assert.rejects(c.lanzar({ operacion_ref: "operacion:test", version_esperada: 1, tipo: "incremental" }));
  assert.equal(count, 0);
});
test("rechaza falso éxito, conflicto y mensajes backend no se muestran", async () => {
  for (const [response, codigo] of [[new Response('{"recibo":{}}', { status: 200, headers: { "Content-Type": "application/json" } }), "respuesta_invalida"], [new Response('secreto servidor', { status: 403 }), "denegado"], [new Response('', { status: 409 }), "conflicto"], [new Response('<html>error</html>', { status: 200 }), "respuesta_invalida"]]) {
    const c = crearClienteCopias({ fetchImpl: async () => response });
    await assert.rejects(c.lanzar({ operacion_ref: "operacion:test", version_esperada: 8 }), e => e.codigo === codigo && !e.message.includes("secreto"));
  }
});
test("detalle descarta paths/tablas/roles/secrets y no infiere verificación", () => {
  const d = normalizarCopia({ ...copia, estado: "verificando", path: "/privado", roles: ["superuser"], secretos: "oculto", componentes: [{ componente: "documentos", estado: "completo", ruta: "/material" }] });
  assert.equal(d.estado, "verificando"); assert.equal(d.path, undefined); assert.equal(d.roles, undefined); assert.equal(d.secretos, undefined); assert.equal(d.componentes[0].ruta, undefined);
});
test("configuración respeta versión, protegidas y doble control por defecto", async () => {
  const c = normalizarConfiguracion({ version: 12, politica });
  assert.equal(c.politica.retencion.doble_control, true); assert.deepEqual(c.politica.retencion.protegidas, ["copia:protegida"]);
  for (const invalida of [{ ...politica, cada_dias: 0 }, { ...politica, cada_dias: 367 }, { ...politica, ventana: { inicio: "23:00", fin: "02:00" } }, { ...politica, zona_horaria: "inventada" }]) assert.throws(() => normalizarConfiguracion({ version: 12, politica: invalida }));
  let cuerpo; const cliente = crearClienteCopias({ fetchImpl: async (_, opciones) => { cuerpo = JSON.parse(opciones.body); return json({ recibo: { operacion_ref: "operacion:config", recibo_ref: "recibo:sintetico", recurso_ref: "politica:sintetica", version: 13, estado: "registrada", registrado_en: fecha } }); } });
  await cliente.guardarConfiguracion("retencion", { operacion_ref: "operacion:config", version_esperada: 12, politica, identidad: "ignorar" });
  assert.equal(cuerpo.version_esperada, 12); assert.equal(cuerpo.identidad, undefined); assert.equal(cuerpo.politica.retencion.doble_control, true);
});
test("no aprueba un resumen sin preimagen, segunda persona o copia previa", () => {
  const propuesta = { propuesta_ref: "propuesta:sintetica", conjunto_ref: copia.copia_ref, conjunto_huella_sha256: "d".repeat(64), destino_ref: "destino:sintetico", motivo_ref: "motivo:ensayo", ventana_ref: "ventana:ensayo", politica_ref: "politica:sintetica", preimagen_sha256: "a".repeat(64), politica_huella_sha256: "b".repeat(64), estado: "propuesta", version: 1, huella_sha256: "c".repeat(64), caduca_en: "2099-10-01T12:00:00Z", ventana_inicio: fecha, ventana_fin: "2026-10-01T13:00:00Z", doble_control: true, copia_previa_requerida: true };
  propuesta.metadatos_revision = metadatosSinteticos(propuesta);
  const opciones = { destinos: [{ ref: propuesta.destino_ref, clave_i18n: "api.admin.copias.opcion.destino_local_aislado" }], motivos: [{ ref: propuesta.motivo_ref, clave_i18n: "api.admin.copias.opcion.motivo_ensayo" }], ventanas: [{ ref: propuesta.ventana_ref, clave_i18n: "api.admin.copias.opcion.ventana_programada" }] };
  assert.equal(resumenRevisable(normalizarPropuesta(propuesta), opciones), true);
  for (const patch of [{ doble_control: false }, { copia_previa_requerida: false }, { preimagen_sha256: "" }, { caduca_en: fecha }]) assert.equal(resumenRevisable({ ...propuesta, ...patch }, opciones), false);
  assert.equal(resumenRevisable(propuesta, { ...opciones, destinos: [{ ref: propuesta.destino_ref, clave_i18n: "secreto.backend" }] }), false);
});
test("ambos catálogos completos usan traductor real y todas las claves", async () => {
  const es = JSON.parse(await readFile(new URL("../../textos/es/copias-admin.json", import.meta.url), "utf8"));
  const en = JSON.parse(await readFile(new URL("../../textos/en/copias-admin.json", import.meta.url), "utf8"));
  for (const grupo of Object.keys(es)) assert.deepEqual(Object.keys(en[grupo]), Object.keys(es[grupo]));
  const t = await cargarTextos("copias-admin", { idioma: "en", porDefecto: "es" });
  assert.deepEqual(t.faltantes, []); assert.equal(t.traducir("general.titulo"), "Backups");
});
test("un recibo de otra operación nunca confirma esta solicitud", async () => {
  const c = crearClienteCopias({ fetchImpl: async () => json({ recibo: { recibo_ref: "recibo:otro", operacion_ref: "operacion:otra", recurso_ref: copia.copia_ref, version: 8, estado: "solicitada", registrado_en: fecha } }) });
  await assert.rejects(c.lanzar({ operacion_ref: "operacion:esperada", version_esperada: 8 }), e => e.codigo === "respuesta_invalida");
  const declarada = normalizarCopia({ ...copia, estado: "verificada_declarada" });
  assert.notEqual(declarada.estado, "valida");
});
test("una propuesta de otro destino no confirma la propuesta enviada", async () => {
  const solicitud = { operacion_ref: "operacion:propuesta", conjunto_ref: copia.copia_ref, destino_ref: "destino:previsto", motivo_ref: "motivo:ensayo", ventana_ref: "ventana:ensayo", ventana_inicio: fecha, ventana_fin: "2026-10-01T13:00:00Z", caduca_en: "2099-10-01T12:00:00Z" };
  const respuesta = { ...solicitud, propuesta_ref: "propuesta:test", destino_ref: "destino:otro", politica_ref: "politica:sintetica", conjunto_huella_sha256: "a".repeat(64), preimagen_sha256: "a".repeat(64), politica_huella_sha256: "a".repeat(64), estado: "propuesta", version: 1, huella_sha256: "a".repeat(64), doble_control: true, copia_previa_requerida: true };
  const cliente = crearClienteCopias({ fetchImpl: async () => json({ propuesta: respuesta }) });
  await assert.rejects(cliente.proponer(solicitud), e => e.codigo === "respuesta_invalida");
});

test("metadatos canónicos de segunda revisión exigen vínculos, fechas y versiones completas", () => {
  const propuesta = { propuesta_ref: "propuesta:sintetica", conjunto_ref: "copia:sintetica", destino_ref: "destino:sintetico", conjunto_huella_sha256: "a".repeat(64), preimagen_sha256: "b".repeat(64), huella_sha256: "c".repeat(64) };
  const m = metadatosSinteticos(propuesta);
  assert.equal(normalizarMetadatosRevision({ ...m, ruta_privada: "/secreto" }, propuesta).ruta_privada, undefined);
  for (const k of ["propuesta_ref", "propuesta_huella_sha256", "conjunto_ref", "conjunto_huella_sha256", "destino_ref", "preimagen_sha256"]) assert.throws(() => normalizarMetadatosRevision({ ...m, [k]: "alterado" }, propuesta));
  for (const k of ["fecha_copia", "perdida_desde", "observada_en"]) assert.throws(() => normalizarMetadatosRevision({ ...m, [k]: undefined }, propuesta));
  for (const grupo of ["actual", "resultante"]) for (const k of ["release_ref", "app_version", "postgresql_version", "esquema_ref", "descriptor_huella_sha256"]) assert.throws(() => normalizarMetadatosRevision({ ...m, [grupo]: { ...m[grupo], [k]: undefined } }, propuesta));
  assert.throws(() => normalizarMetadatosRevision({ ...m, actual: { ...m.actual, app_version: "/privado/binario" } }, propuesta));
});
test("un resumen histórico sin fecha pérdida o versiones nunca habilita segunda revisión", () => {
  const p = { propuesta_ref: "propuesta:sintetica", conjunto_ref: "copia:sintetica", destino_ref: "destino:sintetico", conjunto_huella_sha256: "a".repeat(64), preimagen_sha256: "b".repeat(64), huella_sha256: "c".repeat(64), doble_control: true, copia_previa_requerida: true, motivo_ref: "motivo:ensayo", ventana_ref: "ventana:ensayo", caduca_en: "2099-10-01T12:00:00Z", ventana_inicio: fecha, ventana_fin: "2026-10-01T13:00:00Z" };
  const opciones = { destinos: [{ ref: p.destino_ref, clave_i18n: "api.admin.copias.opcion.destino_local_aislado" }], motivos: [{ ref: p.motivo_ref, clave_i18n: "api.admin.copias.opcion.motivo_ensayo" }], ventanas: [{ ref: p.ventana_ref, clave_i18n: "api.admin.copias.opcion.ventana_programada" }] };
  assert.equal(resumenRevisable(p, opciones), false);
  const m = metadatosSinteticos(p); assert.equal(resumenRevisable({ ...p, metadatos_revision: m }, opciones), true);
  for (const patch of [{ perdida_desde: undefined }, { compatibilidad: { estado: "compatible", razones: [] } }, { compatibilidad: { estado: "compatible", razones: ["api.admin.copias.compatibilidad.no_existente"] } }, { resultante: {} }, { fecha_copia: undefined }, { compatibilidad: { estado: "no_comprobable", razones: [] } }]) assert.equal(resumenRevisable({ ...p, metadatos_revision: { ...m, ...patch } }, opciones), false);
});
