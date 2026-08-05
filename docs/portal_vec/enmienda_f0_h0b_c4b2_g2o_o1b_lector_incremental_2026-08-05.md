# Enmienda F0-H0b/C4b-2 G2-O/O1b: lector incremental puro

Fecha: 5 de agosto de 2026.

Estado: **propuesta para doble revisión independiente; NO-GO para programar**.

Base exacta: `67331c695d217adeca9efd7142c612c3bc6652e6`.

O1a está integrada y publicada. Esta enmienda define únicamente O1b. No
autoriza O2, no activa `--supervisar-m38` y no modifica métricas.

## Responsabilidad única

O1b añade a G2 un lector incremental puro que transforma fragmentos de memoria
en tramas ya validadas por el codec O1a. Debe:

- admitir una trama fragmentada, incluso byte a byte;
- admitir varias tramas `CONTROL` coalescidas;
- producir como máximo una trama por transición;
- conservar el sobrante de forma exacta sin copiarlo ni ocultarlo;
- distinguir espera, trama, trama monoframa confirmada, EOF limpio y error;
- mantener memoria y tiempo acotados por el límite físico de la clase;
- quedar en estado terminal después de EOF o de cualquier error.

No abre, duplica o cierra descriptores; no crea procesos; no instala señales;
no usa pidfd, `prctl`, Bash, Docker, PostgreSQL, SQL, red o reloj; no asigna
causas de la máquina operativa. En particular, O1b nunca decide `CANCELADO` ni
`PROTOCOLO`: O2 hará esa traducción según el resultado físico.

## Clases y régimen

| Clase | Máximo incluido LF | Régimen del flujo |
| --- | ---: | --- |
| `CONTROL` | 1024 bytes | cero o más tramas hasta EOF |
| `SOBRE` | 4096 bytes | exactamente una trama y después EOF |
| `TERMINAL` | 1024 bytes | exactamente una trama y después EOF |
| `TICKET` | 2060 bytes | exactamente una trama y después EOF |

El constructor solo admite esas cuatro clases y obtiene el límite de
`limiteTramaM38`. No replica límites, gramática, dominios, causas, estados o
conversiones numéricas de O1a.

`SOBRE`, `TERMINAL` y `TICKET` son clases monoframa. Una trama monoframa
decodificada se retiene como candidata y no se entrega como válida hasta
observar EOF limpio. Cualquier byte posterior, incluso si llegó en el mismo
fragmento, invalida el lector antes de que exista un resultado utilizable.

`CONTROL` es multitrama. Una trama válida se entrega al alcanzar su LF. El
lector no examina el sobrante en esa transición: el llamador debe volver a
invocarlo antes de ejecutar una acción irreversible. Así
`INICIAR\nCANCELAR\n` conserva ambos mensajes y la futura máquina puede aplicar
la precedencia ya fijada sin ejecutar `Start` entre ambos.

## Propiedad de memoria y sobrante

La llamada recibe un fragmento `[]byte`, un indicador de EOF aplicable al final
de ese fragmento y devuelve, además del resultado, el número de bytes
consumidos.

- Con éxito, `fragmento[consumidos:]` es el sobrante exacto y continúa bajo
  propiedad del llamador.
- El contador solo es significativo si el error es nulo.
- El lector nunca conserva el fragmento completo ni el sobrante.
- Solo copia los bytes que formen una trama parcial.
- La trama parcial usa almacenamiento fijo de 4096 bytes; su longitud nunca
  supera `limite-1`.
- No se usa `append(buffer, fragmento...)`, `bufio.Scanner`, división global
  del fragmento ni reserva proporcional al tamaño aportado.
- Un fragmento arbitrariamente grande con LF temprano solo recorre el prefijo
  hasta ese LF. Sin LF, se detiene al alcanzar el límite, sin reservar memoria
  adicional.
- Modificar el fragmento original después de una transición no puede alterar
  la parcial retenida ni una trama ya producida.
- Los bytes internos procesados o fallidos se ponen a cero antes de abandonar
  el estado que los poseía.

No se fija un máximo artificial al tamaño del fragmento: el contador de
consumo y el almacenamiento fijo hacen el coste independiente de su cola. El
futuro dueño del descriptor fijará por separado el tamaño de cada lectura.

## Resultados y estados

El resultado usa un tipo enumerado interno; no se decide comparando mensajes
de error.

