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

Se admiten únicamente dos parejas POST176 de CHECK/SHA medidas:

| Cadena | CHECK | SHA256 previo |
| --- | --- | --- |
| H9 | auditoria_tipo_disjunto_v2 | 84546bd65669826e85a1dd8de95debbc7407595679f551299334493fce2dab99 |
| POST173→174→176 | auditoria_tipo_disjunto_v4 | 8346e28593ad3b35a2dc88de0023a41c8c4a89571b31ba6226d1b9bf44523e03 |

Se conserva literalmente la condición y su nombre; la variante POST173 exige
`version_consumo IS NULL` en este intento técnico. No predice AD177 de E ni
AD178 de L. Un árbol futuro con otra condición necesita su revisión causal.
La postimagen POST173→174→176→179 medida es CHECKv4,
`e2ce386bbc77b80f237a4923966f98619cfb9f9823a6bea9099c9b3835ed6a93`.

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
`intento_bootstrap_central`. Admite consumos reales v1/v2/v3 de L #557 exacta, las familias instaladas hasta
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

## Unión causal POST173

La dependencia L #557@9e09b8a331afe45090db34fdf552cf9aed9e8ef7 mantiene sus
DTO y SQL originales. El vector independiente
`union_consumos_tecnicos_bootstrap_ad173_ad179.json` enlaza las siete familias
anteriores con los cuatro intentos bootstrap: 19 asientos. Conserva los avisos
de fecha histórica y rechaza consumos nominales cruzados en registros técnicos.

La SQL variante SHA256
`bf5008ef4646a25c265e19869945cf467416df66894a1fdf90e3236c51d5d00a` y la prueba
ampliada SHA256 `49a3d20a8f0a9c5d7fe9defa6cf86077cf64fde19ef2c2e1c54d9635d66cfc67`
terminaron con código 0. Se mantuvieron las 6.240 filas, cabeza e historia
POST173. El INSERT técnico con versión nominal 3 se rechaza sin efectos.
No se aplicaron SQL en la principal ni se reejecutaron dependencias instaladas.

El ensayo usó un solo contenedor 2 GB, disco y red desactivada. La instancia K
original de 6.269 filas se guardó en fría antes del cambio, se restauró y se
cotejó contra su snapshot completo. La copia POST173 expandida se retiró tras
conservar fría y actas. Capturas privadas en el estado K
`vec-codexk-union-post173-20261004`; las dos revisiones exactas siguen pendientes.
