import test from "node:test";
import assert from "node:assert/strict";

import { consultarAvisosBolsa, manejarAccionAvisos, renderizarBloqueAvisos, validarAvisosBolsa } from "./portal-bolsas-avisos.js";

const datos = {
  esquema: "vec.bolsa.rrhh.avisos.v1",
  generado_en: "2026-09-23T09:30:00Z",
  provisionalidad: "Cómputo legal de encadenamiento pendiente de RRHH (duda 13; art. 15.5 ET tras la Ley 20/2021)",
  conteos: { salto_orden: 1, tres_anos: 1 },
  items: [
    { tipo: "salto_orden", bolsa: "bolsa:opaca", referencia: "aviso:salto:opaco", fecha: "2026-09-23T09:00:00Z", detalle: { llamamiento_ref: "llamamiento:opaco", participacion_ref: "participacion:uno", orden: 1, orden_primero_llamado: 2 } },
    { tipo: "tres_anos", bolsa: "bolsa:opaca", referencia: "aviso:tres:opaco", fecha: "2026-10-10T00:00:00Z", detalle: { participacion_ref: "participacion:dos", inicio_periodo_continuo: "2023-10-10T00:00:00Z", alcanza_tres_anos_en: "2026-10-10T00:00:00Z" } },
  ],
  paginacion: { limite: 6, desde: 1, hasta: 2, total: 2, cursor_siguiente: "" },
};

test("renderiza carga, vacío, error y lista R10 con contadores y ficha B5", () => {
  const carga = renderizarBloqueAvisos();
  assert.match(carga, /Cargando avisos/);
  assert.doesNotMatch(carga, /17\/09\/2026|Provisional:/);
  assert.match(renderizarBloqueAvisos({ estado: "error", error: "Fallo controlado" }), /Reintentar/);
  const vacio = renderizarBloqueAvisos({ estado: "listo", datos: { ...datos, items: [], conteos: { salto_orden: 0, tres_anos: 0 }, paginacion: { ...datos.paginacion, desde: 0, hasta: 0, total: 0 } } });
  assert.match(vacio, /Sin avisos/);
  assert.match(vacio, /0 saltos de orden/);
  assert.match(vacio, /0 avisos de tres años/);
  assert.match(vacio, /Provisional:<\/strong> Cómputo legal de encadenamiento pendiente de RRHH/);
  assert.doesNotMatch(vacio, /17\/09\/2026/);
  const html = renderizarBloqueAvisos({ estado: "listo", datos });
  assert.match(html, /<div class="cabecera-panel">[\s\S]*<h2>Avisos<\/h2>[\s\S]*1 salto de orden[\s\S]*1 aviso de tres años/);
  assert.match(html, /class="tabla-contenedor avisos-bolsa-lista" tabindex="0"/);
  assert.match(html, /Mostrando 1 a 2 de 2/);
  assert.match(html, /data-accion="abrir-ficha-b5"/);
  assert.match(html, /Cómputo legal de encadenamiento pendiente de RRHH/);
  assert.match(html, /class="avisos-bolsa-nota"/);
  assert.doesNotMatch(html, /17\/09\/2026/);
  assert.doesNotMatch(html, /nombre|DNI|correo|teléfono/i);
});

test("los contadores en plural conservan la provisionalidad exacta del servidor", () => {
  const provisionalidad = "Pendiente de revisión por RRHH; fuente sin fecha de inicio confirmada.";
  const html = renderizarBloqueAvisos({ estado: "listo", datos: {
    ...datos,
    provisionalidad,
    conteos: { salto_orden: 2, tres_anos: 3 },
  } });
  assert.match(html, /2 saltos de orden/);
  assert.match(html, /3 avisos de tres años/);
  assert.match(html, /Provisional:<\/strong> Pendiente de revisión por RRHH; fuente sin fecha de inicio confirmada\./);
  assert.doesNotMatch(html, /17\/09\/2026/);
});

test("consulta sin credenciales web y valida el contrato", async () => {
  let observada;
  const resultado = await consultarAvisosBolsa({ fetchImpl: async (ruta, opciones) => {
    observada = { ruta, opciones };
    return { ok: true, json: async () => ({ data: datos }) };
  } });
  assert.equal(resultado.ok, true);
  assert.equal(observada.opciones.credentials, "same-origin");
  assert.equal(observada.opciones.method, "GET");
  assert.match(observada.ruta, /^\/api\/vec\/bolsa\/avisos\?limite=6$/);
  assert.equal(validarAvisosBolsa({ data: datos }), datos);
});

test("enlaza cada aviso a la ficha B5 exacta", () => {
  const aperturas = [];
  const control = { dataset: { accion: "abrir-ficha-b5", bolsaRef: "bolsa:opaca", participacionRef: "participacion:uno" } };
  const consumida = manejarAccionAvisos({ target: { closest: () => control } }, { abrirFichaB5: (...argumentos) => aperturas.push(argumentos) });
  assert.equal(consumida, true);
  assert.deepEqual(aperturas[0].slice(0, 2), ["bolsa:opaca", "participacion:uno"]);
});

test("rechaza campos personales y tipos ajenos al contrato", () => {
  assert.throws(() => validarAvisosBolsa({ data: { ...datos, items: [{ ...datos.items[0], tipo: "otro" }] } }), /no válido/);
  const html = renderizarBloqueAvisos({ estado: "listo", datos: { ...datos, provisionalidad: "<script>alert(1)</script>" } });
  assert.doesNotMatch(html, /<script>/);
});
