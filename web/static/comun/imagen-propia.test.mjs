import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { cargarTextosImagen, crearAvatarCabecera, crearClienteImagen, crearSuperficieImagen, ErrorImagen, ICONOS_IMAGEN,
  PALETAS_IMAGEN, peticionesEnSerie, pintarAvatar, prepararFoto, TAMANO_MAXIMO_ENVIO, TAMANO_MAXIMO_SELECCION, validarEleccion } from "./imagen-propia.js";

const RUTA = "/api/vec/usuarios/mi-imagen";
const respuesta = (json, status = 200) => new Response(JSON.stringify(json), { status, headers: { "Content-Type": "application/json" } });
const aleatorio = { randomUUID: () => "123e4567-e89b-12d3-a456-426614174000" };
const catalogo = { version_ref: "usuarios-imagen-v1", paletas: PALETAS_IMAGEN.map((codigo) => ({ codigo, nombre_key: `ui.usuarios.imagen.paleta.${codigo}` })),
  iconos: ICONOS_IMAGEN.map((codigo) => ({ codigo, nombre_key: `ui.usuarios.imagen.icono.${codigo}` })), predeterminada: { modo: "iniciales", paleta: "azul", icono: "" } };
const vista = (eleccion, foto = null, version = 1) => ({ data: { catalogo, estado: { version, catalogo_version_ref: "usuarios-imagen-v1", eleccion }, foto } });
const FOTO = { tipo: "image/jpeg", datos: "/9j/4AAQSkZJRg==" };

test("el cliente usa la ruta exacta y valida catálogo, elección y foto", async () => {
  const peticiones = [];
  const cliente = crearClienteImagen({ ruta: RUTA, fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    return respuesta(vista({ modo: "foto", paleta: "verde", icono: "" }, FOTO));
  } });
  const v = await cliente.consultar();
  assert.equal(v.estado.eleccion.modo, "foto");
  assert.equal(v.foto.datos, FOTO.datos);
  assert.equal(peticiones[0].ruta, RUTA);
  assert.equal(peticiones[0].opciones.credentials, "same-origin");
  assert.equal(peticiones[0].opciones.method, "GET");
  for (const mala of [
    vista({ modo: "svg", paleta: "azul", icono: "" }),
    vista({ modo: "iniciales", paleta: "#ff0000", icono: "" }),
    vista({ modo: "icono", paleta: "azul", icono: "calavera" }),
    vista({ modo: "iniciales", paleta: "azul", icono: "" }, FOTO),
    vista({ modo: "foto", paleta: "azul", icono: "" }, { tipo: "image/svg+xml", datos: "PHN2Zz4=" }),
    vista({ modo: "foto", paleta: "azul", icono: "" }, { tipo: "image/jpeg", datos: "javascript:alert(1)" }),
    { data: { ...vista({ modo: "iniciales", paleta: "azul", icono: "" }).data, catalogo: { ...catalogo, iconos: [{ codigo: "calavera" }] } } },
  ]) {
    const otro = crearClienteImagen({ ruta: RUTA, fetchImpl: async () => respuesta(mala) });
    await assert.rejects(otro.consultar(), TypeError);
  }
  assert.throws(() => crearClienteImagen({ ruta: "/api/vec/usuarios/mis-correos", fetchImpl: async () => {} }), TypeError);
});

