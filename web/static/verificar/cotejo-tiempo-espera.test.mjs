import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";

const codigo = await readFile(new URL("./verificar.js", import.meta.url), "utf8");

function diferida() {
  let resolver;
  const promesa = new Promise((resolve) => { resolver = resolve; });
  return { promesa, resolver };
}

function entorno() {
  const peticiones = [];
  const temporizadores = new Map();
  const boton = { disabled: false };
  const entrada = { value: "REF-PRUEBA-0001" };
  const resultado = {
    hidden: true,
    dataset: {},
    textContent: "",
    innerHTML: "",
    removeAttribute(nombre) { if (nombre === "data-estado") delete this.dataset.estado; },
  };
  let enviar;
  const formulario = {
    querySelector() { return boton; },
    addEventListener(nombre, callback) { if (nombre === "submit") enviar = callback; },
  };
  let siguienteTemporizador = 0;
  runInNewContext(codigo, {
    document: { getElementById(id) {
      return { "formulario-cotejo": formulario, referencia: entrada,
        "resultado-cotejo": resultado, "aviso-presentacion": { hidden: true } }[id];
    } },
    window: { location: { search: "" } },
    URLSearchParams,
    AbortController,
    setTimeout(callback, demora) {
      const id = ++siguienteTemporizador;
      temporizadores.set(id, { callback, demora });
      return id;
    },
    clearTimeout(id) { temporizadores.delete(id); },
    fetch(ruta, opciones) {
      const respuesta = diferida();
      peticiones.push({ ruta, opciones, ...respuesta });
      return respuesta.promesa;
    },
  });
  return {
    boton, entrada, resultado, peticiones, temporizadores,
    enviar() { return enviar({ preventDefault() {} }); },
    vencer() {
      const [id, { callback, demora }] = temporizadores.entries().next().value;
      assert.equal(demora, 12000);
      temporizadores.delete(id);
      callback();
    },
  };
}

async function asentar() {
  for (let indice = 0; indice < 8; indice++) await Promise.resolve();
}

function respuestaValida(referencia) {
  return { ok: true, json: async () => ({ data: {
    valido: true, titulo: "Referencia comprobada", mensaje: "Resultado de prueba",
    referencia, estado: "disponible", alcance: "público",
  } }) };
}

test("una petición pendiente vence, libera el formulario y permite reintentar sin perder la referencia", async () => {
  const pagina = entorno();
  const primerEnvio = pagina.enviar();
  assert.equal(pagina.boton.disabled, true);
  assert.equal(pagina.resultado.textContent, "Comprobando la referencia…");
  assert.equal(pagina.peticiones[0].ruta, "/api/publico/documentos/cotejo");
  assert.equal(pagina.peticiones[0].opciones.method, "POST");
  assert.equal(pagina.peticiones[0].opciones.credentials, "omit");

  pagina.vencer();
  await primerEnvio;
  assert.equal(pagina.peticiones[0].opciones.signal.aborted, true);
  assert.equal(pagina.boton.disabled, false);
  assert.equal(pagina.entrada.value, "REF-PRUEBA-0001");
  assert.match(pagina.resultado.innerHTML, /Puede volver a comprobar la referencia/);

  const reintento = pagina.enviar();
  pagina.peticiones[1].resolver(respuestaValida("REF-PRUEBA-0001"));
  await reintento;
  assert.equal(pagina.resultado.dataset.estado, "valido");
  assert.equal(pagina.boton.disabled, false);
  assert.equal(pagina.temporizadores.size, 0);
  const resultadoNuevo = pagina.resultado.innerHTML;

  pagina.peticiones[0].resolver({ ok: false, status: 404 });
  await asentar();
  assert.equal(pagina.resultado.innerHTML, resultadoNuevo);
});

test("una respuesta antigua no pisa otra comprobación iniciada antes del vencimiento", async () => {
  const pagina = entorno();
  const anterior = pagina.enviar();
  pagina.entrada.value = "REF-PRUEBA-0002";
  const actual = pagina.enviar();
  assert.equal(pagina.peticiones[0].opciones.signal.aborted, true);
  pagina.peticiones[1].resolver(respuestaValida("REF-PRUEBA-0002"));
  await actual;
  const resultadoNuevo = pagina.resultado.innerHTML;
  pagina.peticiones[0].resolver(respuestaValida("REF-PRUEBA-0001"));
  await anterior;
  assert.equal(pagina.resultado.innerHTML, resultadoNuevo);
  assert.match(resultadoNuevo, /REF-PRUEBA-0002/);
  assert.equal(pagina.boton.disabled, false);
});

test("el límite de espera cubre también una lectura JSON que queda pendiente", async () => {
  const pagina = entorno();
  const lectura = diferida();
  const envio = pagina.enviar();
  pagina.peticiones[0].resolver({ ok: true, json: () => lectura.promesa });
  await asentar();
  pagina.vencer();
  await envio;
  assert.equal(pagina.boton.disabled, false);
  assert.equal(pagina.resultado.dataset.estado, "error");
  lectura.resolver({ data: { valido: true } });
  await asentar();
  assert.equal(pagina.resultado.dataset.estado, "error");
});
