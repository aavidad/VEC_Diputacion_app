import assert from "node:assert/strict";
import test from "node:test";
import { cuerpoPortalMiBolsa, enviarPortalMiBolsa, validarPortalMiBolsa } from "./mi-bolsa-portal.js";
import { renderizarContactoMiBolsa, validarContactosMiBolsa } from "./mi-bolsa-contacto.js";

const participaciones = [{ bolsa: "bolsa:demo:1", categoria: "Auxiliar de enfermería" }];
const contacto = (extra = {}) => ({ bolsa: "bolsa:demo:1", version: 2, origen: { estado: "vigente", ultimo_dia: "2027-09-24" }, confirmada_en: null, ...extra });

test("valida los contactos sin admitir datos ajenos ni el claro", () => {
  assert.doesNotThrow(() => validarContactosMiBolsa({ participaciones }));
  assert.doesNotThrow(() => validarPortalMiBolsa({ participaciones, contactos: [contacto()] }));
  assert.throws(() => validarPortalMiBolsa({ participaciones, contactos: [contacto({ bolsa: "bolsa:ajena" })] }), /ajena/u);
  assert.throws(() => validarContactosMiBolsa({ participaciones, contactos: [contacto({ correo: "x@y.es" })] }), /no autorizados/u);
  assert.throws(() => validarContactosMiBolsa({ participaciones, contactos: [contacto({ version: 0 })] }), /no válido/u);
});

test("ofrece confirmar solo el contacto de CONVOCA sin confirmar", () => {
  assert.match(renderizarContactoMiBolsa(participaciones, [contacto()]), /data-portal-mi-bolsa="contacto" data-bolsa="bolsa:demo:1" data-version="2"/u);
  assert.match(renderizarContactoMiBolsa(participaciones, [contacto({ origen: { estado: "vencido", ultimo_dia: "2026-09-24" } })]), /pendiente desde/u);
  const confirmado = renderizarContactoMiBolsa(participaciones, [contacto({ confirmada_en: "2026-09-25T08:00:00.000000Z" })]);
  assert.doesNotMatch(confirmado, /data-portal-mi-bolsa/u);
  assert.match(confirmado, /confirmado el/u);
  assert.match(renderizarContactoMiBolsa(participaciones, [contacto({ origen: null })]), /registrado por RRHH/u);
  assert.equal(renderizarContactoMiBolsa(participaciones, []), "");
});

test("envía la bolsa y la versión vistas con clave estable y explica el cambio", async () => {
  const formulario = { dataset: { portalMiBolsa: "contacto", bolsa: "bolsa:demo:1", version: "2" }, querySelector: () => null };
  const primera = await cuerpoPortalMiBolsa(formulario, new FormData());
  assert.equal(primera.ruta, "/api/vec/bolsa/mi-bolsa/contacto");
  assert.deepEqual({ ...primera.cuerpo, clave: "" }, { bolsa: "bolsa:demo:1", version: 2, clave: "" });
  assert.equal((await cuerpoPortalMiBolsa(formulario, new FormData())).cuerpo.clave, primera.cuerpo.clave);
  const zona = { textContent: "" };
  const conZona = { ...formulario, querySelector: (s) => (s === "[data-portal-resultado]" ? zona : null) };
  const fetchImpl = async () => ({ status: 409, json: async () => ({ error: { codigo: "contacto_cambiado" } }) });
  assert.equal(await enviarPortalMiBolsa(conZona, { fetchImpl, datos: new FormData() }), false);
  assert.match(zona.textContent, /actualizado su contacto/u);
});
