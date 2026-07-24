"""Pruebas del contrato de capturas de contratación solicitado por RRHH."""

from __future__ import annotations

import unittest
from dataclasses import replace

from scripts import capturar_contratacion_rrhh as capturador


class CapturadorContratacionRRHHTest(unittest.TestCase):
    def test_la_matriz_es_exactamente_la_numeracion_recibida_de_rrhh(self) -> None:
        self.assertEqual(capturador.validar_matriz(), [])
        self.assertEqual(len(capturador.PANTALLAS), 17)
        self.assertEqual(
            [pantalla.numero for pantalla in capturador.PANTALLAS],
            list(range(1, 18)),
        )
        self.assertEqual(capturador.PANTALLAS[0].clave, "cuadro-mando")
        self.assertEqual(capturador.PANTALLAS[1].clave, "nueva-peticion")
        self.assertEqual(capturador.PANTALLAS[-1].clave, "generacion-documental")

    def test_ginpix_tiene_preparacion_y_envio_como_pantallas_distintas(self) -> None:
        por_numero = {pantalla.numero: pantalla for pantalla in capturador.PANTALLAS}
        self.assertEqual(por_numero[15].referencia_tarea, "tarea-ginpix")
        self.assertEqual(por_numero[16].referencia_tarea, "tarea-envio-ginpix")
        self.assertNotEqual(
            por_numero[15].referencia_tarea,
            por_numero[16].referencia_tarea,
        )

    def test_captura_la_referencia_y_dos_anchuras_de_escritorio(self) -> None:
        self.assertEqual(
            [(tamano.ancho, tamano.alto) for tamano in capturador.TAMANOS],
            [(1536, 1024), (1440, 1000), (1280, 900)],
        )

    def test_rechaza_matrices_incompletas_duplicadas_o_no_canonicas(self) -> None:
        self.assertTrue(capturador.validar_matriz(capturador.PANTALLAS[:-1]))
        duplicada = (
            *capturador.PANTALLAS[:-1],
            replace(
                capturador.PANTALLAS[-1],
                clave=capturador.PANTALLAS[0].clave,
            ),
        )
        self.assertTrue(capturador.validar_matriz(duplicada))
        no_canonica = (
            replace(capturador.PANTALLAS[0], clave="Cuadro mando"),
            *capturador.PANTALLAS[1:],
        )
        self.assertTrue(capturador.validar_matriz(no_canonica))

    def test_solo_admite_origen_local_sin_credenciales_ni_ruta(self) -> None:
        self.assertEqual(
            capturador.normalizar_url_base(" http://127.0.0.1:8081/ "),
            "http://127.0.0.1:8081",
        )
        self.assertEqual(
            capturador.normalizar_url_base("https://localhost:8443"),
            "https://localhost:8443",
        )
        for invalida in (
            "https://vep.dipgra.es",
            "http://usuario:clave@127.0.0.1:8081",
            "http://127.0.0.1:8081/portal-empleado/",
            "ftp://127.0.0.1:8081",
        ):
            with self.subTest(invalida=invalida):
                with self.assertRaises(ValueError):
                    capturador.normalizar_url_base(invalida)


if __name__ == "__main__":
    unittest.main()
