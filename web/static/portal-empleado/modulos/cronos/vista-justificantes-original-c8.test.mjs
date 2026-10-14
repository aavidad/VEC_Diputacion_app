import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { montarAporteOriginalC8, montarRevisionOriginalC8, montarHistoriaOriginalC8, montarBandejaOriginalC8 } from "./vista-justificantes-original-c8.js";

const SOLICITUD = "permiso:cronos:solicitud:abcdefgh";
const DOCUMENTO = `ref:${"a".repeat(64)}`;
const mensajes = JSON.parse(await readFile(new URL("../../../textos/es/cronos-justificacion-original-c8.json", import.meta.url), "utf8"));

function raizFalsa() {
  const nodos = [];
  const crearElement = (tag) => {
    const e = { tag, children: [], eventos: {}, hidden: false, disabled: false, textContent: "",
      append(...hijos) { this.children.push(...hijos); },
      replaceChildren(...hijos) { this.children = [...hijos]; },
      addEventListener(nombre, f) { this.eventos[nombre] = f; },
      removeEventListener(nombre) { delete this.eventos[nombre]; },
      setAttribute() {}, remove() { this.eliminado = true; }, focus() { this.enfocado = true; } };
    nodos.push(e); return e;
  };
  return { nodos, raiz: { ownerDocument: { createElement: crearElement }, append() {} } };
}

test("C8 reintenta el enlace con su misma clave sin repetir una subida admitida", async () => {
  const { nodos, raiz } = raizFalsa(); const cargas = []; const envios = [];
  let fallo = true;
  const vista = montarAporteOriginalC8({ raiz, solicitudRef: SOLICITUD, versionEsperada: 0, versionPermiso: 3,
    subirOriginal: async (_pdf, orden) => { cargas.push(orden); return { estado: "admitido", comprobante_ref: DOCUMENTO, documento_version: 1, recibo_ref: "recibo:documentos:abc" }; },
    cliente: { aportar: async (entrada) => { envios.push(entrada); if (fallo) throw Error("respuesta incierta");
      return { recibo_ref: "recibo:cronos:00000000-0000-4000-8000-000000000001" }; } }, mensajes });
  const file = nodos.find((e) => e.tag === "input"); file.files = [{ type: "application/pdf", size: 12 }];
  const form = nodos.find((e) => e.tag === "form");
  await form.eventos.submit({ preventDefault() {} });
  assert.equal(cargas.length, 1); assert.equal(envios.length, 1);
  assert.equal(cargas[0].solicitudRef, SOLICITUD);
  assert.equal(cargas[0].versionPermiso, 3, "el selector de carga conserva la versión del permiso");
  assert.equal(file.disabled, true, "el archivo queda fijado al original admitido para el reintento");
  fallo = false;
  await form.eventos.submit({ preventDefault() {} });
  assert.equal(cargas.length, 1, "el original admitido no se vuelve a cargar");
  assert.deepEqual(envios[1], envios[0], "el reintento conserva material y clave");
  assert.equal(nodos.find((e) => e.tag === "button").hidden, true);
  assert.match(nodos.find((e) => e.tag === "p").textContent, /recibo:cronos:/u);
  vista.desmontar();
});

test("C8 muestra los recibos históricos por GET y la bandeja abre solo el pendiente elegido", async () => {
  const propia = raizFalsa(); let consultas = 0; let permitida = true;
  const historia = montarHistoriaOriginalC8({ raiz: propia.raiz, solicitudRef: SOLICITUD,
    cliente: { consultarPropia: async () => { consultas++; if (!permitida) throw Error("permiso revocado"); return { historia: [{ estado: "pendiente_revision",
      recibo_ref: "recibo:cronos:00000000-0000-4000-8000-000000000001", instante_utc: "2026-10-14T18:00:00Z" }] }; } }, mensajes });
  await Promise.resolve(); await Promise.resolve();
  assert.equal(consultas, 1);
  assert.match(propia.nodos.find((e) => e.tag === "li").children.at(-1).textContent, /recibo:cronos:/u);
  permitida = false;
  await historia.recargar();
  assert.equal(propia.nodos.find((e) => e.tag === "ul").children.length, 0, "un permiso revocado retira los recibos anteriores");
  historia.desmontar();

  const rrhh = raizFalsa(); const abiertos = [];
  const pendiente = { solicitud_ref: SOLICITUD, empleado_etiqueta: "Elena Ruiz Martín",
    documento_id: DOCUMENTO, documento_version: 1, sha256: "b".repeat(64), registrada_en: "2026-10-14T18:00:00Z" };
  const bandeja = montarBandejaOriginalC8({ raiz: rrhh.raiz, cliente: { consultarBandeja: async () => ({ pendientes: [pendiente] }) },
    abrirRevision: (v) => abiertos.push(v), mensajes });
  await Promise.resolve(); await Promise.resolve();
  const boton = rrhh.nodos.find((e) => e.tag === "button" && e.textContent === mensajes.ver_justificante);
  boton.eventos.click();
  assert.deepEqual(abiertos, [pendiente]);
  bandeja.desmontar();
});

test("C8 no habilita revisión si la descarga no corresponde a la versión exacta", async () => {
  const { nodos, raiz } = raizFalsa(); let revisiones = 0;
  const vista = montarRevisionOriginalC8({ raiz, solicitudRef: SOLICITUD, versionEsperada: 1,
    documentoID: DOCUMENTO, documentoVersion: 2, documentoSHA256: "b".repeat(64), motivos: [{ ref: "motivo:documentacion:conforme", etiqueta: "Conforme" }],
    abrirOriginal: async () => ({ documento_id: DOCUMENTO, documento_version: 1, sha256: "b".repeat(64), estado: "admitido", abierto: true }),
    cliente: { revisar: async () => { revisiones++; return {}; } }, mensajes });
  const boton = nodos.find((e) => e.tag === "button");
  const form = nodos.find((e) => e.tag === "form");
  await boton.eventos.click();
  assert.equal(form.hidden, true);
  assert.equal(revisiones, 0);
  assert.match(nodos.find((e) => e.tag === "p").textContent, /No se pudo abrir el original/u);
  vista.desmontar();
});
