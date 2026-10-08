import "./inicializar-i18n.test-helper.mjs";
import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import { dirname, extname, join, relative } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { esOrigenSinteticoODesarrollo, exigirDatosOperativos, exigirParametrosConocidos } from "./aplicacion.js";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { renderizarInicio } from "./vistas/inicio-convocatorias.js";
import { renderizarPerfil } from "./vistas/perfil-meritos-solicitud.js";
import { renderizarLlamamientos } from "./vistas/seguimiento-tramites.js";
import { renderizarAyuda } from "./vistas/comunicaciones-ayuda.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

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

// Doble de prueba del panel del área personal: datos mínimos pero completos
// para recorrer todas las vistas con el contrato productivo.
function datosPrueba() {
  return {
    meta: { esquema: "vec.bolsa.area-personal.v1", presentacion: false, origen: "API interna autenticada", generado_en: "2026-07-18T09:00:00Z" },
    sesion: { persona_ref: "persona:prueba:0001", nombre_visible: "Persona de prueba", iniciales: "PP", metodo: "Certificado electrónico" },
    resumen: { acciones_pendientes: 2, convocatorias_abiertas: 1, solicitudes_activas: 2, mensajes_no_leidos: 1, puntuacion_provisional: 14.75 },
    capacidades: Object.fromEntries(["incorporar_merito", "presentar_subsanacion", "presentar_alegacion", "marcar_mensaje", "actualizar_notificaciones", "solicitar_certificado", "solicitar_descarga"].map((clave) => [clave, true])),
    perfil: { referencia: "perfil:prueba:0001", nombre_visible: "Persona de prueba", identificador_visible: "ID-PRUEBA", correo: "persona@prueba.test", telefono: "600 000 000", domicilio: "Calle de prueba 1", estado_verificacion: "Identidad verificada", provincia: "Granada", idioma: "Castellano", canales: ["Correo", "Aviso interno"] },
    preferencias_notificacion: { correo: true, telegram: false, interno: true, convocatorias: true, plazos: true, llamamientos: true, noticias: false },
    plazos: [{ id: "PLAZO-001", dia: "23", mes: "JUL", titulo: "Responder subsanación", detalle: "Expediente SOL-0027 · 14:00", estado: "Acción requerida", ruta: "subsanaciones" }],
    convocatorias: [
      { id: "CONV-001", referencia: "BOP-GRA-2026-043004", cve_bop: "BOP-GRA-2026-043004", titulo: "Bolsa de empleo de Operario", categoria: "Operario/a", estado: "Plazo abierto", plazo: "Del 19/07/2026 al 20/08/2026", descripcion: "Bases públicas de la bolsa de Operario.", plazas: "Bolsa de empleo temporal", tasa: "Según bases", presentacion_hasta: "20/08/2026 23:59", publicada_en: "18/07/2026",
        requisitos: ["Nacionalidad conforme a las bases"], documentos: [{ titulo: "Bases (PDF)", formato: "PDF", url: "/bolsa/documentos/bases-operario.pdf", aviso: "Documento público" }, "Anexo sin enlace"] },
      { id: "CONV-002", referencia: "BOP-GRA-2024-244002", titulo: "Gestión de Administración General", categoria: "Técnico de Gestión", estado: "Histórico cerrado", plazo: "Sin plazo abierto", descripcion: "Proceso histórico.", plazas: "Véanse las bases", tasa: "Según bases", presentacion_hasta: "Cerrado", requisitos: [], documentos: [] },
    ],
    meritos: [
      { id: "MER-001", tipo: "Titulación", titulo: "Titulación de nivel 4", detalle: "Rama administrativa", estado: "Validado", documento_ref: "DOC-001", puntos_estimados: 2 },
      { id: "MER-002", tipo: "Experiencia", titulo: "Experiencia en administración pública", detalle: "18 meses", estado: "Pendiente de contraste", documento_ref: "DOC-002", puntos_estimados: 5.4 },
    ],
    documentos: [{ id: "DOC-001", nombre: "titulo.pdf", tipo: "Titulación", fecha: "10/07/2026", estado: "Verificado", huella: "SHA256-001" }],
    solicitudes: [
      { id: "SOL-0027", convocatoria_id: "CONV-001", referencia: "REG-2026-0027", titulo: "Bolsa de empleo de Operario", estado: "Registrada", actualizado: "17/07/2026 12:10", posicion: "18 de 146", puntuacion: 14.75, siguiente: "Responder subsanación", pago: "Tasa abonada", firma: "Firmada" },
      { id: "SOL-BORRADOR-0001", convocatoria_id: "CONV-001", referencia: "BORRADOR-0001", titulo: "Bolsa de empleo de Operario", estado: "Borrador", actualizado: "18/07/2026", posicion: "No aplica", puntuacion: 2, siguiente: "Registrar", pago: "Pago o exención confirmado", firma: "Firma confirmada", meritos_ids: ["MER-001"] },
    ],
    baremo: [
      { id: "BAR-001", merito_id: "MER-002", nombre: "Experiencia en la Diputación", detalle: "18 meses × 0,30 puntos", estado: "De oficio", puntos: 5.4, maximo: 8 },
      { id: "BAR-002", merito_id: "MER-001", nombre: "Titulaciones adicionales", detalle: "Una titulación", estado: "Validado", puntos: 2, maximo: 3 },
      { id: "BAR-003", nombre: "Ejercicio superado", detalle: "Resultado del proceso", estado: "De oficio", puntos: 4.95, maximo: 5, de_oficio: true },
    ],
    posicion: { bolsa: "Bolsa de empleo de Operario", categoria: "Operario/a", orden: 18, total: 146, puntuacion: 14.75, vigente_desde: "2026-07-01T08:00:00Z" },
    disponibilidad: { disponible: true, estado_clave: "disponible", estado: "Disponible para llamamientos", estado_desde: "2026-07-01T08:00:00Z", disponible_desde: null, motivo_visible: null, desde: "01/07/2026", bolsas: ["Bolsa de empleo de Operario"] },
    llamamientos: [{ id: "LLA-0045", bolsa: "Bolsa de empleo de Operario", puesto: "Operario/a", plazo: "Pendiente de confirmación por RRHH", estado: "Pendiente de respuesta", jornada: "Completa", duracion: "Seis meses", posicion: "Primera persona elegible", canal: "Correo", comunicado_en: null }],
    contratos: [],
    subsanaciones: [{ id: "SUB-0008", solicitud_ref: "SOL-0027", motivo: "Acreditar la jornada", plazo: "23/07/2026 14:00", estado: "Pendiente", documento_solicitado: "Certificado de jornada" }],
    alegaciones: [{ id: "ALE-0003", solicitud_ref: "SOL-0027", asunto: "Revisión de formación", estado: "Borrador", fecha: "Creada 17/07/2026" }],
    mensajes: [{ id: "MSG-001", asunto: "Subsanación disponible", resumen: "Revise el requerimiento.", fecha: "17/07/2026 12:10", estado: "No leído", tipo: "Acción requerida", ruta: "subsanaciones" }],
    certificados: [{ id: "CER-001", tipo: "Certificado de inscripción en bolsa", descripcion: "Acredita la situación en una bolsa.", estado: "Disponible bajo solicitud", formatos: "PDF, ODT o JSON" }],
    actividad: [{ id: "ACT-001", titulo: "Solicitud registrada", detalle: "Registro de la solicitud.", fecha: "12/07/2026 18:42", actor: "Persona de prueba", recibo: "REG-2026-0027" }],
    ayuda: [{ pregunta: "¿Cómo me inscribo en una convocatoria?", respuesta: "Abra Convocatorias y seleccione Iniciar solicitud." }],
  };
}

