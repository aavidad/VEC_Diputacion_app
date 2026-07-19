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
  assert.match(contrato, /getAll\("presentacion"\)[\s\S]*selectores\.length === 1 && selectores\[0\] === "rrhh"/u);
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
  assert.match(arranque, /if \(presentacion\)[\s\S]*import\("\.\.\/portal-empleado\/documentos\/descarga-recibos-presentacion\.js/u);
  assert.match(arranque, /crearClienteHTTPAreaPersonal/u);
  assert.doesNotMatch(aplicacion, /adaptador-presentacion|cliente-http/u);
  assert.doesNotMatch(arranque, /innerHTML\s*=\s*`[^`]*error\.message/su);
  assert.match(arranque, /detalle\.textContent\s*=\s*error instanceof Error/u);
  assert.match(await readFile(join(RAIZ, "index.html"), "utf8"), /id="aviso-presentacion" role="status" hidden/u);
});

test("las convocatorias del candidato reproducen el inventario público y aíslan la tramitación sintética", async () => {
  const datos = await crearAdaptadorPresentacion().cargar();
  const inventario = JSON.parse(await readFile(join(RAIZ, "../../../data/demo/convocatorias_publicas.demo.json"), "utf8"));
  const correspondencias = [
    ["DEMO-CONV-001", "proceso:publico:bolsa-operario-2026"],
    ["DEMO-CONV-002", "proceso:publico:auxiliar-servicios-generales-2025"],
    ["DEMO-CONV-003", "proceso:publico:gestion-administracion-general-2024"],
  ];

  for (const [idDemo, idPublico] of correspondencias) {
    const mostrada = datos.convocatorias.find((item) => item.id === idDemo);
    const publica = inventario.convocatorias.find((item) => item.id === idPublico)?.datos_publicos;
    assert.ok(mostrada && publica, `${idDemo} debe resolver su fuente pública`);
    assert.equal(mostrada.titulo, publica.titulo);
    assert.match(publica.resumen, new RegExp(mostrada.cve_bop, "u"));
    assert.equal(mostrada.publicada_en, new Intl.DateTimeFormat("es-ES", {
      day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC",
    }).format(new Date(publica.publicada_en)));
    assert.deepEqual(mostrada.documentos.map((item) => item.url), publica.documentos.map((item) => item.url));
    if (idDemo === "DEMO-CONV-003") assert.equal(mostrada.estado, "Histórico cerrado");
    else assert.equal(mostrada.estado, "Plazo abierto · DEMO");
  }

  assert.equal(datos.resumen.convocatorias_abiertas, 2);
  assert.equal(datos.convocatorias.filter((item) => item.recorrido_demo === true).length, 2);
  const gestion = datos.convocatorias.find((item) => item.id === "DEMO-CONV-003");
  assert.equal(gestion.titulo, "Ingreso en la Subescala de Gestión de Administración General");
  assert.equal(gestion.categoria, "Técnico de Gestión");
  assert.doesNotMatch(`${gestion.titulo} ${gestion.descripcion}`, /Archivo|Analista Programador/u);
  assert.ok(datos.solicitudes.every((item) => item.id.startsWith("DEMO-") && item.referencia.startsWith("DEMO-")));
  assert.ok(datos.plazos.every((item) => item.id.startsWith("DEMO-")));
  assert.ok(datos.llamamientos.every((item) => item.id.startsWith("DEMO-") && /DEMO|demostración/iu.test(`${item.bolsa} ${item.posicion}`)));
  const detalle = renderizarDetalleConvocatoria(datos, { convocatoriaSeleccionada: "DEMO-CONV-001" });
  assert.match(detalle, /href="\/bolsa\/documentos\/bases-operario-demo\.pdf"/u);
  assert.match(detalle, /BOP-GRA-2026-043004/u);
  assert.match(detalle, /Simular solicitud DEMO/u);
  assert.doesNotMatch(detalle, /\[object Object\]/u);
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

test("el menú móvil gestiona foco, Escape y contención de teclado", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /\.ap-navegacion a\[href\]["']\)\?\.focus/);
  assert.match(aplicacion, /function mantenerFocoEnMenu\(evento\)/);
  assert.match(aplicacion, /evento\.key !== "Escape"[\s\S]{0,220}cerrarMenu\(\{ restaurarFoco: true \}\)/);
});

test("la lectura por voz nunca expone un expediente a una voz remota", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  const inicio = aplicacion.indexOf("function leerPantalla(estado)");
  const fin = aplicacion.indexOf("function alternarPreferencia", inicio);
  const funcion = aplicacion.slice(inicio, fin);
  assert.ok(inicio >= 0 && fin > inicio, "debe existir la lectura gobernada");
  assert.match(funcion, /meta\?\.presentacion !== true[\s\S]*conector y una política aprobados[\s\S]*return;/u);
  assert.match(funcion, /localService === true[\s\S]*\^es\(\?:-\|\$\)\/i/u);
  assert.match(funcion, /if \(!voz\)[\s\S]*No hay una voz local en español[\s\S]*return;/u);
  assert.match(funcion, /locucion\.voice = voz/u);
  assert.ok(funcion.indexOf("meta?.presentacion !== true") < funcion.indexOf("innerText"), "el bloqueo real debe preceder a la lectura del contenido");
  const ayuda = await readFile(join(RAIZ, "vistas/comunicaciones-ayuda.js"), "utf8");
  assert.match(ayuda, /<audio[\s\S]*ayuda-llamamiento-bolsa\.mp3[\s\S]*Leer la transcripción/u);
});

test("el registro final exige declaración y una referencia exacta de solicitud", async () => {
  const adaptador = crearAdaptadorPresentacion();
  const ejecutar = (accion, payload) => adaptador.ejecutar({ accion, payload, confirmacion: true, capacidad: true });
  await ejecutar("guardar_borrador", {
    convocatoria_id: "DEMO-CONV-001",
    requisitos_confirmados: true,
    datos_confirmados: true,
    meritos_ids: ["DEMO-MER-001"],
    autobaremo_revisado: true,
  });
  await ejecutar("iniciar_pago", { id: "DEMO-SOL-BORRADOR-0001" });
  await ejecutar("firmar_solicitud", { id: "DEMO-SOL-BORRADOR-0001" });
  const datos = await adaptador.cargar();
  const html = renderizarSolicitud(datos, {
    pasoSolicitud: 5,
    convocatoriaSolicitud: "DEMO-CONV-001",
    solicitudEdicionId: "DEMO-SOL-BORRADOR-0001",
    progresoSolicitud: {},
    errorPasoSolicitud: "",
  });
  assert.match(html, /data-operacion="registrar_solicitud" data-id="DEMO-SOL-BORRADOR-0001"/u);
  assert.match(html, /name="declaracion_final" value="true" required/u);
  assert.doesNotMatch(await readFile(join(RAIZ, "adaptador-presentacion.js"), "utf8"), /solicitudes\[0\]/u);
});

test("la composición conserva el modo en enlaces y bloquea capacidades antes del diálogo", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /function actualizarEnlacesNavegacion\(estado\)[\s\S]*setAttribute\("href", crearURL\(estado/u);
  assert.match(aplicacion, /function aplicarCapacidadesVisibles\(estado\)[\s\S]*estado\.datos\.capacidades\[operacion\] === true/u);
  assert.match(aplicacion, /function prepararOperacion[\s\S]*estado\.datos\.capacidades\[operacion\] !== true[\s\S]*Operación no disponible/u);
  assert.match(aplicacion, /\["Objetivo", escaparHTML\(recibo\.objetivo\)\]/u);
});
