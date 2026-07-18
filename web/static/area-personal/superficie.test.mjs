import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import { dirname, extname, join, relative } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";
import { renderizarConvocatorias, renderizarDetalleConvocatoria, renderizarInicio } from "./vistas/inicio-convocatorias.js";
import { renderizarAutobaremacion, renderizarMeritos, renderizarPerfil, renderizarSolicitud } from "./vistas/perfil-meritos-solicitud.js";
import { renderizarAlegaciones, renderizarLlamamientos, renderizarSeguimiento, renderizarSubsanaciones } from "./vistas/seguimiento-tramites.js";
import { renderizarAyuda, renderizarCertificados, renderizarMensajes } from "./vistas/comunicaciones-ayuda.js";

const RAIZ = dirname(fileURLToPath(import.meta.url));

async function archivosEn(directorio) {
  const resultado = [];
  for (const entrada of await readdir(directorio, { withFileTypes: true })) {
    const ruta = join(directorio, entrada.name);
    if (entrada.isDirectory()) resultado.push(...await archivosEn(ruta));
    else resultado.push(ruta);
  }
  return resultado;
}

test("el launcher y el área personal comparten el selector único rrhh", async () => {
  const launcher = await readFile(join(RAIZ, "../presentacion/index.html"), "utf8");
  const contrato = await readFile(join(RAIZ, "contrato.js"), "utf8");
  assert.match(launcher, /\/area-personal\/\?presentacion=rrhh/u);
  assert.match(contrato, /get\("presentacion"\) === "rrhh"/u);
  assert.doesNotMatch(launcher, /area-personal\/\?presentacion=aspirante/u);
});

test("la superficie cubre todos los recorridos solicitados y conserva semántica", async () => {
  const html = await readFile(join(RAIZ, "index.html"), "utf8");
  const fuentes = (await Promise.all((await archivosEn(join(RAIZ, "vistas"))).map((ruta) => readFile(ruta, "utf8")))).join("\n");
  for (const texto of [
    "Inicio y plazos", "Convocatorias", "Perfil y contacto", "Méritos y documentos",
    "Nueva solicitud", "Autobaremación", "Mis expedientes", "Disponibilidad y llamamientos",
    "Subsanaciones", "Alegaciones", "Mensajes y noticias", "Certificados y descargas",
    "Ayuda y accesibilidad",
  ]) assert.match(`${html}\n${fuentes}`, new RegExp(texto, "u"), texto);
  for (const etiqueta of ["header", "nav", "main", "footer", "dialog", "form", "table", "fieldset", "label"]) {
    assert.match(`${html}\n${fuentes}`, new RegExp(`<${etiqueta}\\b`, "u"), etiqueta);
  }
  assert.match(html, /name="viewport" content="width=device-width, initial-scale=1"/u);
  assert.match(html, /Saltar al contenido principal/u);
});

