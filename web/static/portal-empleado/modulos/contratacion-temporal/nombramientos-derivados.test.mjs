import test from "node:test";
import assert from "node:assert/strict";
import { plazasDerivables, consultarHijos, montarNombramientosDerivados } from "./nombramientos-derivados.js";
import { crearClienteDerivacionHTTP, validarSolicitudDerivacion } from "./cliente-http-derivacion.js";

const solicitud = { padre_ref: "exp:padre", version_padre: 6, oferta_ref: "oferta:exacta",
  numero_plaza: 2, secuencia_plaza: 1, participacion_ref: "participacion:2",
  recibo_adjudicacion_ref: "recibo:plaza:2", clave_idempotencia: "78b8b011-cb15-4fd2-94d0-d6e358a457b6" };

test("solo deriva plazas cubiertas con aceptación previa y último recibo de adjudicación", () => {
  const recibo = { oferta_ref: "oferta:exacta", bolsa_ref: "bolsa:uno", numero_plazas: 2 };
  const oferta = { ...recibo, confirmacion_adjudicacion: "aceptacion_previa", plazas: [
    { numero_de_plaza: 1, estado: "vacante", secuencia: 0, historial: [] },
    { numero_de_plaza: 2, estado: "cubierta", secuencia: 1, participacion_ref: "participacion:2",
      historial: [{ tipo: "adjudicada", secuencia: 1, recibo_ref: "recibo:plaza:2" }] },
  ] };
  assert.deepEqual(plazasDerivables(oferta, recibo), [{ numero_plaza: 2, secuencia_plaza: 1,
    participacion_ref: "participacion:2", recibo_adjudicacion_ref: "recibo:plaza:2" }]);
  assert.throws(() => plazasDerivables({ ...oferta, oferta_ref: "oferta:otra" }, recibo));
  assert.throws(() => plazasDerivables({ ...oferta, plazas: [oferta.plazas[0],
    { ...oferta.plazas[1], historial: [{ tipo: "aceptada", secuencia: 1, recibo_ref: "recibo:otro" }] }] }, recibo));
});

test("consulta todas las páginas de hijos con filtro del número visible", async () => {
  const solicitudes = [];
  const cliente = { consultarCuadroRRHH: async (entrada) => {
    solicitudes.push(entrada);
    return solicitudes.length === 1
      ? { expedientes: [{ numero_visible: "2026/45-P1", expediente_ref: "exp:hijo1", version: 7 }], hay_mas: true, cursor_siguiente: "cursor" }
      : { expedientes: [{ numero_visible: "2026/45-P2", expediente_ref: "exp:hijo2", version: 6 }], hay_mas: false };
  } };
  const hijos = await consultarHijos(cliente, "2026/45");
  assert.deepEqual([...hijos.keys()], [1, 2]);
  assert.equal(solicitudes[0].filtros.texto, "2026/45-P");
  assert.equal(solicitudes[0].paginacion.limite, 100);
  assert.equal(solicitudes[1].paginacion.cursor, "cursor");
  await assert.rejects(consultarHijos({ consultarCuadroRRHH: async () => ({ expedientes: [], hay_mas: true }) }, "2026/45"));
});

test("el cliente usa el transporte CT y valida la confirmación del mismo padre y plaza", async () => {
  const llamadas = [];
  const cliente = crearClienteDerivacionHTTP({ validarOpciones: (o) => o,
    ejecutar: async (peticion) => {
      llamadas.push(peticion);
      return peticion.validarRespuesta({ esquema: "vec.ct.nombramiento.derivado.v1", estado: "confirmado",
        padre_ref: solicitud.padre_ref, hijo_ref: "exp:hijo2", numero_visible: "2026/45-P2",
        numero_plaza: 2, total_personas: 2, version: 6, aceptacion_ref: "aceptacion:2",
        recibo_ref: "recibo:derivacion:2", confirmada_en: "2026-10-18T08:00:00Z" });
    } });
  const recibo = await cliente.derivarNombramiento(solicitud, {});
  assert.equal(recibo.hijo_ref, "exp:hijo2");
  assert.equal(llamadas[0].ruta, "/api/vec/contratacion-temporal/nombramientos/derivaciones");
  assert.equal(llamadas[0].efecto, true);
  assert.deepEqual(Object.keys(validarSolicitudDerivacion(solicitud)), Object.keys(solicitud));
  assert.throws(() => llamadas[0].validarRespuesta({ ...recibo, numero_plaza: 1 }));
});

test("un catálogo fallido deja el panel visible y su reintento usa el idioma del montaje", async () => {
  const elemento = (tipo) => ({ tipo, hijos: [], atributos: {}, textContent: "", hidden: false,
    append(...nodos) { this.hijos.push(...nodos); },
    prepend(nodo) { this.hijos.unshift(nodo); },
    replaceChildren(...nodos) { this.hijos = nodos; },
    setAttribute(clave, valor) { this.atributos[clave] = valor; },
    addEventListener(_tipo, fn) { this.pulsar = fn; },
    remove() { this.retirado = true; },
  });
  const raiz = elemento("root");
  raiz.ownerDocument = { createElement: elemento };
  raiz.querySelector = () => null;
  const llamadas = [];
  const limpiar = montarNombramientosDerivados({ raiz,
    expediente: { expediente_ref: "exp:padre", numero_visible: "2026/45", version: 6,
      numero_personas: 2, demostracion: false },
    resumen: { version: 6, fase_clave: "fiscalizacion", estado_clave: "en_curso" },
    locale: "en-GB", traducirComun: (clave) => `common:${clave}`,
    cargarCatalogo: async (locale, reintentar) => {
      llamadas.push([locale, reintentar]);
      if (!reintentar) throw new Error("catalogo_no_disponible");
      return { titulo: "Appointments", simbolo_ayuda: "?", abrir_ayuda: "Help", ayuda: "Help text",
        cargando: "Loading", sin_oferta: "No linked offer" };
    },
    clienteCT: { consultarEstadoSeleccionOferta: async () => ({ estado: "ausente" }),
      consultarCuadroRRHH: async () => ({}), derivarNombramiento: async () => ({}) },
    clienteOfertas: { consultar: async () => ({}) },
  });
  await new Promise((resolver) => setImmediate(resolver));
  const panel = raiz.hijos[0];
  assert.equal(panel.atributos["data-ct-nombramientos"], "");
  const estado = panel.hijos[0].hijos[2];
  assert.equal(estado.hijos[0].textContent, "common:txt_el_servicio_no_esta_disponible_ahora_puede_reint");
  estado.hijos[1].pulsar();
  await new Promise((resolver) => setImmediate(resolver));
  assert.deepEqual(llamadas, [["en-GB", false], ["en-GB", true]]);
  assert.equal(panel.hijos[0].hijos[0].textContent, "Appointments");
  assert.equal(estado.hijos[0].textContent, "No linked offer");
  limpiar();
  assert.equal(panel.retirado, true);
});
