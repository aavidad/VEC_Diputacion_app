import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFile } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";
import test from "node:test";
import { cargarCatalogosContratacionEnIdioma } from "./i18n-catalogos.js?v=VEC_WEB_VERSION";
import { cargarTextos } from "../../../comun/textos.js";
import { IDIOMA_ACTUAL, IDIOMAS_DISPONIBLES } from "../../../comun/idioma.js";
import { crearTraductorCancelacion } from "./i18n-cancelacion.js?v=VEC_WEB_VERSION";
import { cargarMensajesTramitePortalEnIdioma, mensajesTramite, rotuloTramite } from "./i18n-fases-rrhh.js?v=VEC_WEB_VERSION";
import { crearTraductorContratacionTemporal, cargarMensajesContratacionTemporalEnIdioma } from "./i18n.js?v=VEC_WEB_VERSION";
import { crearTraductorExpedientesContratacion, cargarMensajesExpedientesContratacionEnIdioma } from "./i18n-expedientes.js?v=VEC_WEB_VERSION";

// Huellas de las exportaciones originales en 463f7c176, anteriores al traslado.
// Incluyen nombres, orden de claves, textos completos y marcadores sin duplicar los textos.
const PREIMAGEN = {
  "i18n-analisis-catalogo.js": {
    "MENSAJES_ANALISIS_CATALOGO_EN": "f800e191693009fbda79315380b47ee28e1c5b5d653552845cd0a88d39d22f3b",
    "MENSAJES_ANALISIS_CATALOGO_ES": "7ebc3646ea6e5b35e13f785b67726b16d2957740c91a899382dd7f05d6ff0aa8"
  },
  "i18n-avisos-via-cobertura.js": {
    "MENSAJES_AVISOS_VIA_COBERTURA_EN": "210732a4a5da432fa0c5d6eb0e544331d9b38ee1c23ad0d4783be0d07bd2387a",
    "MENSAJES_AVISOS_VIA_COBERTURA_ES": "d2c54f54d09f6b274eb268bc29c58f497f7c9cbdb8c4bb0662c8e0e0812ea2e4"
  },
  "i18n-borradores-publicados.js": {
    // Excepción a la preimagen: corrección EN de bp_subtitulo posterior al traslado.
    // F1-88: bp_publicacion_recibo pasa a ser el rótulo del justificante, sin {recibo}.
    "MENSAJES_BORRADORES_PUBLICADOS_EN": "518a57a6a28aed46668e6c47eb12bd4af93c79300a7c4b9119b48b4e9c62effc",
    "MENSAJES_BORRADORES_PUBLICADOS_ES": "25f03d40911fc90b70d485958d2d73edd01f326a77b82e5c4a3ebc87d75d7086"
  },
  "i18n-cambios-expediente.js": {
    "MENSAJES_CAMBIOS_EXPEDIENTE_EN": "9a37c4630d4ed716d2b5176229ab039b5cfd4731828050ed3c2852dfbd5dd8a4",
    "MENSAJES_CAMBIOS_EXPEDIENTE_ES": "ad0866209f3abf073363c0a63082cf218624b357d61130df32731edceff51d5e"
  },
  "i18n-cancelacion.js": {
    "MENSAJES_CANCELACION": "54acc78d485c6ca3ecce0392fb058c881d045453fac7964601c46a51ed831127",
    "MENSAJES_CANCELACION_EN": "46eb4ef0918d3f9215e0224967c2fbdfd246c1ec7ff72aa02870e0c5a708ed18"
  },
  "i18n-fases-rrhh.js": {
    "FASES_RRHH": "3bda2743d6eebf9dae0a6b8848690e07ee361bae241fbbcb416348c96ccc677b",
    "FASE_RRHH_DE_ORIGEN": "0b2e97678a821e146b462eede4ec804d1ddc647fef11a40b0e5c8dfde3a4e91e",
    "rotulos_es": "602e395da8cb8498213ab5a124178d5a3326f9b1c9dcbafeaeb62f030cecdc92",
    "rotulos_en": "4b385dd66a73578fc25568f6b49ab63b40a899577dd202c8b03d652270cca20b"
  },
  "i18n-ficha-lista.js": {
    // F1-74 aporta los mensajes de fiscalización y C el error de firmas históricas.
    "MENSAJES_FICHA_LISTA_EN": "b1b986b0393d8d4843d517b79b425536fb83c39bdcb147e29a502719bc047887",
    "MENSAJES_FICHA_LISTA_ES": "ddf03fbe54a3b91147d7536fd213ec24867692da62b30e2503e9b2c5da0d4009"
  },
  "i18n-informe-tras-subsanacion.js": {
    // Excepción a la preimagen: F1-26 nombra en inglés el apartado «Signing workflow».
    "MENSAJES_INFORME_TRAS_SUBSANACION_EN": "457ff6a48d44b1554c8275e776a9231f098a0d0f0baad802799eaf413f526330",
    "MENSAJES_INFORME_TRAS_SUBSANACION_ES": "d75febfd37a4cf2265d502bcf2ae5af4ccad05b724433807974fea104abef743"
  },
  "i18n-llamamiento.js": {
    // Excepción a la preimagen: textos reescritos en lenguaje llano (05/10/2026), sin
    // clave de operación ni modo manual en pantalla; los límites pasan a la ayuda «?».
    // 12/10/2026: sin avisos de no validez en la validación manual.
    "MENSAJES_LLAMAMIENTO_EN": "bc4a75a142e9fe7c02ea378f9721aa184f2b6c9a92c87671f1c16c135aad1242",
    "MENSAJES_LLAMAMIENTO_ES": "0382ad2891ac444918f8d32c49d8e5d0b2e9c326ffaae6a8416dec5f76bdaffb"
  },
  "i18n-subsanacion-reparos.js": {
    "MENSAJES_SUBSANACION_REPAROS_EN": "d3dc09b6d8b28dfbfd56e64d54e6d40b9f3dd988562a67c28e1ccde882f6a12d",
    "MENSAJES_SUBSANACION_REPAROS_ES": "e8d65c1e4b79ea981ed7cdc39b3444536749a1e192cfa128a68c2cc824cdc8fb"
  },
  "i18n-textos-vistas.js": {
    "MENSAJES_TEXTOS_VISTAS_EN": "3b62dd78417e9910cb3d4bdb6f0cc311d068cf6cef06f01dddb945d14febcd9b",
    "MENSAJES_TEXTOS_VISTAS_ES": "837edbba7b1bbd2e492d3841a1dc65313f37f92e819681785bc819ed137f3ae8"
  }
};
const PREIMAGEN_FICHA_LISTA_ANTERIOR = Object.freeze({
  MENSAJES_FICHA_LISTA_EN: "9150a2587a75e61eaf689150c1ed5a7eb8b154f50da041431d44fb2eb446b06a",
  MENSAJES_FICHA_LISTA_ES: "f104ba579849f09ba63661d23f925889abe7b2f4e8831b060a37527f01c59558",
});
// ANA002 añade cinco rótulos de contexto; la preimagen sigue comprobando todos los textos anteriores.
const CLAVES_CONTEXTO_ANA002 = Object.freeze([
  "ct_txt_contexto_fechas_consulta",
  "ct_txt_contexto_agrupacion",
  "ct_txt_contexto_zona_horaria",
  "ct_txt_contexto_corte_publicado",
  "ct_txt_contexto_no_comunicado",
]);
const CLAVES_DOCUMENTOS_FICHA = Object.freeze([
  "ficha_borrador_sin_firmar", "ficha_descargar_pdf", "ficha_descargar_word",
  "ficha_documentos_cargando", "ficha_documentos_error", "ficha_documentos_no_montados",
  "ficha_borradores_no_montados", "ficha_borradores_cargando", "ficha_borradores_error",
  "ficha_documentos_recuento_indice",
]);
const CLAVES_BORRADORES_FICHA = Object.freeze([
  "bp_ficha_titulo", "bp_estado_borrador", "bp_ficha_docx",
  "bp_estado_pendiente", "bp_no_disponible",
]);
const CLAVES_CUADRO_LIGERO = Object.freeze([
  "lista_centro_nombre_no_disponible", "lista_categoria_nombre_no_disponible",
  "filtros", "tabla_expedientes", "marca_urgente", "lista_actualizar", "lista_vacia_sin_alta",
  "lista_textos_respaldo",
  "lista_filtro_no_disponible", "lista_resultados_pagina",
  "lista_titulo_conjunto", "lista_buscar_pista_servidor",
  "lista_fase_servidor_preparacion", "lista_fase_servidor_llamamiento", "lista_fase_servidor_cierre",
  "lista_centro_nombre_no_disponible", "lista_categoria_nombre_no_disponible",
  "plazo_fase_en_plazo", "plazo_fase_vence_hoy",
  "plazo_fase_vencido", "plazo_fase_sin_calcular",
]);

