import fs from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL, fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { chromeAccesible, comprobarAccesibilidad, activarLectura } from '../../recorridos-f/a11y.mjs';
import { entregar, guardarAlmacenamiento } from './guardas.mjs';
import { montarFixture } from './fixture.mjs';

const carpeta = fileURLToPath(new URL('./', import.meta.url));
const raiz = path.resolve(carpeta, '../../..');
const css = ['portal.css', 'portal-componentes.css', 'modulos/cronos/cronos.css',
  'modulos/cronos/cronos-conectado.css', 'modulos/cronos/vista-solicitudes.css', 'modulos/cronos/vista-resolucion.css'];

export function argumentos(args) {
  if (args.length === 1 && args[0] === '--plan') return { plan: true };
  if (args.length === 2 && args[0] === '--salida' && path.isAbsolute(args[1])) return { salida: args[1] };
  throw new Error('argumentos_invalidos');
}

export async function plan() {
  return JSON.parse(await fs.readFile(new URL('./casos.json', import.meta.url), 'utf8'));
}

async function prepararSalida(salida) {
  const padre = path.dirname(salida), st = await fs.lstat(padre);
  if (salida.startsWith(`${raiz}/`) || salida === raiz || await fs.realpath(padre) !== padre
    || !st.isDirectory() || st.uid !== process.getuid() || (st.mode & 0o077)) throw new Error('salida_invalida');
  await fs.mkdir(salida, { mode: 0o700 }); // EEXIST corta: no pisa evidencia previa.
}

export function html(lang, casos) {
  return `<!doctype html><html lang="${lang}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${casos.textos[lang].titulo}</title>${css.map(s => `<link rel="stylesheet" href="/portal-empleado/${s}">`).join('')}</head><body class="portal-empleado-app"><main id="espacio-trabajo" style="padding:16px;min-width:0"><h1>${casos.textos[lang].titulo}</h1><div id="lectura"></div></main></body></html>`;
}

// Chrome recorre por Tab los segmentos de fecha/hora. El helper F cuenta
// controles DOM; agrupamos hasta 12 segmentos, sin saltar el control siguiente.
export function paginaConCamposNativos(page) {
  const press = async tecla => {
    const anterior = await page.evaluate(() => document.activeElement?.matches('input[type="date"],input[type="time"]') === true);
    if (anterior && ['Tab', 'Shift+Tab'].includes(tecla)) {
      await page.evaluate(() => { window.__campoNativo = document.activeElement; });
      for (let paso = 0; paso < 12; paso++) {
        await page.keyboard.press(tecla);
        if (!(await page.evaluate(() => document.hasFocus() && document.activeElement === window.__campoNativo))) break;
      }
      await page.evaluate(() => { delete window.__campoNativo; });
    } else await page.keyboard.press(tecla);
    // La página usa desplazamiento suave. Medir durante la animación atribuiría
    // una obstrucción a un control que Chrome está llevando a la vista.
    await page.evaluate(() => new Promise(resolve => {
      let anterior = '', estables = 0, pasos = 0;
      const medir = () => {
        const r = document.activeElement?.getBoundingClientRect();
        const posicion = `${scrollX}|${scrollY}|${r?.top}|${r?.left}`;
        estables = posicion === anterior ? estables + 1 : 0; anterior = posicion;
        if (estables >= 3 || ++pasos >= 60) resolve(); else requestAnimationFrame(medir);
      };
      requestAnimationFrame(medir);
    }));
  };
  return new Proxy(page, { get(obj, clave) {
    if (clave === 'keyboard') return { press };
    const valor = Reflect.get(obj, clave); return typeof valor === 'function' ? valor.bind(obj) : valor;
  } });
}

async function comprobarInteraccion(page, pantalla) {
  const activar = selector => activarLectura(page, page.locator(selector).first());
  if (pantalla === 'fichaje') {
    assert.equal(await page.locator('[data-cronos-remoto-movimiento]:not(:disabled)').count(), 1);
    await activar('[data-cronos-remoto-actualizar]');
    await page.locator('[data-cronos-remoto-movimiento="entrada"]:not(:disabled)').waitFor();
  } else if (pantalla === 'saldo') {
    await activar('[data-cronos-saldo-periodo="rango"]');
    await page.locator('[data-cronos-saldo-rango]:not([hidden])').waitFor();
  } else if (pantalla === 'calendario_incidencias') {
    const filtro = page.locator('[data-cronos-filtro="justificante"]');
    await filtro.focus(); await page.keyboard.press('ArrowDown'); await page.keyboard.press('Tab');
    assert.equal(await filtro.inputValue(), 'pendiente');
    await page.locator('[data-cronos-quitar-filtros]').waitFor();
    assert.equal(await page.locator('[data-fecha][data-tipos~="ausencia"]').count(), 2);
    await activar('[data-cronos-olvido="abrir"]');
    await page.locator('[data-cronos-olvido-formulario]').waitFor();
  } else {
    await activar('[data-cronos-resolver]');
    await page.locator('[data-cronos-resolucion-formulario]').waitFor();
    assert.equal(await page.locator('[name="decision"]').first().evaluate(el => el === document.activeElement), true);
  }
}

