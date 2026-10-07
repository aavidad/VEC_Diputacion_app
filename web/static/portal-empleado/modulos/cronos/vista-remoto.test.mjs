import "./test-preparar-textos.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { crearTraductorCronos } from "./i18n.js?v=20260929-i18n-textos-v1";
import { montarVistaRemotoCronos, renderizarVistaRemotoCronos } from "./vista-remoto.js";

const permitido = { autorizado: true, continuidad_confirmada: true,
  movimientos_permitidos: ["entrada", "salida", "inicio_pausa", "fin_pausa"],
  periodo: { desde: "2026-09-24T00:00:00Z", hasta: "2026-10-01T00:00:00Z" }, motivo: "autorizado" };
const denegado = { autorizado: false, continuidad_confirmada: false, movimientos_permitidos: [], motivo: "teletrabajo_no_autorizado" };
const recibo = { referencia: "recibo:1", instante_utc: "2026-09-24T08:12:34Z", marcaje_original_ref: "marcaje:1", replay: false };
const t = crearTraductorCronos();
const siguiente = () => new Promise((resolver) => setImmediate(resolver));

function raizFalsa() {
  const documento = { activeElement: null, createElement: () => nodo };
  let html = ""; let elementos = [];
  const nodo = {
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor;
      elementos = Array.from(valor.matchAll(/<(?:button|p|div)\b([^>]*data-cronos-remoto[^>]*)>/gu), ([, atributos]) => ({
        disabled: /\bdisabled(?:\s|$)/u.test(atributos),
        getAttribute(clave) { return atributos.match(new RegExp(`${clave}="([^"]*)"`, "u"))?.[1] ?? null; },
        hasAttribute(clave) { return new RegExp(`(?:^|\\s)${clave}(?:[=\\s]|$)`, "u").test(atributos); },
        focus() { documento.activeElement = this; },
      }));
    },
    contains(elemento) { return elementos.includes(elemento); },
    querySelector(selector) {
      const [, atributo, valor] = selector.match(/\[([a-z-]+)(?:="([^"]+)")?\]/u);
      return elementos.find((elemento) => elemento.hasAttribute(atributo)
        && (valor === undefined || elemento.getAttribute(atributo) === valor)
        && (!selector.includes(":not(:disabled)") || !elemento.disabled)) ?? null;
    },
    addEventListener() {}, removeEventListener() {}, remove() { this.eliminado = true; },
  };
  return { nodo, raiz: { ownerDocument: documento, append() {} }, documento };
}

