package postgres

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestResolutorPerfilPersonalB11DeniegaSuperficieNoExteriorAntesDeSQL(t *testing.T) {
	r, err := nuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(
		&poolContextoActorDoble{}, bytesReaderB11{}, func() time.Time { return time.Time{} },
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.ResolverContextoActorGobernadoV1(context.Background(), domain.AutenticacionRevalidadaV1{})
	if !errors.Is(err, domain.ErrVinculoAutenticacionActorV2Invalido) {
		t.Fatalf("autenticacion/superficie no acreditada no se denego: %v", err)
	}
}

func TestResolutorPerfilPersonalB11DistingueRechazoDeIndisponibilidad(t *testing.T) {
	autenticacion, _ := autenticacionYFilaPerfilPersonalB11(t)
	casos := []struct {
		nombre       string
		errFila      error
		indisponible bool
	}{
		{nombre: "seleccion revocada", errFila: &pgconn.PgError{Code: "P0002", Message: "detalle privado"}},
		{nombre: "consulta no disponible", errFila: errors.New("detalle privado"), indisponible: true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{err: caso.errFila}}}
			r, err := nuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(
				&poolContextoActorDoble{transacciones: []*txContextoActorDoble{tx}},
				bytesReaderB11{}, func() time.Time { return autenticacion.SesionRevalidadaEn },
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = r.ResolverContextoActorGobernadoV1(context.Background(), autenticacion)
			if !errors.Is(err, domain.ErrVinculoAutenticacionActorV2Invalido) ||
				errors.Is(err, ports.ErrFuenteContextoActorNoDisponible) != caso.indisponible ||
				strings.Contains(err.Error(), "detalle privado") {
				t.Fatalf("clasificacion B11 incorrecta: %v", err)
			}
		})
	}
}

func TestResolutorPerfilPersonalB11ConfirmaRespuestaGobernada(t *testing.T) {
	autenticacion, fila := autenticacionYFilaPerfilPersonalB11(t)
	material := append(bytes.Repeat([]byte{0x61}, bytesAleatoriosReferenciaContextoActorV2), bytes.Repeat([]byte{0x62}, bytesAleatoriosReferenciaContextoActorV2)...)
	operacionEsperada := "oca_" + base64.RawURLEncoding.EncodeToString(material[:bytesAleatoriosReferenciaContextoActorV2])
	reciboEsperado := "rca_" + base64.RawURLEncoding.EncodeToString(material[bytesAleatoriosReferenciaContextoActorV2:])
	fila.valores[0], fila.valores[1] = operacionEsperada, reciboEsperado
	tx := &txContextoActorDoble{filas: []pgx.Row{fila}}
	r, err := nuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(
		&poolContextoActorDoble{transacciones: []*txContextoActorDoble{tx}}, bytes.NewReader(material),
		func() time.Time { return autenticacion.SesionRevalidadaEn },
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := r.ResolverContextoActorGobernadoV1(context.Background(), autenticacion)
	if err != nil || resultado.Validar() != nil {
		t.Fatalf("respuesta B11 gobernada rechazada: %v", err)
	}
	if resultado.RegistroContextoRef != reciboEsperado || tx.commits != 1 || len(tx.consultas) != 1 ||
		!strings.Contains(tx.consultas[0], "resolver_y_registrar_contexto_actor_perfil_personal_b11_v1") {
		t.Fatal("la respuesta B11 no se resolvió ni confirmó con su recibo exacto")
	}
	if got := tx.argumentos[0]; len(got) != 6 || got[0] != operacionEsperada || got[1] != reciboEsperado ||
		got[2] != autenticacion.CuentaRef || got[3] != string(autenticacion.MetodoObservado) || got[4] != string(autenticacion.GarantiaObservada) {
		t.Fatalf("la fachada SQL no recibió la identidad revalidada exacta: %#v", got)
	}
}

func TestResolutorPerfilPersonalB11ReconciliaCommitAmbiguoSinRegenerarIdentidad(t *testing.T) {
	autenticacion, fila := autenticacionYFilaPerfilPersonalB11(t)
	material := append(bytes.Repeat([]byte{0x63}, bytesAleatoriosReferenciaContextoActorV2), bytes.Repeat([]byte{0x64}, bytesAleatoriosReferenciaContextoActorV2)...)
	operacionEsperada := "oca_" + base64.RawURLEncoding.EncodeToString(material[:bytesAleatoriosReferenciaContextoActorV2])
	reciboEsperado := "rca_" + base64.RawURLEncoding.EncodeToString(material[bytesAleatoriosReferenciaContextoActorV2:])
	fila.valores[0], fila.valores[1] = operacionEsperada, reciboEsperado
	primera := &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: errors.New("commit ambiguo")}
	reconciliacion := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{valores: append([]any(nil), fila.valores...)}}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{primera, reconciliacion}}
	r, err := nuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(pool, bytes.NewReader(material), func() time.Time { return autenticacion.SesionRevalidadaEn })
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := r.ResolverContextoActorGobernadoV1(context.Background(), autenticacion)
	if err != nil || resultado.Validar() != nil || resultado.RegistroContextoRef != reciboEsperado {
		t.Fatalf("commit ambiguo B11 no se reconcilió exactamente: %v", err)
	}
	if pool.llamadas != 2 || len(primera.argumentos) != 1 || len(reconciliacion.argumentos) != 1 ||
		primera.argumentos[0][0] != reconciliacion.argumentos[0][0] || primera.argumentos[0][1] != reconciliacion.argumentos[0][1] ||
		primera.argumentos[0][0] != operacionEsperada || primera.argumentos[0][1] != reciboEsperado {
		t.Fatal("la reconciliación B11 regeneró operación o recibo")
	}
	if len(pool.opciones) != 2 || pool.opciones[0].IsoLevel != pgx.Serializable || pool.opciones[1].IsoLevel != pgx.ReadCommitted ||
		!strings.Contains(reconciliacion.consultas[0], "reconciliar_contexto_actor_perfil_personal_b11_v1") {
		t.Fatal("la reconciliación B11 no usó su frontera y aislamiento exactos")
	}
}

