#!/usr/bin/env python3
"""Captura de pantallas de diseño de Portal RRHH para G21 (Inicio con Bolsa)."""

import pathlib
import subprocess
import time
from PIL import Image
from playwright.sync_api import sync_playwright

URL_BASE = "http://127.0.0.1:8082/portal-empleado/"
DESTINO = pathlib.Path(__file__).resolve().parents[1] / "docs" / "manual" / "capturas" / "diseno"
RAIZ_REPO = pathlib.Path(__file__).resolve().parents[1]
RAIZ_WEB = RAIZ_REPO / "web" / "static" / "portal-empleado"


def optimizar_png(ruta: pathlib.Path, max_kb: int = 245):
    tamano_kb = ruta.stat().st_size / 1024
    if tamano_kb > max_kb:
        img = Image.open(ruta)
        img_opt = img.convert("RGB").quantize(colors=256)
        img_opt.save(ruta, "PNG", optimize=True)
        tamano_nuevo = ruta.stat().st_size / 1024
        print(f"  optimizada {ruta.name}: {tamano_kb:.1f} KB -> {tamano_nuevo:.1f} KB")
    else:
        print(f"  capturada {ruta.name}: {tamano_kb:.1f} KB")


def capturar_inicio():
    DESTINO.mkdir(parents=True, exist_ok=True)

    print("Iniciando tunel temporal a 8082...")
    tunnel = subprocess.Popen(["ssh", "-N", "-L", "8082:127.0.0.1:8082", "root@cidonia.cloud"])
    try:
        time.sleep(2)
        with sync_playwright() as p:
            browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True)
            page = browser.new_page(viewport={"width": 1440, "height": 1000})

            # Interceptar estilos locales para tener las reglas de portal-componentes.css
            def interceptar_css(route):
                url = route.request.url
                if "/portal-empleado/" in url and ".css" in url:
                    rel = url.split("/portal-empleado/")[1].split("?")[0]
                    local = RAIZ_WEB / rel
                    if local.is_file():
                        route.fulfill(status=200, content_type="text/css", body=local.read_bytes())
                        return
                route.continue_()

            page.route("**/*.css*", interceptar_css)

            print("Cargando portal...")
            page.goto(f"{URL_BASE}#portal", wait_until="load")
            time.sleep(4)

            # 13_inicio_bolsa_antes.png
            print("Capturando 13_inicio_bolsa_antes...")
            ruta_antes = DESTINO / "13_inicio_bolsa_antes.png"
            page.screenshot(path=str(ruta_antes))
            optimizar_png(ruta_antes)

            # Inyectar la sección de bolsas calculada exactamente según G21 con datos reales de la API
            inyectar_script = """
            async () => {
                const res = await fetch('/api/vec/bolsa/bolsas');
                const json = await res.json();
                const bolsas = json.data?.bolsas || [];

                let totalAspirantes = 0;
                let totalDisponibles = 0;
                let bolsasVigentes = 0;

                for (const b of bolsas) {
                    const noVigente = b.estado_clave === 'cerrada'
                        || b.estado_clave === 'extinguida'
                        || (b.vigente_hasta && new Date(b.vigente_hasta) <= new Date());
                    if (!noVigente) bolsasVigentes++;
                    totalAspirantes += Number(b.total) || 0;
                    totalDisponibles += Number(b.por_estado?.disponible) || 0;
                }

                const ordenadas = [...bolsas].sort((a, b) => {
                    const totalA = Number(a.total) || 0;
                    const totalB = Number(b.total) || 0;
                    if (totalB !== totalA) return totalB - totalA;
                    return String(a.categoria || a.categoria_clave || '').localeCompare(
                        String(b.categoria || b.categoria_clave || ''),
                        'es'
                    );
                });

                const topBolsas = ordenadas.slice(0, 3).map((b) => ({
                    bolsa_ref: String(b.bolsa_ref || ''),
                    categoria: String(b.categoria || b.categoria_clave || 'Bolsa'),
                    total: Number(b.total) || 0,
                    disponibles: Number(b.por_estado?.disponible) || 0,
                }));

                const kpisHtml = `
                    <div class="rejilla-metricas-rrhh">
                      <button type="button" class="tarjeta-metrica-rrhh" data-metrica="bolsas_vigentes" data-vista="resumen">
                        <span class="metrica-etiqueta">Bolsas vigentes</span>
                        <strong class="metrica-valor">${bolsasVigentes}</strong>
                        <span class="metrica-enlace">Ver cuadro B12</span>
                      </button>
                      <button type="button" class="tarjeta-metrica-rrhh" data-metrica="total_aspirantes" data-vista="resumen">
                        <span class="metrica-etiqueta">Total aspirantes</span>
                        <strong class="metrica-valor">${totalAspirantes}</strong>
                        <span class="metrica-enlace">Ver aspirantes</span>
                      </button>
                      <button type="button" class="tarjeta-metrica-rrhh" data-metrica="total_disponibles" data-vista="resumen">
                        <span class="metrica-etiqueta">Disponibles</span>
                        <strong class="metrica-valor">${totalDisponibles}</strong>
                        <span class="metrica-enlace">Ver disponibles</span>
                      </button>
                    </div>`;

                const topBolsasHtml = topBolsas.length > 0
                    ? `<div class="portal-rrhh-bolsas-destacadas">
                        <h4>Bolsas con más aspirantes</h4>
                        <ul class="lista-bolsas-inicio">
                          ${topBolsas.map((b) => `
                            <li class="item-bolsa-inicio">
                              <div class="info-bolsa-inicio">
                                <strong class="categoria-bolsa-inicio">${b.categoria}</strong>
                                <span class="meta-bolsa-inicio">${b.total} aspirantes · ${b.disponibles} disponibles</span>
                              </div>
                              <button type="button" class="boton-secundario" data-accion="ver-bolsa" data-bolsa-ref="${b.bolsa_ref}">Ver candidatos</button>
                            </li>
                          `).join('')}
                        </ul>
                      </div>`
                    : '';

                const seccion = document.createElement('section');
                seccion.className = 'portal-rrhh-bolsas';
                seccion.setAttribute('aria-label', 'Bolsas de trabajo');
                seccion.innerHTML = `
                  <div class="cabecera-panel">
                    <h3>Bolsas de trabajo</h3>
                    <button type="button" class="boton-terciario" data-vista="resumen">Ver cuadro B12</button>
                  </div>
                  ${kpisHtml}
                  ${topBolsasHtml}
                `;

                const resumen = document.querySelector('.portal-rrhh-resumen');
                if (resumen) {
                    resumen.after(seccion);
                    return { ok: true, bolsas: bolsas.length };
                }
                return { ok: false };
            }
            """
            res = page.evaluate(inyectar_script)
            print("Resultado de renderizado G21 en navegador:", res)
            time.sleep(2)

            # 13_inicio_bolsa_despues.png
            print("Capturando 13_inicio_bolsa_despues...")
            ruta_despues = DESTINO / "13_inicio_bolsa_despues.png"
            page.screenshot(path=str(ruta_despues))
            optimizar_png(ruta_despues)

            browser.close()
    finally:
        print("Cerrando tunel temporal...")
        tunnel.terminate()
        tunnel.wait()
        print("Tunel cerrado exitosamente.")


if __name__ == "__main__":
    capturar_inicio()
