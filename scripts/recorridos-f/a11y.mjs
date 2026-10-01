import fs from 'node:fs';
import path from 'node:path';

const CONTROLES = 'a[href],button,input:not([type="hidden"]),select,textarea,[tabindex],[contenteditable="true"],[role="button"],[role="checkbox"],[role="combobox"],[role="radio"],[role="switch"],[role="tab"]';

// Sólo la página interna usa este perfil temporal. Los contextos de lectura
// siguen aislados, con sus certificados y las mismas guardas de transporte.
export async function chromeAccesible(chromium, salida) {
  const perfil = fs.mkdtempSync(path.join(salida, '.chrome-a11y-'));
  fs.chmodSync(perfil, 0o700);
  let interno;
  const cerrar = async () => {
    try { await interno?.close(); }
    finally { fs.rmSync(perfil, { recursive: true, force: true }); }
  };
  try {
    interno = await chromium.launchPersistentContext(perfil, { executablePath: '/usr/bin/google-chrome', headless: true, serviceWorkers: 'block' });
    await interno.route('**/*', route => route.abort());
    await interno.routeWebSocket('**/*', route => route.close());
    const ajustes = interno.pages()[0] || await interno.newPage();
    await ajustes.goto('chrome://settings/appearance', { waitUntil: 'domcontentloaded', timeout: 20000 });
    return { browser: interno.browser(), cerrar, zoom: async factor => {
      if (![1, 2].includes(factor)) throw new Error('zoom_nativo');
      const real = await ajustes.evaluate(factor => new Promise(resolve => {
        chrome.settingsPrivate.setDefaultZoom(factor, () => {
          if (chrome.runtime.lastError) { resolve(null); return; }
          chrome.settingsPrivate.getDefaultZoom(resolve);
        });
      }), factor);
      if (real !== factor) throw new Error('zoom_nativo');
    } };
  } catch (e) { await cerrar(); throw e; }
}

export function verificarZoom(m, ancho, factor) {
  if (![m.ancho_css, m.dpr, m.escala_visual].every(Number.isFinite) || Math.abs(m.ancho_css - ancho / factor) > 1 || Math.abs(m.dpr - factor) > 0.01
      || Math.abs(m.escala_visual - 1) > 0.01 || Number(m.zoom_css) !== 1) throw new Error('zoom_nativo');
}

