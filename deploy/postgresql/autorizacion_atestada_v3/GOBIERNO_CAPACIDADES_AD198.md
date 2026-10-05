# Gobierno de las claves de capacidad de Administración (AD198)

AD188 publica cada día una configuración nueva junto con las dos claves de
usuarios (listar y consultar). El lote ordinario de perfiles (AD190) necesita
una tercera clave, para su propia audiencia, y AD188 no la admite. AD198 no
reescribe AD188: añade una vía que publica, en la misma operación diaria, una
clave por cada audiencia de un **conjunto cerrado y versionado**.

## Conjuntos

`conjunto_audiencias_capacidad_admin_v1` guarda cada conjunto con su versión,
sus audiencias en orden y el tramo de `clave_id` de cada una. Las filas no se
pueden cambiar ni borrar. Para añadir audiencias se crea otra versión con otra
migración.

| Versión | Audiencias (en orden) |
| --- | --- |
| 1 | `vec.admin.usuarios.listar.v1`, `vec.admin.usuarios.consultar.v1`, `vec_autorizacion.administracion_perfiles.lote_ordinario.v1` |

El programa (`bootstrap.AudienciasConjuntoCapacidadesAdmin`) tiene la misma
lista. La base vuelve a comprobar audiencia, orden y tramo de cada clave.

## Operación

1. Preparar con `vec-gobierno-usuarios-admin` y `"conjunto_capacidades": 1`
   en su configuración. El plan es la versión 2 (`gca_…`), lleva
   `conjunto_version` y una orden de puntero por clave.
2. El DBA crea un LOGIN técnico, miembro único de
   `vec_gobierno_capacidades_admin_operador` (INHERIT, sin SET ni ADMIN, sin
   `rolconfig`), e inserta en `config_gobierno_capacidades_admin_v1` el LOGIN,
   el conjunto y las tres huellas de `aprobacion-candidata.json`, con una
   ventana y `entorno = 'desarrollo'`.
3. Aplicar, repetir con otro acuse para comprobar la recuperación y verificar
   la cadena, igual que con AD188.

Las comprobaciones son las de AD188 (huellas aprobadas, preimagen, CAS de
punteros, misma raíz, renovación diaria, claves nuevas de 32 bytes), más el
conjunto: audiencia por posición, tramo de `clave_id` y órdenes numéricas y
crecientes.

La confirmación y cada intento quedan en la auditoría común con los tipos de
AD188 (`gobierno_usuarios_admin` y su intento), así que el verificador de la
cadena no cambia. Esos registros llevan la acción de AD188. Por eso cada
operación de AD198, también un intento denegado o con error, se anota además
en `operacion_gobierno_capacidades_admin_v1`, de solo adición y en la misma
transacción. Esa tabla guarda la referencia de auditoría, el LOGIN, la huella
de la solicitud, el conjunto, el resultado y las claves publicadas. Para saber
si un registro común es de AD188 o de AD198, se cruza por `auditoria_ref`. Esa fila queda fuera de la cadena con huella: la protegen la ACL del propietario y los disparadores de inmutabilidad, igual que a `clave_capacidad_version`.

Con este conjunto hay que dejar de usar AD188 para la renovación diaria. Las
dos vías publican configuración y se excluyen por cerrojo y CAS, pero solo
AD198 renueva también la clave del lote. Si alguien renueva con AD188, la
clave del lote deja de servir cuando caduca (falla en cerrado). Lo seguro es
no dejar ningún LOGIN en `vec_gobierno_usuarios_admin_operador`.

Cada publicación cambia la configuración vigente, y el consumo exige su
secuencia. Por eso, tras renovar, hay que reconfigurar y reiniciar los
procesos que la tienen fijada en su configuración privada (por ejemplo,
`vec-admin`). Es lo mismo que ya ocurría con AD188.

## Ensayo en el clon (5 de octubre de 2026)

Clon desde la copia fría H10-30 con todas las listas de main, el arranque 2+1
y Rol7, más AD190, CA35/AUT44, AUT50 y AD198:

- AD198 se instala una vez. El vector `pruebas_sql/ad198_gobierno_capacidades_admin.sql`
  da 12/12: conjunto cerrado e inmutable, ACL y preimagen, y tres intentos
  denegados que quedan auditados y anotados como de AD198, sin publicar
  claves. El registro de operaciones no se puede borrar.
- Revisión independiente con nueve negativos armados en SQL: orden cambiado,
  tramo ajeno o sin `:`, secreto repetido o de 31 bytes, dos claves, conjunto
  como texto, plan 1 y LOGIN con `rolconfig`. Todos quedan denegados, sin
  efecto y auditados. La misma base sin cambios queda permitida.
- Recorrido real con la CLI y un LOGIN técnico: preparar, aplicar
  (`gobierno_confirmado`), reiniciar PostgreSQL, repetir con otro acuse
  (mismo recibo y `replay`) y verificar la cadena (`cadena_verificada`). Se
  publican tres claves, la tercera de la audiencia del lote, y una
  configuración nueva del día.
