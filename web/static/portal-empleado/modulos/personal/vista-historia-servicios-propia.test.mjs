import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { montarVistaHistoriaServiciosPropia } from "./vista-historia-servicios-propia.js";

function documento() {
  class Nodo {
    constructor(d, tipo = "div") { this.ownerDocument = d; this.tagName = tipo; this.children = []; this.dataset = {}; this.attributes = new Map(); this.listeners = new Map(); this.textContent = ""; }
    append(...hijos) { this.children.push(...hijos); for (const hijo of hijos) hijo.parent = this; }
    replaceChildren(...hijos) { this.children = []; this.append(...hijos); }
    setAttribute(nombre, valor) { this.attributes.set(nombre, valor); }
    addEventListener(tipo, fn) { this.listeners.set(tipo, fn); }
    focus() { this.ownerDocument.activeElement = this; }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((h) => h !== this); }
  }
  const d = { createElement: (tipo) => new Nodo(d, tipo), activeElement: null, hasFocus: () => true };
  const raiz = new Nodo(d); d.body = raiz; return raiz;
}
const nodos = (n) => [n, ...n.children.flatMap(nodos)];
const texto = (n) => nodos(n).map((x) => x.textContent).join(" ");
const buscar = (raiz, clave) => nodos(raiz).find((n) => Object.hasOwn(n.dataset, clave));
const filtros = { efectosDesde: "2024-01-01", efectosHasta: "2025-01-01" };
function datos() {
  const revision = (version) => ({ servicio_ref: "srv_AAAAAAAAAAAAAAAAAAAAAA", relacion_ref: "rel_CCCCCCCCCCCCCCCCCCCCCC", periodo_desde: "2019-01-01", periodo_hasta: "2019-12-31", dias_reconocidos: 365, estado: "reconocido", clase: "<dato fuente>", traza: { desde: "2024-01-01", registrada_en: "2024-03-01T09:00:00.000000Z", version, acto_ref: "acto:uno", fuente_ref: "fuente:uno", fuente_version: 1 } });
  return { historia: { corte: { efectos_desde: filtros.efectosDesde, efectos_hasta: filtros.efectosHasta, conocido_en: "2026-10-04T08:00:00.000000Z" }, cobertura: "parcial", revisiones: [revision(2), revision(1)] }, consultada_en: "2026-10-04T08:00:01.000000Z", recibo_ref: "aud_v3_abcdef0123456789abcdef0123456789" };
}
const completar = () => new Promise((r) => setImmediate(r));
const enviar = (raiz) => nodos(raiz).find((n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
function montar(cliente) {
  const raiz = documento(); const montaje = montarVistaHistoriaServiciosPropia({ raiz, cliente, ...filtros }); return { raiz, montaje };
}

test("sin cliente nominal no activa consulta ni inventa registros; al montar no hay red", () => {
  const { raiz } = montar(); assert.equal(buscar(raiz, "personalHistoriaConsultar").disabled, true);
  assert.match(texto(raiz), /aún no está disponible/u); assert.equal(nodos(raiz).filter((n) => n.tagName === "table").length, 0);
  let consultas = 0; montar({ consultar() { consultas++; } }); assert.equal(consultas, 0);
});

test("muestra revisiones anteriores y separa servicio, efectos, conocimiento y referencia propia", async () => {
  const llamadas = []; const { raiz } = montar({ async consultar(entrada) { llamadas.push(entrada); return datos(); } });
  const boton = buscar(raiz, "personalHistoriaConsultar"); boton.focus(); await enviar(raiz);
  assert.equal(llamadas.length, 1); assert.deepEqual(Object.keys(llamadas[0]), ["efectosDesde", "efectosHasta", "signal"]);
  const filas = nodos(raiz).filter((n) => n.tagName === "tbody")[0].children; assert.equal(filas.length, 2);
  assert.match(texto(raiz), /Información conocida hasta/u); assert.match(texto(raiz), /Consulta realizada/u);
  assert.match(texto(raiz), /Cobertura parcial/u); assert.match(texto(raiz), /Para confirmar antigüedad o derechos/u);
  assert.ok(nodos(raiz).some((n) => n.textContent === "<dato fuente>")); assert.equal(nodos(raiz).some((n) => n.innerHTML), false);
  assert.equal(raiz.ownerDocument.activeElement, buscar(raiz, "personalHistoriaResultado"));
  assert.ok(nodos(raiz).some((n) => n.className === "tabla-contenedor personal-ficha-tabla" && n.attributes.get("tabindex") === "0"));
});

test("error, denegación, sesión finalizada y exceso retiran filas y recibo anteriores sin otra consulta", async () => {
  for (const codigo of ["denegado", "sesion_caducada", "no_disponible", "excede_limite"]) {
    let consultas = 0; const { raiz } = montar({ async consultar() { if (++consultas === 1) return datos(); throw { codigo }; } });
    await enviar(raiz); assert.equal(nodos(raiz).some((n) => n.tagName === "table"), true);
    await enviar(raiz); assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false);
    assert.doesNotMatch(texto(raiz), /aud_v3_/u); assert.equal(consultas, 2);
    assert.ok(nodos(raiz).some((n) => n.attributes.get("role") === "alert"));
    if (codigo === "sesion_caducada") assert.match(texto(raiz), /Identifíquese de nuevo/u);
  }
});

test("cancelar y desmontar ignoran respuestas tardías aunque el proveedor ignore abort", async () => {
  for (const accion of ["cancelar", "desmontar"]) {
    let resolver, señal; const { raiz, montaje } = montar({ consultar(entrada) { señal = entrada.signal; return new Promise((r) => { resolver = r; }); } });
    const consulta = enviar(raiz);
    if (accion === "cancelar") buscar(raiz, "personalHistoriaCancelar").listeners.get("click")(); else montaje.desmontar();
    assert.equal(señal.aborted, true); resolver(datos()); await consulta; await completar();
    assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false);
    if (accion === "cancelar") assert.match(texto(raiz), /Consulta cancelada/u);
    else assert.equal(raiz.children.length, 0);
  }
});

