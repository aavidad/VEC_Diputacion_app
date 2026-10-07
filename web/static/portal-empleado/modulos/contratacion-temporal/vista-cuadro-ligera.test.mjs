import assert from "node:assert/strict";
import test from "node:test";
import { montarCuadroContratacionLigero } from "./vista-cuadro-ligera.js";

const fila = Object.freeze({
  expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-0001", version: 2,
  fase_clave: "analisis", estado_clave: "en_curso", centro_ref: "centro:rpt:600",
  categoria_ref: "categoria:auxiliar", creado_en: "2026-10-01T08:00:00Z",
  actualizado_en: "2026-10-01T09:00:00Z",
});

function raizFalsa() {
  const eventos = new Map();
  return { innerHTML: "", eventos, addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo) { eventos.delete(tipo); }, querySelector() { return null; } };
}

test("la bandeja ligera usa una consulta autorizada y abre detalle solo tras pulsar su referencia", async () => {
  const raiz = raizFalsa(), consultas = [], abiertos = [], errores = [];
  const montaje = await montarCuadroContratacionLigero({
    raiz,
    cliente: { consultarCuadroRRHH: async (solicitud, opciones) => {
      consultas.push({ solicitud, opciones });
      return { generada_en: "2026-10-01T09:00:00Z", expedientes: [fila], hay_mas: false };
    } },
    idioma: "es",
    abrirDetalle: async (datos) => { abiertos.push(datos); },
    mostrarError: (_raiz, datos) => { errores.push(datos); },
  });
  assert.equal(errores.length, 0, errores[0]?.error?.stack);
  assert.equal(consultas.length, 1);
  assert.equal(consultas[0].solicitud.paginacion.limite, 100);
  assert.match(raiz.innerHTML, /2026\/CT-0001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-exp-vista="alta"/u);
  assert.equal(abiertos.length, 0);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctExpAbrir: fila.expediente_ref } }) } });
  assert.equal(abiertos.length, 1);
  assert.equal(abiertos[0].expedienteRef, fila.expediente_ref);
  assert.equal(abiertos[0].textos.vista, "expediente");
  montaje.desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("un fallo de consulta conserva acción de reintento y una sola lectura por intento", async () => {
  const raiz = raizFalsa(), errores = [];
  let intentos = 0;
  const montaje = await montarCuadroContratacionLigero({
    raiz, idioma: "en", abrirDetalle: async () => {},
    cliente: { consultarCuadroRRHH: async () => {
      intentos++;
      if (intentos === 1) throw new Error("503 de prueba");
      return { generada_en: "2026-10-01T09:00:00Z", expedientes: [], hay_mas: false };
    } },
    mostrarError: (_raiz, datos) => { errores.push(datos); },
  });
  assert.equal(intentos, 1);
  assert.equal(errores.length, 1);
  await errores[0].reintentar();
  assert.equal(intentos, 2);
  assert.match(raiz.innerHTML, /There are no requests for this profile|No hay peticiones para este perfil/u);
  assert.match(raiz.innerHTML, /data-ct-exp-recargar/u);
  montaje.desmontar();
});
