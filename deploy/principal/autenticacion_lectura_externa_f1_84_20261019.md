# La lectura de inscripción del aspirante coteja su sesión externa (F1-84)

Fallo encontrado al ensayar F1-40. En el portal externo, la consulta de
convocatorias abiertas del aspirante obtenía la autorización y después
respondía 503. La función que deja constancia de la lectura
(`registrar_lectura_inscripcion_v1`, AD228, rehecha en AD289) ya buscaba el
contexto en el almacén externo (CA47), pero la sesión la seguía buscando solo
en el almacén interno de identidad (IS20). El portal externo separado guarda
sus sesiones en `vec_identidad_externa_v1` (IS07), así que PostgreSQL
respondía P0002 y toda lectura de inscripción del aspirante fallaba.

## Qué cambia

- IS22 añade `cotejar_autenticacion_historica_portal_externo_v1`. Solo acepta
  vínculos de superficie `externa_personal`. Si la sesión está en el almacén
  externo, comprueba la sesión registrada, su huella de control, la cuenta en
  la revisión que usó la sesión, y que cada dato del vínculo coincida. Como
  IS20 y CA47, es un cotejo histórico: no mira el reloj ni el estado actual
  de la cuenta, porque la vigencia en vivo ya la exige la decisión de la misma
  petición. Una sesión que ya había caducado cuando se resolvió el contexto la
  rechaza AD228 por su ventana. Si la sesión está en el almacén interno
  (aspirantes del proceso integrado), delega en IS20 sin cambios, salvo que la
  sesión sea privilegiada o no externa, o que su cuenta pertenezca al almacén
  externo. Una sesión cuya referencia de sesión, autenticación o aserción esté
  en los dos almacenes se rechaza, y también una que no esté en ninguno. Solo
  pueden ejecutarla su propietario y el de AD3.
- AD298 rehace únicamente `registrar_lectura_inscripcion_v1` desde su cuerpo
  instalado (AD289). Con canal `externa_personal` usa el cotejo nuevo; con
  `interna_corporativa` (RRHH) sigue con IS20. Conserva propietario, permisos,
  configuración y firma. No toca el núcleo de autorización ni otras lecturas.

## Activación en una copia

1. Comprobar que la copia tiene IS20, CA47 y AD289 instaladas.
2. Aplicar como superusuario, en este orden y una sola vez,
   `lista_sql_claude_autenticacion_lectura_externa_20261019.txt` (IS22 y
   después AD298). Cada migración se detiene con un error claro si la
   preimagen no coincide o si ya está aplicada.
3. Ensayo SQL (todo se revierte al final):

   ```sh
   psql -X -v AD298_DISPOSABLE_CLONE=1 -f deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad298_autenticacion_lectura_externa_pg18.sql
   ```

   Debe terminar en `AD298-PRUEBA-OK`. Con `-v AD298_REPRODUCIR=1`, antes de
   aplicar la lista, muestra el P0002 original.
4. No hace falta cambiar la configuración. Después, con el aspirante
   sintético, abrir en el portal externo las convocatorias abiertas: debe
   cargar y quedar un asiento `lectura_inscripcion` con canal
   `externa_personal`.

No hay retirada automática. Si hiciera falta volver atrás, dirección decide
una migración nueva; no se aplica ningún DOWN sobre historia.

## Continuación

`registrar_intento_nominal_v1` (AD169, AD252 y AD259) tenía el mismo fallo en
los intentos denegados del portal externo. Lo arregla F1-85 con AD299; ver
`intento_nominal_externo_f1_85_20261019.md`.
