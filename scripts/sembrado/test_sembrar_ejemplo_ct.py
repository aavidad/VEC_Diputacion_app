"""Pruebas offline de la selección y de las barreras antes de cada petición."""

import hashlib
import json
import ssl
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import sembrar_ejemplo_ct as sembrado


HUELLA = hashlib.sha256(b"certificado de prueba").hexdigest()


class CLI(unittest.TestCase):
    def test_huella_debe_ser_sha256(self):
        self.assertEqual(sembrado.validar_huella(HUELLA.upper()), HUELLA)
        for invalida in ("", "0" * 63, "g" * 64, "0" * 65):
            with self.subTest(huella=invalida), self.assertRaises(ValueError):
                sembrado.validar_huella(invalida)

    def test_sin_destino_o_con_url_no_crea_cliente(self):
        base = ["sembrar", "--puerto-interno=8443", f"--huella-servidor-sha256={HUELLA}", "--casos=x"]
        for argumentos in (base, [*base, "--destino=clon", "--base=https://localhost:8443"],
                           [*base, "--destino=clon", "--casos-b64=e30="]):
            with self.subTest(argumentos=argumentos), mock.patch.object(sys, "argv", argumentos), \
                 mock.patch.object(sembrado, "Cliente") as cliente, mock.patch("sys.stderr"):
                with self.assertRaises(SystemExit) as salida:
                    sembrado.main()
                self.assertEqual(salida.exception.code, 2)
                cliente.assert_not_called()

    def test_ejecutar_sin_tty_no_crea_cliente(self):
        argumentos = ["sembrar", "--destino=principal", "--puerto-interno=8443",
                      f"--huella-servidor-sha256={HUELLA}", "--casos=x", "--ejecutar"]
        with mock.patch.object(sys, "argv", argumentos), mock.patch.object(sys.stdin, "isatty", return_value=False), \
             mock.patch.object(sembrado, "Cliente") as cliente, mock.patch("sys.stderr"):
            with self.assertRaises(SystemExit) as salida:
                sembrado.main()
        self.assertEqual(salida.exception.code, 2)
        cliente.assert_not_called()

    def test_plan_y_confirmacion_literal(self):
        with tempfile.TemporaryDirectory() as temporal:
            ruta = Path(temporal) / "casos.json"
            ruta.write_text(json.dumps({"esquema": "vec.ct.sembrado-ejemplo.v1", "espacio_claves": "prueba",
                                        "casos": [{"codigo": "S01", "objetivo": "solicitud"}]}), encoding="utf-8")
            argumentos = ["sembrar", "--destino=clon", "--puerto-interno=8443",
                          f"--huella-servidor-sha256={HUELLA}", f"--casos={ruta}"]
            falso = mock.Mock()
            falso.plan.return_value = {"centro": {"etiqueta": "Centro"}, "categoria": "categoria:c2"}
            with mock.patch.object(sys, "argv", argumentos), mock.patch.object(sembrado, "Cliente"), \
                 mock.patch.object(sembrado, "Sembrador", return_value=falso), \
                 mock.patch("builtins.print") as imprimir:
                self.assertEqual(sembrado.main(), 0)
                self.assertTrue(any("SEMBRADO-PLAN" in str(c) for c in imprimir.call_args_list))
            falso.sembrar.assert_not_called()

            with mock.patch.object(sys, "argv", [*argumentos, "--ejecutar"]), \
                 mock.patch.object(sys.stdin, "isatty", return_value=True), \
                 mock.patch("builtins.input", return_value="principal"), \
                 mock.patch.object(sembrado, "Cliente"), \
                 mock.patch.object(sembrado, "Sembrador", return_value=falso), \
                 mock.patch("builtins.print"):
                self.assertEqual(sembrado.main(), 2)
            falso.sembrar.assert_not_called()

            falso.sembrar.return_value = {"expediente_ref": "expediente:prueba", "numero": "1",
                                          "objetivo": "solicitud", "ya_estaba": 0, "hechos": ["alta"]}
            with mock.patch.object(sys, "argv", [*argumentos, "--ejecutar"]), \
                 mock.patch.object(sys.stdin, "isatty", return_value=True), \
                 mock.patch("builtins.input", return_value="clon"), \
                 mock.patch.object(sembrado, "Cliente"), \
                 mock.patch.object(sembrado, "Sembrador", return_value=falso), \
                 mock.patch.object(sembrado, "resumen"), mock.patch("builtins.print"):
                self.assertEqual(sembrado.main(), 0)
            falso.sembrar.assert_called_once()

    def test_preflight_completo_impide_escritura(self):
        casos = [{"codigo": "S01", "objetivo": "solicitud", "centro": "210", "categoria": "c2",
                  "modalidad": "sustitucion", "inicio": "2026-10-01", "fin": "2026-10-02"},
                 {"codigo": "S02", "objetivo": "solicitud", "centro": "sin-centro", "categoria": "c2",
                  "modalidad": "sustitucion", "inicio": "2026-10-01", "fin": "2026-10-02"}]
        with tempfile.TemporaryDirectory() as temporal:
            ruta = Path(temporal) / "casos.json"
            ruta.write_text(json.dumps({"esquema": "vec.ct.sembrado-ejemplo.v1", "espacio_claves": "prueba",
                                        "casos": casos}), encoding="utf-8")
            clase = sembrado.Sembrador
            falso = object.__new__(clase)
            falso.catalogos = {"centros": [{"referencia": "centro:210", "contactos": [{"referencia": "contacto:1"}]}],
                               "categorias": [{"referencia": "categoria:c2", "grupos_subgrupos": [{"clave": "C2"}]}],
                               "motivos": [{"clave": "sustitucion"}]}
            falso.configuracion = {"modalidades": [{"clave": "sustitucion"}]}
            argumentos = ["sembrar", "--destino=clon", "--puerto-interno=8443",
                          f"--huella-servidor-sha256={HUELLA}", f"--casos={ruta}", "--ejecutar"]
            with mock.patch.object(sys, "argv", argumentos), mock.patch.object(sys.stdin, "isatty", return_value=True), \
                 mock.patch.object(sembrado, "Cliente"), \
                 mock.patch.object(sembrado, "Sembrador", return_value=falso), \
                 mock.patch.object(clase, "sembrar") as escribir, mock.patch("builtins.input") as confirmar, \
                 mock.patch("builtins.print"):
                self.assertEqual(sembrado.main(), 2)
            escribir.assert_not_called()
            confirmar.assert_not_called()


