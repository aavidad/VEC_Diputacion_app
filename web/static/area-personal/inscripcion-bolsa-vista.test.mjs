import assert from "node:assert/strict";
import test from "node:test";
import { montarInscripcionBolsa } from "./inscripcion-bolsa-vista.js";

const pausa = () => new Promise((resolver) => setTimeout(resolver, 20));
const instante = "2026-10-09T09:00:00Z";
const bolsa = { convocatoria_ref: "convocatoria:plaza-42", titulo: "Bolsa de auxiliares 2026",
  categorias_resumen: "Auxiliar administrativo", categorias: [{ categoria_ref: "categoria:auxiliar", categoria: "Auxiliar administrativo" }],
  plazo_inicio: instante, plazo_fin: "2026-10-22T23:59:59Z", requisitos_resumen: "Titulación requerida",
  catalogo_version: 2, puede_iniciar: true, estado_solicitud_propia: null, solicitud_ref: null };
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
    abiertas: async () => ({ convocatorias: vacia ? [] : [bolsa], total: vacia ? 0 : 1, cursor_siguiente: null }),
    convocatoria: async () => ({ convocatoria: { ...bolsa, puede_iniciar: true,
      requisitos: [{ codigo: "titulo", descripcion: "Título exigido", obligatorio: true,
        estado: "pendiente", motivo_codigo: "sin_evaluacion", motivo_etiqueta: "Pendiente de comprobar" }] } }),
    propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
    detallePropio: async () => ({ solicitud: { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", categoria: bolsa.categorias[0].categoria,
      convocatoria_ref: bolsa.convocatoria_ref, estado: "pendiente", version: 1, registrada_en: instante } }),
    inscribir: async ({ claveIdempotencia }) => {
      claves.push(claveIdempotencia);
      if (claves.length === 1) throw Object.assign(new Error("red"), { status: 503 });
      return { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", convocatoria_ref: bolsa.convocatoria_ref,
        categoria: bolsa.categorias[0].categoria,
        estado: "pendiente", version: 1, registrada_en: instante, repetida: true };
    },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa();
  assert.match(contenedor.innerHTML, /no hay bolsas/u);
  vacia = false; pulsar("reintentar"); await pausa();
  assert.match(contenedor.innerHTML, /Auxiliar administrativo/u);
  pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
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
    convocatoria: async () => { throw new Error("no debe cargar convocatoria"); }, propias: async () => ({ solicitudes: [] }),
    detallePropio: async () => ({ solicitud: { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", categoria: bolsa.categorias[0].categoria,
      convocatoria_ref: "convocatoria:plaza-42", estado: "admitida_a_convocatoria", version: 2, registrada_en: instante } }),
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
    abiertas: async () => ({ convocatorias: [bolsa], total: 1, cursor_siguiente: null }),
    convocatoria: async () => ({ convocatoria: { ...bolsa, puede_iniciar: true, requisitos: [] } }),
    propias: async () => {
      lecturaPropia += 1;
      if (lecturaPropia === 2) throw Object.assign(new Error("denegado"), { status: 403 });
      return { solicitudes: [{ solicitud_ref: "solicitud:42", recibo_ref: "recibo:42",
        convocatoria_ref: bolsa.convocatoria_ref, categoria: bolsa.categorias[0].categoria, estado: "pendiente", version: 1,
        registrada_en: instante }], cursor_siguiente: null };
    },
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => new Promise((resolver) => { resolverPost = resolver; }),
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
  pulsar("revisar"); pulsar("confirmar"); await pausa();
  pulsar("volver"); await pausa();
  pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
  resolverPost({ solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", convocatoria_ref: bolsa.convocatoria_ref,
    estado: "pendiente", version: 1, registrada_en: instante, repetida: false });
  await pausa();
  assert.match(ventana.location.href, /id=convocatoria%3A/u);
  assert.doesNotMatch(contenedor.innerHTML, /Su solicitud de inscripción/u);
  assert.match(contenedor.innerHTML, /Consulte Mis solicitudes/u);
  pulsar("propias"); await pausa();
  assert.match(contenedor.innerHTML, /Auxiliar administrativo/u);
  pulsar("actualizar-propias"); await pausa();
  assert.doesNotMatch(contenedor.innerHTML, /recibo:42/u);
  assert.match(contenedor.innerHTML, /no permite hacer esta consulta/u);
  montaje.destruir();
});

