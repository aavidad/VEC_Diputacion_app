# Provisión del portal externo en desarrollo

La herramienta prepara un plan antes de escribir. Publicar exige su huella
exacta y la preimagen que muestra ese plan. Se ejecuta desde el proceso interno;
el servidor externo nunca recibe las credenciales de gobierno ni publica
permisos al atender una petición.

Solo admite datos sintéticos y la activación de desarrollo:

```sh
export VEC_PORTAL_PROCESO=interno
export VEC_EXECUTION_PROFILE=desarrollo
export VEC_AUTH_MODE=desarrollo
export VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
```

Los archivos de fuente, alias y conexión deben estar fuera de Git, con rutas
absolutas, sin enlaces y permisos `0600`. Sus directorios deben ser privados.
Las conexiones PostgreSQL usan TLS verificado. Cada fase emplea un LOGIN
distinto, con una única membresía directa y sin atributos privilegiados.

| Fase | Único grupo del LOGIN de gobierno | INHERIT | SET | ADMIN |
|---|---|---|---|---|
| `identidad` | `vec_identidad_sesiones_v1_propietario` | TRUE | TRUE | FALSE |
| `contexto` | `vec_contexto_actor_v1_propietario` | TRUE | TRUE | FALSE |
| `autorizacion` | `vec_autorizacion_publicador_candidato_externo` | TRUE | FALSE | FALSE |
| `motivos` | `vec_autorizacion_motivos_proyector` | TRUE | TRUE | FALSE |

Esta tabla describe las cuentas de gobierno. Las cuentas
que atienden el portal usan sus grupos runtime propios, siempre con
`SET FALSE` y `ADMIN FALSE`; sus credenciales no sirven para publicar.

Ejemplo de membresía de gobierno de ContextoActor, ejecutado por el operador
del clon después de crear un LOGIN sin privilegios adicionales:

```sql
GRANT vec_contexto_actor_v1_propietario
TO vec_provision_contexto_externo_local
WITH INHERIT TRUE, SET TRUE, ADMIN FALSE;
```

El LOGIN de autorización conserva `SET FALSE`: AUT16 exige que la sesión no
asuma otro rol. Cada cuenta necesita `CONNECT` a su base y una entrada HBA
limitada al origen autorizado. Probar sus atributos por un socket administrador
no acredita una conexión TCP con TLS; compruebe el destino real del proceso.
Mantenga el HBA limitado al origen y los LOGIN que usa cada proceso.

La fuente tiene `version`, `snapshot` y `preimagen`. El snapshot usa el contrato
CTX15 de la población candidata; las preimágenes fijan las versiones y huellas
de ContextoActor, asignación, control de rol y motivos. La correspondencia
entre persona y referencia candidata permanece en el almacén externo.
Bolsa conserva las actuaciones del procedimiento con referencias opacas;
la herramienta no copia persona, perfil ni contexto a sus tablas.

Orden del ejercicio:

1. Exporte los alias con el material externo separado. La exportación es local
   y no crea cuentas ni alias en tablas compartidas.
2. Publique `identidad` con el alias de la cuenta exacta. Esta fase también
   admite el snapshot real de Usuarios, con población `usuarios` y sin vínculo
   candidato; en ese caso no prepara un perfil ni permisos de Bolsa.
3. Publique `contexto` con el snapshot candidato, previamente aprobado.
4. Publique `autorizacion`, que une rol, control y asignación en una transacción.
5. Publique `motivos` si no existen ya los tres catálogos exactos. Si existen,
   compruebe las referencias vigentes con el lector nominal y consérvelas; no
   cambie la secuencia para forzar otra publicación.

Para preparar una fase:

```sh
go run ./cmd/vec-provisionar-candidato-externo \
  --fuente /ruta/privada/fuente.json \
  --fase contexto \
  --dsn-archivo /ruta/privada/gobierno-contexto.dsn
```

Para publicarla, repita el comando con `--aprobar` y `--preimagen`, usando las
dos huellas del plan. `identidad` puede recibir `--alias-archivo` con el resultado
privado de la exportación. El resumen y el recibo solo muestran fase, estado y
huellas; no incluyen cuentas, personas, claves ni conexiones.

Un fallo al confirmar no se reintenta automáticamente. Consulte la preimagen
por el canal de gobierno y reconcilie el resultado antes de preparar otra
operación. Ninguna prueba ni configuración de este ejercicio acredita la
fuente real, su cobertura aprobada o un despliegue en la principal.

Las actuaciones de Bolsa conservan autorización, recibo y auditoría en su
historia común. El ejercicio del portal puede probar consulta, pausa,
reactivación y respuesta propia con referencias opacas. El contacto externo
nuevo requiere su fuente en el almacén externo: no se publica correo ni
teléfono en `datos_contacto_participacion` para completar este recorrido.
La confirmación queda denegada mientras falte una versión válida de esa fuente.
La pregunta 97 de `dudas.md` recoge la conservación pendiente de criterio del DPD.

La instalación del candidato termina con AUT20, AD3-121 y Bolsa65, en ese
orden. Cierran la resolución de tipos en las funciones exteriores y sus
auxiliares, conservando permisos, recibos e historia. AD3-121 mantiene las
marcas que necesita la instalación posterior de Usuarios. Ese corte termina
con AUT21 y AD3-122 antes de activar correos o imagen.

El ensayo `cierre_externo_tipos_000020_000121_000065_pg18.sh` usa un clon
PostgreSQL 18 desechable y revierte todos sus casos. Comprueba la corrección,
la conservación del canon y las filas, y el rechazo de preimágenes modificadas.
No se ejecuta sobre la principal ni instala las migraciones.

Si algunos motivos ya están publicados, puede preparar solo los catálogos
pendientes con `catalogos_motivos` en la fuente de la fase `motivos`. Para el
historial propio, use `["motivos_historial_mi_bolsa_desarrollo"]`. La selección
se limita a los tres catálogos existentes de Bolsa y queda ligada a la huella
que muestra el plan. Una lista vacía, duplicada o desconocida se rechaza.

Coteje `preimagen.secuencia_motivos` con el checkpoint actual del publicador.
La publicación consume la siguiente secuencia en la misma transacción. Para
recuperar una publicación, conserve la fuente original y sus dos huellas:
la función existente verifica el evento, la secuencia, la fecha y el contenido.
Esta fase no cambia la cuenta, el perfil ni su asignación. El servidor del
portal tampoco publica motivos al atender peticiones.