test("POST envía solo los campos de la operación y traduce los errores", async () => {
  let cuerpo;
  const cliente = crearClienteImagen({ ruta: RUTA, fetchImpl: async (_ruta, opciones) => {
    cuerpo = JSON.parse(opciones.body);
    return respuesta({ data: { recibo_ref: "img_" + "1".repeat(32), version: 2, replay: false, foto_nueva: true, foto_retirada: false } }, 201);
  } });
  const recibo = await cliente.guardar({ operacion: "subir_foto", version_esperada: 1, catalogo_version_ref: "usuarios-imagen-v1",
    clave_operacion: "web-imagen-1234567890ab", eleccion: { modo: "foto", paleta: "azul", icono: "" }, foto_base64: "AAAA" });
  assert.deepEqual(Object.keys(cuerpo).sort(), ["catalogo_version_ref", "clave_operacion", "eleccion", "foto_base64", "operacion", "version_esperada"]);
  assert.equal(recibo.foto_nueva, true);
  await assert.rejects(cliente.guardar({ operacion: "elegir", version_esperada: 1, catalogo_version_ref: "usuarios-imagen-v1",
    clave_operacion: "web-imagen-1234567890ab", eleccion: { modo: "iniciales", paleta: "azul", icono: "" }, foto_base64: "AAAA" }), TypeError);
  const grande = crearClienteImagen({ ruta: RUTA, fetchImpl: async () => respuesta({ error: { codigo: "foto_grande", clave_i18n: "api.usuarios.imagen.error.foto_grande" } }, 413) });
  await assert.rejects(grande.guardar({ operacion: "subir_foto", version_esperada: 1, catalogo_version_ref: "usuarios-imagen-v1",
    clave_operacion: "web-imagen-1234567890ab", eleccion: { modo: "foto", paleta: "azul", icono: "" }, foto_base64: "AAAA" }),
  (error) => error instanceof ErrorImagen && error.estado === 413 && error.codigo === "foto_grande");
  const comun = crearClienteImagen({ ruta: RUTA, fetchImpl: async () => new Response("request body too large", { status: 413, headers: { "Content-Type": "text/plain" } }) });
  await assert.rejects(comun.guardar({ operacion: "subir_foto", version_esperada: 1, catalogo_version_ref: "usuarios-imagen-v1",
    clave_operacion: "web-imagen-1234567890ab", eleccion: { modo: "foto", paleta: "azul", icono: "" }, foto_base64: "AAAA" }),
  (error) => error.estado === 413 && error.codigo === "foto_grande");
  assert.throws(() => validarEleccion({ modo: "iniciales", paleta: "azul", icono: "sol" }), TypeError);
});

test("la cola de peticiones no deja dos en vuelo a la vez", async () => {
  let enVuelo = 0;
  let maximo = 0;
  const orden = [];
  const lento = async (ruta) => {
    enVuelo += 1; maximo = Math.max(maximo, enVuelo);
    await new Promise((r) => setTimeout(r, 5));
    orden.push(ruta); enVuelo -= 1;
    if (ruta === "/falla") throw new TypeError("red");
    return respuesta({ ok: ruta });
  };
  const serie = peticionesEnSerie(lento);
  const resultados = await Promise.allSettled([serie("/a"), serie("/falla"), serie("/b")]);
  assert.equal(maximo, 1);
  assert.deepEqual(orden, ["/a", "/falla", "/b"]);
  assert.equal(resultados[1].status, "rejected");
  assert.deepEqual(await resultados[2].value.json(), { ok: "/b" });
});

test("la cola acota preferencias a 64 KiB sin recortar la respuesta de imagen", async () => {
  let llamadas = 0;
  const serie = peticionesEnSerie(async (ruta) => {
    llamadas++;
    if (ruta.endsWith("mis-preferencias")) return new Response("x".repeat(70 * 1024), { headers: { "Content-Type": "application/json" } });
    return new Response("x".repeat(200 * 1024), { headers: { "Content-Type": "application/json" } });
  });
  await assert.rejects(serie("/api/vec/usuarios/mis-preferencias", {}), /respuesta_serie_demasiado_grande/u);
  const imagen = await serie("/api/vec/usuarios/mi-imagen", {});
  assert.equal((await imagen.arrayBuffer()).byteLength, 200 * 1024);
  assert.equal(llamadas, 2);
});