test("fechas iguales no consultan y señalan el campo con foco; vacío no cambia cobertura", async () => {
  let llamadas = 0; const { raiz } = montar({ async consultar() { llamadas++; const r = datos(); r.historia.revisiones = []; r.historia.cobertura = "no_acreditada"; return r; } });
  const campos = nodos(raiz).filter((n) => n.tagName === "input"); campos[1].value = campos[0].value;
  await enviar(raiz); assert.equal(llamadas, 0); assert.equal(raiz.ownerDocument.activeElement, campos[0]);
  assert.equal(campos[0].attributes.get("aria-invalid"), "true"); campos[1].value = filtros.efectosHasta;
  await enviar(raiz); assert.equal(llamadas, 1); assert.match(texto(raiz), /La información puede estar incompleta/u); assert.match(texto(raiz), /No hay revisiones/u);
});

test("catálogos ES/EN completos: mismos campos, estados y mensajes de recuperación", () => {
  const cargar = (idioma) => JSON.parse(readFileSync(new URL(`../../../textos/${idioma}/personal-historia-servicios.json`, import.meta.url), "utf8"));
  const claves = (valor) => Object.entries(valor).flatMap(([k, v]) => typeof v === "string" ? [k] : claves(v).map((c) => `${k}.${c}`)).sort();
  const es = cargar("es"), en = cargar("en"); assert.deepEqual(claves(es), claves(en));
  assert.equal(es.general.hasta, "Efectos hasta (no incluido)"); assert.match(en.general.sesion_caducada, /Sign in again/u);
});

const abrirPreparacion = (raiz) => buscar(raiz, "personalRevisionPreparar").listeners.get("click")();
function rellenarPreparacion(raiz, textoPropuesto = "366", campo = "dias_reconocidos") {
  const selector = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "campo");
  selector.value = campo; selector.listeners.get("change")?.();
  for (const n of nodos(raiz).filter((n) => n.dataset.personalRevisionCampo)) {
    n.value = { campo, propuesta: textoPropuesto, motivo: "El periodo consta en el certificado", evidencia: "Certificado de servicios de 2020" }[n.dataset.personalRevisionCampo];
  }
}
const revisarPreparacion = (raiz) => buscar(raiz, "personalRevisionRevisar").parent.listeners.get("submit")({ preventDefault() {} });

test("fila real prepara y reconsulta con filtros propios antes de revisar, sin enviar borrador ni identidad", async () => {
  const llamadas = []; const { raiz } = montar({ async consultar(entrada) { llamadas.push(entrada); return datos(); } });
  await enviar(raiz); abrirPreparacion(raiz); rellenarPreparacion(raiz, "<img src=x onerror=alert(1)>", "clase");
  assert.match(texto(raiz), /Preparación sin presentar/u); await revisarPreparacion(raiz);
  assert.equal(llamadas.length, 2); assert.deepEqual(Object.keys(llamadas[1]), ["efectosDesde", "efectosHasta", "signal"]);
  assert.ok(nodos(raiz).some((n) => n.textContent === "<img src=x onerror=alert(1)>"));
  assert.match(texto(raiz), /no se ha presentado ni registrado/u); assert.equal(nodos(raiz).some((n) => n.innerHTML), false);
  assert.match(texto(raiz), /Ver procedencia del borrador/u); assert.match(texto(raiz), /Referencia del acto/u);
  assert.match(texto(raiz), /acto:uno/u); assert.match(texto(raiz), /Efectos hasta \(no incluido\)/u);
  assert.equal(buscar(raiz, "personalRevisionRevisar"), undefined);
});