| Resultado | Trama | EOF | Significado |
| --- | --- | --- | --- |
| `NECESITA_DATOS` | no | no | no hay LF y la parcial sigue dentro del límite |
| `TRAMA` | sí | no | una trama `CONTROL` válida; puede quedar sobrante |
| `TRAMA_FINAL` | sí | sí | una monoframa válida confirmada por EOF limpio |
| `EOF_LIMPIO` | no | sí | fin en frontera de un flujo `CONTROL` |

Estados internos:

```text
L0 ABIERTO_VACIO
L1 ABIERTO_PARCIAL
L2 MONOTRAMA_ESPERANDO_EOF
L3 EOF_LIMPIO
L4 ERROR_TERMINAL
```

Invariantes:

- L0 tiene longitud parcial cero;
- L1 tiene `1..limite-1` bytes y ninguno es LF;
- L2 no retiene bytes crudos y conserva una única trama O1a válida;
- L3 y L4 son absorbentes y no producen otra trama;
- EOF repetido sin datos en L3 vuelve a informar `EOF_LIMPIO` sin cambiar
  estado;
- cualquier dato o llamada no terminal después de L3 falla como uso posterior
  a EOF;
- L4 devuelve el mismo error terminal en llamadas posteriores y no intenta
  resincronizarse;
- un error devuelve trama cero y contador cero.

Para una monoframa completa con EOF en la misma llamada, el resultado es
directamente `TRAMA_FINAL`. Si la llamada termina justo después del LF pero no
declara EOF, el lector pasa a L2 y devuelve `NECESITA_DATOS`; la trama solo se
entrega en una llamada posterior vacía con EOF.

En `CONTROL`, incluso si el fragmento termina en EOF justo después de una
trama, primero se devuelve `TRAMA`. El llamador vuelve a invocar con el
sobrante vacío y EOF para obtener `EOF_LIMPIO`. Esto garantiza una sola trama
por transición.

## Errores tipados y precedencia

O1b define centinelas internos diferenciados para:

1. clase o configuración inválida;
2. byte de flujo inválido;
3. exceso físico sin LF;
4. trama completa rechazada por O1a;
5. EOF con trama parcial;
6. datos posteriores a una monoframa;
7. uso posterior a EOF.

El error concreto de O1a puede envolverse conservando su identidad, pero
ninguna decisión depende de su texto. Todo error en consumo lleva a L4 y es
pegajoso.

Precedencia dentro de una transición:

1. L4 devuelve su error previo sin inspeccionar la entrada;
2. L3 solo admite una repetición vacía de EOF;
3. L2 confirma con EOF vacío y rechaza cualquier byte posterior;
4. para cada byte no LF se valida primero que pertenezca a `0x20..0x7e`;
   NUL, CR, TAB, controles, DEL y no ASCII son inválidos;
5. después se comprueba la capacidad y solo entonces se copia;
6. si ya existen `limite-1` bytes, cualquier byte no LF es exceso y no se
   copia;
7. LF completa una trama dentro del máximo y se entrega al codec O1a;
8. un fallo del codec prevalece sobre el EOF declarado al final del fragmento;
9. agotado el fragmento, EOF con parcial es EOF parcial; EOF sin parcial es
   limpio; sin EOF se informa `NECESITA_DATOS`.

En el mismo byte, el alfabeto inválido prevalece sobre el exceso físico porque
se comprueba antes de la capacidad. Ambos resultados siguen siendo fallo
terminal y O2 los tratará como protocolo inválido.

## Matriz mínima de autoprueba

La autoprueba O1b se integra en `autoprobarTramasM38`, sin tocar G1, y cubre:

### Fragmentación y coalescencia

- una trama válida de cada clase en un fragmento;
- todos los puntos de corte posibles de cada muestra válida;
- un byte por llamada y LF aislado;
- dos y tres controles coalescidos;
- `INICIAR` seguido de `CANCELAR`;
- primera trama completa y segunda parcial;
- parcial previa más dos tramas coalescidas;
- una sola entrega por transición, contador exacto y sobrante byte a byte.

### Monoframa y EOF

- monoframa completa sin EOF, seguida de EOF vacío;
- monoframa completa con EOF en la misma llamada;
- dos monoframas, monoframa más byte, más NUL y más LF;
- ausencia de resultado utilizable antes de EOF;
- `CONTROL` vacío más EOF;
- EOF después de una o varias tramas `CONTROL`;
- EOF con parcial de uno y varios fragmentos;
- EOF parcial clasificado de forma distinta al EOF limpio;
- EOF limpio repetido y datos posteriores rechazados;
- error pegajoso que no se recupera con una trama válida posterior.

