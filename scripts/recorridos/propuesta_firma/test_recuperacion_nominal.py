"""Prueba del transporte sintético: no valida semántica Go ni firma documentos."""
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import unittest
from unittest import mock

import recorrer as r
import recuperacion_nominal as nominal
from test_recorrer_v2 import material, consulta_tecnica


def datos_nominales(dos=False):
    _, primero = material()
    firmas = [primero]
    if dos:
        segundo = copy.deepcopy(primero)
        segundo.update(firma_ref="firma:dos", recibo_ref="recibo:dos", paso_orden=2,
                       paso_ref="paso:dos", secuencia=2)
        segundo["documento_custodiado"].update(version=2, huella_sha256="1" * 64)
        segundo["verificacion_tecnica"]["firmado_sha256"] = "1" * 64
        segundo["revision_pdf"].update(orden_firma=2, entrada_sha256="b" * 64,
                                     revision_sha256="1" * 64, evidencia_sha256="2" * 64)
        firmas.append(segundo)
    informe, tecnica = consulta_tecnica(firmas)
    datos = copy.deepcopy(tecnica)
    datos.update(esquema="vec.contratacion-temporal.recuperacion-firmas-r5.v2",
                 recuperacion="recuperada", campos_no_disponibles=[], recuperaciones=[])
    # Canon únicamente de transporte; la semántica histórica corresponde al
    # servidor Go. Unicode demuestra que se calcula sobre bytes, no caracteres.
    for i, firma in enumerate(firmas):
        canon = json.dumps({"dato_sintetico": "á" * 512, "firma": i}, ensure_ascii=False)
        datos["recuperaciones"].append({"firma_ref": firma["firma_ref"],
            "material_root_sha256": firma["material_root_sha256"], "canon_nominal": canon,
            "canon_nominal_sha256": hashlib.sha256(canon.encode()).hexdigest(),
            "canon_nominal_ref": "evidencia:competencia-firmante-ct:" + str(i + 1) * 64})
    return informe, tecnica, datos, firmas


