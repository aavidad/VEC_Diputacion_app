# Activación de la firma del recorrido

Aplicar primero el kit de Administración de copia de
[`admin_copia/LEEME.md`](../admin_copia/LEEME.md). Cada copia usa sus propios
recibos, organización y unidad. Los archivos de identidad, certificados,
conexiones y materiales de publicación se mantienen fuera de Git.

## Preparar los datos de la copia

El kit privado compartido está en `$HOME/.local/state/vec-ensayo-candidata/firma-hz39/`.
Su manifiesto fija los archivos de preparación; los recibos se producen en cada
copia. Completar `kit/semilla/plantilla-config-p.json` en una carpeta privada.
`acuse_unidad_h10` debe apuntar al recibo real de la unidad: organización,
unidad y nodo deben coincidir con él. Los perfiles de Administración tienen
ese alcance; crear otro nodo no lo amplía.

Seguir `kit/semilla/LEEME.md` en este orden:

1. Preparar las cuatro personas, cuentas y perfiles con la clave HMAC propia.
2. Preparar los 16 roles de 90 días con `vec-publicar-cargos-firma`, aprobar el
   plan y aplicar la provisión privada de su LOGIN y aprobación, con ventana
   de dos horas. Exigir el recibo real antes de emitir asignaciones.
3. Preparar los tres certificados del catálogo y renovar el del técnico con
   la misma CA. Los cuatro deben cubrir al menos 60 días.
4. Generar la semilla final con `--roles-fuente-publicada`, `--roles-plan` y
   `--roles-recibo` de esta copia. Aplicar sus datos por la conexión local de
   la copia. No trasladar alias HMAC ni recibos de otro equipo.

`kit/generar_materiales.py` prepara cinco cargos, cinco enlaces y cuatro
certificados nominales. `kit/ejecutar_acto.py` comprueba cada material y envía
un acto por vez; conserva el recibo y detiene el envío ante un resultado
incierto. `kit/generar_plan_v1.py` prepara el catálogo de cinco pasos a partir
de esos cargos y del circuito publicado; `kit/publicar_plan_hz39_v1.sh` crea
con el primer administrador y publica con el segundo. La preparación local
no constituye una decisión de autorización.

Después de los recibos, `kit/generar_config_firma.py` produce `firma-vec.json`
con el paquete y recibo reales del plan, los cuatro recibos de certificados
nominales y las asignaciones, roles y controles exportados de esta copia.
Conservar la configuración y los materiales con permisos privados.

## Empleados de los firmantes

Antes de activar el panel nominal, aplicar `datos_empleados_firmantes.sql`
en la copia con un archivo privado de parámetros de psql. Necesita
`persona_tecnico`, `persona_jefatura`, `persona_diputacion`,
`persona_secretaria` y sus cuatro `perfil_*`; además, `organismo_ref`,
`unidad_ref`, `regimen_ref`, `modalidad_ref`, `vigente_desde` y `vigente_hasta`.
Las personas y perfiles deben estar provistos y la unidad debe tener una
única cabeza vigente. Usar fechas a medianoche UTC y un intervalo con al menos
60 días restantes, cubierto por la unidad, el perfil y su asignación. Las fechas
son parámetros de la copia; una asignación de 90 días no necesita una relación
de 400 días.

El guion añade los cuatro empleados y sus relaciones como datos ficticios,
con origen `datos:recorrido:hz39`, según la orden de dirección. No produce
decisiones ni recibos de negocio y no acredita un catálogo de régimen o
modalidad del organismo. Rechaza datos anteriores distintos; una repetición
exacta conserva la historia y la generación. La autoridad del firmante pide
la proyección de empleado de Personal; la consulta CT general conserva su
alcance anterior. La guarda de empleado sigue siendo obligatoria.

```sh
psql -X -v ON_ERROR_STOP=1 -f /ruta/privada/parametros-firmantes.psql \
  -f deploy/principal/candidata_hz39/firma/datos_empleados_firmantes.sql
```

## Vigencias

El guion de certificados prepara los tres firmantes del catálogo con
`--cert-dias 400 --crl-dias 90`. Requiere la CA, su índice y su lista de
revocación original, y conserva los certificados revocados. Usar una carpeta
nueva; un material anterior incompleto o con plazos distintos se rechaza.
El certificado del técnico también debe cubrir al menos 60 días; en la copia
H se emitió con 400 días y se publicó su huella por el acto nominal.

Copiar `scripts/operacion/firma_ct_tres_pasos/cargos.v2.json` a una carpeta
privada y revisar sus fechas y organización. Sus cargos terminan el 31 de
diciembre de 2027 y la duración propuesta para asignaciones es de 90 días.
Preparar y aplicar el plan con `vec-publicar-cargos-firma`, su aprobación y
su LOGIN segregado. La ventana entre preparación y aplicación sigue siendo
de dos horas en este guion, con máximo de un día. Un COMMIT con respuesta
`denegado` no publica cargos: exigir `estado: permitido` y recibo.

