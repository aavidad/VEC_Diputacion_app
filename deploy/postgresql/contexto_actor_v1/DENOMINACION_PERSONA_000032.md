# Denominación cifrada de Persona — CA32

CA32 conserva el nombre declarado que se usa para mostrar una Persona. No crea
Personas, cuentas, perfiles ni una acreditación de identidad civil. El nombre claro
sólo lo presta el protector existente dentro de un callback autorizado.

La migración está preparada y **no está ensayada ni habilitada**. Su guarda exige
dos fachadas nominales nuevas de AD, todavía pendientes de consenso y revisión.
AUT42 debe publicar el catálogo fijo y el gate de Aplicación; no se cambian
asignaciones v4 por el hecho de publicar otra versión. No se reutilizan permisos
de correo, sesiones, preperfil o Sistemas. La ausencia de esas dependencias cierra
la instalación o el constructor, según corresponda.

## Contrato con AD y AUT42 pendiente de ratificación

Las acciones de los puertos existentes son `vec.persona.denominacion.publicar` y
`vec.persona.denominacion.leer`. El recurso tiene módulo `vec`, tipo
`persona_denominacion`, referencia de Persona y finalidad `presentacion_persona`.
Los campos exactos propuestos son `["denominacion"]` y `["nombre_mostrar"]`.
Las audiencias son la acción seguida de `.v1`. Los dos ámbitos obligatorios son
`organizacion_ref` y `unidad_ref`, resueltos por el servidor.

AD debe aportar:

- `registrar_y_consumir_denominacion_persona_v3_atestada(text, bytea, bytea,
  bytea, bytea, numeric, numeric, bytea, bytea, bytea, bytea)`, con los siete
  campos del consumo común: `decision_ref`, `efecto_ref`,
  `huella_efecto_sha256`, `consumo_huella_sha256`, `auditoria_ref`,
  `consumida_en` y `consumo_nuevo`.
- `cotejar_consumo_denominacion_persona_v3_atestada` con esos once argumentos y
  el consumo original `jsonb`, que devuelve un booleano. Coteja material,
  atestación, consumo COMMIT original y autoridad actual; no consume otra acción.

Ambas fachadas son del propietario AD y sólo se prestan al propietario CA.
Deben comprobar firmas, raíz gobernada, revocación, contexto registrado, perfil
activo de Aplicación, campos, finalidad, ámbitos y la huella del material.
AUT42 aporta `validar_administrador_denominacion_persona_v1(jsonb,jsonb)`;
la nueva fachada AD debe usar esa autoridad nominal, con su contrato ratificado.
El consumidor de publicación debe registrar también los intentos denegados y
fallidos en la auditoría común. Un fallo SQL aborta efecto y consumo; no equivale
a un intento durable. La lectura ya dispone del registrador común del lector
existente. El futuro publicador necesita ese registrador o un envelope nominal
que confirme el rechazo sin afirmar un efecto permitido. Esa composición es
obligatoria antes de habilitar el circuito.

Estos nombres fijan la dependencia del borrador; no afirman que exista una
implementación, ni autorizan una fachada ficticia para pasar las guardas.

## Bytes ligados a la capacidad

`MaterialPublicacion` recibe la preparación original, organización, unidad y una
procedencia concreta con referencia, versión, huella y autoridad declarada. No
elige la última versión de una fuente. Esa procedencia debe existir en CA; su
huella pertenece a evidencia cifrada o a la declaración administrativa aprobada,
nunca al nombre claro. La etiqueta de autoridad no crea una autoridad jurídica.

El material, en este orden, contiene `esquema`, `persona_ref`,
`version_esperada`, `procedencia_ref`, `procedencia_version`,
`procedencia_sha256`, `procedencia_autoridad`, `sobre_sha256`, `sobre` y `ambitos`.
El esquema es `vec.persona.denominacion.publicar.v1`.
La lectura contiene `esquema`, `persona_ref`, `version` y `ambitos`, con esquema
`vec.persona.denominacion.leer.v1`. Los helpers Go devuelven los bytes canónicos
antes de pedir la capacidad al emisor central.

El contexto del recurso contiene `material_sha256`. La publicación añade los
atributos `procedencia_version`, `procedencia_sha256` y `procedencia_autoridad`.
La capacidad liga la huella canónica de ese contexto, no sólo la de una referencia.

