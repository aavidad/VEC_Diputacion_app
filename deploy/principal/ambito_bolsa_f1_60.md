# F1-60: ámbito gobernado de la bolsa (B144 y B145)

## Qué instala

`deploy/postgresql/bolsa_llamamientos/migraciones/000144_ambito_gobernado_bolsa.up.sql`,
al final de `lista_sql_main_tras_hz37.txt`. Depende solo de B7, B14, B79 y B95,
que ya están en el volcado HZ37. No toca el núcleo de autorización ni las
funciones que cambian F1-10, F1-13 o F1-19.

- Tabla `vec_bolsa_llamamientos.ambito_bolsa`: una fila por bolsa con
  `ambito_ref` (ámbito organizativo), `unidad_ref` (unidad gestora), `origen`
  (`constitucion` o `semilla_declarada`) y `fuente_ref` (el acta o la
  semilla). No admite UPDATE, DELETE ni TRUNCATE, tiene RLS forzada y ningún
  otro rol tiene permisos sobre ella ni sobre su tipo.
- `confirmar_carga_convoca_v2`: conserva firma, propietario, configuración y
  permisos. Solo cambia el cuerpo, y únicamente si la definición instalada
  coincide con la huella de B95. Guarda el ámbito autorizado al constituir.
  Antes de consumir la decisión deniega con 42501:
  - una confirmación o una recuperación desde otro ámbito;
  - una constitución nueva sobre una bolsa que ya existía sin ámbito;
  - una carga que sustituiría (B14, por categoría) una bolsa vigente de otro
    ámbito o sin ámbito.
- `declarar_ambito_bolsa_v1(bolsa, ámbito, unidad, semilla:…)`: solo la
  ejecuta el propietario. Si se repite con el mismo par, devuelve la fila que
  ya hay; con otro par da 23505 y no escribe nada. Comparte cerrojo con la
  confirmación.

La migración no rellena filas: si al terminar hubiera alguna, falla. Tampoco se
puede aplicar dos veces: la segunda vez falla en la comprobación previa.

## Bolsas que ya existían

Nadie deduce su ámbito. En la copia local, `copia_local.py crear` aplica tras
las migraciones la semilla del manifiesto privado `identidad/bolsa-bback.json`:
declara su `ambito_ref` y su `unidad_ref` solo para las bolsas de `bolsas_ref`
que existan en la base. El registro queda en `ambito-bolsas.log` de la copia.
Sin manifiesto o sin B144 no hace nada. En la copia con el volcado HZ37 quedan
declaradas las doce bolsas vigentes; las cuatro sustituidas se quedan sin
ámbito.

La semilla va antes de abrir cargas de CONVOCA: mientras la bolsa vigente de
una categoría no tenga ámbito, una carga nueva de esa categoría se deniega.

En un servidor nuevo, el mismo paso se ejecuta como propietario, con la
declaración que dé Administración para cada bolsa:

```sql
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SELECT vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(
  '<bolsa_ref>', '<ambito_ref>', '<unidad_ref>', 'semilla:<referencia-de-la-declaracion>');
COMMIT;
```

Una bolsa que nadie declara se queda sin ámbito, y desde el segundo corte
(B145) todos los actos sobre ella se deniegan.

## Comprobación

- `deploy/postgresql/bolsa_llamamientos/pruebas_sql/b144_ambito_gobernado_bolsa.sql`
  (con ROLLBACK). También la ejecuta `ensayar.py --pruebas` en la CI.
- En la copia:
  `SELECT origen, count(*) FROM vec_bolsa_llamamientos.ambito_bolsa GROUP BY 1;`
  debe dar `semilla_declarada` con tantas bolsas del manifiesto como existan.

## Segundo corte: los actos cotejan el ámbito (B145)

`deploy/postgresql/bolsa_llamamientos/migraciones/000145_ambito_actos_bolsa.up.sql`,
al final de la lista y después de B144. Cambia cuatro cosas:

