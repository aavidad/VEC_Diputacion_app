# Auditoría administrativa antes de seleccionar perfil

AD171 amplía `auditoria_consumo_v3` y su cabeza existente. Conserva la condición
de las dos familias anteriores desde el catálogo y añade dos familias con
columnas disjuntas. No modifica filas, secuencias ni huellas previas.

La migración depende de AD169. Está preparada para el ensayo causal en el clon;
su presencia en Git no acredita instalación, revisión independiente ni recorrido.

## Fuente y acceso

`vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)` es una
función privada. Solo pueden ejecutarla su propietario AD3, el propietario de
Identidad y el propietario de Autorización. Los roles de ejecución, el emisor,
el consumidor general y el registrador de intentos carecen de acceso directo.

El consumidor IS14 registra una observación propia e inmutable mediante su
resolución acreditada de identidad y política administrativa. Construye el
evento desde esa fila. Su `fuente_ref` tiene la forma
`observacion_admin_preperfil:<32 hex>` y `fuente_sha256` compromete la evidencia
privada. El proceso procede de la configuración técnica del LOGIN que atiende
la operación. El consumidor debe comprobar ese vínculo, la vigencia y el canal
antes de insertar la observación y llamar a AD171 en la misma transacción.

El escritor confía en esos propietarios internos. No acepta como acreditación
un evento construido por HTTP y no consulta tablas de Identidad. Esta separación
permite instalar AD171 antes del consumidor IS14 sin dependencia circular.

La familia `bootstrap_operador` queda preparada para un consumidor propietario
que acredite el LOGIN, su configuración, el plan y la aprobación ligada a la
preimagen mediante CAS. El escritor coteja `operador_login` con `session_user`.
Esta migración no instala ese consumidor ni acredita un bootstrap permitido.

## ABI y preimagen

Todos los valores de entrada son cadenas. No se admiten campos adicionales,
valores nulos ni familias desconocidas. El orden siguiente fija los bytes del
material; el orden de las claves del JSON recibido no influye.

| Familia | Campos en orden de preimagen |
| --- | --- |
| `preperfil_autenticado` | `tipo_registro`, `evento_ref`, `actor_ref`, `accion`, `recurso_ref`, `resultado`, `motivo_ref`, `proceso`, `canal`, `finalidad_ref`, `correlacion_ref`, `fuente_ref`, `fuente_sha256` |
| `bootstrap_operador` | `tipo_registro`, `evento_ref`, `operador_login`, `plan_sha256`, `aprobacion_ref`, `accion`, `recurso_ref`, `resultado`, `motivo_ref`, `proceso`, `canal`, `finalidad_ref`, `correlacion_ref`, `fuente_ref`, `fuente_sha256` |

Preperfil admite `listar_perfiles_propios_admin` y `seleccionar_perfil_admin`;
canal `administracion_privilegiada` y finalidad `seleccion_perfil`. Bootstrap
admite `ejecutar_plan_bootstrap_admin`; canal `operacion_tecnica_privada` y
finalidad `bootstrap_admin`. Los resultados son `permitido`, `denegado` y `error`.
El módulo almacenado es `administracion`.

`evento_ref` es `evento_<32 hex>` y `correlacion_ref` es
`correlacion_<32 hex>`. El consumidor genera el evento y recibe la correlación
confiable del adaptador. Una lectura nueva usa un evento nuevo; una recuperación
exacta del mismo evento conserva su recibo. Reutilizar la clave con material
distinto produce `23505`.

El material concatena `encuadrar_mac('vec.auditoria.admin-preperfil.v1')` y los
valores del orden correspondiente. Cada marco es la longitud UTF-8 decimal,
dos puntos, el valor UTF-8 y un salto de línea. `evento_material_sha256` es el
SHA256 de esos bytes.

El eslabón concatena marcos de `vec.auditoria.eslabon.admin-preperfil.v1`,
secuencia decimal, huella anterior, referencia `aud_v3_p_<32 hex>`, huella del
material e instante UTC `YYYY-MM-DDTHH:MM:SS.USZ`. Su SHA256 actualiza la cabeza.
Los vectores sintéticos fijan ambos hashes y las preimágenes en hexadecimal en
[`pruebas_sql/ad171_vectores_cadena.json`](pruebas_sql/ad171_vectores_cadena.json).
Son vectores independientes, cada uno con cabeza anterior sintética de ceros.

El verificador común debe reconstruir ambas familias, comprobar sus nulos
disjuntos y reconocer este dominio de eslabón. La entrega no se cierra antes de
esa extensión y de las revisiones sobre el contenido exacto.

## Transacción y minimización

La función exige `SERIALIZABLE`, escritura y zona UTC. Devuelve
`auditoria_ref`, `secuencia`, `huella_sha256`, `correlacion_ref` y `registrada_en`.
El adaptador solo confirma el recibo después del COMMIT del consumidor. Una
repetición devuelve el mismo acuse; un conflicto de serialización exige repetir
la transacción completa según su contrato.

Las familias nuevas no contienen decisión, efecto V3, perfil activo, contexto
actor, autenticación de sesión ni sesión seleccionada. Preperfil conserva una
referencia opaca de persona y una referencia y huella de la fuente. Bootstrap
conserva el LOGIN técnico, huella de plan y referencia de aprobación; el actor
humano queda nulo. La evidencia privada no se copia a la corriente común.

## Comprobaciones preparadas

La lista causal propia contiene únicamente AD171; Dirección la añadirá después
de AD169 y antes del consumidor IS14 en la lista conjunta del ensayo. La prueba
[`pruebas_sql/ad171_auditoria_admin_preperfil.sql`](pruebas_sql/ad171_auditoria_admin_preperfil.sql)
usa datos sintéticos y termina en ROLLBACK. Comprueba ACL, acuse, replay exacto,
material cambiado, perfil inventado, fuente ausente, actor nulo e inmutabilidad.
También coteja la cabeza y las huellas previas después de los rechazos.

La prueba del escritor privado no acredita la fuente IS14, selección CAS,
concurrencia entre conexiones, confirmación Go ni recuperación tras reinicio.
Esas comprobaciones pertenecen al ensayo del consumidor y del verificador común.
No ejecutar DOWN ni reaplicar migraciones sobre historia conservada.
