# Cuenta y vínculo de sesión ADMIN — IS16

IS16 conecta la cuenta privilegiada nominativa con una sesión real de IS y con
la selección vigente de CA31. No concede perfiles ni cambia el núcleo V2.
Requiere IS14/15, CA31 y la familia de auditoría AD192 ya corregida por AD194.
CA36 consume después el vínculo desde su propietario, con otro LOGIN y pool.
El orden de instalación está en
`deploy/principal/lista_sql_codexk_admin_runtime_20261005.txt`: AD194, IS16 y
CA36. Cada una se aplica una sola vez y ninguna tiene DOWN.

El candidato IS16 tiene dos revisiones independientes. Su estructura se instaló
una vez en el clon aislado
el 4 de octubre de 2026. Aún no hay recuperación con una sesión real ni
recorrido HTTP acreditados.

## Contratos y permisos

El grupo `vec_identidad_sesiones_v1_admin_perfiles_runtime` tiene cuatro
ejecutables: acreditación, resolución de cuenta, vínculo y registro del fallo
de fuente privada. Sólo recibe USAGE de IS y CONNECT; no recibe tablas. El
LOGIN, proceso, entorno, host, audiencia, espacio de identidad y vigencia se
configuran fuera de Git. El grupo y el LOGIN de CA36 son distintos.

Las fachadas de resolución y vínculo reciben la observación mTLS confiable y
referencias de evento/correlación generadas en el servidor. Devuelven un objeto
cerrado `{estado, datos}` y un acuse AD192 de cinco campos. Go valida referencia,
secuencia, huella, correlación e instante antes del COMMIT o de usar los datos.
La denegación se confirma antes de devolver el error funcional. Un acuse
inválido, un fallo de transacción o un COMMIT incierto no devuelven éxito.
BEGIN, SET, RELEASE y COMMIT fallidos devuelven configuración incompleta,
sin datos ni reintento. Un 42501 de QUERY sin envelope y acuse confirmados
tampoco se presenta como denegación funcional.

SQL devuelve referencias opacas y las coordenadas/HMAC del acto inicial exacto.
`NuevoPostgreSQLConFuenteADMIN` exige el proveedor original de SujetoID, CuentaID
y CuentaOrdinariaID, y el seudonimizador existente. Coteja las tres huellas y
sus coordenadas, sin derivar identificadores desde PersonaRef ni desde digests.
El constructor antiguo conserva su firma y deniega sin esa fuente.

El lector opcional de archivo privado exige SHA aprobado, objeto cerrado,
versión 1 y un máximo de 32 KiB. Reutiliza el protocolo privado existente:
ruta absoluta limpia, padre propio 0700, sin Git ni enlaces en sus ancestros,
raíz `os.Root` y archivo regular propio 0600 que no sea un enlace.
El archivo lo entrega el productor original. No hay ejemplo de identidad real
ni generación alternativa de preimágenes. Cuenta y fuente redactan JSON,
formato y logs para impedir que salgan identificadores o material SQL.

Si el proveedor privado o un HMAC fallan, Go revierte el SAVEPOINT que contiene
la resolución positiva y su auditoría. Fuera de ese SAVEPOINT, dentro de la
misma transacción, `rechazar_fuente_cuenta_admin_v1` registra el error AD192.
Lo mismo ocurre al vincular: si SQL devuelve un vínculo favorable que Go no
reconoce como el pedido, se revierte el SAVEPOINT con el vínculo y su acuse y
se registra el error con la acción `vincular_sesion_admin`. Un fallo de ese
acuse o del COMMIT sigue cerrado. Un conflicto de serialización (40001) o un
interbloqueo (40P01) se devuelven como conflicto, no como configuración
incompleta.

## Vínculo exacto y recuperación

El vínculo `vis_` usa 128 bits CSPRNG y conserva versión, SHA, autenticación,
sesión, persona, ambas cuentas, perfil, certificado/CA, política, revisión de
selección, control de sesión y fuente. Es inmutable. El replay sólo admite el
mismo plan, referencia, evento, operador y acuse original; nunca elige otra
sesión por cuenta. Tras cotejar el acuse revalida configuración, cuenta, sesión
y ventanas del vínculo dentro de la misma subtransacción. El evento original
permanece intacto. El adaptador no reinvoca un vínculo ante COMMIT incierto.

