import test from "node:test";
import assert from "node:assert/strict";
import {
  consultarSanciones, crearControladorSanciones, registrarRecursoSancion, registrarSancion,
  renderizarSanciones, rutaRecursoSancion, rutaSanciones, MENSAJES_SANCIONES_ES,
} from "./portal-bolsas-sanciones.js";

const SHA = "a".repeat(64);
const item = {
  sancion_ref: "sancion:" + "c".repeat(64), consecuencia: "b24.sancion.suspension", consecuencia_etiqueta: "Suspensión de seis meses",
  efecto: "pausar", causa: "No se presentó <b>", fecha_notificacion: "2026-09-20",
  resolucion: { referencia: "registro:2026/1", sha256: SHA }, resuelta_por: "persona:jefatura", regla_ref: "vec.bolsa.reglas:1:b24.sancion.suspension",
  suspension_hasta: "2027-03-20", recurso: { vence: "2026-10-20", regla_ref: "r", estado: "interpuesto", eventos: [] },
  situacion_desde: "2026-09-25T09:00:00Z", actor: "per_x", registrada_en: "2026-09-25T09:00:00Z",
};
const datos = {
  esquema: "vec.bolsa.rrhh.sanciones.v1", catalogo_disponible: true, estados_recurso: ["interpuesto", "desestimado"],
  consecuencias: [{ clave: "b24.sancion.suspension", etiqueta: "Suspensión", efecto: "pausar", articulo: "", ejemplo: true },
    { clave: "b24.sancion.baja_sin_contacto", etiqueta: "Baja", efecto: "excluir", articulo: "art. 11.1.a", ejemplo: false }],
  items: [item],
};
const respuesta = (status, cuerpo) => async () => ({ ok: status < 400, status, json: async () => cuerpo });

test("B24 rutas con segmentos codificados", () => {
  assert.equal(rutaSanciones("bolsa:1", "part:2"), "/api/vec/bolsa/bolsas/bolsa:1/candidatos/part:2/sanciones");
  assert.equal(rutaRecursoSancion("b", "p", "sancion:x"), "/api/vec/bolsa/bolsas/b/candidatos/p/sanciones/sancion:x/recursos");
});

test("B24 consulta valida el contrato", async () => {
  const bien = await consultarSanciones("b", "p", { fetchImpl: respuesta(200, { data: datos }) });
  assert.equal(bien.ok, true);
  const mal = await consultarSanciones("b", "p", { fetchImpl: respuesta(200, { data: { ...datos, items: [{ ...item, efecto: "otro" }] } }) });
  assert.equal(mal.codigo, "respuesta_invalida");
  const denegada = await consultarSanciones("b", "p", { fetchImpl: respuesta(403, { error: { codigo: "acceso_denegado" } }) });
  assert.equal(denegada.status, 403);
  assert.equal(denegada.mensaje, MENSAJES_SANCIONES_ES.error_403);
});

test("B24 no envía una referencia con DNI ni datos incompletos", async () => {
  const comando = { consecuencia: "b24.sancion.suspension", causa: "x", fecha_notificacion: "2026-09-20", resuelta_por: "j", resolucion: { referencia: "12345678Z", sha256: SHA } };
  const fetchImpl = () => { throw new Error("No debe enviarse"); };
  assert.equal((await registrarSancion("b", "p", comando, "k", { fetchImpl })).status, 400);
  assert.equal((await registrarSancion("b", "p", { ...comando, resolucion: { referencia: "reg:1", sha256: "x" } }, "k", { fetchImpl })).status, 400);
  assert.equal((await registrarRecursoSancion("b", "p", "s", { estado: "interpuesto", fecha: "2026-09-20", documento: { referencia: "reg:1", sha256: "" } }, "k", { fetchImpl })).status, 400);
});

test("B24 registra con clave idempotente y distingue errores", async () => {
  let peticion;
  const fetchImpl = async (ruta, opciones) => { peticion = { ruta, opciones }; return { ok: true, status: 201, json: async () => ({ data: { sancion_ref: "s", recibo_ref: "r", situacion: "no_disponible", desde: "x", reutilizada: false } }) }; };
  const comando = { consecuencia: "b24.sancion.suspension", causa: "x", fecha_notificacion: "2026-09-20", resuelta_por: "j", resolucion: { referencia: "reg:1", sha256: SHA } };
  const r = await registrarSancion("b", "p", comando, "clave-1", { fetchImpl });
  assert.equal(r.ok, true);
  assert.equal(peticion.opciones.headers["Idempotency-Key"], "clave-1");
  assert.equal(peticion.opciones.method, "POST");
  const conflicto = await registrarSancion("b", "p", comando, "k", { fetchImpl: respuesta(409, { error: { codigo: "clave_reutilizada" } }) });
  assert.equal(conflicto.mensaje, MENSAJES_SANCIONES_ES.error_409_clave);
  const sinCatalogo = await registrarSancion("b", "p", comando, "k", { fetchImpl: respuesta(503, { error: { codigo: "sanciones_no_configuradas" } }) });
  assert.equal(sinCatalogo.mensaje, MENSAJES_SANCIONES_ES.error_503_catalogo);
});

