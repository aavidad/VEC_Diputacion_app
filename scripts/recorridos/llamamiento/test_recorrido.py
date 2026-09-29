"""Prueba focal sintética del cierre previo y de la comparación de recibos."""

import hashlib
import json
import stat
import tempfile
import unittest
from pathlib import Path

from recorrido import (FECHA_ETAPA, FalloRecorrido, NoEjecutado, comprobar_entrada,
                       comprobar_recibo, fichero_externo)


class RecorridoPrueba(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        self.raiz = Path(self.temporal.name) / "repositorio"
        self.raiz.mkdir()
        externo = Path(self.temporal.name) / "material"
        externo.mkdir()
        self.binario = externo / "vec-server"
        self.binario.write_bytes(b"binario sintetico sin servidor")
        self.binario.chmod(self.binario.stat().st_mode | stat.S_IXUSR)
        huella = hashlib.sha256(self.binario.read_bytes()).hexdigest()
        rrhh = externo / "rrhh.crt"
        candidato = externo / "candidato.crt"
        clave_rrhh = externo / "rrhh.key"
        clave_candidato = externo / "candidato.key"
        for ruta, contenido in ((rrhh, b"rrhh"), (candidato, b"candidato"),
                                (clave_rrhh, b"clave rrhh"), (clave_candidato, b"clave candidato")):
            ruta.write_bytes(contenido)
        manifiesto = externo / "clon.json"
        manifiesto.write_text(json.dumps({"origen": "https://127.0.0.1:18443", "clon": "H3-H5",
                                         "hitos_instalados": ["H3", "H4", "H5"], "datos": "sinteticos",
                                         "listo": True, "binario_sha256": huella}), encoding="utf-8")
        self.escenario = {"origen": "https://127.0.0.1:18443", "clon": "H3-H5",
                          "manifiesto_clon": str(manifiesto), "binario": str(self.binario),
                          "binario_sha256": huella, "bolsa_ref": "bolsa:sintetica",
                          "identidades": {
                              "rrhh": {"certificado": str(rrhh), "clave": str(clave_rrhh),
                                       "ruta": "/portal-empleado/"},
                              "candidato": {"certificado": str(candidato), "clave": str(clave_candidato),
                                            "ruta": "/area-personal/?vista=llamamientos"}},
                          "respuesta": "renuncia",
                          "etapas": [{"nombre": nombre, "campos_recibo": {
                              "recibo" if nombre == "candidato_respuesta" else "recibo_ref": "recibo:sintetico",
                              FECHA_ETAPA[nombre]: "2026-09-29T00:00:00.000000Z"}}
                                     for nombre in ("seleccion", "comunicacion", "candidato_respuesta",
                                                    "declaracion_rrhh", "resolucion_rrhh", "siguiente")],
                          }

    def test_escenario_valido_solo_prepara(self):
        self.assertIs(comprobar_entrada(self.escenario, self.raiz), self.escenario)

    def test_faltan_clon_o_binario_cierra(self):
        for campo, valor in (("clon", "HITO1"), ("binario_sha256", "0" * 64)):
            with self.subTest(campo=campo):
                escenario = {**self.escenario, campo: valor}
                with self.assertRaises(NoEjecutado):
                    comprobar_entrada(escenario, self.raiz)

    def test_identidad_separada_y_secuencia_obligatorias(self):
        identidades = dict(self.escenario["identidades"])
        identidades["candidato"] = {**identidades["candidato"],
                                     "certificado": identidades["rrhh"]["certificado"]}
        with self.assertRaises(NoEjecutado):
            comprobar_entrada({**self.escenario, "identidades": identidades}, self.raiz)
        with self.assertRaises(NoEjecutado):
            comprobar_entrada({**self.escenario, "etapas": self.escenario["etapas"][:-1]}, self.raiz)

    def test_recibo_inmutable(self):
        comprobar_recibo({"data": {"recibo_ref": "recibo:1", "version_resultante": 3}},
                        {"recibo_ref": "recibo:1", "version_resultante": 3})
        with self.assertRaises(FalloRecorrido):
            comprobar_recibo({"data": {"recibo_ref": "recibo:2"}}, {"recibo_ref": "recibo:1"})

    def test_material_en_raiz_compartida_no_se_acepta(self):
        worktree = self.raiz / ".worktrees" / "otro"
        worktree.mkdir(parents=True)
        material = self.raiz / "clave.key"
        material.write_bytes(b"sintetico")
        with self.assertRaises(NoEjecutado):
            fichero_externo(str(material), worktree)


if __name__ == "__main__":
    unittest.main()
