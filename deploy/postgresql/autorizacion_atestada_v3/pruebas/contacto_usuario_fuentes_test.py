"""Controles de fuente: no ejecutan PostgreSQL ni acreditan permisos o transacciones."""
import hashlib
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[4]
MIG = ROOT / 'deploy/postgresql/autorizacion_atestada_v3/migraciones'
UP = (MIG / '000035_consumidor_contacto_usuario.up.sql').read_text()
DOWN = (MIG / '000035_consumidor_contacto_usuario.down.sql').read_text()


def cuerpo_revalidacion():
    s = (MIG / '000005_revalidacion_final_consultas_rrhh_v3.up.sql').read_text()
    return s.split('AS $funcion$')[1].split('$funcion$;')[0].replace(
        "d ->> 'valida_hasta' <> c ->> 'decision_valida_hasta'",
        "(d ->> 'valida_hasta')::timestamptz <> (c ->> 'decision_valida_hasta')::timestamptz")


class FuentesContacto(unittest.TestCase):
    def test_revalidacion_extiende_y_restaura_preimagen_exacta(self):
        original = cuerpo_revalidacion()
        self.assertEqual(hashlib.sha256(original.encode()).hexdigest(),
                         '7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff')
        bloque = UP.split('DO $revalidacion$')[1].split('END $revalidacion$;')[0]
        desde = original.index('BEGIN\n    IF pg_catalog.current_setting(') + 13
        hasta = original.index(' THEN\n        RAISE EXCEPTION USING')
        inicio = bloque.split('placing $inicio$')[1].split('$inicio$')[0]
        nuevo = original[:desde] + inicio + original[desde:hasta] + ') END /* AD3-35 RV FIN */' + original[hasta:]
        for previo, posterior in re.findall(r'nueva:=replace\(nueva,\$antes\$(.*?)\$antes\$,\$despues\$(.*?)\$despues\$\)', bloque, re.S):
            self.assertEqual(nuevo.count(previo), 1)
            nuevo = nuevo.replace(previo, posterior)
        self.assertEqual(hashlib.sha256(nuevo.encode()).hexdigest(),
                         '2f15b9ca8b64d453e79063fe706a5ac87ec3453e5c9451e071ecfc6bc30b2ff0')
        # Las comprobaciones caras siguen literales en el helper existente.
        for fragmento in ('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(',
                          'v_consumo.capacidad_canonica <> p_capacidad_canonica',
                          '.revocacion_clave_capacidad r', '.revocacion_configuracion r',
                          '.revocacion_raiz r'):
            self.assertEqual(nuevo.count(fragmento), original.count(fragmento))
        a = nuevo.index('/* AD3-35 RV INICIO */')
        b = nuevo.index('/* AD3-35 RV ORIGINAL */') + len('/* AD3-35 RV ORIGINAL */')
        c = nuevo.index(') END /* AD3-35 RV FIN */')
        restaurado = nuevo[:a] + nuevo[b:c] + nuevo[c+len(') END /* AD3-35 RV FIN */'):]
        bloque_down = DOWN.split('DO $revalidacion$')[1].split('END $revalidacion$;')[0]
        for previo, posterior in re.findall(r'nueva:=replace\(nueva,\$antes\$(.*?)\$antes\$,\$despues\$(.*?)\$despues\$\)', bloque_down, re.S):
            self.assertEqual(restaurado.count(previo), 1)
            restaurado = restaurado.replace(previo, posterior)
        self.assertEqual(restaurado, original)

    def test_revalidacion_no_reutiliza_replay_de_consumo(self):
        f = UP.split('CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_consulta_contacto_usuario_v3_atestada')[1].split('END $f$;')[0]
        self.assertNotIn('consumir_decision_mutacion_v3_interna', f)
        self.assertIn('revalidar_consumo_consulta_rrhh_v3_interna', f)
        self.assertLess(f.index('contacto_validar_material_v1'), f.index('RETURN QUERY'))

    def test_down_no_elimina_historia(self):
        self.assertIn('LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN ACCESS EXCLUSIVE MODE', DOWN)
        self.assertNotRegex(DOWN, r'(?i)\bDELETE\s+FROM\b|\bTRUNCATE\b|\bCASCADE\b')
        self.assertIn('no admite DOWN', DOWN)

    def test_parseo_sql_y_plpgsql_si_parser_disponible(self):
        try:
            from pglast import parse_sql, parse_plpgsql
        except ImportError:
            self.skipTest('pglast opcional ausente; resto son pruebas de fuente stdlib')
        archivos = [MIG / '000035_consumidor_contacto_usuario.up.sql',
                    MIG / '000035_consumidor_contacto_usuario.down.sql',
                    ROOT / 'deploy/postgresql/bolsa_registro_accesos/migraciones/000006_registrar_contacto_usuario.up.sql',
                    ROOT / 'deploy/postgresql/bolsa_registro_accesos/migraciones/000006_registrar_contacto_usuario.down.sql',
                    MIG.parent / 'pruebas_sql/contacto_usuario_codec.sql']
        for archivo in archivos:
            with self.subTest(archivo=archivo.name):
                sql = '\n'.join(l for l in archivo.read_text().splitlines() if not l.startswith('\\'))
                parse_sql(sql)
                parse_plpgsql(sql)


if __name__ == '__main__':
    unittest.main()
