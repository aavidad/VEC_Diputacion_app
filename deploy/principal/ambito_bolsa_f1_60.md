# F1-60, primer corte: ámbito gobernado de la bolsa (B144)

## Qué instala

`deploy/postgresql/bolsa_llamamientos/migraciones/000144_ambito_gobernado_bolsa.up.sql`,
al final de `lista_sql_main_tras_hz37.txt`. Depende solo de B7, B79 y B95, que
ya están en el volcado HZ37. No toca el núcleo de autorización ni funciones
que cambien F1-10, F1-13 o F1-19.

- Tabla `vec_bolsa_llamamientos.ambito_bolsa`: una fila por bolsa con
  `ambito_ref` (ámbito organizativo), `unidad_ref` (unidad gestora), `origen`
  (`constitucion` o `semilla_declarada`) y `fuente_ref` (el acta o la
  semilla). Sin UPDATE, DELETE ni TRUNCATE, con RLS forzada y sin permisos
  para otros roles.
- `confirmar_carga_convoca_v2`: misma firma, propietario, configuración y
  permisos. Solo cambia el cuerpo, y únicamente si la definición instalada
  coincide con la huella de B95. Guarda el ámbito autorizado al constituir y
  deniega (42501) una confirmación o recuperación desde otro ámbito.
- `declarar_ambito_bolsa_v1(bolsa, ámbito, unidad, semilla:…)`: solo la ejecuta
  el propietario. Si se repite con el mismo par, devuelve la fila que ya hay; con otro par
  da 23505 y no escribe nada.

La migración no rellena filas: si al terminar hubiera alguna, falla.

## Bolsas que ya existían

Nadie deduce su ámbito. En la copia local, `copia_local.py crear` aplica tras
las migraciones la semilla del manifiesto privado `identidad/bolsa-bback.json`:
declara su `ambito_ref` y su `unidad_ref` solo para las bolsas de `bolsas_ref`
que existan en la base. El registro queda en `ambito-bolsas.log` de la copia.
Sin manifiesto o sin B144 no hace nada.

En un servidor nuevo, el mismo paso se ejecuta como propietario, con la
declaración que dé Administración para cada bolsa:

```sql
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SELECT vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(
  '<bolsa_ref>', '<ambito_ref>', '<unidad_ref>', 'semilla:<referencia-de-la-declaracion>');
COMMIT;
```

Una bolsa que nadie declara se queda sin ámbito. Cuando entre el segundo corte,
los actos sobre ella se denegarán.

## Comprobación

- `deploy/postgresql/bolsa_llamamientos/pruebas_sql/b144_ambito_gobernado_bolsa.sql`
  (con ROLLBACK). También lo ejecuta `ensayar.py --pruebas` en la CI.
- En la copia:
  `SELECT origen, count(*) FROM vec_bolsa_llamamientos.ambito_bolsa GROUP BY 1;`
  debe dar `semilla_declarada` con tantas bolsas del manifiesto como existan.

## Lo que falta (segundo corte)

Los actos de Bolsa (cambio de situación B100, emisión del llamamiento B17 y
confirmación de la aceptación B128) siguen tomando el ámbito fijo de la
composición. El segundo corte hace que el resolutor de contexto lea el ámbito
de esta tabla y que B128 lo coteje en SQL. Así, un acto sobre una bolsa de
otra unidad gestora se deniega, y también una bolsa sin ámbito.
