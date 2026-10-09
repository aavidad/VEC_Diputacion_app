import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import test from "node:test";
import {
  RUTAS_INCORPORACIONES_CENTRO, crearClienteIncorporacionesCentro, montarIncorporacionesCentro, validarBandejaIncorporaciones,
  validarSolicitudConfirmacionCentro,
  periodoSolicitadoVisible,
} from "./incorporaciones-centro.js";

const fila = (extra = {}) => ({ peticion_ref: "peticion:centro:1", expediente_ref: "expediente:ct:1", numero_visible: "2026/B-124", version: 7,
  fase: "nombramiento", estado: "en_curso", modalidad_clave: "sustitucion", categoria_ref: "cat:1", periodo: { inicio: "2026-09-01", fin: "2026-12-31" },
  confirmacion: null, documento_exigido: "toma_posesion", ...extra });
const bandeja = (extra = {}) => ({ esquema: "vec.contratacion-temporal.incorporaciones-centro.v1", expedientes: [fila()], puede_confirmar: true, limite: 50, ...extra });
const solicitud = { clave_idempotencia: "7c9e6679-7425-40de-944b-e07fc1f90ae7", peticion_ref: "peticion:centro:1", expediente_ref: "expediente:ct:1",
  fecha_incorporacion: "2026-09-01", documento_ref: "registro:centro-520:2026/15", documento_sha256: "f".repeat(64) };
const esperar = () => new Promise((r) => setImmediate(r));

test("la solicitud y la bandeja se validan: nada ajeno, fechas reales y no futuras", () => {
  assert.equal(validarSolicitudConfirmacionCentro({ ...solicitud }, "2026-09-26").fecha_incorporacion, "2026-09-01");
  assert.throws(() => validarSolicitudConfirmacionCentro({ ...solicitud, fecha_incorporacion: "2026-09-27" }, "2026-09-26"), TypeError, "futura");
  assert.throws(() => validarSolicitudConfirmacionCentro({ ...solicitud, fecha_incorporacion: "2026-02-30" }), TypeError);
  assert.throws(() => validarSolicitudConfirmacionCentro({ ...solicitud, fecha_incorporacion: "2026-09-01T00:00:00Z" }), TypeError);
  assert.throws(() => validarSolicitudConfirmacionCentro({ ...solicitud, documento_sha256: "0".repeat(64) }), TypeError);
  assert.throws(() => validarSolicitudConfirmacionCentro({ ...solicitud, actor_ref: "x" }), TypeError);
  assert.equal(validarBandejaIncorporaciones(bandeja()).expedientes.length, 1);
  assert.throws(() => validarBandejaIncorporaciones(bandeja({ esquema: "otro" })), TypeError);
  assert.throws(() => validarBandejaIncorporaciones(bandeja({ expedientes: [fila({ expediente_ref: "x" })] })), TypeError);
});

test("el periodo solicitado histórico conserva días civiles sin inferir incorporación ni fin", () => {
  const inicio = "2 de noviembre de 2026";
  const fin = "31 de diciembre de 2026";
  assert.equal(periodoSolicitadoVisible({ inicio: "2026-11-02T00:00:00Z", fin: "2026-12-31T00:00:00Z" }), `${inicio} — ${fin}`);
  assert.equal(periodoSolicitadoVisible({ inicio: "2026-11-02", fin: "2026-12-31" }), `${inicio} — ${fin}`);
  assert.equal(periodoSolicitadoVisible({ inicio: "2028-02-29T00:00:00.000+00:00", causa_fin: "reincorporacion_titular" }), "29 de febrero de 2028 — —");
  for (const valor of ["2026-02-30", "2027-02-29T00:00:00Z", "2026-11-02T00:00:00+01:00", "2026-11-02T01:00:00Z", "2026-11-02T00:00:00Zextra"]) {
    assert.equal(periodoSolicitadoVisible({ inicio: valor, fin: "2026-12-31" }), `— — ${fin}`, valor);
  }
});

test("el periodo solicitado histórico respeta el idioma inglés elegido", () => {
  const modulo = new URL("./incorporaciones-centro.js", import.meta.url).href;
  const script = `globalThis.location={href:"http://local/peticiones-centro/?lang=en",search:"?lang=en"};
    const {periodoSolicitadoVisible}=await import(${JSON.stringify(modulo)});
    process.stdout.write(periodoSolicitadoVisible({inicio:"2026-11-02T00:00:00Z",fin:"2026-12-31T00:00:00Z"}));`;
  const salida = execFileSync(process.execPath, ["--input-type=module", "-e", script], { encoding: "utf8" });
  assert.equal(salida, "2 November 2026 — 31 December 2026");
});