function diferida() {
  let resolver; let rechazar;
  const promesa = new Promise((resolve, reject) => { resolver = resolve; rechazar = reject; });
  return { promesa, resolver, rechazar };
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
  assert.equal(consultas, 3);
  assert.equal(vista.estado().estado, "listo");
  assert.equal(vista.estado().recibo.replay, true);
  assert.match(nodo.innerHTML, /recibo:1/u);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("recuperación devuelve recibo aun sin repetir POST y un 404 débil falla cerrado", async () => {
  for (const [codigo, estadoHTTP, esperado, llamadasPost] of [
    [null, 200, "listo", 1],
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
    if (esperado === "listo") assert.match(nodo.innerHTML, /recibo:1/u);
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

test("GET 404: «no disponible» neutro, sin acciones de fichaje ni tono de aviso", async () => {
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { throw Object.assign(new Error("respuesta_rechazada"), { codigo: "respuesta_rechazada", estado: 404 }); },
    registrar: async () => { throw new Error("no debe enviar"); },
  } });
  await siguiente();
  assert.equal(vista.estado().estado, "no_disponible");
  assert.match(nodo.innerHTML, /<p[^>]* class="cronos-vacio" role="status" aria-live="polite">El fichaje remoto no está disponible\./u);
  assert.doesNotMatch(nodo.innerHTML, /role="alert"|cronos-estado-aviso|data-cronos-remoto-movimiento/u);
  assert.match(nodo.innerHTML, /data-cronos-remoto-actualizar/u);
  vista.desmontar();
});

test("POST confirmado consulta siguiente movimiento y mantiene recibo durante GET y Actualizar", async () => {
  const consulta = diferida(); let gets = 0; let posts = 0; const notificados = [];
  const { nodo, raiz, documento } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { gets += 1; return gets === 2 ? consulta.promesa
      : { ...permitido, movimientos_permitidos: gets === 1 ? ["entrada"] : ["salida"] }; },
    registrar: async () => { posts += 1; return recibo; },
  }, onRegistrado: (valor) => notificados.push(valor),
  cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente();
  nodo.querySelector('[data-cronos-remoto-movimiento="entrada"]').focus();
  const envio = vista.enviar("entrada"); await siguiente();
  assert.equal(gets, 2); assert.equal(posts, 1);
  assert.equal(vista.estado().estado, "consultando");
  assert.deepEqual(notificados, [recibo]);
  assert.match(nodo.innerHTML, /recibo:1/u);
  assert.match(nodo.innerHTML, /Comprobando qué movimiento/u);
  assert.equal((nodo.innerHTML.match(/disabled aria-disabled="true" title=/gu) ?? []).length, 4);
  await vista.enviar("salida"); assert.equal(posts, 1);
  consulta.resolver({ ...permitido, movimientos_permitidos: ["salida"] }); await envio;
  assert.equal(vista.estado().estado, "listo");
  assert.equal(documento.activeElement.getAttribute("data-cronos-remoto-movimiento"), "salida");
  assert.match(nodo.innerHTML, /Último fichaje confirmado/u);
  assert.match(nodo.innerHTML, /recibo:1/u);
  await vista.actualizar();
  assert.equal(gets, 3); assert.equal(posts, 1); assert.equal(notificados.length, 1);
  assert.deepEqual(vista.estado().recibo, recibo);
  assert.match(nodo.innerHTML, /recibo:1/u);
  vista.desmontar();
});

test("fallo de GET posterior conserva éxito y no permite reenviar; callback fallido se aísla", async () => {
  for (const callback of [() => { throw new Error("consumidor"); }, () => Promise.reject(new Error("consumidor"))]) {
    let gets = 0; let posts = 0; let notificaciones = 0;
    const { nodo, raiz } = raizFalsa();
    const vista = montarVistaRemotoCronos({ raiz, cliente: {
      disponibilidad: async () => { gets += 1; if (gets > 1) throw new Error("red"); return permitido; },
      registrar: async () => { posts += 1; return recibo; },
    }, onRegistrado: () => { notificaciones += 1; return callback(); },
    cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
    await siguiente(); await vista.enviar("entrada"); await siguiente();
    assert.equal(vista.estado().estado, "error");
    assert.equal(vista.estado().pendiente, null);
    assert.equal(vista.estado().actualizacionLecturasPendiente, true);
    assert.deepEqual(vista.estado().recibo, recibo);
    assert.match(nodo.innerHTML, /Queda pendiente actualizar/u);
    assert.match(nodo.innerHTML, /Su último fichaje está confirmado/u);
    assert.match(nodo.innerHTML, /recibo:1/u);
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-remoto-reintentar/u);
    await vista.enviar("entrada"); await vista.enviar(null, true); await vista.actualizar();
    assert.equal(posts, 1); assert.equal(gets, 3); assert.equal(notificaciones, 1);
    assert.match(nodo.innerHTML, /recibo:1/u);
    vista.desmontar();
  }
});

test("recibo recuperado actualiza disponibilidad sin POST y notifica una vez", async () => {
  let gets = 0; let posts = 0; let notificaciones = 0;
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { gets += 1; return { ...permitido,
      movimientos_permitidos: gets === 1 ? ["entrada"] : ["salida"] }; },
    registrar: async () => { posts += 1; throw Object.assign(new Error("red"), { incierto: true }); },
    recuperar: async () => ({ ...recibo, replay: true }),
  }, onRegistrado: (valor) => { assert.equal(valor.replay, true); notificaciones += 1; },
  cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente(); await vista.enviar("entrada");
  assert.equal(gets, 1); assert.equal(notificaciones, 0);
  await vista.actualizar(); assert.equal(gets, 1);
  await vista.enviar(null, true);
  assert.equal(gets, 2); assert.equal(posts, 1); assert.equal(notificaciones, 1);
  assert.equal(vista.estado().estado, "listo");
  assert.match(nodo.innerHTML, /data-cronos-remoto-movimiento="salida"[^>]*>Registrar salida/u);
  assert.match(nodo.innerHTML, /recibo:1/u);
  vista.desmontar();
});

test("respuesta POST incompatible conserva incertidumbre sin GET ni callback", async () => {
  let gets = 0; let notificaciones = 0;
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { gets += 1; return permitido; },
    registrar: async () => ({ ...recibo, replay: "true" }),
  }, onRegistrado: () => { notificaciones += 1; },
  cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente(); await vista.enviar("entrada");
  assert.equal(gets, 1); assert.equal(notificaciones, 0);
  assert.equal(vista.estado().estado, "incierto");
  assert.ok(vista.estado().pendiente); assert.equal(vista.estado().recibo, null);
  assert.doesNotMatch(nodo.innerHTML, /recibo:1/u);
  vista.desmontar();
});

