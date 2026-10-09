import assert from "node:assert/strict";
import test from "node:test";
import { montarInscripcionBolsa } from "./inscripcion-bolsa-vista.js";

const pausa = () => new Promise((resolver) => setTimeout(resolver, 20));
const instante = "2026-10-09T09:00:00Z";
const bolsa = { convocatoria_ref: "convocatoria:plaza-42", titulo: "Bolsa de auxiliares 2026",
  numero_categorias: 1, categorias: [{ categoria_ref: "categoria:auxiliar", categoria: "Auxiliar administrativo" }],
  plazo_inicio: instante, plazo_fin: "2026-10-22T23:59:59Z", requisitos_resumen: "Titulación requerida",
  catalogo_version: 2, puede_iniciar: true, estado_solicitud_propia: null, solicitud_ref: null };
const { categorias: _categoriasDetalle, ...resumen } = bolsa;
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
  const declaracionesRecibidas = [];
  const cliente = {
    abiertas: async () => ({ convocatorias: vacia ? [] : [resumen], total: vacia ? 0 : 1, cursor_siguiente: null }),
    convocatoria: async () => ({ convocatoria: { ...bolsa, puede_iniciar: true,
      requisitos: [{ codigo: "titulo", descripcion: "Título exigido", obligatorio: true,
        estado: "pendiente", motivo_codigo: "sin_evaluacion", motivo_etiqueta: "Pendiente de comprobar" }] } }),
    propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
    detallePropio: async () => ({ solicitud: { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", categoria: bolsa.categorias[0].categoria,
      convocatoria_ref: bolsa.convocatoria_ref, estado: "pendiente", version: 1, registrada_en: instante } }),
    inscribir: async ({ claveIdempotencia, declaraciones }) => {
      claves.push(claveIdempotencia);
      declaracionesRecibidas.push(declaraciones);
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
  assert.match(contenedor.innerHTML, /1 categoría/u);
  pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
  assert.match(contenedor.innerHTML, /Auxiliar administrativo/u);
  assert.match(contenedor.innerHTML, /Título exigido/u);
  pulsar("revisar"); assert.match(contenedor.innerHTML, /Revise su solicitud/u);
  assert.match(contenedor.innerHTML, /data-inscripcion-requisito="titulo"[^>]*disabled/u);
  contenedor.handlers.change({ target: { matches: () => false, closest: () => ({
    dataset: { inscripcionRequisito: "titulo" }, checked: true,
  }) } });
  pulsar("confirmar"); await pausa();
  assert.match(contenedor.innerHTML, /No se ha podido comprobar el envío/u);
  pulsar("confirmar"); await pausa();
  assert.equal(claves.length, 2);
  assert.equal(claves[0], claves[1]);
  assert.deepEqual(declaracionesRecibidas, [[], []], "la revisión no cambia declaraciones a espaldas del resumen");
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
    abiertas: async () => ({ convocatorias: [resumen], total: 1, cursor_siguiente: null }),
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
  const varias = { ...bolsa, numero_categorias: 2,
    categorias: [...bolsa.categorias, { categoria_ref: "categoria:subalterno", categoria: "Subalterno" }] };
  const { categorias: _variasDetalle, ...resumenVarias } = varias;
  const enviados = [];
  const cliente = {
    abiertas: async () => ({ convocatorias: [resumenVarias], total: 1, cursor_siguiente: null }),
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
    abiertas: async () => ({ convocatorias: [{ ...resumen, puede_iniciar: false,
      impedimento_etiqueta: cerrada.impedimento_etiqueta }], total: 1, cursor_siguiente: null }),
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
    abiertas: async () => ({ convocatorias: [resumen], total: 1, cursor_siguiente: null }),
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

test("422 concluyente bloquea un segundo POST hasta volver a consultar la ficha", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  let envios = 0;
  const cliente = {
    abiertas: async () => ({ convocatorias: [resumen], total: 1, cursor_siguiente: null }),
    convocatoria: async () => ({ convocatoria: { ...bolsa, requisitos: [] } }),
    propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => { envios += 1; throw Object.assign(new Error("catálogo cambiado"), {
      status: 422, codigo: "catalogo_cambiado",
    }); },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
  pulsar("revisar"); pulsar("confirmar"); await pausa();
  assert.match(contenedor.innerHTML, /data-inscripcion-accion="confirmar" disabled/u);
  pulsar("confirmar"); await pausa();
  assert.equal(envios, 1);
  pulsar("volver"); await pausa(); pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
  pulsar("revisar"); pulsar("confirmar"); await pausa();
  assert.equal(envios, 2, "la ficha recargada permite intentar con versión nueva del servidor");
  montaje.destruir();
});

test("400 muestra datos no aceptados y ofrece actualizar la ficha", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  let consultas = 0;
  const cliente = {
    abiertas: async () => ({ convocatorias: [resumen], total: 1, cursor_siguiente: null }),
    convocatoria: async () => { consultas += 1; return { convocatoria: { ...bolsa, requisitos: [] } }; },
    propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => { throw Object.assign(new Error("rechazado"), { status: 400 }); },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("bolsa", bolsa.convocatoria_ref); await pausa();
  pulsar("revisar"); pulsar("confirmar"); await pausa();
  assert.match(contenedor.innerHTML, /No se han aceptado los datos/u);
  assert.doesNotMatch(contenedor.innerHTML, /Compruebe la conexión/u);
  assert.match(contenedor.innerHTML, /Actualizar convocatoria/u);
  pulsar("actualizar-ficha"); await pausa();
  assert.equal(consultas, 2);
  montaje.destruir();
});

test("la categoría 33 y la 128 siguen visibles y se envían por su referencia exacta", async (t) => {
  const categorias = Array.from({ length: 128 }, (_, indice) => ({
    categoria_ref: `categoria:${indice + 1}`, categoria: `Categoría ${indice + 1}`,
  }));
  for (const numero of [33, 128]) {
    await t.test(`categoría ${numero}`, async () => {
      const { contenedor, ventana, pulsar } = entorno();
      const convocatoria = { ...bolsa, categorias, numero_categorias: 128 };
      const { categorias: _completas, ...resumenConvocatoria } = convocatoria;
      const enviados = [];
      const cliente = {
        abiertas: async () => ({ convocatorias: [resumenConvocatoria], total: 1, cursor_siguiente: null }),
        convocatoria: async () => ({ convocatoria: { ...convocatoria, requisitos: [] } }),
        propias: async () => ({ solicitudes: [], cursor_siguiente: null }),
        detallePropio: async () => { throw new Error("sin uso"); },
        inscribir: async (datos) => { enviados.push(datos); return {
          solicitud_ref: `solicitud:${numero}`, recibo_ref: `recibo:${numero}`,
          convocatoria_ref: convocatoria.convocatoria_ref, estado: "pendiente", version: 1,
          registrada_en: instante, repetida: false,
        }; },
      };
      const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
      await pausa();
      assert.match(contenedor.innerHTML, /128 categorías/u);
      assert.doesNotMatch(contenedor.innerHTML, /Categoría 128/u, "la lista no precarga el detalle");
      pulsar("bolsa", convocatoria.convocatoria_ref); await pausa();
      assert.equal((contenedor.innerHTML.match(/<option value="categoria:/gu) ?? []).length, 128);
      assert.match(contenedor.innerHTML, /<option value="categoria:128"/u);
      assert.doesNotMatch(contenedor.innerHTML, /<table/u);
      assert.match(contenedor.innerHTML, /Buscar categoría/u);
      contenedor.handlers.input({ target: { matches: () => true, value: "Sin coincidencia", selectionStart: 15 } });
      assert.match(contenedor.innerHTML, /No hay categorías que coincidan/u);
      assert.match(contenedor.innerHTML, /data-inscripcion-categoria disabled/u);
      contenedor.handlers.input({ target: { matches: () => true, value: `Categoria ${numero}`, selectionStart: 12 } });
      assert.equal((contenedor.innerHTML.match(/<option value="categoria:/gu) ?? []).length, 1);
      assert.match(contenedor.innerHTML, new RegExp(`<option value="categoria:${numero}"`, "u"));
      contenedor.handlers.change({ target: { matches: () => true, value: `categoria:${numero}` } });
      pulsar("revisar");
      assert.match(contenedor.innerHTML, new RegExp(`Categoría ${numero}`, "u"));
      assert.match(contenedor.innerHTML, /data-inscripcion-buscar-categoria[^>]*disabled/u);
      assert.match(contenedor.innerHTML, /data-inscripcion-categoria disabled/u);
      contenedor.handlers.input({ target: { matches: () => true, value: "Sin coincidencia", selectionStart: 15 } });
      assert.match(contenedor.innerHTML, new RegExp(`Categoría ${numero}`, "u"));
      pulsar("confirmar"); await pausa();
      assert.equal(enviados.length, 1);
      assert.equal(enviados[0].categoriaRef, `categoria:${numero}`);
      montaje.destruir();
    });
  }
});

test("403 deniega sin ofrecer otro GET idéntico", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  let consultas = 0;
  const cliente = {
    abiertas: async () => { consultas += 1; throw Object.assign(new Error("denegado"), { status: 403 }); },
    convocatoria: async () => { throw new Error("sin uso"); },
    propias: async () => { throw new Error("sin uso"); },
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => { throw new Error("sin uso"); },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa();
  assert.match(contenedor.innerHTML, /Su acceso no permite hacer esta consulta/u);
  assert.doesNotMatch(contenedor.innerHTML, /data-inscripcion-accion="reintentar"/u);
  pulsar("reintentar"); await pausa();
  assert.equal(consultas, 1, "la denegación no repite la lectura");
  montaje.destruir();
});

test("503 permite reintentar la misma página de convocatorias", async () => {
  const { contenedor, ventana, pulsar } = entorno();
  const cursores = [];
  const cliente = {
    abiertas: async ({ cursor }) => {
      cursores.push(cursor);
      if (cursores.length === 2) throw Object.assign(new Error("temporal"), { status: 503 });
      if (cursor) return { convocatorias: [{ ...resumen, convocatoria_ref: "convocatoria:segunda",
        titulo: "Segunda convocatoria" }], total: 2, cursor_siguiente: null };
      return { convocatorias: [resumen], total: 2, cursor_siguiente: "cursor:2" };
    },
    convocatoria: async () => { throw new Error("sin uso"); },
    propias: async () => { throw new Error("sin uso"); },
    detallePropio: async () => { throw new Error("sin uso"); },
    inscribir: async () => { throw new Error("sin uso"); },
  };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa(); pulsar("mas"); await pausa();
  assert.match(contenedor.innerHTML, /La consulta no está disponible ahora/u);
  assert.match(contenedor.innerHTML, /data-inscripcion-accion="reintentar"/u);
  pulsar("reintentar"); await pausa();
  assert.deepEqual(cursores, ["", "cursor:2", "cursor:2"]);
  assert.equal((contenedor.innerHTML.match(/<article/gu) ?? []).length, 2);
  montaje.destruir();
});

test("sin la ruta publicada (404) avisa de que no está abierta y no ofrece reintentar", async () => {
  const { contenedor, ventana } = entorno();
  let consultas = 0;
  const cliente = { abiertas: async () => { ++consultas; throw Object.assign(new Error("ausente"), { status: 404 }); },
    convocatoria: async () => assert.fail(), propias: async () => assert.fail(),
    detallePropio: async () => assert.fail(), inscribir: async () => assert.fail() };
  const montaje = montarInscripcionBolsa({ contenedor, ventana, cliente, idioma: "es", textoBase: () => "Cargando" });
  await pausa();
  assert.equal(consultas, 1);
  assert.match(contenedor.innerHTML, /no está abierta en este momento/u);
  assert.doesNotMatch(contenedor.innerHTML, /data-inscripcion-accion="reintentar"/u);
  montaje.destruir();
});
