import test from "node:test";
import assert from "node:assert/strict";
import {
  consultarOrigenContacto,
  crearControladorOrigenContacto,
  fechaCivilVisible,
  renderizarAvisosContactoEmision,
  renderizarOrigenContacto,
} from "./portal-bolsas-contacto-origen.js";
import { validarEmisionLlamamiento } from "./portal-llamamientos-contrato.js";

const respuesta = (status, cuerpo) => async () => ({ status, ok: status >= 200 && status < 300, json: async () => cuerpo });

test("la consulta pide la forma enmascarada y valida el origen", async () => {
  let pedida = "";
  const res = await consultarOrigenContacto("bolsa:1", "participacion:1", { fetchImpl: async (url, opciones) => {
    pedida = url;
    assert.equal(opciones.method, "GET");
    return respuesta(200, { data: { origen: { origen: "convoca", estado: "vencido", ultimo_dia: "2027-09-28", vigente_hasta: "2027-09-28T22:00:00Z", regla_ref: "r" } } })();
  } });
  assert.equal(pedida, "/api/vec/bolsa/bolsas/bolsa:1/candidatos/participacion:1/datos-contacto");
  assert.ok(!pedida.includes("ver=completo"), "nunca se pide el claro");
  assert.equal(res.ok, true);
  assert.equal(res.datos.origen.estado, "vencido");
  assert.deepEqual(await consultarOrigenContacto("b", "p", { fetchImpl: respuesta(404, {}) }), { ok: true, datos: { sin_contacto: true, origen: null } });
  assert.equal((await consultarOrigenContacto("b", "p", { fetchImpl: respuesta(403, {}) })).status, 403);
  const invalida = await consultarOrigenContacto("b", "p", { fetchImpl: respuesta(200, { data: { origen: { origen: "otro", estado: "vigente", ultimo_dia: "2027-09-28" } } }) });
  assert.equal(invalida.codigo, "respuesta_invalida");
  assert.equal((await consultarOrigenContacto("b", "p", { fetchImpl: async () => { throw new Error("red"); } })).codigo, "error_red");
});

test("la ficha distingue contacto propio, CONVOCA vigente y vencido", () => {
  assert.equal(fechaCivilVisible("2027-09-28"), "28/09/2027");
  assert.match(renderizarOrigenContacto({ estado: { carga: "cargando" } }), /aria-busy="true"/);
  assert.match(renderizarOrigenContacto({ estado: { carga: "listo", datos: { origen: null } } }), /Contacto propio/);
  const vigente = renderizarOrigenContacto({ estado: { carga: "listo", datos: { origen: { origen: "convoca", estado: "vigente", ultimo_dia: "2027-09-28" } } } });
  assert.match(vigente, /Origen CONVOCA/);
  assert.match(vigente, /Vigente hasta el 28\/09\/2027/);
  const vencido = renderizarOrigenContacto({ estado: { carga: "listo", datos: { origen: { origen: "convoca", estado: "vencido", ultimo_dia: "2027-09-28" } } } });
  assert.match(vencido, /estado-chip aviso">Contacto no confirmado/);
  assert.match(vencido, /vencido el 28\/09\/2027/);
  assert.match(renderizarOrigenContacto({ estado: { carga: "error", status: 403 } }), /role="alert">Sin permiso/);
  assert.match(renderizarOrigenContacto({ estado: { carga: "listo", datos: { sin_contacto: true, origen: null } } }), /Sin contacto registrado/);
});

test("los avisos de la emisión nombran a la persona y escapan el texto", () => {
  assert.equal(renderizarAvisosContactoEmision({ avisos: [] }), "");
  const html = renderizarAvisosContactoEmision({
    avisos: [
      { participacion_ref: "participacion:1", aviso: "contacto_origen_convoca_no_confirmado", ultimo_dia: "2027-09-28" },
      { participacion_ref: "participacion:2", aviso: "estado_contacto_no_disponible" },
    ],
    candidatos: [{ participacion_ref: "participacion:1", nombre_visible: "Ana <b>" }],
  });
  assert.match(html, /role="status"/);
  assert.match(html, /Ana &lt;b&gt;: contacto de origen CONVOCA sin confirmar desde el 28\/09\/2027/);
  assert.match(html, /participacion:2: no se pudo comprobar/);
});

test("el controlador descarta respuestas de una ficha ya cerrada", async () => {
  const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: null };
  let pintadas = 0;
  let liberar;
  const controlador = crearControladorOrigenContacto({ estado, renderizar: () => { pintadas += 1; }, consultar: () => new Promise((r) => { liberar = r; }) });
  const ficha = { candidato: { participacion_ref: "participacion:1" } };
  estado.modalFicha = ficha;
  const carga = controlador.cargar(ficha);
  assert.equal(ficha.contactoOrigen.carga, "cargando");
  estado.modalFicha = null;
  liberar({ ok: true, datos: { origen: null } });
  await carga;
  assert.equal(pintadas, 0);
  estado.modalFicha = ficha;
  const otra = controlador.cargar(ficha);
  liberar({ ok: false, status: 503 });
  await otra;
  assert.deepEqual(ficha.contactoOrigen, { carga: "error", status: 503 });
  assert.equal(pintadas, 1);
});

test("el contrato de emisión admite solo avisos de contacto válidos", () => {
  const base = () => ({
    llamamiento_ref: `llamamiento:${"a".repeat(64)}`, recibo_ref: `recibo:llamamiento:${"b".repeat(64)}`, bolsa_ref: "bolsa:1",
    estado: "emitido_pendiente_respuesta", participaciones: ["participacion:1"], emitido_en: "2027-10-01T08:00:00Z", reutilizada: false,
    configuracion: Object.fromEntries(["referencia", "descripcion", "categoria", "centro", "modalidad", "fecha_inicio", "plazo", "plantilla_version", "asunto", "cuerpo"].map((c) => [c, "valor"])),
  });
  assert.doesNotThrow(() => validarEmisionLlamamiento(base()));
  const conAviso = { ...base(), avisos_contacto: [{ participacion_ref: "participacion:1", aviso: "contacto_origen_convoca_no_confirmado", ultimo_dia: "2027-09-28" }] };
  assert.doesNotThrow(() => validarEmisionLlamamiento(conAviso));
  assert.throws(() => validarEmisionLlamamiento({ ...base(), avisos_contacto: [{ participacion_ref: "participacion:9", aviso: "contacto_origen_convoca_no_confirmado" }] }));
  assert.throws(() => validarEmisionLlamamiento({ ...base(), avisos_contacto: [{ participacion_ref: "participacion:1", aviso: "otro" }] }));
  assert.throws(() => validarEmisionLlamamiento({ ...base(), avisos_contacto: [{ participacion_ref: "participacion:1", aviso: "estado_contacto_no_disponible", ultimo_dia: "28/09/2027" }] }));
});