test("el diálogo de sesión no muestra rutas técnicas", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  const inicio = aplicacion.indexOf("function verSesion(estado)");
  const fin = aplicacion.indexOf("function leerPantalla", inicio);
  const funcion = aplicacion.slice(inicio, fin);
  assert.doesNotMatch(funcion, /meta\.origen|autoridadServidor|\/api\//u);
  assert.match(funcion, /sesion\.metodo/u);
});

test("la dirección admite el identificador de enlaces antiguos para poder redirigirlos", () => {
  for (const consulta of ["presentacion=rrhh", "perfil=tecnico", "utm_source=x"]) {
    assert.throws(() => exigirParametrosConocidos(new URLSearchParams(consulta)), /parámetros no admitidos/u, consulta);
  }
  for (const consulta of ["", "vista=llamamientos", "vista=ayuda&lang=en", "vista=convocatoria&id=CONV-1"]) {
    assert.doesNotThrow(() => exigirParametrosConocidos(new URLSearchParams(consulta)), consulta);
  }
});

test("el menú muestra solo las vistas con recorrido actual", async () => {
  const html = await readFile(join(RAIZ, "index.html"), "utf8");
  const vistas = JSON.parse(await readFile(join(RAIZ, "vistas.json"), "utf8"));
  assert.deepEqual(Object.keys(vistas.vistas).sort(), ["ayuda", "inicio", "llamamientos", "oportunidades", "perfil", "preferencias"]);
  for (const nombre of ["inicio", "llamamientos", "perfil", "ayuda"]) assert.match(html, new RegExp(`data-ruta="${nombre}"`, "u"));
  assert.doesNotMatch(html, /data-ruta="(?:convocatorias|convocatoria|meritos|seguimiento|subsanaciones|alegaciones|mensajes|certificados|solicitud)"/u);
  assert.match(html, /<main\b/u);
  assert.match(html, /name="viewport" content="width=device-width, initial-scale=1"/u);
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

test("el arranque de producto solo compone HTTP y no importa adaptadores de presentación", async () => {
  const arranque = await readFile(join(RAIZ, "arranque.js"), "utf8");
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(arranque, /exigirParametrosConocidos\(new URLSearchParams\(window\.location\.search\)\)/u);
  assert.doesNotMatch(arranque, /adaptador-presentacion|descarga-recibos-presentacion|selector-perfiles/u);
  assert.match(arranque, /crearClienteHTTPAreaPersonal/u);
  assert.doesNotMatch(aplicacion, /adaptador-presentacion|cliente-http/u);
  assert.doesNotMatch(arranque, /innerHTML\s*=\s*`[^`]*error\.message/su);
  assert.match(arranque, /detalle\.textContent\s*=\s*traducir\("areaPersonal\.estado\.error\.detalle"\)/u);
  assert.doesNotMatch(arranque, /error\.message/u);
  assert.doesNotMatch(await readFile(join(RAIZ, "index.html"), "utf8"), /id="aviso-presentacion"|Confirmar demostración/u);
  assert.doesNotMatch(await readFile(join(RAIZ, "index.html"), "utf8"), /selector-perfiles\.css/u);
});

test("el área rechaza datos sintéticos, de desarrollo o de presentación", () => {
  assert.equal(esOrigenSinteticoODesarrollo({ origen: "Dataset sintético" }), true);
  assert.equal(esOrigenSinteticoODesarrollo({ origen: "Perfil de desarrollo sin identidad de candidato" }), true);
  assert.equal(esOrigenSinteticoODesarrollo({ origen: "API interna", entorno: "desarrollo" }), true);
  assert.equal(esOrigenSinteticoODesarrollo({ origen: "API interna autenticada" }), false);
  assert.equal(esOrigenSinteticoODesarrollo(null), false);
  assert.throws(() => exigirDatosOperativos({ meta: { presentacion: true, origen: "API interna" } }), /no está configurada/u);
  assert.throws(() => exigirDatosOperativos({ meta: { presentacion: false, origen: "Dataset sintético" } }), /no está configurada/u);
  assert.doesNotThrow(() => exigirDatosOperativos({ meta: { presentacion: false, origen: "API interna autenticada" } }));
});

test("las vistas activas no presentan datos de trámites antiguos", () => {
  const datos = datosPrueba();
  const superficies = [renderizarInicio(), renderizarPerfil(), renderizarLlamamientos(datos), renderizarAyuda()];
  for (const vista of superficies) {
    assert.doesNotMatch(vista, /\bDEMO\b|Simular|demostración|data-operacion="(?:incorporar_merito|presentar_subsanacion|presentar_alegacion)"/iu);
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
    // 5.08a añade preferencias y devuelve el foco tras consultas y guardados;
    // su formulario permanece en un módulo aparte. 5.08b monta «Mis correos»
    // (componente común) con una línea. Aspirantes monta la ficha propia con
    // tres (importar, desmontar y montar).
    const tope = relative(RAIZ, ruta) === "aplicacion.js" ? 964 : 800;
    assert.ok(lineas < tope, `${relative(RAIZ, ruta)} tiene ${lineas} líneas`);
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
  assert.match(aplicacion, /controladorVisual\?\.aplicarPreferenciasServidor/u);
});

test("el menú móvil gestiona foco, Escape y contención de teclado", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /\.ap-navegacion a\[href\]:not\(\[hidden\]\)["']\)\?\.focus/);
  assert.match(aplicacion, /function mantenerFocoEnMenu\(evento\)/);
  assert.match(aplicacion, /evento\.key !== "Escape"[\s\S]*cerrarMenuIdentidad\(\{ restaurarFoco: true \}\)/);
  assert.match(aplicacion, /evento\.key !== "Escape"[\s\S]*cerrarMenu\(\{ restaurarFoco: true \}\)/);
});

test("la lectura remite a la guía sin sintetizar datos privados", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  const inicio = aplicacion.indexOf("function leerPantalla(estado)");
  const fin = aplicacion.indexOf("async function recargarPreferencias", inicio);
  const funcion = aplicacion.slice(inicio, fin);
  assert.ok(inicio >= 0 && fin > inicio);
  assert.doesNotMatch(funcion, /innerText|SpeechSynthesisUtterance/u);
  assert.match(renderizarAyuda(), /guia-ayuda/u);
});

test("la composición limita enlaces y no ofrece operaciones sin servicio", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /function actualizarEnlacesNavegacion\(estado\)[\s\S]*setAttribute\("href", crearURL\(estado/u);
  assert.doesNotMatch(aplicacion, /abrir-expediente|iniciar-solicitud|presentar_subsanacion|incorporar_merito|dialogo-confirmacion|dialogo-recibo/u);
});

test("la ficha propia muestra la participación sin convertirla en una decisión de RRHH", () => {
  const datos = datosPrueba();
  const html = renderizarLlamamientos(datos);

  assert.match(html, /<h3>Mi participación<\/h3>/u);
  assert.match(html, /18 de 146/u);
  assert.match(html, /Operario\/a/u);
  assert.match(html, /146/u);

  // Sin posición ni participación se informa de la ausencia de datos.
  const datosSinPosicion = structuredClone(datos);
  delete datosSinPosicion.posicion;
  const htmlSin = renderizarLlamamientos(datosSinPosicion);
  assert.match(htmlSin, /Sin participaciones activas/u);
});

test("la disponibilidad no admite acciones de pausa o reactivación", () => {
  const datos = datosPrueba();

  const htmlDisponible = renderizarLlamamientos(datos);
  assert.match(htmlDisponible, /Para solicitar una revisión, diríjase al Servicio de RRHH\./u);
  assert.doesNotMatch(htmlDisponible, /data-operacion="cambiar_disponibilidad"|Ensayar pausa|Ensayar reactivación/u);
});

test("el correo propio y el histórico separado esperan una consulta autorizada", () => {
  const datos = datosPrueba();
  datos.llamamientos = [];
  datos.contratos = [];
  const html = renderizarLlamamientos(datos);
  assert.match(html, /Último resultado de correo/u);
  assert.match(html, /No constan llamamientos por correo para estas participaciones\./u);
  assert.doesNotMatch(html, /B7/u);
  assert.match(html, /Histórico de mi bolsa[\s\S]*Cargando histórico autorizado/u);
  assert.doesNotMatch(html, /Sin información de contratos\./u);
  const conLlamamientos = renderizarLlamamientos(datosPrueba());
  assert.doesNotMatch(conLlamamientos, /Comunicado el|Aceptar llamamiento/u);
});

test("el área personal no conserva código ni rótulos de demostración", async () => {
  const produccion = (await archivosEn(RAIZ)).filter((ruta) => !ruta.endsWith(".test.mjs"));
  assert.ok(!produccion.some((ruta) => relative(RAIZ, ruta) === "adaptador-presentacion.js"));
  for (const ruta of produccion) {
    const contenido = await readFile(ruta, "utf8");
    assert.doesNotMatch(contenido, /DEMO-|recibo-demo|recorrido_demo|limiteSintetico|\bSimular\b/u, relative(RAIZ, ruta));
  }
});