test("POST tardío y GET posterior tardío no pintan ni notifican después de desmontar", async () => {
  for (const fase of ["post", "get"]) {
    const respuesta = diferida(); let gets = 0; let notificaciones = 0; let signal;
    const { nodo, raiz } = raizFalsa();
    const vista = montarVistaRemotoCronos({ raiz, cliente: {
      disponibilidad: async (opciones) => { gets += 1; if (fase === "get" && gets === 2) {
        signal = opciones.signal; return respuesta.promesa;
      } return permitido; },
      registrar: async (_solicitud, opciones) => {
        if (fase === "post") { signal = opciones.signal; return respuesta.promesa; } return recibo;
      },
    }, onRegistrado: () => { notificaciones += 1; },
    cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
    await siguiente(); const envio = vista.enviar("entrada"); await siguiente();
    const antes = nodo.innerHTML; const avisosAntes = notificaciones;
    vista.desmontar(); assert.equal(signal.aborted, true);
    respuesta.resolver(fase === "post" ? recibo : permitido); await envio;
    assert.equal(nodo.innerHTML, antes); assert.equal(notificaciones, avisosAntes);
    assert.equal(gets, fase === "post" ? 1 : 2);
  }
});

test("la actualización no roba el foco si la persona ya salió del bloque", async () => {
  const respuesta = diferida(); let gets = 0;
  const { nodo, raiz, documento } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { gets += 1; return gets === 1 ? permitido : respuesta.promesa; },
    registrar: async () => recibo,
  }, cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente(); nodo.querySelector('[data-cronos-remoto-movimiento="entrada"]').focus();
  const envio = vista.enviar("entrada"); await siguiente();
  const externo = {}; documento.activeElement = externo;
  respuesta.resolver({ ...permitido, movimientos_permitidos: ["salida"] }); await envio;
  assert.equal(documento.activeElement, externo);
  vista.desmontar();
});

test("callback recibe copia inmutable y puede desmontar sin otro GET ni aviso tardío", async () => {
  const respuesta = diferida(); let gets = 0; let recibido;
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { gets += 1; return permitido; }, registrar: async () => recibo,
  }, onRegistrado: (valor) => { recibido = valor; vista.desmontar(); return respuesta.promesa; },
  cryptoImpl: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" } });
  await siguiente(); await vista.enviar("entrada");
  assert.equal(gets, 1); assert.equal(nodo.eliminado, true);
  assert.deepEqual(recibido, recibo); assert.ok(Object.isFrozen(recibido));
  assert.notEqual(recibido, vista.estado().recibo);
  const anterior = nodo.innerHTML;
  respuesta.rechazar(new Error("lectura tardía")); await siguiente();
  assert.equal(nodo.innerHTML, anterior);
  assert.equal(vista.estado().actualizacionLecturasPendiente, false);
  assert.deepEqual(vista.estado().recibo, recibo);
});

test("un nuevo fichaje incierto conserva el recibo anterior sin confirmar el nuevo", async () => {
  let posts = 0; let claves = 0; let gets = 0;
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaRemotoCronos({ raiz, cliente: {
    disponibilidad: async () => { gets += 1; return { ...permitido,
      movimientos_permitidos: gets === 1 ? ["entrada"] : ["salida"] }; },
    registrar: async () => { posts += 1; if (posts === 1) return recibo; throw new Error("red"); },
  }, cryptoImpl: { randomUUID: () => `123e4567-e89b-42d3-a456-42661417400${claves++}` } });
  await siguiente(); await vista.enviar("entrada"); await vista.enviar("salida");
  assert.equal(gets, 2); assert.equal(posts, 2);
  assert.equal(vista.estado().estado, "incierto");
  assert.equal(vista.estado().pendiente.movimiento, "salida");
  assert.deepEqual(vista.estado().recibo, recibo);
  assert.match(nodo.innerHTML, /Último fichaje confirmado/u);
  assert.match(nodo.innerHTML, /recibo:1/u);
  assert.match(nodo.innerHTML, /data-cronos-remoto-reintentar/u);
  vista.desmontar();
});
