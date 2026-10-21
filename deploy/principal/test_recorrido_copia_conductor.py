"""Orden de efectos y rechazo previo de la fábrica del recorrido."""
import argparse
import importlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import types
import unittest
from unittest.mock import Mock, patch

from recorrido_copia import conductor


class ConductorTest(unittest.TestCase):
    def test_documentos_y_borradores_se_activan_antes_del_arranque_de_firma(self):
        from recorrido_copia import (catalogo, roles_admin, corporativo, certificados_base,
            firmantes, montajes_firma, actos, plan, admisiones, configuracion_firma,
            documentos_base, documentos, personal, caso_bolsa, dietas)
        from contextlib import ExitStack
        with tempfile.TemporaryDirectory() as temporal, ExitStack() as pila:
            raiz = Path(temporal)
            copia = Mock(raiz=raiz, estado=raiz / 'estado.json')
            copia.entorno_modulos.return_value = {'VARIABLE_PREVIA': 'conservada'}
            copia.leer.return_value = {'revision': 'candidata', 'sid_copia': '12345678901234567'}
            contexto = Mock(carpeta=raiz)
            contexto.leer.return_value = {'version': 1}
            preparar_dietas = pila.enter_context(patch.object(dietas, 'preparar',
                return_value={'activada': False}))
            pila.enter_context(patch.object(conductor, 'Contexto', return_value=contexto))
            for funcion in ('_parametros', '_organizacion', '_verificador', 'provisionar_perfiles'):
                pila.enter_context(patch.object(conductor, funcion))
            for modulo, funcion in [(m, 'preparar') for m in (catalogo, roles_admin,
                    corporativo, certificados_base, firmantes, montajes_firma, admisiones,
                    configuracion_firma, documentos_base, personal, caso_bolsa)] + [
                    (actos, 'preparar_actos'), (plan, 'preparar_plan'),
                    (documentos, 'preparar_documentos')]:
                pila.enter_context(patch.object(modulo, funcion, return_value={}))
            def comprobar(manifiesto):
                entorno = json.loads((raiz / 'entorno_modulos.json').read_bytes())
                self.assertEqual(entorno, {'VARIABLE_PREVIA': 'conservada',
                    'VEC_DOCUMENTOS_ENABLED': 'true',
                    'VEC_CT_PLANTILLAS_DOCUMENTAL_ENABLED': 'true'})
                estado = json.loads(copia.estado.read_bytes())
                from copia_local import sha256
                self.assertEqual(estado['entorno_modulos_sha256'], sha256(raiz / 'entorno_modulos.json'))
            copia.firma.side_effect = comprobar
            conductor.preparar(copia, argparse.Namespace(puerto_verificador=19881))
            copia.firma.assert_called_once()
            preparar_dietas.assert_not_called()
            conductor.preparar(copia, argparse.Namespace(puerto_verificador=19881, dietas=True))
            preparar_dietas.assert_called_once_with(contexto)

    def test_fallo_en_acto_detiene_plan_firma_y_provision(self):
        eventos = []
        etapas = {'catalogo': 'preparar', 'roles_admin': 'preparar', 'corporativo': 'preparar', 'certificados_base': 'preparar',
                  'firmantes': 'preparar', 'montajes_firma': 'preparar',
                  'actos': 'preparar_actos', 'plan': 'preparar_plan', 'admisiones': 'preparar',
                  'configuracion_firma': 'preparar'}
        modulos = {}
        for nombre, funcion in etapas.items():
            modulo = types.ModuleType('recorrido_copia.' + nombre)
            def ejecutar(nombre=nombre):
                def llamada(c):
                    eventos.append(nombre)
                    if nombre == 'actos':
                        raise ValueError('Respuesta incierta; conservar recibo')
                    return {'confirmado': True}
                return llamada
            setattr(modulo, funcion, ejecutar())
            modulos[modulo.__name__] = modulo
        paquete = importlib.import_module('recorrido_copia')
        atributos = {n: modulos['recorrido_copia.' + n] for n in etapas}
        with patch.dict(sys.modules, modulos), patch.multiple(paquete, create=True, **atributos), \
             patch.object(conductor, 'Contexto') as contexto, \
             patch.object(conductor, '_parametros', return_value={}), \
             patch.object(conductor, '_organizacion', side_effect=lambda c: eventos.append('organizacion_historica')), \
             patch.object(conductor, 'provisionar_perfiles') as provisionar:
            copia = Mock(nombre='vec-copia-l-contrato')
            contexto.return_value.carpeta = Path('/tmp/recorrido')
            with self.assertRaisesRegex(ValueError, 'incierta'):
                conductor.preparar(copia, argparse.Namespace(puerto_verificador=19881, config_personal=None))
            self.assertEqual(eventos, ['catalogo', 'roles_admin', 'corporativo', 'organizacion_historica', 'certificados_base', 'firmantes', 'montajes_firma', 'actos'])
            copia.firma.assert_not_called()
            provisionar.assert_not_called()

    def test_no_aprueba_perfiles_sin_escucha_confirmada(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            (raiz / 'app.log').write_text('error al componer\n')
            copia = Mock(raiz=raiz)
            with self.assertRaisesRegex(ValueError, 'no confirmó'):
                conductor.provisionar_perfiles(copia)
            copia.parar.assert_not_called()

    def test_creacion_completa_rechaza_material_heredado_antes_de_efectos(self):
        guion = Path(__file__).with_name('copia_local.py')
        resultado = subprocess.run([sys.executable, str(guion), 'crear', '--equipo', 'l',
            '--instancia', 'rechazo', '--recorrido-completo', '--rrhh-original', '/tmp/ajeno'],
            capture_output=True, check=False)
        self.assertEqual(resultado.returncode, 2)
        self.assertIn(b'requiere crear con material nuevo', resultado.stderr)

    def test_puerto_verificador_no_puede_coincidir_con_administracion(self):
        guion = Path(__file__).with_name('copia_local.py')
        resultado = subprocess.run([sys.executable, str(guion), 'crear', '--equipo', 'l',
            '--instancia', 'rechazo', '--recorrido-completo', '--puerto-verificador', '19481'],
            capture_output=True, check=False)
        self.assertEqual(resultado.returncode, 2)
        self.assertIn(b'puerto del verificador debe ser distinto', resultado.stderr)


if __name__ == '__main__':
    unittest.main()
