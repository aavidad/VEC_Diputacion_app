"""Los perfiles de Dietas usan los actos ADMIN y conservan su material."""

import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

from recorrido_copia import dietas_gobierno as gobierno
from recorrido_copia import catalogo
from recorrido_copia.comun import Contexto


class GobiernoDietasTest(unittest.TestCase):
    def test_sin_aut107_no_prepara_nada(self):
        from recorrido_copia import dietas
        c = Mock()
        c.consultar.return_value = 'f'
        with patch('recorrido_copia.catalogo.preparar_dietas') as preparar:
            with self.assertRaisesRegex(ValueError, 'AUT107 exacta'):
                dietas.preparar(c)
        preparar.assert_not_called()
        c.sql.assert_not_called()
        c.guardar.assert_not_called()

    def test_dos_roles_por_actos_existentes_y_recuperacion_sin_nuevas_propuestas(self):
        self.ejecutar_publicacion(False)

    def test_rol_ya_publicado_con_mas_campos_se_rechaza(self):
        self.ejecutar_publicacion(True)

    def ejecutar_publicacion(self, ampliar):
        repo = Path(__file__).parents[2]
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            c = Mock(fuente=repo, raiz=raiz, carpeta=raiz / 'recorrido',
                     guardar=Contexto.guardar, leer=Contexto.leer)
            c.copia.nombre = 'vec-copia-s-dietas'
            c.copia.leer.return_value = {'sid_copia': '12345678901234567'}
            c.consultar.return_value = 't'
            Contexto.guardar(raiz / 'admin/h10/entrada-vec-admin.json', {'motivo': {'dato': 'motivo'}})
            fuentes = [json.loads((repo / 'deploy/principal/fuentes' / spec[2]).read_bytes())
                       for spec in catalogo.ACTOS_DIETAS]
            publicado = {'referencia': catalogo.CATALOGO, 'version': 5,
                         'entradas': [e for f in fuentes for e in f['entradas']]}
            contratos = json.loads((repo / 'deploy/principal/recorrido_copia/datos/dietas_roles.v1.json').read_bytes())['roles']
            roles = {}
            propuestas = []

            def cerrar(_c, carpeta, tipo, propuesta):
                self.assertIs(_c, c)
                self.assertEqual(tipo, 'roles')
                propuestas.append(propuesta)
                concesiones = [next(e['concesion'] for e in publicado['entradas']
                    if e['referencia'] == s['entrada_ref']) for s in propuesta['selecciones']]
                roles[propuesta['rol_id']] = {'ref': 'rol:' + propuesta['rol_id'] + ':v1',
                    'version': 1, 'control_estado': 'habilitada', 'huella': 'a' * 64,
                    'documento': {'concesiones': concesiones}}
                return {'recibo_ref': 'referencia-ficticia-del-doble'}

            if ampliar:
                primer = contratos[0]
                cerrar(c, None, 'roles', {'rol_id': primer['rol_id'], 'selecciones': [
                    {'entrada_ref': 'accion:' + a} for a in primer['acciones']]})
                roles[primer['rol_id']]['documento']['concesiones'] = copy.deepcopy(
                    roles[primer['rol_id']]['documento']['concesiones'])
                roles[primer['rol_id']]['documento']['concesiones'][0]['campos_permitidos'].append('campo_ajeno')
                propuestas.clear()

            with (patch.object(gobierno.roles_admin, '_cabeza', return_value=({}, publicado)),
                  patch.object(gobierno.roles_admin, '_rol', side_effect=lambda _c, rol: roles.get(rol)),
                  patch.object(gobierno.roles_admin, '_proponer_y_cerrar', side_effect=cerrar)):
                admisiones = {'actos': [{'catalogo_sha256': 'b' * 64}]}
                if ampliar:
                    with self.assertRaisesRegex(ValueError, 'contrato exacto'):
                        gobierno.publicar(c, admisiones)
                    self.assertEqual(propuestas, [])
                    return
                primero = gobierno.publicar(c, admisiones)
                self.assertEqual([p['rol_id'] for p in propuestas], [r['rol_id'] for r in contratos])
                self.assertEqual([len(p['selecciones']) for p in propuestas], [3, 2])
                self.assertEqual(len(primero['roles']), 2)
                self.assertEqual(gobierno.publicar(c, admisiones), primero)
                self.assertEqual(len(propuestas), 2)
                c.sql.assert_not_called()
