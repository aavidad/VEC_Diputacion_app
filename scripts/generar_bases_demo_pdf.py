#!/usr/bin/env python3
"""Genera los PDF sanitizados de bases DEMO desde su HTML accesible.

El HTML es la fuente mantenible. Chromium/Playwright conserva etiquetas y
marcadores; qpdf normaliza fechas e identificador para hacer reproducible el
artefacto. Los PDF solo pertenecen al perfil aislado de presentación: sus
nombres incluyen ``demo`` y la imagen de producción los elimina.
"""

from __future__ import annotations

import argparse
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
from typing import Final, NamedTuple, Sequence


RAIZ: Final = Path(__file__).resolve().parents[1]
DIRECTORIO_DOCUMENTOS: Final = RAIZ / "web" / "static" / "bolsa" / "documentos"
FECHA_PDF_FIJA: Final = b"D:20260719000000+00'00'"
PATRONES_PROHIBIDOS: Final = (
    b"/JavaScript",
    b"/Launch",
    b"/EmbeddedFile",
    b"/RichMedia",
    b"/AcroForm",
    b"/XFA",
    b"/SubmitForm",
    b"/URI",
    b"/home/",
)


class Documento(NamedTuple):
    clave: str
    html: Path
    pdf: Path
    cve: str


DOCUMENTOS: Final = (
    Documento(
        "auxiliar",
        DIRECTORIO_DOCUMENTOS / "bases-auxiliar-demo.html",
        DIRECTORIO_DOCUMENTOS / "bases-auxiliar-demo.pdf",
        "BOP-GRA-2025-125002",
    ),
    Documento(
        "gestion",
        DIRECTORIO_DOCUMENTOS / "bases-gestion-demo.html",
        DIRECTORIO_DOCUMENTOS / "bases-gestion-demo.pdf",
        "BOP-GRA-2024-244002",
    ),
    Documento(
        "operario",
        DIRECTORIO_DOCUMENTOS / "bases-operario-demo.html",
        DIRECTORIO_DOCUMENTOS / "bases-operario-demo.pdf",
        "BOP-GRA-2026-043004",
    ),
)


def ejecutar(comando: Sequence[str]) -> subprocess.CompletedProcess[bytes]:
    """Ejecuta una herramienta local sin shell y conserva su diagnóstico."""

    return subprocess.run(comando, check=True, capture_output=True)


def navegador_configurado() -> str | None:
    """Devuelve Chromium explícito o un navegador local conocido."""

    indicado = os.environ.get("VEC_CHROMIUM_EXECUTABLE", "").strip()
    if indicado:
        ruta = Path(indicado).expanduser().absolute()
        if not ruta.is_file():
            raise RuntimeError(f"VEC_CHROMIUM_EXECUTABLE no existe: {ruta}")
        return str(ruta)
    # Playwright puede estar instalado sin su headless-shell. Los binarios del
    # sistema son un reemplazo válido y evitan una descarga implícita de red.
    for nombre in ("google-chrome-stable", "google-chrome", "chromium", "chromium-browser"):
        encontrado = shutil.which(nombre)
        if encontrado:
            return encontrado
    return None


def normalizar_pdf(origen: Path, destino: Path, temporal: Path) -> None:
    """Normaliza metadatos variables y genera un identificador determinista."""

    qpdf = shutil.which("qpdf")
    if not qpdf:
        raise RuntimeError("qpdf es obligatorio para normalizar y validar los PDF DEMO")

    qdf = temporal / f"{destino.stem}.qdf.pdf"
    normalizado = temporal / f"{destino.stem}.normalizado.pdf"
    ejecutar(
        [
            qpdf,
            "--qdf",
            "--object-streams=disable",
            "--stream-data=uncompress",
            str(origen),
            str(qdf),
        ]
    )
    contenido = qdf.read_bytes()
    contenido, sustituciones_creacion = re.subn(
        rb"(?<=/CreationDate \()D:[^)]*(?=\))", FECHA_PDF_FIJA, contenido
    )
    contenido, sustituciones_modificacion = re.subn(
        rb"(?<=/ModDate \()D:[^)]*(?=\))", FECHA_PDF_FIJA, contenido
    )
    if sustituciones_creacion == 0:
        raise RuntimeError(f"{destino.name}: no se encontró CreationDate para normalizar")
    # Chromium no siempre emite ModDate. Si existe, queda normalizado.
    _ = sustituciones_modificacion
    # El ID de entrada de Chromium incorpora entropía. Se neutraliza antes de
    # pedir a qpdf un ID derivado del contenido estático.
    contenido, sustituciones_id = re.subn(
        rb"/ID \[<([0-9A-Fa-f]+)><([0-9A-Fa-f]+)>\]",
        lambda coincidencia: (
            b"/ID [<" + b"0" * len(coincidencia.group(1)) + b"><"
            + b"0" * len(coincidencia.group(2)) + b">]"
        ),
        contenido,
    )
    if sustituciones_id == 0:
        raise RuntimeError(f"{destino.name}: no se encontró el ID de entrada para normalizar")
    qdf.write_bytes(contenido)

    ejecutar(
        [
            qpdf,
            "--warning-exit-0",
            "--deterministic-id",
            "--object-streams=generate",
            "--stream-data=compress",
            str(qdf),
            str(normalizado),
        ]
    )
    ejecutar([qpdf, "--check", str(normalizado)])

    inspeccion = temporal / f"{destino.stem}.inspeccion.pdf"
    ejecutar(
        [
            qpdf,
            "--qdf",
            "--object-streams=disable",
            "--stream-data=uncompress",
            str(normalizado),
            str(inspeccion),
        ]
    )
    plano = inspeccion.read_bytes()
    encontrados = [patron.decode("ascii") for patron in PATRONES_PROHIBIDOS if patron in plano]
    if encontrados:
        raise RuntimeError(f"{destino.name}: acciones o datos prohibidos: {', '.join(encontrados)}")

    datos = normalizado.read_bytes()
    if not datos.startswith(b"%PDF-") or not datos.rstrip().endswith(b"%%EOF"):
        raise RuntimeError(f"{destino.name}: artefacto PDF incompleto")
    destino.parent.mkdir(parents=True, exist_ok=True)
    # /tmp y el repositorio pueden estar en sistemas de archivos diferentes.
    # Copiamos primero dentro del directorio destino y renombramos allí.
    with tempfile.NamedTemporaryFile(
        prefix=f".{destino.name}.", suffix=".tmp", dir=destino.parent, delete=False
    ) as fichero_temporal:
        ruta_temporal_destino = Path(fichero_temporal.name)
        fichero_temporal.write(datos)
        fichero_temporal.flush()
        os.fsync(fichero_temporal.fileno())
    os.replace(ruta_temporal_destino, destino)


