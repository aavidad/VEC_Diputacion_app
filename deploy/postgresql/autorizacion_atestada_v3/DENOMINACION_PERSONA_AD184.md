# Consumo nominal de la denominación de Persona — AD184

AD184 conecta las operaciones PUBLICAR y LEER de CA32 con el consumidor central
V3 existente. No crea Persona, clave, permiso, sesión ni otra auditoría. No lee
tablas CA ni copia un actor del material de negocio. El actor y el perfil proceden
de la decisión/contexto V3 verificados por las autoridades existentes.

Estado: candidato preparado, sin instalación ni positivo nominal acreditados.
Requiere AUT42 real, fuente de nombres aprobada, CA32 y la composición del consumidor.
Las dos revisiones independientes y el ensayo causal siguen pendientes.

## Contratos

Sólo el propietario CA tiene EXECUTE sobre estas fachadas:

```text
registrar_y_consumir_denominacion_persona_v3_atestada(
 material text, capacidad bytea, decision bytea, motivo bytea, contexto bytea,
 persona_version numeric, perfil_version numeric,
 payload bytea, sobre bytea, evidencia bytea, raiz bytea)
→ decision_ref, efecto_ref, huella_efecto_sha256, consumo_huella_sha256,
  auditoria_ref, consumida_en, consumo_nuevo

cotejar_consumo_denominacion_persona_v3_atestada(
 mismos once argumentos, consumo_original jsonb)
→ boolean
```

El recurso tiene módulo `vec`, tipo `persona_denominacion`, referencia de Persona
`per_…` y finalidad `presentacion_persona`. Las acciones son
`vec.persona.denominacion.publicar` y `vec.persona.denominacion.leer`; la audiencia
es cada acción seguida de `.v1`. Campos exactos: `["denominacion"]` al publicar y
`["nombre_mostrar"]` al leer, con obligación `["auditar"]`. El vínculo exige la
superficie `administracion_privilegiada` y una cuenta privilegiada acreditada.

El material se conserva como lo define CA32: publicación con procedencia exacta,
versión esperada, sobre cifrado y ámbitos; lectura con Persona, versión concreta
y ámbitos. No se elige la última fuente ni se calcula SHA sobre el nombre claro.
El sobre mantiene las claves PascalCase del puerto existente y base64 canónico.

`recurso_denominacion_persona_v1(text)` devuelve material parseado, acción,
audiencia, referencia, contexto canónico y su SHA256. El contexto reproduce los
maps Go con orden lexicográfico: `ambitos` contiene organización/unidad; `atributos`
contiene `material_sha256` y, sólo al publicar, `procedencia_autoridad`,
`procedencia_sha256`, `procedencia_version` como cadena decimal.
`canon_material_denominacion_persona_v1(jsonb)` conserva el orden Go del DTO y del
sobre. Ambos helpers son puros y SECURITY DEFINER, con propietario AD y
`search_path=pg_catalog`. Así pueden usar su cierre de helpers privados sin
prestar ese cierre a AUT. AUT42 recibe EXECUTE sólo sobre las dos fachadas de
formato; no se amplían sus permisos sobre tablas, claves u otros helpers. No son permisos para consultar datos.

## Autoridad y orden causal

Antes de cualquier DDL, la migración exige el gate real
`vec_autorizacion.validar_administrador_denominacion_persona_v1(jsonb,jsonb)` de
AUT42, con propietario AUT, SECURITY DEFINER, retorno boolean y EXECUTE de AD.
La fachada lo usa antes y después del núcleo con la decisión y el material
originales. Debe acreditar exclusivamente Aplicación v5 vigente y sus asignaciones
reales, organización/unidad y las concesiones exactas. Sistemas o un perfil v4
sin estas concesiones no habilitan la faceta.

El gate se crea en AUT42 sin exigir que AD184 exista al declarar su cuerpo
PL/pgSQL. Su uso de los helpers puros se resuelve tras completar el árbol.
Así no se exige que CA32 exista para instalar AD184. CA32 valida canon, Persona,
procedencia y configuración por su autoridad propia antes de consumir AD.

