"""Guardas del guion: ninguna petición con efecto al recuperar, sin secretos."""

import importlib.util
import io
import json
import unittest
import tempfile
from contextlib import redirect_stdout
from types import SimpleNamespace
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("hz39_recorrer", ROOT / "recorrer.py")
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


class GuionHZ39(unittest.TestCase):
    def test_dialogo_solo_confirma_el_expediente_previsto(self):
        eventos = []
        pagina = SimpleNamespace(once=lambda evento, manejador: eventos.append((evento, manejador)))
        runner.accion(pagina, {"tipo": "confirmar_dialogo", "valor": "$expediente_ref"},
                      {"expediente_ref": "expediente:1"}, "https://127.0.0.1:9443")
        self.assertEqual(eventos[0][0], "dialog")
        aceptados, rechazados = [], []
        dialogo = SimpleNamespace(type="confirm", message="Confirmar expediente:1",
                                  accept=lambda: aceptados.append(True), dismiss=lambda: rechazados.append(True))
        eventos[0][1](dialogo)
        self.assertEqual(aceptados, [True])
        for mensaje in ("Confirmar expediente:2", "Referencia: expediente:10"):
            dialogo.message = mensaje
            with self.assertRaises(runner.EntradaInvalida):
                eventos[0][1](dialogo)
        self.assertEqual(rechazados, [True, True])
        self.assertEqual(aceptados, [True])

    def test_fecha_del_recibo_de_peticion_se_conserva_al_recuperar(self):
        recibo = {"recibo_ref": "recibo:centro:1", "version": 1,
                  "registrado_en": "2026-10-14T17:00:00.000001Z"}
        self.assertEqual(runner.recibos({"data": recibo}), [recibo])
        original = {"recibos_efecto": [recibo]}
        runner.comprobar_recuperacion({"http": [{"metodo": "POST"}]}, original, {"recibos": [recibo]})
        distinto = {**recibo, "registrado_en": "2026-10-14T17:00:00.000002Z"}
        with self.assertRaises(runner.EntradaInvalida):
            runner.comprobar_recuperacion({"http": [{"metodo": "POST"}]}, original, {"recibos": [distinto]})

    def test_callback_playwright_con_dos_argumentos_conserva_actor_y_origen(self):
        with patch.object(runner.Ejecucion, "filtrar") as filtrar:
            ejecucion = runner.Ejecucion(Path("/fuera"), {"valores": {}})
            transporte_a, transporte_b = object(), object()
            origen_a, origen_b = "https://127.0.0.1:9443", "https://127.0.0.1:9553"
            actor_a = runner.filtro_del_actor(ejecucion, origen_a, transporte_a)
            actor_b = runner.filtro_del_actor(ejecucion, origen_b, transporte_b)
            ruta, solicitud = object(), object()
            actor_a(ruta, solicitud)
            filtrar.assert_called_with(ruta, origen_a, transporte_a)
            actor_b(ruta, solicitud)
            filtrar.assert_called_with(ruta, origen_b, transporte_b)
            actor_a(ruta, solicitud)
            filtrar.assert_called_with(ruta, origen_a, transporte_a)

    def test_avance_admite_correccion_y_recuperacion_exige_commit_congelado(self):
        with tempfile.TemporaryDirectory() as directorio:
            raiz = Path(directorio)
            claves, datos, anterior = [raiz / n for n in ("claves.json", "datos.json", "anterior.json")]
            for ruta, valor in ((claves, {}), (datos, {"candidata": "a" * 40, "valores": {}})):
                ruta.write_text(json.dumps(valor))
                ruta.chmod(0o600)
            for modo, previa, esperado in (
                ("continuar", "b" * 40, 0),
                ("recuperar", "b" * 40, 2),
                ("recuperar", "a" * 40, 0),
                ("continuar", "main", 2),
            ):
                anterior.write_text(json.dumps({"candidata": previa, "valores": {}, "pasos": []}))
                anterior.chmod(0o600)
                argumentos = ["recorrer.py", "--base-url", "https://127.0.0.1:9443",
                              "--ca", str(raiz / "ca.crt"), "--certificados", str(raiz),
                              "--claves", str(claves), "--datos", str(datos), "--salida", str(raiz / "salida"),
                              "--modo", modo, "--anterior", str(anterior), "--reinicio-acreditado"]
                with self.subTest(modo=modo, previa=previa), patch.object(runner.sys, "argv", argumentos), \
                     patch.object(runner, "identidades", return_value={}), \
                     patch.object(runner, "ejecutar", return_value={"completo": True}) as ejecutar, \
                     redirect_stdout(io.StringIO()):
                    self.assertEqual(runner.main(), esperado)
                    self.assertEqual(ejecutar.called, esperado == 0)

    def test_espera_renderizado_y_rechaza_botones_duplicados(self):
        eventos = []

        class Elemento:
            cantidad = 0

            def count(self):
                return self.cantidad

            def wait_for(self, *, state):
                eventos.append(state)
                self.cantidad = 1

            def click(self):
                eventos.append("click")

        elemento = Elemento()
        pagina = SimpleNamespace(locator=lambda _selector: elemento)
        accion = {"tipo": "click", "selector": "[data-accion]"}
        runner.accion(pagina, accion, {}, "https://127.0.0.1:9443")
        self.assertEqual(eventos, ["attached", "click"])
        elemento.cantidad = 2
        with self.assertRaises(runner.EntradaInvalida):
            runner.accion(pagina, accion, {}, "https://127.0.0.1:9443")
        self.assertEqual(eventos, ["attached", "click"])

    def test_recuperacion_bloquea_un_acto_antes_de_enviarlo(self):
        class Request:
            method = "POST"
            url = "https://127.0.0.1:9443/api/vec/contratacion-temporal/analisis"

        class Route:
            request = Request()
            aborted = False

            def abort(self):
                self.aborted = True

        e = runner.Ejecucion(Path("/fuera"), {"valores": {}})
        e.actual = {"red_fallida": []}
        e.solo_lectura = True
        r = Route()
        with patch.object(runner, "responder_sin_redireccion") as enviar:
            e.filtrar(r, "https://127.0.0.1:9443")
            enviar.assert_not_called()
        self.assertTrue(r.aborted)
        self.assertEqual(
            e.actual["red_fallida"][0]["codigo"], "efecto_bloqueado_en_recuperacion"
        )

    def test_intencion_no_guarda_credenciales_del_cuerpo(self):
        class Request:
            method = "POST"
            url = "https://127.0.0.1:9443/api/vec/acto"
            post_data_json = {
                "clave_idempotencia": "clave-123",
                "password": "no_guardar",
                "sesion_token": "no_guardar",
            }
            post_data = json.dumps(post_data_json)

        class Route:
            request = Request()

        e = runner.Ejecucion(Path("/fuera"), {"valores": {}})
        e.actual = {"actor": "rrhh", "id": "01"}
        with (
            patch.object(runner, "escribir") as guardar,
            patch.object(runner, "responder_sin_redireccion"),
        ):
            e.filtrar(Route(), "https://127.0.0.1:9443")
            contenido = guardar.call_args.args[2]
        self.assertNotIn("no_guardar", json.dumps(contenido))
        self.assertEqual(contenido["clave_idempotencia"], "clave-123")

    def test_firma_parcial_o_sin_verificador_no_cierra_paso(self):
        e = runner.Ejecucion(Path("/fuera"), {"valores": {}})
        clave = ("POST", "/api/vec/contratacion-temporal/firmas-documento/consultas-v2")
        e.respuestas[clave] = {
            "data": {
                "verificacion_disponible": True,
                "documentos": [
                    {"documento": "resolucion", "completo": True, "pasos": []}
                ],
            }
        }
        self.assertFalse(e.firmas_completas("15"))
        e.respuestas[clave]["data"]["verificacion_disponible"] = False
        self.assertFalse(e.firmas_completas("07b"))

    def test_acto_sin_recibo_no_se_confirma_por_abrir_ficha(self):
        paso = {"http": [{"metodo": "POST"}]}
        with self.assertRaises(runner.PendienteExterno):
            runner.comprobar_recuperacion(
                paso, {"efecto_iniciado": True}, {"recibos": [{"recibo_ref": "ajeno"}]}
            )
        original = {
            "recibos_efecto": [{"recibo_ref": "r", "fecha": "2026-10-14", "version": 1}]
        }
        with self.assertRaises(runner.EntradaInvalida):
            runner.comprobar_recuperacion(
                paso,
                original,
                {"recibos": [{"recibo_ref": "r", "fecha": "2026-10-14", "version": 2}]},
            )
        runner.comprobar_recuperacion(
            paso, original, {"recibos": original["recibos_efecto"]}
        )

    def test_dos_continuaciones_conservan_la_intencion_incierta(self):
        paso = {
            "id": "01",
            "actor": "rrhh",
            "dueno": "CT",
            "http": [{"metodo": "POST"}],
        }
        anterior = {"pasos": [{"id": "01", "estado": "FALLO", "efecto_iniciado": True}]}
        with (
            tempfile.TemporaryDirectory() as carpeta,
            patch("playwright.sync_api.sync_playwright") as pw,
        ):
            browser = (
                pw.return_value.__enter__.return_value.chromium.launch.return_value
            )
            for n, ident in enumerate(
                ({"rrhh": {"origin": "https://127.0.0.1:9443"}}, {})
            ):
                args = SimpleNamespace(
                    salida=Path(carpeta) / str(n),
                    modo="continuar",
                    pasos=None,
                    visible=False,
                )
                with patch("builtins.print"):
                    anterior = runner.ejecutar(
                        args, [paso], {"valores": {}}, ident, anterior
                    )
                self.assertTrue(anterior["pasos"][0]["efecto_iniciado"])
                self.assertEqual(anterior["pasos"][0]["estado"], "PENDIENTE")
            browser.new_context.assert_not_called()

    def test_documentos_vacios_duplicados_o_distintos_no_pasan(self):
        paso = next(
            p for p in runner.catalogo(ROOT / "pasos_hz39.json") if p["id"] == "14"
        )
        with self.assertRaises(runner.EntradaInvalida):
            runner.comprobar_documentos(paso, {}, {})
        documentos = [
            {"documento": a["documento"], "bytes": 10, "sha256": "a" * 64}
            for a in paso["acciones"]
            if a["tipo"] == "descargar"
        ]
        original = {"documentos": documentos}
        runner.comprobar_documentos(paso, original, original)
        alterado = {"documentos": [dict(d) for d in documentos]}
        alterado["documentos"][0]["bytes"] = 11
        with self.assertRaises(runner.EntradaInvalida):
            runner.comprobar_documentos(paso, alterado, original)
        alterado["documentos"][0] = alterado["documentos"][1]
        with self.assertRaises(runner.EntradaInvalida):
            runner.comprobar_documentos(paso, alterado)

    def test_aceptacion_unica_y_recuperacion_descarga_los_mismos_documentos(self):
        pasos = {p["id"]: p for p in runner.catalogo(ROOT / "pasos_hz39.json")}
        acciones_aceptacion = pasos["10"]["acciones"]
        formulario = '[data-portal-mi-bolsa="responder"][data-bolsa="$bolsa_ref"]'
        self.assertEqual(
            [a for a in acciones_aceptacion if a["tipo"] == "click"],
            [{"tipo": "click", "selector": formulario + " [data-portal-iniciar-revision]"}],
        )

        iniciales = [
            (a["documento"], a["selector"])
            for a in pasos["14"]["acciones"] if a["tipo"] == "descargar"
        ]
        recuperacion = pasos["14"]["acciones_recuperacion"]
        self.assertEqual(
            recuperacion[:2],
            [
                {"tipo": "click", "selector": '[data-ct-exp-abrir="$expediente_ref"]'},
                {"tipo": "click", "selector": '[data-ct-exp-vista="documentos"]'},
            ],
        )
        self.assertTrue(all(a["tipo"] == "descargar" for a in recuperacion[2:]))
        self.assertEqual(len(iniciales), 12)
        self.assertEqual(len({nombre for nombre, _ in iniciales}), 12)
        self.assertEqual(
            [(a["documento"], a["selector"]) for a in recuperacion[2:]],
            iniciales,
        )

    def test_ginpix_sigue_alta_b2_y_respeta_dependencias(self):
        pasos = runner.catalogo(ROOT / "pasos_hz39.json")
        por_id = {paso["id"]: paso for paso in pasos}
        cadena = ("15", "17", "18", "16a", "16b", "19", "20", "21")
        ids = [paso["id"] for paso in pasos]
        self.assertEqual(ids[ids.index("15"):ids.index("21") + 1], list(cadena))
        for anterior, siguiente in zip(cadena, cadena[1:]):
            self.assertEqual(por_id[siguiente]["depende_de"], [anterior])

        ficha = por_id["16a"]
        self.assertEqual(
            [(accion["tipo"], accion.get("documento")) for accion in ficha["acciones"][-1:]],
            [("descargar", "ginpix")],
        )
        self.assertEqual(ficha["acciones"][-1], ficha["acciones_recuperacion"][-1])
        self.assertIn("[data-ct-exp-ficha-ginpix-b2]", ficha["acciones"][-1]["selector"])
        self.assertEqual(ficha["http"][0]["metodo"], "GET")
        confirmacion = por_id["16b"]
        self.assertEqual(confirmacion["http"][0]["ruta"],
                         "/api/vec/contratacion-temporal/confirmaciones-ginpix")
        self.assertEqual(confirmacion["acciones"][-1]["selector"],
                         '[data-ct-seg-form="ginpix"] button[type="submit"]')

    def test_candidata_ausente_no_equivale_a_la_misma(self):
        for datos in ({}, {"candidata": "main"}, {"candidata": None}):
            with self.assertRaises(runner.EntradaInvalida):
                runner.candidata(datos)
        self.assertEqual(runner.candidata({"candidata": "a" * 40}), "a" * 40)

    def test_demostracion_opcional_no_bloquea_el_principal(self):
        pasos = [{"id": "01"}, {"id": "e1", "opcional": True}]
        self.assertTrue(
            runner.recorrido_principal_completo(
                pasos, [{"id": "01", "estado": "OK"}, {"id": "e1", "estado": "FALLO"}]
            )
        )
        self.assertFalse(
            runner.recorrido_principal_completo(pasos, [{"id": "e1", "estado": "OK"}])
        )
        self.assertFalse(
            runner.recorrido_principal_completo(
                [{"id": "e1", "opcional": True}], [{"id": "e1", "estado": "OK"}]
            )
        )

    def test_catalogo_y_referencias_faltantes_no_se_inventan(self):
        self.assertEqual(len(runner.catalogo(ROOT / "pasos_hz39.json")), 28)
        with self.assertRaises(runner.EntradaInvalida):
            runner.resolver("$expediente_ref", {})
        with self.assertRaises(runner.EntradaInvalida):
            runner.ruta_local("//cidonia.cloud/api")


if __name__ == "__main__":
    unittest.main()
