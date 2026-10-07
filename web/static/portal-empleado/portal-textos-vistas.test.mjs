import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";

const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");
const funcion = (nombre, siguiente) => {
  const inicio = fuente.indexOf(`function ${nombre}(`);
  const fin = fuente.indexOf(siguiente, inicio);
  assert.ok(inicio >= 0 && fin > inicio, `función ${nombre} disponible`);
  return fuente.slice(inicio, fin);
};
const vistaVigente = funcion("vistaVigenteParaTextos", "function esperarTextosDeVista(");
const esperar = funcion("esperarTextosDeVista", "function instalarReintentoTextosVista(");
const resumen = funcion("prepararResumenInicioVisible", "const cargasTextosVistas =");
const vaciarMicrotareas = () => new Promise((resolver) => setImmediate(resolver));

function escenario() {
  let resolver;
  const pinturas = [];
  const estado = { vista: "resumen" };
  const permiso = { disponible: true };
  const contexto = {
    estado, LOCALIZACION_PORTAL: "es-ES", cargasTextosVistas: new Map(),
    erroresTextosVistas: new Set(), gruposTextosMontables: new Set(),
    grupoTextosDeVista: (vista) => ["resumen", "estadisticas"].includes(vista) ? "bolsa" : null,
    vistaPermitida: () => permiso.disponible,
    prepararTextosBolsa: () => new Promise((resolve) => { resolver = resolve; }),
    prepararTextosPortal: () => { throw new Error("grupo equivocado"); },
    renderizar: () => pinturas.push(estado.vista),
  };
  runInNewContext(`${vistaVigente}\n${esperar}\nglobalThis.esperar = esperarTextosDeVista;`, contexto);
  return { contexto, estado, permiso, pinturas, resolver: (valor) => resolver(valor) };
}

test("dos subvistas de Bolsa comparten la carga y se repinta la que sigue abierta", async () => {
  const prueba = escenario();
  prueba.contexto.esperar("bolsa");
  prueba.estado.vista = "estadisticas";
  prueba.contexto.esperar("bolsa");
  prueba.resolver({ idioma: "es" });
  await vaciarMicrotareas();
  assert.deepEqual(prueba.pinturas, ["estadisticas"]);
  assert.equal(prueba.contexto.gruposTextosMontables.has("bolsa"), true);
});

test("la respuesta no repinta otra ruta, un perfil denegado ni otro idioma", async () => {
  for (const cambio of [
    (p) => { p.estado.vista = "portal"; },
    (p) => { p.permiso.disponible = false; },
    (p) => { p.contexto.LOCALIZACION_PORTAL = "en-GB"; },
  ]) {
    const prueba = escenario();
    prueba.contexto.esperar("bolsa");
    cambio(prueba);
    prueba.resolver({ idioma: "es" });
    await vaciarMicrotareas();
    assert.deepEqual(prueba.pinturas, []);
  }
});

test("Inicio RRHH muestra fallo si falta el contrato del coordinador", async () => {
  const contexto = {
    estado: { vista: "portal" }, esPerfilRRHH: () => true,
    coordinadorModulos: {}, pinturas: 0,
    renderizarConservandoFoco: () => { contexto.pinturas += 1; },
  };
  runInNewContext(`let estadoResumenInicio = "cargando"; let promesaResumenInicio = null;
    ${resumen}\nglobalThis.preparar = prepararResumenInicioVisible;
    globalThis.estadoActual = () => estadoResumenInicio;`, contexto);
  await contexto.preparar();
  assert.equal(contexto.estadoActual(), "error");
  assert.equal(contexto.pinturas, 1);
});
