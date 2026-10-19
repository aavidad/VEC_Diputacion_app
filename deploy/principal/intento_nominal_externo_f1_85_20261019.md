# El intento denegado del aspirante coteja su sesión externa (F1-85)

Continuación de F1-84. Cuando al aspirante se le deniega una operación en el
portal externo separado, el servidor deja constancia del intento con
`registrar_intento_nominal_v1` (AD169, rehecha en AD252 y AD259). Esa función
admite el canal `externa_personal`, pero cotejaba siempre con los almacenes
internos: el contexto con CA26/CA27 y la sesión con IS20. El portal externo
guarda el contexto en `registros_contexto_externo_v2` (CA15) y la sesión en
`vec_identidad_externa_v1` (IS07), así que PostgreSQL respondía P0002 y el
servidor devolvía 503 en vez de asentar el intento y responder la denegación.

En la copia claude-f185, antes de AD299, la prueba con `AD299_REPRODUCIR`
muestra `P0002 contexto historico no disponible`. Falla ya en el cotejo del
contexto, antes de llegar al de la sesión, así que no basta con cambiar IS20
por IS22: hacen falta los dos cotejos del portal externo.

## Qué cambia

- AD299 rehace solo `registrar_intento_nominal_v1` desde su cuerpo instalado
  (AD259). Con canal `externa_personal` coteja el contexto con CA47 y la
  sesión con IS22, igual que AD289 y AD298 en la lectura de inscripción. Los
  canales `interna_corporativa` y `administracion_privilegiada` siguen con
  CA26/CA27 e IS20, sin ningún cambio.
- Los dos `CASE` van entre paréntesis dentro del mismo `IF` de AD259, de modo
  que el asiento sigue sin poder escribirse sin los dos cotejos.
- Antes de cambiar nada comprueba la huella del cuerpo instalado, su
  propietario y su ACL, y que CA47 e IS22 estén instaladas con su huella
  exacta y solo las puedan ejecutar su propietario y el de AD3 (no el
  registrador de intentos ni Bolsa). Después comprueba la huella nueva
  (`93f9c7ea…`) y que propietario, ACL, configuración y firma no cambian.
- No toca el núcleo de autorización, CA47, IS22, IS20, tablas, datos ni
  permisos. Tampoco cambia el código Go.

## Activación en una copia

1. Comprobar que la copia tiene CA47, IS22, AD259 y AD298 instaladas (F1-84).
2. Aplicar como superusuario, una sola vez,
   `lista_sql_claude_intento_externo_20261019.txt` (solo AD299). Se detiene
   con un error claro si la preimagen no coincide o si ya está aplicada.
   En la lista principal (`lista_sql_main_tras_hz37.txt`) va detrás de AD298.
3. Ensayo SQL (todo se revierte al final):

   ```sh
   psql -X -v AD299_DISPOSABLE_CLONE=1 -f deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad299_intento_nominal_externo_pg18.sql
   ```

   Debe terminar en `AD299-PRUEBA-OK`. Con `-v AD299_REPRODUCIR=1`, antes de
   aplicar AD299, muestra el P0002 original. La prueba necesita una copia con
   historia de sesiones externas del proceso integrado (la copia local la
   trae) y crea sus propios registradores de intentos, que se revierten.
4. No hace falta cambiar la configuración: la copia ya tiene un registrador
   de intentos con canal `externa_personal` para el portal externo.

No hay retirada automática. Si hiciera falta volver atrás, dirección decide
una migración nueva; no se aplica ningún DOWN sobre historia.
