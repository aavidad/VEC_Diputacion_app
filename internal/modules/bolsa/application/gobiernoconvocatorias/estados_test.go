package gobiernoconvocatorias

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var instanteEstadosPrueba = time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)

func TestConjuntoDeEstadosEsCerradoYExacto(t *testing.T) {
	validos := []EstadoDiario{
		EstadoDecisionVinculada,
		EstadoConfirmacionIniciada,
		EstadoIndeterminada,
		EstadoConfirmada,
		EstadoNoAplicada,
	}
	if len(validos) != 5 {
		t.Fatalf("conjunto de estados inesperado: %d", len(validos))
	}
	vistos := make(map[EstadoDiario]struct{}, len(validos))
	for _, estado := range validos {
		if !estado.Valido() {
			t.Errorf("estado declarado rechazado: %q", estado)
		}
		if _, repetido := vistos[estado]; repetido {
			t.Errorf("estado repetido: %q", estado)
		}
		vistos[estado] = struct{}{}
	}
	for _, invalido := range []EstadoDiario{"", "reservada", "confirmando", "CONFIRMADA", "no-aplicada"} {
		if invalido.Valido() {
			t.Errorf("estado ajeno aceptado: %q", invalido)
		}
	}
	if EstadoDecisionVinculada.Terminal() || EstadoConfirmacionIniciada.Terminal() ||
		EstadoIndeterminada.Terminal() || !EstadoConfirmada.Terminal() || !EstadoNoAplicada.Terminal() {
		t.Fatal("clasificacion terminal incorrecta")
	}
}

func TestMatrizDeTransicionesEsExplicitaYNoAdmiteRegresiones(t *testing.T) {
	type par struct{ origen, destino EstadoDiario }
	permitidas := map[par]DesenlaceTransicionDiario{
		{EstadoDecisionVinculada, EstadoConfirmacionIniciada}: DesenlaceTransicionAplicada,
		{EstadoDecisionVinculada, EstadoNoAplicada}:           DesenlaceTransicionAplicada,
		{EstadoConfirmacionIniciada, EstadoIndeterminada}:     DesenlaceTransicionAplicada,
		{EstadoConfirmacionIniciada, EstadoConfirmada}:        DesenlaceTransicionAplicada,
		{EstadoConfirmacionIniciada, EstadoNoAplicada}:        DesenlaceTransicionAplicada,
		{EstadoIndeterminada, EstadoIndeterminada}:            DesenlaceTransicionAplicada,
		{EstadoIndeterminada, EstadoConfirmada}:               DesenlaceTransicionAplicada,
		{EstadoIndeterminada, EstadoNoAplicada}:               DesenlaceTransicionAplicada,
		{EstadoConfirmada, EstadoConfirmada}:                  DesenlaceTransicionIdempotente,
		{EstadoNoAplicada, EstadoNoAplicada}:                  DesenlaceTransicionIdempotente,
	}
	estados := []EstadoDiario{
		EstadoDecisionVinculada, EstadoConfirmacionIniciada, EstadoIndeterminada,
		EstadoConfirmada, EstadoNoAplicada,
	}
	for _, origen := range estados {
		for _, destino := range estados {
			t.Run(string(origen)+"_a_"+string(destino), func(t *testing.T) {
				control := controlEnEstadoPrueba(t, origen)
				revisionAnterior := control.revision
				resultado, desenlace, err := control.Transicionar(
					destino, control.revision, control.cercado,
					instanteEstadosPrueba.Add(30*time.Second),
				)
				esperado, permitida := permitidas[par{origen, destino}]
				if !permitida {
					if !errors.Is(err, ErrTransicionDiarioProhibida) {
						t.Fatalf("transicion prohibida aceptada o error incorrecto: %v", err)
					}
					return
				}
				if err != nil || desenlace != esperado || resultado.estado != destino {
					t.Fatalf("transicion permitida rechazada: estado=%q desenlace=%q err=%v", resultado.estado, desenlace, err)
				}
				if esperado == DesenlaceTransicionIdempotente {
					if !resultado.revision.coincide(revisionAnterior) {
						t.Fatal("repeticion terminal incremento la revision")
					}
				} else if resultado.revision.valor != revisionAnterior.valor+1 {
					t.Fatal("transicion durable no incremento exactamente una revision")
				}
			})
		}
	}
}

