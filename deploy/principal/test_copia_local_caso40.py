"""Datos del caso 4.0 en la copia local: guiones, identidad de Antonio y contacto (F1-99)."""
import importlib.util
import json
from pathlib import Path
import shutil
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

spec = importlib.util.spec_from_file_location(
    'caso40', Path(__file__).with_name('copia_local_caso40.py'))
caso40 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(caso40)

REPO = Path(__file__).resolve().parents[2]


class Caso40Test(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.tmp)
        self.repo = self.tmp / 'repo'
        datos = self.repo / caso40.DATOS
        datos.mkdir(parents=True)
        shutil.copy(REPO / caso40.DATOS / 'seleccion.json', datos / 'seleccion.json')
        for nombre in caso40.GUIONES:
            (datos / nombre).write_bytes(b'SELECT 1;\n')
        (self.repo / caso40.REGLAS_BOLSA).parent.mkdir(parents=True)
        (self.repo / caso40.REGLAS_BOLSA).write_text('{}')
        self.material = self.tmp / 'material'
        (self.material / 'identidad').mkdir(parents=True)
        self.raiz = self.tmp / 'raiz'
        self.raiz.mkdir()
        self.copia = SimpleNamespace(nombre='vec-copia-d-prueba', raiz=self.raiz, material=self.material,
                                     leer=Mock(return_value={'puerto_app': 18999}))

    def test_la_seleccion_del_repositorio_es_la_candidatura_vigente(self):
        sel = caso40.seleccion(REPO)
        self.assertTrue(sel['candidato'].startswith('can_Vxd6'))
        self.assertTrue(sel['participacion'].startswith('participacion:'))

    def test_rechaza_referencias_que_saldrian_a_un_argumento_o_a_la_ruta(self):
        ruta = self.repo / caso40.DATOS / 'seleccion.json'
        datos = json.loads(ruta.read_text())
        datos['aspirante']['participacion_ref'] = 'participacion:x/../../admin'
        ruta.write_text(json.dumps(datos))
        with self.assertRaises(ValueError):
            caso40.seleccion(self.repo)

    def test_aplica_los_tres_guiones_con_el_organismo_y_parada_en_error(self):
        with patch.object(caso40.subprocess, 'run') as run:
            caso40.aplicar_guiones(self.copia, self.repo)
        self.assertEqual(run.call_count, 3)
        orden = run.call_args_list[0].args[0]
        self.assertIn('ON_ERROR_STOP=1', orden)
        self.assertIn('organismo=' + caso40.seleccion(REPO)['organismo'], orden)
        self.assertTrue(all(c.kwargs['check'] for c in run.call_args_list))

    def test_antonio_queda_ligado_a_la_candidatura_vigente(self):
        sel = caso40.seleccion(self.repo)

        def preparar(orden, **kwargs):
            self.assertIn('--candidato-ref', orden)
            self.assertEqual(orden[orden.index('--candidato-ref') + 1], sel['candidato'])
            self.assertEqual(kwargs['env']['VEC_CANDIDATE_BOLSA_REF'], sel['bolsa'])
            (self.material / 'identidad/bolsa-candidato.json').write_text(
                json.dumps({'candidato_ref': sel['candidato']}))

        with patch.object(caso40.subprocess, 'run', side_effect=preparar):
            entorno = caso40.preparar_antonio(self.copia, self.repo)
        # F1-109: el selector del portal nunca va sin el catálogo de reglas que
        # el servidor exige; sin él el arranque se para en selectores_bolsa_ct.
        self.assertEqual(entorno, {'VEC_BOLSA_PORTAL_CANDIDATO_ENABLED': 'true',
                                   'VEC_BOLSA_REGLAS_SOURCE_PATH': str(self.repo / caso40.REGLAS_BOLSA)})

    def test_falla_sin_el_catalogo_de_reglas_de_bolsa(self):
        (self.repo / caso40.REGLAS_BOLSA).unlink()

        def preparar(orden, **kwargs):
            (self.material / 'identidad/bolsa-candidato.json').write_text(
                json.dumps({'candidato_ref': caso40.seleccion(self.repo)['candidato']}))

        with patch.object(caso40.subprocess, 'run', side_effect=preparar), self.assertRaises(ValueError):
            caso40.preparar_antonio(self.copia, self.repo)

    def test_el_catalogo_de_reglas_de_bolsa_existe_en_el_repositorio(self):
        self.assertTrue((REPO / caso40.REGLAS_BOLSA).is_file())

    def test_el_entorno_respeta_lo_que_el_kit_ya_fijo(self):
        propuesto = {'VEC_BOLSA_PORTAL_CANDIDATO_ENABLED': 'true', 'VEC_BOLSA_REGLAS_SOURCE_PATH': '/repo/r.json'}
        entorno = caso40.entorno_portal({'VEC_BOLSA_REGLAS_SOURCE_PATH': '/privado/r.json', 'OTRA': '1'}, propuesto)
        self.assertEqual(entorno, {'VEC_BOLSA_REGLAS_SOURCE_PATH': '/privado/r.json', 'OTRA': '1',
                                   'VEC_BOLSA_PORTAL_CANDIDATO_ENABLED': 'true'})

    def test_falla_si_la_identidad_sigue_en_la_candidatura_antigua(self):
        def preparar(orden, **kwargs):
            (self.material / 'identidad/bolsa-candidato.json').write_text(
                json.dumps({'candidato_ref': 'can_otra'}))

        with patch.object(caso40.subprocess, 'run', side_effect=preparar), self.assertRaises(ValueError):
            caso40.preparar_antonio(self.copia, self.repo)

    def test_el_contacto_es_ficticio_y_va_al_buzon_de_la_copia(self):
        ruta, cuerpo = caso40.peticion_contacto(caso40.seleccion(self.repo))
        self.assertTrue(ruta.endswith('/datos-contacto'))
        datos = json.loads(cuerpo)
        self.assertTrue(datos['correo'].endswith('@example.test'))
        self.assertNotIn('ejemplo', json.dumps(datos, ensure_ascii=False).lower())

    def test_registrar_contacto_acepta_la_repeticion_y_rechaza_otros_estados(self):
        for estado, falla in ((201, False), (200, False), (503, True)):
            conexion = Mock()
            conexion.getresponse.return_value = Mock(status=estado)
            with self.subTest(estado=estado), \
                    patch.object(caso40.ssl, 'create_default_context'), \
                    patch.object(caso40.http.client, 'HTTPSConnection', return_value=conexion):
                if falla:
                    with self.assertRaises(ValueError):
                        caso40.registrar_contacto(self.copia, self.repo)
                else:
                    self.assertEqual(caso40.registrar_contacto(self.copia, self.repo), estado)
                cabeceras = conexion.request.call_args.kwargs['headers']
                self.assertEqual(cabeceras['Idempotency-Key'], caso40.CLAVE_CONTACTO)
                conexion.close.assert_called_once()


if __name__ == '__main__':
    unittest.main()
