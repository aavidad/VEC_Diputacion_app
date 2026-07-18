from __future__ import annotations

from html.parser import HTMLParser
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest


RAIZ = Path(__file__).resolve().parents[2]
DIRECTORIO = RAIZ / "web" / "static" / "bolsa" / "documentos"
DOCUMENTOS = (
    ("bases-auxiliar-demo", "BOP-GRA-2025-125002", "Auxiliar de Servicios Generales"),
    ("bases-gestion-demo", "BOP-GRA-2024-244002", "Subescala de Gestión"),
    ("bases-operario-demo", "BOP-GRA-2026-043004", "bolsa de empleo de Operario"),
)
MARCA = "DEMOSTRACIÓN · SIN VALIDEZ ADMINISTRATIVA"


class EstructuraHTML(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.etiquetas: list[str] = []
        self.atributos: list[tuple[str, dict[str, str | None]]] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        self.etiquetas.append(tag)
        self.atributos.append((tag, dict(attrs)))


def ejecutar(*comando: str) -> str:
    resultado = subprocess.run(comando, check=True, capture_output=True, text=True)
    return resultado.stdout + resultado.stderr


class TestFuentesHTMLBasesDemo(unittest.TestCase):
    def test_fuentes_son_semanticas_comunes_y_sanitizadas(self) -> None:
        for base, cve, titulo in DOCUMENTOS:
            with self.subTest(documento=base):
                ruta = DIRECTORIO / f"{base}.html"
                contenido = ruta.read_text(encoding="utf-8")
                estructura = EstructuraHTML()
                estructura.feed(contenido)

                self.assertIn('<html lang="es">', contenido)
                self.assertIn(MARCA, contenido)
                self.assertIn(cve, contenido)
                self.assertIn(titulo, contenido)
                for etiqueta in ("main", "article", "header", "nav", "h1", "h2", "table", "section", "footer"):
                    self.assertIn(etiqueta, estructura.etiquetas)
                hojas = [
                    atributos.get("href", "") or ""
                    for etiqueta, atributos in estructura.atributos
                    if etiqueta == "link" and atributos.get("rel") == "stylesheet"
                ]
                self.assertTrue(any(hoja.endswith("bases-demo.css?v=20260719-bases-v1") for hoja in hojas))
                self.assertNotRegex(contenido.lower(), r"<style\b|\sstyle\s*=")
                self.assertNotRegex(contenido, r"\bES\d{22}\b")
                self.assertNotRegex(contenido, r"(?<!\d)[6789]\d{8}(?!\d)")
                self.assertNotIn("Firmado por:", contenido)


@unittest.skipUnless(
    all(shutil.which(utilidad) for utilidad in ("qpdf", "pdfinfo", "pdftotext", "pdffonts")),
    "se requieren qpdf, pdfinfo, pdftotext y pdffonts",
)
class TestArtefactosPDFBasesDemo(unittest.TestCase):
    def test_pdf_a4_etiquetados_pasivos_y_con_marca_en_cada_pagina(self) -> None:
        for base, cve, titulo in DOCUMENTOS:
            with self.subTest(documento=base):
                ruta = DIRECTORIO / f"{base}.pdf"
                datos = ruta.read_bytes()
                self.assertGreater(len(datos), 10_000)
                self.assertTrue(datos.startswith(b"%PDF-"))
                self.assertTrue(datos.rstrip().endswith(b"%%EOF"))
                ejecutar("qpdf", "--check", str(ruta))

                informacion = ejecutar("pdfinfo", str(ruta))
                self.assertRegex(informacion, r"(?m)^Tagged:\s+yes$")
                self.assertRegex(informacion, r"(?m)^Encrypted:\s+no$")
                self.assertRegex(informacion, r"(?m)^JavaScript:\s+no$")
                self.assertRegex(informacion, r"(?m)^Form:\s+none$")
                self.assertRegex(informacion, r"(?m)^Page size:.*\(A4\)$")
                paginas = int(re.search(r"(?m)^Pages:\s+(\d+)$", informacion).group(1))

                texto = ejecutar("pdftotext", str(ruta), "-")
                self.assertIn(cve, texto)
                self.assertIn(titulo, re.sub(r"\s+", " ", texto))
                for pagina in range(1, paginas + 1):
                    texto_pagina = ejecutar(
                        "pdftotext", "-f", str(pagina), "-l", str(pagina), str(ruta), "-"
                    )
                    self.assertIn(MARCA, texto_pagina, f"falta la marca en página {pagina}")

                fuentes = ejecutar("pdffonts", str(ruta)).splitlines()[2:]
                self.assertTrue(fuentes)
                for fuente in fuentes:
                    self.assertRegex(fuente, r"\s+yes\s+yes\s+yes\s+\d+\s+\d+\s*$")

                with tempfile.TemporaryDirectory(prefix="vec-prueba-pdf-") as temporal:
                    qdf = Path(temporal) / "inspeccion.pdf"
                    ejecutar(
                        "qpdf", "--qdf", "--object-streams=disable", "--stream-data=uncompress",
                        str(ruta), str(qdf),
                    )
                    plano = qdf.read_bytes()
                    for patron in (
                        b"/JavaScript", b"/Launch", b"/EmbeddedFile", b"/RichMedia",
                        b"/AcroForm", b"/XFA", b"/SubmitForm", b"/URI", b"/home/",
                    ):
                        self.assertNotIn(patron, plano)
                    self.assertIn(b"D:20260719000000+00'00'", plano)


if __name__ == "__main__":
    unittest.main()
