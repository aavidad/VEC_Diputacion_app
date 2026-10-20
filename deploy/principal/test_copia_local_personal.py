"""Contrato focal de Personal en una copia local, sin PostgreSQL ni compilación."""

import json
import csv
from datetime import datetime, timedelta, timezone
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlsplit

import copia_local_personal as personal


def guardar(ruta, datos):
    ruta.parent.mkdir(parents=True, exist_ok=True)
    ruta.write_text(json.dumps(datos) + '\n')
    ruta.chmod(0o600)


def referencia():
    operaciones = {nombre: {'motivo_ref': 'motivo:' + nombre,
                            'capacidad': {'file': 'legado-' + nombre + '.cap'}}
                   for nombre in personal.OBLIGATORIAS}
    return {'esquema': 'incorporacion_v2',
            'referencias': {'norma_ref': 'norma:B2', 'motivo_ref': 'motivo:aprobado'},
            'dsn_files': {'fuente_autorizacion': '/legado/fuente.dsn'},
            'personal_b2': {
                'protocolo': 'personal_b2_v1',
                'cese_fecha_efecto': 'primer_dia_sin_relacion',
                'organismo_ref': 'org:prueba',
                'catalogo_rpt_id': 'catalogo:aprobado',
                'modulo_rpt_id': 'modulo:rpt',
                'database_url': 'postgresql://legado/ajeno',
                'capacidad': {'file': '/legado/capacidad-global.cap'},
                'dsn_files': {nombre: '/legado/' + nombre + '.dsn' for nombre in personal.B2},
                'operaciones': operaciones}}


class CopiaFalsa:
    nombre = 'vec-copia-l-personal'

    def __init__(self, raiz):
        self.raiz = raiz
        self.material = raiz / 'material-base'
        (self.material / 'idempotencia').mkdir(parents=True)
        (self.material / 'ca').mkdir()
        guardar(self.material / 'idempotencia/configuracion.json', {'ok': True})
        (self.material / 'ca/ca.crt').write_text('CA de prueba\n')
        self.estado = raiz / 'estado.json'
        guardar(self.estado, {'puerto_pg': 55481})
        self.entorno = {'VEC_CT_GOBIERNO_DATABASE_URL': 'postgresql://gobierno-anterior',
                        'VEC_CT_MOTIVOS_RRHH_DATABASE_URL': 'postgresql://motivos-anterior',
                        'VEC_OTRO_MODULO': 'conservado'}
        guardar(raiz / 'entorno_modulos.json', self.entorno)
        self.sql_llamadas = []
        self.eventos = []

    def leer(self):
        return json.loads(self.estado.read_text())

    def entorno_modulos(self):
        return json.loads((self.raiz / 'entorno_modulos.json').read_text())

    def sql_bytes(self, sql, log):
        self.sql_llamadas.append((sql.decode(), log))

    def sql(self, ruta, log):
        self.eventos.append(('sql', ruta.name, log))

    def parar(self):
        self.eventos.append(('parar',))

    def pg(self):
        self.eventos.append(('pg',))

    def arrancar(self, comprobar_modulos=True):
        self.eventos.append(('arrancar', comprobar_modulos))

    def comprobar(self):
        self.eventos.append(('comprobar',))