class TransporteNominalTest(unittest.TestCase):
    def validar(self, datos, informe, firmas):
        return nominal.validar(datos, informe, firmas, r.validar_consulta_tecnica_v2, r.Corte)

    def test_una_y_dos_firmas_conservan_solo_metadatos(self):
        for dos in (False, True):
            informe, _, datos, firmas = datos_nominales(dos)
            resumen = self.validar(datos, informe, firmas)
            self.assertEqual(len(resumen), len(firmas))
            self.assertTrue(all(set(f) == nominal.CAMPOS_RESUMEN for f in resumen))
            informe["recuperacion_nominal_v2"] = resumen
            self.assertNotIn("recuperacion_nominal_v2", r.informe_publico(informe))
            self.assertNotIn("dato_sintetico", json.dumps(resumen))
            self.assertNotEqual(resumen[0]["canon_nominal_ref"].split(":")[-1],
                                resumen[0]["canon_nominal_sha256"])

    def test_canon_root_firmas_y_contrato_alterados_se_rechazan(self):
        informe, _, datos, firmas = datos_nominales(True)
        cambios = [lambda d: d.update(esquema="otro"),
            lambda d: d.update(recuperacion="parcial"), lambda d: d.update(firma_eficaz=True),
            lambda d: d.update(campos_no_disponibles=["canon_nominal"]),
            lambda d: d.update(extra="no-propagable"),
            lambda d: d["recuperaciones"][0].update(canon_nominal="alterado"),
            lambda d: d["recuperaciones"][0].update(material_root_sha256="0" * 64),
            lambda d: d["recuperaciones"][0].update(canon_nominal_ref="evidencia:otra"),
            lambda d: d["recuperaciones"][0].update(firma_ref="firma:ajena"),
            lambda d: d["recuperaciones"].pop(),
            lambda d: d["recuperaciones"].__setitem__(1, copy.deepcopy(d["recuperaciones"][0])),
            lambda d: d["recuperaciones"][0].update(extra="no-propagable")]
        for cambio in cambios:
            copia = copy.deepcopy(datos)
            cambio(copia)
            with self.assertRaises(r.Corte):
                self.validar(copia, informe, firmas)

    def test_limites_son_bytes_utf8_exactos(self):
        informe, _, datos, firmas = datos_nominales()
        for canon in ("x" * 511, "á" * 16385, "\ud800" * 512):
            copia = copy.deepcopy(datos)
            copia["recuperaciones"][0].update(canon_nominal=canon,
                canon_nominal_sha256=hashlib.sha256(canon.encode(errors="replace")).hexdigest())
            with self.assertRaises(r.Corte):
                self.validar(copia, informe, firmas)
        for canon in ("x" * 512, "á" * 16384):
            copia = copy.deepcopy(datos)
            copia["recuperaciones"][0].update(canon_nominal=canon,
                canon_nominal_sha256=hashlib.sha256(canon.encode()).hexdigest())
            self.validar(copia, informe, firmas)

    def test_http_no_tiene_fallback_y_reutiliza_solicitud_real(self):
        informe, _, datos, firmas = datos_nominales()
        pagina = mock.Mock()
        for estado in (403, 404, 503):
            pagina.evaluate.return_value = {"status": estado, "data": None}
            pagina.evaluate.reset_mock()
            with self.assertRaises(r.Corte):
                r.consultar_recuperacion_nominal_v2(pagina, informe, firmas)
            pagina.evaluate.assert_called_once()
            args = pagina.evaluate.call_args.args[1]
            self.assertEqual(args[0], nominal.RUTA)
            self.assertEqual(args[2], 8 << 20)
            self.assertEqual(args[1], r.solicitud_consulta_tecnica_v2(informe, firmas))
        pagina.evaluate.return_value = {"status": 200, "data": {"data": datos}}
        self.assertEqual(r.consultar_recuperacion_nominal_v2(pagina, informe, firmas),
                         self.validar(datos, informe, firmas))

    def test_transporte_corta_cabeceras_y_cuerpo_que_no_terminan(self):
        pagina = mock.Mock()
        pagina.evaluate.return_value = {"status": 200, "data": {"data": {}}}
        r.consultar_json_firmas_v2(pagina, nominal.RUTA, {}, nominal.MAX_RESPUESTA)
        fuente = pagina.evaluate.call_args.args[0]
        # Ejecuta el JS real sin red. Sólo se acelera el reloj; fetch retiene
        # cabeceras o cuerpo hasta que recibe la cancelación del consumidor.
        guion = r"""
        const ejecutar = (0,eval)(process.argv[1]);
        const reloj = globalThis.setTimeout;
        globalThis.setTimeout = (fn,ms) => {
          if(ms!==30000)throw Error('plazo_incorrecto');
          return reloj(fn,5);
        };
        const resultados=[];
        for(const fase of ['cabeceras','cuerpo']){
          let cancelada=false;
          globalThis.fetch=async (_,opciones)=>{
            const espera=()=>new Promise((_,rechazar)=>
              opciones.signal.addEventListener('abort',()=>{
                cancelada=true;rechazar(Error('abortada'));
              },{once:true}));
            if(fase==='cabeceras')return espera();
            return {status:200,headers:{get:()=> 'application/json'},body:{getReader:()=>({
              read:espera,cancel:async()=>{}
            })}};
          };
          const resultado=await ejecutar(['ruta_local',{},8<<20]);
          resultados.push({fase,cancelada,resultado});
        }
        process.stdout.write(JSON.stringify(resultados));
        """
        resultado = subprocess.run(["node", "--input-type=module", "-e", guion, fuente],
                                   capture_output=True, text=True, timeout=5, check=True)
        for caso in json.loads(resultado.stdout):
            self.assertTrue(caso["cancelada"])
            self.assertEqual(caso["resultado"], {"status": 0, "data": None})

    def test_reinicio_compara_baseline_y_no_adopta_informes_antiguos(self):
        previo, tecnica, datos, firmas = datos_nominales(True)
        contexto = {"expediente_ref": "expediente:sintetico", "documento": "informe_definitivo",
                    "binario_sha256": "a" * 64, "propuesta": {"version_actual": 7}, "pdf": {}}
        baseline = self.validar(datos, previo, firmas)
        previo.update(contexto, estado="COMPLETO", registro_incierto=False,
                      pdf_firmado={"sha256": "1" * 64}, consulta_tecnica_v2=tecnica,
                      recuperacion_nominal_v2=baseline)
        acta = {"expediente_ref": contexto["expediente_ref"], "aplicacion_reiniciada": True,
                "postgresql_reiniciado": True, "instante_utc": "2026-10-03T13:00:00Z"}
        a = mock.Mock(comparar=Path("segunda.json"), reinicio=Path("reinicio.json"),
                      documento="informe_definitivo", recuperacion_nominal=True)
        for anterior in (baseline, None, [{**baseline[0], "canon_nominal_sha256": "0" * 64}]):
            previo["recuperacion_nominal_v2"] = anterior
            informe = copy.deepcopy(contexto)
            with mock.patch.object(r, "leer_informe_privado", side_effect=[previo, acta]), \
                 mock.patch.object(r, "consultar_estado_v2", return_value={}) as consulta, \
                 mock.patch.object(r, "comparar_estado_v2"), \
                 mock.patch.object(r, "descargar_revision_v2", return_value=previo["pdf_firmado"]), \
                 mock.patch.object(r, "consultar_metadatos_v2", return_value=tecnica), \
                 mock.patch.object(r, "consultar_recuperacion_nominal_v2", return_value=baseline):
                if anterior != baseline:
                    with self.assertRaises(r.Corte):
                        r.recorrer_firmas_v2(None, a, informe, {}, TimeoutError)
                    if anterior is None:
                        consulta.assert_not_called()
                else:
                    r.recorrer_firmas_v2(None, a, informe, {}, TimeoutError)
                    self.assertEqual(informe["estado"], "RECUPERACION_NOMINAL_CONFIRMADA")
                    self.assertFalse(informe["e2e"])
                    self.assertFalse(informe["verificacion_criptografica_repetida"])
                    self.assertEqual(informe["recuperacion_nominal_v2"], baseline)


if __name__ == "__main__":
    unittest.main()