Una versión ya publicada conserva sus bytes. Si la copia contiene una versión
anterior de los cargos, recuperar y revisar sus recibos antes de preparar otra
publicación. No reutilizar la versión 1 con fechas o duración distintas.

## Personas, cargos y certificados nominales

Las cuatro personas del recorrido son el técnico, Jefatura, Diputación
delegada y Secretaría General. Los perfiles físicos de firma, consulta de CT,
Documentos y consulta del circuito son distintos. Sus asignaciones y ámbitos
se cargan como datos ficticios de la copia, según la decisión de dirección;
las huellas se calculan con el canon de la autoridad existente.

Publicar los cargos y sus enlaces por la entrada nominal de Administración:

```text
POST /api/admin/perfiles/v1/personal/cargos-competenciales/publicacion
POST /api/admin/perfiles/v1/certificados-nominales/publicacion
```

Ambas reciben `material_base64` de su contrato existente. Los actos usan el
certificado del administrador, su perfil seleccionado y decisión de
autorización real. Conservar los recibos y la auditoría. La composición del
pool de certificados fija y acredita `TimeZone=UTC`, igual que cargos; el
valor predeterminado `Etc/UTC` del clon no cumple el consumo de autorización.
No añadir ajustes de rol ni parámetros libres a la conexión.

El plan y las asignaciones usan la organización operativa de Contratación
temporal. Los cargos y CA25 usan la canónica. La correspondencia CA45
publicada las relaciona; AUT93 y CT214 la comprueban durante la firma.

Antes de generar los materiales, cotejar `unidad_ref` con la asignación
histórica del expediente que se va a firmar. En el recorrido es
`unidad:desarrollo:rrhh`. El nodo de Personal, los cargos, los enlaces, las
asignaciones de firma y las cinco entradas del plan deben usar esa unidad.
La unidad creada al dar de alta Administración puede ser otra: su recibo
no sustituye a la unidad gestora del expediente. Preparar los datos ficticios
de RRHH en la copia y los ámbitos de Administración antes de publicar.

En H se publicaron cinco cargos competenciales, uno por paso del plan, y
cinco enlaces de titularidad. Jefatura ocupa los de informe y resolución;
cada documento tiene una sola firma de esa persona. Así se conserva la
guarda de una titularidad vigente por cargo. Son declaraciones ficticias
publicadas mediante actos, con sus recibos reales.

## Catálogo nuevo de cinco pasos

Crear un catálogo nuevo, `ct.plan.firma.hz39.operativa`, versión 1. Conservar las
referencias antiguas. Las cinco entradas, en orden, son:

| Documento | Paso | Persona |
| --- | --- | --- |
| Informe definitivo | 1 | Técnico |
| Informe definitivo | 2 | Jefatura |
| Resolución | 1 | Jefatura |
| Resolución | 2 | Diputación delegada |
| Resolución | 3 | Secretaría General |

Cada entrada conserva `circuito_ref` completo
`vec.contratacion_temporal.circuito_firma:2`, su versión 2 y su huella real,
además del cargo, rol, organización, unidad y perfil lógico del paso.
Preparar los materiales canónicos con el modelo común de catálogos y
`vec-plan-firma-validar preparar`. El validador acredita solo la preparación.
`publicar_plan_v2.sh` exige versión 2 y no sirve para este catálogo nuevo.

Enviar creación y publicación a la entrada existente:

```text
POST /api/admin/perfiles/v1/plan-firma/gobierno
```

El primer administrador crea el borrador y el segundo, distinto, lo publica.
Conservar los dos materiales, claves de operación y recibos. Después de un
resultado incierto, recuperar con esos mismos bytes y claves antes de crear
otro material. Las cinco entradas duran al menos 60 días.

Si ya hay una versión 1 publicada para otra unidad, conservarla y preparar
la versión 2 desde su catálogo canónico con `CatalogoConfigurable.NuevaVersion`.
Publicar cargos y enlaces nuevos para RRHH; después, crear y publicar el plan
por los dos administradores. `publicar_plan_v2.sh` coteja los materiales de
versión 2. Actualizar el archivo, la versión y la huella del plan en la
configuración solo tras recibir la publicación confirmada. No cambiar los
bytes de la versión anterior ni el expediente para salvar una diferencia
de ámbito.

## Configuración del panel de firma

Guardar el paquete publicado `{version_esquema,fuente,catalogo}` en una ruta
privada. En `identidad/firma-vec.json`, usar `plan.archivo`, referencia
`ct.plan.firma.hz39.operativa`, versión 1 y huella del catálogo canónico publicado.
Esta huella es `catalogo_sha256` del material validado, no la del paquete
ni la del recibo. Los cuatro
firmantes y sus admisiones llevan las huellas reales de certificado, rol,
control y política de autenticación. La retirada de la configuración y de
las admisiones debe cubrir al menos 60 días.