class TLS(unittest.TestCase):
    def test_ca_obligatoria(self):
        with mock.patch.object(sembrado.os.path, "isfile", side_effect=[True, True, False]), \
             mock.patch.object(sembrado.ssl, "create_default_context") as contexto:
            with self.assertRaises(sembrado.ErrorDestino):
                sembrado.Cliente(8443, "cliente", HUELLA)
        contexto.assert_not_called()

    def test_contexto_verificado_y_pin_antes_de_cada_request(self):
        contexto = mock.Mock()
        contexto.check_hostname = True
        contexto.verify_mode = ssl.CERT_REQUIRED
        conexiones, eventos = [], []

        def nueva_conexion(host, puerto, **kwargs):
            self.assertEqual((host, puerto), ("localhost", 8443))
            self.assertIs(kwargs["context"], contexto)
            con = mock.Mock()
            con.sock.getpeercert.return_value = b"certificado de prueba"
            con.connect.side_effect = lambda: eventos.append("connect")
            con.request.side_effect = lambda *args, **kw: eventos.append("request")
            con.getresponse.return_value.status = 200
            con.getresponse.return_value.read.return_value = b'{"data":{}}'
            conexiones.append(con)
            return con

        with mock.patch.object(sembrado.os.path, "isfile", return_value=True), \
             mock.patch.object(sembrado.ssl, "create_default_context", return_value=contexto) as crear, \
             mock.patch.object(sembrado.http.client, "HTTPSConnection", side_effect=nueva_conexion):
            cliente = sembrado.Cliente(8443, "cliente", HUELLA)
            cliente.pedir("GET", "/uno")
            cliente.pedir("GET", "/dos")
        crear.assert_called_once_with(cafile="/vec-material/ca/ca.crt")
        contexto.load_cert_chain.assert_called_once_with("/vec-material/mtls/cliente.crt",
                                                          "/vec-material/mtls/cliente.key")
        self.assertTrue(cliente.ctx.check_hostname)
        self.assertEqual(cliente.ctx.verify_mode, ssl.CERT_REQUIRED)
        self.assertEqual(eventos, ["connect", "request", "connect", "request"])
        self.assertEqual(len(conexiones), 2)

    def test_pin_distinto_corta_antes_del_request(self):
        con = mock.Mock()
        con.sock.getpeercert.return_value = b"otro certificado"
        with mock.patch.object(sembrado.os.path, "isfile", return_value=True), \
             mock.patch.object(sembrado.ssl, "create_default_context"), \
             mock.patch.object(sembrado.http.client, "HTTPSConnection", return_value=con):
            cliente = sembrado.Cliente(8443, "cliente", HUELLA)
            with self.assertRaises(sembrado.ErrorDestino):
                cliente.pedir("POST", "/escritura", {"dato": 1})
        con.connect.assert_called_once()
        con.request.assert_not_called()
        con.close.assert_called_once()


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