func TestTerminalIdempotenteExigeRevisionYCercadoPeroNoLeaseVigente(t *testing.T) {
	control := controlEnEstadoPrueba(t, EstadoConfirmada)
	despuesDelVencimiento := instanteEstadosPrueba.Add(10 * time.Minute)
	recuperado, desenlace, err := control.Transicionar(
		EstadoConfirmada, control.revision, control.cercado, despuesDelVencimiento,
	)
	if err != nil || desenlace != DesenlaceTransicionIdempotente || recuperado.estado != EstadoConfirmada {
		t.Fatalf("terminal exacto no fue idempotente tras expirar lease: %v", err)
	}
	revisionObsoleta, _ := NuevaRevisionDiario(control.revision.valor - 1)
	if _, _, err := control.Transicionar(
		EstadoConfirmada, revisionObsoleta, control.cercado, despuesDelVencimiento,
	); !errors.Is(err, ErrConflictoRevisionDiario) {
		t.Fatalf("terminal acepto revision obsoleta: %v", err)
	}
	cercadoObsoleto, _ := NuevoCercadoDiario(control.cercado.valor - 1)
	if _, _, err := control.Transicionar(
		EstadoConfirmada, control.revision, cercadoObsoleto, despuesDelVencimiento,
	); !errors.Is(err, ErrCercadoDiarioObsoleto) {
		t.Fatalf("terminal acepto cercado obsoleto: %v", err)
	}
	if _, _, err := control.Transicionar(
		EstadoNoAplicada, control.revision, control.cercado,
		instanteEstadosPrueba.Add(30*time.Second),
	); !errors.Is(err, ErrTransicionDiarioProhibida) {
		t.Fatalf("terminal regreso a otro desenlace: %v", err)
	}
}

func TestRevisionCercadoYArrendamientoRechazanCerosYLimitesInvalidos(t *testing.T) {
	if _, err := NuevaRevisionDiario(0); !errors.Is(err, ErrRevisionDiarioInvalida) {
		t.Fatalf("revision cero aceptada: %v", err)
	}
	if _, err := NuevoCercadoDiario(0); !errors.Is(err, ErrCercadoDiarioInvalido) {
		t.Fatalf("cercado cero aceptado: %v", err)
	}
	revisionMaxima, _ := NuevaRevisionDiario(math.MaxUint64)
	if _, err := revisionMaxima.Siguiente(); !errors.Is(err, ErrRevisionDiarioInvalida) {
		t.Fatalf("desbordamiento de revision aceptado: %v", err)
	}
	cercadoMaximo, _ := NuevoCercadoDiario(math.MaxUint64)
	if _, err := cercadoMaximo.Siguiente(); !errors.Is(err, ErrCercadoDiarioInvalido) {
		t.Fatalf("desbordamiento de cercado aceptado: %v", err)
	}
	invalidos := [][2]time.Time{
		{{}, instanteEstadosPrueba},
		{instanteEstadosPrueba, instanteEstadosPrueba},
		{instanteEstadosPrueba, instanteEstadosPrueba.Add(-time.Microsecond)},
		{instanteEstadosPrueba, instanteEstadosPrueba.Add(DuracionMaximaArrendamientoDiario + time.Microsecond)},
		{instanteEstadosPrueba.Add(time.Nanosecond), instanteEstadosPrueba.Add(time.Minute)},
		{instanteEstadosPrueba.Local(), instanteEstadosPrueba.Add(time.Minute)},
	}
	for _, limites := range invalidos {
		if _, err := NuevoArrendamientoDiario(limites[0], limites[1]); !errors.Is(
			err, ErrArrendamientoDiarioInvalido,
		) {
			t.Errorf("arrendamiento invalido aceptado: %v -- %v: %v", limites[0], limites[1], err)
		}
	}
}

func TestIgualdadConVencimientoEstaExpiradaYPermiteNuevoCercado(t *testing.T) {
	inicio := instanteEstadosPrueba
	fin := inicio.Add(time.Minute)
	arrendamiento, _ := NuevoArrendamientoDiario(inicio, fin)
	if !arrendamiento.VigenteEn(inicio) || !arrendamiento.VigenteEn(fin.Add(-time.Microsecond)) ||
		arrendamiento.VigenteEn(fin) || arrendamiento.ExpiradoEn(fin.Add(-time.Microsecond)) ||
		!arrendamiento.ExpiradoEn(fin) {
		t.Fatal("frontera semicerrada de arrendamiento incorrecta")
	}
	control, _ := nuevoControlEstadoDiario(arrendamiento)
	nuevoArrendamiento, _ := NuevoArrendamientoDiario(fin, fin.Add(time.Minute))
	if _, err := control.ReclamarTrasExpiracion(
		control.revision, control.cercado, fin.Add(-time.Microsecond), nuevoArrendamiento,
	); !errors.Is(err, ErrArrendamientoDiarioInvalido) {
		t.Fatalf("reclamacion anterior al limite aceptada: %v", err)
	}
	reclamado, err := control.ReclamarTrasExpiracion(
		control.revision, control.cercado, fin, nuevoArrendamiento,
	)
	if err != nil || reclamado.revision.valor != control.revision.valor+1 ||
		reclamado.cercado.valor != control.cercado.valor+1 {
		t.Fatalf("reclamacion exacta no incremento revision y cercado: %v", err)
	}
	if _, _, err := reclamado.Transicionar(
		EstadoConfirmacionIniciada, reclamado.revision, control.cercado, fin.Add(time.Second),
	); !errors.Is(err, ErrCercadoDiarioObsoleto) {
		t.Fatalf("trabajador con cercado antiguo avanzo: %v", err)
	}
	if _, _, err := reclamado.Transicionar(
		EstadoConfirmacionIniciada, control.revision, reclamado.cercado, fin.Add(time.Second),
	); !errors.Is(err, ErrConflictoRevisionDiario) {
		t.Fatalf("trabajador con revision antigua avanzo: %v", err)
	}
}

