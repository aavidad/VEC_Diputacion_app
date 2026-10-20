"""El caso usa los vínculos de la carga confirmada, sin identidades heredadas."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

from recorrido_copia import caso_bolsa


class CasoBolsaTest(unittest.TestCase):
    def test_vinculo_ausente_detiene_la_preparacion_de_identidad(self):
        c = Mock()
        c.consultar.return_value = ''
        carga = {'bolsa_ref': 'bolsa:nueva', 'acta_ref': 'acta:nueva', 'version': 1,
                 'aspirante': {'fila_numero': 2, 'nombre': 'Nombre del fichero'}}
        with patch.object(caso_bolsa.carga_bolsa, 'preparar', return_value=carga):
            with self.assertRaisesRegex(ValueError, 'vínculo único'):
                caso_bolsa.preparar(c)
        c.guardar.assert_not_called()
        c.copia.parar.assert_not_called()

    def test_identidad_nueva_usa_los_vinculos_y_se_prepara_con_la_app_parada(self):
        import copia_local_caso40
        eventos = []
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            copia = Mock(raiz=raiz, estado=raiz / 'estado.json', nombre='vec-copia-l-caso')
            copia.leer.return_value = {'sid_copia': '12345678901234567'}
            copia.entorno_modulos.return_value = {}
            for nombre in ('parar', 'pg', 'arrancar'):
                getattr(copia, nombre).side_effect = lambda nombre=nombre: eventos.append(nombre)
            c = Mock(raiz=raiz, copia=copia)
            c.consultar.return_value = json.dumps({'candidato': 'can_' + 'a' * 43,
                'participacion': 'participacion:nueva', 'instantanea_ref': 'instantanea:nueva'})
            carga = {'bolsa_ref': 'bolsa:nueva', 'acta_ref': 'acta:nueva', 'version': 1,
                     'aspirante': {'fila_numero': 2, 'nombre': 'Nombre del fichero'}}
            with patch.object(caso_bolsa.carga_bolsa, 'preparar', return_value=carga), \
                 patch.object(copia_local_caso40, 'preparar_aspirante',
                     side_effect=lambda *_: eventos.append('identidad') or {}), \
                 patch.object(copia_local_caso40, 'registrar_contacto',
                     side_effect=lambda *_: eventos.append('contacto')):
                caso_bolsa.preparar(c)
            seleccion = c.guardar.call_args.args[1]['seleccion']
            self.assertEqual(seleccion['candidato'], 'can_' + 'a' * 43)
            self.assertEqual(seleccion['bolsa'], 'bolsa:nueva')
            self.assertEqual(eventos, ['parar', 'pg', 'identidad', 'arrancar', 'contacto'])
