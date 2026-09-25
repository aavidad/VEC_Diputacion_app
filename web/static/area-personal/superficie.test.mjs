import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import { dirname, extname, join, relative } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { esOrigenSinteticoODesarrollo, exigirDatosOperativos, exigirParametrosConocidos } from "./aplicacion.js";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";
import { renderizarConvocatorias, renderizarDetalleConvocatoria, renderizarInicio } from "./vistas/inicio-convocatorias.js";
import { renderizarAutobaremacion, renderizarMeritos, renderizarPerfil, renderizarSolicitud } from "./vistas/perfil-meritos-solicitud.js";
import { renderizarAlegaciones, renderizarLlamamientos, renderizarSeguimiento, renderizarSubsanaciones } from "./vistas/seguimiento-tramites.js";
import { renderizarAyuda, renderizarCertificados, renderizarMensajes } from "./vistas/comunicaciones-ayuda.js";

const RAIZ = dirname(fileURLToPath(import.meta.url));

// Doble de prueba del panel del área personal: datos mínimos pero completos
// para recorrer todas las vistas con el contrato productivo.
function datosPrueba() {
  return {
    meta: { esquema: "vec.bolsa.area-personal.v1", presentacion: false, origen: "API interna autenticada", generado_en: "2026-07-18T09:00:00Z" },
    sesion: { persona_ref: "persona:prueba:0001", nombre_visible: "Persona de prueba", iniciales: "PP", metodo: "Certificado electrónico" },
    resumen: { acciones_pendientes: 2, convocatorias_abiertas: 1, solicitudes_activas: 2, mensajes_no_leidos: 1, puntuacion_provisional: 14.75 },
    capacidades: Object.fromEntries(["actualizar_contacto", "incorporar_merito", "guardar_borrador", "calcular_autobaremo", "iniciar_pago", "firmar_solicitud", "registrar_solicitud", "presentar_subsanacion", "presentar_alegacion", "marcar_mensaje", "actualizar_notificaciones", "solicitar_certificado", "solicitar_descarga"].map((clave) => [clave, true])),
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

test("el diálogo de sesión usa el catálogo común para la autoridad del servidor", async () => {
  const catalogo = JSON.parse(await readFile(join(RAIZ, "locales/es.json"), "utf8"));
  const claves = ["titulo", "persona", "referencia", "metodo", "origen", "autoridadServidor"];
  const copia = await import("./i18n.js?respaldo-dialogo-sesion");
  for (const nombre of claves) {
    const clave = `areaPersonal.sesion.${nombre}`;
    assert.equal(copia.traducir(clave), catalogo[clave], clave);
  }
  await iniciarI18nAreaPersonal({ querySelectorAll: () => [] }, async () => ({ ok: true, json: async () => catalogo }));
  for (const nombre of claves) assert.equal(traducir(`areaPersonal.sesion.${nombre}`), catalogo[`areaPersonal.sesion.${nombre}`]);
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /escaparHTML\(traducir\(`\$\{prefijoTraduccion\}autoridadServidor`\)\)/);
});

async function archivosEn(directorio) {
  const resultado = [];
  for (const entrada of await readdir(directorio, { withFileTypes: true })) {
    const ruta = join(directorio, entrada.name);
    if (entrada.isDirectory()) resultado.push(...await archivosEn(ruta));
    else resultado.push(ruta);
  }
  return resultado;
}

test("la dirección solo admite los parámetros que genera el área personal", () => {
  for (const consulta of ["presentacion=rrhh", "presentacion=", "vista=inicio&presentacion=rrhh", "perfil=tecnico", "utm_source=x"]) {
    assert.throws(() => exigirParametrosConocidos(new URLSearchParams(consulta)), /parámetros no admitidos/u, consulta);
  }
  for (const consulta of ["", "vista=llamamientos", "vista=convocatoria&id=CONV-001"]) {
    assert.doesNotThrow(() => exigirParametrosConocidos(new URLSearchParams(consulta)), consulta);
  }
});

test("la superficie cubre todos los recorridos solicitados y conserva semántica", async () => {
  const html = await readFile(join(RAIZ, "index.html"), "utf8");
  const fuentes = (await Promise.all((await archivosEn(join(RAIZ, "vistas"))).map((ruta) => readFile(ruta, "utf8")))).join("\n");
  for (const texto of [
    "Inicio y plazos", "Convocatorias", "Perfil y contacto", "Méritos y documentos",
    "Nueva solicitud", "Autobaremación", "Mis expedientes", "Mi bolsa",
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

test("ninguna vista ofrece rótulos de demostración o simulación", () => {
  const datos = datosPrueba();
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
    assert.doesNotMatch(superficie, /\bDEMO\b|Simular|demostración|class="nota demo"/iu);
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
    // B15 añade una ruta y su montaje al coordinador; el resto conserva el límite anterior.
    const tope = relative(RAIZ, ruta) === "aplicacion.js" ? 820 : 800;
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
  assert.match(aplicacion, /accion === "alternar-texto" \? document\.documentElement : document\.body/u);
});

test("el menú móvil gestiona foco, Escape y contención de teclado", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /\.ap-navegacion a\[href\]["']\)\?\.focus/);
  assert.match(aplicacion, /function mantenerFocoEnMenu\(evento\)/);
  assert.match(aplicacion, /evento\.key !== "Escape"[\s\S]{0,220}cerrarMenu\(\{ restaurarFoco: true \}\)/);
});

test("la lectura de expedientes remite a la guía textual sin sintetizar datos privados", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  const inicio = aplicacion.indexOf("function leerPantalla(estado)");
  const fin = aplicacion.indexOf("function alternarPreferencia", inicio);
  const funcion = aplicacion.slice(inicio, fin);
  assert.ok(inicio >= 0 && fin > inicio, "debe existir la lectura gobernada");
  assert.match(funcion, /conector y una política aprobados[\s\S]*guía textual de Ayuda/u);
  assert.doesNotMatch(funcion, /innerText|speechSynthesis|SpeechSynthesisUtterance/u);
  const ayuda = renderizarAyuda({ meta: { presentacion: false }, ayuda: [] });
  assert.match(ayuda, /Esta guía explica el área personal de Bolsa/u);
  assert.match(ayuda, /Cada operación real depende de la confirmación del servicio autorizado/u);
  assert.doesNotMatch(ayuda, /<audio\b|ayuda-llamamiento-bolsa\.mp3/u);
});

test("el registro final exige declaración y una referencia exacta de solicitud", () => {
  const html = renderizarSolicitud(datosPrueba(), {
    pasoSolicitud: 5,
    convocatoriaSolicitud: "CONV-001",
    solicitudEdicionId: "SOL-BORRADOR-0001",
    progresoSolicitud: {},
    errorPasoSolicitud: "",
  });
  assert.match(html, /data-operacion="registrar_solicitud" data-id="SOL-BORRADOR-0001"/u);
  assert.match(html, /name="declaracion_final" value="true" required/u);
  assert.match(html, />Registrar solicitud<\/button>/u);
});

test("la composición limita enlaces al área y bloquea capacidades antes del diálogo", async () => {
  const aplicacion = await readFile(join(RAIZ, "aplicacion.js"), "utf8");
  assert.match(aplicacion, /function actualizarEnlacesNavegacion\(estado\)[\s\S]*setAttribute\("href", crearURL\(estado/u);
  assert.doesNotMatch(aplicacion, /\/presentacion\/|presentacionSolicitada/u);
  assert.match(aplicacion, /inicioInstitucional\.dataset\.ruta = "inicio"[\s\S]*crearURL\(estado, "inicio"\)/u);
  assert.match(aplicacion, /function aplicarCapacidadesVisibles\(estado\)[\s\S]*estado\.datos\.capacidades\[operacion\] === true/u);
  assert.match(aplicacion, /function prepararOperacion[\s\S]*estado\.datos\.capacidades\[operacion\] !== true[\s\S]*Operación no disponible/u);
  assert.match(aplicacion, /\["Objetivo", escaparHTML\(recibo\.objetivo\)\]/u);
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

test("la disponibilidad B8 no se simula mientras sigue pendiente de integración", () => {
  const datos = datosPrueba();

  const htmlDisponible = renderizarLlamamientos(datos);
  assert.match(htmlDisponible, /Pausar o reactivar su disponibilidad todavía no se puede solicitar aquí\./u);
  assert.doesNotMatch(htmlDisponible, /data-operacion="cambiar_disponibilidad"|Ensayar pausa|Ensayar reactivación/u);
});

test("el correo propio y los contratos vacíos son explícitos sin inventar fecha de comunicación", () => {
  const datos = datosPrueba();
  datos.llamamientos = [];
  datos.contratos = [];
  const html = renderizarLlamamientos(datos);
  assert.match(html, /Último resultado de correo/u);
  assert.match(html, /No constan llamamientos por correo para estas participaciones\./u);
  assert.doesNotMatch(html, /B7/u);
  assert.match(html, /Contratos/u);
  assert.match(html, /Sin información de contratos\./u);
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