La fachada propietaria para CA36 es:

```text
cotejar_vinculo_sesion_admin_v1(
  referencia text, version numeric, huella_sha256 text,
  autenticacion_ref text, sesion_ref text, cuenta_ref text,
  perfil_ref text, solicitado_en timestamptz, metodo_observado text
) -> jsonb
```

Devuelve exactamente doce claves: `referencia`, `version`, `huella_sha256`,
`autenticacion_ref`, `sesion_ref`, `persona_ref`, `cuenta_ref`, `perfil_ref`,
`seleccion_revision`, `fuente_ref`, `fuente_sha256` y `vigente_hasta`.
La ausencia o una autoridad no vigente devuelven NULL. El método debe coincidir
con la sesión real, certificado o DNIe; no basta con pertenecer al mismo conjunto.
Sólo el owner CA recibe EXECUTE, nunca el LOGIN ni el grupo CA.

En SERIALIZABLE coteja también CA31 mediante su fachada. En READ COMMITTED
coteja IS certificado/CA/CRL/política/cuentas/sesión/control actuales; CA36
coteja su selección CA31 actual bajo sus propios cerrojos. Ese resultado IS
por sí solo no acredita un contexto recuperado. CA36 conserva registro,
enlace y acuse original exactos; la recuperación no añade auditoría.

`ContextoConVinculoSesionADMIN` y `VinculoSesionADMINDeContexto` transmiten la
ligadura tipada antes de `CrearVinculoAutenticacionActorV2ConResultado`, sin
ampliar puertos generales ni aceptar datos de la petición HTTP.

## Comprobaciones y dependencia pendiente

Las pruebas focales Go cubren discrepancias de las tres huellas, coordenadas y
fuente; acuses ajenos/incompletos/futuros; JSON con claves repetidas o ajenas;
y redacción de identificadores. Usan dobles aislados del contrato, no una
sesión productiva. El vector SQL de ACL y ausencia bajo ambas transacciones
terminó con salida 0 en el clon.

El ensayo PostgreSQL del replay sigue pendiente. Con un vínculo real, debe
retener el cerrojo `vec:is16:sesion:<sesion_ref>`, iniciar el replay exacto y
comprobar que espera. Si una ventana de configuración, sesión o vínculo vence
durante esa espera, al liberar el cerrojo no puede devolver el recibo permitido.
Se comparan después las huellas del evento y vínculo originales, sin cambiar
el reloj ni renovar fuentes para superar el caso.

Las preimágenes originales del ejercicio se han perdido: el plan y el material
HMAC conservados sólo contienen referencias y digests. Dirección autorizó para
la retoma un nuevo juego sintético por el circuito oficial de fuentes,
titularidad, huellas y CAS, con sus originales guardados en un fixture privado.
El nuevo juego se generó por ese circuito y conserva sus originales privados.
El acceso nominal aún debe ensayarse con una selección propia vigente.

## Resultado estructural del 4 de octubre de 2026

El código `2d24894cf56b2a08aa55a45439b2b1c4e387c95f` y SQL SHA256
`2f88dafdcf8f2a6038cb42bc5879bc5b143f23e240184b6c040352e721bd5c92`
recibieron dos GO exactos. Las pruebas focales y race, vet y Semgrep local pasan.

El frío posterior a AD192 se restauró en un clon aislado. CA31 e IS14 se
instalaron una vez, en ese orden, y sus fachadas quedaron disponibles. Después,
IS16 con el SQL indicado arriba terminó su UP con salida 0. El vector
`runtime_admin_000016_acl.sql` también terminó con salida 0. CA36 se instaló
después y pasó su vector de ACL; estos resultados no corresponden a la base
principal.