El núcleo conserva sus validaciones de firma/COSE, raíz, capacidad, decisión,
contexto V2/vínculo, revocación y PDP vivo. Sólo añade dos casos nominales cerrados.
El nuevo grupo técnico `vec_persona_denominacion_ejecutor` pertenece a CA32;
se consulta con `to_regrole` para no romper consumidores anteriores mientras
la pieza siguiente aún no existe. No hay grupo genérico, sello o ledger nuevo.
El LOGIN conserva una sola pertenencia INHERIT, sin ADMIN ni SET; CA32 acredita
además sus ACL exactas. AD no concede ese grupo ni configura LOGIN durante una
petición.

## Preimágenes medidas y preservación

El núcleo requerido es POST168→172→173, con definición SHA256
`6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9` y fuente
`bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c`.
El CHECK de audiencias medido por el escritor del clon tiene SHA256
`8935639700c8923c815400626135ce23add08a610a592eaaccc9ccd5d331b8c9`, sobre
`pg_get_constraintdef(false)` en UTF8 sin salto final.

Cada guarda comunica clave, valor actual y esperado. Las tres marcas del núcleo
son únicas y la sustitución se comprueba reversible, preservando OID, firma,
propietario, SECURITY DEFINER, configuración, ACL y dependencias. Si otra rama
cambia la preimagen, se detiene para coordinar una variante medida y revisada.
No modifica SQL instalada de L/B/E, ni reescribe historia o funciones ajenas.

El CHECK conserva su predicado anterior íntegro y añade sólo las dos audiencias.
No crea claves de capacidad, raíces o configuraciones favorables. Hace falta el
origen técnico real de AD172 para audiencia/acción/canal. El eslabón nuevo sigue
siendo `consumo_confirmado_v3` de AD173, con proceso, canal, actor, perfil,
finalidad y las dos fechas originales ligadas. Los registros anteriores permanecen
intactos; AD184 no introduce una familia técnica falsa para la lectura.

## Confirmación y errores

La fachada exige consumo nominal nuevo, el acuse común completo y nueva revalidación
viva del contexto y del gate. CA une ese consumo, auditoría, CAS, historia, recibo
y outbox a su efecto dentro de la misma transacción. Go sólo devuelve datos después
del COMMIT confirmado. Recuperar una publicación exige material original y nuevo
V3; conserva el nonce y el recibo del efecto.

El cotejo sólo comprueba las piezas originales conservadas en AD, su consumo/acuse,
gobierno actual y autoridad viva. No consume otra decisión ni consulta datos CA.
Antes del callback vuelve a cotejar con el reloj actual las vigencias de la
capacidad, decisión, configuración, raíz y clave. Una espera en FOR SHARE o
en el gate no conserva una autorización cuya vigencia haya terminado.
Un acuse histórico coherente no permite una consulta nueva: ésta necesita nuevo V3.
Un COMMIT incierto no autoriza regenerar el sobre ni repetir automáticamente el efecto.

Si SQL aborta por denegación o error, efecto y consumo se revierten. El publicador
Go todavía necesita el registrador común de intentos con contexto V2 acreditado
para registrar ese resultado después del rollback, sin un permitido ficticio.
La lectura dispone del registrador del lector existente. Crear la fachada no
acredita esa composición ni habilita el circuito completo por sí solo.

## Comprobaciones preparadas

`ad184_vectores_recurso.json` fija bytes y SHA del recurso por cálculo independiente.
`ad184_recurso_acl.sql` comprueba el canon y los ámbitos desde el propietario
AUT, y el cierre de las fachadas desde el operador de ensayo, en ROLLBACK; sus sobres son transportes sintéticos, no una autoridad V3 favorable.
Los positivos con firma/PDP/origen/gate reales y las negativas de retirada,
Sistemas, ámbito, fuente, campos y recuperación quedan para el ensayo autorizado.
El productor no ejecutó PostgreSQL ni pruebas Go, ya que no cambia paquetes Go.

`ad184_cotejo_caducidad.sql` prepara el caso de demora después de los bloqueos.
Se ejecuta con un acuse CA32 COMMIT real y un gobierno de ensayo configurado
con una vigencia corta, antes de la capacidad/decisión. Comprueba un cotejo
favorable previo y su rechazo después de la demora; no fabrica un gate o
concesión. Hay que preparar un fixture nuevo para configuración, raíz y clave.
Queda pendiente de ejecutar tras completar AUT42→AD184→CA32.