const CLAVES_FIN_MODALIDAD = Object.freeze({
  "i18n-analisis-catalogo.js": [
    "causa_fin_reincorporacion_titular", "causa_fin_cobertura_reglamentaria", "pc_periodo_con_causa",
    "error_fecha_no_aplica", "error_causa_fin", "analisis_error_fecha_no_aplica", "analisis_error_causa_fin",
  ],
  "i18n-avisos-via-cobertura.js": ["avisos_via_propuesta_oferta_sae_sin_fin"],
});
const CLAVES_REINCORPORACION_CAPACIDAD = Object.freeze([
  "reincorporacion_capacidad_denegada", "reincorporacion_capacidad_no_disponible",
  "reincorporacion_capacidad_reintentar", "reincorporacion_capacidad_comprobando",
  "reincorporacion_capacidad_no_habilitada",
]);
const CLAVES_FICHA_SIN_BOLSA = Object.freeze([
  "ficha_llamamiento_sin_bolsa", "ficha_llamamiento_sin_bolsa_historica", "ficha_llamamiento_bolsa_error",
  "ficha_llamamiento_bolsa_denegado", "ficha_llamamiento_bolsa_reintentar",
  "ficha_llamamiento_bolsa_no_localizada",
]);
const CLAVES_GUIA_COBERTURA = Object.freeze([
  "ficha_siguiente_paso_cobertura_titulo", "ficha_siguiente_paso_cobertura_que",
  "ficha_siguiente_paso_fiscalizacion_intervencion",
]);
// La respuesta real de Bolsa amplía la ficha; la huella de las claves anteriores sigue fija.
const CLAVES_RESULTADO_BOLSA_FICHA = Object.freeze([
  "resultado_bolsa_titulo", "resultado_bolsa_llamamiento", "resultado_bolsa_seguimiento",
  "resultado_bolsa_respuesta", "resultado_bolsa_acepta", "resultado_bolsa_renuncia",
  "resultado_bolsa_renuncia_justificada", "resultado_bolsa_propuesta", "resultado_bolsa_sin_respuesta",
  "resultado_bolsa_fecha", "resultado_bolsa_justificante", "resultado_bolsa_situacion",
  "resultado_bolsa_situacion_renuncia", "resultado_bolsa_fecha_situacion",
  "resultado_bolsa_justificante_situacion", "resultado_bolsa_elegir", "resultado_bolsa_seleccionar",
  "resultado_bolsa_vincular", "resultado_bolsa_mas", "resultado_bolsa_sin_llamamientos",
  "resultado_bolsa_confirmar", "resultado_bolsa_guardando", "resultado_bolsa_guardado",
  "resultado_bolsa_error", "resultado_bolsa_denegado", "resultado_bolsa_conflicto",
  "resultado_bolsa_reintentar", "resultado_bolsa_actualizacion", "resultado_bolsa_continuar",
  "resultado_bolsa_anterior", "resultado_bolsa_error_lectura", "resultado_bolsa_revisar",
  // F1-100: el selector y la confirmación dicen a quién se llamó.
  "resultado_bolsa_confirmar_persona", "resultado_bolsa_personas_varias",
  "resultado_bolsa_contacto", "resultado_bolsa_fecha_contacto", "resultado_bolsa_justificante_contacto", "resultado_bolsa_contacto_contactado", "resultado_bolsa_contacto_comunica", "resultado_bolsa_contacto_no_contesta", "resultado_bolsa_contacto_buzon", "resultado_bolsa_contacto_acepta", "resultado_bolsa_contacto_rechaza", "resultado_bolsa_contacto_aplazado", "resultado_bolsa_contacto_otro", "resultado_bolsa_contacto_enviado", "resultado_bolsa_contacto_no_enviado", "resultado_bolsa_contacto_numero_erroneo", "resultado_bolsa_contacto_no_entregado", "resultado_bolsa_contacto_entrega_declarada",
]);
// Las claves añadidas se comprueban aparte sin reescribir la preimagen anterior.
const CLAVES_ADICIONALES = Object.freeze({
  "i18n-cancelacion.js": ["error_seleccion_oferta_pendiente",
    // F1-102: motivos en el idioma elegido y el expediente por su número en el diálogo.
    "confirmar_referencia", "motivo_necesidad_desaparecida", "motivo_desistimiento_centro",
    "motivo_cobertura_otra_via", "motivo_error_solicitud", "motivo_solicitud_duplicada", "motivo_falta_credito"],
  // F1-96: nombres de los documentos generados en el idioma elegido.
  "i18n-borradores-publicados.js": ["bp_documento_informe_definitivo", "bp_documento_resolucion",
    "bp_documento_diligencia", "bp_documento_toma_posesion", "bp_documento_notificacion",
    "bp_documento_comunicacion_centro", "bp_documento_contrato_laboral", "bp_documento_nombramiento",
    "bp_documento_cese", "bp_documento_modificacion_nombramiento",
    // F1-88: la huella del documento queda tras «Ver detalle técnico».
    "bp_detalle_tecnico"],
  // R1 añade el aviso de comprobación pendiente; CT231, las del aviso pendiente
  // tras la selección confirmada; F1-16, las de las ofertas de varias plazas.
  "i18n-llamamiento.js": [
    "llamamiento_estado_oferta_cargando", "llamamiento_estado_oferta_error",
    "llamamiento_estado_oferta_denegado", "llamamiento_estado_oferta_reintentar",
    "llamamiento_cantidad_no_disponible",
    "llamamiento_orden_bolsa_provisional", "llamamiento_iniciar_seleccion",
    "llamamiento_publicar_oferta", "llamamiento_confirmacion_seleccion_neutra",
    "llamamiento_confirmacion_oferta", "llamamiento_oferta_recibo",
    "llamamiento_oferta_plazas", "llamamiento_oferta_publicada_en",
    "llamamiento_oferta_vence_antes_de", "llamamiento_oferta_justificante",
    "llamamiento_oferta_vinculo", "llamamiento_oferta_sin_aviso",
    "llamamiento_oferta_gestionar", "llamamiento_recuperar_oferta",
    "llamamiento_oferta_resultado_incierto", "llamamiento_confirmacion_recuperar_oferta",
    "llamamiento_recuperando_oferta", "llamamiento_recuperacion_oferta_incierta",
    "llamamiento_oferta_sin_reserva", "llamamiento_oferta_descartada",
    "llamamiento_oferta_descartada_en", "llamamiento_oferta_cierre_bolsa",
    "llamamiento_oferta_recibo_descarte", "llamamiento_oferta_descartada_sin_publicar",
    "llamamiento_preparar_nueva_oferta", "llamamiento_confirmacion_nueva_oferta",
    "llamamiento_comprobacion_pendiente", "llamamiento_comprobacion_version",
    "llamamiento_comprobacion_ver_bolsa", "llamamiento_comprobacion_comprobar",
    "llamamiento_comunicaciones_respuesta_aviso_pendiente", "llamamiento_comunicaciones_respuesta_aviso_anotado",
    "llamamiento_seleccion_pendiente_aviso",
    "llamamiento_continuar_aviso",
  ],
  // F1-93: textos de recibos e historial de la ficha, trasladados desde el código en lenguaje llano.
  "i18n-textos-vistas.js": ["motivo_sustitucion", "cobertura_recibo_descripcion",
    "asignacion_recibo_descripcion", "informe_recibo_descripcion", "informe_historial_titulo",
    "informe_historial_descripcion", "informe_historial_tabla",
    // F1-96: la confirmación del informe nombra el expediente por su número.
    "informe_resumen_numero",
    // F1-88: confirmaciones sin referencia interna (traslado de informe_resumen desde i18n.js)
    // y rótulos claros de cobertura y asignación.
    "informe_resumen", "cobertura_motivo_alternativa", "cobertura_estado_motivo_obligatorio",
    "asignacion_confirmacion", "asignacion_confirmacion_advertencia"],
  "i18n-ficha-lista.js": ["alta_solo_sustituciones", "necesidades_alta_no_disponibles_titulo",
    "necesidades_alta_no_disponibles_detalle",
    // F1-89: orientación de la ficha tras preparar el informe jurídico.
    "ficha_siguiente_paso_informe_titulo", "ficha_siguiente_paso_informe_que",
    "ficha_siguiente_paso_informe_quien", "ficha_siguiente_paso_intervencion_quien",
    // F1-93: la cabecera nombra el paso dentro de la fase.
    "ficha_fase_con_subfase",
    // F1-96: el paso propio de la ficha tras asignar la unidad.
    "ficha_siguiente_paso_preparar_informe_titulo", "ficha_siguiente_paso_preparar_informe_que",
    // F1-102: la ficha cancelada lo dice y el historial nombra la propuesta y la resolución.
    "ficha_siguiente_paso_cancelado", "ficha_siguiente_paso_cancelado_que",
    "hito_propuesta_formalizacion", "hito_resolucion_formalizacion"],
});
const huella = (valor) => createHash("sha256").update(JSON.stringify(valor)).digest("hex");
const codigos = (await cargarTextos("contratacion-temporal-compatibilidad")).seccion("idiomas_exportados");