test("cancelar un cuerpo bloqueado libera la cola para la siguiente petición", async () => {
  let leyendo;
  const lectura = new Promise((resolve) => { leyendo = resolve; });
  let llamadas = 0;
  const serie = peticionesEnSerie(async () => {
    llamadas++;
    if (llamadas === 1) return new Response(new ReadableStream({ pull() { leyendo(); return new Promise(() => {}); } }));
    return respuesta({ recuperado: true });
  });
  const controlador = new AbortController();
  const primera = serie("/api/vec/usuarios/mis-preferencias", { signal: controlador.signal });
  const segunda = serie("/api/vec/usuarios/mis-preferencias", {});
  await lectura;
  controlador.abort();
  await assert.rejects(primera, /respuesta_serie_cancelada/u);
  assert.deepEqual(await (await segunda).json(), { recuperado: true });
  assert.equal(llamadas, 2);
});

test("Content-Length excesivo no bloquea la cola si cancel no termina", { timeout: 2000 }, async () => {
  let primeraSignal;
  let segundaTrasAbortar = false;
  let llamadas = 0;
  const serie = peticionesEnSerie(async (_ruta, opciones) => {
    llamadas++;
    if (llamadas === 1) {
      primeraSignal = opciones.signal;
      return { status: 200, headers: { get: () => String(70 * 1024) }, body: { cancel: () => new Promise(() => {}) } };
    }
    segundaTrasAbortar = primeraSignal.aborted;
    return respuesta({ siguiente: true });
  });
  const primera = serie("/api/vec/usuarios/mis-preferencias", {});
  const segunda = serie("/api/vec/usuarios/mis-preferencias", {});
  await assert.rejects(primera, (fallo) => fallo.message === "respuesta_serie_demasiado_grande"
    && fallo.causa_limpieza?.codigo === "cancelacion_sin_acuse");
  assert.deepEqual(await (await segunda).json(), { siguiente: true });
  assert.equal(llamadas, 2);
  assert.equal(segundaTrasAbortar, true);
});

test("stream excesivo no bloquea la cola si reader.cancel no termina", { timeout: 2000 }, async () => {
  let primeraSignal;
  let segundaTrasAbortar = false;
  let llamadas = 0;
  const serie = peticionesEnSerie(async (_ruta, opciones) => {
    llamadas++;
    if (llamadas === 1) {
      primeraSignal = opciones.signal;
      return new Response(new ReadableStream({
        start(controlador) { controlador.enqueue(new Uint8Array(70 * 1024)); },
        cancel() { return new Promise(() => {}); },
      }), { headers: { "Content-Type": "application/json" } });
    }
    segundaTrasAbortar = primeraSignal.aborted;
    return respuesta({ siguiente: true });
  });
  const primera = serie("/api/vec/usuarios/mis-preferencias", {});
  const segunda = serie("/api/vec/usuarios/mis-preferencias", {});
  await assert.rejects(primera, (fallo) => fallo.message === "respuesta_serie_demasiado_grande"
    && fallo.causa_limpieza?.codigo === "cancelacion_sin_acuse");
  assert.deepEqual(await (await segunda).json(), { siguiente: true });
  assert.equal(llamadas, 2);
  assert.equal(segundaTrasAbortar, true);
});

test("un rechazo de cancelación conserva sólo la causa técnica cerrada", async () => {
  const serie = peticionesEnSerie(async () => ({ status: 200,
    headers: { get: () => String(70 * 1024) },
    body: { cancel: () => Promise.reject(new Error("DNI 12345678Z")) } }));
  await assert.rejects(serie("/api/vec/usuarios/mis-preferencias", {}), (fallo) =>
    fallo.message === "respuesta_serie_demasiado_grande"
      && fallo.causa_limpieza?.codigo === "cancelacion_fallida"
      && !JSON.stringify(fallo).includes("12345678Z"));
});

test("una señal ya cancelada no inicia fetch ni retiene la cola", async () => {
  let llamadas = 0;
  const serie = peticionesEnSerie(async () => { llamadas++; return respuesta({ ok: true }); });
  const controlador = new AbortController();
  controlador.abort();
  const primera = serie("/api/vec/usuarios/mis-preferencias", { signal: controlador.signal });
  const segunda = serie("/api/vec/usuarios/mis-preferencias", {});
  await assert.rejects(primera, /respuesta_serie_cancelada/u);
  assert.deepEqual(await (await segunda).json(), { ok: true });
  assert.equal(llamadas, 1);
});

