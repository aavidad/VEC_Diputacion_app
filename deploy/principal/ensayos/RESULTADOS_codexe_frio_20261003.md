# Ensayo frío del circuito de firmas, 3 de octubre de 2026

El orden autorizado aplicó quince migraciones nuevas y se detuvo en AD151. La candidata no quedó instalada. La copia PostgreSQL se conserva para las reparaciones y el siguiente ensayo.

```text
AD3-151: PARO clave=consumidor_ad149_instalado actual=false esperado=true
SQLSTATE: 55000
```

Base de código: `936aac665`, con la corrección AD155 `a5ffc334a`. Copia fría SHA256: `a1f56a65d53c6aaa20ab9f9b753f08ce80d390178ae03d6926072722ae192ce4`. Imagen PostgreSQL 18.4: `sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a`.

El ensayo usó un contenedor propio, usuario sin privilegios de sistema, sin red externa y sin puertos publicados. No se ejecutaron DOWN ni se reaplicaron migraciones instaladas. La principal y cidonia no participaron.

## Resultados

- Inventario inicial: las 25 piezas de la lista estaban ausentes.
- Instaladas en la copia: H7 (CT162, B71, B70, CT165, CT166, CT167, B74, B75 y B72), H8 (B73, B76 y CT169), AD155 corregida, B77 y Convoca XLSX5.
- Historia: las 469 tablas anteriores conservan recuentos y SHA256 al comparar sus atributos originales. CT165, CT166, B72 y B76 añaden atributos; sus proyecciones históricas coinciden con la preimagen. B77 crea dos tablas nuevas, vacías.
- Roles: huella completa conservada durante las quince instalaciones.
- AD151 fallida: snapshots previos y posteriores idénticos en historia, roles, ACL de tablas y definiciones/ACL de funciones.
- CT163, CT164, AD156, AD157, AD158, AD159, Documentos13, AUT30 y CT170: no aplicadas por el bloqueo causal.
- No se probó el recorrido de firma en navegador ni recuperación de un recibo de firma.

## Reparación requerida

AD151 exige la postimagen AD149, que falta en la copia. Las SQL AD141, AD142 y AD144 tampoco están instaladas; faltan sus roles y esquemas de Convocatorias, Méritos y Baremo. AD149 exacta necesita esos consumidores y una huella completa del núcleo post-AD144. No puede instalarse sola en el estado actual.

El núcleo posterior a AD155 tiene definición SHA256 `5dbdac03a2a4e52ca4cb45c57b3bf18e621091da9e818091e30a3f90e68e7330` y fuente SHA256 `cdc8cb87f27360741a2d0d52e8b9d58d0a1423be8abd8ff9063f389b2c8ea75e`. Mantiene propietario, SECURITY DEFINER, ACL privada y configuración `search_path=pg_catalog, pg_temp; lock_timeout=2s`.

La reparación de las candidatas R5 debe comprobar la preimagen real completa y conservar los consumidores ya instalados, sus ACL, dependencias e historia. Las guardas históricas de AD151, AD156, AD157, AD158 y AD159 requieren revisión conjunta. CT164 comparte con CT165 la función `expediente_analisis_valido_v2`, cuyo contenido actual también debe cotejarse. No se modificaron las SQL candidatas durante este ensayo.

## Artefactos exactos

| Pieza | SHA256 de SQL | Resultado |
| --- | --- | --- |
| h7-000162 | `8733aa0709500212976bd99d1c68a620bfdb04d48b15f345da8fdb3ba9a6cfe7` | OK |
| h7-000071 | `7e3dbe53e22cc16e7013b15b4cb6715e6b4d7f053eaa74f7f92402645360c6fc` | OK |
| h7-000070 | `b4c2a3fad2e878c19098ec3d464092eafd39bf0b795880d20b714be7019786cf` | OK |
| h7-000165 | `7a7ac82c0137d77339996022e234c416843a2525cf426f306430c0c66a05bf6e` | OK |
| h7-000166 | `e6642ac29bf4b76f9315901f7f1c1e1bcd435d4f2e409990fcc5b039a17e94ab` | OK |
| h7-000167 | `961951e7f34fe120ded4e8230efde321eb33d824d0f1af1ebf6f365cc6494790` | OK |
| h7-000074 | `9a99fa259160d18d206fe9efa21d5769f1717e767f3faabff32b79d287b48b09` | OK |
| h7-000075 | `73a6a7eb5e3eba0dc2fe60852f030ba6ca10ab5af237bde38a9058751b247b9f` | OK |
| h7-000072 | `9946449b4f33c2026adec976a41046e2a4469e40e000ccf7f02f9af000556e40` | OK |
| h8-000073 | `dbebf36ac124130e668188281515606efd4a547d49aa6fefc24e942a42602bce` | OK |
| h8-000076 | `2674b156d46a7bcf14b33a496c79a97224accf4edf9e3189e8c16a12d73e598c` | OK |
| h8-000169 | `4a9f843125416de8f1683e20df4b6ed192f2b8e7ea947a688c33dc8718846ab4` | OK |
| AD155 | `c4380299caf3e00e3b14df343adf63754e40ea11afb393fee971a69af4b9e21d` | OK |
| B77 | `93f47d3dd2e0c9ed4065818416fc83b4f67b4c4e0c7e334a04e5911185f01a44` | OK |
| XLSX5 | `f575f6cf87d73d58531c4dfdd8f1ff272b24390060af32fdd4c74b22df11ebea` | OK |
| AD151 | `690da70327476b9bf780088c7468fd0c80bb3db3e8e714b0587c1e0f7816ed01` | PARO |
| CT163 | `4d8e54209c8c6bab2b6f061edb53d31b8648f434f15a1ec286748a44068931c1` | No ejecutada |
| CT164 | `cc26d7937799c908496f18d28f245fb526bd0c51d7d0c2394cc914cd67e32da8` | No ejecutada |
| AD156 | `41e8868634ce78e7a34f647d8898f954913ce53bfcf810ed405e97f2786ad747` | No ejecutada |
| AD157 | `b7d81c5cf25b6083b0be2e94d34e4d2834c3f21cbdf5a4a3a0f958ec87dc99b1` | No ejecutada |
| AD158 | `bad441b8c9d12cf10910c179452dab7c1bf63514c70dc57d690000ca10ec96be` | No ejecutada |
| AD159 | `05e2910ed66dbe07f461b73cb2f87f5758009871d6e9b775b25dd9d5ad9e27e1` | No ejecutada |
| DOC13 | `358603cc746aa3db4ffacbecf965da81f345dbc3ec07af2080705245d820d2b1` | No ejecutada |
| AUT30 | `704933566ff5170feb01bfc5f9a3a650be21189306d20348293733c4ae676c32` | No ejecutada |
| CT170 | `688998cca70e0f72665ea1e76d023c109702c5aa18848fc48a634e2d52c6cb48` | No ejecutada |

