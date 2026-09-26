import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearClienteSolicitudesSeleccion, RUTAS_SELECCION_RRHH } from "./cliente-http-solicitudes.js";
import { renderizarSolicitudesSeleccion } from "./vista-solicitudes.js";
import { MENSAJES_SELECCION_ES } from "./i18n.js";
import { crearCoordinadorModulosPortal } from "../../portal-modulos-coordinador.js";
import { crearServidorSimulado } from "../../../area-personal/solicitudes-servidor-simulado.test-helper.mjs";

const REF = "bolsa-operario-diputacion-2026";

test("la bandeja pagina por cursor y la ficha llega completa con mismo origen", async () => {
  const servidor = crearServidorSimulado();
  const cliente = crearClienteSolicitudesSeleccion({ fetchImpl: servidor.fetch });
  const convocatorias = await cliente.convocatorias();
  assert.equal(convocatorias[0].convocatoria_ref, REF);
  const primera = await cliente.consultar({ convocatoriaRef: REF, limite: 25 });
  assert.equal(primera.solicitudes.length, 25);
  assert.equal(primera.cursorSiguiente, "c:25");
  const segunda = await cliente.consultar({ convocatoriaRef: REF, cursor: primera.cursorSiguiente, limite: 25 });
  assert.equal(segunda.solicitudes.length, 5);
  assert.equal(segunda.cursorSiguiente, "");
  const ficha = await cliente.detalle(primera.solicitudes[0].solicitud_ref);
  assert.equal(ficha.requisitos[0].procedencia, "declarado_persona");
  assert.equal(ficha.historia.length, 2);
  const consulta = servidor.peticiones.find((item) => item.ruta === RUTAS_SELECCION_RRHH.consultas);
  assert.equal(consulta.metodo, "POST");
  const fuente = await readFile(new URL("./cliente-http-solicitudes.js", import.meta.url), "utf8");
  assert.match(fuente, /credentials: "same-origin", cache: "no-store",\s+redirect: "error", referrerPolicy: "no-referrer"/u);
  await assert.rejects(cliente.detalle("sol_inexistente"), (error) => error.codigo === "recurso_no_encontrado");
});

test("la vista muestra justificante y persona sin referencias internas y usa solo el catálogo", async () => {
  const servidor = crearServidorSimulado();
  const cliente = crearClienteSolicitudesSeleccion({ fetchImpl: servidor.fetch });
  const pagina = await cliente.consultar({ convocatoriaRef: REF, limite: 25 });
  const estado = { faseConvocatorias: "listo", convocatorias: await cliente.convocatorias(), convocatoriaRef: REF, turno: "", fase: "listo",
    filas: [...pagina.solicitudes], paginas: [""], indicePagina: 0, cursorSiguiente: pagina.cursorSiguiente, ficha: null, fichaFase: "" };
  const bandeja = renderizarSolicitudesSeleccion(estado);
  assert.match(bandeja, /2026\/SOL-000001/u);
  assert.match(bandeja, /Reyes Álvarez, Antonio/u);
  assert.match(bandeja, /data-seleccion-pagina="1"/u);
  assert.doesNotMatch(bandeja.replace(/data-seleccion-ficha="[^"]*"/gu, ""), /sol_rrhh/u);
  assert.match(renderizarSolicitudesSeleccion({ ...estado, turno: "discapacidad" }), /Jiménez Ortega/u);
  assert.doesNotMatch(renderizarSolicitudesSeleccion({ ...estado, turno: "discapacidad" }), /Reyes Álvarez/u);
  const ficha = renderizarSolicitudesSeleccion({ ...estado, ficha: await cliente.detalle(pagina.solicitudes[0].solicitud_ref) });
  assert.match(ficha, /Declarado por la persona/u);
  assert.match(ficha, /Meses trabajados en la Diputación/u);
  assert.doesNotMatch(ficha.replace(/data-copiar-justificante="[^"]*"/gu, ""), /sol_rrhh|meses_diputacion|declarado_persona/u);
  const fuente = await readFile(new URL("./vista-solicitudes.js", import.meta.url), "utf8");
  for (const [, clave] of fuente.matchAll(/\bt\("([a-z_]+)"/gu)) assert.equal(typeof MENSAJES_SELECCION_ES[clave], "string", clave);
  assert.doesNotMatch(fuente, />[¿¡A-ZÁÉÍÓÚÑ][a-záéíóúñ]+[^<$]*</u);
});

function catalogo(claves) {
  return async () => Object.freeze(claves.map((clave) => ({ clave })));
}
const temporizadores = { setTimeout, clearTimeout };

test("Selección solo se ofrece si el servidor publica el módulo y su consulta responde", async () => {
  const servidor = crearServidorSimulado();
  const ofrecido = crearCoordinadorModulosPortal({ escaparHTML: String, cargarCatalogoInterno: catalogo(["bolsa", "seleccion"]), entorno: { fetch: servidor.fetch }, temporizadores });
  await ofrecido.cargarInterno();
  assert.deepEqual(ofrecido.resolverAcceso("seleccion"), { disponible: true, vista: "seleccion" });
  assert.equal(ofrecido.vistaDisponible("seleccion"), true);

  const sinModulo = crearCoordinadorModulosPortal({ escaparHTML: String, cargarCatalogoInterno: catalogo(["bolsa"]), entorno: { fetch: servidor.fetch }, temporizadores });
  await sinModulo.cargarInterno();
  assert.equal(sinModulo.obtenerCatalogo().some((modulo) => modulo.clave === "seleccion"), false);
  assert.equal(sinModulo.vistaDisponible("seleccion"), false);

  const sinRuta = crearCoordinadorModulosPortal({ escaparHTML: String, cargarCatalogoInterno: catalogo(["seleccion"]), entorno: { fetch: crearServidorSimulado({ montado: false }).fetch }, temporizadores });
  await sinRuta.cargarInterno();
  assert.equal(sinRuta.resolverAcceso("seleccion").disponible, false);
});