async function preimagenEnIdioma(archivo, nombre) {
  const codigo = nombre.endsWith("_EN") || nombre === "rotulos_en" ? codigos.EN : codigos.ES;
  if (nombre.startsWith("rotulos_")) {
    const { actual } = await cargarCatalogosContratacionEnIdioma("portal", codigo, "fases_rrhh");
    return actual;
  }
  const base = archivo.replace(/^i18n-/u, "contratacion-temporal-").replace(/\.js$/u, "");
  if (archivo === "i18n-ficha-lista.js") {
    const [ficha, plazos] = await Promise.all([
      cargarCatalogosContratacionEnIdioma(base, codigo),
      cargarCatalogosContratacionEnIdioma("contratacion-temporal-lista-plazos", codigo, "lista"),
    ]);
    return Object.freeze({ ...plazos.actual, ...ficha.actual });
  }
  return (await cargarCatalogosContratacionEnIdioma(base, codigo)).actual;
}

for (const [archivo, exportaciones] of Object.entries(PREIMAGEN)) {
  test(`${archivo}: conserva la preimagen de las exportaciones y los rótulos`, async () => {
    const modulo = await import(new URL(archivo, import.meta.url));
    for (const [nombre, anterior] of Object.entries(exportaciones)) {
      const valor = nombre === "FASES_RRHH" || nombre === "FASE_RRHH_DE_ORIGEN"
        ? modulo[nombre] : await preimagenEnIdioma(archivo, nombre);
      let preimagen = valor;
	  if (CLAVES_FIN_MODALIDAD[archivo]) {
	    for (const clave of CLAVES_FIN_MODALIDAD[archivo]) {
	      assert.ok(typeof valor[clave] === "string" && valor[clave].trim(), `${nombre}.${clave}`);
	    }
	    preimagen = Object.fromEntries(Object.entries(valor)
	      .filter(([clave]) => !CLAVES_FIN_MODALIDAD[archivo].includes(clave)));
	  }
      if (archivo === "i18n-textos-vistas.js") {
        for (const clave of CLAVES_CONTEXTO_ANA002) {
          assert.ok(Object.hasOwn(valor, clave), `${nombre}.${clave}`);
          assert.equal(typeof valor[clave], "string", `${nombre}.${clave}`);
          assert.ok(valor[clave].trim(), `${nombre}.${clave}`);
        }
        preimagen = Object.fromEntries(Object.entries(valor)
          .filter(([clave]) => !CLAVES_CONTEXTO_ANA002.includes(clave)));
      }
      if (archivo === "i18n-ficha-lista.js") {
        for (const clave of [...CLAVES_CUADRO_LIGERO, ...CLAVES_REINCORPORACION_CAPACIDAD,
          ...CLAVES_DOCUMENTOS_FICHA, ...CLAVES_FICHA_SIN_BOLSA, ...CLAVES_GUIA_COBERTURA,
          ...CLAVES_RESULTADO_BOLSA_FICHA]) {
          assert.ok(typeof valor[clave] === "string" && valor[clave].trim(), `${nombre}.${clave}`);
        }
        preimagen = Object.fromEntries(Object.entries(valor)
          .filter(([clave]) => !CLAVES_CUADRO_LIGERO.includes(clave) && !CLAVES_REINCORPORACION_CAPACIDAD.includes(clave)
            && !CLAVES_DOCUMENTOS_FICHA.includes(clave) && !CLAVES_FICHA_SIN_BOLSA.includes(clave)
            && !CLAVES_GUIA_COBERTURA.includes(clave) && !CLAVES_RESULTADO_BOLSA_FICHA.includes(clave)));

      }
      if (archivo === "i18n-borradores-publicados.js") {
        for (const clave of CLAVES_BORRADORES_FICHA) {
          assert.ok(typeof valor[clave] === "string" && valor[clave].trim(), `${nombre}.${clave}`);
        }
        preimagen = Object.fromEntries(Object.entries(valor)
          .filter(([clave]) => !CLAVES_BORRADORES_FICHA.includes(clave)));
      }
      if (CLAVES_ADICIONALES[archivo]) {
        for (const clave of CLAVES_ADICIONALES[archivo]) {
          assert.ok(typeof valor[clave] === "string" && valor[clave].trim(), `${nombre}.${clave}`);
        }
        preimagen = Object.fromEntries(Object.entries(preimagen)
          .filter(([clave]) => !CLAVES_ADICIONALES[archivo].includes(clave)));
      }
      if (archivo === "i18n-ficha-lista.js") {
        assert.ok(typeof valor.firmas_historicas_error === "string" && valor.firmas_historicas_error.trim(), nombre);
        const anteriorSinFirmas = Object.fromEntries(Object.entries(preimagen)
          .filter(([clave]) => clave !== "firmas_historicas_error"));
        assert.equal(huella(anteriorSinFirmas), PREIMAGEN_FICHA_LISTA_ANTERIOR[nombre], nombre);
      }
      assert.equal(huella(preimagen), anterior, nombre);
      assert.ok(Object.isFrozen(valor), nombre);
    }
  });
}

