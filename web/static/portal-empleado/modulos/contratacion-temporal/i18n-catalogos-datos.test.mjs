import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFile } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";
import test from "node:test";
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";
import { cargarTextos } from "../../../comun/textos.js";
import { IDIOMAS_DISPONIBLES } from "../../../comun/idioma.js";
import { crearTraductorCancelacion } from "./i18n-cancelacion.js?v=20261001-ct-a-i18n-v1";
import { mensajesTramite, mensajesTramitePortal, rotuloTramite } from "./i18n-fases-rrhh.js?v=20261001-ct-a-i18n-v1";
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261001-ct-a-i18n-v1";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js?v=20261001-ct-a-i18n-v1";

// Huellas de las exportaciones originales en 463f7c176, anteriores al traslado.
// Incluyen nombres, orden de claves, textos completos y marcadores sin duplicar los textos.
const PREIMAGEN = {
  "i18n-analisis-catalogo.js": {
    "MENSAJES_ANALISIS_CATALOGO_EN": "f800e191693009fbda79315380b47ee28e1c5b5d653552845cd0a88d39d22f3b",
    "MENSAJES_ANALISIS_CATALOGO_ES": "7ebc3646ea6e5b35e13f785b67726b16d2957740c91a899382dd7f05d6ff0aa8"
  },
  "i18n-avisos-via-cobertura.js": {
    "MENSAJES_AVISOS_VIA_COBERTURA_EN": "8b1395a10ed65f002ccec703cb5fe8aef1b66d37b4bf2f29bc1fe6fc1feb1d34",
    "MENSAJES_AVISOS_VIA_COBERTURA_ES": "6d7de62fde85ab0b5004464ce518235e0c18f7041449414197b419812821dbcb"
  },
  "i18n-borradores-publicados.js": {
    // Excepción a la preimagen: corrección EN de bp_subtitulo posterior al traslado.
    "MENSAJES_BORRADORES_PUBLICADOS_EN": "595e5671fbc26190164f2b12bbb6a8f6ee52f872443d6a22385a64960a9cb1ef",
    "MENSAJES_BORRADORES_PUBLICADOS_ES": "6c9b8fb8abac09567848e5fbe00c33a54a26d8014032302648af8b6e297997ad"
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
    // Excepción a la preimagen (06/10/2026): filtros «Vencen hoy», «Con una
    // incidencia abierta» y «Sin plazo calculado» a los que lleva la portada, y
    // aviso de filtro parcial que ya no habla solo de la búsqueda.
    "MENSAJES_FICHA_LISTA_EN": "f7eb988f54b9ac84f9b1844b889f16a9c5db4a819f575ae5839364e3b1eba719",
    "MENSAJES_FICHA_LISTA_ES": "7ff24a5cdc854030c4b125889f8f3b59e4716ac73845674a842744335c4c59b0"
  },
  "i18n-informe-tras-subsanacion.js": {
    "MENSAJES_INFORME_TRAS_SUBSANACION_EN": "f2cf9af7062644110924a3c5a72d29235c556281e30b46dc008781723678067d",
    "MENSAJES_INFORME_TRAS_SUBSANACION_ES": "d75febfd37a4cf2265d502bcf2ae5af4ccad05b724433807974fea104abef743"
  },
  "i18n-llamamiento.js": {
    // Excepción a la preimagen: textos reescritos en lenguaje llano (05/10/2026), sin
    // clave de operación ni modo manual en pantalla; los límites pasan a la ayuda «?».
    "MENSAJES_LLAMAMIENTO_EN": "1f57663b12641ccc9b3189f2bc6cc35b2217e9a4d74519e3fafed7a1d8cecabf",
    "MENSAJES_LLAMAMIENTO_ES": "88a4c249b8fdd7f430c74356ffefa84e4c82a30af369ef21aeff4eae3e3ba054"
  },
  "i18n-subsanacion-reparos.js": {
    "MENSAJES_SUBSANACION_REPAROS_EN": "64125e70d662c685f79970383cc504776bf0923b034ea24825d30209c139833a",
    "MENSAJES_SUBSANACION_REPAROS_ES": "d153f27640795c4a871f3a5ac083127b259c9ac4d59398c3e6ced1366551b867"
  },
  "i18n-textos-vistas.js": {
    "MENSAJES_TEXTOS_VISTAS_EN": "c0bcbbd15ba13c327909b80f2b5ea8f721583d9a0703bf144185c9e2607f01d5",
    "MENSAJES_TEXTOS_VISTAS_ES": "8e3d9f5fc7ff76ae4c729bb9ec1bd308cfa91e4888c1200d40d67e47a55dcea4"
  }
};
// ANA002 añade cinco rótulos de contexto; la preimagen sigue comprobando todos los textos anteriores.
const CLAVES_CONTEXTO_ANA002 = Object.freeze([
  "ct_txt_contexto_fechas_consulta",
  "ct_txt_contexto_agrupacion",
  "ct_txt_contexto_zona_horaria",
  "ct_txt_contexto_corte_publicado",
  "ct_txt_contexto_no_comunicado",
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
const huella = (valor) => createHash("sha256").update(JSON.stringify(valor)).digest("hex");
const codigos = (await cargarTextos("contratacion-temporal-compatibilidad")).seccion("idiomas_exportados");

for (const [archivo, exportaciones] of Object.entries(PREIMAGEN)) {
  test(`${archivo}: conserva la preimagen de las exportaciones y los rótulos`, async () => {
    const modulo = await import(new URL(archivo, import.meta.url));
    for (const [nombre, anterior] of Object.entries(exportaciones)) {
      const valor = nombre.startsWith("rotulos_")
        ? Object.fromEntries(Object.entries(mensajesTramitePortal(nombre.slice("rotulos_".length)))
          .map(([clave, texto]) => [clave.slice("tramite_".length), texto]))
        : modulo[nombre];
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
        for (const clave of CLAVES_REINCORPORACION_CAPACIDAD) {
          assert.ok(typeof valor[clave] === "string" && valor[clave].trim(), `${nombre}.${clave}`);
        }
        preimagen = Object.fromEntries(Object.entries(valor)
          .filter(([clave]) => !CLAVES_REINCORPORACION_CAPACIDAD.includes(clave)));
      }
      assert.equal(huella(preimagen), anterior, nombre);
      assert.ok(Object.isFrozen(valor) || nombre.startsWith("rotulos_"), nombre);
    }
  });
}

test("los catálogos y los traductores conservan marcadores, sobrescrituras y claves desconocidas", async () => {
  const contratacion = await import("./i18n.js");
  const expedientes = await import("./i18n-expedientes.js");
  for (const [nombre, codigo] of Object.entries(codigos)) {
    for (const [mensajes, crear] of [
      [contratacion[`MENSAJES_CONTRATACION_TEMPORAL_${nombre}`], crearTraductorContratacionTemporal],
      [expedientes[`MENSAJES_EXPEDIENTES_CONTRATACION_${nombre}`], crearTraductorExpedientesContratacion],
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
    const fases = (await cargarTextos("portal", { idioma: codigo })).seccion("fases_rrhh");
    for (const [clave, texto] of Object.entries(fases)) assert.equal(rotuloTramite(clave, {}, codigo), texto);
    const tramite = mensajesTramite(codigo);
    assert.equal(tramite.fase_rrhh_orden_nombre, fases.fase_de_nombre);
    assert.equal(tramite.etiqueta_fase_incorporacion, fases.fase_incorporacion);
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
    const original = await import("./i18n-analisis-catalogo.js");
    assert.deepEqual(modulo.MENSAJES_ANALISIS_CATALOGO_ES, original.MENSAJES_ANALISIS_CATALOGO_ES);
    assert.deepEqual(modulo.MENSAJES_ANALISIS_CATALOGO_EN, original.MENSAJES_ANALISIS_CATALOGO_EN);
    const helper = await import(pathToFileURL(join(temporal, "portal-empleado/modulos/contratacion-temporal/i18n-catalogos.js")));
    assert.deepEqual((await helper.cargarCatalogosContratacion("contratacion-temporal-analisis-catalogo")).actual, original.MENSAJES_ANALISIS_CATALOGO_EN);
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
