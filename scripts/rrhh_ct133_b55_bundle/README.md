# Ensayo RRHH 4.08 + 2.05 en PostgreSQL 18.4

Ejecutar desde el candidato ensamblado `bb67ac3ac6535cda5acd27a62677d9e09a531725` o un descendiente que solo añada este runner:

```bash
bash scripts/rrhh_ct133_b55_bundle/probar_pg18.sh --synthetic
```

El runner usa `postgres:18.4-alpine`, `--rm --network none` y directorios temporales propios en `/dev/shm`. Elimina contenedores y datos al salir. Solo usa catálogos y expediente sintéticos. Verifica la versión, imprime SHA256 de las ocho migraciones, rechaza doble UP, instala CT131, CT133, AD3-99, CT135, AD3-100 y CT137 reales en ese orden, y provisiona la publicación inicial por `cmd/vec-provision-plantillas` con LOGIN migrador propio. Añade el tipo `prueba_tipo_rrhh` al catálogo temporal, comprueba el replay, el preflight Go con LOGIN CT, fuente y motivos separados, ACL directas cerradas, revocaciones, tríada de organización, procedencia y huellas, y repite provisión y lectura tras reiniciar PostgreSQL.

Después ejecuta copias de las pruebas focales del candidato para AD3-101 y B55, cada una en su base efímera. Las copias fijan PostgreSQL 18.4, limpian por el ID del contenedor que crearon y omiten la prueba de DOWN después de crear historia. AD3-101 necesita la preimagen exacta del núcleo AD3-97; B55 usa B46 real y una fachada AD3-101 sintética. El ensayo conjunto sobre un único núcleo V3 real **no está acreditado**. Las pruebas focales no prueban criptografía, PDP, sesión corporativa, API ni navegador. El actor mal formado se rechaza en CT137; la vinculación de un actor ajeno a una decisión válida exige V3 real. Los dobles de autorización, fuente/motivos y el expediente se usan solo para estructura y guardas locales. Los DOWN focales se ejecutan únicamente antes de crear historia.

## Evidencia del ensamblado

Comando exacto: `bash scripts/rrhh_ct133_b55_bundle/probar_pg18.sh --synthetic` sobre descendiente de `bb67ac3ac6535cda5acd27a62677d9e09a531725`; salida **0** el 28 de septiembre de 2026. Pasaron CT131/133/135/137, AD3-99/100, provisión CLI y replay, preflight positivo y revocaciones negativas con roles separados, lectura documental local, persistencia tras reinicio, y los ensayos focales AD3-101/B55. Cada migración se probó primero en `ROLLBACK` cuando su objeto nuevo aún no existía, conservando las filas anteriores; se rechazó el segundo UP. No se instaló nada en bases conservadas.

SHA256 de los SQL ejecutados en el árbol ensamblado:

| Migración | SHA256 |
| --- | --- |
| CT131 | `a03d6d5192fc8f5b632a4d047aa27fa67e360845a1d65e88e2e16877436083a2` |
| CT133 | `884ac3bd58c9c0ec09a09c1d3449db4386f00a2a7fdb47c16de7e0365204bb57` |
| AD3-99 | `6e2168f14a2b66d250328995ca966f948f725888439bff4da0fe84088d676eaa` |
| CT135 | `69d21d66453b5492d5d3569e97325267546edcd9d24f0ac0de16371e3f9e9d3c` |
| AD3-100 | `a50c12e94fddc79701bba2e6a84a8d48049b5f63323e67883f631ce75e2836ac` |
| CT137 | `7544e889b097f070aa8115649604db0df6201588a137dac6f75ddf0463ae0d44` |
| AD3-101 | `6f53ac600930bbbfbd0cdc057e24c639f10f7fb2c143f508f2e7d65635beb418` |
| B55 | `37ade59e00684df3b7b026db96a89f81c164291eb0a11afda77a9267b0ee0866` |
