package postgres

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type transaccionAcreditacionCaptura struct {
	consultas  []string
	argumentos [][]any
	filas      []pgx.Row
}

func (t *transaccionAcreditacionCaptura) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consultas = append(t.consultas, consulta)
	t.argumentos = append(t.argumentos, append([]any(nil), argumentos...))
	indice := len(t.consultas) - 1
	if indice >= len(t.filas) {
		return filaAcreditacionPrueba{err: errors.New("fila no configurada")}
	}
	return t.filas[indice]
}

type filaAcreditacionPrueba struct {
	instante *time.Time
	err      error
}

func (f filaAcreditacionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("destino inesperado")
	}
	destino, correcto := destinos[0].(*pgtype.Timestamptz)
	if !correcto {
		return errors.New("tipo inesperado")
	}
	if f.instante == nil {
		*destino = pgtype.Timestamptz{Valid: false}
		return nil
	}
	*destino = pgtype.Timestamptz{Time: *f.instante, Valid: true}
	return nil
}

func TestAcreditadorPostgreSQLV2UsaMismaTransaccionDosVecesYArgumentosExactos(t *testing.T) {
	t.Parallel()
	orden, resultado, emitida, validaHasta := ordenAcreditacionPostgreSQLV2Prueba(t, math.MaxUint64)
	primera := emitida.Add(10 * time.Microsecond)
	segunda := emitida.Add(20 * time.Microsecond)
	transaccion := &transaccionAcreditacionCaptura{filas: []pgx.Row{
		filaAcreditacionPrueba{instante: &primera}, filaAcreditacionPrueba{instante: &segunda},
	}}
	acreditador, err := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(transaccion)
	if err != nil {
		t.Fatalf("crear acreditador: %v", err)
	}
	obtenidaPrimera, err := acreditador.AcreditarUsoRegistroContextoActorV2(context.Background(), orden)
	if err != nil || !obtenidaPrimera.Equal(primera) {
		t.Fatalf("primera acreditacion: %s, %v", obtenidaPrimera, err)
	}
	obtenidaSegunda, err := acreditador.AcreditarUsoRegistroContextoActorV2(context.Background(), orden)
	if err != nil || !obtenidaSegunda.Equal(segunda) {
		t.Fatalf("segunda acreditacion: %s, %v", obtenidaSegunda, err)
	}
	if len(transaccion.argumentos) != 2 || len(transaccion.argumentos[0]) != 17 ||
		!strings.Contains(transaccion.consultas[0], "vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2") {
		t.Fatalf("consulta inesperada: %#v %#v", transaccion.consultas, transaccion.argumentos)
	}
	actor := resultado.Contexto
	esperados := []any{
		resultado.RegistroContextoRef,
		domain.EsquemaRepresentacionContextoActorV2,
		resultado.HuellaSHA256,
		resultado.ManifiestoProcedenciaHuellaSHA256,
		string(resultado.AutoridadEfectiva),
		actor.Instantanea.CuentaRef,
		"18446744073709551615",
		actor.PersonaRef,
		"3",
		actor.PerfilActivoRef,
		"4",
		actor.Instantanea.VinculoRef,
		"5",
		string(actor.Principal.AuthMethod),
		string(actor.Principal.AuthAssurance),
		emitida,
		validaHasta,
	}
	for indice, esperado := range esperados {
		if transaccion.argumentos[0][indice] != esperado {
			t.Fatalf("argumento %d: obtenido=%#v esperado=%#v", indice+1, transaccion.argumentos[0][indice], esperado)
		}
	}
}

