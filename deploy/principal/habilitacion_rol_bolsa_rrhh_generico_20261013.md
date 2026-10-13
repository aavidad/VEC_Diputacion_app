# Publicar permisos del rol RRHH de Bolsa

El gobierno B103 permite añadir al rol RRHH un conjunto de entradas admitidas
del catálogo. Cada concesión conserva exactamente la acción, tipo de recurso,
finalidades, garantía, campos y obligaciones de su entrada. No hay una lista
fija de acciones nuevas. Se conserva el rol, su nombre, las concesiones previas
y los identificadores de las asignaciones aprobadas.

CONFIG NUEVA: ninguna. Reutiliza `VEC_ADMIN_VERSION_BOLSA_B103_CONFIG_FILE`,
los pools segregados, las dos audiencias ADMIN y el material del
[circuito B103 existente](habilitacion_b103_respuestas_20261012.md).

## Orden para dirección

1. Instalar una vez AUT92 mediante
   `lista_sql_codexn_rol_rrhh_generico_20261013.txt`. Sus dos preimágenes proceden
   de la copia de cidonia del 13/10, posterior a HZ36. No reaplicar AUT75 ni
   AUT84 y no ejecutar DOWN. Actualizar `vec-admin` con esta entrega.
2. Leer el catálogo y censo completos, la cabeza vigente del rol RRHH, su
   control y las asignaciones nominales que se van a conservar. En la copia
   13/10b, el rol está en v10 y el catálogo v5 aún contiene su v9: el paquete
   nuevo debe incluir el perfil v10 y su control actuales. No se omite este
   cotejo ni se sustituye la cabeza con datos de un catálogo anterior.
3. Preparar y admitir una nueva versión completa del catálogo con las fuentes
   N, K y J aprobadas, conservando todas las entradas y perfiles anteriores
   que deban seguir publicados. Reutilizar AUT58/AUT61 y
   `vec-catalogo-acciones-admitir`; conservar su recibo y huellas. Los JSON de
   fuente por sí solos no conceden ningún permiso.
4. La primera persona ADMIN propone en
   `/api/admin/perfiles/v1/gobierno-version-bolsa-rrhh/propuestas`.
   El cuerpo conserva el formato B103: `operacion_ref`, referencia, versión y
   huella del catálogo, `base_ref`, `base_huella_sha256`, revisión y huella del
   control, `selecciones`, `asignaciones`, `motivo` y `referencia_acto`.
   Cada selección lleva `entrada_ref`, `entrada_version` y
   `entrada_huella_sha256` de su entrada admitida. El servidor recupera y copia
   la concesión; el cuerpo no acepta identidad ni una concesión elegida por
   el cliente. El orden de las selecciones forma parte del material aprobado.
5. Otra persona ADMIN aprueba la referencia y huella de esa propuesta en
   `/api/admin/perfiles/v1/gobierno-version-bolsa-rrhh/cierres`, con el formato
   de cierre B103 existente. Se conservan las dos decisiones V3, CAS, recibo,
   auditoría y escritura transaccional. Las rutas B103 anteriores siguen
   disponibles. Si cambia alguna preimagen antes del primer efecto, preparar
   otra propuesta; si el resultado es incierto, recuperar la operación original
   con las mismas referencias y bytes.

El límite existente es 512 concesiones en la definición final y 16
asignaciones por plan. Las entradas nuevas pertenecen a Bolsa, son ordinarias
y tienen exactamente las dimensiones `unidad_ref` y `ambito_ref`. Las fuentes
del candidato externo conservan su circuito separado.

## Fuentes y resultado en la web

| Fuente | Acción | Pantalla que la consume |
| --- | --- | --- |
| `bolsa_configuracion_rrhh_v1.json`, integrada en #1094 | `bolsa.configuracion.consultar` | Bolsas → Configuración → Mi perfil en Bolsa |
| `bolsa_politica_cese_rrhh_v1.json`, integrada en #1091 | `bolsa.politica_cese.consultar` | Bolsas → Configuración → política tras el cese |
| `bolsa_origen_ct_rrhh_v1.json`, PR #1097 | `bolsa.origen_ct.consultar` | Selección desde Bolsa en el expediente CT, consumidor #1088 |

Las fuentes están en `deploy/principal/fuentes/`. Para K se usa la cabeza
aprobada de #1097, que incluye las hojas de su reserva; no una copia anterior.
No seleccionar una acción ya concedida. La nueva versión conserva los permisos
existentes; el especializador no crea roles, perfiles ni asignaciones de oficio.

Tras publicar, dirección abre las tres consultas con la identidad RRHH vigente
y comprueba su asiento nominal. Para N, la petición es
`GET /api/vec/bolsa/bolsas?perfil=propio`. Si responde 403, el diagnóstico
`bolsa_configuracion_denegada` identifica la guarda con valores esperado y actual
sin datos personales. El GET sin parámetros es otro recorrido y conserva su
permiso propio. La instalación SQL y el ensayo local no sustituyen esta
comprobación en la pantalla.