| PR | Commit ensayado o preparado |
| --- | --- |
| 440 | `8390c49045be0bfa34fd12b10e05bb9718751333` |
| 439 | `fbc2ad07e03021e4f62eddf0988667cac0718646` |
| 441 | `9df3ff4561000b9c5d6111bc8ec52035d2769e0e` |
| 448 | `ec6585abbd75843124f84552032808e3edc4277d` |
| 451 | `3be9250943df66f139f5d5560007fe8372d9bcad` |
| 457 | `82597cd4bf4c82a9958051e2c7a659ee4f797224` |
| 456 | `bfe0567a2a0a26b2861763856b0fe7f6fc0578dc` |
| 460 | `b65e4d35b2be9efeb618348c7c22745c2ff5915e` |
| 453 | `3caa4a50267c77ce0493b13ac50ca6a958ae76f0` |
| 464 | `824373a19417caa7cab7c53435bdf2a164d28d34` |

El script propio recibe un contenedor existente, una lista JSON con rutas y SHA256 exactos y un directorio privado de salida. Captura recuentos y huellas en el servidor; no devuelve filas de negocio. Los manifiestos, fuentes de preimagen y logs completos quedan fuera de Git.

## Continuación sobre la misma copia

Se ensayaron después las candidatas reancladas a la preimagen real. La cadena AD151, AD156, AD157 y AD158, commit `c49730001e5797983eaf5e288d78e591e6ce71a9`, terminó en `ENSAYO-OK`; las 471 tablas de historia y los roles conservaron sus huellas. AD157 mantiene el cierre al uso de firma VEC previsto en su contrato.

| Candidata nueva | SHA256 | Resultado |
| --- | --- | --- |
| AD151-r2 | `78a642daa482e9d7c6dbc33b9ec51d96dfabc66cd194d1290a4670f82a6593cf` | ENSAYO-OK |
| AD156-r2 | `07a774db534503a845d34eb710239d0a7b9508c684f4cec7c408c6b66cd2c87f` | ENSAYO-OK |
| AD157-r2 | `0faa42b50dd092f78d71d6693ad285c6a3489c83df6bc5293a38500219d12ccb` | ENSAYO-OK |
| AD158-r2 | `4a5c0004e01ca7eb8f0c8cb8e41f0fe18b137b910cbb254f28f0cfc8c73a0b7a` | ENSAYO-OK |

CT163 y Documentos13, con los artefactos originales identificados arriba, también terminaron en `ENSAYO-OK`. Las tablas anteriores conservaron sus datos y roles; Documentos13 añadió sus cuatro tablas de reserva y el catálogo gobernado de seis tipos. Total: 475 tablas.

Las pruebas SQL de guardas CT163, cierre al uso AD157, ACL AD158 y estructura Documentos13 pasaron en transacciones revertidas. La prueba de lógica de reserva documental pasó usando un doble temporal de V3. Acredita reserva, reintento, confirmación e idempotencia; no acredita criptografía ni concesión real. El ROLLBACK conservó exactamente historia, funciones, ACL, roles y catálogo de esquema.

AD159 conserva dos cierres expresos de instalación: preimagen posterior a AD158 acreditada y perfiles R5 publicados y vigentes. CT170 exige su fachada. Ambas capacidades siguen pendientes de candidatas y fuentes válidas.
