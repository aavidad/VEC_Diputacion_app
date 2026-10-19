"""Semilla declarada del ámbito de las bolsas de una copia local (F1-60, B144).

Las bolsas constituidas antes de B144 no guardan su ámbito. En la copia, el
manifiesto privado `identidad/bolsa-bback.json` declara qué bolsas gestiona el
perfil de RRHH de Bolsa y con qué ámbito y unidad gestora las opera. Esta
semilla copia esa declaración a `ambito_bolsa` con la función del propietario,
solo para las bolsas que existan en la base. Una bolsa que no figure en el
manifiesto queda sin ámbito.

La copia usa datos inventados y un solo perfil de RRHH de Bolsa: el generador
del material pone en `bolsas_ref` todas las bolsas vigentes, así que todas
reciben el mismo ámbito. Con varias unidades gestoras, cada declaración sale de
Administración, no de este manifiesto.
"""
import json
import re

DECLARACION = 'semilla:copia-local:bolsa-bback'
REFERENCIA = re.compile(r'[A-Za-z0-9:._/-]{1,256}')


def sentencias(manifiesto):
    """Devuelve el SQL de la semilla o None si el manifiesto no declara bolsas."""
    ambito, unidad = manifiesto.get('ambito_ref'), manifiesto.get('unidad_ref')
    bolsas = manifiesto.get('bolsas_ref')
    if not isinstance(bolsas, list) or not bolsas:
        return None
    valores = [ambito, unidad, *bolsas]
    if not all(isinstance(v, str) and REFERENCIA.fullmatch(v) for v in valores):
        raise ValueError('El manifiesto de Bolsa declara referencias no válidas')
    lista = ','.join(f"'{b}'" for b in bolsas)
    return f"""\\set ON_ERROR_STOP on
SELECT pg_catalog.to_regclass('vec_bolsa_llamamientos.ambito_bolsa') IS NOT NULL AS b144 \\gset
\\if :b144
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SELECT pg_catalog.count(*) AS bolsas_declaradas FROM (
 SELECT vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(r.bolsa_ref,'{ambito}','{unidad}','{DECLARACION}')
 FROM pg_catalog.unnest(ARRAY[{lista}]::text[]) AS r(bolsa_ref)
 WHERE EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida b WHERE b.bolsa_ref=r.bolsa_ref)
) AS declaradas;
COMMIT;
\\endif
""".encode()


def declarar(copia):
    """Aplica la semilla en la copia si su material trae el manifiesto de Bolsa."""
    ruta = copia.material / 'identidad' / 'bolsa-bback.json'
    if not ruta.is_file():
        return
    sql = sentencias(json.loads(ruta.read_text()))
    if sql is not None:
        copia.sql_bytes(sql, 'ambito-bolsas.log')
