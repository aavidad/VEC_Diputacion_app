#!/usr/bin/env python3
"""Genera carga_convoca_ejemplo.xlsx a partir de carga_convoca_ejemplo.csv.

Todos los datos son sintéticos (nombres verosímiles inventados y documentos
enmascarados ficticios). El libro imita la exportación «resumen por persona»
de CONVOCA en su formato real (cabeceras literales y hoja
«grupo de méritos (Tribunal) (1)»). Las puntuaciones se escriben como celdas
numéricas. Solo usa la biblioteca estándar; el resultado es reproducible
(fecha fija en las entradas del ZIP).

Uso: python3 generar_ejemplo.py
"""
import csv
import os
import zipfile
from xml.sax.saxutils import escape

AQUI = os.path.dirname(os.path.abspath(__file__))
HOJA = "grupo de méritos (Tribunal) (1)"
COLUMNAS_NUMERICAS = {5, 6, 7}
FECHA = (2026, 10, 5, 0, 0, 0)


def columna(indice):
    return chr(ord("A") + indice)


def celda(fila, indice, valor):
    ref = f"{columna(indice)}{fila}"
    if valor == "":
        return ""
    if fila > 1 and indice in COLUMNAS_NUMERICAS:
        return f'<c r="{ref}"><v>{valor.replace(",", ".")}</v></c>'
    return f'<c r="{ref}" t="inlineStr"><is><t>{escape(valor)}</t></is></c>'


def main():
    with open(os.path.join(AQUI, "carga_convoca_ejemplo.csv"), encoding="utf-8", newline="") as f:
        filas = list(csv.reader(f))
    hoja = "".join(
        f'<row r="{n}">' + "".join(celda(n, i, v) for i, v in enumerate(fila)) + "</row>"
        for n, fila in enumerate(filas, start=1)
    )
    partes = {
        "[Content_Types].xml": '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
        '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
        '<Default Extension="xml" ContentType="application/xml"/>'
        '<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>'
        '<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>'
        "</Types>",
        "_rels/.rels": '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>'
        "</Relationships>",
        "xl/workbook.xml": '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" '
        'xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">'
        f'<sheets><sheet name="{escape(HOJA)}" sheetId="1" r:id="rId1"/></sheets></workbook>',
        "xl/_rels/workbook.xml.rels": '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>'
        "</Relationships>",
        "xl/worksheets/sheet1.xml": '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">'
        f"<sheetData>{hoja}</sheetData></worksheet>",
    }
    destino = os.path.join(AQUI, "carga_convoca_ejemplo.xlsx")
    with zipfile.ZipFile(destino, "w", zipfile.ZIP_DEFLATED) as libro:
        for nombre, contenido in partes.items():
            info = zipfile.ZipInfo(nombre, FECHA)
            info.compress_type = zipfile.ZIP_DEFLATED
            libro.writestr(info, contenido.encode("utf-8"))


if __name__ == "__main__":
    main()