test("un error del stream no conserva cause, pila ni propiedades libres", async () => {
  for (const mensaje of ["respuesta_serie_cancelada", "respuesta_serie_DNI_12345678Z"]) {
    const ajeno = new TypeError(mensaje, { cause: new Error("DNI 12345678Z") });
    ajeno.dato = "persona privada";
    ajeno.stack = "DNI 12345678Z en ruta privada";
    const serie = peticionesEnSerie(async () => new Response(new ReadableStream({ pull() { throw ajeno; } })));
    let recibido;
    await assert.rejects(serie("/api/vec/usuarios/mis-preferencias", {}), (fallo) => { recibido = fallo; return true; });
    assert.notStrictEqual(recibido, ajeno);
    assert.equal(recibido.message, mensaje === "respuesta_serie_cancelada" ? mensaje : "respuesta_serie_error");
    assert.equal(recibido.cause, undefined);
    assert.equal(recibido.dato, undefined);
    assert.doesNotMatch(recibido.stack, /12345678Z|persona privada|ruta privada/u);
    assert.ok([undefined, "cancelacion_fallida"].includes(recibido.causa_limpieza?.codigo));
  }
});

/** Elemento mínimo para pintar avatares sin DOM real. */
function elementoPrueba() {
  const clases = new Set(["avatar"]);
  const documento = {
    createElement: (tipo) => ({ tipo, alt: null, src: "", decoding: "" }),
    createElementNS: (_ns, tipo) => ({ tipo, atributos: {}, hijos: [], setAttribute(n, v) { this.atributos[n] = v; }, append(h) { this.hijos.push(h); } }),
    createTextNode: (texto) => ({ tipo: "#text", texto }),
  };
  return { ownerDocument: documento, dataset: {}, hijos: [{ tipo: "svg-generico" }], textContent: "",
    classList: { add: (...c) => c.forEach((x) => clases.add(x)), remove: (c) => clases.delete(c), has: (c) => clases.has(c) },
    replaceChildren(...h) { this.hijos = h; }, clases };
}

test("el avatar pinta iniciales, icono o foto sin HTML libre", () => {
  const el = elementoPrueba();
  pintarAvatar(el, { eleccion: { modo: "foto", paleta: "verde", icono: "" }, foto: FOTO, iniciales: "AR" });
  assert.equal(el.hijos[0].tipo, "img");
  assert.equal(el.hijos[0].alt, "");
  assert.equal(el.hijos[0].src, `data:image/jpeg;base64,${FOTO.datos}`);
  assert.ok(el.clases.has("avatar-imagen--verde"));
  pintarAvatar(el, { eleccion: { modo: "icono", paleta: "morado", icono: "sol" }, iniciales: "AR" });
  assert.equal(el.hijos[0].tipo, "svg");
  assert.equal(el.hijos[0].atributos["aria-hidden"], "true");
  assert.ok(!el.clases.has("avatar-imagen--verde") && el.clases.has("avatar-imagen--morado"));
  pintarAvatar(el, { eleccion: { modo: "foto", paleta: "gris", icono: "" }, foto: null, iniciales: "AR" });
  assert.deepEqual(el.hijos, [{ tipo: "#text", texto: "AR" }], "sin foto legible se muestran las iniciales");
  const cabecera = crearAvatarCabecera(el);
  cabecera.fijarImagen({ estado: { eleccion: { modo: "iniciales", paleta: "naranja", icono: "" } }, foto: null });
  cabecera.fijarIniciales("<b>X</b>Y");
  assert.equal(el.hijos[0].texto, "<b>", "las iniciales son texto, nunca HTML, y se acortan");
});