test("una convocatoria de dos categorías exige elección antes de presentar", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  const varias = { ...bolsa, categorias_resumen: "Auxiliar administrativo y subalterno",
    categorias: [...bolsa.categorias, { categoria_ref: "categoria:subalterno", categoria: "Subalterno" }] };
  const enviados = [];
  const cliente = {
    abiertas: async () => ({ convocatorias: [varias], total: 1, cursor_siguiente: null }),
    convocatoria: async () => ({ convocatoria: { ...varias, requisitos: [] } }),
    propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async (datos) => { enviados.push(datos); return { solicitud_ref: "solicitud:subalterno", recibo_ref: "recibo:subalterno",
      convocatoria_ref: varias.convocatoria_ref, estado: "pendiente", version: 1, registrada_en: instante, repetida: false }; },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("bolsa", varias.convocatoria_ref); await pausa();
  assert.match(contenedor.innerHTML, /Elija una categoría/u);
  assert.match(contenedor.innerHTML, /data-inscripcion-accion="revisar" disabled/u);
  pulsar("revisar"); assert.doesNotMatch(contenedor.innerHTML, /Revise su solicitud/u);
  contenedor.handlers.change({ target: { matches: () => true, value: "categoria:subalterno" } });
  pulsar("revisar"); assert.match(contenedor.innerHTML, /Revise su solicitud/u);
  pulsar("confirmar"); await pausa();
  assert.equal(enviados.length, 1);
  assert.equal(enviados[0].categoriaRef, "categoria:subalterno");
  assert.match(contenedor.innerHTML, /Subalterno/u);
  montaje.destruir();
});

test("impedimento acreditado desactiva presentación y explica por qué", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  const cerrada = { ...bolsa, puede_iniciar: false, impedimento_etiqueta: "El plazo de inscripción ha terminado" };
  let envios = 0;
  const cliente = {
    abiertas: async () => ({ convocatorias: [cerrada], total: 1, cursor_siguiente: null }),
    convocatoria: async () => ({ convocatoria: { ...cerrada, requisitos: [] } }),
    propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => { envios += 1; },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("bolsa", cerrada.convocatoria_ref); await pausa();
  assert.match(contenedor.innerHTML, /El plazo de inscripción ha terminado/u);
  assert.match(contenedor.innerHTML, /data-inscripcion-accion="revisar" disabled/u);
  pulsar("revisar"); pulsar("confirmar"); await pausa();
  assert.equal(envios, 0);
  montaje.destruir();
});

test("403 tardío del POST borra la lista personal abierta después de navegar", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  let rechazarPost;
  const cliente = {
    abiertas: async () => ({ convocatorias: [bolsa], total: 1, cursor_siguiente: null }),
    convocatoria: async () => ({ convocatoria: { ...bolsa, requisitos: [] } }),
    propias: async () => ({ solicitudes: [{ solicitud_ref: "solicitud:secreta", recibo_ref: "recibo:secreto",
      convocatoria_ref: bolsa.convocatoria_ref, categoria: bolsa.categorias[0].categoria,
      estado: "pendiente", version: 1, registrada_en: instante }], cursor_siguiente: null }),
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => new Promise((_resolver, rechazar) => { rechazarPost = rechazar; }),
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
  pulsar("revisar"); pulsar("confirmar"); await pausa();
  pulsar("propias"); await pausa();
  assert.match(contenedor.innerHTML, /Auxiliar administrativo/u);
  rechazarPost(Object.assign(new Error("denegado"), { status: 403 }));
  await pausa();
  assert.doesNotMatch(contenedor.innerHTML, /solicitud:secreta|recibo:secreto|Auxiliar administrativo/u);
  assert.match(contenedor.innerHTML, /no permite hacer esta consulta/u);
  montaje.destruir();
});
