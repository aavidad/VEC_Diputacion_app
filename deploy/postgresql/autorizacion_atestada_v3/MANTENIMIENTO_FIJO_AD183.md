# Auditoría del mantenimiento del perfil fijo

AD183 añade confirmación `mantenimiento_perfil_fijo_admin` e intento
`intento_mantenimiento_perfil_fijo_admin` en la auditoría común. No crea
otra tabla de auditoría ni usa una acción de bootstrap. El actor técnico es
`session_user`, sin Persona, perfil activo o decisión V3 inventados.

## Contrato del consumidor AUT42

```text
registrar_mantenimiento_perfil_fijo_admin_v1(jsonb)
registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb)
→ auditoria_ref, secuencia, huella_sha256, correlacion_ref, registrada_en
```

Funciones del esquema `vec_autorizacion_atestada_v3`; sólo los propietarios
AD3 y AUT pueden ejecutarlas. Los runtimes, Personal, Identidad y PUBLIC
no reciben EXECUTE. Exigen SERIALIZABLE de escritura y UTC.

La confirmación exige exactamente 22 cadenas, en este orden:

```text
tipo_registro
evento_ref
operador_login
plan_sha256
preimagen_sha256
catalogo_sha256
rol_origen_ref
rol_origen_sha256
rol_destino_ref
rol_destino_sha256
asignacion_1_origen_ref
asignacion_1_origen_sha256
asignacion_1_destino_ref
asignacion_1_destino_sha256
asignacion_2_origen_ref
asignacion_2_origen_sha256
asignacion_2_destino_ref
asignacion_2_destino_sha256
proceso
canal
finalidad_ref
correlacion_ref
```

Acción implícita `mantener_version_perfil_fijo_admin_v1`, módulo `administracion`,
resultado `permitido` y motivo `mantenimiento_registrado`. Recurso
`mantenimiento_admin:<primeros 32 hex del SHA del plan>`. El proceso es
`postgresql`, canal `operacion_tecnica_privada` y finalidad
`mantenimiento_perfil_fijo_admin`. Sólo se conservan referencias opacas y
huellas reales: no nombres personales ni datos claros de las asignaciones.

AUT42 coteja el plan/aprobación/configuración, preimagen y CAS. Publica el rol,
control y catálogo y avanza las mismas dos asignaciones de Aplicación juntos,
con sus sellos existentes. Persona, perfil, ámbitos y vigencias no cambian;
CA y Sistemas permanecen intactos. AD183 no consulta sus tablas ni sustituye
ese cotejo. La confirmación va dentro de la misma subtransacción del efecto;
si algo falla se revierte también el append. AUT42 registra después el intento,
y sólo entrega el sobre después de COMMIT. Fallar el append aborta todo.

El intento exige las 12 cadenas estándar:
`tipo_registro`, `evento_ref`, `operador_login`, `solicitud_sha256`, `accion`,
`recurso_ref`, `resultado`, `motivo_ref`, `proceso`, `canal`, `finalidad_ref`,
`correlacion_ref`. Tipo y acción propios; recurso
`solicitud_mantenimiento:<32 hex>` y SHA de la solicitud real sin guardarla.
No contiene fuente, plan, rol o asignación ficticios.

| Resultado | Motivo de intento |
| --- | --- |
| permitido | mantenimiento_registrado |
| permitido | mantenimiento_replay |
| denegado | mantenimiento_denegado |
| error | mantenimiento_error |

## Representación e integridad

La confirmación usa un detalle JSONB cerrado de 14 campos: preimagen, catálogo,
roles de origen/destino y las dos asignaciones de origen/destino por referencia
y SHA. El SHA del plan usa la columna común; no se duplica el envelope ni el
actor. El intento sólo usa `mantenimiento_solicitud_sha256`; los campos de
confirmación son nulos. Las dos columnas nuevas son nulas en toda la historia
anterior. Si existe AD173, ambas familias técnicas exigen `version_consumo IS NULL`.

Se admiten únicamente las preimágenes medidas POST179:

| Variante | CHECK | SHA256 |
| --- | --- | --- |
| H9 actual | auditoria_tipo_disjunto_v2 | 0996f678e1bec083fcd4a09f07424de842e1ee5ee83f6026040924db38118cfa |
| Cadena L | auditoria_tipo_disjunto_v4 | e2ce386bbc77b80f237a4923966f98619cfb9f9823a6bea9099c9b3835ed6a93 |

Se conserva literalmente la condición anterior y su nombre. No se admiten
variantes por nombre ni se adivinan preimágenes de otras ramas.

Los dominios de material son `vec.auditoria.mantenimiento-perfil-fijo-admin.v1`
y `vec.auditoria.intento-mantenimiento-perfil-fijo-admin.v1`; los del eslabón
intercalan `.eslabon.`. Se reutilizan `encuadrar_mac`, la cabeza y el cerrojo
comunes. Referencias `aud_v3_mf_<32 hex>` y `aud_v3_mfi_<32 hex>`. Mismo evento
y material recuperan coordenadas originales; una diferencia se rechaza.

## Evidencia y límites

Cinco vectores independientes (incluido UTF-8) cotejan canon, material y eslabón
entre Python, Go y SQL. Las pruebas SQL hacen ROLLBACK y comprueban ACL,
confirmación/intentos, replay, datos cruzados/nulos, LOGIN ajeno e inmutabilidad.
El verificador añade el esquema propio
`vec.auditoria.verificacion.mantenimiento-perfil-fijo-admin.v1`, conservando
consumos L y las familias técnicas anteriores sin cambiar sus bytes. No
verifica autenticidad del checkpoint, LOGIN, COSE ni aprobación.

UP y prueba SQL terminaron con código 0 en el único clon K, con copia fría
previa SHA `6d345aabb59fb15e3be55e3e5c5c66102fdbb154d17eb3845cd618c89d559aec`.
Se conservaron 6.269 filas/cabeza/historia. PostCHECK H9 medido:
`31f8d6c5a907149b8bb044d1dd5706789ecdd65b5eeb947d15a24bf8bf298c73`.
La variante L está preparada sobre su preimagen real; no se declara ensayada
por este corte. No se reaplicó SQL instalada ni se escribió en principal.

Go focal y race, vet, gosec, vecsilencio y Semgrep local sin hallazgos nuevos;
sin baseline/VSJ ni puerta global repetida. Datos sintéticos y textos de
protocolo, sin secretos. Pendientes dos revisiones independientes exactas y
el recorrido completo AUT42; la mera auditoría no acredita el mantenimiento.
