# PostgreSQL: calculo oficial de experiencia

Almacén inmutable, de inquilino único en la fase 1 y cerrado por defecto para
confirmar y reconciliar resultados oficiales del calculador de experiencia.
El esquema del almacén no contiene funciones `SECURITY DEFINER` ni columnas
`json/jsonb` libres. Este paquete sí instala en `vec_autorizacion` una única
frontera `SECURITY DEFINER`: tiene `search_path` cerrado, firma fija, acceso
revocado a `PUBLIC` y solo coteja decisiones del registro V2 existente. Su
definición y ACL se verifican automáticamente.

## Instalacion

Requiere, en este orden, autorización `000001`, vínculo actual de autenticación
`ejecucion_documental_v4/.../000002`, roles V2, motivos `000003` y registro V2
`000004`. Después se ejecutan `roles_up.sql`,
`migraciones_autorizacion/000001_frontera_v2.up.sql` y
`migraciones/000001_resultados_oficiales.up.sql`. El script de integración usa
exactamente ese orden. Las cuentas `LOGIN` y sus secretos se gestionan fuera
del repositorio.

La aplicacion usa exclusivamente
`vec_bolsa_calculo_experiencia_aplicacion`; el lector operativo recibe solo
columnas de inventario y el publicador solo lee la outbox. Ninguno puede
actualizar, borrar o truncar historia.

## Contrato transaccional del adaptador

Cada confirmacion se ejecuta en una unica transaccion `SERIALIZABLE`, con
`SET LOCAL TIME ZONE 'UTC'`. En la primera fase, el único inquilino se fija al
instalar como `diputacion_granada` en una tabla inmutable del propietario; no
procede de la petición, del contexto de actor ni de un GUC modificable por la
sesión. RLS compara todas las filas con ese valor fijo. Esta versión no se debe
presentar como multiinquilino: un futuro conector exigirá otra frontera fiable
y una migración explícita.

El adaptador obtiene del contrato Go
`SelectorFuenteExactaCalculoReglasBaremo` su representación canónica V1 y su
huella, sin copiar ni reinterpretar el algoritmo. PostgreSQL conserva ambos,
verifica la huella y exige que el recurso autorizado de lectura sea exactamente
`fuente:<huella_selector>`. También conserva la clave semántica canónica, su
SHA-256 y un índice `HMAC-SHA-256` versionado; la clave HMAC nunca llega a la
base. La clave y el selector incluyen referencias pseudonimizadas y, por tanto,
siguen siendo datos personales protegidos aunque no contengan DNI, nombre o
correo.

Para un efecto nuevo inserta el grafo completo: resultado, intento nominal,
evidencias separadas de lectura/escritura, recibo oficial, auditoría y outbox.
La lectura aporta el consumo exacto ya emitido por la fuente. La decisión de
escritura deberá llegar desde la puerta atestada VEC-AD-2 y se consume en la
misma transacción. Las claves
foráneas diferidas impiden confirmar un grafo incompleto. En conflicto de
`(tenant_id, generacion_clave_hmac, indice_efecto_hmac_sha256)`, lee el
resultado existente:

- misma `huella_clave_semantica_sha256`: registra otro intento con
  autorizaciones nuevas, consumos y auditoria; devuelve el recibo oficial
  original y no crea resultado, recibo ni outbox;
- huella publica distinta: aborta como colision criptografica;
- cualquier dato exacto incompatible: aborta de forma cerrada.

Tras un timeout, `intento_ref` es el acuse tecnico durable y permite reconciliar
mediante la union `intento -> recibo oficial -> resultado`. No constituye ni
duplica el recibo oficial del efecto. Una ausencia tras resolver la incertidumbre
de la transaccion autoriza un reintento con un `intento_ref` nuevo; nunca se
reutiliza un identificador nominal con contenido distinto.

El adaptador encadena la auditoria leyendo la ultima fila del inquilino. La
unicidad del predecesor y `SERIALIZABLE` evitan ramas; un conflicto se reintenta
desde una lectura nueva. Todos los instantes son texto UTC canónico con seis
decimales y sufijo `Z`.

Una rectificación es otro efecto oficial con autorización interna de garantía
alta. Debe apuntar al recibo anterior por referencia y huella, mantener sujeto
y convocatoria exactos y tener tiempo posterior. El esquema impide dos
sucesores del mismo recibo, cruces de expediente y enlaces temporales hacia
atrás. Un cálculo inicial externo exige superficie personal no privilegiada y
garantías mínima y observada de nivel sustancial o alto; el perfil interno exige
garantía alta y superficie corporativa o administrativa coherente.

Los `bytea` canónicos obedecen a esquemas cerrados de la aplicación y no deben
contener identificadores directos. El resultado admite como máximo 64 MiB; la
clave, intención, selector y recibo, 32 KiB cada uno. La referencia exacta del
sujeto la emite la capa confiable y PostgreSQL no la genera ni la resuelve. El
pseudónimo recibe las mismas medidas de acceso, retención, auditoría y
protección que el resto de datos personales.

### Barreras NO-GO pendientes

La autorización de lectura no se consume de nuevo al guardar el resultado:
`FuenteReglasBaremoParaCalculo` ya devuelve su consumo y el consumo de la prueba
como referencias versionadas exactas. Este esquema conserva ambos vínculos y
revalida la decisión V2, pero todavía no existe un almacén PostgreSQL productivo
de esa fuente que permita una clave foránea o una consulta cerrada contra esos
dos recibos. Hasta implementar y probar ese adaptador, la composición
productiva debe fallar al arrancar; no se puede habilitar la confirmación
oficial usando solo los campos declarados.

La segunda barrera es la emisión de autoridad: conforme al contrato del módulo
de autorización, este paquete no concede al runtime `EXECUTE` sobre
`registrar_decision_solicitud_ligada_v2_si_vigente`. La frontera estrecha solo
coteja una decisión ya registrada; no demuestra criptográficamente quién la
emitió. Hasta disponer de la puerta atestada VEC-AD-2 y su consumo durable, no
existe camino productivo para registrar la decisión de escritura y el sistema
debe permanecer cerrado.

Además, este despliegue no habilita por sí solo producción: falta el adaptador
Go que componga todas las inserciones y reconciliaciones en una transacción
`SERIALIZABLE`, gestione colisiones y falle al arrancar mientras siga pendiente
el repositorio exacto anterior. La gestión/rotación de la clave HMAC, las copias
de seguridad cifradas y la política formal de retención pertenecen a sistemas y
deben estar implantadas antes de tratar datos reales. La fase 1 es de un solo
inquilino; no se puede simular multiinquilinato mediante GUC ni datos de la
petición.

## Verificacion y retirada

`./probar_integracion.sh` levanta PostgreSQL 18 efímero y comprueba contratos Go,
ACL con cuentas `LOGIN` reales, RLS, huellas, límites, inmutabilidad incluso
frente a `TRUNCATE`, idempotencia, repetición, rectificación, retirada y
reinstalación. El down
rechaza destruir historia salvo confirmacion operativa explicita:

`vec.confirmar_destruccion_calculo_experiencia=DESTRUIR_HISTORIA_CALCULO_EXPERIENCIA_IRREVERSIBLE`

La confirmacion no sustituye el expediente formal de retencion y destruccion.
Los disparadores de truncado solo se retiran, bajo bloqueo exclusivo, después
de superar esa confirmación. El down exige superusuario y hace el inventario
sin adoptar el rol propietario: `FORCE RLS` ocultaría a ese rol toda la historia
y convertiría el control en una falsa comprobación vacía.