test("B24 pinta histórico accesible y escapa el contenido", () => {
  const salida = renderizarSanciones({ estado: { carga: "listo", datos } });
  assert.match(salida, /<caption>Histórico de sanciones<\/caption>/);
  assert.match(salida, /No se presentó &lt;b&gt;/);
  assert.match(salida, /<summary aria-label="Ayuda sobre las sanciones">\?<\/summary>/);
  assert.match(salida, /data-b24-accion="nueva"/);
  assert.match(salida, /data-b24-accion="abrir-recurso"/);
  const sinCatalogo = renderizarSanciones({ estado: { carga: "listo", datos: { ...datos, catalogo_disponible: false, consecuencias: [] } } });
  assert.doesNotMatch(sinCatalogo, /data-b24-accion="nueva"/);
  assert.match(sinCatalogo, /no está configurado/);
  const baja = renderizarSanciones({ estado: { carga: "listo", datos, formularioAbierto: true, formulario: { consecuencia: "b24.sancion.baja_sin_contacto" } } });
  assert.match(baja, /no se puede deshacer/);
  assert.match(baja, /Baja \(art\. 11\.1\.a\)/);
  // La resolución se elige como archivo; su huella viaja oculta y conservada.
  assert.match(baja, /<input id="b24-resolucion-archivo" type="file" data-huella-archivo aria-describedby="b24-resolucion-archivo-estado" aria-required="true"/);
  const conHuella = renderizarSanciones({ estado: { carga: "listo", datos, formularioAbierto: true, formulario: { sha256: SHA } } });
  assert.match(conHuella, new RegExp(`<input type="hidden" name="sha256" value="${SHA}" data-huella-valor>`));
  assert.match(conHuella, /Documento comprobado en este equipo/);
  assert.match(salida, /<summary aria-label="Ayuda sobre las sanciones">\?<\/summary><p>[^<]*no se envía ni se guarda en VEC/);
  assert.match(renderizarSanciones({ estado: { carga: "listo", datos: { ...datos, items: [] } } }), /No hay sanciones/);
});

test("B24 controlador registra, conserva la clave en el reintento y recarga", async () => {
  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.v = f.valores; } get(c) { return this.v[c]; } };
  try {
    const claves = [];
    let fallar = true;
    const fetchImpl = async (ruta, opciones) => {
      if (opciones.method === "GET") return { ok: true, status: 200, json: async () => ({ data: datos }) };
      claves.push(opciones.headers["Idempotency-Key"]);
      if (fallar) { fallar = false; return { ok: false, status: 503, json: async () => ({ error: { codigo: "servicio_no_disponible" } }) }; }
      return { ok: true, status: 201, json: async () => ({ data: { sancion_ref: "s", recibo_ref: "recibo:1", situacion: "no_disponible", desde: "d", reutilizada: false } }) };
    };
    let recargas = 0;
    const modal = { candidato: { participacion_ref: "p", estado_clave: "disponible" } };
    const estado = { bolsaSeleccionada: "b", modalFicha: modal };
    const c = crearControladorSanciones({ estado, renderizar() {}, recargar: async () => { recargas += 1; }, fetchImpl });
    await c.cargar(modal);
    assert.equal(modal.sancionesB24.carga, "listo");
    c.manejarClick({ target: { closest: () => ({ dataset: { b24Accion: "nueva" } }) } });
    const formulario = { dataset: { b24Form: "sancion" }, valores: { consecuencia: "b24.sancion.suspension", causa: "x", fecha_notificacion: "2026-09-20", resuelta_por: "j", referencia: "reg:1", sha256: SHA } };
    const evento = { target: { closest: () => formulario } };
    c.manejarSubmit(evento);
    await new Promise((r) => setTimeout(r, 0));
    assert.equal(modal.sancionesB24.errorOperacion, MENSAJES_SANCIONES_ES.error_503);
    c.manejarSubmit(evento);
    await new Promise((r) => setTimeout(r, 5));
    assert.equal(claves.length, 2);
    assert.equal(claves[0], claves[1]);
    assert.equal(recargas, 1);
    assert.equal(modal.candidato.estado_clave, "no_disponible");
    assert.match(modal.sancionesB24.exito, /recibo:1/);
  } finally {
    globalThis.FormData = anterior;
  }
});

