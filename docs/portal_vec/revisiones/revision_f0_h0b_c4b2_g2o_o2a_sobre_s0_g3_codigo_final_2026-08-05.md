# Revisión final F0-H0b/C4b-2/G2-O/O2a/S0/G3

Fecha: 5 de agosto de 2026.

Estado: **GO integrado, publicado y con CI verde; P0=0, P1=0 y P2=0**.

## Alcance exacto

La minitarea incorpora la fuente privada G3 que recibe una única trama
`SOBRE` en la fase S0 del supervisor M38. Conserva y prueba estas garantías:

- recepción incremental, incluso byte a byte;
- transición S0 a S1 solo después de un EOF limpio;
- retención inmutable de seis campos mediante una única asignación canónica;
- rechazo de datos posteriores, segundas recepciones y estados imposibles;
- error enclavado y borrado del búfer, su longitud y cualquier sobre sensible
  ante todos los fallos producidos en S0;
- rechazo del uso posterior en S1 sin alterar la fase ni el sobre ya retenido;
- límites de 4096 bytes para la trama y 2048 bytes para el ticket;
- ausencia de catálogo de selectores, fases posteriores, red, SQL, procesos,
  FD, Bash operativo o nueva autoridad.

No abre todavía el modo operativo. `--supervisar-m38` y los modos desconocidos
continúan fallando cerrados con estado 64. O2b conserva la autoridad exclusiva
sobre `ARMAR` y la cancelación.

## Trazabilidad

| Hito | Referencia |
| --- | --- |
| Base exacta | `d148d7625ffa14a435b5ffdfa3315cd3984dd52f` |
| Contrato autorizado | `d148d7625ffa14a435b5ffdfa3315cd3984dd52f` |
| Primer candidato rechazado | `2ec9886cc0a669bf62a29e4cb2660ec8647bc03e` |
| Candidato corregido | `50aec513c535980d80bee28ca10321ed4ed7e04f` |
| Evidencia previa | `6a8c1ee` |
| Integración | `0caa140` |
| Publicación y relevo | `ef1f08b` |
| CI | `31021785711`, cinco de cinco puertas verdes |

El primer candidato se rechazó antes de integrar porque un fallo interno podía
dejar material sensible en el lector y porque el analizador AST y el mutador no
tenían evidencia durable. El candidato corregido centraliza el fallo en
`fallarRecepcionSobreS0M38`, limpia todos los estados sensibles y amplía la
autoprueba con una invariante L1 material.

## Write-set y ledger

El candidato modifica exclusivamente tres ficheros:

1. el runner H0 PostgreSQL;
2. la fuente privada G1 del supervisor;
3. la nueva fuente privada G3 de S0.

```text
runner  +20/-9
G1      +3/-0
G3      +431/-0
```

| Unidad | Líneas | SHA-256 |
| --- | ---: | --- |
| Runner H0 | 794 | `fc8c27a3b6ef1651a9ef97676a4ca9d34924fa5dbb026921a7a1e529cda81176` |
| D2d invariante | 164 | `039b75dd15a2888798c7f257c46fdbb97587cbdd4a6519e11cb043cce0e72e5e` |
| G1 | 689 | `f9ab7b20accac9af56cfcb5e42c25c62b087d7e0ee81a2fea09250a35fc0c58f` |
| G2 | 798 | `01acb818e9abefcbfe4c279bb0dd5e3317bf03f082f1ed3fba4f257c5642866b` |
| G3 | 431 | `d608868ecb2cb753876f488b522975e05af06c013c82222959be5d85100c3633` |
| Binario, dos construcciones aisladas | — | `46d247156316b56ca5f30082c4964aaaefdc2442eb8fb8685f595aeb230dde30` |

El manifiesto privado aumenta de seis a siete fuentes. `go list` mantiene G1,
G2 y G3 fuera de `GoFiles`; solo el capturador pertenece al paquete ordinario.

## Revisiones independientes

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Funcional, memoria sensible y fallo cerrado | 0 | 0 | 0 | GO |
| Ledger, AST, mutantes y portabilidad | 0 | 0 | 0 | GO |

Ambas revisiones se realizaron sobre el candidato inmutable `50aec51`. La
segunda revisión emitió primero NO-GO por una ruta privada y por seleccionar
Go de forma no uniforme dentro del mutador. La evidencia se corrigió sin tocar
el candidato, se recalculó y se reprodujo de nuevo. El veredicto final cerró
ese P1 con `P0=P1=P2=0`.

