import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { readFile } from "node:fs/promises";
import { promisify } from "node:util";
import test from "node:test";

import { cargarTextos, esCatalogoValido } from "../../comun/textos.js";

const ejecutar = promisify(execFile);
const catalogo = async (idioma) => JSON.parse(await readFile(new URL(`../../textos/${idioma}/peticiones-centro.json`, import.meta.url)));

test("los catálogos PC ES y EN contienen las mismas claves y ninguna sección vacía", async () => {
  const [es, en] = await Promise.all([catalogo("es"), catalogo("en")]);
  assert.equal(esCatalogoValido(es), true);
  assert.equal(esCatalogoValido(en), true);
  for (const seccion of ["general", "ayuda", "incorporaciones", "cancelaciones"]) {
    assert.deepEqual(Object.keys(en[seccion]).sort(), Object.keys(es[seccion]).sort(), seccion);
  }
});

test("la navegación EN presenta bandeja, formulario y ayudas propias sin rótulos castellanos", async () => {
  const rutas = ["peticiones-centro.js", "i18n-peticiones-centro.js", "incorporaciones-centro.js", "cancelaciones-centro.js"]
    .map((nombre) => new URL(nombre, import.meta.url).href);
  const contrato = new URL("../modulos/contratacion-temporal/contrato.js", import.meta.url).href;
  const script = `globalThis.location={href:"http://local/peticiones-centro/?lang=en",search:"?lang=en"};
    const [pc,i18n,inc,can,ct]=await Promise.all([${[...rutas, contrato].map((ruta) => `import(${JSON.stringify(ruta)})`).join(",")}]);
    const catalogos={centros:[{referencia:"c1",etiqueta:"Centre",contactos:[{referencia:"x1",etiqueta:"Contact"}]}],
      categorias:[{referencia:"cat1",etiqueta:"Category",grupos_subgrupos:[{clave:"C2",etiqueta:"C2"}]}],
      motivos:[{clave:"sustitucion",etiqueta:"Replacement"}],documentos:[]};
    const contexto={actor:{referencia:"actor:test",puede_presentar:true,puede_ratificar:false},catalogos};
    const estado={fase:"edicion",disponible:true,ocupado:false,borrador:ct.crearBorradorAlta({conPeticionCentro:true,jornadaReferenciaMinutos:2250}),catalogos,
      errores:{puesto_solicitado:"puesto_solicitado",jornada_minutos:"jornada"},
      mensaje_clave:"estado_disponible",tipo_mensaje:"informacion"};
    process.stdout.write(JSON.stringify({lista:pc.renderizarPeticionCentro({contexto}),
      formulario:pc.renderizarPeticionCentro({contexto,modo:"formulario",estado}),
      ayuda:i18n.MENSAJES_AYUDA_PETICIONES_CENTRO.pc_ayuda_titulo,
      incorporaciones:inc.crearTraductorIncorporacionesCentro()( "titulo" ),
      cancelaciones:can.crearTraductorCancelacionesCentro()( "titulo" )}));`;
  const { stdout } = await ejecutar(process.execPath, ["--input-type=module", "-e", script]);
  const pantalla = JSON.parse(stdout);
  assert.match(pantalla.lista, /Staff requests from your centre/u);
  assert.match(pantalla.formulario, /Centre and staffing need/u);
  assert.equal((pantalla.formulario.match(/up to 160 characters/gu) || []).length, 2);
  assert.equal((pantalla.formulario.match(/Maximum: 168 hours/gu) || []).length, 2);
  assert.doesNotMatch(pantalla.formulario, /up to 4,000 characters/iu);
  assert.equal(pantalla.ayuda, "Centre request: what it does and does not do");
  assert.equal(pantalla.incorporaciones, "Centre incorporation confirmations");
  assert.equal(pantalla.cancelaciones, "Cancel a case");
  assert.doesNotMatch(JSON.stringify(pantalla), /Petici[oó]n del centro|Peticiones de personal|Incorporaciones del centro|Cancelar un expediente/u);
});

test("un catálogo EN inválido se reintenta y usa el ES completo sin bloquear la vista", async () => {
  const es = await catalogo("es");
  const pedidos = [];
  const textos = await cargarTextos("peticiones-centro", { idioma: "en", porDefecto: "es",
    leer: async (url) => { pedidos.push(url.pathname); return url.pathname.includes("/en/") ? { general: {} } : es; },
    avisar() {},
  });
  assert.equal(textos.idioma, "es");
  assert.equal(textos.incidenciaCatalogo.codigo, "catalogo_no_disponible");
  assert.equal(textos.seccion("general").pc_titulo, "Peticiones de personal de su centro");
  assert.deepEqual(pedidos.map((url) => url.includes("/en/") ? "en" : "es"), ["en", "en", "es"]);
});
