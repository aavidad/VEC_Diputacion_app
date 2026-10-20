import test from "node:test";
import assert from "node:assert/strict";
import { prepararTextosPortal } from "./portal-i18n.js?v=VEC_WEB_VERSION";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=VEC_WEB_VERSION";
await prepararTextosPortal("bolsa");
await prepararMensajesContratos();
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=VEC_WEB_VERSION";

// F1-25: con una sola persona elegida, la revisión del llamamiento no dice
// «1 seleccionados» ni «1 candidatos».
const BOLSA = { bolsa_ref: "bolsa:sintetica:1", categoria_clave: "administrativo", categoria: "Administrativo", tipo_lista: "rotatoria",
  vigente_desde: "2026-01-15", vigente_hasta: null, total: 2, por_estado: { disponible: 2 }, llamamientos_en_curso: 0,
  politica_orden: { politica_ref: "politica:1", version: 1, criterio: "puntuacion", tipo_lista: "rotatoria", reposicion: "misma_posicion", provisional: false, rotulo: "", actor: "actor:1", vigente_desde: "2026-01-15" } };
const persona = (n, nombre) => ({ participacion_ref: `participacion:sintetica:${n}`, orden: n, orden_acta: n, razon_orden: "orden_acta",
  nombre_visible: nombre, documento_enmascarado: `***${String(1000 + n)}**`, estado_clave: "disponible", estado_desde: "2026-09-01T08:00:00Z",
  disponible_desde: null, contactos_total: 0, ultimo_llamamiento: null });
const datos = { carga: "listo", error: "", datos: { bolsa: BOLSA, candidatos: [persona(1, "Antonio Reyes Álvarez"), persona(2, "Lucía Martín Serrano")],
  contactos: [], respuestas_portal: { llamamiento_ref: "", respuestas: [] }, hay_mas: false, cursor_siguiente: null } };

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
}
function revision(participaciones) {
  const flujo = { paso: 4, participaciones, canales: { correo: true, telefono: false },
    configuracion: { referencia: "2026/93001", centro: "Transformación Digital", modalidad: "Acumulación de tareas", plazo: "2 días hábiles" } };
  return crearPresentadorPanelInterno({
    claseEstado: () => "info", encabezadoVista: (_s, titulo) => `<header><h2>${titulo}</h2></header>`,
    escaparHTML: html, numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => datos, obtenerEstadoCandidatos: () => ({ nuevo_llamamiento: flujo }), obtenerModalFicha: () => null,
  }).renderizarVista("bolsa-candidatos");
}

test("una sola persona: la revisión habla en singular", () => {
  const vista = revision(["participacion:sintetica:1"]);
  assert.match(vista, /1 seleccionado,/u);
  assert.match(vista, /exactamente 1 candidato\. /u);
  assert.doesNotMatch(vista, /1 seleccionados|1 candidatos/u);
});

test("dos personas: la revisión sigue en plural", () => {
  const vista = revision(["participacion:sintetica:1", "participacion:sintetica:2"]);
  assert.match(vista, /2 seleccionados/u);
  assert.match(vista, /exactamente 2 candidatos\. /u);
});
