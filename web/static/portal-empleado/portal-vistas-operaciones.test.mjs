import assert from "node:assert/strict";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js";
import { crearTraductorContratos, MENSAJES_CONTRATOS_ES } from "./portal-i18n-contratos.js";

function utilidades() {
  return crearUtilidadesVista({
    escaparHTML: (valor) => String(valor),
    numero: (valor) => String(valor ?? 0),
    claseEstado: () => "neutro",
    encabezadoVista: (_sobrelinea, titulo, descripcion, acciones = "") => `<h2>${titulo}</h2><p>${descripcion}</p>${acciones}`,
    esPresentacion: () => true,
    operacionPermitida: () => true,
  });
}

test("las vistas operativas rotulan el recorrido DEMO sin alterar sus comandos", () => {
  const vistas = crearVistasOperaciones(utilidades());
  const datos = obtenerDatosPresentacion();
  const llamamientos = vistas.renderizarLlamamientos(datos);
  const contratos = vistas.renderizarContratos(datos);
  const documentos = vistas.renderizarDocumentos(datos);
  const comunicaciones = vistas.renderizarComunicaciones(datos);
  assert.match(llamamientos, /Llamamientos DEMO/);
  assert.match(llamamientos, /no fijan una regla ni un plazo operativo/);
  assert.match(contratos, /no acredita relación jurídica, cese ni incorporación/);
  assert.match(documentos, /La autenticación no firma documentos y ningún borrador es oficial/);
  assert.match(documentos, /Firmar DEMO · sin firma legal/);
  assert.match(comunicaciones, /Un aviso no acredita envío, entrega, notificación ni acuse/);
});

test("contratos distingue los hechos de Bolsa, Personal, firma y GINPIX sin ofrecer efectos volátiles", () => {
  const html = crearVistasOperaciones(utilidades()).renderizarContratos(obtenerDatosPresentacion());
  assert.match(html, /Una propuesta de llamamiento no acredita aceptación/);
  assert.match(html, /Solo Personal confirma la relación y la incorporación/);
  assert.match(html, /El borrador y la autenticación no son firma/);
  assert.match(html, /La ficha o descarga no acredita entrega al sistema/);
  assert.match(html, /El cese acreditado precede a la política de Bolsa/);
  assert.match(html, /Estado en muestra/);
  assert.match(html, /tabindex="0" role="region"/);
  assert.match(html, /<summary>\? Ayuda sobre el circuito<\/summary>/);
  assert.equal((html.match(/disabled aria-disabled="true"/g) || []).length, 3);
  assert.doesNotMatch(html, /data-comando="(?:registrar-contrato|registrar-cese|reincorporar-bolsa)"/);
  assert.doesNotMatch(html, /data-accion="operacion-presentacion"/);
});

test("contratos indica vacío y usa el catálogo completo", () => {
  const html = crearVistasOperaciones(utilidades()).renderizarContratos({ contratos: [] });
  assert.match(html, /No hay ejemplos de contratos para mostrar/);
  assert.throws(() => crearTraductorContratos({}), /incompleto/);
  assert.throws(() => crearTraductorContratos()("desconocida"), /desconocida/);
  for (const valor of Object.values(MENSAJES_CONTRATOS_ES)) assert.ok(valor.length > 0);
});
