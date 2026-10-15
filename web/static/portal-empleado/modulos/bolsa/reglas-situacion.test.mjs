import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearClienteReglasSituacionBolsa, validarPoliticaTransiciones } from "./reglas-situacion-api.js";
import { montarReglasSituacionBolsa } from "./reglas-situacion-ui.js";
import { crearTextos } from "../../../comun/textos.js";

const catalogo = JSON.parse(await readFile(new URL("../../../textos/es/bolsa-reglas-transiciones.json", import.meta.url), "utf8"));
const textos = crearTextos({ modulo: "bolsa-reglas-transiciones", idioma: "es", localizacion: "es-ES", respaldo: catalogo });
const A = "disponible>no_disponible", B = "disponible>renuncia", C = "renuncia>disponible";
const base = { esquema: "vec.bolsa.rrhh.politica-transiciones.v1", version: 2,
  transiciones_guardadas: [A, C], transiciones_efectivas: [A, C], transiciones_obligatorias: [C],
  limites: { catalogo_ref: "catalogo:reglas:2", catalogo_version: 2, catalogo_sha256: "a".repeat(64), transiciones: [A, B, C] },
  puede_publicar: true, catalogo_activado: false };
const sobre = (p = base) => ({ data: p });
const respuesta = (p, status = 200) => new Response(JSON.stringify(sobre(p)), { status, headers: { "content-type": "application/json" } });
const pausa = () => new Promise((resolve) => setImmediate(resolve));
function raizFalsa() {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; }, querySelector: () => null };
  const pulsar = (accion) => eventos.get("click")({ target: { closest: () => ({ dataset: { reglasAccion: accion } }) } });
  const cambiar = (par, checked) => eventos.get("change")({ target: { dataset: { reglasPar: par }, checked } });
  const elegir = (value) => eventos.get("change")({ target: { matches: () => true, value } });
  const escribir = (value) => eventos.get("input")({ target: { matches: () => true, value } });
  return { raiz, eventos, pulsar, cambiar, elegir, escribir };
}

test("consulta cerrada y publicación exacta, sin credenciales añadidas", async () => {
  const llamadas = [];
  const cliente = crearClienteReglasSituacionBolsa({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    return opciones.method === "GET" ? respuesta(base) : respuesta({ ...base, version: 3,
      transiciones_guardadas: [A, B, C], transiciones_efectivas: [A, B, C], recibo_ref: "recibo:3", reutilizada: false }, 201);
  } });
  await cliente.consultar();
  const cuerpo = { version_esperada: 2, transiciones: [A, B, C], motivo: "Cambio aprobado", clave_idempotencia: "d054da0e-1b6d-4e92-8fc5-bba91588368b" };
  assert.equal((await cliente.publicar(cuerpo)).recibo_ref, "recibo:3");
  assert.equal(llamadas.length, 2);
  assert.ok(llamadas.every(([ruta, opciones]) => ruta === "/api/vec/bolsa/politica-transiciones"
    && opciones.credentials === "same-origin" && opciones.headers.Authorization === undefined && opciones.cache === "no-store"));
  assert.deepEqual(JSON.parse(llamadas[1][1].body), cuerpo);
  await assert.rejects(cliente.publicar({ ...cuerpo, actor: "inventado" }), TypeError);
  assert.equal(llamadas.length, 2);
});

test("admite legado fuera de límites en lectura; cierra publicación y denegación", async () => {
  assert.equal(validarPoliticaTransiciones(sobre({ ...base, transiciones_guardadas: [A, "trabajando>disponible"],
    transiciones_efectivas: [A], transiciones_obligatorias: [B] })).version, 2);
  assert.throws(() => validarPoliticaTransiciones(sobre({ ...base, limites: { ...base.limites, transiciones: [A] } })));
  assert.throws(() => validarPoliticaTransiciones(sobre({ ...base, transiciones_guardadas: [A], recibo_ref: "recibo:3", reutilizada: false }), { publicacion: true }));
  assert.throws(() => validarPoliticaTransiciones(sobre({ ...base, transiciones_efectivas: ["desconocida>renuncia"] })));
  const cliente = crearClienteReglasSituacionBolsa({ fetchImpl: async () => new Response("", { status: 403 }) });
  await assert.rejects(cliente.consultar(), (error) => error.estado === 403);
});