def generar(documentos: Sequence[Documento], ejecutable: str | None) -> list[tuple[Documento, str]]:
    """Genera los documentos elegidos y devuelve sus huellas SHA-256."""

    try:
        from playwright.sync_api import sync_playwright  # type: ignore[import-not-found]
    except ImportError as error:
        raise RuntimeError(
            "Falta Playwright para Python. Instale Playwright y su navegador Chromium."
        ) from error

    for documento in documentos:
        if not documento.html.is_file():
            raise RuntimeError(f"no existe la fuente HTML: {documento.html}")
        html = documento.html.read_text(encoding="utf-8")
        for esperado in (
            '<html lang="es">',
            "DEMOSTRACIÓN · SIN VALIDEZ ADMINISTRATIVA",
            documento.cve,
            "bases-demo.css",
        ):
            if esperado not in html:
                raise RuntimeError(f"{documento.html.name}: falta {esperado!r}")

    resultados: list[tuple[Documento, str]] = []
    with tempfile.TemporaryDirectory(prefix="vec-bases-demo-") as nombre_temporal:
        temporal = Path(nombre_temporal)
        with sync_playwright() as playwright:
            opciones: dict[str, object] = {"headless": True}
            if ejecutable:
                opciones["executable_path"] = ejecutable
            browser = playwright.chromium.launch(**opciones)
            try:
                contexto = browser.new_context(
                    locale="es-ES",
                    timezone_id="Europe/Madrid",
                    viewport={"width": 1440, "height": 1000},
                )
                pagina = contexto.new_page()
                for documento in documentos:
                    pagina.goto(documento.html.as_uri(), wait_until="networkidle")
                    pagina.emulate_media(media="print")
                    pagina.evaluate("document.fonts.ready")
                    # Un PDF DEMO es deliberadamente pasivo: conserva el texto
                    # visible de las fuentes, pero no acciones ni enlaces activos.
                    pagina.eval_on_selector_all(
                        "a",
                        "elementos => elementos.forEach(elemento => elemento.removeAttribute('href'))",
                    )
                    pagina.eval_on_selector_all(
                        ".marca-pdf-repetida, .pie-pdf-repetido",
                        "elementos => elementos.forEach(elemento => elemento.remove())",
                    )
                    bruto = temporal / f"{documento.pdf.stem}.bruto.pdf"
                    pagina.pdf(
                        path=str(bruto),
                        format="A4",
                        print_background=True,
                        tagged=True,
                        outline=True,
                        prefer_css_page_size=True,
                        display_header_footer=True,
                        header_template=(
                            '<div style="box-sizing:border-box;color:#8a1538;font-family:Arial,sans-serif;'
                            'font-size:8px;font-weight:700;letter-spacing:.5px;text-align:center;'
                            'width:100%">DEMOSTRACIÓN · SIN VALIDEZ ADMINISTRATIVA</div>'
                        ),
                        footer_template=(
                            '<div style="box-sizing:border-box;color:#56616d;font-family:Arial,sans-serif;'
                            'font-size:7px;text-align:center;width:100%">Adaptación sanitizada · Fuente pública '
                            + documento.cve
                            + ' · Página <span class="pageNumber"></span> de '
                            '<span class="totalPages"></span></div>'
                        ),
                    )
                    normalizar_pdf(bruto, documento.pdf, temporal)
                    huella = hashlib.sha256(documento.pdf.read_bytes()).hexdigest()
                    resultados.append((documento, huella))
                contexto.close()
            finally:
                browser.close()
    return resultados


def argumentos(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "documentos",
        nargs="*",
        choices=[documento.clave for documento in DOCUMENTOS],
        help="Subconjunto que se generará; por defecto, todos.",
    )
    parser.add_argument(
        "--ejecutable-navegador",
        type=Path,
        help="Ruta opcional de Chromium; prevalece sobre VEC_CHROMIUM_EXECUTABLE.",
    )
    return parser.parse_args(argv)


def main(argv: Sequence[str] | None = None) -> int:
    args = argumentos(argv)
    elegidos = [
        documento for documento in DOCUMENTOS
        if not args.documentos or documento.clave in set(args.documentos)
    ]
    ejecutable = navegador_configurado()
    if args.ejecutable_navegador:
        ruta = args.ejecutable_navegador.expanduser().absolute()
        if not ruta.is_file():
            raise RuntimeError(f"no existe el navegador: {ruta}")
        ejecutable = str(ruta)
    for documento, huella in generar(elegidos, ejecutable):
        print(f"{documento.pdf.relative_to(RAIZ)}  sha256={huella}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (RuntimeError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        raise SystemExit(1)
