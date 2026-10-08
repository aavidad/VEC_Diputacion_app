import assert from "node:assert/strict";
import test from "node:test";
import { montarInscripcionBolsa } from "./inscripcion-bolsa-vista.js";

const pausa = () => new Promise((resolver) => setTimeout(resolver, 20));
const instante = "2026-10-09T09:00:00Z";
const bolsa = { bolsa_ref: "bolsa:plaza-42", categoria: "Auxiliar administrativo",
  plazo_inicio: instante, plazo_fin: "2026-10-22T23:59:59Z", requisitos_resumen: "Titulación requerida",
  catalogo_version: 2 };
function entorno(id = "") {
  const contenedor = { innerHTML: "", handlers: {}, addEventListener(tipo, fn) { this.handlers[tipo] = fn; },
    removeEventListener(tipo) { delete this.handlers[tipo]; }, replaceChildren() { this.innerHTML = ""; },
    contains() { return true; }, querySelector() { return null; } };
  const ventana = { location: { href: `https://vec.test/area-personal/?vista=inscripcion${id ? `&id=${encodeURIComponent(id)}` : ""}` },
    history: { pushState(_estado, _titulo, url) { ventana.location.href = new URL(url, ventana.location.href).href; } } };
  const pulsar = (accion, ref) => contenedor.handlers.click({ target: { closest: () => ({ dataset: {
    inscripcionAccion: accion, ref,
  } }) } });
  return { contenedor, ventana, pulsar };
}

test("lista vacía, plazo, revisión, reintento con la misma clave y recibo recuperable", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  let vacia = true;
  const claves = [];
  const cliente = {
    abiertas: async () => ({ bolsas: vacia ? [] : [bolsa], total: vacia ? 0 : 1, cursor_siguiente: null }),
    bolsa: async () => ({ bolsa: { ...bolsa, requisitos: [{ codigo: "titulo", descripcion: "Título exigido", obligatorio: true }] } }),
    propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
    detallePropio: async () => ({ solicitud: { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", categoria: bolsa.categoria,
      bolsa_ref: bolsa.bolsa_ref, estado: "pendiente", version: 1, registrada_en: instante } }),
    inscribir: async ({ claveIdempotencia }) => {
      claves.push(claveIdempotencia);
      if (claves.length === 1) throw Object.assign(new Error("red"), { status: 503 });
      return { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", bolsa_ref: bolsa.bolsa_ref, categoria: bolsa.categoria,
        estado: "pendiente", version: 1, registrada_en: instante, repetida: true };
    },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa();
  assert.match(contenedor.innerHTML, /no hay bolsas/u);
  vacia = false; pulsar("reintentar"); await pausa();
  assert.match(contenedor.innerHTML, /Auxiliar administrativo/u);
  pulsar("bolsa", bolsa.bolsa_ref); await pausa();
  assert.match(contenedor.innerHTML, /Título exigido/u);
  pulsar("revisar"); assert.match(contenedor.innerHTML, /Revise su solicitud/u);
  pulsar("confirmar"); await pausa();
  assert.match(contenedor.innerHTML, /No se ha podido completar/u);
  pulsar("confirmar"); await pausa();
  assert.equal(claves.length, 2);
  assert.equal(claves[0], claves[1]);
  assert.match(contenedor.innerHTML, /recibo:42/u);
  assert.match(ventana.location.href, /solicitud%3A42/u);
  montaje.destruir();
  const recuperado = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa();
  assert.match(contenedor.innerHTML, /Pendiente de revisión/u);
  assert.equal(claves.length, 2, "recuperar no repite el POST");
  recuperado.destruir();
});

test("carga únicamente el catálogo de la vista y traduce la admisión sin afirmar posición", async () => {
  const { contenedor, ventana } = entorno("solicitud:42");
  const cliente = { abiertas: async () => { throw new Error("no debe cargar listado"); },
    bolsa: async () => { throw new Error("no debe cargar bolsa"); }, propias: async () => ({ solicitudes: [] }),
    detallePropio: async () => ({ solicitud: { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", categoria: bolsa.categoria,
      bolsa_ref: "bolsa:plaza-42", estado: "admitida_a_convocatoria", version: 2, registrada_en: instante } }),
    inscribir: async () => { throw new Error("no debe inscribir"); } };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "en", textoBase: () => "Loading" });
  await pausa();
  assert.match(contenedor.innerHTML, /Awaiting entry into the job pool/u);
  assert.doesNotMatch(contenedor.innerHTML, /position|available/u);
  montaje.destruir();
});

test("un POST tardío no sustituye la misma ficha reabierta y 403 limpia solicitudes previas", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  let resolverPost;
  let lecturaPropia = 0;
  const cliente = {
    abiertas: async () => ({ bolsas: [bolsa], total: 1, cursor_siguiente: null }),
    bolsa: async () => ({ bolsa: { ...bolsa, requisitos: [] } }),
    propias: async () => {
      lecturaPropia += 1;
      if (lecturaPropia === 2) throw Object.assign(new Error("denegado"), { status: 403 });
      return { solicitudes: [{ solicitud_ref: "solicitud:42", recibo_ref: "recibo:42",
        bolsa_ref: bolsa.bolsa_ref, categoria: bolsa.categoria, estado: "pendiente", version: 1,
        registrada_en: instante }], cursor_siguiente: null };
    },
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => new Promise((resolver) => { resolverPost = resolver; }),
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("bolsa", bolsa.bolsa_ref); await pausa();
  pulsar("revisar"); pulsar("confirmar"); await pausa();
  pulsar("volver"); await pausa();
  pulsar("bolsa", bolsa.bolsa_ref); await pausa();
  resolverPost({ solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", bolsa_ref: bolsa.bolsa_ref,
    estado: "pendiente", version: 1, registrada_en: instante, repetida: false });
  await pausa();
  assert.match(ventana.location.href, /id=bolsa%3A/u);
  assert.doesNotMatch(contenedor.innerHTML, /Su solicitud de inscripción/u);
  assert.match(contenedor.innerHTML, /Consulte Mis solicitudes/u);
  pulsar("propias"); await pausa();
  assert.match(contenedor.innerHTML, /Auxiliar administrativo/u);
  pulsar("actualizar-propias"); await pausa();
  assert.doesNotMatch(contenedor.innerHTML, /recibo:42/u);
  assert.match(contenedor.innerHTML, /no permite hacer esta consulta/u);
  montaje.destruir();
});
