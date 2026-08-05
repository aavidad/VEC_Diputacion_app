# Revisión F0-H0b/C4b-2 G2-O/O2b: contrato de control previo

Fecha: 5 de agosto de 2026.

Estado: **doble `GO` documental final; `P0=0`, `P1=0`, `P2=0`**.

Documento revisado:

`docs/portal_vec/enmienda_f0_h0b_c4b2_g2o_o2b_control_preinicio_2026-08-05.md`.

Huella del contenido contractual que recibió los dos dictámenes finales:

```text
a120ad7c7e0471a57427acb2699d11cbac0cc08afb6e6971a397f9ee998d997c
```

Huella después de actualizar únicamente su estado de propuesta a doble `GO`:

```text
e30d5fae9bdce22b9e350cc5fffc1508899e7c5648dbe4d96e9f9ee07f490ad4
```

La segunda huella no cambia alcance, API, conducta, ledger, pruebas o paradas.
La única diferencia es el párrafo de estado inicial. Antes de confirmar, los
dos revisores deben comprobar esa diferencia focal y la coherencia de esta
acta.

## Base contrastada

- código material O2a: `0caa140409db5ac0a2f1312f13e002b702691e1b`;
- corte publicado O2a: `ef1f08baacc891d60b3ecb422510921294691f8b`;
- CI material O2a: `31021785711`, cinco de cinco puertas verdes;
- relevo documental: `fd8c2d8e1c39a315136dff5f49cc5904002887e9`;
- CI del relevo: `31023302012`, cinco de cinco puertas verdes.

Se contrastaron el runner material, G1, G2, G3, la corrección canónica O1a,
el contrato y la evidencia O2a, y la decisión C4b-2. No se ejecutó Docker para
la revisión documental y no se modificó código.

## Revisiones independientes

### Revisión funcional y de seguridad

La primera vuelta emitió `NO-GO`, `P0=0`, `P1=7`, `P2=2`. Detectó:

1. prevalencia incompleta frente a hilo, señales y `prctl` de C4b-2;
2. promesa de drenar también después de S5;
3. normalización como protocolo de errores O1b físicamente imposibles;
4. falta de matriz exacta O1b para evitar contador cero y bucles;
5. invalidación sin estado final completo;
6. contradicción entre no leer campos y exigir su presencia;
7. AST que omitía las transiciones terminales S1/S2/S3 -> S5;
8. ausencia de límite computacional explícito del drenaje;
9. clasificación demasiado categórica del nonce como no sensible.

Tras corregir los nueve puntos, la reauditoría encontró solo una ambigüedad
editorial P2 entre «solamente S1 -> S2 y S2 -> S3» y las entradas S5. La
redacción final distingue transiciones no terminales y terminales, y fija que
el drenaje termina al agotar entrada o entrar en S5.

Dictamen final sobre `a120ad7c...`:

```text
GO
P0=0
P1=0
P2=0
```

### Revisión de ledger, integración y evidencia

La primera vuelta emitió `NO-GO` y detectó:

1. una transcripción inicial incorrecta del SHA completo `0caa140`, corregida
   antes del cierre;
2. orden incorrectamente implícito del manifiesto, que el capturador ordena;
3. contradicción de presencia frente a lectura de campos;
4. ausencia de tuplas externas exactas;
5. prohibición AST incompatible con referencias tipadas a O2a/S0;
6. una rama imposible de probar sin mutación del constructor `CONTROL`;
7. marcador `<sha7>` no definido;
8. prohibición de toda comparación del ticket incompatible con `ticket != ""`.

La revisión comprobó materialmente que:

- runner `794 -> 799` cabe con exactamente cinco líneas legibles;
- G1 `689 -> 692` cabe con exactamente tres líneas;
- G2 permanece en 798 y G3 en 431, invariantes byte a byte;
- el manifiesto puede pasar de siete a ocho con G4 en índice ordenado 5;
- captura, cardinalidad, variables, vet y builds admiten G4 mediante edición
  legible de líneas existentes;
- el baseline positivo G1--G4 evita falsos mutantes muertos;
- la evidencia histórica O2a queda ligada correctamente a `0caa140`.

Dictamen final y contrarrevisión sobre `a120ad7c...`:

```text
GO
P0=0
P1=0
P2=0
```

## Contrato final acreditado

O2b queda programable, después de publicar este corte y obtener CI verde,
únicamente con esta frontera:

```text
O2a confirmado en S1
-> lector CONTROL puro
-> ARMAR coherente en S2
-> INICIAR reconocido en S3, todavía sin Start
o
-> S5 anterior a Bash, con causa inmutable y referencias sensibles retiradas
```

La enmienda fija además:

- propiedad exclusiva y no concurrente;
- copia fija única del nonce potencialmente correlacionable;
- ausencia de copia, exposición o uso funcional del ticket e identidad;
- EOF, coalescencia, parciales, contadores y tuplas exactos;
- como máximo tres tramas y 3072 bytes de prefijo por llamada;
- solo cuatro errores O1b normalizables a protocolo;
- fallos internos pegajosos y causa funcional separada;
- G4 pura, sin FD, proceso, Bash, señal, reloj, red o terminal;
- 25 mutantes, AST reproducible y baseline positivo obligatorio;
- write-set limitado a runner, G1 y G4;
- paradas físicas y funcionales antes de confirmar.

## Autorización condicionada

Este `GO` autoriza crear un candidato solo cuando:

1. contrato y acta estén confirmados en un único corte documental;
2. el corte esté publicado en la rama integradora;
3. su CI termine con cinco de cinco puertas verdes;
4. el productor registre el SHA completo del padre y todas las huellas base
   antes de editar;
5. use una rama y worktree aislados con el write-set exacto.

El productor no integra ni publica su código. El candidato necesita dos
revisiones de implementación independientes, `P0=P1=P2=0`, antes de llegar a
la integradora.

O2b no cierra G2-O, C4b-2, H0b, F0, O4-05 ni producción. Las métricas siguen
en F0 `10/23`, O4-05 `3/5`, Contratación temporal `24/46` y Bolsa productiva
`1/14`; producción permanece `NO-GO`.
