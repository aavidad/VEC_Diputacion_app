"""AD3-36/T13-7/almacén-2: controles de fuente, sin ejecución PostgreSQL."""
import hashlib
import os
from pathlib import Path
import re
import unittest

import contacto_usuario_fuentes_test as anterior

ROOT = anterior.ROOT
MIG = anterior.MIG
UP = (MIG / '000036_consulta_recibo_contacto_propio.up.sql').read_text()
DOWN = (MIG / '000036_consulta_recibo_contacto_propio.down.sql').read_text()
T13 = (ROOT / 'deploy/postgresql/bolsa_registro_accesos/migraciones/000007_consulta_recibo_contacto_propio.up.sql').read_text()
STORE = (ROOT / 'deploy/postgresql/contacto_usuario_vec/migraciones/000002_consulta_recibo_contacto_propio.up.sql').read_text()


def bloque(sql, nombre):
    return sql.split('DO $' + nombre + '$')[1].split('END $' + nombre + '$;')[0]


def aplicar(sql, nombre, cuerpo):
    b = bloque(sql, nombre)
    hashes = re.findall(r"'([0-9a-f]{64})'", b)
    if hashlib.sha256(cuerpo.encode()).hexdigest() != hashes[0]:
        raise AssertionError('preimagen completa divergente')
    cambios = re.findall(r'\(\$antes(\d+)\$(.*?)\$antes\1\$,\$despues\1\$(.*?)\$despues\1\$,(\d+)\)', b, re.S)
    if not cambios:
        raise AssertionError('parches ausentes')
    for _, viejo, nuevo, veces in cambios:
        if cuerpo.count(viejo) != int(veces):
            raise AssertionError('fragmento no exacto')
        cuerpo = cuerpo.replace(viejo, nuevo)
    if hashlib.sha256(cuerpo.encode()).hexdigest() != hashes[-1]:
        raise AssertionError('postimagen completa divergente')
    return cuerpo


def revalidacion35():
    cuerpo = anterior.cuerpo_revalidacion()
    b = bloque(anterior.UP, 'revalidacion')
    a = cuerpo.index('BEGIN\n    IF pg_catalog.current_setting(') + 13
    z = cuerpo.index(' THEN\n        RAISE EXCEPTION USING')
    cuerpo = cuerpo[:a] + b.split('placing $inicio$')[1].split('$inicio$')[0] + cuerpo[a:z] + ') END) /* AD3-35 RV FIN */' + cuerpo[z:]
    for viejo, nuevo in re.findall(r'nueva:=replace\(nueva,\$antes\$(.*?)\$antes\$,\$despues\$(.*?)\$despues\$\)', b, re.S):
        cuerpo = anterior.sustituir_unico(cuerpo, viejo, nuevo)
    return cuerpo


def nucleo35(fuente32):
    cabecera, cuerpo = anterior.nucleo_hasta_32(fuente32)
    b = bloque(anterior.UP, 'nucleo')
    colocacion = b.split('nueva:=overlay(definicion placing')[1].split('FROM inicio+13')[0]
    textos = [v.replace("''", "'").replace('\\n', '\n') for v in re.findall(r"E'((?:[^']|'')*)'", colocacion)]
    a = cuerpo.index('BEGIN\n    IF pg_catalog.current_setting(') + 13
    z = cuerpo.index(' THEN\n        RAISE EXCEPTION USING')
    cuerpo = cuerpo[:a] + ''.join(textos[:-1]) + cuerpo[a:z] + textos[-1] + cuerpo[z:]
    marca = "       )\n       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'"
    return cabecera, anterior.sustituir_unico(cuerpo, marca, anterior.literal(b, 'extension') + marca)