export async function recorrer(salida, casos) {
  await prepararSalida(salida);
  const scratch = await fs.mkdtemp(path.join(salida, '.scratch-'));
  const resultados = [];
  const informe = { version: 1, alcance: casos.alcance, e2e: false, autenticacion: false,
    autorizacion: false, persistencia: false, reinicio: false,
    medicion: { campos_nativos: 'agrupados', max_segmentos: 12, frames_estables: 3, max_frames: 60,
      foco: 'cinco_puntos_conservador', conformidad_wcag: false }, resultados };
  let chrome, etapa = 'chrome';
  try {
    const { chromium } = await import(pathToFileURL(process.env.VEC_PLAYWRIGHT_MODULE).href);
    chrome = await chromeAccesible(chromium, scratch);
    for (const idioma of casos.idiomas) for (const vista of casos.vistas) for (const pantalla of casos.pantallas) {
      etapa = `${pantalla}_${idioma}_${vista.ancho}_${vista.zoom}_montaje`;
      await chrome.zoom(vista.zoom);
      const contexto = await chrome.browser.newContext({ viewport: { width: vista.ancho, height: 900 }, serviceWorkers: 'block' });
      const red = { entregadas: 0, bloqueadas: 0, websocket: 0, paginas_extra: 0 };
      const errores = [];
      try {
        await contexto.route('**/*', route => entregar(route, path.join(raiz, 'web/static'), html(idioma, casos), red, casos));
        await contexto.routeWebSocket('**/*', route => { red.websocket++; route.close(); });
        await contexto.addInitScript(guardarAlmacenamiento);
        const pagina = await contexto.newPage();
        contexto.on('page', extra => { red.paginas_extra++; void extra.close(); });
        pagina.on('pageerror', () => errores.push('error_js')); // No registra cuerpos ni textos de persona.
        await pagina.goto(`${casos.origen}/fixture?lang=${idioma}`, { waitUntil: 'load', timeout: 20000 });
        await pagina.evaluate(montarFixture, { pantalla, casos });
        const selector = { fichaje: '[data-cronos-remoto-movimiento="entrada"]:not(:disabled)',
          saldo: '[data-cronos-saldo-estado="listo"]', calendario_incidencias: '.cronos-movimientos-propios[data-estado="listo"]',
          bandeja: '.cronos-bandeja-permisos[data-estado="listo"]' }[pantalla];
        await pagina.locator(selector).waitFor({ timeout: 10000 });
        etapa = `${pantalla}_${idioma}_${vista.ancho}_${vista.zoom}_auditoria`;
        const accesible = paginaConCamposNativos(pagina);
        const estados = [await comprobarAccesibilidad(accesible, '#lectura', 'inicial', vista.ancho, vista.zoom)];
        etapa = `${pantalla}_${idioma}_${vista.ancho}_${vista.zoom}_interaccion`;
        await comprobarInteraccion(pagina, pantalla);
        estados.push(await comprobarAccesibilidad(accesible, '#lectura', 'tras_interaccion', vista.ancho, vista.zoom));
        const guardas = await pagina.evaluate(() => ({ almacenamiento: window.__almacenamiento, escrituras: window.__fixture.escrituras }));
        guardas.cookies = (await contexto.cookies()).length;
        const correcto = estados.every(e => e.estado === 'COMPROBADO') && !errores.length
          && Object.values(guardas).every(v => v === 0) && !red.bloqueadas && !red.websocket && !red.paginas_extra;
        resultados.push({ pantalla, idioma, ...vista, estado: correcto ? 'COMPROBADO' : 'CORTADO', estados, guardas, red, errores });
        await pagina.screenshot({ path: path.join(salida, `${pantalla}-${idioma}-${vista.ancho}-${vista.zoom}.png`), fullPage: true });
        await pagina.evaluate(() => window.__vista.desmontar());
      } finally { await contexto.close(); }
    }
  } catch {
    informe.error = 'recorrido_incompleto'; informe.etapa = etapa;
  } finally {
    try { await chrome?.cerrar(); }
    finally { await fs.rm(scratch, { recursive: true, force: true }); }
    informe.estado = !informe.error && resultados.length === casos.idiomas.length * casos.vistas.length * casos.pantallas.length
      && resultados.every(r => r.estado === 'COMPROBADO') ? 'COMPROBADO' : 'CORTADO';
    await fs.writeFile(path.join(salida, 'informe.json'), `${JSON.stringify(informe, null, 2)}\n`, { mode: 0o600 });
  }
  return informe;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const opciones = argumentos(process.argv.slice(2));
    const casos = await plan();
    if (opciones.plan) process.stdout.write(`${JSON.stringify(casos, null, 2)}\n`);
    else {
      const informe = await recorrer(opciones.salida, casos);
      process.stdout.write(`${JSON.stringify({ estado: informe.estado, visitas: informe.resultados.length, e2e: false })}\n`);
      if (informe.estado !== 'COMPROBADO') process.exitCode = 1;
    }
  } catch {
    const casos = await plan();
    process.stderr.write(`${casos.textos[casos.idiomas[0]].error}\n`); process.exitCode = 1;
  }
}