test("revisa, envía una vez, conserva operación incierta y confirma solo recibo válido", async () => {
  const vista = raizFalsa(); const envios = [];
  let respuestaPendiente = false;
  const cliente = { consultar: async () => base, publicar: async (cuerpo) => {
    envios.push(cuerpo); if (!respuestaPendiente) { respuestaPendiente = true; throw new Error("red cortada"); }
    return { ...base, version: 3, recibo_ref: "recibo:3", reutilizada: true };
  } };
  const montada = montarReglasSituacionBolsa({ ...vista, cliente, textos }); await pausa();
  assert.match(vista.raiz.innerHTML, /todas las bolsas/u);
  vista.elegir("renuncia"); assert.match(vista.raiz.innerHTML, /Obligatorio/u);
  vista.elegir("disponible");
  vista.cambiar(B, true); vista.escribir("Nueva regla aprobada"); vista.pulsar("revisar");
  assert.match(vista.raiz.innerHTML, /Revise los cambios/u);
  assert.equal(envios.length, 0);
  vista.pulsar("confirmar"); await pausa();
  assert.match(vista.raiz.innerHTML, /No se ha podido confirmar/u);
  assert.doesNotMatch(vista.raiz.innerHTML, /Reglas publicadas/u);
  vista.pulsar("repetir"); await pausa();
  assert.equal(envios.length, 2);
  assert.deepEqual(envios[0], envios[1]);
  assert.match(vista.raiz.innerHTML, /Versión publicada: 3/u);
  assert.match(vista.raiz.innerHTML, /recibo:3/u);
  montada.desmontar(); assert.equal(vista.eventos.size, 0);
});

test("una respuesta posterior de permiso o conflicto conserva la operación incierta", async () => {
  for (const estado of [401, 403, 409, 422]) {
    const vista = raizFalsa(); const envios = [];
    const cliente = { consultar: async () => base, publicar: async (cuerpo) => {
      envios.push(cuerpo);
      if (envios.length === 1) throw new Error("respuesta perdida");
      if (envios.length === 2) throw Object.assign(new Error("rechazada después"), { estado });
      return { ...base, version: 3, recibo_ref: "recibo:3", reutilizada: true };
    } };
    const montada = montarReglasSituacionBolsa({ ...vista, cliente, textos }); await pausa();
    vista.cambiar(B, true); vista.escribir("Publicar esta regla"); vista.pulsar("revisar");
    vista.pulsar("confirmar"); await pausa();
    vista.pulsar("repetir"); await pausa();
    assert.match(vista.raiz.innerHTML, /No se ha podido confirmar/u, `estado ${estado}`);
    assert.match(vista.raiz.innerHTML, /data-reglas-accion="repetir"/u, `estado ${estado}`);
    assert.doesNotMatch(vista.raiz.innerHTML, /Reglas publicadas/u, `estado ${estado}`);
    vista.pulsar("repetir"); await pausa();
    assert.equal(envios.length, 3);
    assert.ok(envios.every((cuerpo) => cuerpo === envios[0]), `cuerpo y UUID originales, estado ${estado}`);
    assert.match(vista.raiz.innerHTML, /recibo:3/u);
    montada.desmontar();
  }
});

test("conflicto exige recargar sin publicar otra vez", async () => {
  const vista = raizFalsa(); let lecturas = 0, envios = 0;
  const cliente = { consultar: async () => { lecturas++; return base; }, publicar: async () => {
    envios++; throw Object.assign(new Error("cambió"), { estado: 409 });
  } };
  const montada = montarReglasSituacionBolsa({ ...vista, cliente, textos }); await pausa();
  vista.cambiar(B, true); vista.escribir("Nueva regla aprobada"); vista.pulsar("revisar"); vista.pulsar("confirmar"); await pausa();
  assert.match(vista.raiz.innerHTML, /Vuelva a cargar/u);
  assert.match(vista.raiz.innerHTML, /data-reglas-accion="revisar" disabled/u);
  assert.equal(envios, 1);
  vista.pulsar("recargar"); await pausa(); assert.equal(lecturas, 2);
  montada.desmontar();
});

test("permite publicar límites actuales aunque las casillas no cambien", async () => {
  const vista = raizFalsa(); let envios = 0;
  const montada = montarReglasSituacionBolsa({ ...vista, cliente: { consultar: async () => base,
    publicar: async () => { envios++; throw new Error("no debe enviarse en revisión"); } }, textos });
  await pausa(); vista.escribir("Aprobar los límites actuales"); vista.pulsar("revisar");
  assert.match(vista.raiz.innerHTML, /Las opciones no cambian/u);
  assert.equal(envios, 0);
  montada.desmontar();
});

test("desmontar cancela carga tardía", async () => {
  const vista = raizFalsa(); let consultar;
  const cliente = { consultar: () => new Promise((resolve) => { consultar = resolve; }), publicar: async () => base };
  const montada = montarReglasSituacionBolsa({ ...vista, cliente, textos });
  montada.desmontar(); consultar(base); await pausa(); assert.equal(vista.raiz.innerHTML, "");
});