test("los catálogos y los traductores conservan marcadores, sobrescrituras y claves desconocidas", async () => {
  for (const codigo of Object.values(codigos)) {
    for (const [mensajes, crear] of [
      [await cargarMensajesContratacionTemporalEnIdioma(codigo), crearTraductorContratacionTemporal],
      [await cargarMensajesExpedientesContratacionEnIdioma(codigo), crearTraductorExpedientesContratacion],
    ]) {
      const traducir = crear(mensajes);
      for (const [clave, original] of Object.entries(mensajes)) {
        const nombres = [...original.matchAll(/\{([A-Za-z_][A-Za-z0-9_]*)\}/gu)].map(([, variable]) => variable);
        const variables = Object.fromEntries(nombres.map((variable) => [variable, `valor-${variable}`]));
        const esperado = Object.entries(variables).reduce((texto, [variable, valor]) => texto.replaceAll(`{${variable}}`, valor), original);
        assert.equal(traducir(clave, variables), esperado, `${codigo}.${clave}`);
        assert.equal(traducir(clave), original, `marcadores sin valor: ${codigo}.${clave}`);
      }
      assert.throws(() => traducir("clave_inexistente"));
      assert.throws(() => crear({ titulo: "" }));
      assert.throws(() => crear(null));
    }
    const fases = (await cargarCatalogosContratacionEnIdioma("portal", codigo, "fases_rrhh")).actual;
    const mensajesPortal = await cargarMensajesTramitePortalEnIdioma(codigo);
    for (const [clave, texto] of Object.entries(fases)) assert.equal(mensajesPortal[`tramite_${clave}`], texto);
    if (codigo === IDIOMA_ACTUAL) {
      for (const [clave, texto] of Object.entries(fases)) assert.equal(rotuloTramite(clave, {}, codigo), texto);
      const tramite = mensajesTramite(codigo);
      assert.equal(tramite.fase_rrhh_orden_nombre, fases.fase_de_nombre);
      assert.equal(tramite.etiqueta_fase_incorporacion, fases.fase_incorporacion);
    } else {
      assert.throws(() => rotuloTramite("fase_incorporacion", {}, codigo), /idioma solicitado/u);
      assert.throws(() => mensajesTramite(codigo), /idioma solicitado/u);
    }
  }
  const traducir = crearTraductorCancelacion({ "cancelacion.titulo": "{uno} {dos}" });
  assert.equal(traducir("titulo", { uno: 0 }), "0 {dos}");
  assert.equal(traducir("clave_inexistente"), "clave_inexistente");
});