- `leer_ambito_bolsa_v1(bolsa)`: devuelve el ámbito y la unidad de la bolsa, o
  ninguna fila si no tiene. Es la única lectura de `ambito_bolsa` concedida al
  ejecutor web. El resolutor de contexto de Bolsa la usa en todos los actos
  sobre una bolsa: cambio de situación, ficha y contactos, emisión del
  llamamiento, ofertas y confirmación de la aceptación. Pone ese par en el
  recurso que autoriza la decisión V3, y la decisión lo compara con la
  asignación real del actor. Una bolsa de otra unidad gestora se deniega aunque
  el actor tenga permiso en la suya. Una bolsa sin ámbito se deniega siempre,
  también si figura en el manifiesto de la copia.
- `confirmar_aceptacion_directa_v1` (B128) coteja además en SQL que el ámbito
  del recurso autorizado es el de la bolsa. Si no lo es, o la bolsa no tiene
  ámbito, responde 42501 antes de consumir la decisión.
- `confirmar_carga_convoca_v2` deniega con 42501 cualquier confirmación o
  recuperación de una bolsa que ya existe sin ámbito. Antes la recuperación de
  una carga anterior a B144 devolvía el recibo original.
- `constituir_bolsa_v1` deja de estar concedida al rol constituidor de B80.
  Se retira también la orden `vec-server constituir-bolsa` y su conexión
  `VEC_BOLSA_CONSTITUCION_DATABASE_URL`. Una bolsa nueva solo se constituye
  con la carga gobernada de CONVOCA, que guarda su ámbito.

Las tres funciones conservan firma, propietario, configuración y permisos, con
huellas fijas de lo instalado y de lo nuevo. Reaplicar la migración falla en la
comprobación previa.

### Por qué se deniega también la recuperación sin ámbito

Se consultó como decisión de permisos (Claude con Fable, 19/10/2026). La
recuperación no escribe nada, pero devuelve el recibo de la constitución y sus
vínculos y deja auditado un acceso como válido. La decisión V3 se emite con el
ámbito del actor y no hay ámbito de la bolsa con el que compararlo: una
autorización que no se puede comprobar se trata como denegada. Además, con este
corte cualquier otro acto sobre esa bolsa ya se deniega, y dejar abierta solo la
recuperación sería una excepción. El coste es pequeño: la bolsa necesita su
declaración para usarse, y en un servidor nuevo no habrá bolsas anteriores a
B144. Si una carga se interrumpió antes de B144, Administración declara primero
el ámbito y después se repite la confirmación.

### Por qué se retira la constitución por línea de órdenes

Era una segunda vía para el mismo acto, sin decisión V3, sin la custodia del
acta y sin ámbito. Exigirle ámbito obligaba a repetir en esa orden la
autorización que ya hace la carga gobernada. Nada la usaba ya. El rol
constituidor conserva `registrar_vinculos_candidato_v1`, que necesita
`rellenar-vinculos-bolsa`.

### Comprobación del segundo corte

- `deploy/postgresql/bolsa_llamamientos/pruebas_sql/b145_ambito_actos_bolsa.sql`
  (con ROLLBACK), también en `ensayar.py --pruebas`.
- Prueba Go de aceptación:
  `TestFichaOperacionesBolsaDeOtraUnidadGestoraSeDeniega`
  (`internal/app/bootstrap/bolsa_capacidades_ficha_operaciones_test.go`).
- En la copia: una carga nueva de CONVOCA deja la fila con origen
  `constitucion`, y un acto sobre una bolsa declarada en otra unidad se
  deniega aunque el actor tenga permiso en la suya.

### Lo que queda fuera

- La carga de CONVOCA y el borrador de llamamiento siguen tomando el ámbito
  del perfil del manifiesto, porque todavía no hay bolsa de la que leerlo.
- Con varias unidades gestoras hará falta una política de transiciones por
  unidad: B128 compara además el ámbito con el de la política vigente.
- El servicio Go de constitución directa (`constitucion.NuevoServicio` y el
  método `Constituir` del repositorio) ya no tiene llamadas fuera de sus
  pruebas; se puede borrar en una limpieza aparte.
