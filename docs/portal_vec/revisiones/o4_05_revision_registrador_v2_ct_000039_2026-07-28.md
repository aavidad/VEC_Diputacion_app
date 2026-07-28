# O4-05: revisión final del registrador RRHH v2

Fecha: 28 de julio de 2026

Ámbito: C2-D2-C, migración CT `000039`, registro durable y minimizado de
accesos RRHH y soporte de lectura *as-of*

Estado: `GO` técnico con dos revisiones independientes; no autoriza producción

## Alcance cerrado

La migración `000039` aporta:

- control estable y versionado del registrador v2;
- vínculo uno a uno entre acceso y referencias opacas de Identidad;
- registro encadenado y prueba canónica sin LOGIN nominativo;
- índice de organización, expediente y corte para consultas *as-of*;
- revalidación de sesión, cuenta ordinaria, perfil, ámbito y revocación dentro
  de la misma transacción;
- reversión `RESTRICT` con catálogo semántico y barreras futuras.

Solo se persisten `cuenta_ref` y `cuenta_ordinaria_ref` opacas, iguales y no
privilegiadas. El LOGIN técnico, filtros, material criptográfico, token y datos
personales no forman parte de tablas, cánones ni recibos.

## Evidencia reproducida

El runner
[`probar_o4_05_registrador_acceso_rrhh_v2_pg18_4.sh`](../../../deploy/postgresql/contratacion_temporal/probar_o4_05_registrador_acceso_rrhh_v2_pg18_4.sh)
usa PostgreSQL 18.4 fijado por resumen, contenedor efímero, red deshabilitada,
sistema de ficheros de solo lectura y volúmenes temporales.

Terminó correctamente en cuatro ejecuciones sobre los mismos hashes:

- productor de pruebas: 18,89 segundos;
- dirección: 17,75 segundos;
- revisión independiente de seguridad: 18,17 segundos;
- segunda revisión independiente: salida cero y matriz completa verde.

La matriz acredita:

- estructura, propietarios, RLS, ACL, restricciones, índices, disparadores y
  función exterior;
- dos identidades corporativas reales, cruces A/B en ambos sentidos y ausencia
  de historia parcial;
- escrituras concurrentes con una única cadena;
- carreras reales registro–revocación en ambos órdenes, observando
  `PgSleep` y `Lock`;
- errores por protocolo extendido con SQLSTATE y mensaje exactos;
- cuatro centinelas adversos posteriores a la ranura lógica, con marcador y
  SHA-256 como evidencia positiva de tránsito;
- ausencia de texto claro y de su representación hexadecimal en WAL, logs,
  recibos y trazas;
- referencias opacas presentes únicamente en tablas técnicas permitidas;
- barreras `000040` y `000041` sin efectos parciales.

## Reversión segura

La retirada no usa `CASCADE`. Compara manifestaciones exactas de relaciones,
columnas, restricciones, índices, políticas, ACL, disparadores propios e
internos, reglas, dependencias, publicación, estadísticas y etiquetas de
seguridad. Rechaza:

- cualquier deriva de catálogo;
- dependencias o herencia posteriores;
- barreras futuras;
- historia v2 existente.

La historia anterior se conserva y su huella se verifica después del ciclo
vacío de instalación y retirada.

## Hashes congelados

| Fichero | Líneas | SHA-256 |
|---|---:|---|
| `000039 up` | 800 | `13c1db8e477df871ec7614fcb994fab75ce41da5a3ec46888c468d2ba60377de` |
| `000039 down` | 548 | `f52fdb5195f3f05396f6539434594c5799ca129cbb0b7abf95ae9d12c791215b` |
| runner | 800 | `ada0315463a21019afb6233b066dcd2677e8d105c5937c5e5ca122fbd8f9fa5c` |
| prueba estructural | 303 | `3968538b53daea21c1acd6f82369c1d81a87396b62ded753d7f2fd22d4641395` |
| apoyo de pruebas | 316 | `fa6c48033c2be5ae9e1dc414e3efab5d147522a2b32268ce25a99086f65c61bf` |

`bash -n`, ShellCheck, `git diff --check` y Gitleaks quedaron verdes. No
quedaron contenedores, ranuras lógicas ni temporales.

## Dictamen y límites

Los dos revisores independientes emitieron `GO` sobre estos hashes. El corte
resuelve CT `000039`, pero no cierra la vertical O4-05. Permanecen abiertos:

1. contrato tipado CT `000040`;
2. ejecución interna CT `000041`;
3. fachadas exteriores CT `000042`;
4. adaptador Go y composición raíz;
5. matriz TLS viva y E2E HTTP/web.

Por ello se mantienen Contratación temporal `19/46` (41 %), O4-05 `3/5`,
Bolsa `1/14` (7 %) y producción `NO-GO`.