func TestIndeterminadaPuedeRegistrarOtroIntentoDurable(t *testing.T) {
	control := controlEnEstadoPrueba(t, EstadoIndeterminada)
	revision := control.revision
	resultado, desenlace, err := control.Transicionar(
		EstadoIndeterminada, revision, control.cercado,
		instanteEstadosPrueba.Add(30*time.Second),
	)
	if err != nil || desenlace != DesenlaceTransicionAplicada ||
		resultado.revision.valor != revision.valor+1 {
		t.Fatalf("nuevo intento de reconciliacion no quedo durable: %v", err)
	}
}

func TestCienCarrerasSoloPermitenUnaTransicionCAS(t *testing.T) {
	control := nuevoControlEstadosPrueba(t)
	doble := &dobleControlEstadoDiario{control: control}
	revisionEsperada := control.revision
	cercadoEsperado := control.cercado
	const trabajadores = 100
	var espera sync.WaitGroup
	var aplicadas atomic.Int64
	var conflictos atomic.Int64
	errores := make(chan error, trabajadores)
	espera.Add(trabajadores)
	for indice := 0; indice < trabajadores; indice++ {
		go func() {
			defer espera.Done()
			desenlace, err := doble.transicionar(
				EstadoConfirmacionIniciada, revisionEsperada, cercadoEsperado,
				instanteEstadosPrueba.Add(time.Second),
			)
			switch {
			case err == nil && desenlace == DesenlaceTransicionAplicada:
				aplicadas.Add(1)
			case errors.Is(err, ErrConflictoRevisionDiario):
				conflictos.Add(1)
			default:
				errores <- err
			}
		}()
	}
	espera.Wait()
	close(errores)
	for err := range errores {
		t.Errorf("desenlace inesperado en carrera: %v", err)
	}
	if aplicadas.Load() != 1 || conflictos.Load() != trabajadores-1 {
		t.Fatalf("CAS no eligio un unico ganador: aplicadas=%d conflictos=%d", aplicadas.Load(), conflictos.Load())
	}
	if doble.estado() != EstadoConfirmacionIniciada {
		t.Fatalf("estado final inesperado: %q", doble.estado())
	}
}

type dobleControlEstadoDiario struct {
	mu      sync.Mutex
	control ControlEstadoDiario
}

func (d *dobleControlEstadoDiario) transicionar(
	destino EstadoDiario,
	revision RevisionDiario,
	cercado CercadoDiario,
	instante time.Time,
) (DesenlaceTransicionDiario, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	siguiente, desenlace, err := d.control.Transicionar(destino, revision, cercado, instante)
	if err == nil {
		d.control = siguiente
	}
	return desenlace, err
}

func (d *dobleControlEstadoDiario) estado() EstadoDiario {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.control.estado
}

func nuevoControlEstadosPrueba(t *testing.T) ControlEstadoDiario {
	t.Helper()
	arrendamiento, err := NuevoArrendamientoDiario(
		instanteEstadosPrueba, instanteEstadosPrueba.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	control, err := nuevoControlEstadoDiario(arrendamiento)
	if err != nil {
		t.Fatal(err)
	}
	if control.revision.valor != 1 || control.cercado.valor != 1 ||
		control.estado != EstadoDecisionVinculada {
		t.Fatal("el constructor inicial invento historia")
	}
	return control
}

func controlEnEstadoPrueba(t *testing.T, estado EstadoDiario) ControlEstadoDiario {
	t.Helper()
	control := nuevoControlEstadosPrueba(t)
	transicionar := func(destino EstadoDiario, segundo int) {
		var err error
		control, _, err = control.Transicionar(
			destino, control.revision, control.cercado,
			instanteEstadosPrueba.Add(time.Duration(segundo)*time.Second),
		)
		if err != nil {
			t.Fatalf("preparar estado %q: %v", destino, err)
		}
	}
	switch estado {
	case EstadoDecisionVinculada:
	case EstadoConfirmacionIniciada:
		transicionar(EstadoConfirmacionIniciada, 1)
	case EstadoIndeterminada:
		transicionar(EstadoConfirmacionIniciada, 1)
		transicionar(EstadoIndeterminada, 2)
	case EstadoConfirmada:
		transicionar(EstadoConfirmacionIniciada, 1)
		transicionar(EstadoConfirmada, 2)
	case EstadoNoAplicada:
		transicionar(EstadoNoAplicada, 1)
	default:
		t.Fatalf("estado de prueba invalido: %q", estado)
	}
	return control
}
