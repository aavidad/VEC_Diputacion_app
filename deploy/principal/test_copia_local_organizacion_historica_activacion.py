"""La preparación OH nunca reejecuta una preparación interrumpida."""

from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import json

from copia_local_organizacion_historica_activacion import activar
from copia_local import guardar, sha256


class ActivacionOHTest(unittest.TestCase):
    def test_sin_modulos_no_prepara_permisos_ni_compila(self):
        copia = SimpleNamespace(leer=lambda: {'preparada': True})
        with patch('copia_local_organizacion_historica_activacion.subprocess.check_output') as compilar:
            with self.assertRaisesRegex(ValueError, 'copia interna'):
                activar(copia, Path('/fuente'))
            compilar.assert_not_called()

    def test_marca_previa_o_interrupcion_exigen_conciliar_sin_nuevos_efectos(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            estado = {'preparada': True, 'entorno_modulos_sha256': 'a' * 64}
            copia = SimpleNamespace(raiz=raiz, leer=lambda: estado, existe=lambda: True)
            for guardada in (False, True):
                with self.subTest(estado_guardado=guardada):
                    if guardada:
                        estado['organizacion_historica_semilla_sha256'] = 'b' * 64
                    else:
                        (raiz / 'organizacion-historica-semilla.json').write_text('{}')
                    with patch('copia_local_organizacion_historica_activacion.oh.sembrar') as sembrar:
                        with self.assertRaisesRegex(ValueError, 'concili'):
                            activar(copia, Path('/fuente'))
                        sembrar.assert_not_called()

    def test_reinicio_fallido_restaura_entorno_sin_borrar_datos_confirmados(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            archivo = raiz / 'entorno_modulos.json'
            guardar(archivo, {'OTRA_FUNCION': 'true'})
            estado = raiz / 'estado.json'
            guardar(estado, {'preparada': True, 'entorno_modulos_sha256': sha256(archivo), 'revision': 'a' * 40})
            acciones = []
            copia = SimpleNamespace(raiz=raiz, estado=estado, existe=lambda: True,
                                    leer=lambda: json.loads(estado.read_text()),
                                    parar=lambda: acciones.append('parar'),
                                    arrancar=lambda: acciones.append('arrancar'),
                                    comprobar=lambda: acciones.append('comprobar'))

            def fallo(*args):
                guardar(archivo, {'CONSULTA_OH': 'true'})
                datos = copia.leer()
                datos['organizacion_historica_semilla_sha256'] = 'b' * 64
                guardar(estado, datos)
                raise RuntimeError('Fallo del primer arranque')

            with patch('copia_local.verificar_fuente'), patch('copia_local.memoria'), \
                    patch('copia_local.ejecutar'), \
                    patch('copia_local_organizacion_historica_activacion.subprocess.check_output', return_value='/go'), \
                    patch('copia_local_organizacion_historica_activacion.oh.comprobar_rol_publicado'), \
                    patch('copia_local_organizacion_historica_activacion._montar', side_effect=fallo):
                with self.assertRaisesRegex(RuntimeError, 'primer arranque'):
                    activar(copia, Path('/fuente'))
            self.assertEqual(json.loads(archivo.read_text()), {'OTRA_FUNCION': 'true'})
            self.assertEqual(copia.leer()['entorno_modulos_sha256'], sha256(archivo))
            self.assertEqual(copia.leer()['organizacion_historica_semilla_sha256'], 'b' * 64)
            self.assertEqual(copia.leer()['organizacion_historica_activacion'], 'interrumpida')
            self.assertEqual(acciones, ['parar', 'arrancar', 'comprobar'])


if __name__ == '__main__':
    unittest.main()
