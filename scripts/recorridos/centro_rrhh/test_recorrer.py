"""Prueba focal sin navegador ni servicios; nunca crea una petición."""

import unittest
from types import SimpleNamespace

from recorrer import NoEjecutado, origen_local, preflight, verificar_entrega, verificar_recibo_centro


class RecorridoCentroTest(unittest.TestCase):
    def test_origen_solo_loopback_https(self):
        self.assertEqual(origen_local("https://127.0.0.1:8443/"), "https://127.0.0.1:8443")
        for valor in ("http://127.0.0.1:8443", "https://cidonia.cloud",
                      "https://127.0.0.1:8443/ruta", "https://127.0.0.1:invalido"):
            with self.subTest(valor=valor), self.assertRaises(NoEjecutado):
                origen_local(valor)

    def test_sin_acreditacion_no_se_abre_navegador(self):
        with self.assertRaisesRegex(NoEjecutado, "acreditacion"):
            preflight(SimpleNamespace(origen="https://127.0.0.1:8443", acreditacion=None))

    def test_recibos_exigen_identidad_version_y_misma_alta(self):
        centro = {"peticion_ref": "peticion:centro:uno", "version": 2, "estado": "ratificada",
                  "actor_ref": "ratificador", "recibo_ref": "recibo:dos",
                  "registrado_en": "2026-09-29T12:00:00Z", "estado_local": "registrado"}
        self.assertEqual(verificar_recibo_centro(centro, "peticion:centro:uno", 2, "ratificada", "ratificador"),
                         ("recibo:dos", "2026-09-29T12:00:00Z"))
        with self.assertRaises(AssertionError):
            verificar_recibo_centro(centro, "peticion:centro:uno", 2, "ratificada", "solicitante")
        entrega = {"peticion": {"referencia": "peticion:centro:uno"}, "estado_entrega": "confirmada",
                   "recibo_alta": {"recibo_ref": "recibo:alta", "confirmada_en": "2026-09-29T12:01:00Z",
                                    "expediente_ref": "expediente:uno", "numero_visible": "2026/CT-1",
                                    "version": 1, "auditoria_ref": "auditoria:uno", "evento_ref": "evento:uno"}}
        original = verificar_entrega(entrega, "peticion:centro:uno")
        self.assertEqual(original, verificar_entrega(entrega, "peticion:centro:uno"))
        entrega["recibo_alta"]["expediente_ref"] = "expediente:dos"
        self.assertNotEqual(original, verificar_entrega(entrega, "peticion:centro:uno"))


if __name__ == "__main__":
    unittest.main()
