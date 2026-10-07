"""Contratos sintéticos del driver; estas pruebas no ejecutan el circuito VEC."""
import copy
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock
from contextlib import redirect_stdout

import recorrer as r


def material():
    solicitud = {"expediente_ref": "expediente:sintetico", "version_expediente": 7,
                 "documento": "informe_definitivo", "paso_orden": 1,
                 "original_sha256": "a" * 64, "entrada_sha256": "a" * 64}
    recibo = {"esquema": "vec.contratacion-temporal.registro-firma-vec.v2", "recibo_ref": "recibo:uno",
              "firma_ref": "firma:uno", "ya_registrada": False, "expediente_ref": "expediente:sintetico",
              "version_expediente": 7, "documento": "informe_definitivo", "paso_orden": 1,
              "paso_ref": "paso:uno", "secuencia": 1, "registrada_en": "2026-10-03T10:00:00Z",
              "documento_custodiado": {"expediente_ref": "ref:" + "c" * 64, "documento_ref": "ref:" + "d" * 64,
                                       "version": 1, "huella_sha256": "b" * 64},
              "verificacion_tecnica": {"estado": "valida", "motivo": "verificada", "politica": "politica:v2",
                                       "revocacion": "vigente", "sello_tiempo": "no_presente",
                                       "original_sha256": "a" * 64, "firmado_sha256": "b" * 64},
              "firma_eficaz": False, "material_root_sha256": "e" * 64,
              "revision_pdf": {"orden_firma": 1, "entrada_sha256": "a" * 64,
                               "revision_sha256": "b" * 64, "evidencia_sha256": "f" * 64}}
    return solicitud, recibo


def estado(recibo):
    return {"firma_eficaz": False, "documentos": [{"documento": "informe_definitivo", "paso_pendiente": 2,
        "pasos": [{"orden": 1, "estado": "firmado", "recibo_ref": recibo["recibo_ref"],
                   "registrada_en": recibo["registrada_en"], "documento_custodiado": recibo["documento_custodiado"]}]}]}


def consulta_tecnica(firmas):
    informe={"firmas_v2":firmas,"intento_v2":{"expediente_ref":"expediente:sintetico","version_expediente":7,
        "documento":"informe_definitivo","paso_orden":firmas[-1]["paso_orden"],"original_ref":"ref:"+"a"*64,
        "original_version":7,"clave_idempotencia":"clave-conservada-sintetica", "firmado_sha256":firmas[-1]["documento_custodiado"]["huella_sha256"]},
        "preflight_v2":{"original_ref":"ref:"+"a"*64,"original_version":7,"catalogo_ref":"catalogo:v2","catalogo_huella":"c"*64}}
    datos={"esquema":"vec.contratacion-temporal.consulta-firmas-r5.v2","expediente_ref":"expediente:sintetico",
        "version_expediente":7,"documento":"informe_definitivo","historia_revision":len(firmas),"historia_sha256":"d"*64,
        "firmas":[],"recuperacion":"parcial","campos_no_disponibles":["material_root_sha256","canon_nominal"],"firma_eficaz":False}
    for i,rec in enumerate(firmas):
        original={"documento_ref":"ref:"+"a"*64,"version":7,"huella_sha256":rec["verificacion_tecnica"]["original_sha256"]}
        entrada=original if i==0 else {k:firmas[i-1]["documento_custodiado"][k] for k in ("documento_ref","version","huella_sha256")}
        longitud=300 if i==0 else 500;entrada_longitud=100 if i==0 else 300
        datos["firmas"].append({k:rec[k] for k in ("firma_ref","recibo_ref","registrada_en","secuencia","paso_orden","paso_ref","version_expediente")} | {
            "via":"certificado_vec","resultado":"firmado","catalogo_ref":"catalogo:v2","catalogo_huella":"c"*64,
            "original":original,"documento_custodiado":rec["documento_custodiado"],"revision_pdf":{
                "orden_firma":rec["paso_orden"],"firma_anterior_ref":firmas[i-1]["firma_ref"] if i else "",
                "recibo_anterior_ref":firmas[i-1]["recibo_ref"] if i else "","entrada_documento":entrada,"entrada_longitud":entrada_longitud,
                "byte_range":[0,entrada_longitud,longitud-100,100],"revision_sha256":rec["revision_pdf"]["revision_sha256"],
                "contenido_firmado_sha256":"e"*64,"revision_longitud":longitud,"evidencia_firmas_sha256":rec["revision_pdf"]["evidencia_sha256"]}})
    return informe,datos


