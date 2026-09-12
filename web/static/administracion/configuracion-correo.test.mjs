import test from "node:test";
import assert from "node:assert/strict";
import { CORREO_ENDPOINT, crearClienteCorreo, montarConfiguracionCorreo, normalizarEntrada } from "./configuracion-correo.js";

test("cliente usa la ruta administrativa y no transmite secreto vacío", async () => {
  const llamadas=[]; const cliente=crearClienteCorreo({fetchImpl:async(u,o)=>{llamadas.push({u,o});return new Response("{}",{status:200,headers:{"Content-Type":"application/json"}})}});
  await cliente.consultar(); await cliente.actualizar({host:"10.0.0.2",secreto:""});
  assert.equal(CORREO_ENDPOINT,"/api/vec/administracion/configuracion-correo"); assert.equal(llamadas[0].u,CORREO_ENDPOINT); assert.deepEqual(JSON.parse(llamadas[1].o.body),{host:"10.0.0.2"});
});
test("solo deja pasar campos declarados y secreto escrito",()=>assert.deepEqual(normalizarEntrada({host:"x",secreto:"nuevo",bearer:"no"}),{host:"x",secreto:"nuevo"}));

test("descarta una carga pendiente cuando se desmonta la vista", async () => {
  let resolver;
  const pendiente = new Promise((resolve) => { resolver = resolve; });
  const contenedor = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const vista = montarConfiguracionCorreo(contenedor, { cliente: { consultar: () => pendiente } });
  const carga = vista.cargar();
  vista.desmontar();
  resolver({ configuracion: {} });
  assert.equal(await carga, false);
  assert.equal(contenedor.innerHTML, "");
});

test("renderiza usando el traductor inyectado y escapa sus textos", async () => {
  const claves = []; const alerta = { textContent: "" }; const estado = { textContent: "" };
  const formulario = { elements: {}, querySelector: (selector) => selector === "[role=alert]" ? alerta : estado, addEventListener() {} };
  const contenedor = { innerHTML: "", replaceChildren() { this.innerHTML = ""; }, querySelector: (selector) => selector === "form" ? formulario : null };
  const traducir = (clave) => { claves.push(clave); return clave === "admin_correo_titulo_formulario" ? "Correo <ADMIN>" : clave; };
  const vista = montarConfiguracionCorreo(contenedor, { traducir, cliente: { consultar: async () => ({ configuracion: {} }) } });
  assert.equal(await vista.cargar(), true);
  assert.match(contenedor.innerHTML, /Correo &lt;ADMIN&gt;/);
  assert.ok(claves.includes("admin_correo_titulo_formulario"));
});