Configurar también `VEC_CT_CIRCUITO_RRHH_SOURCE_PATH` con una copia privada
de `config/contratacion_temporal_circuito_rrhh_v2.json`. Es distinto del
catálogo de firma de `VEC_CT_CIRCUITO_FIRMA_SOURCE_PATH`; el montaje nominal
necesita la consulta del circuito de RRHH.

Mantener los selectores de firma, originales, registro, Documentos y
verificación existentes descritos en
[`scripts/operacion/firma_ct_tres_pasos/LEEME.md`](../../../../scripts/operacion/firma_ct_tres_pasos/LEEME.md).
Reiniciar vigilado. El preflight debe ofrecer `certificado_vec` a la persona
del paso; una respuesta 200 con la lista de vías vacía no permite firmar.

## Plantillas y sellado de lectura

Instalar CT218 una sola vez, después de las dependencias ya instaladas de la
candidata. Su lista es `../lista_sql_firma_vigencias.txt`. Conserva la
validación de la provisión inicial y coteja el archivo configurado con la
última publicación, su procedencia y su evento. No cambia permisos.

Publicar la plantilla 4 mediante el guion existente `publicar_informe.sh`.
Conservar la historia anterior y el recibo de la publicación. Configurar
`VEC_CT_PLANTILLAS_SOURCE_PATH` con una copia privada de
`data/catalogos/contratacion-temporal/plantillas_documentos.v4.json` y
reiniciar vigilado. Un archivo diferente de la publicación se rechaza.
Antes de CT218, esa ruta se cotejaba con la primera publicación y no admitía
la plantilla 4; no desactivar la comprobación para resolverlo.

La aplicación debe tener `VEC_AUDITORIA_SELLADO_DATABASE_URL` con el LOGIN
exclusivo de `vec_auditoria_encadenador`, conexión TCP autenticada y TLS
verificado. El sellador nativo mantiene la auditoría común y los accesos de
CT. Un sellador que atienda solo la primera cadena no permite leer la ficha.
No renovar latidos por SQL ni aumentar sus plazos para ocultar un fallo.

Instalar también IS20 una sola vez, con su preimagen instalada. El cotejo de
auditoría histórica reconoce la política de certificado personal de
desarrollo en la fecha de autenticación: sesión interna, certificado,
garantía sustancial y cuenta ordinaria. Administración conserva garantía
alta. El firmante nominal declara sustancial; cambiarlo a alta no sustituye
una autenticación acreditada ni es una opción de esta composición.

El botón de PDF firmado usa `firmas-documento/descargas-v2` para la sesión
nominal. El servidor consulta la historia autorizada en esa misma petición
y descarga por Documentos su referencia, versión y huella. El navegador
coteja también la huella de los bytes. La API general de Documentos conserva
su frontera de garantía alta.

## Rol y cargo del paso publicado

Instalar CT219 una sola vez. El material mantiene dos identificadores:
`RolIDFirmante` es el rol de la asignación y `CargoFirmante` la referencia
`car_…` de Personal. El descriptor debe acreditar cada valor exactamente
contra el paso del plan publicado. Compararlos entre sí rechazaba todos los
pasos preparados por Administración. La versión 3 del registro conserva la
autorización, el consumo en la misma transacción y su historia; las funciones
históricas de versión 2 y la entrada de versión 4 permanecen intactas.

## Verificador de cada copia

El kit `kit/grxfirma_p/` prepara GrxFirma con un puerto propio de loopback,
anclas, CRL, TLS y token privados de la misma copia. Su modo `--comprobar`
valida el material antes de `--arrancar`. Configurar el cliente con:

```sh
export VEC_FIRMA_VERIFICACION_ENABLED=true
export VEC_FIRMA_VERIFICACION_URL=https://127.0.0.1:63120
export VEC_FIRMA_VERIFICACION_CA_FILE=/ruta/privada/ca.crt
export VEC_FIRMA_VERIFICACION_TOKEN_FILE=/ruta/privada/token
export VEC_FIRMA_VERIFICACION_TIMEOUT=20s
```

La URL es el origen HTTPS, sin `/v2/verify`. Cada copia usa su propio servicio.
El verificador debe responder 200 a su comprobación autenticada; una escucha
abierta no acredita que pueda verificar firmas.

## Estado comprobado

En H: publicación AUT53 de los 16 roles con recibo; cinco cargos y sus cinco
enlaces por Administración; cuatro certificados nominales publicados con
recibos; creación y publicación del catálogo nuevo por dos personas distintas.
El primer intento de certificado devolvía 403 por `Etc/UTC`; con el pool
corregido, el mismo material devuelve 200 y su recibo.

Pendiente: arranque del panel con esa configuración, las dos firmas del
informe antes de Intervención, las tres de la resolución, descarga de los
cuatro documentos con firma pendiente y recuperación de PDF y recibos tras
reiniciar. Solo al superar esos pasos se entrega la firma a P para su
recorrido completo.