class PersonalLocal(unittest.TestCase):
    def setUp(self):
        temporal = tempfile.TemporaryDirectory()
        self.addCleanup(temporal.cleanup)
        self.raiz = Path(temporal.name)
        self.copia = CopiaFalsa(self.raiz)
        self.origen = self.raiz / 'aprobada.json'
        self.aprobada = referencia()
        guardar(self.origen, self.aprobada)

    def test_configuracion_rehace_dsn_y_capacidades_sin_perder_motivos(self):
        datos = personal.configuracion(self.origen)
        self.assertEqual(datos['referencias'], self.aprobada['referencias'])
        self.assertEqual(datos['personal_b2']['organismo_ref'], 'org:prueba')
        self.assertEqual(datos['personal_b2']['catalogo_rpt_id'], 'catalogo:aprobado')
        self.assertEqual(datos['personal_b2']['modulo_rpt_id'], 'modulo:rpt')
        self.assertEqual(datos['personal_b2']['cese_fecha_efecto'], 'primer_dia_sin_relacion')
        self.assertNotIn('database_url', datos['personal_b2'])
        self.assertNotIn('capacidad', datos['personal_b2'])
        self.assertEqual(datos['dsn_files'],
                         {nombre: 'base-' + nombre + '.dsn' for nombre in personal.BASE})
        self.assertEqual(datos['personal_b2']['dsn_files'],
                         {nombre: 'b2-' + nombre + '.dsn' for nombre in personal.B2})
        for nombre, operacion in datos['personal_b2']['operaciones'].items():
            self.assertEqual(operacion['motivo_ref'], 'motivo:' + nombre)
            self.assertEqual(operacion['capacidad'], {'file': nombre + '.cap'})
        self.assertNotIn('/legado/', json.dumps(datos))
        self.assertEqual(self.aprobada['personal_b2']['operaciones']['rpt_listar']['capacidad'],
                         {'file': 'legado-rpt_listar.cap'})

    def test_rechaza_categoria_o_cese_incompletos_antes_de_parar(self):
        for quitar, mensaje in [('rpt_listar', 'categoría'),
                                ('personal_catalogo_retirar', 'catálogos')]:
            with self.subTest(operacion=quitar):
                datos = referencia()
                del datos['personal_b2']['operaciones'][quitar]
                guardar(self.origen, datos)
                with self.assertRaisesRegex(ValueError, mensaje):
                    personal.configuracion(self.origen)
        datos = referencia()
        datos['personal_b2']['cese_fecha_efecto'] = ''
        guardar(self.origen, datos)
        with self.assertRaisesRegex(ValueError, 'cese'):
            personal.preparar(self.copia, self.origen, repo=self.raiz,
                             go='go-falso', entorno_go={}, guardar=guardar)
        self.assertEqual(self.copia.eventos, [])

    def test_conexiones_propias_tls_con_vigencia_y_reuso(self):
        carpeta = self.raiz / 'personal'
        carpeta.mkdir(mode=0o700)
        urls = personal.conexiones(self.copia, carpeta, guardar)
        self.assertEqual(len(urls), len(personal.BASE) + len(personal.B2) + 1)
        self.assertEqual(len(self.copia.sql_llamadas), 1)
        sql, log = self.copia.sql_llamadas[0]
        self.assertEqual(log, 'personal-conexiones.log')
        vencimientos = re.findall(r"VALID UNTIL '([^']+)'", sql)
        self.assertEqual(len(vencimientos), len(urls))
        self.assertEqual(len(set(vencimientos)), 1)
        vence = datetime.strptime(vencimientos[0], '%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=timezone.utc)
        self.assertLess(datetime.now(timezone.utc) + timedelta(days=89), vence)
        self.assertLess(vence, datetime.now(timezone.utc) + timedelta(days=91))
        usuarios = set()
        for campo, url in urls.items():
            partes = urlsplit(url)
            self.assertEqual((partes.scheme, partes.hostname, partes.port, partes.path),
                             ('postgresql', 'localhost', 55481, '/postgres'))
            self.assertEqual(parse_qs(partes.query),
                             {'sslmode': ['verify-full'],
                              'sslrootcert': [str(self.copia.material / 'ca/ca.crt')]})
            self.assertNotIn(partes.username, usuarios)
            usuarios.add(partes.username)
            if campo in personal.LOGINS_ORIGEN:
                self.assertEqual(partes.username, personal.LOGINS_ORIGEN[campo])
            self.assertNotIn('anterior', url)
        self.assertEqual(len(usuarios & set(personal.LOGINS_ORIGEN.values())), 6)
        self.assertEqual(len(usuarios - set(personal.LOGINS_ORIGEN.values())), 3)
        self.assertEqual(personal.conexiones(self.copia, carpeta, guardar), urls)
        self.assertEqual(len(self.copia.sql_llamadas), 1)

    def test_logins_canonicos_coinciden_con_inventario_ad172(self):
        ruta = Path(__file__).with_name('origen_consumos_ad172') / 'ternas.tsv'
        with ruta.open(newline='') as fichero:
            filas = list(csv.reader(fichero, delimiter='\t'))
        logins = {fila[1] for fila in filas if fila and fila[0] == 'incorporacionb'}
        self.assertEqual(len(logins), 6)
        self.assertEqual(set(personal.LOGINS_ORIGEN.values()), logins)

    def test_conexiones_guardadas_de_otra_copia_o_sin_tls_se_rechazan_sin_sql(self):
        carpeta = self.raiz / 'personal'
        carpeta.mkdir(mode=0o700)
        urls = personal.conexiones(self.copia, carpeta, guardar)
        self.copia.sql_llamadas.clear()
        marca = carpeta / 'conexiones.json'
        for alterada in (urls['cese'].replace(':55481/', ':55482/'),
                         urls['cese'].replace('sslmode=verify-full', 'sslmode=disable')):
            with self.subTest(url=alterada):
                guardar(marca, {**urls, 'cese': alterada})
                with self.assertRaisesRegex(ValueError, 'ajena|TLS'):
                    personal.conexiones(self.copia, carpeta, guardar)
                self.assertEqual(self.copia.sql_llamadas, [])

    def test_entrada_enlazada_se_rechaza_antes_de_parar(self):
        carpeta = self.raiz / 'personal'
        carpeta.mkdir(mode=0o700)
        destino = self.raiz / 'ajena'
        destino.mkdir(mode=0o700)
        (carpeta / 'entrada').symlink_to(destino, target_is_directory=True)
        with patch.object(personal.subprocess, 'run',
                          side_effect=AssertionError('no debe compilar')):
            with self.assertRaises(ValueError):
                personal.preparar(self.copia, self.origen, repo=self.raiz,
                                 go='go-falso', entorno_go={}, guardar=guardar)
        self.assertEqual(self.copia.eventos, [])
        self.assertEqual(self.copia.sql_llamadas, [])
        self.assertFalse((carpeta / 'preparacion.json').exists())

    def ejecutar(self, *, fallar_productor=False, fallar_comprobacion=False):
        llamadas = []
        gobierno = []

        def crear(_):
            gobierno.append('crear')
            return 'postgresql://gobierno-temporal'

        def retirar(_):
            gobierno.append('retirar')

        def proceso(comando, **_):
            llamadas.append(comando)
            if '-salida' in comando:
                if fallar_productor:
                    raise subprocess.CalledProcessError(12, comando)
                salida = Path(comando[comando.index('-salida') + 1])
                salida.mkdir(mode=0o700)
                guardar(salida / 'servidor.json', {'material': 'nuevo'})
            return subprocess.CompletedProcess(comando, 0)

        try:
            with (patch.object(personal.subprocess, 'run', side_effect=proceso),
                  patch.object(personal, 'comprobar_personal',
                               side_effect=ValueError('Personal no está listo: HTTP 503')
                               if fallar_comprobacion else None),
                  patch('copia_local_conexiones.crear_preparador_gobierno', side_effect=crear,
                        create=True),
                  patch('copia_local_conexiones.retirar_preparador_gobierno', side_effect=retirar,
                        create=True)):
                personal.preparar(self.copia, self.origen, repo=self.raiz,
                                 go='go-falso', entorno_go={}, guardar=guardar)
        finally:
            self.assertEqual(gobierno, ['crear', 'retirar'])
            self.assertFalse((self.raiz / 'personal/entrada/gobierno.dsn').exists())
        return llamadas

    def test_preparacion_repetida_comprueba_y_no_republica(self):
        llamadas = self.ejecutar()
        self.assertEqual(len(llamadas), 2)
        self.assertEqual(len(self.copia.sql_llamadas), 1)
        marca = self.raiz / 'personal/preparada.json'
        self.assertTrue(marca.exists())
        anterior = self.copia.entorno_modulos()
        with (patch.object(personal.subprocess, 'run', side_effect=AssertionError('recompilación')),
              patch.object(personal, 'comprobar_personal') as comprobacion):
            personal.preparar(self.copia, self.origen, repo=self.raiz,
                             go='go-falso', entorno_go={}, guardar=guardar)
        comprobacion.assert_called_once_with(self.copia)
        self.assertEqual(self.copia.eventos[-1], ('comprobar',))
        self.assertEqual(self.copia.entorno_modulos(), anterior)
        self.assertEqual(len(self.copia.sql_llamadas), 1)
        guardar(self.raiz / 'personal/material/servidor.json', {'material': 'alterado'})
        with self.assertRaisesRegex(ValueError, 'cambió'):
            personal.preparar(self.copia, self.origen, repo=self.raiz,
                             go='go-falso', entorno_go={}, guardar=guardar)
        self.assertEqual(len(self.copia.sql_llamadas), 1)

    def test_fallo_productor_recupera_entorno_y_no_marca_exito(self):
        anterior = self.copia.entorno_modulos()
        with self.assertRaises(subprocess.CalledProcessError):
            self.ejecutar(fallar_productor=True)
        self.assertEqual(self.copia.entorno_modulos(), anterior)
        self.assertEqual(self.copia.eventos[-2:], [('parar',), ('arrancar', True)])
        self.assertFalse((self.raiz / 'personal/preparada.json').exists())
        self.assertEqual(len(self.copia.sql_llamadas), 1)

    def test_interrupcion_con_borrador_recupera_entorno_anterior(self):
        anterior = self.copia.entorno_modulos()
        carpeta = self.raiz / 'personal'
        carpeta.mkdir(mode=0o700)
        guardar(carpeta / 'preparacion.json', {
            'origen_sha256': personal._sha256(self.origen),
            'entorno_anterior': anterior})
        borrador = {**anterior, 'VEC_CT_INCORPORACION_V2_FILE': '/borrador/servidor.json',
                    'VEC_PERSONAL_B2_GOBIERNO_ENABLED': 'true'}
        guardar(self.raiz / 'entorno_modulos.json', borrador)
        with self.assertRaises(subprocess.CalledProcessError):
            self.ejecutar(fallar_productor=True)
        self.assertEqual(self.copia.entorno_modulos(), anterior)
        self.assertFalse((carpeta / 'preparada.json').exists())

    def test_dos_fallos_restauran_configuracion_posterior_al_primero(self):
        carpeta = self.raiz / 'personal'
        with self.assertRaises(subprocess.CalledProcessError):
            self.ejecutar(fallar_productor=True)
        self.assertFalse((carpeta / 'preparacion.json').exists())
        posterior = {**self.copia.entorno_modulos(),
                     'VEC_OTRO_MODULO': 'configuracion_posterior'}
        guardar(self.raiz / 'entorno_modulos.json', posterior)
        with self.assertRaises(subprocess.CalledProcessError):
            self.ejecutar(fallar_productor=True)
        self.assertEqual(self.copia.entorno_modulos(), posterior)
        self.assertFalse((carpeta / 'preparada.json').exists())
        self.assertFalse((carpeta / 'preparacion.json').exists())

    def test_http_503_de_personal_recupera_entorno_y_no_marca_exito(self):
        anterior = self.copia.entorno_modulos()
        with self.assertRaisesRegex(ValueError, 'HTTP 503'):
            self.ejecutar(fallar_comprobacion=True)
        self.assertEqual(self.copia.entorno_modulos(), anterior)
        self.assertEqual(self.copia.eventos[-2:], [('parar',), ('arrancar', True)])
        self.assertFalse((self.raiz / 'personal/preparada.json').exists())

    def test_comprobacion_real_rechaza_respuesta_503(self):
        with (patch.object(personal.ssl, 'create_default_context') as tls,
              patch.object(personal.http.client, 'HTTPSConnection') as conexion):
            conexion.return_value.getresponse.return_value.status = 503
            self.copia.leer = lambda: {'puerto_app': 18446}
            with self.assertRaisesRegex(ValueError, 'HTTP 503'):
                personal.comprobar_personal(self.copia)
        tls.return_value.load_cert_chain.assert_called_once()
        conexion.return_value.close.assert_called_once()


if __name__ == '__main__':
    unittest.main()
