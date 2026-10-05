import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaContactoPropio } from "./vista-contacto-propio.js";

function raizFalsa() {
  class Nodo {
    constructor(d, tipo) { this.ownerDocument = d; this.tagName = tipo; this.children = []; this.dataset = {}; this.atributos = new Map(); this.listeners = new Map(); this.textContent = ""; }
    append(...hijos) { this.children.push(...hijos); hijos.forEach(h => { h.parent = this; }); }
    replaceChildren(...hijos) { this.children = []; this.append(...hijos); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter(h => h !== this); }
    setAttribute(k, v) { this.atributos.set(k, v); }
    addEventListener(k, fn) { this.listeners.set(k, fn); }
    click() { return this.listeners.get("click")?.(); }
  }
  const d = { createElement: tipo => new Nodo(d, tipo) }; return new Nodo(d, "root");
}
const nodos = n => [n, ...n.children.flatMap(nodos)];
const texto = n => nodos(n).map(x => x.textContent).join(" ");
const completar = () => new Promise(resolve => setImmediate(resolve));
const correo = { correo_ref: "correo:" + "a".repeat(32), direccion: "carmen.ruiz@example.org", estado: "verificado", activo: true, creado_utc: "2026-10-03T10:00:00Z" };
const datos = { version: 1, correos: [correo] };
const respuesta = data => new Response(JSON.stringify({ data }), { status: 200, headers: { "Content-Type": "application/json" } });

test("consulta sólo el GET interno de Usuarios, muestra la fecha de alta y no entrega contacto a Personal", async () => {
  const raiz = raizFalsa(), llamadas = [];
  montarVistaContactoPropio({ raiz, fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta(datos); } });
  await completar();
  assert.equal(llamadas.length, 1); assert.equal(llamadas[0].ruta, "/api/vec/usuarios/mis-correos");
  assert.equal(llamadas[0].opciones.method, "GET"); assert.equal(llamadas[0].opciones.body, undefined);
  assert.equal(llamadas[0].opciones.credentials, "same-origin"); assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[0].opciones.redirect, "error"); assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  assert.match(texto(raiz), /carmen.ruiz@example.org/); assert.match(texto(raiz), /Fecha de alta/); assert.match(texto(raiz), /Recibe los avisos/);
  assert.doesNotMatch(texto(raiz), /correo:|persona:|Empleado|certificado/i);
});

test("actualizar retira las direcciones antes de consultar y un rechazo no conserva datos", async () => {
  const raiz = raizFalsa(); let siguiente, llamada = 0;
  montarVistaContactoPropio({ raiz, cliente: { consultar() {
    llamada += 1; return llamada === 1 ? Promise.resolve(datos) : new Promise((_resolve, reject) => { siguiente = reject; });
  } } }); await completar();
  assert.match(texto(raiz), /carmen.ruiz@example.org/);
  const actualizar = nodos(raiz).find(n => Object.hasOwn(n.dataset, "personalContactoActualizar"));
  const intento = actualizar.click(); assert.doesNotMatch(texto(raiz), /carmen.ruiz@example.org/);
  siguiente({ estado: 403 }); await intento;
  assert.match(texto(raiz), /perfil que tiene activo/); assert.doesNotMatch(texto(raiz), /carmen.ruiz@example.org/); assert.equal(actualizar.disabled, false);
});

test("al desmontar cancela el transporte y descarta una respuesta tardía de otro contexto", async () => {
  const raiz = raizFalsa(); let resolver, signal;
  const vista = montarVistaContactoPropio({ raiz, cliente: { consultar(opciones) { signal = opciones.signal; return new Promise(r => { resolver = r; }); } } });
  vista.desmontar(); assert.equal(signal.aborted, true); resolver(datos); await completar();
  assert.equal(raiz.children.length, 0); assert.doesNotMatch(texto(raiz), /carmen.ruiz@example.org/);
});

test("una sesión caducada retira datos y avisa al shell; un conjunto inválido nunca muestra filas parciales", async () => {
  const raiz = raizFalsa(); let caducadas = 0;
  montarVistaContactoPropio({ raiz, cliente: { consultar: async () => { throw { estado: 401 }; } }, alCaducarSesion: () => { caducadas += 1; } }); await completar();
  assert.equal(caducadas, 1); assert.match(texto(raiz), /sesión ha caducado/);
  const invalida = raizFalsa();
  montarVistaContactoPropio({ raiz: invalida, cliente: { consultar: async () => ({ ...datos, correos: [correo, { ...correo, direccion: "luis.martin@example.org" }] }) } }); await completar();
  assert.doesNotMatch(texto(invalida), /carmen.ruiz@example.org|luis.martin@example.org/); assert.match(texto(invalida), /No se han podido/);
});

test("en el móvil cada fila se apila con su etiqueta y la dirección larga se parte", async () => {
  const raiz = raizFalsa();
  montarVistaContactoPropio({ raiz, cliente: { consultar: async () => datos } }); await completar();
  const tabla = nodos(raiz).find(n => n.tagName === "table");
  assert.equal(tabla.className, "tabla-datos tabla-apilable");
  const celdas = nodos(tabla).filter(n => n.tagName === "td");
  assert.deepEqual(celdas.map(c => c.dataset.etiqueta), ["Dirección", "Estado", "Fecha de alta"]);
  assert.equal(celdas[0].className, "correos-direccion");
  const estado = nodos(raiz).find(n => Object.hasOwn(n.dataset, "personalContactoEstado"));
  assert.equal(estado.textContent, ""); assert.equal(estado.hidden, true);
});

test("un rechazo se muestra como aviso y se anuncia; la sesión caducada la anuncia sólo la ficha", async () => {
  const anuncios = [];
  const raiz = raizFalsa();
  montarVistaContactoPropio({ raiz, anunciar: (t) => anuncios.push(t), cliente: { consultar: async () => { throw { estado: 403 }; } } }); await completar();
  const estado = nodos(raiz).find(n => Object.hasOwn(n.dataset, "personalContactoEstado"));
  assert.equal(estado.className, "personal-ficha-mensaje"); assert.equal(estado.hidden, false); assert.equal(anuncios.length, 1);
  const caducada = raizFalsa(); let avisos = 0;
  montarVistaContactoPropio({ raiz: caducada, anunciar: () => { avisos += 1; }, alCaducarSesion: () => {}, cliente: { consultar: async () => { throw { estado: 401 }; } } }); await completar();
  assert.equal(avisos, 0);
});
