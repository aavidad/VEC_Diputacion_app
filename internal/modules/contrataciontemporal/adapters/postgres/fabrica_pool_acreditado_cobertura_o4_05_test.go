package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestFabricaPoolO405ConservaDefaultsRealesSinCallbacksInyectables(
	t *testing.T,
) {
	configuracionPGX, err := pgxpool.ParseConfig(
		"host=localhost user=vec_o405 dbname=postgres " +
			"password='' sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}
	if configuracionPGX.ConnConfig.DialFunc == nil ||
		configuracionPGX.ConnConfig.LookupFunc == nil ||
		configuracionPGX.ConnConfig.BuildFrontend == nil ||
		configuracionPGX.ConnConfig.BuildContextWatcherHandler == nil {
		t.Fatal("pgx no expuso sus cuatro defaults reales")
	}

	dependencia, err := nuevoPoolRecuperacionCoberturaO405PostgreSQL(
		context.Background(),
		"host=db-o405.example,replica-o405.example "+
			"port=5432,5432 user=vec_o405 dbname=postgres "+
			"password='' sslmode=verify-full pool_min_conns=0",
		modoTLSAcreditacionPoolO405Produccion,
	)
	if err != nil {
		t.Fatalf("fábrica cerrada rechazó defaults pgx: %v", err)
	}
	cierreReal := dependencia.cierre.cerrar
	cierres := 0
	dependencia.cierre.cerrar = func() {
		cierres++
		cierreReal()
	}
	defer dependencia.Cerrar()
	configuracion := dependencia.pool.Config()
	if configuracion.ConnConfig.DialFunc == nil ||
		configuracion.ConnConfig.LookupFunc == nil ||
		configuracion.ConnConfig.BuildFrontend == nil ||
		configuracion.ConnConfig.BuildContextWatcherHandler == nil ||
		len(configuracion.ConnConfig.Fallbacks) == 0 {
		t.Fatal("fábrica no conservó defaults o fallbacks reales")
	}
	configuracion.ConnConfig.DialFunc = nil
	configuracion.ConnConfig.LookupFunc = nil
	configuracion.ConnConfig.BuildFrontend = nil
	configuracion.ConnConfig.BuildContextWatcherHandler = nil
	efectiva := dependencia.pool.Config()
	if efectiva.ConnConfig.DialFunc == nil ||
		efectiva.ConnConfig.LookupFunc == nil ||
		efectiva.ConnConfig.BuildFrontend == nil ||
		efectiva.ConnConfig.BuildContextWatcherHandler == nil {
		t.Fatal("copia de configuración alteró pool acreditado")
	}
	for _, alternativa := range efectiva.ConnConfig.Fallbacks {
		if alternativa == nil ||
			!tlsAcreditacionPoolO405VerificaIdentidad(
				alternativa.TLSConfig,
				alternativa.Host,
			) {
			t.Fatal("fallback no quedó bajo configuración compartida acreditada")
		}
	}
	copia := *dependencia
	if ejecutor, err :=
		NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			context.Background(),
			&copia,
		); ejecutor != nil || !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("copia de pool aceptada: ejecutor=%v err=%v", ejecutor, err)
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if ejecutor, err :=
		NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			ctxCancelado,
			dependencia,
		); ejecutor != nil || !errors.Is(err, context.Canceled) ||
		cierres != 0 {
		t.Fatalf(
			"constructor tomó ownership: ejecutor=%v cierres=%d err=%v",
			ejecutor,
			cierres,
			err,
		)
	}
}

func TestPoolO405OwnershipExplicitoRechazaNuloYCopiaYCierraUnaVez(
	t *testing.T,
) {
	var nulo *PoolRecuperacionCoberturaO405PostgreSQL
	nulo.Cerrar()
	if ejecutor, err :=
		NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			context.Background(),
			nulo,
		); ejecutor != nil || !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("dependencia nula aceptada: ejecutor=%v err=%v", ejecutor, err)
	}

	cierres := 0
	original := &PoolRecuperacionCoberturaO405PostgreSQL{
		cierre: &cierrePoolRecuperacionCoberturaO405{
			cerrar: func() { cierres++ },
		},
	}
	original.sello = &selloFabricaPoolO405{
		dependencia:              original,
		modo:                     modoTLSAcreditacionPoolO405Produccion,
		callbacksPredeterminados: true,
	}
	copia := *original
	copia.Cerrar()
	if cierres != 0 ||
		selloAcreditacionO405Valido(
			copia.sello,
			modoTLSAcreditacionPoolO405Produccion,
		) &&
			copia.sello.dependencia == &copia {
		t.Fatal("copia obtuvo ownership o sello propio")
	}
	original.Cerrar()
	original.Cerrar()
	if cierres != 1 {
		t.Fatalf("cierres=%d; quiere 1", cierres)
	}

	cero := &PoolRecuperacionCoberturaO405PostgreSQL{}
	cero.Cerrar()
	if selloAcreditacionO405Valido(
		cero.sello,
		modoTLSAcreditacionPoolO405Produccion,
	) {
		t.Fatal("wrapper cero acreditado")
	}
}

func TestFabricaPoolO405RechazaCadenaYContextoSinRetenerRecurso(
	t *testing.T,
) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if pool, err := NuevoPoolRecuperacionCoberturaO405PostgreSQL(
		ctx,
		"",
	); pool != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado creó pool: pool=%v err=%v", pool, err)
	}
	if pool, err := NuevoPoolRecuperacionCoberturaO405PostgreSQL(
		context.Background(),
		"sslmode=modo_invalido",
	); pool != nil || err == nil {
		t.Fatalf("cadena inválida creó pool: pool=%v err=%v", pool, err)
	}
}
