"""Pruebas offline de las barreras previas a la escritura."""

import base64
import json
import ssl
import sys
import unittest
from unittest import mock

import sembrar_ejemplo_ct as sembrado


class BarreraDestino(unittest.TestCase):
    def test_base_de_escritura_local_y_sin_componentes_extra(self):
        for base in ("https://localhost:18443", "https://127.0.0.1:18443", "https://[::1]:18443"):
            with self.subTest(base=base):
                sembrado.validar_base(base, True)
        for base in ("https://cidonia.cloud", "https://localhost:18443@cidonia.cloud",
                     "http://localhost:18443", "https://localhost:18443/api",
                     "https://localhost:18443?x=1", "https://localhost:18443#x",
                     "https://localhost:bad", "https://localhost:0"):
            with self.subTest(base=base), self.assertRaises(ValueError):
                sembrado.validar_base(base, True)

    def test_cli_rechaza_escritura_sin_confirmacion_antes_de_cliente(self):
        with mock.patch.object(sys, "argv", ["sembrar", "--ejecutar", "--casos-b64", "e30="]), \
             mock.patch.object(sembrado, "Cliente") as cliente:
            with self.assertRaises(SystemExit) as salida:
                sembrado.main()
        self.assertEqual(salida.exception.code, 2)
        cliente.assert_not_called()

    def test_cli_rechaza_destino_remoto_antes_de_cliente(self):
        with mock.patch.object(sys, "argv", ["sembrar", "--ejecutar", "--confirmar-entorno-sintetico",
                                            "--base", "https://cidonia.cloud", "--casos-b64", "e30="]), \
             mock.patch.object(sembrado, "Cliente") as cliente:
            with self.assertRaises(SystemExit) as salida:
                sembrado.main()
        self.assertEqual(salida.exception.code, 2)
        cliente.assert_not_called()

    def test_catalogo_sin_marca_positiva_bloquea(self):
        for catalogo in ({}, {"preparacion_vias": None},
                         {"preparacion_vias": {"es_ejemplo": False}},
                         {"preparacion_vias": {"es_ejemplo": "true"}}):
            with self.subTest(catalogo=catalogo), self.assertRaises(RuntimeError):
                sembrado.exigir_catalogo_de_ejemplo(catalogo)
        sembrado.exigir_catalogo_de_ejemplo({"preparacion_vias": {"es_ejemplo": True}})

    def test_catalogo_incompleto_impide_todo_post(self):
        casos = {"esquema": "vec.ct.sembrado-ejemplo.v1", "espacio_claves": "prueba",
                 "casos": [{"codigo": "S01", "objetivo": "solicitud", "centro": "210",
                            "categoria": "c2", "modalidad": "sustitucion", "inicio": "2026-10-01",
                            "fin": "2026-10-02"},
                           {"codigo": "S02", "objetivo": "solicitud", "centro": "sin-centro",
                            "categoria": "c2", "modalidad": "sustitucion", "inicio": "2026-10-01",
                            "fin": "2026-10-02"}]}
        datos = base64.b64encode(json.dumps(casos).encode()).decode()
        clase = sembrado.Sembrador
        falso = object.__new__(clase)
        falso.catalogos = {"preparacion_vias": {"es_ejemplo": True},
                           "centros": [{"referencia": "centro:210", "contactos": [{"referencia": "contacto:1"}]}],
                           "categorias": [{"referencia": "categoria:c2", "grupos_subgrupos": [{"clave": "C2"}]}],
                           "motivos": [{"clave": "sustitucion"}]}
        falso.configuracion = {"modalidades": [{"clave": "sustitucion"}]}
        with mock.patch.object(sys, "argv", ["sembrar", "--ejecutar", "--confirmar-entorno-sintetico",
                                            "--casos-b64", datos]), \
             mock.patch.object(sembrado, "Cliente"), \
             mock.patch.object(sembrado, "Sembrador", return_value=falso), \
             mock.patch.object(clase, "sembrar") as escribir:
            self.assertEqual(sembrado.main(), 2)
        escribir.assert_not_called()


class TLS(unittest.TestCase):
    def test_ca_obligatoria(self):
        with mock.patch.object(sembrado.os.path, "isfile", side_effect=[True, True, False]), \
             mock.patch.object(sembrado.ssl, "create_default_context") as contexto:
            with self.assertRaises(SystemExit):
                sembrado.Cliente("https://localhost:18443", "/material", "cliente")
        contexto.assert_not_called()

    def test_ca_y_nombre_verificados_por_contexto_estandar(self):
        contexto = mock.Mock()
        contexto.check_hostname = True
        contexto.verify_mode = ssl.CERT_REQUIRED
        with mock.patch.object(sembrado.os.path, "isfile", return_value=True), \
             mock.patch.object(sembrado.ssl, "create_default_context", return_value=contexto) as crear:
            cliente = sembrado.Cliente("https://localhost:18443", "/material", "cliente")
        crear.assert_called_once_with(cafile="/material/ca/ca.crt")
        contexto.load_cert_chain.assert_called_once_with("/material/mtls/cliente.crt", "/material/mtls/cliente.key")
        self.assertTrue(cliente.ctx.check_hostname)
        self.assertEqual(cliente.ctx.verify_mode, ssl.CERT_REQUIRED)


class Seleccion(unittest.TestCase):
    def setUp(self):
        self.sembrador = object.__new__(sembrado.Sembrador)
        self.sembrador.catalogos = {
            "centros": [{"referencia": "centro:210", "contactos": [{"referencia": "contacto:1"}]}],
            "categorias": [{"referencia": "categoria:auxiliar", "grupos_subgrupos": [{"clave": "C2"}]}],
        }

    def test_referencia_exacta(self):
        self.assertEqual(self.sembrador.centro("210")["referencia"], "centro:210")
        self.assertEqual(self.sembrador.categoria("auxiliar"), ("categoria:auxiliar", "C2"))
        for selector in (lambda: self.sembrador.centro("211"),
                         lambda: self.sembrador.categoria("administrativo")):
            with self.assertRaises(RuntimeError):
                selector()

    def test_ambiguedad_no_elige_el_primero(self):
        self.sembrador.catalogos["centros"][0]["contactos"].append({"referencia": "contacto:2"})
        self.sembrador.catalogos["categorias"][0]["grupos_subgrupos"].append({"clave": "C1"})
        with self.assertRaises(RuntimeError):
            self.sembrador.centro("210")
        with self.assertRaises(RuntimeError):
            self.sembrador.categoria("auxiliar")


if __name__ == "__main__":
    unittest.main()