test("el cliente pide solo rutas fijas del mismo origen, sin caché, redirecciones ni referente", async () => {
  const llamadas = [];
  const fetchFalso = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    const cuerpo = opciones.method === "POST"
      ? { data: { recibo_ref: "recibo:c:1", expediente_ref: "expediente:ct:1", fecha_incorporacion: "2026-09-01", estado_local: "registrado" } }
      : { data: bandeja() };
    return { ok: true, status: opciones.method === "POST" ? 201 : 200, text: async () => JSON.stringify(cuerpo) };
  };
  const cliente = crearClienteIncorporacionesCentro(fetchFalso);
  await cliente.bandeja();
  await cliente.confirmar({ ...solicitud }, "2026-09-26");
  assert.deepEqual(llamadas.map((l) => l.ruta), [RUTAS_INCORPORACIONES_CENTRO.bandeja, RUTAS_INCORPORACIONES_CENTRO.confirmaciones]);
  for (const { opciones } of llamadas) {
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.mode, "same-origin");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.redirect, "error");
    assert.equal(opciones.referrerPolicy, "no-referrer");
  }
  assert.deepEqual(Object.keys(JSON.parse(llamadas[1].opciones.body)).sort(), Object.keys(solicitud).sort());
});

function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", hidden: true, eventos, ownerDocument: { addEventListener() {} },
    addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}

test("la sección se oculta si el servidor no la compone y lista con su documento exigido", async () => {
  const oculto = contenedorFalso();
  montarIncorporacionesCentro({ contenedor: oculto, cliente: { bandeja: async () => { throw Object.assign(new Error("x"), { estado: 404 }); } } });
  await esperar();
  assert.equal(oculto.hidden, true);
  assert.equal(oculto.innerHTML, "");
  const c = contenedorFalso();
  const desmontar = montarIncorporacionesCentro({ contenedor: c, cliente: { bandeja: async () => validarBandejaIncorporaciones(bandeja({ expedientes: [fila(),
    fila({ expediente_ref: "expediente:ct:2", numero_visible: "2026/B-125", confirmacion: { fecha_incorporacion: "2026-09-02", documento_tipo: "contrato_firmado",
      documento_ref: "registro:x:1", recibo_ref: "recibo:c:2", registrada_en: "2026-09-02T08:00:00Z" } })] })) } });
  await esperar();
  assert.equal(c.hidden, false);
  assert.match(c.innerHTML, /2026\/B-124/u);
  assert.match(c.innerHTML, /data-ic-abrir="expediente:ct:1"/u);
  assert.match(c.innerHTML, /Confirmada el 2 de septiembre de 2026 con el contrato firmado/u);
  assert.doesNotMatch(c.innerHTML, /registro:x:1|recibo:c:2/u, "sin referencias internas en pantalla");
  c.eventos.get("click")({ target: { closest: () => ({ matches: () => false, dataset: { icAbrir: "expediente:ct:1" } }) } });
  assert.match(c.innerHTML, /data-ic-form="expediente:ct:1"/u);
  assert.match(c.innerHTML, /Documento que acredita la incorporación: la toma de posesión/u);
  assert.match(c.innerHTML, /type="file" data-huella-archivo/u);
  assert.match(c.innerHTML, /<input type="hidden" name="documento_sha256" value="" data-huella-valor>/u);
  desmontar();
  assert.equal(c.eventos.size, 0);
});

test("sin permiso de confirmar solo se ve el estado pendiente", async () => {
  const c = contenedorFalso();
  montarIncorporacionesCentro({ contenedor: c, cliente: { bandeja: async () => validarBandejaIncorporaciones(bandeja({ puede_confirmar: false })) } });
  await esperar();
  assert.doesNotMatch(c.innerHTML, /data-ic-abrir/u);
  assert.match(c.innerHTML, /Pendiente de confirmar/u);
});

test("la bandeja en tramitación muestra el periodo solicitado y conserva «Aún no procede»", async () => {
  const c = contenedorFalso();
  montarIncorporacionesCentro({ contenedor: c, cliente: { bandeja: async () => validarBandejaIncorporaciones(bandeja({ expedientes: [fila({
    fase: "solicitud", documento_exigido: "", periodo: { inicio: "2026-11-02T00:00:00Z", fin: "2026-12-31T00:00:00Z" },
  })] })) } });
  await esperar();
  assert.match(c.innerHTML, /2 de noviembre de 2026 — 31 de diciembre de 2026/u);
  assert.match(c.innerHTML, /En tramitación en RRHH/u);
  assert.match(c.innerHTML, /Aún no procede/u);
  assert.doesNotMatch(c.innerHTML, /data-ic-abrir/u);
});
