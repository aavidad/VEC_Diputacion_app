# Intentos del arranque central ADMIN

AD179 registra `intento_bootstrap_central_admin` en la cadena común
`auditoria_consumo_v3`. Conserva la confirmación `bootstrap_operador` de AD171,
las familias AD174/176, sus funciones y sus recibos. No crea otra tabla de
auditoría ni concede roles o permisos de negocio.

Estado: candidata preparada para el ensayo del escritor del clon y las dos
revisiones independientes. El productor no ha ejecutado SQL ni instalado nada
en la principal. AUT40 es el consumidor dependiente; esta pieza aislada no
acredita un arranque completo ni auditoría de conexiones/cancelaciones.

## ABI y autoridad

```text
vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(jsonb)
→ auditoria_ref, secuencia, huella_sha256, correlacion_ref, registrada_en
```

Solo tienen EXECUTE el propietario AD3 y `vec_autorizacion_propietario`.
PUBLIC, Personal, Identidad y los ejecutores runtime no reciben acceso.
El actor es `operador_login = session_user`, un LOGIN técnico real. No se
inventan Persona, perfil, sesión o decisión V3 antes de tenerlos acreditados.

La entrada contiene exactamente estas 12 cadenas obligatorias, en este orden:

```text
tipo_registro
evento_ref
operador_login
solicitud_sha256
accion
recurso_ref
resultado
motivo_ref
proceso
canal
finalidad_ref
correlacion_ref
```

Tipo `intento_bootstrap_central_admin`, acción
`registrar_bootstrap_central_admin_v3`, módulo `administracion`, finalidad
`bootstrap_admin`, canal `operacion_tecnica_privada` y proceso `postgresql`.
El proceso identifica al motor observador, sin atribuirse a la CLI de origen.
`recurso_ref` es `solicitud_bootstrap:<32 hex>` de la invocación real;
`solicitud_sha256` compromete la entrada que AUT40 observa, sin conservarla.
El intento no contiene plan, aprobación, fuente, recibo de efecto o preimagen
ficticios. La nueva columna `bootstrap_solicitud_sha256` es nula en toda la
historia anterior y pertenece exclusivamente a esta familia.

El catálogo técnico de datos SQL y el catálogo JSON del verificador admiten:

| Resultado | Motivo |
| --- | --- |
| permitido | bootstrap_registrado |
| permitido | bootstrap_replay |
| denegado | bootstrap_denegado |
| error | bootstrap_error |

AUT40 ejecuta el efecto dentro de una subtransacción. En el éxito confirma
junto a él la confirmación histórica de AD171 y el intento nuevo; en un rechazo
gestionado revierte el efecto y confirma únicamente el intento. Si falla el
append, se aborta todo. El consumidor entrega el sobre únicamente tras COMMIT.
Cada invocación usa un evento distinto, incluido el replay del plan. El replay
del append con evento/material idénticos conserva sus coordenadas originales.

## Preimagen e integridad

Se exige el CHECK real `auditoria_tipo_disjunto_v2` posterior a AD176,
observado en el clon PostgreSQL 18.4. SHA256 de
`pg_get_constraintdef(oid, false)` en UTF-8 sin salto final:
`84546bd65669826e85a1dd8de95debbc7407595679f551299334493fce2dab99`.
La migración envuelve literalmente esa condición y mantiene sus nulos; no
predice AD177 de E ni AD178 de L. Un árbol futuro con otra condición exige
una revisión causal propia, no se acepta por el nombre de la familia.

Se reutilizan `encuadrar_mac`, SHA256, la cabeza y el cerrojo comunes de evento.
El material usa `vec.auditoria.intento-bootstrap-central-admin.v1` seguido de
los 12 campos. El eslabón usa
`vec.auditoria.eslabon.intento-bootstrap-central-admin.v1`, secuencia, huella
anterior, referencia, huella del material y fecha UTC con microsegundos.
La referencia es `aud_v3_bi_<32 hex>`; la confirmación AD171 sigue con
`aud_v3_p_<32 hex>`. Una diferencia de material o reutilizar un evento de otra
familia se rechaza sin modificar la cabeza ni la historia.

## Verificador y pruebas focales

La CLI añade `vec.auditoria.verificacion.bootstrap-central-admin.v1` y el objeto
`intento_bootstrap_central`. Admite únicamente las familias instaladas hasta
AD176 y este intento; los esquemas anteriores rechazan AD179. El informe añade
`material_intentos_bootstrap_recalculado`, sin autenticar LOGIN, fuente o
checkpoint. Conserva los cálculos y bytes de los registros anteriores.

Los cuatro vectores independientes, incluido un LOGIN UTF-8, están en
`pruebas_sql/ad179_vectores_cadena.json` y
`cmd/vec-auditoria-verificar/testdata/bootstrap_intentos_ad179.json`.
La prueba `pruebas_sql/ad179_intentos_bootstrap.sql` coteja los mismos marcos
con PostgreSQL, los cuatro resultados, replay, ACL, campos cruzados, LOGIN
ajeno, fecha y formato, resultado/motivo incompatibles, inmutabilidad e historia
completa. Cuando hay confirmación bootstrap AD171, comprueba también que no se
puede reutilizar su evento como intento. Todo termina en ROLLBACK en el clon.

La prueba Go conserva las familias previas y la confirmación AD171, comprueba
rechazos de cruces y códigos de fecha cerrados, sin exponer el valor recibido.
Las reglas Semgrep locales están en `pruebas_sql/ad179_semgrep_local.yml`.
No se añaden justificaciones VSJ ni cambios de baseline para silenciar errores.

La lista causal sirve para un frío sin AD174/176. Si esas dependencias ya
están instaladas, el escritor del clon aplica únicamente AD179. No se repiten
UP ni DOWN de migraciones instaladas. La revisión focal del productor comprende
ACL, campos disjuntos, límites, idempotencia y proyección mínima; no sustituye
las dos revisiones independientes ni el ensayo de AUT40.

Comprobaciones locales ejecutadas, todas con código 0: pruebas normales y
`-race` de `internal/vec/auditoria` y `cmd/vec-auditoria-verificar` con `-p 8`,
`go vet` de esos paquetes, gosec solo sobre ambos, vecsilencio focal sin fallos
nuevos y Semgrep local `--metrics=off` (cinco archivos, tres reglas, sin
hallazgos). Caché Go en disco y `git diff --check` limpio. No se repitió la
puerta global ni se modificaron las SQL anteriores o la baseline.
