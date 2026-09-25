#!/usr/bin/env python3
"""Concesiones V3 exactas para activar Dietas D5/D6 (documento y devolución).

Solo datos: no conecta a ninguna base, no publica gobierno V3 ni crea roles o
asignaciones. Quien publica la nueva versión del rol de la persona titular y
la del revisor toma de aquí las concesiones, que deben coincidir byte a byte
con lo que exigen las fachadas AD3 y los cotejos de Dietas:

- AD3-59/AD3-75 `registrar_y_consumir_dietas_documento_v3_atestada`
  (editar, borrar y enviar el documento propio);
- AD3-59/AD3-75 `registrar_y_consumir_dietas_documento_consulta_v3_atestada`
  (detalle y listado del documento propio);
- AD3-80/AD3-75 `registrar_y_consumir_dietas_revisor_documento_v3_atestada`
  (lectura del documento por quien lo revisa en su etapa).

Cada fachada admite EXACTAMENTE la lista base o la base con la devolución
(Dietas 000011 coteja lo mismo). `campos_permitidos` va en orden de bytes: la
comparación SQL es de arrays jsonb y el orden cuenta. Las listas se copian de
AD3-75 y `test_politica_dietas_d5_d6.py` las compara con la migración.

Orden de activación: AD3-75 → Dietas 000009 → 000010 → 000011 (paquete
`04_dietas_migraciones.sh --incremental`) → esta política → binario.

Uso: `python3 politica_dietas_d5_d6.py [--sin-devolucion]` imprime JSON con
las concesiones de `titular` y `revisor`. Sin secretos ni datos personales.
"""

from __future__ import annotations

import json
import sys

GARANTIA_MINIMA = "alto"

# Lista base del documento propio (mutación y respuesta), la de Dietas 000006.
_DOCUMENTO = (
    "comision.calculo", "comision.centro_ref", "comision.codigos_ruta", "comision.documento",
    "comision.estado", "comision.fecha_apertura", "comision.fecha_fin", "comision.fecha_inicio",
    "comision.motivo", "comision.numero_documento", "comision.referencia", "comision.relacion_ref",
    "comision.rutas", "comision.unidad_ref", "comision.vehiculo_propio", "comision.version",
    "recibo.referencia", "recibo.registrado_en", "recibo.regla_huella_sha256", "recibo.regla_ref",
    "recibo.repeticion", "recibo.version",
)
# Lista base de quien revisa (AD3-80, Dietas 000008).
_REVISOR = (
    "comision.calculo", "comision.codigos_ruta", "comision.documento", "comision.estado",
    "comision.fecha_apertura", "comision.fecha_fin", "comision.fecha_inicio", "comision.hora_fin",
    "comision.hora_inicio", "comision.motivo", "comision.numero_documento", "comision.referencia",
    "comision.rutas", "comision.vehiculo_propio", "comision.version", "resultado",
)
DEVOLUCION = "comision.devolucion"


def _ordenar(campos: list[str]) -> list[str]:
    return sorted(campos, key=lambda campo: campo.encode("utf-8"))


def campos_documento(devolucion: bool) -> list[str]:
    """22 campos, o 23 con la devolución."""
    return _ordenar(list(_DOCUMENTO) + ([DEVOLUCION] if devolucion else []))


def campos_consulta(devolucion: bool) -> list[str]:
    """Detalle y listado: 45 campos, o 47 con la devolución y su `items.`."""
    base = campos_documento(devolucion)
    return _ordenar(base + ["items." + campo for campo in base] + ["siguiente_cursor"])


def campos_revisor(devolucion: bool) -> list[str]:
    """16 campos, o 17 con la devolución."""
    return _ordenar(list(_REVISOR) + ([DEVOLUCION] if devolucion else []))


def _concesion(accion: str, tipo_recurso: str, finalidad: str, campos: list[str]) -> dict:
    return dict(accion=accion, modulo_id="dietas", tipo_recurso=tipo_recurso, finalidades=[finalidad],
                garantia_minima=GARANTIA_MINIMA, campos_permitidos=campos)


def concesiones_titular(devolucion: bool = True) -> list[dict]:
    """Editar, borrar, enviar y consultar el documento propio."""
    resultado = [
        _concesion(f"dietas.borrador.propio.{op}", "comision_borrador", f"{op}_borrador_propio", campos_documento(devolucion))
        for op in ("editar", "borrar", "enviar")
    ]
    resultado.append(_concesion("dietas.documento.propio.consultar", "comision_borrador",
                                "consultar_documento_propio_dietas", campos_consulta(devolucion)))
    return resultado


def concesiones_revisor(devolucion: bool = True) -> list[dict]:
    """Lectura del documento por quien lo revisa en su etapa del circuito."""
    return [_concesion("dietas.circuito.documento.consultar", "documento_dietas",
                       "revisar_documento_circuito_dietas", campos_revisor(devolucion))]


def main(argumentos: list[str]) -> int:
    if argumentos not in ([], ["--sin-devolucion"]):
        print("uso: politica_dietas_d5_d6.py [--sin-devolucion]", file=sys.stderr)
        return 2
    devolucion = not argumentos
    json.dump({"titular": concesiones_titular(devolucion), "revisor": concesiones_revisor(devolucion)},
              sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
