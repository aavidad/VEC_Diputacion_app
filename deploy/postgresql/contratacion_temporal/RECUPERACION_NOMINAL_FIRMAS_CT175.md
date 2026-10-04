# Recuperación histórica de firmas: CT175

Borrador privado. Requiere AD178, su comprobador de consumo actual y AUT41, todavía pendientes de ensayo causal. No instalar ni reaplicar CT172. No hay recorrido nominal acreditado.

La nueva fachada `recuperar_firmas_r5_atestadas_v2` recibe los once argumentos habituales de la consulta CT. Consume una autorización nueva de recuperación, con 48 campos y sin obligaciones, y conserva su auditoría común en la misma transacción. El adaptador Go debe validar la respuesta y confirmar la transacción antes de entregar datos.

CT selecciona las filas de su expediente, tipo de documento y versión. Si la clave corresponde a una operación histórica, comprueba también paso, catálogo, vía y revisión V2. Limita la lista a la secuencia de ese recibo y los indicadores al corte global que observó. Una firma posterior no debe cambiar una respuesta histórica. Si faltan posiciones acreditadas en el corte global original —por ejemplo, firmas legacy sin revisión observada—, la separación permanece no acreditada. Se cuentan posiciones distintas de todas las filas previas del expediente; una duplicación no cubre una laguna. El legado futuro no altera ese cálculo. El máximo es 128 filas; una historia mayor se rechaza, sin truncarla.

El lector privado CT170 proyecta cada fila seleccionada. Para las revisiones V2, CT forma el selector AUT41 desde sus referencias originales. AUT41 comprueba el consumo actual y devuelve el canon original. No reconstruye nombres, cargos o competencia con los datos actuales. `MaterialRootSHA256` es la huella del material de registro original (`solicitud_huella_sha256`), no la raíz criptográfica de la capacidad de consulta.

La respuesta conserva `Firmas` y `RevisionesPDF` y añade `Recuperaciones`: una fila por revisión V2, con `FirmaRef`, `MaterialRootSHA256`, `CanonNominal`, `CanonNominalSHA256` y `CanonNominalRef`. Falta de evidencia, huella distinta o consulta caducada abortan la operación completa. Los bytes del canon se convierten a UTF-8 sin volver a serializarlos.

Sólo el ejecutor CT recibe EXECUTE; las tablas y el lector AUT siguen privados. No hay nuevas tablas, permisos humanos, auditoría propia ni cambios al histórico instalado.

Pendiente: dos revisiones sensibles del hash final, ensayo PostgreSQL 18 sobre la copia fría causal de K/L, adaptador/HTTP con auditoría de denegados y errores, y recorrido con dos firmas y recuperación tras reiniciar. Una prueba estructural no sustituye ese ensayo.

El lector no envuelve el consumo en un bloque EXCEPTION: conserva los errores PostgreSQL de serialización, bloqueos o validación para el adaptador y su auditoría. La comparación TopXID/SubXID de las funciones instaladas sigue pendiente de corrección por la autoridad AD; retirar este bloque no resuelve por sí solo ese bloqueo común.
