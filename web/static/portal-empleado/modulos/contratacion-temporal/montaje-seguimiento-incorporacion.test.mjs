import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { RUTA_SEGUIMIENTO_INCORPORACION } from "./cliente-http-seguimiento-incorporacion.js";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

const exp = "expediente:ct:original";
const vista = {
  esquema: "vec.contratacion-temporal.seguimiento-incorporacion.v2", alcance: "original_incorporacion",
  expediente_ref: exp, version_expediente: 8, recibo_incorporacion_ref: "recibo:original",
  seguimiento_ref: "seguimiento:original", version_seguimiento: 1, estado_clave: "vigente",
  periodo: { desde: "2027-01-01T00:00:00Z", hasta: "2027-03-31T00:00:00Z" },
  registrado_en: "2026-09-10T13:07:06.614186Z",
  actuaciones: [{ actuacion_ref: "actuacion:original", transicion_clave: "confirmar_incorporacion",
    estado_origen: "pendiente_incorporacion", estado_destino: "vigente", efectivo_en: "2027-01-01T00:00:00Z",
    registrada_en: "2026-09-10T13:07:06.614186Z", documentos: [{ tipo_clave: "resolucion_ejercicio", referencia: "documento:original" }] }],
  ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false,
};

test("seguimiento: transporte común usa GET único sin cuerpo ni identidad del navegador", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    llamadas++;
    assert.equal(ruta, `${RUTA_SEGUIMIENTO_INCORPORACION}?expediente_ref=${encodeURIComponent(exp)}`);
    assert.equal(opciones.method, "GET"); assert.equal(opciones.body, undefined);
    assert.equal(opciones.redirect, "error"); assert.equal(opciones.cache, "no-store");
    assert.deepEqual([...opciones.headers.keys()], ["accept"]);
    return new Response(JSON.stringify({ data: vista }), { headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  assert.deepEqual(await cliente.seguimientoIncorporacion.consultar(exp), vista);
  assert.equal(llamadas, 1);
});

test("seguimiento: errores CT y límite se conservan sin producir una vista", async () => {
  for (const [estado, codigo] of [[403, "acceso_denegado"], [409, "conflicto"], [503, "servicio_no_disponible"]]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => new Response(JSON.stringify({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.incorporacion_ejercicio.error.${codigo}`, correlacion_ref: "corr_no_disponible",
    } }), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } }) });
    await assert.rejects(cliente.seguimientoIncorporacion.consultar(exp), (e) => e.estado === estado && e.codigo === codigo && e.envelopeValido === true);
  }
  const excesivo = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => new Response("{}", {
    headers: { "content-type": "application/json; charset=utf-8", "content-length": String(512 * 1024 + 1) },
  }) });
  await assert.rejects(excesivo.seguimientoIncorporacion.consultar(exp));
});


const EXPEDIENTE_MONTAJE = "expediente:ct:montaje";
const RECIBO_RECUPERADO = Object.freeze({
  esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2",
  expediente_ref: EXPEDIENTE_MONTAJE, solicitud_personal_ref: "solicitud:personal:montaje",
  relacion_ref: "relacion:personal:montaje", recibo_ref: "recibo:ct:montaje",
  actuacion_ref: "actuacion:ct:montaje", registrada_en: "2026-09-10T13:07:06Z",
  periodo_incorporacion: { desde: "2027-01-01T00:00:00Z", hasta: "2027-03-31T00:00:00Z" },
  version_solicitud_personal: 7, version_actual_expediente: 8,
  seguimiento_ref: "seguimiento:ct:montaje", version_seguimiento_anterior: 0,
  version_seguimiento_resultante: 1, auditoria_ref: "auditoria:ct:montaje",
  outbox_ref: "outbox:ct:montaje", ejercicio_sintetico: true,
  firma_oficial: false, eficacia_administrativa: false,
});

function resumenMontaje() {
  return {
    expediente_ref: EXPEDIENTE_MONTAJE, numero_visible: "2026/CT-0001", version: 8,
    flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
    fase_clave: "nombramiento", estado_clave: "en_curso", centro_ref: "centro:001",
    categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z",
  };
}

function detalleMontaje() {
  return {
    esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen: resumenMontaje(),
    solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion", periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
    hitos: Array.from({ length: 8 }, (_, i) => ({
      secuencia: i + 1, version_expediente: i + 1, accion_clave: "registrar_solicitud",
      realizada_en: "2026-09-03T09:00:00Z", fase_destino: "solicitud", estado_origen: "en_curso", estado_destino: "en_curso",
    })),
  };
}

function raizMontajePrincipal() {
  const eventos = new Map(), vigentes = new Set(), hijos = [];
  let html = "", contenedor = null;
  const raiz = {
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor; vigentes.clear(); contenedor = null;
      if (valor.includes('<div data-ct-exp-incorporacion-ejercicio>')) {
        contenedor = {
          innerHTML: "", addEventListener() {}, removeEventListener() {},
          replaceChildren() { this.innerHTML = ""; }, append(nodo) { hijos.push(nodo); },
          ownerDocument: { createElement() {
            const eventosHijo = new Map();
            return {
              innerHTML: "", addEventListener: (tipo, fn) => eventosHijo.set(tipo, fn),
              removeEventListener: (tipo) => eventosHijo.delete(tipo), replaceChildren() { this.innerHTML = ""; },
              eventos: eventosHijo,
            };
          } },
        };
        vigentes.add(contenedor);
      }
    },
    contains(nodo) { return vigentes.has(nodo); },
    querySelector(selector) { return selector === "[data-ct-exp-incorporacion-ejercicio]" ? contenedor : null; },
    querySelectorAll() { return []; },
    addEventListener: (tipo, fn) => eventos.set(tipo, fn), removeEventListener: (tipo) => eventos.delete(tipo),
  };
  return {
    raiz, hijos,
    async abrir() {
      const control = { dataset: { ctExpAbrir: EXPEDIENTE_MONTAJE }, closest: (selector) => selector === "[data-ct-exp-abrir]" ? control : null };
      vigentes.add(control);
      await eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

const esperarMontaje = () => new Promise((resolver) => setImmediate(resolver));

test("seguimiento: el montaje principal recupera el recibo, presenta solo el panel original y lo destruye", async () => {
  const dom = raizMontajePrincipal(), lecturas = [];
  const clienteLectura = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    lecturas.push(ruta);
    const entrada = JSON.parse(opciones.body);
    const data = ruta.endsWith("/cuadro/consultas")
      ? { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-03T09:05:00Z", expedientes: [resumenMontaje()], hay_mas: false }
      : detalleMontaje(entrada.expediente_ref);
    return new Response(JSON.stringify({ data }), { headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteLectura });
  await fuente.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  const llamadas = [];
  const modulo = await montarModuloContratacionTemporal({
    raiz: dom.raiz, presentador, llamamiento: { cliente: {
      async prepararIncorporacionEjercicio(expedienteRef, opciones) {
        llamadas.push([expedienteRef, opciones.signal]);
        return { esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", expediente_ref: expedienteRef,
          version_actual_expediente: 8, preparacion: null, recibo: RECIBO_RECUPERADO };
      },
      confirmarIncorporacionEjercicio() { assert.fail("un recibo recuperado no confirma otra incorporación"); },
      seguimientoIncorporacion: { consultar() { assert.fail("el panel no consulta hasta un clic explícito"); } },
    } },
  });
  try {
    await dom.abrir(); await esperarMontaje(); await esperarMontaje();
    assert.equal(llamadas.length, 1);
    assert.equal(llamadas[0][0], EXPEDIENTE_MONTAJE);
    assert.ok(llamadas[0][1] instanceof AbortSignal);
    assert.equal(dom.hijos.length, 1);
    assert.match(dom.hijos[0].innerHTML, /Seguimiento de la incorporación/u);
    assert.match(dom.hijos[0].innerHTML, /recibo:ct:montaje/u);
    assert.match(dom.hijos[0].innerHTML, /No permite anotar, cerrar ni alterar el expediente/u);
    assert.doesNotMatch(dom.hijos[0].innerHTML, /firma oficial|eficacia administrativa/u);
  } finally {
    modulo.desmontar();
  }
  assert.equal(dom.hijos[0].innerHTML, "");
  assert.equal(dom.hijos[0].eventos.size, 0);
});
