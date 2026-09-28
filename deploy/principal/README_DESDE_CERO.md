# CT y Bolsa Llamamientos desde cero: diagnóstico pendiente

Estado: **NO-GO**. Este corte conserva un plan SQL candidato y una comprobación
estática. La dirección aparcó la cadena desde cero antes de cerrar sus
preimágenes. No se ha instalado en una base de destino, no se ha arrancado el
binario contra la cadena y no hay recorrido navegador → API → PostgreSQL →
recibo ni recuperación tras reinicio para este plan.

## Archivos y cobertura acreditada

- [`lista_sql_ct_bolsa_llamamientos_desde_cero_20260928.txt`](lista_sql_ct_bolsa_llamamientos_desde_cero_20260928.txt) contiene 210 rutas propias rastreadas de `contratacion_temporal` y `bolsa_llamamientos` (`roles*_up.sql` y `*.up.sql`) y 101 rutas de dependencia, sin duplicados. Entre las dependencias figura el DBA RLS B1 ya revisado. La lista incluye CT138 → AD3-104 → CT139 → AD3-105 → CT140.
- [`validar_lista_sql_ct_bolsa_desde_cero_20260928.py`](validar_lista_sql_ct_bolsa_desde_cero_20260928.py) comprueba rutas rastreadas, cobertura propia exacta, ausencia de duplicados y aristas seleccionadas. Su resultado no acredita que las preimágenes SQL acepten el orden restante.
- `contexto_actor_v1/migraciones/000004a` sustituye `000004` desde cero: nunca aplicar ambas. La ACL C3 ya rastreada cierra 14 tipos fila de Bolsa B1 y uno de Autorización4 antes del selector. El paquete DBA B1 `20260928_b1_rls_propietario/01_cerrar_politicas.sql` cierra sus 14 políticas `solo_propietario` antes del selector. Ninguno de esos paquetes cierra los tipos fila posteriores de CT y AD3.

## Bloqueos causales

1. En PostgreSQL 18.4 efímero, el selector corporativo rechazó el plan tras CT1–16, AD3-1/2 y B1. Había 42 tipos fila con `USAGE` de `PUBLIC`: 28 de CT (creados en CT1/2/3/4/6/7/8/14) y 14 de AD3-1. Todos tenían `typacl = NULL`; el propietario era el de su esquema. La ACL C3 no los incluye. Un delta ACL separado para esos 42 tipos está pendiente de revisión e integración; no forma parte de este candidato.
2. Con el DBA RLS B1 rastreado y una revocación diagnóstica de los 42 tipos **solo en el contenedor efímero**, el selector aceptó la preimagen. El siguiente archivo, Contexto `000004a_vinculo_corporativo_rrhh_v1.up.sql`, rechazó el manifiesto simbólico observado `a52f2568dede7df72db15fd45b9a81fbed54af8d6c0c70971261e6e55c27e9ff`. Su lista cerrada admite `bddc5574…`, `613d1837…` o `6fa63aa7…`; la huella observada no está acreditada por esas preimágenes. No se conservó el manifiesto completo pre/post Autorización5 para atribuir cada diferencia. Autorización5 concede `USAGE` y `EXECUTE` sobre Contexto; CT `migraciones_autorizacion/000001` exige la cadena de Autorización5–7. Una secuencia alternativa temprana del selector fue planteada para revisión, sin decisión ni ensayo final acreditados. No se ha cambiado Contexto4a ni su guarda.
3. El tramo de incorporación todavía tiene dependencias Personal mal intercaladas en la lista candidata. Personal2 exige AD3-26; Personal3 exige AD3-28; Personal5 exige AD3-29; CT76/79/84 necesitan el lector Personal5. Falta además `roles_localizador_solicitud_alta_up.sql` antes de Personal6. El orden propuesto en análisis, **no probado ni incorporado**, es Personal roles → P1 → AD3-26 → P2 → AD3-27/28 → P3/P4 → AD3-29 → P5 → rol localizador → P6 → CT70–85. Debe comprobarse contra las preimágenes exactas antes de actualizar la lista.

## Evidencia y siguiente condición

La comprobación estática pasó con 210 propios y 101 dependencias. El ensayo PG18.4 se hizo en un contenedor sin puertos publicados y con `PGDATA` en tmpfs; se detuvo en las guardas descritas. Se retiró el contenedor al terminar. La revocación diagnóstica no es una migración ni una instalación acreditada. No se ejecutó `DOWN`, no se alteró historia conservada y no se usaron datos personales reales.

El runner y runtime de Calidad son un trabajo separado (`trabajo/codexg-calidad-20260928`@`4330ad75e` al redactar este diagnóstico). Aceptan un `--plan` externo y el DBA RLS B1 exacto, pero **no se han integrado ni ejecutado con esta lista final**. Falta resolver por escrito la preimagen Contexto4a y el lugar seguro del delta ACL42, intercalar Personal, revalidar la cobertura, obtener dos revisiones independientes del contenido final sensible y ensayar la cadena completa en PostgreSQL 18.4. Solo después corresponde al runner de Calidad probar arranque del binario y consumidor web real. Este README no autoriza publicación de producto, instalación ni uso de un servicio compartido.
