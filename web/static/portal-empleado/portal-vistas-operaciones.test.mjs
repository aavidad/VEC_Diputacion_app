import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js";
import { crearTraductorContratos, MENSAJES_CONTRATOS_ES } from "./portal-i18n-contratos.js";

test("el import de i18n de contratos renueva la URL del asset inmutable", () => {
  const codigo = readFileSync(new URL("./portal-vistas-operaciones.js", import.meta.url), "utf8");
  assert.match(codigo, /from "\.\/portal-i18n-contratos\.js\?v=20260924-f2-web2";/u);
});

function utilidades() {
  return crearUtilidadesVista({
    escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;"),
    numero: (valor) => String(valor ?? 0),
    claseEstado: () => "neutro",
    encabezadoVista: (_sobrelinea, titulo, descripcion, acciones = "") => `<h2>${titulo}</h2><p>${descripcion}</p>${acciones}`,
  });
}

test("operaciones expone únicamente la consulta de contratos", () => {
  const vistas = crearVistasOperaciones(utilidades());
  assert.deepEqual(Object.keys(vistas), ["renderizarContratos"]);
  assert.match(vistas.renderizarContratos({}), /No configurado/);
});

test("contratos sin fuente ignora filas ajenas y no ofrece efectos volátiles", () => {
  const html = crearVistasOperaciones(utilidades()).renderizarContratos({
    contratos: [{ expediente: "EXP-LEGADO", acto: "Alta", bolsa: "Bolsa 1", estado: "Vigente" }],
  });
  assert.match(html, /Una propuesta de llamamiento no acredita aceptación/);
  assert.match(html, /Solo Personal confirma la relación y la incorporación/);
  assert.match(html, /El borrador y la autenticación no son firma/);
  assert.match(html, /La ficha o descarga no acredita entrega al sistema/);
  assert.match(html, /El cese acreditado precede a la política de Bolsa/);
  assert.match(html, /No configurado/);
  assert.match(html, /No hay relaciones disponibles para mostrar/);
  assert.doesNotMatch(html, /EXP-LEGADO/);
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
  const contratos = [{ expediente: "EXP-1", acto: "Contrato", bolsa: "Bolsa 1",
    inicio: "2026-09-24", fin: "2026-10-24", estado: "Vigente" }];
  const disponible = vista.renderizarContratos({ contratos_fuente: { estado: "disponible", registros: contratos } });
  assert.match(disponible, /EXP-1/);
  assert.match(disponible, /Disponible/);
  for (const estado of ["cargando", "vacio", "denegado", "error", "no_configurado", "desconocido"]) {
    const html = vista.renderizarContratos({ contratos_fuente: { estado, registros: contratos } });
    assert.doesNotMatch(html, /EXP-1/);
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
