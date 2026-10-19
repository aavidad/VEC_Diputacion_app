import assert from "node:assert/strict";
import test from "node:test";
import { validarConsultaB2ParaFichaGINPIX, validarFichaGINPIX } from "./contrato-ficha-ginpix.js?v=VEC_WEB_VERSION";
import { crearFichaGINPIXClienteHTTP, RUTA_FICHA_GINPIX, NOMBRE_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js?v=VEC_WEB_VERSION";
import { CLAVES_I18N_FICHA_GINPIX, montarFichaGINPIX } from "./ficha-ginpix.js?v=VEC_WEB_VERSION";
import { MENSAJES_CONTRATACION_TEMPORAL_ES } from "./i18n.js?v=VEC_WEB_VERSION";

const SHA = "a".repeat(64);
const expedienteRef = "expediente:ct:001";
const reciboRef = "recibo:incorporacion-personal-b2:001";
const consulta = () => ({
  esquema: "vec.contratacion-temporal.incorporacion-personal-b2.consulta.v1", expediente_ref: expedienteRef,
  version_expediente_actual: 9, estado: "incorporacion_confirmada", prerrequisitos: [],
  opciones: { vacantes: [], regimenes: [], modalidades: [], catalogo_clases_ocupacion: { ref: "", version: 0, huella_sha256: "" },
    clases_ocupacion: [], motivos: [], documentos: [], periodo: { desde: "", hasta: "", fuente_ref: "" } },
  plan: { plan_ref: "plan:b2:001", version: 1, sha256: SHA,
    intencion: { expediente_ref: expedienteRef, version_expediente: 8, puesto_ref: "puesto:001", plaza_ref: "plaza:001",
      version_plantilla_ref: "plantilla:001", version_rpt_ref: "rpt:001", regimen: { ref: "regimen:001", version: 1 },
      modalidad: { ref: "modalidad:001", version: 1 }, clase_ocupacion: "temporal", desde: "2026-10-01", hasta: "",
      motivo_clave: "incorporar", documento_ref: "documento:001", documento_sha256: SHA,
      clave_idempotencia: "10000000-0000-4000-8000-000000000001" } },
  recibo: { esquema: "vec.contratacion-temporal.incorporacion-personal-b2.recibo.v1", expediente_ref: expedienteRef,
    plan_ref: "plan:b2:001", plan_version: 1, recibo_ref: reciboRef, registrada_en: "2026-10-01T10:00:00Z",
    empleado_ref: "empleado:001", relacion_ref: "relacion:001", ocupacion_ref: "ocupacion:001",
    firma_oficial: false, eficacia_administrativa: false },
});
const ficha = () => ({
  esquema: "vec.dipgra.contratacion-temporal.ginpix.fichero.v1", version: 1,
  metadatos: { esquema_modelo: "vec.dipgra.contratacion-temporal.ginpix.modelo.v1",
    esquema_mapeo: "vec.dipgra.contratacion-temporal.ginpix.mapeo.v1",
    esquema_carga: "vec.dipgra.contratacion-temporal.ginpix.carga.v1", version_expediente: 9,
    expediente_ref: expedienteRef, incorporacion_ref: reciboRef, procedencia_modelo_ref: reciboRef,
    correlacion_ref: "solicitud:personal:001", idempotencia_ref: "10000000-0000-4000-8000-000000000002",
    huella_modelo_sha256: SHA, mapeo_ref: "mapeo:ct:ginpix:ficha-personal-b2", mapeo_version: 1,
    procedencia_mapeo_ref: "configuracion:ct:ginpix:ficha-trazabilidad-personal-b2:v1",
    huella_mapeo_sha256: SHA, huella_carga_sha256: SHA },
  campos: [
    { clave: "actuacion_ref", estado: "valor", valor: reciboRef },
    { clave: "expediente_ref", estado: "valor", valor: expedienteRef },
    { clave: "recibo_ref", estado: "valor", valor: "recibo:personal:alta:001" },
    { clave: "relacion_ref", estado: "valor", valor: "relacion:001" },
    { clave: "seguimiento_ref", estado: "valor", valor: "recibo:personal:ejecucion:001" },
    { clave: "solicitud_personal_ref", estado: "valor", valor: "solicitud:personal:001" },
  ],
});

function dom() {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (k, v) => eventos.set(k, v), removeEventListener() {},
    replaceChildren() { this.innerHTML = ""; } };
  return { raiz, eventos };
}