test("no hay estado de negocio persistido, cookies, credenciales ni red externa", async () => {
  const produccion = (await archivosEn(RAIZ)).filter((ruta) => !ruta.endsWith(".test.mjs") && [".js", ".html"].includes(extname(ruta)));
  for (const ruta of produccion) {
    const contenido = await readFile(ruta, "utf8");
    assert.doesNotMatch(contenido, /\blocalStorage\b|\bsessionStorage\b|document\.cookie|credentials\s*:\s*["']include["']|\bAuthorization\b/u, relative(RAIZ, ruta));
    assert.doesNotMatch(contenido, /https?:\/\//u, relative(RAIZ, ruta));
    assert.doesNotMatch(contenido, /\b(?:[XYZ]\d{7}[A-Z]|\d{8}[A-Z])\b/iu, relative(RAIZ, ruta));
  }
});

test("la demo está aislada y el arranque normal solo compone HTTP", async () => {
  const arranque = await readFile(join(RAIZ, "arranque.js"), "utf8");
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(arranque, /if \(presentacion\)[\s\S]*import\("\.\/adaptador-presentacion\.js/u);
  assert.match(arranque, /crearClienteHTTPAreaPersonal/u);
  assert.doesNotMatch(aplicacion, /adaptador-presentacion|cliente-http/u);
  assert.doesNotMatch(arranque, /innerHTML\s*=\s*`[^`]*error\.message/su);
  assert.match(arranque, /detalle\.textContent\s*=\s*error instanceof Error/u);
  assert.match(await readFile(join(RAIZ, "index.html"), "utf8"), /id="aviso-presentacion" role="status" hidden/u);
});

test("los mismos renderizadores eliminan etiquetas de simulación en modo productivo", async () => {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  datos.meta.presentacion = false;
  const estado = {
    filtros: { termino: "", estado: "Todas", categoria: "Todas" },
    convocatoriaSeleccionada: datos.convocatorias[0].id,
    convocatoriaSolicitud: datos.convocatorias[0].id,
    expedienteSeleccionado: datos.solicitudes[0].id,
    pasoSolicitud: 5,
    operacionesSolicitud: {},
    consultaAyuda: "",
  };
  const superficies = [
    renderizarInicio(datos), renderizarConvocatorias(datos, estado), renderizarDetalleConvocatoria(datos, estado),
    renderizarPerfil(datos), renderizarMeritos(datos), renderizarSolicitud(datos, estado), renderizarAutobaremacion(datos),
    renderizarSeguimiento(datos, estado), renderizarLlamamientos(datos), renderizarSubsanaciones(datos),
    renderizarAlegaciones(datos), renderizarMensajes(datos), renderizarCertificados(datos), renderizarAyuda(datos, estado),
  ];
  for (const superficie of superficies) {
    for (const coincidencia of superficie.matchAll(/<button\b[^>]*>([\s\S]*?)<\/button>/giu)) {
      const etiqueta = coincidencia[1].replace(/<[^>]+>/gu, "").trim();
      assert.doesNotMatch(etiqueta, /\bDEMO\b|Simular/iu, etiqueta);
    }
  }
});

test("no existen estilos en línea ni botones sin tipo explícito", async () => {
  const produccion = (await archivosEn(RAIZ)).filter((ruta) => !ruta.endsWith(".test.mjs") && [".js", ".html"].includes(extname(ruta)));
  for (const ruta of produccion) {
    const contenido = await readFile(ruta, "utf8");
    assert.doesNotMatch(contenido, /\sstyle\s*=/iu, relative(RAIZ, ruta));
    for (const coincidencia of contenido.matchAll(/<button\b[^>]*>/giu)) {
      assert.match(coincidencia[0], /\btype="(?:button|submit)"/u, `${relative(RAIZ, ruta)}: ${coincidencia[0]}`);
    }
  }
});

test("los archivos se mantienen acotados y la UI cubre 390, 1024 y 1440", async () => {
  for (const ruta of await archivosEn(RAIZ)) {
    if (![".js", ".mjs", ".css", ".html"].includes(extname(ruta))) continue;
    const lineas = (await readFile(ruta, "utf8")).split("\n").length;
    assert.ok(lineas < 800, `${relative(RAIZ, ruta)} tiene ${lineas} líneas`);
  }
  const css = await readFile(join(RAIZ, "area-personal.css"), "utf8");
  assert.ok(css.split("\n").length < 500, "la hoja principal debe permanecer por debajo de 500 líneas");
  assert.match(css, /@media \(max-width: 1180px\)/u);
  assert.match(css, /@media \(max-width: 1480px\)[\s\S]*\.sesion-usuario > span:last-child \{ display: none; \}/u);
  assert.match(css, /@media \(max-width: 920px\)/u);
  assert.match(css, /@media \(max-width: 680px\)/u);
  assert.match(css, /\.acciones-tabla button, \.acciones-tabla a \{ min-height: 44px; \}/u);
  assert.match(css, /\.etiqueta-amplia \{ display: none; \}[\s\S]*\.etiqueta-corta \{ display: inline; \}/u);
  assert.match(css, /prefers-reduced-motion/u);
  assert.match(css, /html\[data-texto-grande="true"\] \{ font-size: 125%; \}/u);
  assert.match(css, /body\.area-personal-app \{[\s\S]*font-size: 1rem;/u);
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /accion === "alternar-texto" \? document\.documentElement : document\.body/u);
});