test("las exportaciones no cambian al reordenar el índice ni al cambiar el idioma por defecto", async () => {
  const temporal = await mkdtemp(join(tmpdir(), "vec-ct-i18n-"));
  try {
    const fuentes = [
      "comun/idioma.js", "comun/textos.js", "textos/idiomas.json",
      "portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js",
      "portal-empleado/modulos/contratacion-temporal/i18n-analisis-catalogo.js",
    ];
    for (const { codigo } of IDIOMAS_DISPONIBLES) {
      fuentes.push(`textos/${codigo}/contratacion-temporal-compatibilidad.json`, `textos/${codigo}/contratacion-temporal-analisis-catalogo.json`);
    }
    const raiz = new URL("../../../", import.meta.url);
    for (const archivo of fuentes) {
      const destino = join(temporal, archivo);
      await mkdir(dirname(destino), { recursive: true });
      await cp(new URL(archivo, raiz), destino);
    }
    await writeFile(join(temporal, "package.json"), JSON.stringify({ type: "module" }));
    const indice = JSON.parse(await readFile(join(temporal, "textos/idiomas.json"), "utf8"));
    indice.idiomas.reverse();
    indice.por_defecto = codigos.EN;
    await writeFile(join(temporal, "textos/idiomas.json"), JSON.stringify(indice));
    const modulo = await import(pathToFileURL(join(temporal, "portal-empleado/modulos/contratacion-temporal/i18n-analisis-catalogo.js")));
    const esperado = (await cargarCatalogosContratacionEnIdioma("contratacion-temporal-analisis-catalogo", codigos.EN)).actual;
    assert.equal(modulo.MENSAJES_ANALISIS_CATALOGO_ES, undefined);
    assert.deepEqual(modulo.MENSAJES_ANALISIS_CATALOGO_EN, esperado);
    const helper = await import(pathToFileURL(join(temporal, "portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js")));
    assert.deepEqual((await helper.cargarCatalogosContratacion("contratacion-temporal-analisis-catalogo")).actual, esperado);
  } finally {
    await rm(temporal, { recursive: true, force: true });
  }
});