Antes y después de las migraciones se conservaron las 6.254 filas de auditoría,
su huella completa y la cabeza de cadena. El núcleo de auditoría AD192 mantuvo
la huella de fuente `b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45`;
el CHECK `auditoria_tipo_disjunto_v4` mantuvo
`0f6d15ebdc61ba5ff67903bde824db878a6396fa593029e98498946e2d8d1331`.
No se reaplicó AD192 ni se ejecutó DOWN.

El ensayo acreditó estructura y ACL. No creó un vínculo favorable con sesión
real ni un nuevo evento nominal. Las identidades originales no se reconstruyen:
el nuevo juego sintético ya pasó el circuito oficial de fuentes, unidad y
bootstrap 2+1, con originales privados en modo 0600. Las asignaciones del primer
ejercicio han caducado y no se prolongaron. Quedan pendientes el ensayo de espera y replay con sesión real,
los LOGIN y pools segregados, la garantía alta, PDP y el montaje ADMIN.


## Composición nominal preparada

La rama de trabajo conecta `NuevoPostgreSQLConFuenteADMIN` con el productor
original aprobado por SHA y el proveedor HMAC existente. El alias ordinario se
coteja con propósito `cuenta`, separado de las cinco huellas de alta; no se
reescriben los alias persistidos.

`vec-admin` carga una configuración nominal adicional y entrega a CA36 un pool
distinto del de cuentas, registro y revalidación. La frontera existente verifica
mTLS, CRL y vínculo del canal. Para el ensayo se prepararon con la misma CA
sintética una CRL firmada y un certificado de servidor, además del archivo de
originales de las dos cuentas de Aplicación. No son una sesión favorable ni una
prueba de PDP. Los detalles de configuración están en `cmd/vec-admin/README.md`.


## Dependencia de AD194

IS16 deja su auditoría en AD192. AD192, tal como está instalada, falla cuando
se repite un evento que ya existe: devuelve la secuencia como `numeric` en una
salida declarada `bigint` y PostgreSQL aborta la llamada. AD194 corrige las dos
funciones internas afectadas.

Antes de crear nada, IS16 comprueba que esa versión defectuosa ya no está en
uso y, si sigue, se detiene con «IS16: falta AD194». La comprobación compara la
fuente instalada de AD192 con su huella conocida, así que no depende del texto
exacto de AD194. CA36 hace la misma comprobación.

## Alta del LOGIN de cuentas ADMIN

`vec-admin` usa un LOGIN propio para el pool de cuentas ADMIN. IS16 sólo lo
acepta si el DBA ha preparado tres cosas, como superusuario y en una única
transacción:

1. El LOGIN, sin privilegios especiales y miembro del grupo
   `vec_identidad_sesiones_v1_admin_perfiles_runtime` con herencia, sin SET y
   sin ADMIN. No puede pertenecer a otro rol, ser dueño de objetos, tener
   permisos propios ni ajustes con `ALTER ROLE … SET`. El grupo admite un solo
   miembro: si hay dos, IS16 no acredita a ninguno.
2. Su fila en `vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1`.
3. Su fila en `vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1`,
   la configuración de AD192. El `proceso` tiene que ser el mismo en las dos
   filas, porque IS16 envía el suyo en cada evento y AD192 rechaza el evento si
   no coincide.

```sql
BEGIN;
CREATE ROLE <login> LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_admin_perfiles_runtime TO <login>
  WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
INSERT INTO vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1
  (identidad_login, proceso, entorno, host_admin, audiencia, espacio_identidad, vigente_hasta)
VALUES ('<login>', '<proceso>', '<desarrollo|cidonia>', '<host>', '<audiencia>',
        'https://<espacio de identidad>', '<fin de vigencia>');
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1
  (login_nombre, proceso, canal, vigente_desde, vigente_hasta)
VALUES ('<login>', '<proceso>', 'administracion_privilegiada', clock_timestamp(), '<fin de vigencia>');
COMMIT;
```

Valores admitidos en la fila de IS16:

- `proceso`: minúsculas, cifras, punto, guion y guion bajo; empieza por letra
  y tiene de 2 a 80 caracteres.
- `entorno`: `desarrollo` o `cidonia`.
- `host_admin`: nombre de host en minúsculas, de 4 a 253 caracteres, con el
  mismo formato que la política de certificado de IS15.
