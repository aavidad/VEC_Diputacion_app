#!/usr/bin/env python3
"""Genera el HTML imprimible del informe del Comité de Seguridad."""

from pathlib import Path

from markdown_it import MarkdownIt


DIRECTORIO = Path(__file__).resolve().parent
ORIGEN = DIRECTORIO / "informe_validacion_arquitectura_seguridad.md"
ESTILO = DIRECTORIO / "estilo_informe.css"
DESTINO = DIRECTORIO / "informe_validacion_arquitectura_seguridad.html"


def main() -> None:
    motor = MarkdownIt(
        "commonmark",
        {"html": True, "linkify": True, "typographer": True},
    ).enable("table")
    contenido = motor.render(ORIGEN.read_text(encoding="utf-8"))
    css = ESTILO.read_text(encoding="utf-8")
    documento = f"""<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Validación de arquitectura y seguridad del Portal VEC</title>
  <style>{css}</style>
</head>
<body>
{contenido}
<p class="pie-documento">Portal VEC · Informe para revisión del Comité de Seguridad · Fecha de corte: 15 de julio de 2026</p>
</body>
</html>
"""
    DESTINO.write_text(documento, encoding="utf-8")
    print(DESTINO)


if __name__ == "__main__":
    main()
