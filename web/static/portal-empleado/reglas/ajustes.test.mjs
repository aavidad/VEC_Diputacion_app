import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { readFile, mkdtemp, rm } from "node:fs/promises";
import { existsSync } from "node:fs";
import { createServer } from "node:http";
import { tmpdir } from "node:os";
import { join, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { API_AJUSTES, ErrorAjustes, crearClienteAjustes, renderizarAjustes, validarLecturaAjustes, validarReciboAjustes } from "./ajustes.js";
import { cargarTextosAjustes } from "./ajustes-i18n.js";

await cargarTextosAjustes();

const lectura = () => ({ data: { esquema: "vec.contratacion_temporal.reglas.ajustes.v1",
  catalogo_id: "vec.contratacion_temporal.reglas.ajustes", version_esperada: 2, puede_ajustar: true,
  activacion: { estado: "activa" },
  motivos: [{ clave: "respuesta_rrhh_duda", texto_clave: "ajustesMotivo_respuesta_rrhh_duda" }],
  reglas: [{ clave: "c03.plazo_fiscalizacion", etiqueta: "Plazo de fiscalización", unidad: "dias_habiles", cantidad: 10,
    valores: { cantidad: "10", cantidad_urgente: "5", unidad: "dias_habiles" },
    edicion: { campos: ["cantidad", "cantidad_urgente"], opciones_unidad: ["dias_habiles"], opciones_computo: [],
      cantidad_minima: 1, cantidad_maxima: 30 }, ajuste_no_aplicable: false }],
  historial: [{ version: 2, vigente_desde: "2026-09-30T12:00:00Z", motivo_clave: "respuesta_rrhh_duda",
    referencia: "Duda 63", nota: "Revisado", recibo_ref: "recibo:ejemplo",
    cambios: [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad", anterior: "12", nuevo: "10" }] }], hay_mas: false } });

const recibo = () => ({ data: { esquema: "vec.contratacion_temporal.reglas.ajustes.v1", recibo: { recibo_ref: "recibo:syntetico", version: 3, vigente_desde: "2026-09-30T13:00:00Z",
  clave_idempotencia: "c3102148-38bb-4bbc-9b9b-b60b34521ec2", huella_sha256: "a".repeat(64),
  consumo_huella_sha256: "b".repeat(64), decision_ref: "decision:syntetica",
  auditoria_ref: "auditoria:syntetica" }, replay: false } });

const reglaNoAplicable = () => ({ clave: "c04.plazo_subsanacion", etiqueta: "Plazo de subsanación",
  ajuste_no_aplicable: true, edicion: { campos: ["cantidad"], opciones_unidad: [], opciones_computo: [],
    cantidad_minima: 1, cantidad_maxima: 30 } });

test("la lectura valida versión, campos, motivos catalogados y evita valores ambiguos", () => {
  assert.equal(validarLecturaAjustes(lectura()).reglas[0].valores.cantidad_urgente, "5");
  const sinMotivo = lectura(); sinMotivo.data.motivos[0].texto_clave = "desconocida";
  assert.throws(() => validarLecturaAjustes(sinMotivo), ErrorAjustes);
  const sinUrgente = lectura(); delete sinUrgente.data.reglas[0].valores.cantidad_urgente;
  assert.throws(() => validarLecturaAjustes(sinUrgente), ErrorAjustes);
  const otraVersion = lectura(); otraVersion.data.version_esperada = -1;
  assert.throws(() => validarLecturaAjustes(otraVersion), ErrorAjustes);
  const sinActivacion = lectura(); delete sinActivacion.data.activacion;
  assert.throws(() => validarLecturaAjustes(sinActivacion), ErrorAjustes);
  const activacionConHuella = lectura(); activacionConHuella.data.activacion.huella_sha256 = "a".repeat(64);
  assert.throws(() => validarLecturaAjustes(activacionConHuella), ErrorAjustes);
});

test("una base sin publicar o inactiva muestra historia sin afirmar valores vigentes", () => {
  for (const estado of ["sin_publicar", "inactiva"]) {
    const respuesta = lectura();
    respuesta.data.activacion.estado = estado;
    respuesta.data.puede_ajustar = false;
    respuesta.data.reglas = [];
    const modelo = validarLecturaAjustes(respuesta);
    const html = renderizarAjustes(modelo);
    assert.match(html, /Ver historial/u);
    assert.match(html, /Duda 63/u);
    assert.doesNotMatch(html, /Identificador de la regla:/u);
    assert.match(html, /Cantidad: 12 → 10/u);
    assert.match(html, /Revisado/u);
    assert.doesNotMatch(html, /data-ajustes-editar/u);
    assert.match(html, /Plazo de fiscalización · Cantidad: 12 → 10/u);
    assert.match(html, estado === "sin_publicar" ? /no hay plazos publicados/u : /plazos están desactivados/u);
    const falsa = lectura(); falsa.data.activacion.estado = estado;
    assert.throws(() => validarLecturaAjustes(falsa), ErrorAjustes);
  }
});

test("una regla con ajuste no aplicable omite valores sin ocultar las reglas válidas", async () => {
  const mixta = lectura();
  mixta.data.reglas.push(reglaNoAplicable());
  const cliente = crearClienteAjustes(async () => new Response(JSON.stringify(mixta), { status: 200 }));
  const datos = await cliente.leer();
  assert.equal(datos.reglas.length, 2);
  const html = renderizarAjustes(datos, { reglaActiva: "c04.plazo_subsanacion" });
  assert.match(html, /Plazo de fiscalización/u);
  assert.match(html, /10 Días hábiles/u);
  assert.match(html, /Plazo de subsanación/u);
  assert.match(html, /RRHH debe revisarlo/u);
  assert.match(html, /data-ajustes-editar="c04.plazo_subsanacion" disabled/u);
  assert.doesNotMatch(html, /data-ajustes-form="c04.plazo_subsanacion"/u);
  const sinEdicion = lectura();
  sinEdicion.data.reglas.push({ clave: "c04.plazo_subsanacion", etiqueta: "Plazo de subsanación", ajuste_no_aplicable: true });
  assert.equal(validarLecturaAjustes(sinEdicion).reglas.length, 2);
  for (const campo of ["valores", "cantidad", "cantidad_urgente", "unidad", "computo"]) {
    const filtrada = lectura();
    filtrada.data.reglas.push({ ...reglaNoAplicable(), [campo]: campo === "valores" ? { cantidad: "10" } : 10 });
    assert.throws(() => validarLecturaAjustes(filtrada), ErrorAjustes, campo);
  }
});

test("el cliente GET y POST conserva origen, clave y señal; distingue conflicto y dependencia", async () => {
  const pedidos = [];
  const cliente = crearClienteAjustes(async (url, opciones) => {
    pedidos.push({ url, opciones });
    return new Response(JSON.stringify(opciones.method === "GET" ? lectura() : recibo()), { status: opciones.method === "GET" ? 200 : 201 });
  });
  await cliente.leer({ antesDeVersion: 2 });
  assert.equal(pedidos[0].url, `${API_AJUSTES}?limite=20&antes_de_version=2`);
  assert.equal(pedidos[0].opciones.credentials, "same-origin");
  assert.equal(pedidos[0].opciones.cache, "no-store");
  const comando = { clave_idempotencia: "c3102148-38bb-4bbc-9b9b-b60b34521ec2", version_esperada: 2,
    cambios: [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad", nuevo: "7" }], motivo_clave: "respuesta_rrhh_duda" };
  assert.equal((await cliente.guardar(comando)).recibo.version, 3);
  const respuestaAjena = crearClienteAjustes(async () => new Response(JSON.stringify({ data: { ...recibo().data,
    recibo: { ...recibo().data.recibo, clave_idempotencia: "otra" } } }), { status: 201 }));
  await assert.rejects(respuestaAjena.guardar(comando), ErrorAjustes);
  assert.deepEqual(Object.keys(pedidos[1].opciones.headers), ["Accept", "Content-Type"]);
  assert.deepEqual(JSON.parse(pedidos[1].opciones.body), comando);
  for (const [estado, codigo] of [[409, "ajustesConflicto"], [422, "ajustesValorInvalido"], [403, "ajustesSinPermiso"], [503, "ajustesNoDisponible"]]) {
    const fallido = crearClienteAjustes(async () => new Response("{}", { status: estado }));
    await assert.rejects(fallido.guardar(comando), (error) => error.codigo === codigo);
  }
  assert.equal(validarReciboAjustes(recibo()).recibo.recibo_ref, "recibo:syntetico");
});

test("la pantalla muestra resumen, historia y recibo, escapa datos; desactiva cambios sin motivos", () => {
  const datos = validarLecturaAjustes(lectura());
  const html = renderizarAjustes(datos, { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision",
    borrador: { cantidad: "7", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda",
      referencia: "Acuerdo 12/2026", nota: "Revisión anual" }, recibo: recibo().data.recibo, aviso: "Cambio guardado" });
  assert.match(html, /Plazo de fiscalización/u);
  assert.match(html, /10 Días hábiles/u);
  assert.match(html, /10 Días hábiles → 7 Días hábiles/u);
  assert.match(html, /data-ajustes-revision/u);
  assert.match(html, /Respuesta de RRHH a una duda/u);
  assert.match(html, /Acuerdo 12\/2026/u);
  assert.match(html, /Revisión anual/u);
  assert.match(html, /data-ajustes-recibo/u);
  assert.match(html, /data-ajustes-estado/u);
  assert.match(html, /Duda 63/u);
  assert.match(html, /Persona de RRHH/u);
  assert.match(html, /<summary>Ver justificante<\/summary>/u);
  assert.doesNotMatch(html, /Auditoría:/u);
  const sinMotivos = { ...datos, motivos: [] };
  assert.match(renderizarAjustes(sinMotivos), /data-ajustes-editar="c03.plazo_fiscalizacion" disabled/u);
  assert.match(renderizarAjustes(datos, { bloqueado: true, error: true, aviso: "Conflicto" }), /data-ajustes-reintentar/u);
  const motivoRetirado = renderizarAjustes({ ...datos, motivos: [{ clave: "acuerdo_instruccion", texto_clave: "ajustesMotivo_acuerdo_instruccion" }] },
    { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision", borrador: { cantidad: "7", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda" } });
  assert.match(motivoRetirado, /Motivo anterior no disponible/u);
  assert.match(motivoRetirado, /data-ajustes-enviar disabled/u);
  const revision = { ...datos, reglas: [reglaNoAplicable()] };
  assert.doesNotMatch(renderizarAjustes(revision), /0 Días hábiles/u);
  assert.match(renderizarAjustes(revision), /RRHH debe revisarlo/u);
  const malicioso = { ...datos, reglas: [{ ...datos.reglas[0], clave: "c99.plazo_prueba", etiqueta: "<img src=x>" }] };
  assert.ok(!renderizarAjustes(malicioso).includes("<img src=x>"));
  assert.match(renderizarAjustes(malicioso), /&lt;img src=x&gt;/u);
  const camposMaliciosos = renderizarAjustes(datos, { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision",
    borrador: { cantidad: "7", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda",
      referencia: "<script>x</script>", nota: "<img src=x>" } });
  assert.ok(!camposMaliciosos.includes("<script>x</script>"));
  assert.ok(!camposMaliciosos.includes("<img src=x>"));
});

test("los textos de ajuste existen en ambos idiomas y el módulo no incluye frases visibles", () => {
  const ficheros = ["es", "en"].map((idioma) => JSON.parse(readFileSync(new URL(`../../textos/${idioma}/reglas-plazos.json`, import.meta.url), "utf8")).general);
  assert.deepEqual(Object.keys(ficheros[0]).sort(), Object.keys(ficheros[1]).sort());
  for (const texto of ["ajustesMotivo_respuesta_rrhh_duda", "ajustesConflicto", "ajustesEfecto", "ajustesAuditoria"]) {
    assert.ok(ficheros.every((fichero) => fichero[texto]), texto);
  }
});

test("los cuatro plazos editables y su historial usan el idioma elegido", async () => {
  await cargarTextosAjustes({ idioma: "en", porDefecto: "es" });
  try {
    const datos = validarLecturaAjustes(lectura());
    for (const [clave, nombre] of [
      ["c01.plazo_analisis", "Analysis deadline"],
      ["c02.plazo_informes", "Reports deadline"],
      ["c03.plazo_fiscalizacion", "Financial review deadline"],
      ["c04.plazo_subsanacion", "Correction deadline"],
    ]) {
      const html = renderizarAjustes({ ...datos, reglas: [{ ...datos.reglas[0], clave, etiqueta: "Nombre en español" }] });
      assert.match(html, new RegExp(`<h3[^>]*>${nombre}</h3>`, "u"));
      assert.doesNotMatch(html, /Nombre en español/u);
    }
    const sinBase = { ...datos, activacion: { estado: "sin_publicar" }, puede_ajustar: false, reglas: [] };
    assert.match(renderizarAjustes(sinBase), /Financial review deadline · Amount: 12 → 10/u);
  } finally { await cargarTextosAjustes({ idioma: "es", porDefecto: "es" }); }
});

test("un fallo del catálogo común no oculta los plazos CT cuya API responde", { skip: !existsSync("/usr/bin/google-chrome") }, async () => {
  const raiz = resolve(fileURLToPath(new URL("../../", import.meta.url)));
  let lecturasCT = 0;
  let fallosCatalogo = 0;
  let escenario = "comun";
  const rutasPedidas = [];
  const servidor = createServer(async (peticion, respuestaHTTP) => {
    const url = new URL(peticion.url, "http://127.0.0.1");
    const ruta = url.pathname;
    rutasPedidas.push(ruta);
    const responder = (estado, tipo, cuerpo) => {
      respuestaHTTP.writeHead(estado, { "Content-Type": tipo, "Cache-Control": "no-store" });
      respuestaHTTP.end(cuerpo);
    };
    if (ruta === "/textos/es/reglas.json" && escenario === "comun") {
      fallosCatalogo++;
      responder(503, "application/json", "{}");
      return;
    }
    if (ruta === "/textos/es/reglas-plazos.json" && escenario === "textos_ct") {
      responder(503, "application/json", "{}");
      return;
    }
    if (ruta === API_AJUSTES) {
      lecturasCT++;
      responder(escenario === "api_ct" ? 503 : 200, "application/json", JSON.stringify(lectura()));
      return;
    }
    const destino = resolve(raiz, `.${ruta === "/portal-empleado/reglas/" ? "/portal-empleado/reglas/index.html" : ruta}`);
    if (!destino.startsWith(raiz + sep)) { responder(404, "text/plain", ""); return; }
    try {
      let contenido = await readFile(destino);
      if (destino.endsWith("index.html") && url.searchParams.has("zoom")) {
        contenido = Buffer.from(contenido.toString().replace("</body>", `<script>setTimeout(() => {
          document.body.style.zoom = '2';
          document.documentElement.dataset.anchos = document.documentElement.scrollWidth + '/' + document.documentElement.clientWidth;
        }, 1500);</script></body>`));
      }
      const tipo = destino.endsWith(".js") ? "text/javascript" : destino.endsWith(".json") ? "application/json"
        : destino.endsWith(".css") ? "text/css" : destino.endsWith(".html") ? "text/html" : "application/octet-stream";
      responder(200, tipo, contenido);
    } catch { responder(404, "text/plain", ""); }
  });
  const cargarPagina = async (puerto, listo, zoom = false, idioma = "es") => {
    let salida = "";
    for (let intento = 0; intento < 3; intento++) {
      lecturasCT = 0;
      const perfil = await mkdtemp(join(tmpdir(), "vec-reglas-ct-"));
      try {
        const { stdout } = await promisify(execFile)("/usr/bin/google-chrome", ["--headless=new", "--no-sandbox",
          "--disable-gpu", "--disable-background-networking", "--no-proxy-server", "--no-first-run", `--user-data-dir=${perfil}`,
          "--window-size=390,844", "--virtual-time-budget=15000", "--dump-dom",
          `http://127.0.0.1:${puerto}/portal-empleado/reglas/?lang=${idioma}${zoom ? "&zoom=200" : ""}`],
        { timeout: 20000, maxBuffer: 4 * 1024 * 1024 });
        salida = stdout;
        if (listo.test(stdout)) return stdout;
      } finally { await rm(perfil, { recursive: true, force: true }); }
    }
    return salida;
  };
  try {
    await new Promise((listo) => servidor.listen(0, "127.0.0.1", listo));
    const puerto = servidor.address().port;
    const stdout = await cargarPagina(puerto, /data-ajustes-editar/u);
    assert.ok(fallosCatalogo > 0, `el catálogo común realmente falló: ${JSON.stringify(rutasPedidas)} ${stdout.match(/ERR_[A-Z_]+/u)?.[0] ?? ""}`);
    assert.equal(lecturasCT, 1, "el panel CT hizo una sola lectura");
    assert.match(stdout, /id="rg-ajustes"[\s\S]*Plazos de Contratación temporal/u);
    assert.match(stdout, /data-ajustes-editar="c03\.plazo_fiscalizacion"/u);
    assert.match(stdout, /id="rg-estado"[^>]*>El servicio no está disponible/u);
    escenario = "textos_ct";
    lecturasCT = 0;
    const sinTextosCT = await cargarPagina(puerto, /data-ajustes-catalogo-reintentar/u);
    assert.equal(lecturasCT, 0, "sin textos CT no se consulta la API");
    assert.match(sinTextosCT, /data-ajustes-catalogo-reintentar/u);
    assert.match(sinTextosCT, /id="rg-ajustes"[\s\S]*El servicio no está disponible/u);
    escenario = "api_ct";
    lecturasCT = 0;
    const sinAPICT = await cargarPagina(puerto, /data-ajustes-reintentar/u);
    assert.equal(lecturasCT, 1, "la API CT falló una vez");
    assert.match(sinAPICT, /data-ajustes-reintentar/u);
    assert.match(sinAPICT, /id="rg-ajustes"[\s\S]*No se han podido consultar/u);
    escenario = "normal";
    const conZoom = await cargarPagina(puerto, /data-anchos/u, true);
    const [, ancho, visible] = /data-anchos="(\d+)\/(\d+)"/u.exec(conZoom) ?? [];
    assert.ok(ancho && visible, "la página midió su ancho tras ampliar al 200 %");
    assert.ok(Number(ancho) <= Number(visible), `sin desbordamiento: ${ancho}/${visible}`);
    lecturasCT = 0;
    const ingles = await cargarPagina(puerto, /Financial review deadline/u, false, "en");
    assert.equal(lecturasCT, 1, "la pantalla inglesa hace una lectura CT");
    assert.match(ingles, /<h3[^>]*>Financial review deadline<\/h3>/u);
    assert.doesNotMatch(ingles, /<h3[^>]*>Plazo de fiscalización<\/h3>/u);
  } finally {
    await new Promise((listo) => servidor.close(listo));
  }
});
