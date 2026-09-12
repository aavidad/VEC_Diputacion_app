"""Controles de fuente: no ejecutan PostgreSQL ni acreditan permisos o transacciones."""
import hashlib
import os
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


def literal(sql, nombre):
    m = re.search(r'\b' + nombre + r'\s+text\s*:=\s*(\$\w+\$)(.*?)\1;', sql, re.S)
    if not m:
        raise AssertionError('literal no localizado: ' + nombre)
    return m[2]


def sustituir_unico(texto, previo, posterior):
    if texto.count(previo) != 1:
        raise AssertionError('preimagen no única: ' + previo[:80])
    return texto.replace(previo, posterior)


def nucleo_hasta_32(fuente32):
    """Reconstruye prosrc, sin ejecutar SQL: 8/9/10 completos y parches 11–32."""
    marca = "       )\n       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'"
    cuerpo = None
    cabecera = None
    for numero in range(8, 33):
        if numero == 21:  # AD3-21 modifica consumidores RRHH, no este núcleo.
            continue
        archivo = fuente32 if numero == 32 else next(MIG.glob(f'{numero:06d}_*.up.sql'))
        sql = archivo.read_text()
        if numero <= 10:
            m = re.search(r'(CREATE (?:OR REPLACE )?FUNCTION vec_autorizacion_atestada_v3\.consumir_decision_mutacion_v3_interna\(.*?AS \$funcion\$)(.*?)\$funcion\$;', sql, re.S)
            if not m:
                raise AssertionError('definición completa ausente')
            cabecera, cuerpo = m[1], m[2]
            continue
        if numero in (11, 26):
            cuerpo = sustituir_unico(cuerpo, literal(sql, 'v_runtime_anterior'), literal(sql, 'v_runtime_nuevo'))
        elif numero == 28:
            previo = literal(sql, 'v_runtime_anterior')
            nuevo = sustituir_unico(previo, "p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio'",
                "p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio' AND p_perfil_mutacion IS DISTINCT FROM 'lectura_registro_personal_incorporacion'")
            nuevo = sustituir_unico(nuevo, '       ) THEN', '           ' + literal(sql, 'v_lector') + '       ) THEN')
            cuerpo = sustituir_unico(cuerpo, previo, nuevo)
        elif numero == 29:
            for nombre in ('v_perfil', 'v_lector'):
                previo = literal(sql, nombre + '1')
                cuerpo = sustituir_unico(cuerpo, previo, previo + literal(sql, nombre + '2'))
            cuerpo = sustituir_unico(cuerpo, literal(sql, 'v_exclusion1'), literal(sql, 'v_exclusion2'))
        if numero in (15, 18, 19):
            cuerpo = sustituir_unico(cuerpo, literal(sql, 'v_anterior'), literal(sql, 'v_nuevo'))
        if numero not in (15, 18, 29):
            extension = literal(sql, 'extension' if numero == 32 else 'v_extension')
            cuerpo = sustituir_unico(cuerpo, marca, extension + marca)
        if numero == 14:
            cuerpo = sustituir_unico(cuerpo, literal(sql, 'v_fecha_textual'), literal(sql, 'v_fecha_instante'))
    return cabecera, cuerpo


