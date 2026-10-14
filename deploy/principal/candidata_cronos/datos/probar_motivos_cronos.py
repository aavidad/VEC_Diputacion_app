#!/usr/bin/env python3
"""Comprueba que la semilla y la migración publican el mismo catálogo."""
import hashlib
import json
from pathlib import Path
import re
import unittest

RAIZ = Path(__file__).resolve().parents[4]
CATALOGO = RAIZ / "config/cronos_motivos_v1.json"
MIGRACION = RAIZ / "deploy/postgresql/cronos_v1/migraciones/000018_catalogo_motivos_cronos_v1.up.sql"


class CatalogoCronosTest(unittest.TestCase):
    def test_referencias_y_publicacion(self):
        catalogo = json.loads(CATALOGO.read_text(encoding="utf-8"))
        sql = MIGRACION.read_text(encoding="utf-8")
        entradas_sql = json.loads(re.search(r"entradas constant jsonb := '(\[.*?\])';", sql, re.S).group(1))
        canon = {clave: catalogo[clave] for clave in ("catalogo_id", "catalogo_version", "entradas")}
        huella = hashlib.sha256(json.dumps(canon, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()

        self.assertEqual(catalogo["catalogo_id"], "motivos_cronos_v1")
        self.assertEqual(catalogo["catalogo_version"], 1)
        self.assertEqual(catalogo["fuente"]["tipo"], "semilla_sintetica")
        self.assertEqual(catalogo["entradas"], entradas_sql)
        self.assertEqual(catalogo["catalogo_huella_publicada_sha256"], huella)
        self.assertIn(huella, sql)
        self.assertIn(catalogo["publicado_en"], sql)
        self.assertEqual(set(catalogo["referencias_tecnicas"]), {"consulta", "actuacion", "denegado"})
        claves = list(catalogo["referencias_tecnicas"].values())
        self.assertEqual(len(set(claves)), 3)
        self.assertEqual(set(claves), {entrada["clave"] for entrada in entradas_sql})
        for entrada in entradas_sql:
            self.assertRegex(entrada["clave"], r"^motivo_[0-9a-f]{32}$")
            self.assertLessEqual(entrada["vigente_desde"], "2026-10-14T00:00:00.000000Z")
            self.assertIsNone(entrada["vigente_hasta"])
        self.assertRegex(sql, r"evento_[0-9a-f]{32}")
        self.assertNotIn("INSERT INTO vec_autorizacion.", sql)
        self.assertIn("publicada := vec_autorizacion.publicar_motivos_autorizacion_v2(", sql)


if __name__ == "__main__":
    unittest.main()