test("valor recibido acompaña la propuesta y permanece al cambiar fecha, días, estado o texto", async () => {
  let llamadas = 0; const { raiz } = montar({ async consultar() { llamadas++; return datos(); } });
  await enviar(raiz); abrirPreparacion(raiz);
  const selector = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "campo");
  const actual = nodos(raiz).find((n) => n.textContent.startsWith("Valor recibido: "));
  const grupo = actual.parent;
  assert.equal(grupo.className, "personal-ficha-corte-campo");
  assert.equal(grupo.children[2].dataset.personalRevisionCampo, "propuesta");
  const formulario = grupo.parent;
  const motivo = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "motivo");
  assert.ok(formulario.children.indexOf(grupo) < formulario.children.indexOf(motivo.parent));
  for (const [campo, recibido, tipo] of [
    ["periodo_desde", "1 ene 2019", "date"], ["periodo_hasta", "31 dic 2019", "date"],
    ["dias_reconocidos", "365", "number"], ["estado", "Reconocido", "select"], ["clase", "<dato fuente>", "text"],
  ]) {
    selector.value = campo; selector.listeners.get("change")();
    const propuesta = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "propuesta");
    assert.equal(propuesta.parent, grupo);
    assert.equal(grupo.children[1], actual);
    assert.equal(grupo.children[2], propuesta);
    assert.equal(tipo === "select" ? propuesta.tagName : propuesta.type, tipo);
    assert.equal(actual.textContent, `Valor recibido: ${recibido}`);
    assert.equal(nodos(raiz).filter((n) => n.textContent.startsWith("Valor recibido: ")).length, 1);
  }
  assert.equal(llamadas, 1);
  buscar(raiz, "personalRevisionCancelar").listeners.get("click")();
  assert.equal(nodos(raiz).includes(actual), false);
});

test("propuesta guiada por dato, errores por campo y foco conservan lo escrito sin consultar", async () => {
  let llamadas = 0; const { raiz } = montar({ async consultar() { llamadas++; return datos(); } });
  await enviar(raiz); abrirPreparacion(raiz);
  assert.equal(nodos(raiz).find((n) => n.tagName === "form" && nodos(n).some((h) => h.dataset.personalRevisionCampo)).noValidate, true);
  const selector = buscar(raiz, "personalRevisionRevisar").parent.children
    .flatMap(nodos).find((n) => n.dataset.personalRevisionCampo === "campo");
  selector.value = "periodo_desde"; selector.listeners.get("change")();
  let propuesta = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "propuesta");
  assert.equal(propuesta.type, "date");
  selector.value = "dias_reconocidos"; selector.listeners.get("change")();
  propuesta = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "propuesta");
  assert.equal(propuesta.type, "number"); assert.equal(propuesta.min, "0"); assert.equal(propuesta.step, "1");
  const motivo = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "motivo");
  const evidencia = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "evidencia");
  propuesta.value = "-2"; motivo.value = "Mi certificado indica otro periodo"; evidencia.value = "";
  motivo.focus(); propuesta.listeners.get("blur")();
  assert.equal(llamadas, 1); assert.equal(raiz.ownerDocument.activeElement, motivo);
  const avisoPropuesta = nodos(raiz).find((n) => n.id === propuesta.attributes.get("aria-describedby"));
  assert.match(avisoPropuesta.textContent, /número entero de días/u);
  await revisarPreparacion(raiz);
  assert.equal(llamadas, 1); assert.equal(propuesta.value, "-2"); assert.equal(motivo.value, "Mi certificado indica otro periodo");
  assert.equal(raiz.ownerDocument.activeElement, propuesta);
  assert.equal(propuesta.attributes.get("aria-invalid"), "true");
  assert.equal(motivo.attributes.get("aria-invalid"), "false");
  assert.match(texto(raiz), /número entero de días/u); assert.match(texto(raiz), /Complete este campo/u);
  assert.match(texto(raiz), /El borrador sigue sin presentar/u);
  assert.equal(buscar(raiz, "personalRevisionRevisar").parent.parent.children[1].id, "personal-revision-error");
  const avisoEvidencia = nodos(raiz).find((n) => n.id === evidencia.attributes.get("aria-describedby"));
  assert.match(avisoEvidencia.textContent, /Complete este campo/u);
  const enlaceError = nodos(raiz).find((n) => n.tagName === "a" && n.href === "#personal-revision-propuesta");
  assert.ok(enlaceError); enlaceError.listeners.get("click")({ preventDefault() {} });
  assert.equal(raiz.ownerDocument.activeElement, propuesta);
  selector.value = "estado"; selector.listeners.get("change")();
  propuesta = nodos(raiz).find((n) => n.dataset.personalRevisionCampo === "propuesta");
  assert.equal(propuesta.tagName, "select");
  assert.deepEqual(propuesta.children.map((n) => n.textContent), ["Seleccione un estado", "Declarado", "Comprobado", "Reconocido"]);
  assert.equal(motivo.value, "Mi certificado indica otro periodo");
  assert.doesNotMatch(texto(raiz), /número entero de días/u);
  propuesta.value = "comprobado"; evidencia.value = "Certificado de servicios";
  await revisarPreparacion(raiz);
  assert.equal(llamadas, 2); assert.match(texto(raiz), /no se ha presentado ni registrado/u);
});