## Evidencia durable y portable

| Artefacto | SHA-256 |
| --- | --- |
| [Analizador AST](evidencias/f0_h0b_c4b2_g2o_o2a_ast_50aec51.go.txt) | `86de530ad50726137e84c12ac9c4e7829eb55d6bae67fc3df2161a6ff685ef27` |
| [Mutador](evidencias/f0_h0b_c4b2_g2o_o2a_mutantes_50aec51.sh.txt) | `1a876441eb28e65d425fdfe5840bc5c456509ba3292f56d2cb93230e0bb4120d` |

Los artefactos se conservan como texto no ejecutable y no se cargan desde el
runner. No incluyen rutas privadas. El mutador recibe la raíz del repositorio
y el analizador materializado, acepta opcionalmente `VEP_GO_BIN` y exige de
forma uniforme Go 1.26.5 para `version`, `build` y `run`.

Reproducción histórica: crear un worktree limpio separado y situarlo
exactamente en `0caa140`. El analizador y el mutador están ligados a las
huellas de O2a y no deben ejecutarse como si fueran validadores genéricos sobre
un descendiente con O2b u otras fases:

```bash
raiz=$(git rev-parse --show-toplevel)
temporal=$(mktemp -d)
cp docs/portal_vec/revisiones/evidencias/f0_h0b_c4b2_g2o_o2a_ast_50aec51.go.txt \
  "${temporal}/analizador.go"
cp docs/portal_vec/revisiones/evidencias/f0_h0b_c4b2_g2o_o2a_mutantes_50aec51.sh.txt \
  "${temporal}/mutantes.sh"
env GOENV=off GOTOOLCHAIN=go1.26.5 go run "${temporal}/analizador.go" -- \
  deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_sobre_s0.go
bash "${temporal}/mutantes.sh" "${raiz}" "${temporal}/analizador.go"
find "${temporal}" -type f -exec unlink -- {} \;
find "${temporal}" -depth -type d -exec rmdir -- {} \;
```

La extensión `.go.txt` es deliberada: antes de invocar Go se materializa una
copia temporal con extensión `.go`.

## Salida íntegra reproducida

