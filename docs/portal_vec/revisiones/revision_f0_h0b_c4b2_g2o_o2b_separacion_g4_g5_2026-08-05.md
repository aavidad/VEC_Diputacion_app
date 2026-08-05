# Revisión O2b: separación de producción G4 y autoprueba G5

Fecha: 5 de agosto de 2026.

Dictamen: **GO documental final, `P0=0`, `P1=0`, `P2=0`**.

## Objeto

Se revisó íntegramente la enmienda:

```text
docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o2b_separacion_g4_g5_2026-08-05.md
```

La versión normativa revisada antes de actualizar únicamente su estado tenía
269 líneas y esta huella:

```text
d747d3d1344945790a23b1a62c05675885b97da64f7de4669d2b124c8e32cdc2
```

La actualización de estado y esta acta se someten a una contrarrevisión final
antes del commit. El commit publicado será la autoridad material y se
registrará como padre exacto del futuro candidato O2b.

## Revisiones independientes

La revisión funcional comprobó la máquina S1--S5, propiedad y limpieza del
sobre, error pegajoso, matriz de secuencias/EOF y cobertura negativa. Emitió
`GO`, sin hallazgos abiertos.

La revisión de ledger y evidencia comprobó:

- G4 productiva y G5 de pruebas separadas;
- imports exactos por fichero y `fmt`/`strings` exclusivos de G5;
- runner 794 -> 802 mediante ocho líneas legibles;
- G1 689 -> 692 y manifiesto 7 -> 9 en orden exacto;
- build, `go vet`, captura y acreditación de G1+G4+G5+G2+G3;
- línea base positiva, AST y 25 mutantes sin falsos muertos;
- límites absolutos G4 <= 420 y G5 <= 540.

Durante la primera vuelta detectó una contradicción: se exigían imports
«disjuntos» aunque G4 y G5 comparten `errors`. Dirección sustituyó esa palabra
por bloques comprobados por separado, con `fmt` y `strings` exclusivos de G5.
Ambas revisiones contracomprobaron la nueva huella y emitieron
`P0=P1=P2=0`.

## Hallazgo material incorporado

La revisión funcional del prototipo detenido detectó que la invariante activa
no volvía a comprobar la presencia de todos los campos estructurales del único
sobre después del constructor. La enmienda obliga a que una alteración por
alias de nonce, PID, selector, identidad, longitud o ticket termine en fallo
interno, tupla cero, retirada completa y primer error pegajoso. También enumera
los vectores ausentes que G5 debe añadir; separar ficheros no permite rebajar
la cobertura.

## Trazabilidad de la parada

El prototipo G4 detenido permanece sin commit sobre `acf6016`, ocupa 647
líneas y tiene SHA-256:

```text
fcddd5ba2e748e490f44fc0eee2f364f396c5056b02eca3613a1bee904702cd7
```

No se modificaron G1, runner, G2, G3 ni otros ficheros. La parada por superar
600 líneas funcionó como estaba diseñada. El prototipo no es evidencia ni
candidato aceptado.

## Autorización limitada

Este dictamen autoriza confirmar y publicar exclusivamente la enmienda y esta
acta. La edición material solo se reanuda tras CI de cinco puertas verde sobre
ese commit y avance por fast-forward de la rama candidata al nuevo padre.

No se autoriza ampliar la API, omitir pruebas, mover producción a G5, tocar
G2/G3, activar Bash o aceptar un ledger distinto. La integración de código
seguirá necesitando línea base verde, AST, 25 mutantes, H0 real, puertas
globales y dos revisiones de código independientes.

No se cierra O2b ni cambia ninguna métrica: F0 `10/23`, O4-05 `3/5`,
Contratación temporal `24/46`, Bolsa productiva `1/14` y producción `NO-GO`.