test("un índice ausente conserva el idioma del documento sin bloquear el catálogo", async () => {
  const temporal = await mkdtemp(join(tmpdir(), "vec-ct-idioma-"));
  try {
    const codigo = codigos.ES;
    const fuentes = [
      "comun/idioma.js", "comun/textos.js",
      "portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js",
      `textos/${codigo}/contratacion-temporal-compatibilidad.json`,
      `textos/${codigo}/contratacion-temporal-analisis-catalogo.json`,
    ];
    const raiz = new URL("../../../", import.meta.url);
    for (const archivo of fuentes) {
      const destino = join(temporal, archivo);
      await mkdir(dirname(destino), { recursive: true });
      await cp(new URL(archivo, raiz), destino);
    }
    await writeFile(join(temporal, "package.json"), JSON.stringify({ type: "module" }));
    const entrada = pathToFileURL(join(temporal, "portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js")).href;
    const script = `
      import assert from 'node:assert/strict';
      globalThis.document = { documentElement: { lang: process.argv[2] } };
      const { cargarCatalogosContratacion } = await import(process.argv[1]);
      const catalogo = await cargarCatalogosContratacion('contratacion-temporal-analisis-catalogo');
      assert.ok(Object.keys(catalogo.actual).length > 0);
      for (const exportacion of Object.values(catalogo.exportaciones)) assert.deepEqual(exportacion, catalogo.actual);
    `;
    await promisify(execFile)(process.execPath, ["--input-type=module", "-e", script, entrada, codigo]);
  } finally {
    await rm(temporal, { recursive: true, force: true });
  }
});