class ConsultaTecnicaV2Test(unittest.TestCase):
    def test_solicitud_reutiliza_clave_privada_sin_otros_campos(self):
        _,rec=material();informe,datos=consulta_tecnica([rec])
        solicitud=r.solicitud_consulta_tecnica_v2(informe,[rec])
        self.assertEqual(solicitud["clave_idempotencia"],informe["intento_v2"]["clave_idempotencia"])
        self.assertEqual(set(solicitud),{"expediente_ref","version_expediente","documento","paso_orden","clave_idempotencia","catalogo_huella","via"})
        informe["consulta_tecnica_v2"]=r.validar_consulta_tecnica_v2(datos,informe,[rec])
        self.assertNotIn("clave-conservada-sintetica",json.dumps(r.informe_publico(informe)))
        del informe["intento_v2"]
        with self.assertRaises(r.Corte):r.solicitud_consulta_tecnica_v2(informe,[rec])

    def test_respuesta_cruzada_alterada_y_extra_se_rechazan(self):
        _,rec=material();informe,datos=consulta_tecnica([rec])
        self.assertEqual(r.validar_consulta_tecnica_v2(datos,informe,[rec]),datos)
        mutaciones=[lambda d:d.update(expediente_ref="expediente:ajeno"),lambda d:d.update(documento="resolucion"),
            lambda d:d.update(campo_extra="no-propagable"),lambda d:d["firmas"][0].update(recibo_ref="recibo:ajeno"),
            lambda d:d["firmas"][0]["revision_pdf"].update(evidencia_firmas_sha256="0"*64),
            lambda d:d["firmas"][0]["revision_pdf"].update(byte_range=[0,100,200,99]),
            lambda d:d["firmas"][0]["revision_pdf"].update(canon_nominal="no-propagable"),
            lambda d:d.update(recuperacion="completa"),lambda d:d.update(campos_no_disponibles=[])]
        for mutacion in mutaciones:
            copia=copy.deepcopy(datos);mutacion(copia)
            with self.assertRaises(r.Corte):r.validar_consulta_tecnica_v2(copia,informe,[rec])
        copia=copy.deepcopy(datos);copia["firmas"]=[]
        with self.assertRaises(r.Corte) as corte:r.validar_consulta_tecnica_v2(copia,informe,[rec])
        self.assertEqual(corte.exception.paso,"consulta_v2_pendiente")

    def test_consulta_ausente_no_usa_fallback(self):
        _,rec=material();informe,datos=consulta_tecnica([rec])
        pagina=mock.Mock();pagina.evaluate.return_value={"status":404,"data":None}
        with self.assertRaises(r.Corte) as corte:r.consultar_metadatos_v2(pagina,informe,[rec])
        self.assertEqual(corte.exception.paso,"consulta_v2_pendiente")
        self.assertEqual(pagina.evaluate.call_count,1)
        args=pagina.evaluate.call_args.args[1]
        self.assertEqual(args[0],r.RUTA_CONSULTA_TECNICA_V2)
        self.assertEqual(args[1]["clave_idempotencia"],informe["intento_v2"]["clave_idempotencia"])
        pagina.evaluate.return_value={"status":200,"data":{"data":datos}}
        self.assertEqual(r.consultar_metadatos_v2(pagina,informe,[rec]),datos)


