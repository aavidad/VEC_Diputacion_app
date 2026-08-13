# Revisión de seguridad O4C-P0-CONTRATO

Fecha: 13 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/o4c-p0-seguridad-20260813`, sin edición del worktree ni de la rama
productora.

Dictamen: **GO**, `P0=0`, `P1=0`, `P2=0`.

## Corte exacto y aislamiento

Se revisó exclusivamente el candidato documental
`64351a469df4e7bba4850ef1500d9e2c4bf378de`, sobre la base publicada
`5345d5d097b51ab3567983f048feabeceaf2957b`. Su árbol es
`2e289121938cde46f6682ef779a03e5d6766a417` y el rango contiene cuatro
commits:

```text
26f55ed docs(O4C-P0-CONTRATO): fija terminalidad y limpieza
870805e docs(O4C-P0): fija el SHA de relevo O4A-P4
909d19f docs(O4C-P0): actualiza el relevo O4A-P4
64351a4 docs(O4C-P0): actualiza el SHA final O4A-P4
```

El write-set productor son solo dos altas Markdown:

| Fichero | Líneas | SHA-256 |
| --- | ---: | --- |
| `docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md` | 552 | `924f76b92d5988875866065eaaacf5405fd7d5c9a2f3e7013ba516c8fc6cc79c` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md` | 97 | `3e6f603c81dd1e00b10da9889e2160f01a235c97e73a34928ce6a13700f51156` |

No se revisó como parte de este dictamen código O4c inexistente ni el candidato
O4A-P4. La decisión conserva el primer SHA de aquella cadena para dejarlo fuera
de genealogía; el checkpoint del SHA objetivo enumera la cadena completa hasta
`2b7eaf498f8f68a90b66c004f166a62f070b5064` y también la mantiene fuera,
pendiente de revisión. No existe acreditación implícita de esos bytes.

## Autoridades y huellas

Coinciden con los bytes del checkout todas las autoridades declaradas:

- O4a: 535 líneas, SHA-256
  `ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`;
- checkpoint O4a:
  `0ba877cd831f7957d36cab885644dd6f8e5069d80e1629669b92d282e99d1513`;
- O4b: 443 líneas, SHA-256
  `675d33b6f96ef441843721effd332a82242ed9257a2af2246c75ed22f2984c7f`;
- checkpoint O4b:
  `1335c81e8a2ba574d21bbdd86ff853c7211fa58d8274c8763797cd1380906a45`;
- O3c y su ledger final:
  `f47395d68fa3f9e39e118f81b07fde8d8792aa61d4820dfb676ff4c7216515b6`
  y `a1de39d1c80492cae8b6b858a096d8f3051b346913064fe51a83dd6573dbb3b1`;
- corrección canónica O1a:
  `29c1520f6aab91d832ec6ddb41efd75df587d240b8fb1e7dc51f107df831a659`.

La base O4b y la CI `31546649383` se citan solo como genealogía; este revisor no
consultó producción ni modifica su acreditación histórica.

## Controles de seguridad acreditados

- **Autoridad opaca y lineal.** O4A-P5 deberá preasignar un único agregado no
  serializable, transferir observador y después lease mediante CAS, anular el
  origen y dejar A8. O4c consume por doble puntero con ganador único; nulo,
  alias, clon, replay, estado, owner, generación, TID, `pending`, sello o
  recurso adverso niegan por defecto sin fabricar custodia.
- **Partición de autoridad.** O4c no decide causa, tiempo o etapa y no señala.
  Es el único propietario posterior del Wait funcional, drenaje, ECHILD,
  ESRCH, TERMINAL, cierres e inventario; no hay arista inversa ni getter que
  devuelva PID, pidfd, nonce o handles.
- **Terminalidad antes de recolección.** Primario y reserva deben concordar por
  evidencia pidfd natural y dentro del límite absoluto. Vida, discordancia,
  error, igualdad o vencimiento niegan Wait. Solo entonces se consume una vez
  el handle mediante `esperarConLeaseO3aM38`; no hay segundo Wait, waitid,
  fallback por PID/PGID ni inferencia desde `ProcessState` previa.