function contenedorPrueba() {
  const manejadores = {};
  const raiz = { innerHTML: "", querySelector: () => null, contains: () => true, toggleAttribute() {} };
  return { raiz, manejadores, addEventListener: (tipo, f) => { manejadores[tipo] = f; }, removeEventListener: () => {},
    querySelector: (selector) => (selector === "[data-imagen-raiz]" ? raiz : null) };
}

test("la superficie guarda la elección, avisa del borrado de la foto y repinta la cabecera", async () => {
  const textos = await cargarTextosImagen({ idioma: "es" });
  const enviados = [];
  let actual = vista({ modo: "foto", paleta: "azul", icono: "" }, FOTO, 3);
  const cliente = {
    consultar: async () => (await crearClienteImagen({ ruta: RUTA, fetchImpl: async () => respuesta(actual) }).consultar()),
    guardar: async (cuerpo) => { enviados.push(cuerpo); actual = vista(cuerpo.eleccion, null, 4); return { recibo_ref: "img_x", version: 4, foto_retirada: true }; },
  };
  const cambios = [];
  const s = crearSuperficieImagen({ cliente, textos, aleatorio, alCambiar: (v) => cambios.push(v), iniciales: () => "AR" });
  const c = contenedorPrueba();
  s.instalar(c);
  await s.cargar();
  assert.equal(cambios.length, 1);
  assert.match(c.raiz.innerHTML, /Mi imagen/u);
  assert.match(c.raiz.innerHTML, /data:image\/jpeg;base64,/u);
  assert.match(c.raiz.innerHTML, /type="file"[^>]*accept="image\/jpeg,image\/png,image\/webp"/u);
  assert.doesNotMatch(c.raiz.innerHTML, /per_|docimg_|<script/u);
  c.manejadores.change({ target: { name: "imagen-modo", value: "icono" } });
  assert.match(c.raiz.innerHTML, /Al guardar se borrará su foto/u);
  assert.match(c.raiz.innerHTML, /name="imagen-icono"/u);
  c.manejadores.change({ target: { name: "imagen-paleta", value: "#fff" } });
  c.manejadores.submit({ target: { matches: (s2) => s2 === "[data-imagen-form]" }, preventDefault() {} });
  await new Promise((r) => setTimeout(r, 0));
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(enviados.length, 1);
  assert.deepEqual(enviados[0].eleccion, { modo: "icono", paleta: "azul", icono: ICONOS_IMAGEN[0] });
  assert.equal(enviados[0].operacion, "elegir");
  assert.equal(enviados[0].version_esperada, 3);
  assert.equal(enviados[0].foto_base64, undefined);
  assert.match(c.raiz.innerHTML, /Su foto se ha borrado/u);
  assert.equal(cambios.length, 2, "tras guardar se vuelve a consultar y se repinta la cabecera");
});

test("sin foto elegida no se envía el modo foto y los textos existen en los dos idiomas", async () => {
  const textos = await cargarTextosImagen({ idioma: "es" });
  const enviados = [];
  const cliente = { consultar: async () => (await crearClienteImagen({ ruta: RUTA, fetchImpl: async () => respuesta(vista({ modo: "iniciales", paleta: "azul", icono: "" }, null, 0)) }).consultar()),
    guardar: async (cuerpo) => { enviados.push(cuerpo); return {}; } };
  const s = crearSuperficieImagen({ cliente, textos, aleatorio });
  const c = contenedorPrueba();
  s.instalar(c);
  await s.cargar();
  c.manejadores.change({ target: { name: "imagen-modo", value: "foto" } });
  assert.match(c.raiz.innerHTML, /Todavía no ha elegido ninguna foto/u);
  c.manejadores.submit({ target: { matches: () => true }, preventDefault() {} });
  assert.equal(enviados.length, 0);
  assert.match(c.raiz.innerHTML, /id="imagen-foto-error"/u);
  assert.match(c.raiz.innerHTML, /aria-describedby="imagen-foto-estado imagen-foto-limites imagen-foto-error"/u);
  assert.match(c.raiz.innerHTML, /aria-invalid="true"/u);
  const [es, en] = await Promise.all(["es", "en"].map(async (idioma) => JSON.parse(await readFile(new URL(`../textos/${idioma}/preferencias.json`, import.meta.url), "utf8")).imagen));
  assert.deepEqual(Object.keys(en).sort(), Object.keys(es).sort());
  for (const codigo of [...PALETAS_IMAGEN.map((p) => `paleta_${p}`), ...ICONOS_IMAGEN.map((i) => `icono_${i}`), "modo_iniciales", "modo_icono", "modo_foto"]) {
    assert.ok(es[codigo] && en[codigo], `falta ${codigo}`);
  }
});