class ContratosV2Test(unittest.TestCase):
    def test_recibo_exactamente_ligado_a_revision_original_y_bytes(self):
        solicitud, recibo = material()
        self.assertEqual(r.recibo_v2(recibo, solicitud, "b" * 64, 201), recibo)
        replay = {**recibo, "ya_registrada": True}
        self.assertEqual(r.recibo_v2(replay, solicitud, "b" * 64, 200), replay)
        for cambio in ({"firma_eficaz": True}, {"version_expediente": 8}, {"extra": "dato"},
                       {"ya_registrada": True}, {"paso_orden": 2}, {"secuencia": True}):
            with self.subTest(cambio=cambio), self.assertRaises(r.Corte):
                r.recibo_v2({**recibo, **cambio}, solicitud, "b" * 64, 201)
        with self.assertRaises(r.Corte):
            r.recibo_v2(recibo, solicitud, "0" * 64, 201)

    def test_segunda_revision_no_admite_el_original_como_entrada(self):
        solicitud, recibo = material()
        solicitud.update(paso_orden=2, entrada_sha256="b" * 64)
        recibo.update(paso_orden=2)
        recibo["revision_pdf"]["orden_firma"] = 2
        with self.assertRaises(r.Corte):
            r.recibo_v2(recibo, solicitud, "b" * 64, 201)
        recibo["revision_pdf"]["entrada_sha256"] = "b" * 64
        r.recibo_v2(recibo, solicitud, "b" * 64, 201)

    def test_rechaza_dictamen_no_valido_y_custodia_ajena(self):
        solicitud, recibo = material()
        for campo, cambio in (("verificacion_tecnica", {"revocacion": "no_comprobada"}),
                              ("verificacion_tecnica", {"estado": "indeterminada"}),
                              ("revision_pdf", {"evidencia_sha256": ""}),
                              ("documento_custodiado", {"huella_sha256": "0" * 64})):
            copia = copy.deepcopy(recibo)
            copia[campo].update(cambio)
            with self.subTest(campo=campo, cambio=cambio), self.assertRaises(r.Corte):
                r.recibo_v2(copia, solicitud, "b" * 64, 201)

    def test_consulta_revalida_custodia_fecha_orden_y_unicidad(self):
        _, recibo = material()
        consulta = estado(recibo)
        r.comparar_estado_v2(consulta, [recibo], "informe_definitivo")
        for campo, valor in (("registrada_en", "2026-10-03T11:00:00Z"), ("orden", 2), ("estado", "pendiente_firma")):
            copia = copy.deepcopy(consulta)
            copia["documentos"][0]["pasos"][0][campo] = valor
            with self.subTest(campo=campo), self.assertRaises(r.Corte):
                r.comparar_estado_v2(copia, [recibo], "informe_definitivo")
        consulta["documentos"][0]["pasos"] *= 2
        with self.assertRaises(r.Corte):
            r.comparar_estado_v2(consulta, [recibo], "informe_definitivo")

    def test_continuacion_bloquea_post_incierto_o_mismo_certificado(self):
        _, recibo = material()
        actual = {"expediente_ref": "expediente:sintetico", "documento": "informe_definitivo",
                  "binario_sha256": "a" * 64, "propuesta": {"version_actual":7}, "pdf": {}}
        previo = {**actual, "estado": "PRIMERA_FIRMA_CONFIRMADA", "registro_incierto": False,
                  "firmas_v2": [recibo], "canal_certificado_sha256": "b" * 64}
        self.assertEqual(r.validar_continuacion(previo, actual, "c" * 64), [recibo])
        for cambios, canal in (({"registro_incierto": True}, "c" * 64), ({}, "b" * 64),
                               ({"estado": "CORTE"}, "c" * 64), ({"binario_sha256": "d" * 64}, "c" * 64)):
            with self.subTest(cambios=cambios, canal=canal), self.assertRaises(r.Corte):
                r.validar_continuacion({**previo, **cambios}, actual, canal)

    def test_preflight_segundo_exige_revision_distinta(self):
        solicitud = {"version": 7, "documento": "informe_definitivo", "original_ref": "ref:" + "a" * 64,
                     "original_version": 7, "catalogo_ref": "catalogo:v2", "catalogo_huella": "b" * 64}
        datos = {"esquema": "vec.contratacion-temporal.preflight-firma.v2", "version_expediente": 7,
                 "documento": "informe_definitivo", "catalogo_ref": "catalogo:v2", "catalogo_huella": "b" * 64,
                 "paso_pendiente": 2, "original_ref": solicitud["original_ref"], "original_version": 7,
                 "vias_disponibles": ["certificado_vec"], "entrada_documento_ref": "ref:" + "c" * 64,
                 "entrada_documento_version": 1, "entrada_documento_sha256": "d" * 64}
        r.preflight_v2(datos, solicitud)
        with self.assertRaises(r.Corte):
            r.preflight_v2({**datos, "entrada_documento_ref": datos["original_ref"]}, solicitud)
        with self.assertRaises(r.Corte):
            r.preflight_v2({**datos, "esquema": "vec.contratacion-temporal.preflight-firma.v1"}, solicitud)

    def test_recibos_externos_rechazan_clave_extra_antes_de_copiarlos(self):
        _, recibo = material()
        actual = {"expediente_ref":"expediente:sintetico","documento":"informe_definitivo",
                  "binario_sha256":"a"*64,"propuesta":{"version_actual":7},"pdf":{}}
        previo = {**actual,"estado":"PRIMERA_FIRMA_CONFIRMADA","registro_incierto":False,
                  "firmas_v2":[recibo],"canal_certificado_sha256":"b"*64}
        for campo in (None,"documento_custodiado","verificacion_tecnica","revision_pdf"):
            copia=copy.deepcopy(previo)
            destino=copia["firmas_v2"][0] if campo is None else copia["firmas_v2"][0][campo]
            destino["clave_extra"]="material-sintetico-no-propagable"
            with self.subTest(campo=campo),self.assertRaises(r.Corte):
                r.validar_continuacion(copia,actual,"c"*64)
            with self.subTest(lectura=campo),self.assertRaises(r.Corte):
                r.recibos_guardados_v2(copia,1)
            publico=r.informe_publico({**copia,"clave_extra":"top-no-propagable"})
            self.assertNotIn("no-propagable",json.dumps(publico))
            self.assertNotIn("clave_extra",json.dumps(publico))

    def test_reinicio_externo_no_se_presenta_como_observado(self):
        acta = {"expediente_ref": "expediente:sintetico", "aplicacion_reiniciada": True,
                "postgresql_reiniciado": True, "instante_utc": "2026-10-03T13:00:00Z"}
        self.assertEqual(r.comprobar_reinicio(acta, {"expediente_ref": acta["expediente_ref"]})["origen_evidencia"],
                         "declaracion_externa")
        with self.assertRaises(r.Corte):
            r.comprobar_reinicio({**acta, "postgresql_reiniciado": False}, {"expediente_ref": acta["expediente_ref"]})

    def test_recuperacion_queda_parcial_por_falta_de_evidencia_http_v2(self):
        _, recibo = material()
        segundo = copy.deepcopy(recibo)
        segundo.update(paso_orden=2, recibo_ref="recibo:dos", firma_ref="firma:dos", secuencia=2)
        segundo["revision_pdf"].update(orden_firma=2,entrada_sha256="b"*64,revision_sha256="0"*64)
        segundo["verificacion_tecnica"]["firmado_sha256"]="0"*64
        segundo["documento_custodiado"].update(version=2,huella_sha256="0"*64)
        informe = {"expediente_ref":"expediente:sintetico", "documento":"informe_definitivo",
                   "binario_sha256":"a"*64,"propuesta":{"version_actual":7},"pdf":{}}
        previo = {**informe, "estado":"COMPLETO", "registro_incierto":False,
                  "firmas_v2":[recibo,segundo], "pdf_firmado":{"sha256":"b"*64}}
        privado,tecnico=consulta_tecnica([recibo,segundo])
        previo.update(privado,consulta_tecnica_v2=tecnico)
        a = mock.Mock(comparar=Path("segunda.json"),reinicio=Path("reinicio.json"),documento="informe_definitivo")
        acta = {"expediente_ref":informe["expediente_ref"],"aplicacion_reiniciada":True,
                "postgresql_reiniciado":True,"instante_utc":"2026-10-03T13:00:00Z"}
        with mock.patch.object(r,"leer_informe_privado",side_effect=[previo,acta]), \
             mock.patch.object(r,"consultar_estado_v2",return_value={}), \
             mock.patch.object(r,"comparar_estado_v2"), \
             mock.patch.object(r,"descargar_revision_v2",return_value=previo["pdf_firmado"]), \
             mock.patch.object(r,"consultar_metadatos_v2",return_value=tecnico):
            r.recorrer_firmas_v2(None,a,informe,{},TimeoutError)
        self.assertEqual(informe["estado"],"RECUPERACION_PARCIAL")
        self.assertFalse(informe["e2e"])
        self.assertFalse(informe["verificacion_criptografica_repetida"])
        self.assertEqual(informe["pendiente"],"material_root_sha256_y_canon_nominal_no_expuestos")
        self.assertEqual(informe["campos_no_revalidados_por_consulta"],["material_root_sha256","canon_nominal"])
        self.assertEqual(informe["consulta_tecnica_v2"],tecnico)

    def test_fallo_externo_conserva_operacion_incierta(self):
        with tempfile.TemporaryDirectory() as tmp:
            salida = Path(tmp)/"estado.json"
            def recorrido(a,*_):
                a.guardar_progreso({"estado":"CORTE","registro_incierto":True,"intento_v2":{"clave_idempotencia":"solo-privada"}})
                raise RuntimeError("diagnostico_no_publicable")
            with mock.patch.object(r,"validar_entrada",return_value=("chrome",{})), \
                 mock.patch.object(r,"recorrer",side_effect=recorrido),redirect_stdout(io.StringIO()) as texto:
                self.assertEqual(r.main(["--salida",str(salida)]),2)
                self.assertNotIn("solo-privada",texto.getvalue())
                self.assertNotIn("diagnostico_no_publicable",texto.getvalue())
            self.assertTrue(json.loads(salida.read_text())["registro_incierto"])

    def test_main_reserva_salida_antes_de_red_y_no_imprime_clave(self):
        with tempfile.TemporaryDirectory() as tmp:
            salida = Path(tmp) / "estado.json"
            def recorrido(a, *_):
                self.assertTrue(salida.exists())
                a.guardar_progreso({"estado": "CORTE", "registro_incierto": True,
                                    "intento_v2": {"clave_idempotencia": "clave-sintetica-privada"}})
                self.assertTrue(json.loads(salida.read_text())["registro_incierto"])
                return {"estado": "PRIMERA_FIRMA_CONFIRMADA", "intento_v2": {"clave_idempotencia": "clave-sintetica-privada"}}
            with mock.patch.object(r, "validar_entrada", return_value=("chrome", {})), \
                 mock.patch.object(r, "recorrer", side_effect=recorrido), redirect_stdout(io.StringIO()) as texto:
                self.assertEqual(r.main(["--salida", str(salida)]), 0)
                self.assertNotIn("clave-sintetica-privada", texto.getvalue())
            with mock.patch.object(r, "validar_entrada", return_value=("chrome", {})), \
                 mock.patch.object(r, "recorrer") as red, redirect_stdout(io.StringIO()):
                self.assertEqual(r.main(["--salida", str(salida)]), 2)
                red.assert_not_called()


if __name__ == "__main__":
    unittest.main()
