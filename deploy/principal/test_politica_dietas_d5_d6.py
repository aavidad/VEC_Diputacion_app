"""Las concesiones D5/D6 coinciden byte a byte con AD3-75 y Dietas 000011."""

from __future__ import annotations

import json
from pathlib import Path
import re
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import politica_dietas_d5_d6 as politica  # noqa: E402

RAIZ = Path(__file__).resolve().parents[2] / "deploy/postgresql"
AD3_75 = RAIZ / "autorizacion_atestada_v3/migraciones/000075_campo_devolucion_documento_dietas.up.sql"
DIETAS_11 = RAIZ / "dietas_borradores/migraciones/000011_campo_devolucion_decision.up.sql"


def listas_ad3_75(fachada: str) -> tuple[list[str], list[str]]:
    """Lista base y lista con devolución que AD3-75 instala en la fachada."""
    sql = AD3_75.read_text(encoding="utf-8")
    inicio = sql.index(f"('{fachada}'")
    siguiente = sql.find("('registrar_y_consumir_", inicio + 1)
    tramo = sql[inicio:siguiente if siguiente >= 0 else sql.index(") v(nombre", inicio)]
    campos = re.findall(r"campos jsonb:='(\[.*?\])'::jsonb", tramo)
    devolucion = re.findall(r"campos_devolucion jsonb:='(\[.*?\])'::jsonb", tramo)
    # La primera `campos` es la preimagen (AD3-59/80); la segunda, la nueva base.
    assert len(campos) == 2 and len(devolucion) == 1, fachada
    return json.loads(campos[1]), json.loads(devolucion[0])


def listas_dietas_11() -> set[str]:
    sql = DIETAS_11.read_text(encoding="utf-8")
    return set(re.findall(r"campos(?:_devolucion)?:='(\[.*?\])'::jsonb", sql))


def texto(lista: list[str]) -> str:
    return json.dumps(lista, separators=(",", ":"))


class PoliticaDietasD5D6Test(unittest.TestCase):
    def test_listas_iguales_a_ad3_75_y_en_orden_de_bytes(self) -> None:
        casos = (
            ("registrar_y_consumir_dietas_documento_v3_atestada", politica.campos_documento, 22),
            ("registrar_y_consumir_dietas_documento_consulta_v3_atestada", politica.campos_consulta, 45),
            ("registrar_y_consumir_dietas_revisor_documento_v3_atestada", politica.campos_revisor, 16),
        )
        for fachada, funcion, tamano in casos:
            base, devolucion = listas_ad3_75(fachada)
            self.assertEqual(funcion(False), base, fachada)
            self.assertEqual(funcion(True), devolucion, fachada)
            self.assertEqual(len(base), tamano)
            self.assertEqual(len(devolucion), tamano + (2 if tamano == 45 else 1))
            for lista in (base, devolucion):
                self.assertEqual(lista, sorted(lista, key=lambda campo: campo.encode()))

    def test_dietas_000011_coteja_las_mismas_listas(self) -> None:
        cotejadas = listas_dietas_11()
        for devolucion in (False, True):
            for lista in (politica.campos_documento(devolucion), politica.campos_consulta(devolucion),
                          politica.campos_revisor(devolucion)):
                self.assertIn(texto(lista), cotejadas)

    def test_concesiones_exactas(self) -> None:
        titular = politica.concesiones_titular()
        self.assertEqual([(c["accion"], c["tipo_recurso"], c["finalidades"]) for c in titular], [
            ("dietas.borrador.propio.editar", "comision_borrador", ["editar_borrador_propio"]),
            ("dietas.borrador.propio.borrar", "comision_borrador", ["borrar_borrador_propio"]),
            ("dietas.borrador.propio.enviar", "comision_borrador", ["enviar_borrador_propio"]),
            ("dietas.documento.propio.consultar", "comision_borrador", ["consultar_documento_propio_dietas"]),
        ])
        self.assertEqual([len(c["campos_permitidos"]) for c in titular], [23, 23, 23, 47])
        self.assertEqual([len(c["campos_permitidos"]) for c in politica.concesiones_titular(False)], [22, 22, 22, 45])
        revisor = politica.concesiones_revisor()
        self.assertEqual([(c["accion"], c["tipo_recurso"], c["finalidades"], len(c["campos_permitidos"])) for c in revisor],
                         [("dietas.circuito.documento.consultar", "documento_dietas", ["revisar_documento_circuito_dietas"], 17)])
        for concesion in titular + revisor:
            self.assertEqual(concesion["modulo_id"], "dietas")
            self.assertEqual(concesion["garantia_minima"], "alto")


if __name__ == "__main__":
    unittest.main()