```text
ast_o2a=ok imports=3 construcciones=1 pruebas_inalcanzables=ok
mutante=01_constructor_clase huella=38071a5866dea7c039a9259302511a1dcf995f2078ea877fc623ed6f03ea7c36 resultado=MUERTO
mutante=02_s1_antes_eof huella=3c64733fd4110174578dfc328bfc74cc025fbba389c4fa0d5abe53d52a8ea9c8 resultado=MUERTO
mutante=03_eof_inicial huella=b3c4372c2dc8a5343fc8b08a602818b9930b66c9247d353d37cc850bc6ae8ac4 resultado=MUERTO
mutante=04_eof_parcial huella=f03911b4e6efad129285614695904a014105992472e4ceba6b08bcba14bf5623 resultado=MUERTO
mutante=05_cola_ignorada huella=9883bc8f1d14ab2145bcdd8859780246efdd3ab16c8dbd7e85415584cb08e985 resultado=MUERTO
mutante=06_resultado_imposible_aceptado huella=58e15f6ca0a363eb01d46957716330a14b6e389379f3ddd19e2a1439964c5fbb resultado=MUERTO
mutante=07_codec_omitido huella=c1941f052d71e6a5fe1c5a3f5884e155b07b23015e26ba74b9c75a4f044ed612 resultado=MUERTO
mutante=08_fase_antes_retencion huella=2ebc842a71e577eef94fee9c4c6a0e30cc37df1981c3904c8c10a3b69d23bd44 resultado=MUERTO_AST
mutante=09_campos_intercambiados huella=1f1e90d3e5f3a3cd9621247f79764040e43ab16317be861f400337ac4379ccad resultado=MUERTO
mutante=10_retencion_incompleta huella=f9d618199d0ff2e8569474c4356c89c0a5173e89ebfae1638c5ac0ec381b935c resultado=MUERTO
mutante=11_copia_mutable huella=93d3dab36a046886500831d1cd45833e2e970f8ad13e590ece810a6a7065e28b resultado=MUERTO_AST
mutante=12_error_pegajoso_sustituido huella=490f3c2474ea3ac6df5511e46019bd81a0e2279aff28e706215ab1d2d3486a10 resultado=MUERTO
mutante=13_error_con_resultado_util huella=f1cec0889bc0e8dd3c63a52976f199373afc650041ab07f14b90685c37c6ee09 resultado=MUERTO
mutante=14_segunda_recepcion huella=bb6d146933e8b94040e37b5d04953b960d2f02b31405d9bb492461cf5e91a86e resultado=MUERTO
mutante=15_catalogo_local huella=bccc2c8bbf7a860b12c0bad79cd63c2342be3a34cdce78a93b8d4d913d78f73f resultado=MUERTO_AST
mutante=16_fase_incorporada huella=099c1f9f6c4801a2695e9fb7b83d577801f7b1e6f1d6f8a2a8cab4f3bcd70adb resultado=MUERTO_AST
mutante=17_modo_operativo huella=8bdab88bf9e1150ba56018646386009c93d2cc4cc840bc7bb458912850938c3a resultado=MUERTO estado=0
mutante=estructural_segunda_copia huella=6af9d1069fc5484510937169e33ea105816375dd43f65c420c658c2e9a8343ca resultado=MUERTO_AST
mutante=estructural_getter huella=59b85b7d6714d91a1486c1824b7f2e07a11169fcd29f8a0f88e987c83977d4db resultado=MUERTO_AST
mutante=estructural_import_log huella=ef39fec511f9a0fd4d0065df69f492ab18274d904d9e7ffc481076df6d30a6c8 resultado=MUERTO_AST
mutante=estructural_import_os huella=989c5348ec48bdd73072e38ad0b3789c6de6b976489733aae751423e5af13768 resultado=MUERTO_AST
mutante=estructural_import_net_http huella=ffabb65dd88ce5f947d3cb5beea0d408891388d150c7e4d7962dacf7d119ab58 resultado=MUERTO_AST
mutante=estructural_import_time huella=abb1392e8f03186a2dde35733456b776b00de6908994f881a55d6376ebe70c56 resultado=MUERTO_AST
mutante=estructural_import_sync huella=fd7a26c117e75b6e6932f1f602c40121305a4603f9b80cce891944feca884072 resultado=MUERTO_AST
mutante=estructural_import_encoding_json huella=43b3da4e1692d1c6ae7d5a5e8c33325cbac76f42f5119f3692b35414c260941b resultado=MUERTO_AST
mutante=estructural_import_os_exec huella=dde58c521b86d27f23ed249e12be7d01339451f83f97e2c0cc22067dfcb3d42e resultado=MUERTO_AST
mutante=estructural_import_syscall huella=fe909fd4582cda7149175bb532314b2e4b5374316bd378ad383e713707f767f7 resultado=MUERTO_AST
mutantes_o2a=ok conductuales=17 estructurales=10
```

## Puertas reproducidas

- AST positivo y restricciones negativas: verde;
- 17 mutantes conductuales y 10 estructurales: muertos;
- dos construcciones privadas reproducibles: misma huella;
- manifiesto de siete fuentes, modos y permisos: verde;
- H0 completo en PostgreSQL 18.4 por digest y sin red: verde;
- `go test ./...`, `go test -race ./...` y `go vet ./...`: verde;
- `scripts/verificar_calidad.sh`: verde;
- Bash, ShellCheck, `git diff --check` y Gitleaks: verde;
- contenedores y temporales residuales: cero.

Dirección reprodujo esas puertas después de integrar. La primera llamada H0
mediante `bash` fue rechazada en 65 antes de reservar recursos porque no
respetaba el régimen acreditado del ejecutable. La ejecución material directa,
con `LD_LIBRARY_PATH` retirado, completó PostgreSQL 18.4, snapshot, arnés
transaccional y clasificación de errores reales. Después,
`scripts/verificar_calidad.sh` terminó verde, incluida la carrera global, los
grafos y manifiestos aislados, TLS como proceso no privilegiado, auditoría de
vulnerabilidades y tamaños. No quedaron contenedores ni temporales F0.

## Métrica y continuación

Este cierre interno no modifica Contratación temporal `24/46` ni Bolsa
productiva `1/14`. Producción permanece en `NO-GO`.

La siguiente minitarea es O2b: `ARMAR` y cancelación sin Bash, mediante un
contrato nuevo y acotado; O2a no autoriza esa conducta.
