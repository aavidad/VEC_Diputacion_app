# Firmantes de la copia local

`firmantes.preparar(c)` prepara cuatro personas ficticias para el recorrido de
Contratación temporal. Recibe el contexto de la fábrica de la copia. Usa la
organización operativa de `c.parametros` y el recibo de la unidad H10 de esa
misma copia. No acepta certificados, claves, aprobaciones ni recibos de P o H.

La etapa emite cuatro certificados nominales nuevos con el productor público
`certificados.py --incluir-tecnico`, de 397 días. El técnico firmante usa
`certificados/rrhh_tecnico/cliente.crt` y la persona ficticia nominal de la
plantilla. El certificado original de la copia conserva la tramitación RRHH;
no recibe el perfil nominal de firma. El generador actual prepara una persona
nominal distinta; no ofrece un parámetro para reutilizar la persona operativa.
Antes de publicar identidad o asignaciones, y al recuperar la etapa, se cotejan
las huellas reales: los cuatro firmantes deben ser distintos entre sí y del
certificado de tramitación. Una coincidencia detiene la receta.
Los 397 días describen los certificados X.509;
las identidades, los cargos y las asignaciones de esta semilla terminan antes,
entre 60 y 90 días desde la preparación. Personal usa fechas de medianoche que
caben en ese intervalo. La etapa conserva copias privadas de los cinco archivos
del generador público, con sus huellas iguales al origen. Solo cambia el dato
`retirada_en` en una segunda copia de `firmantes.template.json`; registra ambas
huellas y la fecha elegida en `fuentes-manifiesto.json`.

El productor conserva su catálogo y comportamiento de tres certificados para
las recetas anteriores; la fábrica completa selecciona el catálogo nuevo de
cuatro. Esta composición requiere una copia nueva. No modifica certificados,
perfiles ni actuaciones de una copia que ya esté en uso.

El orden es identidad, publicación de dieciséis cargos V1 por AUT53,
asignaciones V1, publicación de cuatro cargos de consulta V2, sucesión de sus
cuatro asignaciones y proyección de empleados en Personal. Las sucesoras V2
usan las ocho huellas V1 publicadas de firma y consulta. Solo se publican las
cuatro de consulta; el productor H también genera cuatro propuestas de firma
que esta etapa no aplica. Las asignaciones V2 conservan persona, perfil y
ámbitos; la transacción exige la versión y huella V1 exactas y actualiza el
puntero con control de concurrencia.

AUT53 usa un LOGIN técnico durante dos horas. La aprobación declara
expresamente la excepción de dirección para la semilla ficticia. No acredita
aprobación por dos administradores, ni sustituye los catorce actos nominales o
la firma. La publicación gobernada por dos administradores queda en F1-133.
Ante un resultado incierto de AUT53 o SQL, se conservan plan y salidas para
conciliar con la base antes de reintentar. La etapa no crea otro plan para
ocultar un resultado incierto.

La salida privada está en `c.carpeta/firmantes/`. `metadata.json` y el objeto
devuelto por `preparar(c)` indican las rutas exactas:

| Clave | Uso posterior |
| --- | --- |
| `firmantes_datos` | `semilla-final/firmantes-datos.json`, fuente de las cuatro personas y perfiles para los catorce actos |
| `preparacion` | Plazos y cotejos del generador público |
| `roles_plan`, `roles_acuse` | Plan y recibo AUT53 de los dieciséis cargos V1 |
| `consulta_v2_plan`, `consulta_v2_acuse` | Plan y recibo de cuatro sucesoras de consulta |
| `certificados`, `certificados_crl` | Certificados nominales y lista de revocación preparada |
| `entrada_admin` | Entrada propia de Administración H10 para los actos posteriores |
| `autoridad_hasta`, `unidad_ref`, `organizacion_operativa_ref` | Límite temporal y ámbito de esta copia |

Las tres huellas de recibos en `metadata.json` ligan las salidas a la copia.
Los marcadores `*-aplicadas.json` guardan la huella del SQL confirmado; una
repetición idéntica recupera la etapa y una diferencia detiene el recorrido.
Los certificados y el generador mantienen sus comprobaciones de propietarios,
permisos privados y material de origen. Las pruebas aisladas de este archivo
de etapa usan `test_recorrido_copia_firmantes.py` y no levantan servicios.
