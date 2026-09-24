import assert from "node:assert/strict";
import test from "node:test";
import { crearTraductorCronos } from "./i18n.js";
import { montarVistaRemotoCronos, renderizarVistaRemotoCronos } from "./vista-remoto.js";

const permitido = { autorizado: true, continuidad_confirmada: true,
  movimientos_permitidos: ["entrada", "salida", "inicio_pausa", "fin_pausa"],
  periodo: { desde: "2026-09-24T00:00:00Z", hasta: "2026-10-01T00:00:00Z" }, motivo: "autorizado" };
const denegado = { autorizado: false, continuidad_confirmada: false, movimientos_permitidos: [], motivo: "teletrabajo_no_autorizado" };
const recibo = { referencia: "recibo:1", instante_utc: "2026-09-24T08:12:34Z", marcaje_original_ref: "marcaje:1", replay: false };
const t = crearTraductorCronos();
const siguiente = () => new Promise((resolver) => setImmediate(resolver));

function raizFalsa() {
  const nodo = { innerHTML: "", addEventListener() {}, removeEventListener() {}, remove() { this.eliminado = true; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}

test("sin autorización ni periodo muestra causa y bloquea las cuatro acciones", () => {
  const html = renderizarVistaRemotoCronos({ disponibilidad: denegado, estado: "listo", t });
  assert.match(html, /Fichaje remoto/u);
  assert.match(html, /No tiene autorización vigente/u);
  assert.equal((html.match(/data-cronos-remoto-movimiento=/gu) ?? []).length, 4);
  assert.equal((html.match(/disabled aria-disabled="true" title=/gu) ?? []).length, 4);
  assert.match(html, /class="panel cronos-panel"/u);
  assert.match(html, /class="cabecera-panel/u);
  assert.match(html, /class="cuerpo-panel"/u);
  assert.doesNotMatch(html, /navigator\.geolocation|coords|fingerprint/iu);
});

test("sólo el recibo confirmado muestra hora de servidor y referencia escapada", () => {
  const previo = renderizarVistaRemotoCronos({ disponibilidad: permitido, estado: "listo", t });
  assert.doesNotMatch(previo, /recibo:1/u);
  assert.match(previo, /hasta antes de/u);
  assert.match(previo, /data-cronos-remoto-movimiento="entrada"[^>]*>Registrar entrada/u);
  const confirmado = renderizarVistaRemotoCronos({ disponibilidad: permitido, estado: "registrado",
    recibo: { ...recibo, referencia: "<recibo:1>" }, t });
  assert.match(confirmado, /&lt;recibo:1&gt;/u);
  assert.match(confirmado, /<time datetime="2026-09-24T08:12:34Z">/u);
  assert.match(confirmado, /data-cronos-remoto-movimiento="entrada"[^>]+disabled/u);
  assert.match(confirmado, />\?<\/button>/u);
});

test("solo el movimiento enumerado por GET queda habilitado; lista vacía cierra todos", async () => {
  const parcial = { ...permitido, movimientos_permitidos: ["fin_pausa"] };
  const html = renderizarVistaRemotoCronos({ disponibilidad: parcial, estado: "listo", t });
  assert.match(html, /data-cronos-remoto-movimiento="fin_pausa"[^>]*>Finalizar pausa/u);
  assert.match(html, /data-cronos-remoto-movimiento="entrada"[^>]+disabled/u);
  const vacia = { ...permitido, movimientos_permitidos: [], motivo: "secuencia_no_permitida" };
  assert.match(renderizarVistaRemotoCronos({ disponibilidad: vacia, estado: "listo", t }), /No hay movimientos disponibles/u);
  let posts = 0;
  const { raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: { disponibilidad: async () => parcial,
    registrar: async () => { posts += 1; return recibo; } },
  cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente(); await vista.enviar("entrada");
  assert.equal(posts, 0);
  vista.desmontar();
});

test("409 de secuencia actualiza GET, muestra motivo y no reenvía POST", async () => {
  let consultas = 0; let posts = 0;
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { consultas += 1; return consultas === 1 ? permitido
      : { ...permitido, movimientos_permitidos: ["fin_pausa"] }; },
    registrar: async () => { posts += 1; throw Object.assign(new Error("secuencia"),
      { codigo: "secuencia_no_permitida", estado: 409, incierto: false }); },
  }, cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente(); await vista.enviar("entrada"); await siguiente();
  assert.equal(posts, 1);
  assert.equal(consultas, 2);
  assert.match(nodo.innerHTML, /Este movimiento no está permitido/u);
  assert.match(nodo.innerHTML, /data-cronos-remoto-movimiento="entrada"[^>]+disabled/u);
  assert.match(nodo.innerHTML, /data-cronos-remoto-movimiento="fin_pausa"[^>]*>Finalizar pausa/u);
  vista.desmontar();
});

test("la recarga bloquea nueva clave si falta continuidad o el servidor la deja pendiente", async () => {
  const sinCampo = { autorizado: true, periodo: permitido.periodo, motivo: "autorizado" };
  assert.match(renderizarVistaRemotoCronos({ disponibilidad: sinCampo, estado: "listo", t }),
    /El servidor aún no ha confirmado/u);
  for (const disponibilidad of [sinCampo, { ...permitido, continuidad_confirmada: false,
    movimientos_permitidos: [], motivo: "continuidad_no_confirmada" }]) {
    let envios = 0;
    const { nodo, raiz } = raizFalsa();
    const vista = montarVistaRemotoCronos({ raiz, cliente: { disponibilidad: async () => disponibilidad,
      registrar: async () => { envios += 1; return recibo; } },
    cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
    await siguiente();
    assert.match(nodo.innerHTML, /El servidor aún no ha confirmado/u);
    assert.equal((nodo.innerHTML.match(/disabled aria-disabled="true" title=/gu) ?? []).length, 4);
    await vista.enviar("entrada");
    assert.equal(envios, 0);
    vista.desmontar();
  }
});

test("GET distingue autenticación, denegación central y caída del servicio", async () => {
  for (const [codigo, estado, texto] of [
    ["autenticacion_requerida", "autenticacion", /Debe identificarse/u],
    ["acceso_denegado", "acceso_denegado", /No tiene permiso/u],
    ["servicio_no_disponible", "servicio", /servicio de fichaje remoto no está disponible/u],
  ]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarVistaRemotoCronos({ raiz, cliente: {
      disponibilidad: async () => { throw Object.assign(new Error(codigo), { codigo }); },
      registrar: async () => { throw new Error("no debe enviar"); },
    } });
    await siguiente();
    assert.equal(vista.estado().estado, estado);
    assert.match(nodo.innerHTML, texto);
    assert.match(nodo.innerHTML, /data-cronos-remoto-movimiento="entrada"[^>]+disabled/u);
    vista.desmontar();
  }
});

test("resultado incierto conserva una sola clave, bloquea alta nueva y reintenta el mismo POST", async () => {
  const solicitudes = []; let llamadas = 0; let recuperaciones = 0; let consultas = 0;
  const cliente = {
    async disponibilidad() { consultas += 1; return permitido; },
    async registrar(solicitud) { solicitudes.push({ ...solicitud }); llamadas += 1;
      if (llamadas === 1) throw Object.assign(new Error("red"), { incierto: true });
      return { ...recibo, replay: true }; },
    async recuperar(solicitud) { recuperaciones += 1; assert.deepEqual(solicitud, solicitudes[0]);
      throw Object.assign(new Error("ausente"), { estado: 404, codigo: "ausencia_confirmada" }); },
  };
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente, cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente();
  assert.equal(vista.estado().estado, "listo");
  await vista.enviar("entrada");
  assert.equal(vista.estado().estado, "incierto");
  assert.match(nodo.innerHTML, /data-cronos-remoto-reintentar/u);
  await vista.actualizar();
  assert.equal(vista.estado().estado, "incierto");
  await vista.enviar("salida");
  assert.equal(solicitudes.length, 1);
  await vista.enviar(null, true);
  assert.deepEqual(solicitudes, [solicitudes[0], solicitudes[0]]);
  assert.equal(recuperaciones, 1);
  assert.equal(consultas, 2);
  assert.equal(vista.estado().estado, "registrado");
  assert.equal(vista.estado().recibo.replay, true);
  assert.match(nodo.innerHTML, /recibo:1/u);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("recuperación devuelve recibo aun sin repetir POST y un 404 débil falla cerrado", async () => {
  for (const [codigo, estadoHTTP, esperado, llamadasPost] of [
    [null, 200, "registrado", 1],
    ["respuesta_rechazada", 404, "recuperacion_no_disponible", 1],
    ["conflicto", 409, "conflicto", 1],
    ["servicio_no_disponible", 503, "servicio_incierto", 1],
  ]) {
    let posts = 0;
    const { nodo, raiz } = raizFalsa();
    const vista = montarVistaRemotoCronos({ raiz, cliente: {
      disponibilidad: async () => permitido,
      registrar: async () => { posts += 1; throw Object.assign(new Error("red"), { incierto: true }); },
      recuperar: async () => { if (codigo) throw Object.assign(new Error(codigo), { codigo, estado: estadoHTTP });
        return { ...recibo, replay: true }; },
    }, cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
    await siguiente(); await vista.enviar("entrada"); await vista.enviar(null, true);
    assert.equal(vista.estado().estado, esperado);
    assert.equal(posts, llamadasPost);
    if (esperado === "registrado") assert.match(nodo.innerHTML, /recibo:1/u);
    vista.desmontar();
  }
});

test("404 fuerte no reenvía si GET actual revoca permiso, continuidad o movimiento", async () => {
  for (const actual of [
    denegado,
    { ...permitido, continuidad_confirmada: false, movimientos_permitidos: [], motivo: "continuidad_no_confirmada" },
    { ...permitido, movimientos_permitidos: ["salida"] },
  ]) {
    let consultas = 0; let posts = 0;
    const { raiz } = raizFalsa();
    const vista = montarVistaRemotoCronos({ raiz, cliente: {
      disponibilidad: async () => { consultas += 1; return consultas === 1 ? permitido : actual; },
      registrar: async () => { posts += 1; throw Object.assign(new Error("incierto"), { incierto: true }); },
      recuperar: async () => { throw Object.assign(new Error("ausente"), { codigo: "ausencia_confirmada", estado: 404 }); },
    }, cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
    await siguiente(); await vista.enviar("entrada"); await vista.enviar(null, true);
    assert.equal(posts, 1);
    assert.equal(consultas, 2);
    assert.equal(vista.estado().pendiente, null);
    vista.desmontar();
  }
});

test("el 403 central impide otro marcaje sin atribuir falta de teletrabajo", async () => {
  const { raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => permitido,
    registrar: async () => { throw Object.assign(new Error("403"), { estado: 403, incierto: false }); },
  }, cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente(); await vista.enviar("entrada");
  assert.equal(vista.estado().disponibilidad, null);
  assert.equal(vista.estado().estado, "acceso_denegado");
  assert.equal(vista.estado().pendiente, null);
  vista.desmontar();
});

test("POST 403 nominal y 503 de continuidad bloquean; 503 de servicio conserva clave", async () => {
  for (const [codigo, estadoHTTP, incierto, estadoVista, texto] of [
    ["teletrabajo_no_autorizado", 403, false, "listo", /No tiene autorización vigente/u],
    ["continuidad_no_confirmada", 503, false, "continuidad", /El servidor aún no ha confirmado/u],
    ["servicio_no_disponible", 503, true, "servicio_incierto", /servicio de fichaje remoto no está disponible/u],
  ]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarVistaRemotoCronos({ raiz, cliente: {
      disponibilidad: async () => permitido,
      registrar: async () => { throw Object.assign(new Error(codigo), { codigo, estado: estadoHTTP, incierto }); },
    }, cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
    await siguiente(); await vista.enviar("entrada");
    assert.equal(vista.estado().estado, estadoVista);
    assert.match(nodo.innerHTML, texto);
    assert.match(nodo.innerHTML, /data-cronos-remoto-movimiento="entrada"[^>]+disabled/u);
    assert.equal(Boolean(vista.estado().pendiente), incierto);
    vista.desmontar();
  }
});

test("el desmontaje ignora respuestas tardías", async () => {
  let resolver;
  const otra = raizFalsa();
  const pendiente = montarVistaRemotoCronos({ raiz: otra.raiz, cliente: {
    disponibilidad: () => new Promise((r) => { resolver = r; }), registrar: async () => recibo,
  } });
  pendiente.desmontar(); resolver(permitido); await siguiente();
  assert.equal(otra.nodo.eliminado, true);
  assert.equal(pendiente.estado().estado, "consultando");
});