func TestResolutorPerfilPersonalB11CommitAmbiguoRespetaRevocacionDuranteReconciliacion(t *testing.T) {
	autenticacion, fila := autenticacionYFilaPerfilPersonalB11(t)
	primera := &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: errors.New("commit ambiguo")}
	reconciliacion := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{
		err: &pgconn.PgError{Code: "P0002", Message: "seleccion revocada"},
	}}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{primera, reconciliacion}}
	r, err := nuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(
		pool, bytesReaderB11{}, func() time.Time { return autenticacion.SesionRevalidadaEn },
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.ResolverContextoActorGobernadoV1(context.Background(), autenticacion)
	if !errors.Is(err, domain.ErrVinculoAutenticacionActorV2Invalido) ||
		errors.Is(err, ports.ErrFuenteContextoActorNoDisponible) || pool.llamadas != 2 {
		t.Fatalf("revocacion durante reconciliacion mal clasificada: %v", err)
	}
}

func autenticacionYFilaPerfilPersonalB11(t *testing.T) (domain.AutenticacionRevalidadaV1, filaContextoActorDoble) {
	t.Helper()
	_, fila := solicitudYFilaContextoActorV2(t)
	contexto, err := domain.RehidratarContextoActorVinculadoV2(fila.valores[2].([]byte))
	if err != nil {
		t.Fatal(err)
	}
	ahora := contexto.ResueltoEn.UTC().Truncate(time.Microsecond)
	return domain.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2, ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef: contexto.Instantanea.CuentaRef, CuentaOrdinariaRef: contexto.Instantanea.CuentaRef,
		Superficie: domain.SuperficieAutenticacionExternaPersonalV1, MetodoObservado: contexto.Principal.AuthMethod, GarantiaObservada: contexto.Principal.AuthAssurance,
		PoliticaGarantiaRef: "pga_0123456789abcdefghijkl", PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn: ahora.Add(-3 * time.Minute), SesionEmitidaEn: ahora.Add(-2 * time.Minute), SesionRevalidadaEn: ahora.Add(-time.Minute), SesionValidaHasta: ahora.Add(time.Minute),
	}, fila
}

type bytesReaderB11 struct{}

func (bytesReaderB11) Read(p []byte) (int, error) { return len(p), nil }