test("ficha: exige consulta B2 confirmada y conserva origen, versión y alta originales", () => {
  assert.equal(validarFichaGINPIX(ficha(), consulta()).metadatos.incorporacion_ref, reciboRef);
  for (const cambio of [
    { version_expediente: 8 }, { incorporacion_ref: "recibo:ajeno" }, { procedencia_modelo_ref: "recibo:ajeno" },
    { mapeo_ref: "mapeo:ct:ginpix:ficha-incorporacion-ejercicio" },
  ]) assert.throws(() => validarFichaGINPIX({ ...ficha(), metadatos: { ...ficha().metadatos, ...cambio } }, consulta()));
  const sinAlta = consulta(); sinAlta.recibo = null; sinAlta.estado = "plan_preparado";
  assert.throws(() => validarConsultaB2ParaFichaGINPIX(sinAlta));
  assert.throws(() => validarFichaGINPIX(ficha(), { ...consulta(), recibo: { ...consulta().recibo, relacion_ref: "relacion:otra" } }));
  assert.throws(() => validarFichaGINPIX({ ...ficha(), campos: ficha().campos.map((v) => v.clave === "relacion_ref" ? { ...v, valor: "relacion:otra" } : v) }, consulta()));
});

test("cliente: descarga bytes originales sólo con recibo B2 validado", async () => {
  let orden; const contenido = new Uint8Array([123, 125]);
  const cliente = crearFichaGINPIXClienteHTTP({ validarOpciones: (x) => x ?? {}, descargar: async (x) => {
    orden = x; return { tipo: "application/json", nombre: NOMBRE_FICHA_GINPIX, json: ficha(), contenido };
  } });
  assert.equal((await cliente.descargarFichaGINPIX(consulta())).contenido, contenido);
  assert.equal(orden.ruta, `${RUTA_FICHA_GINPIX}?expediente_ref=${encodeURIComponent(expedienteRef)}`);
  assert.equal(orden.metodo, "GET"); assert.equal(orden.maximoRespuesta, 512 * 1024);
  await assert.rejects(cliente.descargarFichaGINPIX(null));
});

test("controlador: muestra alta B2 y descarga sin afirmar envío a GINPIX", async () => {
  for (const clave of CLAVES_I18N_FICHA_GINPIX) assert.equal(typeof MENSAJES_CONTRATACION_TEMPORAL_ES[clave], "string");
  const { raiz, eventos } = dom(); let descargado;
  montarFichaGINPIX({ raiz, consulta: consulta(), resumen: { centro: "Centro", categoria: "Administrativo" },
    cliente: { descargarFichaGINPIX: async () => ({ json: ficha(), contenido: new Uint8Array([123, 125]) }) },
    descargarArchivo: async (archivo, nombre) => { descargado = [archivo, nombre]; } });
  assert.match(raiz.innerHTML, /Recibo de incorporación/u);
  assert.match(raiz.innerHTML, /ficha-ginpix-personal\.json/u);
  assert.match(raiz.innerHTML, /Sin fecha de fin/u);
  assert.doesNotMatch(raiz.innerHTML, /enviado a GINPIX|ejercicio/u);
  await eventos.get("click")({ target: { matches: () => true } });
  assert.equal(descargado[1], NOMBRE_FICHA_GINPIX);
  assert.ok(descargado[0].contenido instanceof Uint8Array);
});

test("controlador: conflicto y denegación muestran respuesta nominal sin descargar", async () => {
  for (const [estado, frase] of [[409, "El alta ha cambiado"], [403, "No tiene permiso"]]) {
    const { raiz, eventos } = dom(); let descargas = 0;
    montarFichaGINPIX({ raiz, consulta: consulta(), cliente: { descargarFichaGINPIX: async () => {
      throw Object.assign(new Error("dato privado"), { estado, envelopeValido: true });
    } }, descargarArchivo: () => { descargas++; } });
    await eventos.get("click")({ target: { matches: () => true } });
    assert.equal(descargas, 0);
    assert.doesNotMatch(raiz.innerHTML, /dato privado/u);
    if (estado === 403) assert.doesNotMatch(raiz.innerHTML, /recibo:incorporacion-personal-b2/u);
    else assert.match(raiz.innerHTML, /Ficha GINPIX/u);
    assert.match(MENSAJES_CONTRATACION_TEMPORAL_ES[estado === 409 ? "ficha_ginpix_conflicto" : "ficha_ginpix_denegada"], new RegExp(frase, "u"));
  }
});
