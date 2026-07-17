#!/usr/bin/env python3
"""Genera el PDF conjunto del paquete de cumplimiento con Chrome headless."""
import html
import pathlib
import re
import subprocess

AQUI = pathlib.Path(__file__).parent
DOCS = ["LEEME.md", "ens_categorizacion_y_aplicabilidad.md",
        "rat_registro_actividades_tratamiento.md",
        "eipd_evaluacion_impacto_bolsas.md"]


def tr(linea):
    celdas = [c.strip() for c in linea.strip().strip("|").split("|")]
    return celdas


def md_a_html(texto):
    salida, lista, tabla = [], False, []
    for linea in texto.split("\n"):
        if linea.startswith("|"):
            tabla.append(tr(linea)); continue
        if tabla:
            filas = [f for f in tabla if not all(re.match(r"^[-: ]*$", c) for c in f)]
            salida.append("<table><tr>" + "".join(f"<th>{p(c)}</th>" for c in filas[0]) + "</tr>" +
                          "".join("<tr>" + "".join(f"<td>{p(c)}</td>" for c in f) + "</tr>" for f in filas[1:]) + "</table>")
            tabla = []
        if linea.startswith("- "):
            if not lista: salida.append("<ul>"); lista = True
            salida.append(f"<li>{p(linea[2:])}</li>"); continue
        if lista and not linea.startswith(("- ", "  ")):
            salida.append("</ul>"); lista = False
        if linea.startswith("### "): salida.append(f"<h3>{p(linea[4:])}</h3>")
        elif linea.startswith("## "): salida.append(f"<h2>{p(linea[3:])}</h2>")
        elif linea.startswith("# "): salida.append(f"<h1>{p(linea[2:])}</h1>")
        elif linea.strip() == "---": salida.append("<hr>")
        elif linea.strip() == "": salida.append("</p><p>")
        elif lista and linea.startswith("  "): salida[-1] = salida[-1][:-5] + " " + p(linea.strip()) + "</li>"
        else: salida.append(p(linea))
    if lista: salida.append("</ul>")
    return "<p>" + "\n".join(salida) + "</p>"


def p(t):
    t = html.escape(t)
    t = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", t)
    t = re.sub(r"`([^`]+)`", r"<code>\1</code>", t)
    t = re.sub(r"\[([^]]+)\]\(([^)]+)\)", r"\1", t)
    return t


cuerpo = '<div style="page-break-after: always"></div>'.join(
    md_a_html((AQUI / d).read_text()) for d in DOCS)
pagina = ('<!doctype html><html lang="es"><head><meta charset="utf-8">'
          "<title>Paquete de cumplimiento VEC</title><style>"
          "body{font-family:'Segoe UI',system-ui,sans-serif;color:#1a2733;max-width:200mm;"
          "margin:0 auto;line-height:1.5;font-size:10.5pt}"
          "h1{color:#0d2d52;border-bottom:3px solid #0d2d52;padding-bottom:6px;font-size:17pt}"
          "h2{color:#0d2d52;margin-top:22px;border-bottom:1px solid #c9d6e4;font-size:13pt}"
          "h3{color:#17466e;font-size:11.5pt}"
          "table{border-collapse:collapse;width:100%;margin:10px 0;font-size:9.5pt;page-break-inside:avoid}"
          "th,td{border:1px solid #c9d6e4;padding:5px 7px;text-align:left;vertical-align:top}"
          "th{background:#eef3f8}"
          "code{background:#eef3f8;padding:1px 4px;border-radius:3px;font-size:9pt}"
          "</style></head><body>" + cuerpo + "</body></html>")
(AQUI / "paquete_validacion_cumplimiento.html").write_text(pagina)
subprocess.run(["google-chrome", "--headless", "--disable-gpu", "--no-pdf-header-footer",
                f"--print-to-pdf={AQUI}/paquete_validacion_cumplimiento.pdf",
                str(AQUI / "paquete_validacion_cumplimiento.html")], check=True, capture_output=True)
(AQUI / "paquete_validacion_cumplimiento.html").unlink()
print("PDF generado")
