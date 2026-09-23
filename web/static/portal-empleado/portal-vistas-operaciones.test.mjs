import assert from "node:assert/strict";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js";
import { crearTraductorContratos, MENSAJES_CONTRATOS_ES } from "./portal-i18n-contratos.js";

function utilidades() {
  return crearUtilidadesVista({
    escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;"),
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
  assert.match(contratos, /no acredita por sí sola relación jurídica, cese ni incorporación/);
  assert.match(documentos, /La autenticación no firma documentos y ningún borrador es oficial/);
  assert.match(documentos, /Firmar DEMO · sin firma legal/);
  assert.match(comunicaciones, /Un aviso no acredita envío, entrega, notificación ni acuse/);
});

test("contratos sin fuente no incorpora las filas antiguas y no ofrece efectos volátiles", () => {
  const html = crearVistasOperaciones(utilidades()).renderizarContratos(obtenerDatosPresentacion());
  assert.match(html, /Una propuesta de llamamiento no acredita aceptación/);
  assert.match(html, /Solo Personal confirma la relación y la incorporación/);
  assert.match(html, /El borrador y la autenticación no son firma/);
  assert.match(html, /La ficha o descarga no acredita entrega al sistema/);
  assert.match(html, /El cese acreditado precede a la política de Bolsa/);
  assert.match(html, /No configurado/);
  assert.match(html, /No hay relaciones disponibles para mostrar/);
  assert.doesNotMatch(html, /DEMO-CON-184|DEMO-CES-089|DEMO-REI-032/);
  assert.match(html, /tabindex="0" role="region"/);
  assert.match(html, /<summary>\? Ayuda sobre el circuito<\/summary>/);
  assert.equal((html.match(/name="contratos-recorrido"/g) || []).length, 3);
  assert.match(html, /<summary><strong>Contratos y relaciones<\/strong>/);
  assert.match(html, /<summary><strong>Ceses<\/strong>/);
  assert.match(html, /<summary><strong>Reincorporación en Bolsa<\/strong>/);
  assert.equal((html.match(/disabled aria-disabled="true"/g) || []).length, 3);
  assert.doesNotMatch(html, /data-comando="(?:registrar-contrato|registrar-cese|reincorporar-bolsa)"/);
  assert.doesNotMatch(html, /data-accion="operacion-presentacion"/);
});

test("contratos conserva solo filas inyectadas con estado disponible y cierra los demás estados", () => {
  const vista = crearVistasOperaciones(utilidades());
  const contratos = obtenerDatosPresentacion().contratos;
  const disponible = vista.renderizarContratos({ contratos_fuente: { estado: "disponible", registros: contratos } });
  assert.match(disponible, /DEMO-CON-184/);
  assert.match(disponible, /Disponible/);
  for (const estado of ["cargando", "vacio", "denegado", "error", "no_configurado", "desconocido"]) {
    const html = vista.renderizarContratos({ contratos_fuente: { estado, registros: contratos } });
    assert.doesNotMatch(html, /DEMO-CON-184/);
    assert.match(html, /No hay relaciones disponibles para mostrar/);
  }
  assert.match(vista.renderizarContratos({ contratos_fuente: { estado: "disponible", registros: [] } }), /Sin registros/);
  assert.match(vista.renderizarContratos({ contratos_fuente: { estado: "disponible" } }), /Error de consulta/);
  assert.match(vista.renderizarContratos({ contratos_fuente: { estado: "disponible", registros: [null] } }), /Error de consulta/);
  const malicioso = [{ ...contratos[0], expediente: '<img src=x onerror=alert(1)>' }];
  const escapado = vista.renderizarContratos({ contratos_fuente: { estado: "disponible", registros: malicioso } });
  assert.match(escapado, /&lt;img src=x onerror=alert\(1\)&gt;/);
  assert.doesNotMatch(escapado, /<img src=x/);
  assert.throws(() => crearTraductorContratos({}), /incompleto/);
  assert.throws(() => crearTraductorContratos()("desconocida"), /desconocida/);
  for (const valor of Object.values(MENSAJES_CONTRATOS_ES)) assert.ok(valor.length > 0);
});
