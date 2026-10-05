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
conjunto. La confirmación y cada intento quedan en la auditoría común con los
tipos de AD188 (`gobierno_usuarios_admin` y su intento), así que el
verificador de la cadena no cambia. El conjunto aplicado queda ligado por la
huella del plan y por `claves_sha256`.

Con este conjunto hay que dejar de usar AD188 para la renovación diaria: las
dos vías publican configuración y se excluyen por cerrojo y CAS, pero solo
AD198 renueva también la clave del lote.

## Ensayo en el clon (5 de octubre de 2026)

Clon desde la copia fría H10-30 con todas las listas de main, el arranque 2+1
y Rol7, más AD190, CA35/AUT44, AUT50 y AD198:

- AD198 se instala una vez. El vector `pruebas_sql/ad198_gobierno_capacidades_admin.sql`
  da 10/10: conjunto cerrado e inmutable, ACL, preimagen y tres intentos
  denegados que quedan auditados sin publicar claves.
- Recorrido real con la CLI y un LOGIN técnico: preparar, aplicar
  (`gobierno_confirmado`), reiniciar PostgreSQL, repetir con otro acuse
  (mismo recibo y `replay`) y verificar la cadena (`cadena_verificada`). Se
  publican tres claves, la tercera de la audiencia del lote, y una
  configuración nueva del día.