test("revisión sustituida, denegación y dependencia caída borran historia y borrador anteriores", async () => {
  for (const codigo of ["revision_sustituida", "denegado", "sesion_caducada", "no_disponible"]) {
    let llamadas = 0; const { raiz } = montar({ async consultar() {
      if (++llamadas === 1) return datos();
      if (codigo !== "revision_sustituida") throw { codigo };
      const r = datos(); r.historia.revisiones[0].traza.version = 3; return r;
    } });
    await enviar(raiz); abrirPreparacion(raiz); rellenarPreparacion(raiz); await revisarPreparacion(raiz);
    assert.equal(llamadas, 2); assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false);
    assert.equal(buscar(raiz, "personalRevisionRevisar"), undefined);
    assert.doesNotMatch(texto(raiz), /Certificado de servicios|aud_v3_|366/u);
  }
});

test("cancelación y desmontaje durante reconsulta ignoran respuestas tardías", async () => {
  for (const accion of ["cancelar", "desmontar"]) {
    let resolver, señal, llamadas = 0; const { raiz, montaje } = montar({ consultar({ signal }) {
      if (++llamadas === 1) return Promise.resolve(datos());
      señal = signal; return new Promise((r) => { resolver = r; });
    } });
    await enviar(raiz); abrirPreparacion(raiz); rellenarPreparacion(raiz); const revision = revisarPreparacion(raiz);
    assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false);
    if (accion === "cancelar") buscar(raiz, "personalRevisionCancelar").listeners.get("click")(); else montaje.desmontar();
    assert.equal(señal.aborted, true); resolver(datos()); await revision;
    assert.equal(buscar(raiz, "personalRevisionRevisar"), undefined);
    assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false); assert.doesNotMatch(texto(raiz), /Certificado de servicios/u);
  }
});

test("actualizar historia o cambiar filtros elimina preparación y sus campos", async () => {
  const { raiz } = montar({ async consultar() { return datos(); } });
  await enviar(raiz); abrirPreparacion(raiz); rellenarPreparacion(raiz);
  const camposAnteriores = nodos(raiz).filter((n) => n.dataset.personalRevisionCampo);
  await enviar(raiz); assert.ok(camposAnteriores.every((n) => n.value === ""));
  assert.equal(buscar(raiz, "personalRevisionRevisar"), undefined);
  abrirPreparacion(raiz); rellenarPreparacion(raiz);
  nodos(raiz).find((n) => n.dataset.personalHistoriaFecha).listeners.get("input")();
  assert.equal(buscar(raiz, "personalRevisionRevisar"), undefined);
});

test("cambiar fechas o cancelar consulta mientras revisa aborta y retira todos los datos anteriores", async () => {
  for (const accion of ["fecha", "consulta"]) {
    let resolver, señal, llamadas = 0;
    const { raiz } = montar({ consultar({ signal }) { if (++llamadas === 1) return Promise.resolve(datos()); señal = signal; return new Promise((r) => { resolver = r; }); } });
    await enviar(raiz); abrirPreparacion(raiz); rellenarPreparacion(raiz); const revision = revisarPreparacion(raiz);
    if (accion === "fecha") nodos(raiz).find((n) => n.dataset.personalHistoriaFecha).listeners.get("input")();
    else buscar(raiz, "personalHistoriaCancelar").listeners.get("click")();
    assert.equal(señal.aborted, true); resolver(datos()); await revision;
    assert.equal(nodos(raiz).some((n) => n.tagName === "table"), false);
    assert.equal(buscar(raiz, "personalHistoriaConsultar").disabled, false); assert.doesNotMatch(texto(raiz), /Certificado de servicios/u);
  }
});