test("la hoja de estilo usa solo tokens del tema y cubre todas las paletas", async () => {
  const css = await readFile(new URL("./imagen-propia.css", import.meta.url), "utf8");
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu, "sin colores fijos");
  for (const paleta of PALETAS_IMAGEN) assert.match(css, new RegExp(`\\.avatar-imagen\\.avatar-imagen--${paleta}[^}]*background: var\\(--avatar-${paleta}\\)`, "u"));
  const tema = await readFile(new URL("./tema-vec.css", import.meta.url), "utf8");
  for (const paleta of PALETAS_IMAGEN) {
    assert.equal((tema.match(new RegExp(`--avatar-${paleta}:`, "gu")) || []).length, 2, `--avatar-${paleta} en claro y en alto contraste`);
  }
  assert.doesNotMatch(tema.slice(tema.indexOf('html[data-tema="granate"]'), tema.indexOf("body[data-modo-color")), /--avatar-/u, "la marca no cambia los colores del avatar");
});

test("la foto se reduce en el navegador cuando se puede y se rechaza por tipo o tamaño", async () => {
  const fichero = (tipo, tamano) => ({ type: tipo, size: tamano, arrayBuffer: async () => new Uint8Array(tamano).buffer });
  const sinLienzo = {};
  const pequena = await prepararFoto(fichero("image/png", 30), sinLienzo);
  assert.equal(pequena.tipo, "image/png");
  assert.equal(pequena.base64, "A".repeat(40));
  await assert.rejects(prepararFoto(fichero("image/svg+xml", 30), sinLienzo), (e) => e.codigo === "foto_no_admitida");
  await assert.rejects(prepararFoto(fichero("image/gif", 30), sinLienzo), (e) => e.codigo === "foto_no_admitida");
  await assert.rejects(prepararFoto(fichero("image/jpeg", TAMANO_MAXIMO_SELECCION + 1), sinLienzo), (e) => e.codigo === "foto_grande");
  await assert.rejects(prepararFoto(fichero("image/jpeg", TAMANO_MAXIMO_ENVIO + 1), sinLienzo), (e) => e.codigo === "foto_grande");
  const dibujos = [];
  const conLienzo = {
    createImageBitmap: async (_f, opciones) => ({ width: 4000, height: 3000, opciones, close() {} }),
    OffscreenCanvas: class { constructor(w, h) { this.w = w; this.h = h; }
      getContext() { return { fillRect() {}, drawImage: (_m, x, y, w, h) => dibujos.push([w, h]) }; }
      async convertToBlob(o) { dibujos.push(o); return { size: 3, arrayBuffer: async () => new Uint8Array([0xff, 0xd8, 0xff]).buffer }; } },
  };
  const ilegible = { createImageBitmap: async () => { throw new Error("no es una imagen"); } };
  await assert.rejects(prepararFoto(fichero("image/jpeg", 3 * 1024 * 1024), ilegible), (e) => e.codigo === "foto_no_admitida",
    "un archivo que el navegador no sabe leer no es «demasiado grande»");
  const reducida = await prepararFoto(fichero("image/png", 9 * 1024 * 1024), conLienzo);
  assert.equal(reducida.tipo, "image/jpeg");
  assert.equal(reducida.base64, "/9j/");
  assert.deepEqual(dibujos[0], [1024, 768], "lado mayor de 1024 px, sin deformar");
  assert.deepEqual(dibujos[1], { type: "image/jpeg", quality: 0.9 });
});