func TestAcreditadorPostgreSQLV2DistingueNULLFalloYCancelacionSinFiltrar(t *testing.T) {
	t.Parallel()
	orden, _, emitida, _ := ordenAcreditacionPostgreSQLV2Prueba(t, 7)

	t.Run("null_deniega", func(t *testing.T) {
		transaccion := &transaccionAcreditacionCaptura{filas: []pgx.Row{filaAcreditacionPrueba{}}}
		acreditador, _ := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(transaccion)
		_, err := acreditador.AcreditarUsoRegistroContextoActorV2(context.Background(), orden)
		if !errors.Is(err, ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada) ||
			errors.Is(err, ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible) {
			t.Fatalf("NULL no fue denegacion limpia: %v", err)
		}
	})

	t.Run("fallo_saneado", func(t *testing.T) {
		transaccion := &transaccionAcreditacionCaptura{filas: []pgx.Row{filaAcreditacionPrueba{
			err: errors.New("postgresql://usuario:secreto@servidor SELECT dni"),
		}}}
		acreditador, _ := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(transaccion)
		_, err := acreditador.AcreditarUsoRegistroContextoActorV2(context.Background(), orden)
		if !errors.Is(err, ports.ErrAcreditacionUsoRegistroContextoActorV2Denegada) ||
			!errors.Is(err, ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible) ||
			strings.Contains(err.Error(), "secreto") || strings.Contains(err.Error(), "dni") {
			t.Fatalf("fallo no saneado: %v", err)
		}
	})

	t.Run("cancelacion_previa", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		transaccion := &transaccionAcreditacionCaptura{}
		acreditador, _ := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(transaccion)
		_, err := acreditador.AcreditarUsoRegistroContextoActorV2(ctx, orden)
		if !errors.Is(err, context.Canceled) || len(transaccion.consultas) != 0 {
			t.Fatalf("cancelacion: %v, consultas=%d", err, len(transaccion.consultas))
		}
	})

	t.Run("instante_fuera_de_ventana", func(t *testing.T) {
		fuera := emitida.Add(-time.Microsecond)
		transaccion := &transaccionAcreditacionCaptura{filas: []pgx.Row{filaAcreditacionPrueba{instante: &fuera}}}
		acreditador, _ := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(transaccion)
		_, err := acreditador.AcreditarUsoRegistroContextoActorV2(context.Background(), orden)
		if !errors.Is(err, ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible) {
			t.Fatalf("salida imposible aceptada: %v", err)
		}
	})
}

func TestNuevoAcreditadorPostgreSQLV2RechazaTransaccionNulaYTipada(t *testing.T) {
	t.Parallel()
	if _, err := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(nil); !errors.Is(
		err, ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible,
	) {
		t.Fatalf("nil: %v", err)
	}
	var tipada *transaccionAcreditacionCaptura
	if _, err := nuevoAcreditadorUsoRegistroContextoActorPostgreSQLV2EnTransaccion(tipada); !errors.Is(
		err, ports.ErrAcreditadorUsoRegistroContextoActorV2NoDisponible,
	) {
		t.Fatalf("nil tipada: %v", err)
	}
}

func ordenAcreditacionPostgreSQLV2Prueba(
	t *testing.T,
	cuentaVersion uint64,
) (ports.OrdenAcreditacionUsoRegistroContextoActorV2, domain.ResultadoContextoActorRegistradoV2, time.Time, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 7, 22, 10, 0, 0, 123_456_000, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: cuentaVersion,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(30 * time.Minute),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("a", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{CuentaRef: instantanea.CuentaRef, Version: cuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Persona: domain.ProcedenciaPersonaContextoActorV1{PersonaRef: instantanea.PersonaRef, Version: 3,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{PerfilRef: instantanea.PerfilActivoRef, Version: 4,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{VinculoRef: instantanea.VinculoRef, Version: 5,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Vinculos: []domain.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanonico, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(manifiestoCanonico)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanonico,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	validaHasta := ahora.Add(time.Minute)
	orden, err := ports.NuevaOrdenAcreditacionUsoRegistroContextoActorV2(resultado, ahora, validaHasta)
	if err != nil {
		t.Fatalf("crear orden: %v", err)
	}
	return orden, resultado, ahora, validaHasta
}