### Límites y bytes hostiles

- `limite-1` bytes de cuerpo más LF: dentro del límite físico, aunque el codec
  rechace su gramática;
- `limite-1` bytes de parcial más otro byte no LF: exceso sin ampliar;
- frontera exacta válida de cada clase, incluida `SOBRE` 4096 y `TICKET` 2060;
- un byte por encima de cada límite;
- fragmento enorme con LF temprano y fragmento enorme sin LF;
- NUL, CR, TAB, `0x1f`, `0x7f` y `0x80`, también separados entre fragmentos;
- línea vacía, versión, clase, cardinalidad y dominio adversos mediante O1a;
- mutación del `[]byte` original tras alimentar una parcial;
- memoria interna a cero tras éxito, EOF y fallo.

### Aislamiento

- `--autoprueba` permanece verde;
- `--supervisar-m38` y modo desconocido permanecen en 64;
- no cambian descriptores ni hijos;
- no aparece operación real, red o dependencia nueva.

Las pruebas de límite deben distinguir el centinela físico del error de
gramática; no se aceptan fronteras vacuas basadas únicamente en que cualquier
error sea distinto de nulo.

## Ledger físico de O1b

Base comprobada:

| Alias | Unidad | Líneas | SHA-256 abreviado |
| --- | --- | ---: | --- |
| R | runner PostgreSQL | 800 | `5ce51623…f1153` |
| G1 | supervisor principal | 686 | `9fab2cae…e1afe` |
| G2 | supervisor operativo | 400 | `20980b27…ab04f5` |
| C | capturador | 799 | `4a967fd1…78902` |
| A | adaptador M38 | 527 | `98d22a30…a8cb7` |
| D2d | operaciones del runner | 145 | `9b137f13…92c5e81` |
| D2c | arnés de contexto | 588 | `a07057fb…badde5` |
| H0b | arnés sintético | 580 | `02a00f2f…bafded` |

Write-set de código:

1. G2: estados, errores, lector y autoprueba O1b;
2. R: sustitución in situ de los literales SHA-256 de G2 y del binario
   compuesto; cero líneas netas y ningún otro cambio.

G1, su literal SHA, C, A, D2d, D2c y H0b quedan byte a byte invariantes. El
manifiesto conserva exactamente seis fuentes y no se crea un séptimo fichero.

| Parte G2 | Delta conservador |
| --- | ---: |
| Estados, centinelas y enlace de autoprueba | +15..+25 |
| Lector productivo | +70..+100 |
| Matriz focal | +100..+140 |
| **Delta total** | **+185..+265** |

| Unidad | Base | Total previsto | Parada dura O1b |
| --- | ---: | ---: | ---: |
| R | 800 | 800 | 800 |
| G1 | 686 | 686 | 686 |
| G2 | 400 | 585..665 | 680 |
| C | 799 | 799 | 799 |
| A | 527 | 527 | 527 |
| D2d | 145 | 145 | 145 |
| D2c | 588 | 588 | 588 |
| H0b | 580 | 580 | 580 |

Se detiene y vuelve a revisión si G2 supera 680 líneas o +265, si R cambia
algo distinto de los dos literales, si cambia una unidad invariante, si se
minifica o retira una prueba para caber, si aparece un fichero nuevo o si el
modo operativo deja de devolver 64. El margen restante no se transfiere a O2.

## Puertas y aprobación

Antes de programar, dos revisores independientes deben obtener
`P0=P1=P2=0` sobre este contrato y su ledger.

El candidato posterior exige:

- `gofmt` y `go vet` de G1+G2;
- dos builds privados con `HOME`, `TMPDIR` y `GOCACHE` disjuntos y SHA binaria
  idéntica;
- SHA de fuentes estable antes y después de cada build;
- G1, su literal y todas las unidades declaradas invariantes;
- runner de 800 líneas con solo dos literales sustituidos;
- autoprueba positiva y matriz adversarial completa;
- Bash, ShellCheck, `git diff --check`, Gitleaks y puertas globales;
- al menos dos revisiones independientes posteriores al productor.

Docker, PostgreSQL, red y E2E del supervisor se omiten en O1b porque el lector
es puro y el modo operativo sigue cerrado. O1b no cierra C4b-2, C4b, H0b, C2,
F0 u O4-05. Las métricas permanecen F0 `10/23`, O4-05 `3/5`, Contratación
temporal `24/46`, Bolsa productiva `1/14` y producción `NO-GO`.
