# Recuperación histórica de firmas: CT175

Requiere AD178 (fachada y comprobador de recuperación) y AUT41, en ese orden; se aplica con `deploy/principal/lista_sql_claude_firmas_recuperacion_20261005.txt`. No instala ni reaplica CT172. No hay recorrido nominal acreditado.

La nueva fachada `recuperar_firmas_r5_atestadas_v2` recibe los once argumentos habituales de la consulta CT. Consume una autorización nueva de recuperación, con 48 campos y sin obligaciones, y conserva su auditoría común en la misma transacción. El adaptador Go debe validar la respuesta y confirmar la transacción antes de entregar datos.

CT selecciona las filas de su expediente, tipo de documento y versión. Si la clave corresponde a una operación histórica, comprueba también paso, catálogo, vía y revisión V2. Limita la lista a la secuencia de ese recibo y los indicadores al corte global que observó. Una firma posterior no debe cambiar una respuesta histórica. Si faltan posiciones acreditadas en el corte global original —por ejemplo, firmas legacy sin revisión observada—, la separación permanece no acreditada. Se cuentan posiciones distintas de todas las filas previas del expediente; una duplicación no cubre una laguna. El legado futuro no altera ese cálculo. El máximo es 128 filas; una historia mayor se rechaza, sin truncarla.

El lector privado CT170 proyecta cada fila seleccionada. Para las revisiones V2, CT forma el selector AUT41 desde sus referencias originales. AUT41 comprueba el consumo actual y devuelve el canon original. No reconstruye nombres, cargos o competencia con los datos actuales. `MaterialRootSHA256` es la huella del material de registro original (`solicitud_huella_sha256`), no la raíz criptográfica de la capacidad de consulta.

La respuesta conserva `Firmas` y `RevisionesPDF` y añade `Recuperaciones`: una fila por revisión V2, con `FirmaRef`, `MaterialRootSHA256`, `CanonNominal`, `CanonNominalSHA256` y `CanonNominalRef`. Falta de evidencia, huella distinta o consulta caducada abortan la operación completa. Los bytes del canon se convierten a UTF-8 sin volver a serializarlos.

Sólo el ejecutor CT recibe EXECUTE; las tablas y el lector AUT siguen privados. No hay nuevas tablas, permisos humanos, auditoría propia ni cambios al histórico instalado.

Ensayo del 5 de octubre de 2026 en el clon de la principal posterior a AD193 (repetido sobre main con AD195 y AD196), con AD178, AD177 y AUT41: UP único con código 0. `pruebas_sql/recuperacion_firmas_ct175.sql`, ejecutada con un LOGIN del runtime CT, comprueba que el runtime sólo ve esta fachada (no la de AD178 ni la lectura AUT41) y tres rechazos: material con una clave de más (22023), capacidad de otro expediente (42501, antes de consumir) y una decisión con la forma correcta pero sin atestación, que llega al núcleo y se deniega allí.

`pruebas_sql/recuperacion_firmas_ct175_positivo_clon.sql` (sólo clon desechable, ROLLBACK) recorre el camino positivo CT175 → AUT41 → comprobador AD178 reales con un doble de la fachada de consumo y filas sintéticas: devuelve una firma, una revisión y una recuperación con la huella del canon correcta, y un canon manipulado se rechaza en AUT41 (55000). Esa prueba encontró un fallo del borrador: `SELECT revision,... INTO cabeza` chocaba con la variable `revision` (42702) y la función nunca devolvía datos; ahora las columnas van calificadas.

El parseo inicial usa un bloque EXCEPTION antes de consumir. Con el sello `transaccion_origen` de AD193 esto ya no afecta al comprobador: el sello es el TopXID aunque haya subtransacciones.

Pendiente: recorrido causal con un consumo real, adaptador/HTTP con auditoría de denegados y errores, y recorrido con dos firmas y recuperación tras reiniciar.
