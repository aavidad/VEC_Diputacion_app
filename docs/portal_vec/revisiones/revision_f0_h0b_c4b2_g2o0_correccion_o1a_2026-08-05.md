# Revisión independiente G2-O0, corrección O1a

Fecha: 5 de agosto de 2026.

Commit revisado: `bc2e583`.

Estado: **NO-GO documental**. No se modificó código ni se autoriza O1a.

| Revisión | P0 | P1 | P2 | Veredicto |
| --- | ---: | ---: | ---: | --- |
| Seguridad y topología | 0 | 4 | 0 | NO-GO |
| Ledger y write-set | 0 | 1 | 2 | NO-GO |

## Hallazgos P1

1. al retirar `ACK_LISTO` falta el predicado Shell equivalente que acredita al
   supervisor antes de enviar `INICIAR`;
2. EOF del canal de vida no basta para llamar `wait -f`: Go no puede cerrar el
   escritor de forma anticipada y el trabajo debe ser inmediatamente
   recolectable;
3. el lector retenido y el escritor del FD 8 deben proceder de aperturas
   independientes, no de `dup`, para no compartir offset; también falta fijar
   su herencia y escritor único;
4. el binario privado debe reacreditarse antes del supervisor, antes del
   validador y después de este para impedir sustitución entre ejecuciones;
5. el runner cambia tres literales: SHA de G1, G2 y del binario compilado; el
   write-set solo había declarado los dos primeros.

## Hallazgos P2

- faltan D2c y H0b en el ledger de invariantes;
- el máximo de 220 líneas G2 necesita separar codec y autoprueba dentro del
  mismo corte y justificar su cohesión.

## Aspectos conformes

- O1a tiene una única responsabilidad y deja O1b y el resto sin autorizar;
- topología supervisor/validador secuencial, descarte de Bash para entrada
  binaria, autoridad única del runner y D2d inmutable son correctos;
- framing, precedencia, terminal desconocido y aritmética de los elementos
  medidos son coherentes;
- `git show --check bc2e583` es verde.

## Corrección mínima

La siguiente propuesta debe añadir los cuatro predicados de seguridad,
autorizar exactamente los tres reemplazos SHA del runner, incorporar D2c/H0b
y descomponer el delta G2 entre codec y autoprueba. Después requiere otra doble
revisión independiente antes de programar.
