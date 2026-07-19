"""Resumen y serialización JSON/Markdown de la revisión web."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Sequence


def resumir_resultados(resultados: Sequence[dict[str, Any]]) -> dict[str, Any]:
    escenarios_correctos = sum(1 for resultado in resultados if resultado["correcto"])
    return {
        "escenarios": len(resultados),
        "correctos": escenarios_correctos,
        "con_hallazgos": len(resultados) - escenarios_correctos,
        "hallazgos": sum(len(resultado["hallazgos"]) for resultado in resultados),
        "capturas": sum(1 for resultado in resultados if resultado.get("captura")),
        "vistas": sum(1 for resultado in resultados if resultado["tipo"] == "vista"),
        "flujos": sum(1 for resultado in resultados if resultado["tipo"] == "flujo"),
    }


def codigo_salida(resultados: Sequence[dict[str, Any]], tolerante: bool) -> int:
    """El modo tolerante informa hallazgos pero no falla la invocación."""
    if tolerante:
        return 0
    return 1 if any(not resultado["correcto"] for resultado in resultados) else 0


def _celda_markdown(valor: Any) -> str:
    return str(valor).replace("|", "\\|").replace("\n", " ")


def crear_informe_markdown(informe: dict[str, Any]) -> str:
    resumen = informe["resumen"]
    modo = "tolerante" if informe["tolerante"] else "estricto"
    estado = "CORRECTO" if informe["correcto"] else "CON HALLAZGOS"
    lineas = [
        "# Revisión de la presentación web", "",
        f"**Estado:** {estado} · **Modo:** {modo}", "",
        f"- URL base: `{informe['url_base']}`",
        f"- Generado: `{informe['generado_en']}`",
        f"- Escenarios: {resumen['escenarios']} ({resumen['vistas']} vistas, {resumen['flujos']} flujos)",
        f"- Correctos: {resumen['correctos']}",
        f"- Con hallazgos: {resumen['con_hallazgos']}",
        f"- Capturas: {resumen['capturas']}", "",
        "## Cobertura y capturas", "",
        "| Tipo | Superficie | Escenario | Tamaño | Estado | Captura |",
        "|---|---|---|---:|---|---|",
    ]
    for resultado in informe["resultados"]:
        if resultado.get("captura"):
            partes = 1 + len(resultado.get("capturas_adicionales", []))
            sufijo = f" ({partes} partes)" if partes > 1 else ""
            captura = f"[ver]({resultado['captura']}){sufijo}"
        else:
            captura = "—"
        lineas.append(
            f"| {_celda_markdown(resultado['tipo'])} | {_celda_markdown(resultado['nombre_superficie'])} "
            f"| {_celda_markdown(resultado['nombre'])} | {resultado['tamano']['ancho']}×{resultado['tamano']['alto']} "
            f"| {'OK' if resultado['correcto'] else 'ERROR'} | {captura} |"
        )

    lineas.extend(["", "## Hallazgos", ""])
    con_hallazgos = [resultado for resultado in informe["resultados"] if resultado["hallazgos"]]
    if not con_hallazgos:
        lineas.append("No se detectaron hallazgos.")
    else:
        lineas.extend([
            "| Escenario | Tamaño | Código | Descripción |",
            "|---|---:|---|---|",
        ])
        for resultado in con_hallazgos:
            for hallazgo in resultado["hallazgos"]:
                lineas.append(
                    f"| {_celda_markdown(resultado['nombre'])} | {resultado['tamano']['ancho']}×{resultado['tamano']['alto']} "
                    f"| `{_celda_markdown(hallazgo['codigo'])}` | {_celda_markdown(hallazgo['mensaje'])} |"
                )
    lineas.extend([
        "", "> Cada escenario se abrió en un contexto limpio. No se reutilizaron cookies, localStorage ni sessionStorage.", "",
    ])
    return "\n".join(lineas)


def guardar_informes(informe: dict[str, Any], directorio_salida: Path) -> tuple[Path, Path]:
    directorio_salida.mkdir(parents=True, exist_ok=True)
    ruta_json = directorio_salida / "resultados.json"
    ruta_markdown = directorio_salida / "informe.md"
    temporal_json = ruta_json.with_suffix(".json.tmp")
    temporal_markdown = ruta_markdown.with_suffix(".md.tmp")
    temporal_json.write_text(json.dumps(informe, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    temporal_markdown.write_text(crear_informe_markdown(informe), encoding="utf-8")
    temporal_json.replace(ruta_json)
    temporal_markdown.replace(ruta_markdown)
    return ruta_json, ruta_markdown
