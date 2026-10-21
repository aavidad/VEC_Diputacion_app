"""Aislamiento y recuperación de las conexiones opcionales de Dietas."""

from datetime import datetime, timezone
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
import subprocess
import sys
from urllib.parse import parse_qs, urlsplit

from recorrido_copia import dietas
from recorrido_copia.comun import Contexto


class ConexionesDietasTest(unittest.TestCase):
    def setUp(self):
        temporal = tempfile.TemporaryDirectory()
        self.addCleanup(temporal.cleanup)
        self.raiz = Path(temporal.name)
        self.raiz.chmod(0o700)
        material = self.raiz / 'material'
        (material / 'ca').mkdir(mode=0o700, parents=True)
        material.chmod(0o700)
        (material / 'ca/ca.crt').write_text('CA ficticia de esta prueba')
        (material / 'ca/ca.crt').chmod(0o600)
        carpeta = self.raiz / 'recorrido'
        carpeta.mkdir(mode=0o700)
        self.estado = {'recorrido_completo': True, 'sid_copia': '12345678901234567',
                       'puerto_pg': 55581}
        copia = Mock(nombre='vec-copia-s-dietas', leer=lambda: self.estado)
        self.c = Mock(copia=copia, raiz=self.raiz, material=material, carpeta=carpeta,
                      guardar=Contexto.guardar, leer=Contexto.leer)
        self.c.consultar.return_value = self.estado['sid_copia']
        self.c.ahora.return_value = datetime(2026, 10, 21, tzinfo=timezone.utc)

    def test_once_identidades_tls_separadas_y_recuperacion_sin_otra_alta(self):
        valores = dietas.conexiones(self.c)
        self.assertEqual(len(valores), 11)
        urls = [urlsplit(v) for v in valores.values()]
        self.assertEqual(len({u.username for u in urls}), 11)
        self.assertEqual(len({u.password for u in urls}), 11)
        for u in urls:
            self.assertEqual((u.hostname, u.port, u.path), ('localhost', 55581, '/postgres'))
            self.assertEqual(parse_qs(u.query), {'sslmode': ['verify-full'],
                'sslrootcert': [str(self.c.material / 'ca/ca.crt')]})
        sentencia = self.c.sql.call_args.args[0].decode()
        self.assertEqual(sentencia.count('CREATE ROLE '), 11)
        self.assertEqual(sentencia.count('WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;'), 11)
        self.assertEqual(sentencia.count("VALID UNTIL '2027-01-19T00:00:00Z'"), 11)
        self.assertNotIn('INSERT', sentencia)
        self.assertNotIn('VEC_DIETAS_BORRADORES_ENABLED', valores)
        self.assertEqual(dietas.conexiones(self.c), valores)
        self.c.sql.assert_called_once()
        self.assertEqual((self.c.carpeta / 'dietas/conexiones.json').stat().st_mode & 0o777, 0o600)

    def test_sql_incierto_no_permite_repetir_el_alta(self):
        self.c.sql.side_effect = ValueError('respuesta incierta')
        with self.assertRaisesRegex(ValueError, 'respuesta incierta'):
            dietas.conexiones(self.c)
        with self.assertRaisesRegex(ValueError, 'conciliar'):
            dietas.conexiones(self.c)
        self.c.sql.assert_called_once()

    def test_no_admite_conexion_ajena_guardada(self):
        dietas.conexiones(self.c)
        ruta = self.c.carpeta / 'dietas/conexiones.json'
        valores = json.loads(ruta.read_bytes())
        campo = next(iter(valores))
        valores[campo] = valores[campo].replace('@localhost:', '@otra-copia:')
        ruta.write_text(json.dumps(valores))
        with self.assertRaisesRegex(ValueError, 'ajena'):
            dietas.conexiones(self.c)
        self.c.sql.assert_called_once()

    def test_sid_ajeno_no_crea_cuentas_ni_material(self):
        self.c.consultar.return_value = '98765432109876543'
        with self.assertRaisesRegex(ValueError, 'identificador PostgreSQL'):
            dietas.conexiones(self.c)
        self.c.sql.assert_not_called()
        self.assertFalse((self.c.carpeta / 'dietas').exists())

    def test_preparar_no_activa_ni_anuncia_un_recorrido(self):
        with patch('recorrido_copia.catalogo.preparar_dietas', return_value={'recibos_reales': 2}) as admitir:
            resultado = dietas.preparar(self.c)
        admitir.assert_called_once_with(self.c)
        self.assertFalse(resultado['activada'])
        self.assertEqual(resultado['conexiones_entorno'], 4)
        self.assertEqual(resultado['conexiones_manifiesto'], 7)
        self.assertIn('perfil_gobernado', resultado['pendiente'])
        self.c.copia.arrancar.assert_not_called()
        self.c.copia.parar.assert_not_called()
        self.assertFalse((self.c.material / 'identidad/dietas-comisiones.json').exists())

    def test_selector_fuera_de_crear_completo_se_rechaza_antes_de_efectos(self):
        for orden in ('crear', 'reiniciar'):
            with self.subTest(orden=orden):
                resultado = subprocess.run([sys.executable,
                    str(Path(__file__).with_name('copia_local.py')), orden,
                    '--equipo', 's', '--instancia', 'invalida', '--dietas'],
                    capture_output=True, check=False)
                self.assertEqual(resultado.returncode, 2)
                self.assertIn(b'--dietas requiere crear --recorrido-completo', resultado.stderr)


if __name__ == '__main__':
    unittest.main()