export async function comprobarAccesibilidad(page, selector, lectura, ancho, factor) {
  const r = { lectura, estado: 'EN_CURSO', zoom_porcentaje: factor * 100 };
  const zoom = await page.evaluate(() => ({ ancho_css: innerWidth, dpr: devicePixelRatio,
    escala_visual: visualViewport.scale, zoom_css: getComputedStyle(document.documentElement).zoom }));
  Object.assign(r, zoom);
  verificarZoom(zoom, ancho, factor);
  const scope = page.locator(selector).first();
  await scope.waitFor({ state: 'visible', timeout: 10000 });
  const visibles = await scope.locator(CONTROLES).evaluateAll(elements => elements.map((el, indice) => {
    const css = getComputedStyle(el);
    const visible = el.getClientRects().length > 0 && css.visibility === 'visible' && css.display !== 'none';
    const excluido = Boolean(el.closest('[inert],[aria-hidden="true"]'));
    const habilitado = !el.matches(':disabled') && el.getAttribute('aria-disabled') !== 'true';
    const etiqueta = el.matches('input:not([type="hidden"]):not([type="button"]):not([type="submit"]):not([type="reset"]),select,textarea');
    const labels = [...(el.labels || []), ...(el.getAttribute('aria-labelledby') || '').split(/\s+/).map(id => document.getElementById(id)).filter(Boolean)];
    const etiquetaVisible = !etiqueta || labels.some(label => {
      const s = getComputedStyle(label), box = label.getBoundingClientRect();
      for (let actual = label; actual; actual = actual.parentElement) {
        if (Number(getComputedStyle(actual).opacity) <= 0) return false;
      }
      return label.textContent.trim() && box.width > 0 && box.height > 0 && s.display !== 'none' && s.visibility === 'visible'
        && s.clipPath !== 'inset(50%)' && s.clip === 'auto';
    });
    const interactivo = el.matches('a[href],button,input,select,textarea,[contenteditable="true"],[role="button"],[role="checkbox"],[role="combobox"],[role="radio"],[role="switch"],[role="tab"]');
    const radios = el.matches('input[type="radio"]') && el.name ? elements.filter(v => v.matches('input[type="radio"]') && v.name === el.name && v.form === el.form && !v.disabled) : [];
    const entradaRadio = !radios.length || (radios.find(v => v.checked) || radios[0]) === el;
    return { indice, visible, excluido, habilitado, etiquetaVisible, interactivo, entradaRadio, tab: el.tabIndex };
  }));
  const objetivo = visibles.filter(v => v.visible && v.habilitado && (v.interactivo || v.tab >= 0));
  r.controles_visibles = objetivo.length;
  r.etiquetas_ausentes = objetivo.filter(v => !v.etiquetaVisible).length;
  r.controles_ocultos_a_lector = objetivo.filter(v => v.excluido).length;
  r.controles_fuera_de_tab = objetivo.filter(v => !v.excluido && v.tab < 0).length;
  r.nombres_ausentes = 0;
  const cdp = await page.context().newCDPSession(page);
  try {
    const doc = await cdp.send('DOM.getDocument');
    const { nodeId } = await cdp.send('DOM.querySelector', { nodeId: doc.root.nodeId, selector });
    const { nodeIds } = await cdp.send('DOM.querySelectorAll', { nodeId, selector: CONTROLES });
    const { nodes } = await cdp.send('Accessibility.getFullAXTree');
    const nombres = new Map(nodes.filter(n => !n.ignored).map(n => [n.backendDOMNodeId, Boolean(n.name?.value?.trim())]));
    for (const v of objetivo) {
      const { node } = await cdp.send('DOM.describeNode', { nodeId: nodeIds[v.indice] });
      if (!nombres.get(node.backendNodeId)) r.nombres_ausentes++;
    }
  } finally { await cdp.detach(); }

  r.teclado_comprobados = 0; r.foco_invisible = 0; r.foco_tapado = 0; r.trampas_teclado = 0;
  const pendientes = new Set(objetivo.filter(v => !v.excluido && v.tab >= 0 && v.entradaRadio).map(v => v.indice));
  // Tab recorre controles reales; no se activan botones ni se rellenan campos.
  // Acotado: una pantalla con más de 256 pasos exige revisión manual.
  for (let paso = 0; pendientes.size && paso < 256; paso++) {
    await page.keyboard.press('Tab');
    const foco = await scope.locator(CONTROLES).evaluateAll(elements => {
      const el = document.activeElement, indice = elements.indexOf(el);
      if (indice < 0) return { indice };
      const css = getComputedStyle(el), box = el.getBoundingClientRect();
      const transparente = color => color === 'transparent' || /rgba\([^)]*,\s*0\)$/.test(color);
      const anillo = parseFloat(css.outlineWidth) > 0 && css.outlineStyle !== 'none' && !transparente(css.outlineColor);
      // Las esquinas del rectángulo quedan fuera de un botón redondeado.
      const cx = (box.left + box.right) / 2, cy = (box.top + box.bottom) / 2;
      const dx = Math.min(1, box.width / 4), dy = Math.min(1, box.height / 4);
      const puntos = [[cx, cy], [box.left + dx, cy], [box.right - dx, cy], [cx, box.top + dy], [cx, box.bottom - dy]];
      const libre = box.width > 0 && box.height > 0 && puntos.every(([x, y]) => {
        if (x < 0 || y < 0 || x >= innerWidth || y >= innerHeight) return false;
        const top = document.elementFromPoint(x, y);
        return top === el || el.contains(top);
      });
      return { indice, visible: el.matches(':focus-visible'), anillo,
        contorno: [css.outlineWidth, css.outlineStyle, css.outlineColor, css.outlineOffset].join('|'),
        sombra: css.boxShadow, borde: css.borderColor, libre };
    });
    if (!pendientes.has(foco.indice)) continue;
    pendientes.delete(foco.indice); r.teclado_comprobados++;
    if (!foco.libre) r.foco_tapado++;
    const control = scope.locator(CONTROLES).nth(foco.indice);
    const sigueEnControl = () => control.evaluate(el => document.activeElement === el && document.hasFocus());
    const regresar = async tecla => {
      await page.keyboard.press(tecla);
      // Chrome puede pasar por su barra al cruzar el inicio o final del Tab.
      // Sólo se permite otro paso si el documento ha perdido el foco.
      if (!(await page.evaluate(() => document.hasFocus()))) await page.keyboard.press(tecla);
      return sigueEnControl();
    };
    // Debe abandonar el control, además de volver a él: un Tab bloqueado
    // en el último control no se detecta sólo comparando el retorno.
    await page.keyboard.press('Shift+Tab');
    if (await sigueEnControl()) { r.trampas_teclado++; break; }
    const base = await control.evaluate(el => {
      const css = getComputedStyle(el);
      return { contorno: [css.outlineWidth, css.outlineStyle, css.outlineColor, css.outlineOffset].join('|'),
        sombra: css.boxShadow, borde: css.borderColor };
    });
    if (!foco.visible || !((foco.anillo && foco.contorno !== base.contorno)
        || (foco.sombra !== 'none' && foco.sombra !== base.sombra) || foco.borde !== base.borde)) r.foco_invisible++;
    if (!(await regresar('Tab'))) { r.trampas_teclado++; break; }
    await page.keyboard.press('Tab');
    if (await sigueEnControl()) { r.trampas_teclado++; break; }
    if (!(await regresar('Shift+Tab'))) { r.trampas_teclado++; break; }
  }
  r.teclado_no_alcanzados = pendientes.size;
  r.desbordamiento = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
  r.estado = [r.etiquetas_ausentes, r.nombres_ausentes, r.controles_ocultos_a_lector, r.controles_fuera_de_tab,
    r.teclado_no_alcanzados, r.trampas_teclado, r.foco_invisible, r.foco_tapado, r.desbordamiento].some(Boolean) ? 'CORTADO' : 'COMPROBADO';
  return r;
}

// Sólo se usa con los dos controles de apertura ya presentes en lectura v1.
export async function activarLectura(page, boton) {
  for (let i = 0; i < 256; i++) {
    if (await boton.evaluate(el => document.activeElement === el)) { await page.keyboard.press('Enter'); return; }
    await page.keyboard.press('Tab');
  }
  throw new Error('teclado_accesibilidad');
}