- `host_admin` y `audiencia`: los mismos que observa la frontera mTLS de
  `vec-admin` (véase `cmd/vec-admin/README.md`). Si no coinciden, cada petición
  se deniega.
- `espacio_identidad`: empieza por `https://`, sin espacios y con 508
  caracteres como máximo.
- `vigente_hasta`: una fecha finita posterior al momento del alta.

Para comprobar el alta, el DBA se conecta con el nuevo LOGIN, abre una
transacción `SERIALIZABLE` con zona `UTC` y ejecuta
`SELECT * FROM vec_identidad_sesiones_v1.acreditar_runtime_admin_perfiles_v1();`.
Debe devolver el LOGIN, `true` y el proceso. Es la misma llamada que hace
`vec-admin` al arrancar. La contraseña o el certificado del LOGIN quedan fuera
de Git.

## Cuando caduca la vigencia

Las dos tablas de configuración no admiten cambios ni borrados, y su clave es
el LOGIN. Una fila caducada no se puede alargar ni sustituir por otra del mismo
LOGIN. Cuando vence `vigente_hasta` en cualquiera de las dos filas, o el
`VALID UNTIL` del LOGIN si se fijó, IS16 deja de acreditarlo: `vec-admin` no
arranca y las peticiones en curso se deniegan.

La renovación se hace con un LOGIN nuevo, preferiblemente antes del
vencimiento y en una ventana de mantenimiento:

Aquí se retira primero el grupo al LOGIN anterior, al revés que en CA36,
porque el grupo de IS16 sólo admite un miembro. Por eso `vec-admin` se queda
sin servicio desde el COMMIT de la renovación hasta que se reinicia con el
LOGIN nuevo.

1. En una sola transacción, retirar el grupo al LOGIN anterior
   (`REVOKE vec_identidad_sesiones_v1_admin_perfiles_runtime FROM <anterior>;`)
   y dar de alta el nuevo con el procedimiento de arriba. Un nombre con la
   fecha ayuda a distinguirlos.
2. Cambiar la conexión del pool de cuentas ADMIN de `vec-admin` al LOGIN nuevo
   y reiniciar el proceso.
3. Dejar el LOGIN anterior sin conexión (`ALTER ROLE <anterior> NOLOGIN;`) y no
   borrarlo. Sus filas de configuración, sus vínculos y sus eventos de
   auditoría lo citan por nombre y se conservan como historia.

Los vínculos de sesión creados con el LOGIN anterior no se pueden repetir con
el nuevo. Esas sesiones tienen que volver a identificarse.

## Corrección del 5 de octubre de 2026

El SQL del 4 de octubre declaraba `espacio_identidad` con una repetición
`{1,500}`. PostgreSQL no admite repeticiones de más de 255, así que cualquier
alta en `config_runtime_admin_perfiles_v1` fallaba con «invalid repetition
count(s)». El ensayo de ese día sólo cubrió estructura y permisos y no llegó a
insertar ninguna fila. Ahora el patrón no lleva cota y la longitud se limita
aparte. IS16 exige además AD194, `host_admin` tiene formato y longitud
comprobados, y `rechazar_fuente_cuenta_admin_v1` recibe la acción rechazada
(resolver o vincular) para auditar también un vínculo que Go no reconoce.

El SQL corregido tiene SHA256
`bd6a3b5507a556f53623aaff2df34e14e88e9d2c907b4d42887ff052c6a3f73f`. Se ensayó
en un clon desechable de la copia fría de la principal (PostgreSQL 18.4, sin
red), donde se aplicaron AD194, IS16 y CA36 una vez cada una. Sin AD194, IS16
se detuvo con «falta AD194». Con AD194 instalada, el alta de un LOGIN de ensayo
siguiendo el procedimiento anterior funcionó y la acreditación devolvió `true`.
Repetir un mismo evento de rechazo devolvió el acuse original sin añadir otra
fila. El vínculo favorable con una sesión real sigue pendiente, porque la
copia fría no tiene fuentes de arranque.
