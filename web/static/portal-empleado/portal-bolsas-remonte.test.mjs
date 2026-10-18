import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
import { crearControladorBolsas, consultarCandidatosBolsa } from "./portal-bolsas-api.js";
import { ESQUEMA_CANDIDATOS } from "./portal-bolsas-contrato.js?v=20261007-pantallas-textos-final-v1";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";
import { crearCoordinadorModulosPortal } from "./portal-modulos-coordinador.js";
import { VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js?v=20261013-alegaciones-v2";

await prepararTextosPortal("bolsa");
await prepararMensajesContratos();

const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
const inicio = portal.indexOf("  montajeBolsa: Object.freeze({");
const fin = portal.indexOf("  confirmarOperacion:", inicio);
assert.ok(inicio > 0 && fin > inicio, "el montaje de Bolsa procede de portal.js");

const bolsa = {
  bolsa_ref: "bolsa:sintetica:administrativo", categoria_clave: "administrativo",
  categoria: "Administrativo", tipo_lista: "ordinaria", vigente_desde: "2026-01-01",
  vigente_hasta: null, total: 1,
  por_estado: { disponible: 1, no_disponible: 0, trabajando: 0,
    pendiente_incorporacion: 0, renuncia: 0, excluido: 0,
    disponible_desde: 0, en_revision: 0 },
  llamamientos_en_curso: 0,
  politica_orden: { politica_ref: "politica:orden:sintetica", version: 1,
    criterio: "puntuacion_desc_acta", tipo_lista: "rotatoria", reposicion: "misma_posicion",
    provisional: false, rotulo: "Orden según acta", actor: "sistema", vigente_desde: "2026-01-01" },
};
const respuestaCandidatos = { data: {
  esquema: ESQUEMA_CANDIDATOS, generado_en: "2026-10-16T00:00:00Z", bolsa,
  candidatos: [{ participacion_ref: "participacion:sintetica:01", orden: 1, orden_acta: 1,
    razon_orden: "orden_acta", nombre_visible: "Alicia Soto", documento_enmascarado: "***1234**",
    estado_clave: "disponible", estado_desde: "2026-01-01T00:00:00Z",
    disponible_desde: null, ultimo_llamamiento: null, contactos_total: 0 }],
  contactos: [], hay_mas: false, cursor_siguiente: null,
} };

function montajeRealDeBolsa(entorno) {
  // Evalúa solo el objeto de montaje ya usado por el portal; el coordinador,
  // controlador, cliente y presentador se importan de sus módulos reales.
  const objeto = portal.slice(inicio, fin);
  return runInNewContext(`({ ${objeto} confirmarOperacion: () => true }).montajeBolsa`, entorno);
}

async function esperarPeticion(peticiones, cantidad) {
  for (let intento = 0; intento < 20 && peticiones.length < cantidad; intento += 1) {
    await new Promise((resolver) => setImmediate(resolver));
  }
  assert.equal(peticiones.length, cantidad, "el GET de candidatos está pendiente");
}

test("repintar la misma vista conserva el GET; salir descarta la respuesta tardía", async () => {
  const estado = { vista: "bolsa-candidatos", bolsaSeleccionada: null, filtrosBolsa: {},
    datosBolsas: { carga: "listo", datos: { bolsas: [bolsa] } }, datosCandidatos: null };
  const raiz = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const peticiones = [];
  let renderizados = 0;
  let avisosCancelados = 0;
  const presentador = crearPresentadorPanelInterno({
    claseEstado: (clave) => clave, encabezadoVista: (_seccion, titulo) => `<header><h2>${titulo}</h2></header>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;"),
    numero: (valor) => String(valor ?? 0), obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: () => "", obtenerDatosBolsas: () => estado.datosBolsas,
    obtenerDatosCandidatosBolsa: () => estado.datosCandidatos,
    obtenerEstadoCandidatos: () => estado.filtrosBolsa, obtenerModalFicha: () => null,
  });
  const pintar = () => {
    renderizados += 1;
    if (estado.vista === "bolsa-candidatos") raiz.innerHTML = presentador.renderizarSoloBolsas(estado.vista);
  };
  const fetchPendiente = (ruta, opciones) => {
    assert.match(ruta, /\/api\/vec\/bolsa\/bolsas\/[^/]+\/candidatos/u);
    assert.equal(opciones.method, "GET");
    return new Promise((resolver) => peticiones.push({ resolver, signal: opciones.signal }));
  };
  const controlador = crearControladorBolsas({ estado, renderizar: pintar, navegar: () => {},
    documento: { querySelector: () => null },
    obtenerFuenteLectura: () => ({ consultarCandidatosBolsa: (referencia, filtros, { signal }) =>
      consultarCandidatosBolsa(referencia, filtros, { fetchImpl: fetchPendiente, signal }) }),
  });
  const montajeBolsa = montajeRealDeBolsa({ estado, controladorBolsas: controlador,
    VISTAS_INTERNAS_BOLSA, cancelarAvisosBolsa: () => { avisosCancelados += 1; },
    prepararBolsaFichaCT: () => assert.fail("la ficha de CT no interviene en este remonte"),
    montarVistaBolsa: (vista, contenedor) => {
      contenedor.innerHTML = vista === "bolsa-candidatos"
        ? presentador.renderizarSoloBolsas(vista) : '<section data-vista="resumen">Cuadro</section>';
    } });
  const coordinador = crearCoordinadorModulosPortal({ escaparHTML: String, montajeBolsa });

  assert.equal(await coordinador.montarVista("bolsa-candidatos", raiz), true);
  const primera = controlador.cargarCandidatosBolsa(bolsa.bolsa_ref);
  await esperarPeticion(peticiones, 1);
  assert.equal(estado.datosCandidatos.carga, "cargando");

  assert.equal(await coordinador.montarVista("bolsa-candidatos", raiz), true);
  assert.equal(peticiones[0].signal.aborted, false, "el repintado no cancela la lectura de la misma vista");
  assert.equal(avisosCancelados, 0);
  peticiones[0].resolver(new Response(JSON.stringify(respuestaCandidatos), { status: 200 }));
  await primera;
  assert.equal(estado.datosCandidatos.carga, "listo");
  assert.match(raiz.innerHTML, /Alicia Soto/u, "la respuesta 200 llega a la lista visible");

  const segunda = controlador.cargarCandidatosBolsa(bolsa.bolsa_ref);
  await esperarPeticion(peticiones, 2);
  estado.vista = "resumen";
  assert.equal(await coordinador.montarVista("resumen", raiz), true);
  assert.equal(peticiones[1].signal.aborted, true, "salir cancela la petición pendiente");
  assert.equal(avisosCancelados, 1);
  const renderizadosAlSalir = renderizados;
  const contenidoAlSalir = raiz.innerHTML;
  peticiones[1].resolver(new Response(JSON.stringify(respuestaCandidatos), { status: 200 }));
  await segunda;
  assert.equal(estado.datosCandidatos, null);
  assert.equal(renderizados, renderizadosAlSalir, "la respuesta tardía no vuelve a pintar candidatos");
  assert.equal(raiz.innerHTML, contenidoAlSalir);
  assert.doesNotMatch(raiz.innerHTML, /Alicia Soto/u);
  coordinador.desmontarVistaActual();
});