- **Drenaje cerrado.** `Wait4(-1, WNOHANG)` solo acepta PID positivo o ECHILD;
  PID cero, hijo vivo, otro error o borde son fatal. ECHILD debe preceder a la
  sonda de grupo y solo ESRCH acredita ausencia. No se acepta retorno cero,
  EPERM, EINTR, `/proc`, reserva ni señal repetida.
- **Syscalls y lease.** Cada observación, Wait4, sonda, cierre o escritura usa
  permiso distinto y su raw solo se interpreta después de consolidación
  crítica. La máquina es de una goroutine y TID, sin Wait/cierre concurrente,
  callback, canal o actor auxiliar.
- **Cierres y publicación.** Pidfd primario, reserva y CONTROL se cierran una
  vez tras ECHILD/ESRCH. TERMINAL usa una única trama prevalidada; un parcial
  continúa solo el sufijo, con máximo ocho EINTR, y nunca reinicia el offset.
  Incidente previo o nuevo impide terminal normal y fuerza 65/cuarentena.
- **Liberación.** Inventario exacto precede al CAS de observador; lease se
  libera al final y ninguna syscall, lectura, escritura, snapshot, log o
  asignación falible ocurre después. No hay rollback ni éxito degradado ante
  propiedad o cierre inciertos.
- **Límites y minimización.** Todos los relojes son absolutos monotónicos y no
  reiniciables; terminalidad/drenaje están acotados por el límite prestado, la
  trama completa por 1024 bytes, EINTR por ocho y cada fuente material por
  parada 650/tope 800. El resultado a O5 excluye PID, PGID, pidfd, nonce,
  rutas, errores libres, actores, tenant y datos humanos.
- **Cobertura exigida.** La matriz OC01--OC23 y sus mutantes incluyen carreras,
  bordes temporales, Wait cardinal uno, PID cero, hijo vivo, ECHILD/ESRCH,
  escritura parcial, cierres, liberación partida, residuos y ausencia de
  procesos ajenos. Cualquier superviviente es parada dura.

La contradicción terminal O4a/O4b observada en la revisión separada de
O4A-P4 no se oculta ni se resuelve aquí. Este contrato no posee esas etapas,
mantiene O4A-P4 fuera de su genealogía y bloquea O4C-P1 hasta recibir O4A-P5
material acreditado. Por ello no cambia el dictamen interno de O4C-P0 ni
desbloquea su implementación.

## Puertas reproducidas

- base, HEAD, padre, árbol, rama exclusiva y rango de cuatro commits: exactos;
- `git diff --name-status` desde la base: solo las dos altas declaradas;
- `sha256sum` y `wc -l` de decisión, checkpoint y siete autoridades: exactos;
- comprobación de todos los enlaces Markdown locales: verde;
- búsqueda focal de patrones de secretos, tokens, claves, DSN y datos
  sensibles: solo menciones normativas, cero material sensible;
- Gitleaks sobre
  `5345d5d097b51ab3567983f048feabeceaf2957b..64351a469df4e7bba4850ef1500d9e2c4bf378de`:
  4 commits, 32,60 KB, cero filtraciones;
- `git diff --check` sobre el rango exacto: verde;
- worktree antes del acta: limpio.

Focal Go normal/race, gofmt, vet, mutantes, PostgreSQL y E2E no aplican a este
P0 de dos documentos; no hay paquete, fuente, ejecutable, SQL ni composición
modificados. Esas puertas quedan exigidas explícitamente para P1--P7 y no se
declaran ejecutadas por este GO.

## Cierre

No quedan hallazgos P0, P1 ni P2 en el contrato O4C-P0 exacto. El GO solo
habilita su revisión/integración documental por dirección cuando exista el
segundo GO independiente; no abre O4C-P1, O4A-P5, O5/O6, publicación,
despliegue, producción ni cambio de métricas.
