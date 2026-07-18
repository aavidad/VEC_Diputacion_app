package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	variableDSNRegistro      = "VEC_POSTGRES_TEST_IDENTIDAD_REGISTRO_DSN"
	variableDSNRevalidacion  = "VEC_POSTGRES_TEST_IDENTIDAD_REVALIDACION_DSN"
	variableDSNProvisionador = "VEC_POSTGRES_TEST_IDENTIDAD_PROVISIONADOR_DSN"
	variableDSNMixto         = "VEC_POSTGRES_TEST_IDENTIDAD_MIXTO_DSN"
)

type seudonimizadorIntegracion struct{ resultado SeudonimosAlta }

func (s seudonimizadorIntegracion) SeudonimizarAlta(
	context.Context,
	IdentificadoresAlta,
) (SeudonimosAlta, error) {
	return s.resultado, nil
}

func TestIntegracionRegistroSesionesPostgreSQL18(t *testing.T) {
	dsnRegistro := os.Getenv(variableDSNRegistro)
	dsnRevalidacion := os.Getenv(variableDSNRevalidacion)
	dsnProvisionador := os.Getenv(variableDSNProvisionador)
	dsnMixto := os.Getenv(variableDSNMixto)
	if dsnRegistro == "" || dsnRevalidacion == "" ||
		dsnProvisionador == "" || dsnMixto == "" {
		t.Skipf(
			"defina %s, %s, %s y %s o ejecute deploy/postgresql/identidad_sesiones_v1/probar_integracion.sh",
			variableDSNRegistro,
			variableDSNRevalidacion,
			variableDSNProvisionador,
			variableDSNMixto,
		)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	poolRegistro := abrirPoolIntegracion(t, ctx, dsnRegistro)
	defer poolRegistro.Close()
	poolRevalidacion := abrirPoolIntegracion(t, ctx, dsnRevalidacion)
	defer poolRevalidacion.Close()
	poolProvisionador := abrirPoolIntegracion(t, ctx, dsnProvisionador)
	defer poolProvisionador.Close()
	poolMixto := abrirPoolIntegracion(t, ctx, dsnMixto)
	defer poolMixto.Close()
	comprobarIdentidadesIntegracion(
		t,
		ctx,
		poolRegistro,
		poolRevalidacion,
		poolProvisionador,
		poolMixto,
	)

	seudonimos := SeudonimosAlta{
		Esquema:          EsquemaHMACSHA256V1,
		EspacioIdentidad: espacioIdentidadPrueba,
		DominioRef:       dominioHMACPrueba, ClaveID: "clave-hsm-integracion", ClaveVersion: 7,
		AsercionIDHMAC: [32]byte{0xa1}, SesionIDHMAC: [32]byte{0xa2},
		SujetoIDHMAC: [32]byte{0xa3}, CuentaIDHMAC: [32]byte{0xa4},
	}
	adaptadorInvalido, err := NuevoRegistroSesionesPostgreSQL(
		ctx,
		poolProvisionador,
		poolRevalidacion,
		seudonimizadorIntegracion{resultado: seudonimos},
		espacioIdentidadPrueba,
		dominioHMACPrueba,
	)
	if adaptadorInvalido != nil ||
		!errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
		t.Fatal("el constructor acepto provisionamiento como capacidad de registro")
	}
	adaptadorInvalido, err = NuevoRegistroSesionesPostgreSQL(
		ctx,
		poolMixto,
		poolRevalidacion,
		seudonimizadorIntegracion{resultado: seudonimos},
		espacioIdentidadPrueba,
		dominioHMACPrueba,
	)
	if adaptadorInvalido != nil ||
		!errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
		t.Fatal("el constructor acepto un LOGIN con capacidades mezcladas")
	}
	var cuentaRef string
	err = poolProvisionador.QueryRow(ctx, `
		SELECT cuenta_ref
		FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
			$1,$2,$3,$4,$5,$6,$7,$8,$9
		)`,
		"opr_jjjjjjjjjjjjjjjjjjjjjjjj", seudonimos.Esquema,
		seudonimos.DominioRef, seudonimos.ClaveID, int64(seudonimos.ClaveVersion),
		seudonimos.CuentaIDHMAC[:], seudonimos.SujetoIDHMAC[:], false, nil,
	).Scan(&cuentaRef)
	if err != nil || !referenciaTecnicaValida(cuentaRef, "cta_") {
		t.Fatal("provisionar la cuenta sintetica mediante su capacidad")
	}

	adaptador, err := NuevoRegistroSesionesPostgreSQL(
		ctx,
		poolRegistro, poolRevalidacion,
		seudonimizadorIntegracion{resultado: seudonimos},
		espacioIdentidadPrueba, dominioHMACPrueba,
	)
	if err != nil {
		t.Fatal("componer el adaptador PostgreSQL")
	}
	emitida := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Second)
	alta := httpseguridad.AltaSesionAtomica{
		EspacioIdentidad: espacioIdentidadPrueba,
		AsercionID:       "asercion-integracion", SesionID: "sesion-integracion",
		SujetoID: "sujeto-integracion", CuentaID: "cuenta-integracion",
		Superficie:                httpseguridad.SuperficieInternaCorporativa,
		MetodoObservado:           domain.AuthMethodKerberos,
		GarantiaObservada:         domain.AuthAssuranceHigh,
		AutenticacionHuellaSHA256: strings.Repeat("a", 64),
		AutenticacionVerificadaEn: emitida.Add(-time.Second),
		SesionEmitidaEn:           emitida, AsercionExpiraEn: emitida.Add(4 * time.Minute),
		PoliticaGarantiaRef:          referencia("pga_", "i"),
		PoliticaGarantiaHuellaSHA256: strings.Repeat("b", 64),
	}
	confirmacion, err := adaptador.ConsumirAsercionYRegistrar(ctx, alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil ||
		confirmacion.CuentaRef != cuentaRef {
		t.Fatal("registrar y confirmar la sesion sintetica")
	}
	if confirmacion.SesionRevalidadaEn.Location() != time.UTC ||
		confirmacion.SesionValidaHasta.Location() != time.UTC ||
		confirmacion.SesionRevalidadaEn.Nanosecond()%1_000 != 0 ||
		confirmacion.SesionValidaHasta.Nanosecond()%1_000 != 0 {
		t.Fatal("pgx no se normalizo al contrato UTC de microsegundos")
	}

	consulta := httpseguridad.ConsultaSesionActiva{
		AutenticacionRef:          confirmacion.AutenticacionRef,
		AutenticacionHuellaSHA256: alta.AutenticacionHuellaSHA256,
		AsercionRef:               confirmacion.AsercionRef, SesionRef: confirmacion.SesionRef,
		CuentaRef: confirmacion.CuentaRef, CuentaOrdinariaRef: confirmacion.CuentaOrdinariaRef,
		CuentaPrivilegiada: alta.CuentaPrivilegiada, Superficie: alta.Superficie,
		MetodoObservado: alta.MetodoObservado, GarantiaObservada: alta.GarantiaObservada,
		PoliticaGarantiaRef:          alta.PoliticaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: alta.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn:    alta.AutenticacionVerificadaEn,
		SesionEmitidaEn:              alta.SesionEmitidaEn,
		ControlSesionRef:             confirmacion.ControlSesionRef,
		ControlSesionRevision:        confirmacion.ControlSesionRevision,
		ControlSesionEstado:          confirmacion.ControlSesionEstado,
		ControlSesionHuellaSHA256:    confirmacion.ControlSesionHuellaSHA256,
		SesionRevalidadaEn:           confirmacion.SesionRevalidadaEn,
		SesionValidaHasta:            confirmacion.SesionValidaHasta,
	}
	if err = adaptador.ComprobarSesionYCuentaActivas(ctx, consulta); err != nil {
		t.Fatal("revalidar la sesion por el pool separado")
	}

	// Dos invocaciones arrancan desde la misma barrera, sin temporizadores. La
	// unicidad durable de la asercion permite exactamente una confirmacion.
	seudonimosConcurrentes := seudonimos
	seudonimosConcurrentes.AsercionIDHMAC = [32]byte{0xb1}
	seudonimosConcurrentes.SesionIDHMAC = [32]byte{0xb2}
	adaptadorConcurrente, err := NuevoRegistroSesionesPostgreSQL(
		ctx,
		poolRegistro, poolRevalidacion,
		seudonimizadorIntegracion{resultado: seudonimosConcurrentes},
		espacioIdentidadPrueba, dominioHMACPrueba,
	)
	if err != nil {
		t.Fatal("componer adaptador concurrente")
	}
	altaConcurrente := alta
	altaConcurrente.AsercionID = "asercion-concurrente"
	altaConcurrente.SesionID = "sesion-concurrente"
	altaConcurrente.AutenticacionHuellaSHA256 = strings.Repeat("c", 64)
	altaConcurrente.SesionEmitidaEn = time.Now().UTC().Truncate(time.Microsecond).Add(-time.Second)
	altaConcurrente.AutenticacionVerificadaEn = altaConcurrente.SesionEmitidaEn.Add(-time.Second)
	altaConcurrente.AsercionExpiraEn = altaConcurrente.SesionEmitidaEn.Add(4 * time.Minute)
	inicio := make(chan struct{})
	resultados := make(chan error, 2)
	for indice := 0; indice < 2; indice++ {
		go func() {
			<-inicio
			_, errAlta := adaptadorConcurrente.ConsumirAsercionYRegistrar(
				ctx, altaConcurrente,
			)
			resultados <- errAlta
		}()
	}
	close(inicio)
	exitos, rechazos := 0, 0
	for indice := 0; indice < 2; indice++ {
		errAlta := <-resultados
		switch {
		case errAlta == nil:
			exitos++
		case errors.Is(errAlta, httpseguridad.ErrSesionNoValida):
			rechazos++
		default:
			t.Fatal("la carrera devolvio un error no saneado")
		}
	}
	if exitos != 1 || rechazos != 1 {
		t.Fatal("la asercion concurrente no se consumio exactamente una vez")
	}

	// Dos aserciones distintas del mismo identificador de sesion IdP tampoco
	// pueden abrir dos sesiones VEC solapadas.
	seudonimosSesionUno := seudonimos
	seudonimosSesionUno.AsercionIDHMAC = [32]byte{0xb3}
	seudonimosSesionUno.SesionIDHMAC = [32]byte{0xb5}
	seudonimosSesionDos := seudonimosSesionUno
	seudonimosSesionDos.AsercionIDHMAC = [32]byte{0xb4}
	adaptadoresSesion := make([]*RegistroSesionesPostgreSQL, 2)
	for indice, resultado := range []SeudonimosAlta{
		seudonimosSesionUno,
		seudonimosSesionDos,
	} {
		adaptadoresSesion[indice], err = NuevoRegistroSesionesPostgreSQL(
			ctx,
			poolRegistro,
			poolRevalidacion,
			seudonimizadorIntegracion{resultado: resultado},
			espacioIdentidadPrueba,
			dominioHMACPrueba,
		)
		if err != nil {
			t.Fatal("componer adaptador para exclusion concurrente de sesion")
		}
	}
	altasSesion := []httpseguridad.AltaSesionAtomica{alta, alta}
	for indice := range altasSesion {
		altasSesion[indice].AsercionID = "asercion-sesion-" + string(rune('a'+indice))
		altasSesion[indice].SesionID = "sesion-idp-compartida"
		altasSesion[indice].AutenticacionHuellaSHA256 = strings.Repeat(
			string(rune('d'+indice)),
			64,
		)
		altasSesion[indice].SesionEmitidaEn = time.Now().UTC().
			Truncate(time.Microsecond).Add(-time.Second)
		altasSesion[indice].AutenticacionVerificadaEn =
			altasSesion[indice].SesionEmitidaEn.Add(-time.Second)
		altasSesion[indice].AsercionExpiraEn =
			altasSesion[indice].SesionEmitidaEn.Add(4 * time.Minute)
	}
	inicioSesion := make(chan struct{})
	resultadosSesion := make(chan error, 2)
	for indice := range adaptadoresSesion {
		indice := indice
		go func() {
			<-inicioSesion
			_, errAlta := adaptadoresSesion[indice].ConsumirAsercionYRegistrar(
				ctx,
				altasSesion[indice],
			)
			resultadosSesion <- errAlta
		}()
	}
	close(inicioSesion)
	exitos, rechazos = 0, 0
	for indice := 0; indice < 2; indice++ {
		errAlta := <-resultadosSesion
		switch {
		case errAlta == nil:
			exitos++
		case errors.Is(errAlta, httpseguridad.ErrSesionNoValida):
			rechazos++
		default:
			t.Fatal("la exclusion de sesion devolvio un error no saneado")
		}
	}
	if exitos != 1 || rechazos != 1 {
		t.Fatal("dos aserciones abrieron sesiones VEC para la misma sesion IdP")
	}
}

func abrirPoolIntegracion(
	t *testing.T,
	ctx context.Context,
	dsn string,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("crear pool de integracion")
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal("conectar pool de integracion")
	}
	return pool
}

func comprobarIdentidadesIntegracion(
	t *testing.T,
	ctx context.Context,
	pools ...*pgxpool.Pool,
) {
	t.Helper()
	vistas := make(map[string]struct{}, len(pools))
	for _, pool := range pools {
		var sesion, efectiva string
		if err := pool.QueryRow(ctx, `SELECT session_user, current_user`).Scan(
			&sesion, &efectiva,
		); err != nil || sesion == "" || sesion != efectiva {
			t.Fatal("un pool suplanta su identidad mediante SET ROLE")
		}
		if _, repetida := vistas[sesion]; repetida {
			t.Fatal("dos capacidades comparten LOGIN")
		}
		vistas[sesion] = struct{}{}
	}
}
