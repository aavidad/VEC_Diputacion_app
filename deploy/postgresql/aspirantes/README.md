# Aspirantes: ficha propia (fase 1, paso 5)

Diseño: [`docs/diseno/aspirantes_modulo_2026-09-29.md`](../../../docs/diseno/aspirantes_modulo_2026-09-29.md).

## Orden de instalación

Lista: `deploy/principal/lista_sql_trabajo_aspirantes_sql_20260929.txt`.

1. `roles_up.sql`, una sola vez, por el DBA. Crea `vec_aspirantes_propietario`,
   `vec_aspirantes_migrador` y `vec_aspirantes_ejecutor_externo`. El LOGIN técnico
   del portal externo se crea fuera de Git como miembro directo y exclusivo del
   ejecutor (`INHERIT TRUE`, `SET FALSE`, `ADMIN FALSE`). No hay ejecutor interno.
2. `autorizacion_atestada_v3/migraciones/000110_consumidor_aspirantes.up.sql`: tres
   perfiles y tres audiencias `vec_aspirantes.ficha.{consultar,alta,rectificar}.externa_personal.v1`.
   No depende de Usuarios: se ha ensayado antes y después de AD3-106/107/108.
3. `migraciones/000001_ficha_propia.up.sql`: esquema, políticas y cuatro fachadas.

Ningún `DOWN` se ejecuta: los ficheros `down.sql` rechazan la operación.

## Contrato

- Cada llamada va en una transacción `SERIALIZABLE READ WRITE` con el LOGIN del
  portal externo. `p_material` es el JSON literal de `canonico.SerializarMaterial`;
  detrás van las diez piezas de la exportación V3.
- La V3 se consume antes de mirar ninguna fila. Un marcador por transacción, PID y
  sesión liga las políticas de fila al índice ciego y a la ficha de esa operación, y
  se retira antes de devolver.
- `consultar_ficha_propia_v1`: `{"estado":"sin_ficha"}` o la ficha con sus sobres, y
  anota la lectura en `acceso` (finalidad `consulta_propia`).
- `alta_ficha_propia_v1(material, ficha, V3…)`: ficha, documento, índice, valores,
  historia, recibo y evento en una transacción. Un documento que ya tiene ficha da
  el recibo original (misma clave y huella) o `P1411`.
- `preparar_rectificacion_ficha_v1` y `aplicar_rectificacion_ficha_v1` van en la
  misma transacción: la primera bloquea la ficha y devuelve qué campos tienen valor;
  Go cifra con la versión nueva; la segunda escribe y coteja el motivo.
- Todo es de solo adición salvo la versión de `ficha`, que solo sube de uno en uno.
- Errores: `P1409` conflicto, `P1411` ficha existente, `P1404` sin ficha, `22023`
  petición inválida, `42501` denegación, `40001` repetir con la misma clave.

## Pruebas

`pruebas_sql/ficha_pg18.sh` levanta PostgreSQL 18.4 efímero en `/dev/shm`, sin red,
con una V3 sintética de forma, y comprueba ACL, parche del núcleo, vector Go/SQL,
alta, repetición, duplicado, rectificación con motivo, retirada, inmutabilidad,
recuento de accesos y sesiones ajenas. Con `--go` publica el puerto solo en
127.0.0.1 y recorre la aplicación Go real contra la base
(`adapters/postgres/ficha_pg18_test.go`). No acredita COSE ni KMS.
