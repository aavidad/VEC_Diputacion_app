# Revisión independiente F0-H0b/C4b-2 G2-O0

Fecha: 5 de agosto de 2026.

Commit revisado: `c71cab964116c44164a46a022ba397b900192e63`.

Estado: **NO-GO documental consolidado**. Ningún fichero de producción fue
modificado por los revisores y no se autoriza O1.

## Revisiones recibidas

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Máquina de estados | 0 | 5 | 2 | NO-GO |
| Ledger y write-set | 0 | 7 | 2 | NO-GO |
| Seguridad y procesos | 0 | 6 | 0 | NO-GO |

Los tres revisores comprobaron el árbol exacto, los tamaños y las huellas. El
ledger físico de partida es correcto: runner 800, G1 683, G2 91, capturador
799, adaptador M38 527 y D2d 145 líneas. `git show --check c71cab9` también es
verde. El rechazo se debe al contrato, no a cifras inventadas ni a código
operativo perdido.

## Hallazgos consolidados P1

1. faltan gramáticas literales, cardinalidades y dominios completos de las
   siete tramas y del ticket entregado al Bash;
2. la precedencia no ordena PDEATHSIG, cambio de PPID, señal local, EOF parcial,
   terminalidad, control y plazos en cada estado;
3. no se declara qué cláusulas exactas de la decisión C4b-2 quedan sustituidas;
4. no se cierra la secuencia entre `Start`, pidfd doble, ticket, STOP, `/proc`,
   plazo, CONT y `ACK_CASO`;
5. la topología y propiedad de espera del lector binario Shell quedan abiertas;
6. se permite tocar D2d aunque la decisión C4b-2 exige su invariancia;
7. una postcondición desconocida no tiene representación canónica y nunca se
   puede fabricar como cero;
8. O3, O4, O5 y O6 siguen siendo tareas demasiado grandes y sin write-set,
   delta y criterio único por minitarea;
9. el runner y el adaptador aparecen como autoridades simultáneas de
   cuarentena; solo el runner puede decidir reutilización y caso siguiente;
10. el propietario y cardinalidad de `cmd.Wait` no están cerrados en todas las
    vías posteriores a `Start`;
11. O6 no reserva rutas físicas para matriz y oráculos y no puede corregir
    producción dentro del mismo corte que detecta un NO-GO.

## Corrección vinculante

La corrección debe reducir el alcance. G2-O0 solo puede autorizar el codec O1;
las fases operativas posteriores necesitan su propio ledger aprobado. Debe:

- incorporar la gramática completa sin heredar texto de G2-O, que permanece
  rechazado;
- declarar prevalencia exacta frente a C4b-2 y conservar expresamente D2d,
  capturador y adaptadores no autorizados byte a byte;
- fijar una precedencia total por estados, incluido EOF parcial como
  `PROTOCOLO`, y separar política de cuarentena de su materialización;
- prohibir un terminal normal si algún campo de postcondición es desconocido;
- dividir O3 en O3a/O3b/O3c, O4 en O4a/O4b/O4c, O5 en una decisión documental
  O5D y O5a/O5b/O5c, y O6 en O6a/O6b/O6c;
- exigir antes de cada corte rutas exactas, delta conservador, total previsto,
  tope de parada, modo 64 cerrado y revisión independiente;
- mantener el runner en 800 líneas sin minificación ni controles retirados.

## Pruebas y límites

Se ejecutaron inspección exacta de Git, `wc -l`, SHA-256 y
`git show --check`. No se ejecutaron Docker, PostgreSQL, red ni E2E porque la
revisión era exclusivamente documental.

Siguiente corte permitido: redactar una corrección G2-O0 reducida a contrato
canónico y ledger de O1. Sigue prohibido programar O1 hasta obtener doble GO
documental `P0=P1=P2=0` de esa corrección.