class FuentesReciboContacto(unittest.TestCase):
    def test_parseo_fuentes_y_fixture(self):
        try:
            from pglast import parse_sql, parse_plpgsql
        except ImportError:
            self.skipTest('pglast opcional ausente')
        archivos = []
        for area, numero in [('autorizacion_atestada_v3', '000036'), ('bolsa_registro_accesos', '000007'), ('contacto_usuario_vec', '000002')]:
            archivos += list((ROOT / 'deploy/postgresql' / area / 'migraciones').glob(numero + '_consulta_recibo_contacto_propio.*.sql'))
        archivos.append(MIG.parent / 'pruebas_sql/contacto_recibo_codec.sql')
        self.assertEqual(len(archivos), 7)
        for archivo in archivos:
            with self.subTest(archivo=archivo.name):
                s = '\n'.join(l for l in archivo.read_text().splitlines() if not l.startswith('\\'))
                parse_sql(s)
                parse_plpgsql(s)

    def test_prosrc_nucleo_completo_y_reversion(self):
        try:
            from pglast import parse_sql, parse_plpgsql
        except ImportError:
            self.skipTest('pglast opcional ausente')
        fuente32 = os.environ.get('VEC_CONTACTO_AD3_32_UP')
        if not fuente32:
            self.skipTest('se requiere referencia final AD3-32')
        fuente32 = Path(fuente32)
        self.assertEqual(hashlib.sha256(fuente32.read_bytes()).hexdigest(), '2d2225f31da45b1a62df230a27b0c61c695228415951f21006cfd9c7e8050bc6')
        cabecera, cuerpo = nucleo35(fuente32)
        posterior = aplicar(UP, 'nucleo', cuerpo)
        parse_sql(cabecera + posterior + '$funcion$;')
        parse_plpgsql(cabecera + posterior + '$funcion$;')
        self.assertEqual(aplicar(DOWN, 'nucleo', posterior), cuerpo)
        # Una edición ajena, aunque sólo cambie whitespace, bloquea la migración.
        with self.assertRaisesRegex(AssertionError, 'preimagen completa'):
            aplicar(UP, 'nucleo', cuerpo + '\n')
        # El núcleo no transforma la consulta de recibo en alta o consulta de envío.
        self.assertIn("p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_usuario_recibo'\n"
                      "               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.contacto_usuario.recibo.v1'", posterior)

    def test_prosrc_revalidacion_completa_y_frescura_intacta(self):
        try:
            from pglast import parse_sql, parse_plpgsql
        except ImportError:
            self.skipTest('pglast opcional ausente')
        original = revalidacion35()
        posterior = aplicar(UP, 'revalidacion', original)
        base = (MIG / '000005_revalidacion_final_consultas_rrhh_v3.up.sql').read_text()
        cabecera = base[base.index('CREATE FUNCTION\nvec_autorizacion_atestada_v3.revalidar_consumo'):base.index('AS $funcion$')]
        parse_sql(cabecera + 'AS $funcion$' + posterior + '$funcion$;')
        parse_plpgsql(cabecera + 'AS $funcion$' + posterior + '$funcion$;')
        self.assertEqual(aplicar(DOWN, 'revalidacion', posterior), original)
        cola = original[original.index('    v_viva_en :='):]
        self.assertTrue(posterior.endswith(cola))
        self.assertIn('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(', cola)
        for fragmento in ('.revocacion_clave_capacidad r', '.revocacion_raiz r', '.revocacion_configuracion r',
                          'v_ahora := pg_catalog.clock_timestamp();'):
            self.assertIn(fragmento, cola)
        with self.assertRaisesRegex(AssertionError, 'postimagen completa'):
            aplicar(UP.replace("v_finalidad := 'gestion_contacto_propio';", "v_finalidad := 'envio_llamamiento';"), 'revalidacion', original)

    def test_lectura_historica_exacta_sin_efecto_de_guardado(self):
        f = STORE.split('AS $f$')[1].split('END $f$;')[0]
        self.assertNotRegex(f, r'\b(INSERT|UPDATE|DELETE|TRUNCATE)\b')
        self.assertNotIn('vec_contacto_usuario_v1.actual', f)
        self.assertNotRegex(f, r'v\.(cifrado|nonce|clave_ref)\b')
        self.assertIn('WHERE v.sujeto_ref=v_sujeto AND v.version=v_version', f)
        self.assertLess(f.index('registrar_y_consumir_recibo_contacto_usuario_v3_atestada'), f.index("set_config('vec.contacto.sujeto_ref'"))
        self.assertLess(f.index("::jsonb->>'persona_ref'"), f.index("set_config('vec.contacto.sujeto_ref'"))
        self.assertLess(f.index('registrar_consulta_recibo_contacto_v1'), f.index('revalidar_recibo_contacto_usuario_v3_atestada'))
        self.assertLess(f.index('revalidar_recibo_contacto_usuario_v3_atestada'), f.index('RETURN QUERY'))
        self.assertIn('v_encontrado:=FOUND;', f)
        self.assertIn('v_original_bytes:=v_original.auditoria_central;', f)
        self.assertIn("v_original_bytes bytea:=''::bytea", f)

    def test_auditoria_original_se_liga_sin_reescribir(self):
        f = T13.split('AS $f$')[1].split('END $f$;')[0]
        self.assertIn("original_hash:=encode(sha256(p_recibo_original),'hex')", f)
        self.assertIn("'recibo_encontrado',CASE WHEN encontrado THEN 'true' ELSE 'false' END", f)
        self.assertIn("'recibo_original_ref',original_ref,'recibo_original_sha256',original_hash", f)
        self.assertIn("'authorization_ref',p_decision_ref", f)
        self.assertIn('registrar_interno_v1(entrada)', f)
        self.assertNotRegex(f, r'original\s*:=\s*jsonb_(build|set)')
        self.assertIn("original->>'correlation_ref' IS NOT DISTINCT FROM a->>'correlation_ref'", f)

    def test_concesiones_nominales_sin_reader(self):
        self.assertIn("('consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','vec_contacto_usuario_writer')", STORE)
        self.assertIn("contacto_sesion_nominal_v1('contacto_usuario_alta') IS NOT TRUE", UP)
        self.assertNotRegex(UP + T13 + STORE, r'GRANT .* TO vec_contacto_usuario_reader')
        self.assertNotRegex(UP + T13 + STORE, r'CREATE (ROLE|TABLE|POLICY)\b')
        for sql in (UP, T13, STORE, DOWN):
            self.assertIn('vec_contacto_usuario_v1:dependencias:v1', sql)
        self.assertNotRegex(DOWN, r'\b(DELETE FROM|TRUNCATE|CASCADE)\b')
        self.assertIn("->>'audiencia_consumo'='vec.contacto_usuario.recibo.v1'", DOWN)


if __name__ == '__main__':
    unittest.main()
