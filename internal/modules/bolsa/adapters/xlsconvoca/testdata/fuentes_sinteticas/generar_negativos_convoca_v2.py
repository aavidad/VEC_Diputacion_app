"""Genera los libros XLS sintéticos negativos del formato convoca:v2.

Solo usa literales de cabecera y nombres de hoja; la única fila de datos es
inventada. Uso: python3 generar_negativos_convoca_v2.py <directorio_salida>
"""
import sys
import unicodedata
from pathlib import Path

import xlwt

V1 = ["DNI/NIE", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
      "Experiencia", "Formacion", "Total"]
V2 = ["DNI/NIE enmascarado", "Primer Apellido", "Segundo Apellido", "Nombre", "Turno",
      "Experiencia", "Formación", "Total"]
FILA = ["***0001**", "Prueba", "Sintética", "Ana", "Libre", 1.5, 2.0, 3.5]
HOJA_RESUMEN = "grupo de méritos (Tribunal) (1)"
HOJA_DETALLE = "méritos (1)"

CASOS = {
    # Cabeceras reales con el sufijo « (2)», no acreditado: se rechaza.
    "convoca_v2_hoja_sufijo_2.xls": ("grupo de méritos (Tribunal) (2)", V2),
    # Cabeceras reales en una hoja con nombre ajeno a CONVOCA.
    "convoca_v2_hoja_ajena.xls": ("Hoja1", V2),
    # Cabeceras reales de resumen en la hoja del detalle.
    "convoca_v2_hoja_cruzada.xls": (HOJA_DETALLE, V2),
    # Cabeceras v1 de resumen en la hoja real del detalle.
    "convoca_v1_hoja_cruzada.xls": (HOJA_DETALLE, V1),
    # Mezcla de formatos: documento v2 con el resto de cabeceras v1.
    "convoca_mezcla_formatos.xls": (HOJA_RESUMEN, V2[:1] + V1[1:]),
    # Cabeceras reales en forma canónica descompuesta (NFD): mismo texto, se
    # acepta. La hoja queda en NFC porque en NFD supera los 31 caracteres.
    "convoca_v2_nfd.xls": (HOJA_RESUMEN, [unicodedata.normalize("NFD", c) for c in V2]),
}


def main() -> None:
    salida = Path(sys.argv[1])
    for nombre, (hoja, cabeceras) in CASOS.items():
        libro = xlwt.Workbook(encoding="utf-8")
        h = libro.add_sheet(hoja)
        for columna, titulo in enumerate(cabeceras):
            h.write(0, columna, titulo)
        for columna, valor in enumerate(FILA):
            h.write(1, columna, valor)
        libro.save(str(salida / nombre))


if __name__ == "__main__":
    main()