test("B37 muestra el efecto aplicado y la readmisión", () => {
  const conEfectos = {
    ...datos, estados_revocatorios: ["estimado"],
    items: [
      { ...item, efecto_aplicado: { situacion: "disponible_desde", vuelve_al_turno: "2027-03-20T23:00:00Z", orden_final: false }, reversion: null },
      { ...item, sancion_ref: "sancion:" + "d".repeat(64), efecto: "ninguna", suspension_hasta: null, efecto_aplicado: { situacion: null, vuelve_al_turno: null, orden_final: true }, reversion: null },
      { ...item, sancion_ref: "sancion:" + "e".repeat(64), efecto: "excluir", suspension_hasta: null, efecto_aplicado: { situacion: "excluido", vuelve_al_turno: null, orden_final: false },
        reversion: { estado_recurso: "estimado", regla_ref: "r", efecto_revertido: "excluir", situacion_restaurada: "disponible", situacion_desde: "2026-09-26T08:00:00Z",
          resuelta_por: "persona:jefatura", actor: "per_x", recibo_ref: "recibo:readmision:1", registrada_en: "2026-09-26T08:00:00Z" } },
    ],
  };
  const salida = renderizarSanciones({ estado: { carga: "listo", datos: conEfectos } });
  assert.match(salida, /Vuelve al turno el 21 mar 2027/);
  assert.match(salida, /Al final de la lista/);
  assert.match(salida, /Readmitida el 26 sept 2026/);
  assert.match(salida, /Vuelve a: Disponible/);
  assert.match(salida, /Resuelve persona:jefatura/);
  // Una sanción ya revocada no admite otro recurso.
  assert.equal((salida.match(/data-b24-accion="abrir-recurso"/g) || []).length, 2);
});

test("B37 el estado revocatorio pide quien resuelve y la resolución", async () => {
  const conEfectos = { ...datos, estados_recurso: ["interpuesto", "estimado"], estados_revocatorios: ["estimado"] };
  const formulario = renderizarSanciones({ estado: { carga: "listo", datos: conEfectos, recursoAbierto: item.sancion_ref, formularioRecurso: { estado: "estimado" } } });
  assert.match(formulario, /name="resuelta_por" required/);
  assert.match(formulario, /revoca la sanción/);
  assert.match(formulario, /name="referencia" maxlength="240" required/);
  const normal = renderizarSanciones({ estado: { carga: "listo", datos: conEfectos, recursoAbierto: item.sancion_ref, formularioRecurso: { estado: "interpuesto" } } });
  assert.doesNotMatch(normal, /name="resuelta_por"/);
  assert.match(formulario, /id="b24-recurso-archivo"[^>]*aria-required="true"/);
  assert.match(normal, /<span>Archivo del escrito \(opcional\)<\/span>/);
  assert.doesNotMatch(normal, /id="b24-recurso-archivo"[^>]*aria-required/);
  const fetchImpl = () => { throw new Error("No debe enviarse"); };
  const incompleto = await registrarRecursoSancion("b", "p", "s", { estado: "estimado", fecha: "2026-09-24" }, "k", { fetchImpl, revocatorios: ["estimado"] });
  assert.equal(incompleto.mensaje, MENSAJES_SANCIONES_ES.error_revierte);
  let cuerpo;
  const ok = await registrarRecursoSancion("b", "p", "s", { estado: "estimado", fecha: "2026-09-24", resuelta_por: "persona:jefatura", documento: { referencia: "reg:9", sha256: SHA } }, "k", {
    revocatorios: ["estimado"],
    fetchImpl: async (_r, o) => { cuerpo = JSON.parse(o.body); return { ok: true, status: 201, json: async () => ({ data: { sancion_ref: "s", estado: "estimado", reutilizada: false, revertida: true, recibo_ref: "recibo:readmision:1", situacion: "disponible", desde: "2026-09-26T08:00:00Z" } }) }; },
  });
  assert.equal(ok.ok, true);
  assert.equal(cuerpo.resuelta_por, "persona:jefatura");
});