class FuentesContacto(unittest.TestCase):
    def test_revalidacion_extiende_y_restaura_preimagen_exacta(self):
        original = cuerpo_revalidacion()
        self.assertEqual(hashlib.sha256(original.encode()).hexdigest(),
                         '7cfc002cff8878fc36288fa1200de1c51965e9ae4d84b4ffe6179bc62372b5ff')
        bloque = UP.split('DO $revalidacion$')[1].split('END $revalidacion$;')[0]
        desde = original.index('BEGIN\n    IF pg_catalog.current_setting(') + 13
        hasta = original.index(' THEN\n        RAISE EXCEPTION USING')
        inicio = bloque.split('placing $inicio$')[1].split('$inicio$')[0]
        nuevo = original[:desde] + inicio + original[desde:hasta] + ') END) /* AD3-35 RV FIN */' + original[hasta:]
        for previo, posterior in re.findall(r'nueva:=replace\(nueva,\$antes\$(.*?)\$antes\$,\$despues\$(.*?)\$despues\$\)', bloque, re.S):
            self.assertEqual(nuevo.count(previo), 1)
            nuevo = nuevo.replace(previo, posterior)
        self.assertEqual(hashlib.sha256(nuevo.encode()).hexdigest(),
                         '7b93dd88a453fc052344825762a8ea0d46a8072fe6e058db09186c3bd50bde80')
        # Las comprobaciones caras siguen literales en el helper existente.
        for fragmento in ('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(',
                          'v_consumo.capacidad_canonica <> p_capacidad_canonica',
                          '.revocacion_clave_capacidad r', '.revocacion_configuracion r',
                          '.revocacion_raiz r'):
            self.assertEqual(nuevo.count(fragmento), original.count(fragmento))
        a = nuevo.index('/* AD3-35 RV INICIO */')
        b = nuevo.index('/* AD3-35 RV ORIGINAL */') + len('/* AD3-35 RV ORIGINAL */')
        c = nuevo.index(') END) /* AD3-35 RV FIN */')
        restaurado = nuevo[:a] + nuevo[b:c] + nuevo[c+len(') END) /* AD3-35 RV FIN */'):]
        bloque_down = DOWN.split('DO $revalidacion$')[1].split('END $revalidacion$;')[0]
        for previo, posterior in re.findall(r'nueva:=replace\(nueva,\$antes\$(.*?)\$antes\$,\$despues\$(.*?)\$despues\$\)', bloque_down, re.S):
            self.assertEqual(restaurado.count(previo), 1)
            restaurado = restaurado.replace(previo, posterior)
        self.assertEqual(restaurado, original)

    def test_revalidacion_no_reutiliza_replay_de_consumo(self):
        for nombre, perfil, accion in (
                ('consulta', 'contacto_usuario', 'consultar'),
                ('alta', 'contacto_usuario_alta', 'alta'),
                ('actualizar', 'contacto_usuario_actualizar', 'actualizar')):
            with self.subTest(fachada=nombre):
                f = UP.split('CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_' + nombre + '_contacto_usuario_v3_atestada')[1].split('END $f$;')[0]
                self.assertNotIn('consumir_decision_mutacion_v3_interna', f)
                self.assertNotIn('consumo_nuevo', f)
                self.assertNotRegex(f, r'\b(INSERT|UPDATE|DELETE)\b')
                self.assertIn('revalidar_consumo_consulta_rrhh_v3_interna', f)
                self.assertIn("'" + perfil + "',p_capacidad,p_decision,p_motivo,p_contexto", f)
                self.assertIn("'vec.contacto_usuario." + accion + "',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto", f)
                self.assertLess(f.index('contacto_validar_material_v1'), f.index('RETURN QUERY'))

    def test_revalidacion_writer_no_amplia_la_ruta_reader(self):
        bloque = UP.split('DO $revalidacion$')[1].split('END $revalidacion$;')[0]
        for accion in ('alta', 'actualizar'):
            self.assertIn("p_perfil_consulta = 'contacto_usuario_" + accion + "' THEN\n"
                          "        v_audiencia := 'vec.contacto_usuario.registro.v1';\n"
                          "        v_operacion := 'vec.contacto_usuario." + accion + "';\n"
                          "        v_tipo_recurso := 'contacto_usuario';\n"
                          "        v_finalidad := 'gestion_contacto_propio';", bloque)
            nombre = 'revalidar_' + accion + '_contacto_usuario_v3_atestada'
            concesiones = re.findall(r'GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3\.' + nombre + r'\([^;]+? TO ([^;]+);', UP)
            self.assertEqual(concesiones, ['vec_contacto_usuario_owner'])
            self.assertIn('DROP FUNCTION vec_autorizacion_atestada_v3.' + nombre, DOWN)
        self.assertIn("p_perfil_consulta='contacto_usuario' THEN 'contacto_usuario_consultar' ELSE p_perfil_consulta END", bloque)

    def test_control_final_de_frescura_se_conserva_integro(self):
        original = cuerpo_revalidacion()
        # La ampliación sólo altera selección de perfil y su guarda inicial;
        # toda la revalidación tras las esperas queda literal, hasta RETURN.
        cola = original[original.index('    v_viva_en :='):]
        self.assertIn('v_ahora := pg_catalog.clock_timestamp();', cola)
        self.assertGreater(cola.index("v_ahora >= (c ->> 'expira_en')::timestamptz"),
                           cola.index('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva('))
        self.assertGreater(cola.index('RETURN QUERY'), cola.index('.revocacion_raiz r'))
        bloque = UP.split('DO $revalidacion$')[1].split('END $revalidacion$;')[0]
        for previo, posterior in re.findall(r'nueva:=replace\(nueva,\$antes\$(.*?)\$antes\$,\$despues\$(.*?)\$despues\$\)', bloque, re.S):
            self.assertNotIn(previo, cola)

    def test_parseo_prosrc_revalidacion_completo_despues_de_execute(self):
        try:
            from pglast import parse_sql, parse_plpgsql
        except ImportError:
            self.skipTest('pglast opcional ausente')
        original = cuerpo_revalidacion()
        bloque = UP.split('DO $revalidacion$')[1].split('END $revalidacion$;')[0]
        desde = original.index('BEGIN\n    IF pg_catalog.current_setting(') + 13
        hasta = original.index(' THEN\n        RAISE EXCEPTION USING')
        inicio = bloque.split('placing $inicio$')[1].split('$inicio$')[0]
        nuevo = original[:desde] + inicio + original[desde:hasta] + ') END) /* AD3-35 RV FIN */' + original[hasta:]
        for previo, posterior in re.findall(r'nueva:=replace\(nueva,\$antes\$(.*?)\$antes\$,\$despues\$(.*?)\$despues\$\)', bloque, re.S):
            nuevo = sustituir_unico(nuevo, previo, posterior)
        base = (MIG / '000005_revalidacion_final_consultas_rrhh_v3.up.sql').read_text()
        cabecera = base[base.index('CREATE FUNCTION\nvec_autorizacion_atestada_v3.revalidar_consumo'):base.index('AS $funcion$')]
        parse_sql(cabecera + 'AS $funcion$' + nuevo + '$funcion$;')
        parse_plpgsql(cabecera + 'AS $funcion$' + nuevo + '$funcion$;')
        # Regresión del defecto: parsear el DO envolvente no detectaba que el
        # THEN del CASE sin paréntesis se tomaba por el THEN de PL/pgSQL.
        mutante = nuevo.replace('/* AD3-35 RV INICIO */ (CASE', '/* AD3-35 RV INICIO */ CASE').replace(
            ') END) /* AD3-35 RV FIN */', ') END /* AD3-35 RV FIN */')
        with self.assertRaises(Exception):
            parse_plpgsql(cabecera + 'AS $funcion$' + mutante + '$funcion$;')

    def test_parseo_nucleo_completo_8_a_32_y_postimagen_35(self):
        try:
            from pglast import parse_sql, parse_plpgsql
        except ImportError:
            self.skipTest('pglast opcional ausente')
        fuente32 = os.environ.get('VEC_CONTACTO_AD3_32_UP')
        if not fuente32:
            self.skipTest('indicar fuente final AD3-32 mediante VEC_CONTACTO_AD3_32_UP')
        fuente32 = Path(fuente32)
        self.assertEqual(hashlib.sha256(fuente32.read_bytes()).hexdigest(),
                         '2d2225f31da45b1a62df230a27b0c61c695228415951f21006cfd9c7e8050bc6')
        cabecera, original = nucleo_hasta_32(fuente32)
        self.assertEqual(hashlib.sha256(original.encode()).hexdigest(),
                         '1594b9e38adf09a4d036d22281e8f452d2935bf394178ce74b91d41904d25018')
        parse_sql(cabecera + original + '$funcion$;')
        parse_plpgsql(cabecera + original + '$funcion$;')
        bloque = UP.split('DO $nucleo$')[1].split('END $nucleo$;')[0]
        colocacion = bloque.split('nueva:=overlay(definicion placing')[1].split('FROM inicio+13')[0]
        textos = [v.replace("''", "'").replace('\\n', '\n')
                  for v in re.findall(r"E'((?:[^']|'')*)'", colocacion)]
        self.assertEqual(len(textos), 4)
        desde = original.index('BEGIN\n    IF pg_catalog.current_setting(') + 13
        hasta = original.index(' THEN\n        RAISE EXCEPTION USING')
        nuevo = original[:desde] + ''.join(textos[:-1]) + original[desde:hasta] + textos[-1] + original[hasta:]
        marca = "       )\n       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'"
        extension = literal(bloque, 'extension')
        nuevo = sustituir_unico(nuevo, marca, extension + marca)
        parse_sql(cabecera + nuevo + '$funcion$;')
        parse_plpgsql(cabecera + nuevo + '$funcion$;')
        mutante = nuevo.replace('/* AD3-35 GUARDA INICIO */ (CASE', '/* AD3-35 GUARDA INICIO */ CASE').replace(
            ') END) /* AD3-35 GUARDA FIN */', ') END /* AD3-35 GUARDA FIN */')
        with self.assertRaises(Exception):
            parse_plpgsql(cabecera + mutante + '$funcion$;')
        a = nuevo.index('/* AD3-35 GUARDA INICIO */')
        b = nuevo.index('/* AD3-35 GUARDA ORIGINAL */') + len('/* AD3-35 GUARDA ORIGINAL */')
        c = nuevo.index(') END) /* AD3-35 GUARDA FIN */')
        restaurado = nuevo[:a] + nuevo[b:c] + nuevo[c+len(') END) /* AD3-35 GUARDA FIN */'):]
        self.assertEqual(sustituir_unico(restaurado, extension, ''), original)

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
