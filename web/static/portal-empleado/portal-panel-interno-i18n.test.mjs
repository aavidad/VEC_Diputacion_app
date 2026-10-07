import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { traducirPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20261001-ct-a-i18n-v1";
import { traducirAvisoPanelInterno } from "./portal-panel-interno-i18n.js";

test("B7 traduce los cuatro pasos y distingue registro, recibo y entrega", () => {
  const bolsa = { bolsa_ref: "bolsa:01", categoria: "Auxiliar", tipo_lista: "ordinaria",
    vigente_desde: "2026-09-01", total: 1, por_estado: { disponible: 1 } };
  const flujo = { paso: 1, estados: ["disponible"], participaciones: ["participacion:01"],
    configuracion: null, error: "", recibo: "" };
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "info",
    encabezadoVista: (_area, titulo, descripcion, acciones) => `<header><h2>${titulo}</h2><p>${descripcion}</p>${acciones}</header>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;"),
    numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (valor) => valor,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: {
      bolsa, candidatos: [{ participacion_ref: "participacion:01", estado_clave: "disponible", orden: 1, nombre_visible: "Persona" }],
      contactos: [],
    } }),
    obtenerEstadoCandidatos: () => ({ nuevo_llamamiento: flujo }),
  });
  const renderizar = () => presentador.renderizarVista("bolsa-candidatos");

  assert.equal(traducirPortal("panel_b7_ayuda_limite_aria"), "Ayuda sobre el límite de selección");
  assert.match(renderizar(), /1\. Seleccionar bolsa/);
  flujo.paso = 2;
  const seleccion = renderizar();
  // El límite de selección se explica en la ayuda «?», no en la pantalla.
  assert.doesNotMatch(seleccion, /<details|<summary/);
  flujo.paso = 3;
  const configuracion = renderizar();
  assert.match(configuracion, /name="plazo" required minlength="2" maxlength="160" value=""/);
  assert.doesNotMatch(configuracion, /no presupone un plazo legal|no acreditan entrega/);
  assert.match(traducirPortal("ayuda_b7_configurar_limite"), /no presupone un plazo legal.*no acreditan la entrega/);
  assert.doesNotMatch(configuracion, /48 horas|relay de desarrollo|Recorrido real B7/);
  flujo.configuracion = { plazo: "Pendiente de definición por RRHH" };
  assert.match(renderizar(), /name="plazo" required minlength="2" maxlength="160" value=""/);
  flujo.paso = 4;
  flujo.configuracion = { referencia: "NEC-1", centro: "Centro", modalidad: "Sustitución", plazo: "Indicado por RRHH" };
  flujo.error = "Permiso denegado";
  flujo.recibo = "recibo:123";
  flujo.llamamiento_ref = "llamamiento:123";
  const confirmacion = renderizar();
  assert.match(confirmacion, /Permiso denegado/);
  // El justificante se ofrece para copiar; la referencia no se muestra como texto.
  assert.match(confirmacion, /Justificante registrado<\/span> <button[^>]+data-copiar-justificante="recibo:123"/);
  assert.match(confirmacion, /Llamamiento emitido/);
  assert.doesNotMatch(confirmacion, />llamamiento:123|<code>/);
  assert.match(confirmacion, /la entrega del correo debe comprobarse/);
  assert.doesNotMatch(confirmacion, /relay|PostgreSQL|Recorrido real B7/);
});

test("Bolsa distingue anotaciones de contacto y respuesta formal sin inventar expediente", async () => {
  const bolsa = { bolsa_ref: "bolsa:01", categoria: "Auxiliar", tipo_lista: "ordinaria",
    vigente_desde: "2026-09-01", vigente_hasta: null, total: 1, por_estado: { disponible: 1 },
    politica_orden: { tipo_lista: "ordinaria", reposicion: "misma_posicion", version: 1, vigente_desde: "2026-09-01" } };
  const candidato = { participacion_ref: "participacion:01", orden: 1, orden_acta: 1,
    razon_orden: "orden_acta", nombre_visible: "Persona", documento_enmascarado: "***0001**",
    estado_clave: "disponible", estado_desde: "2026-09-01T09:00:00Z", disponible_desde: null,
    ultimo_llamamiento: { llamamiento_ref: "llamamiento:01", comunicado_en: "2026-09-02T09:00:00Z", canal: "correo", resultado: "sin_respuesta" },
    contactos_total: 0 };
  let modalFicha = null;
  let carga = "listo";
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "info", encabezadoVista: (_area, titulo) => `<h2>${titulo}</h2>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;"),
    numero: (valor) => String(valor ?? 0), obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (valor) => valor,
    obtenerDatosCandidatosBolsa: () => ({ carga, datos: { bolsa, candidatos: [candidato], contactos: [] } }),
    obtenerEstadoCandidatos: () => ({ filtros: { pestana: "candidatos", estado: "", texto: "" } }),
    obtenerModalFicha: () => modalFicha,
  });
  const html = presentador.renderizarVista("bolsa-candidatos");
  // La aceptación o renuncia se registra en Contratación temporal: el botón lleva allí, nunca queda muerto.
  assert.match(html, /<a class="boton-secundario boton-ancho" href="\/portal-empleado\/#contratacion-temporal">Registrar resultado en Peticiones de personal temporal<\/a>/);
  assert.doesNotMatch(html, /bolsa-resultado-sin-expediente|disabled aria-disabled="true">Registrar resultado/);
  assert.doesNotMatch(html, /data-bolsa-accion="abrir-resultado"|data-bolsa-form="resultado"|href="[^"]*llamamiento:01/);

  modalFicha = { abierto: true, candidato, bolsa, registroContacto: {}, intentosContacto: { carga: "cargando" }, reciboContacto: "recibo:contacto:01" };
  const ficha = presentador.renderizarVista("bolsa-candidatos");
  assert.match(ficha, /data-bolsa-form="contacto"[^>]*aria-describedby="bolsa-contacto-no-respuesta"/);
  assert.match(ficha, /Este formulario anota contactos/);
  assert.match(ficha, /<option value="sms">|<option value="presencial">|<option value="buzon">|<option value="aplazado">/);
  assert.doesNotMatch(ficha, /<option value="acepta"|<option value="rechaza"/);
  assert.match(ficha, /data-copiar-justificante="recibo:contacto:01"/);
  assert.match(ficha, /data-contacto-rrhh-accion="abrir"/);

  candidato.ultimo_llamamiento = null;
  const fichaSinLlamamiento = presentador.renderizarVista("bolsa-candidatos");
  assert.match(fichaSinLlamamiento, /data-bolsa-form="contacto"/);
  assert.match(fichaSinLlamamiento, /<option value="telefono">|<option value="sms">|<option value="presencial">/);
  assert.doesNotMatch(fichaSinLlamamiento, /name="llamamiento_ref"|<option value="acepta"|<option value="rechaza"/);

  carga = "denegado";
  const sinPermiso = presentador.renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(sinPermiso, /Registrar resultado|bolsa-resultado-sin-expediente|data-bolsa-accion="abrir-ficha"/);

  // En Node la interfaz está en el idioma por defecto; el inglés se comprueba en su catálogo de datos.
  const ingles = JSON.parse(await readFile(new URL("../textos/en/portal.json", import.meta.url), "utf8")).panel_interno;
  assert.equal(Object.hasOwn(ingles, "panel_resultado_sin_expediente"), false);
  assert.match(ingles.panel_contacto_no_respuesta, /If the person accepts or declines/);
});