El sobre mantiene exactamente `json.Marshal(ports.SobreDenominacionPersona)`:
`Esquema`, `PersonaRef`, `ClaveRef`, `Version`, `Nonce`, `Cifrado`, `Indice`.
El índice conserva `AmbitoRef`, `NormaRef`, `NormaSHA256`, `ClaveRef`, `Tokens`.
Nonce, ciphertext y tokens usan base64 canónico; los tokens tienen 32 bytes y
están ordenados, sin duplicados. SHA256 se calcula sobre los bytes del sobre
cifrado. El adaptador no modifica el protector ni los puertos.

## Historia, lectura y configuración

La configuración externa liga ámbito, organización y unidad a norma/huella,
referencias de las dos claves del KMS, límites y vigencia. Las filas son inmutables.
No hay valores funcionales por defecto, secretos, LOGIN ni semillas en SQL.

Publicar bloquea Persona real y su puntero actual, comprueba su vigencia, la
procedencia exacta y la configuración. Consumo V3, auditoría común, versión de la
denominación, puntero, recibo y outbox comparten transacción SERIALIZABLE de
escritura con UTC. La historia anterior se conserva. CAS sólo admite la versión
siguiente. Un replay exige el material original completo, incluido el nonce; no
se vuelve a preparar otro sobre. El recibo del efecto permanece igual y el acceso
actual necesita un consumo nominal nuevo.

Leer exige la versión actual concreta. Guarda el acuse del consumo común y lo
coteja antes de prestar el sobre al lector existente. Go entrega el resultado sólo
tras COMMIT confirmado. Un COMMIT incierto devuelve indisponibilidad y no hace un
reintento automático. La comprobación del acuse y la revalidación previa al callback
cotejan el consumo original y vuelven a comprobar Persona, versión y configuración.
La tabla de acuses no es otra auditoría: el registro nominal está en AD.

`version_actual_denominacion_persona_v1(text)` sólo se presta al propietario AUT.
Devuelve `{persona_ref, version, sobre_sha256}` o `null` para la lista/ficha que AUT
ya haya autorizado y auditado. No devuelve un nombre ni ciphertext. El primer
consumidor todavía requiere su composición real y los permisos de AUT42.

El LOGIN del pool sólo hereda `vec_persona_denominacion_ejecutor`, con
`ADMIN FALSE`, `INHERIT TRUE` y `SET FALSE`. El grupo tiene CONNECT, USAGE en CA y
las cinco fachadas de runtime; carece de acceso a tablas y a los helpers privados.
No se cambia de rol dentro de las transacciones del adaptador. Además de los
objetos de `pg_shdepend`, la guarda exige las ACL exactas: un CONNECT, un USAGE
y cinco EXECUTE sin grant option. El LOGIN no tiene ACL directas y tampoco
CREATE en el esquema ni CREATE/TEMP efectivos en la base. Los vectores nuevos
añaden esos privilegios o grant option sobre los mismos objetos y comprueban
el rechazo dentro de ROLLBACK; quedan preparados, sin ejecución PostgreSQL.

La búsqueda del puerto se mantiene cerrada. La búsqueda de nombres con filtros
compuestos, la lectura histórica explícita y la rotación operativa quedan para
un corte posterior; no se filtra una página de cien filas descifrada en Go.

## Validación de este candidato

Las pruebas Go comprueban canon estable, vínculo a procedencia/ámbito/sobre,
rechazo de material alterado y ausencia de datos tras COMMIT incierto. Los
transportes sintéticos de estas pruebas no son claves ni una autoridad V3.
Los vectores SQL preparados comprueban el canon y el cierre de ACL dentro de
ROLLBACK. Dirección ejecutará el ensayo y los positivos reales después de cerrar
AD y AUT42. Este documento no acredita instalación, persistencia real, reinicio,
lectura nominal positiva ni montaje en navegador.

Comprobaciones focales del bloque preparado: `go test -p 8`, `go test -race -p 8`
y `go vet -p 8` del paquete nuevo, Gosec sin hallazgos, Vecsilencio sin fallos y
Semgrep local con métricas desactivadas (7 reglas, 5 archivos, sin hallazgos).
`gopls` resolvió el puerto de protección existente. No se ejecutó PostgreSQL,
ninguna puerta global, servidor ni navegador durante este corte.
