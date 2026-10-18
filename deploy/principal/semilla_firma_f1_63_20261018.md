# F1-63: datos de firma para una copia local

Dirección decidió el 18/10 a las 07:26 conservar la organización operativa
legada de Contratación temporal. La correspondencia CA45 une esa referencia
con la organización canónica publicada. Esta semilla prepara en una copia
ficticia la correspondencia y las asignaciones de firma de Técnico, Jefatura,
Diputación y Secretaría en `unidad:desarrollo:rrhh`. Es un **dato de copia**;
no constituye un acto de Administración ni una aprobación.

La semilla está en
`deploy/principal/candidata_hz39/firma/datos_asignaciones_firma_legada.sql`.
Forma una sola transacción serializable. Publica CA45 por su función existente
si falta; si ya está, comprueba organización, procedencia y vigencia. Exige el
registro real AUT53 de los 16 cargos iniciales, sus cuatro roles de firma V1,
controles habilitados y perfiles de las cuatro personas. Si el kit inicial ya
dejó esas asignaciones exactas, las conserva. Si falta alguna, la añade con el
canon de asignación de AUT y un identificador estable. Repetir la misma
entrada no añade filas, no mueve punteros y no vuelve a publicar CA45. Si
alguna de las cuatro personas tiene una asignación canónica vigente, se
detiene antes de escribir.

L incorpora el archivo al guion único cuando haya preparado la copia completa.
El fichero privado de parámetros psql `0600` aporta:

| Parámetro | Fuente y condición |
| --- | --- |
| `organizacion_canonica_ref` | Organización `org_…` activa de la copia, la misma de la unidad inicial de Administración. |
| `persona_tecnico`, `persona_jefatura`, `persona_diputacion`, `persona_secretaria` | Personas ficticias del kit de firmantes de esa copia. |
| `perfil_tecnico`, `perfil_jefatura`, `perfil_diputacion`, `perfil_secretaria` | Perfil de firma activo y propio de cada persona; distinto de los perfiles de consulta y Documentos. |
| `plan_aut53_sha256` | SHA-256 del plan de 16 cargos V1 **publicado** por AUT53 en esa copia, con registro, auditoría y recibo reales. |
| `vigente_desde`, `vigente_hasta`, `emitida_en` | Texto UTC de segundos enteros `YYYY-MM-DDTHH:MM:SSZ`, fijado una vez y conservado para el reintento. La emisión no es posterior al inicio. La vigencia restante queda entre 60 y 90 días y está cubierta por perfil y cargos publicados. |

No se pasan claves, certificados, alias HMAC, aprobaciones ni recibos como
datos de la semilla. Se toma de la copia el registro AUT53 y su recibo. La
fuente `kit/semilla/generar_semilla_p.py` prepara las identidades y los cargos
iniciales; el guion no sustituye sus actos nominales. Si falta la organización
o la unidad, se provisionan antes con el kit ADMIN de la propia copia.

Ejecutar con el DBA de **esa copia**, desde la raíz del repositorio, después
de dos revisiones independientes del SQL y de su prueba focal. El destino se
fija al contenedor propio: primero comprobar su marca de propiedad y salud;
después cotejar su etiqueta `vec.copia` con la raíz de la instancia. El SQL
entra por la entrada estándar y psql conecta al socket de PostgreSQL dentro
de ese mismo contenedor. No usa una conexión predeterminada del equipo ni
un servicio privado que pueda apuntar a otra base. L conservará este cotejo
al incorporar la semilla al guion común.

```bash
set -euo pipefail
deploy/principal/copia_local.sh comprobar --equipo "$VEC_COPIA_FIRMA_EQUIPO" \
  --instancia "$VEC_COPIA_FIRMA_INSTANCIA"
VEC_COPIA_FIRMA_CONTENEDOR="vec-copia-${VEC_COPIA_FIRMA_EQUIPO}-${VEC_COPIA_FIRMA_INSTANCIA}"
test "$(docker container inspect --format '{{index .Config.Labels "vec.copia"}}' "$VEC_COPIA_FIRMA_CONTENEDOR")" = \
  "/dev/shm/vec/pg/${VEC_COPIA_FIRMA_EQUIPO}-${VEC_COPIA_FIRMA_INSTANCIA}" || exit 1
cat /ruta/privada/parametros-firma-f1-63.psql \
  deploy/principal/candidata_hz39/firma/datos_asignaciones_firma_legada.sql | \
  docker exec -i --user postgres "$VEC_COPIA_FIRMA_CONTENEDOR" psql -X -v ON_ERROR_STOP=1 \
    --host=/var/run/postgresql --username=postgres --dbname=postgres
```

La salida da la huella SHA-256 de la correspondencia y las cuatro huellas de
asignación, sin referencias de persona. Guardar esa salida privada junto a
la huella del archivo SQL
`325ba75e879218433d7cb97708a80c4cfc6ab6fee0ba479d87ce91a17373d329`
y al inventario de filas anterior y posterior.
Repetir con los mismos parámetros y comprobar cero cambios. La prueba focal
`probar_datos_asignaciones_firma_legada.sql` hace dos pasadas y comprueba
historia y punteros, introduce una asignación canónica ficticia y espera su
denegación `42501`; termina en `ROLLBACK`. No se ejecuta en la principal ni
en cidonia.

Para el recorrido, arrancar la copia sin semilla y comprobar que el preflight
de firma queda cerrado con `vias_disponibles` vacío sin tumbar la aplicación.
Después de activar la semilla y el resto del material nominal, firmar de
verdad los pasos 7b y 15 de `docs/producto/FLUJO_APP.md` 4.0: dos firmas del
informe y tres de la resolución. Conservar los cinco recibos y PDF nuevos;
reiniciar PostgreSQL y la aplicación y comprobar los mismos recibos y PDF.
La consulta histórica posterior utiliza esos originales, sin firmarlos de
nuevo. Esta semilla se retira con F2-12 (permisos por datos) y F2-14 (base
consolidada).

El ensayo del 18/10 pasó en una copia local con las 78 migraciones de la
lista de la principal y los 16 cargos publicados con recibo real. La prueba
focal comprobó dos pasadas sin cambios y el rechazo `42501` de una
asignación canónica; terminó en `ROLLBACK`. La aplicación posterior añadió
una correspondencia de organización y conservó las cuatro asignaciones ya
preparadas. Repetirla dejó iguales la historia y los punteros.

El ensayo posterior corrigió la comparación de fechas para seguir la regla
del dominio: la emisión no puede ser posterior al inicio de vigencia. El
nuevo negativo exige `22023` para ese caso, con ambas fechas en el pasado.
Pasaron ese negativo, las dos pasadas y el rechazo canónico; todo terminó
en `ROLLBACK`. Dos revisiones independientes comprobaron el SQL final y el
nuevo ensayo.

Faltan las catorce publicaciones nominales de Administración: cinco cargos,
cinco enlaces y cuatro certificados; también el plan de cinco pasos. Después
hay que comprobar en Chrome las cinco firmas nuevas del expediente y
recuperar sus PDF desde la ficha tras reiniciar. Este ensayo de datos no
acredita esos actos.
