package postgres

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func acreditarPoolRecuperacionCoberturaO405(
	ctx context.Context,
	origen origenAcreditacionPoolO405,
	modo modoTLSAcreditacionPoolO405,
) error {
	_, err := acreditarPoolRecuperacionCoberturaO405ConManifiesto(
		ctx,
		origen,
		modo,
	)
	return err
}

func acreditarPoolRecuperacionCoberturaO405ConManifiesto(
	ctx context.Context,
	origen origenAcreditacionPoolO405,
	modo modoTLSAcreditacionPoolO405,
) (oidResultado uint32, errResultado error) {
	var conexion conexionAcreditacionPoolO405
	liberar := false
	defer func() {
		panico := recover()
		falloLiberacion := liberar &&
			liberarConexionAcreditacionO405(conexion)
		if panico != nil || falloLiberacion {
			oidResultado = 0
			errResultado =
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
		}
	}()
	if dependenciaNula(ctx) || dependenciaNula(origen) {
		return 0,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	selloEsperado := origen.Sello()
	if !selloAcreditacionO405Valido(selloEsperado, modo) ||
		!configuracionPoolAcreditacionO405Valida(
			origen.Configuracion(),
			modo,
		) {
		return 0,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	conexion, err := origen.Adquirir(ctx)
	if !dependenciaNula(conexion) {
		liberar = true
	}
	if err != nil || dependenciaNula(conexion) {
		return 0, errorAcreditacionPoolO405(ctx)
	}
	if conexion.Sello() != selloEsperado {
		return 0,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	return acreditarConexionRecuperacionCoberturaO405ConManifiesto(
		ctx,
		conexion,
		modo,
		0,
	)
}

func acreditarConexionRecuperacionCoberturaO405(
	ctx context.Context,
	conexion conexionAcreditacionPoolO405,
	modo modoTLSAcreditacionPoolO405,
) error {
	_, err := acreditarConexionRecuperacionCoberturaO405ConManifiesto(
		ctx,
		conexion,
		modo,
		0,
	)
	return err
}

func acreditarConexionRecuperacionCoberturaO405ConManifiesto(
	ctx context.Context,
	conexion conexionAcreditacionPoolO405,
	modo modoTLSAcreditacionPoolO405,
	oidEsperado uint32,
) (oidResultado uint32, errResultado error) {
	defer func() {
		if recover() != nil {
			oidResultado = 0
			errResultado =
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
		}
	}()
	if dependenciaNula(ctx) || dependenciaNula(conexion) ||
		!selloAcreditacionO405Valido(conexion.Sello(), modo) ||
		!configuracionConexionAcreditacionO405Valida(
			conexion.Configuracion(),
			modo,
		) {
		return 0,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var (
		oidFuncion                  uint32
		usuarioSesion               string
		usuarioEfectivo             string
		tlsActivo                   bool
		primaria                    bool
		loginSeguro                 bool
		grupoSeguro                 bool
		membresiaDirectaExacta      bool
		membresiaTotalExacta        bool
		loginSinAutoridad           bool
		grupoSinAutoridad           bool
		privilegiosEfectivosExactos bool
		funcionExacta               bool
		propietarioExacto           bool
		seguridadFuncionExacta      bool
		configuracionFuncionExacta  bool
		firmaRetornoFuncionExactos  bool
		lenguajeProbinExactos       bool
		prosrcFuncionExacto         bool
		definicionFuncionExacta     bool
	)
	err := conexion.QueryRow(
		ctx,
		consultaAcreditacionPoolRecuperacionCoberturaO405,
		firmaFuncionRecuperacionResultadoCoberturaO405,
		rolLectorResultadoCoberturaO405,
		esquemaFuncionRecuperacionResultadoCoberturaO405,
		nombreFuncionRecuperacionResultadoCoberturaO405,
		propietarioFuncionRecuperacionResultadoCoberturaO405,
		configuracionFuncionRecuperacionResultadoCoberturaO405(),
		argumentosFuncionRecuperacionResultadoCoberturaO405,
		retornoFuncionRecuperacionResultadoCoberturaO405,
		lenguajeFuncionRecuperacionResultadoCoberturaO405,
		huellaProsrcFuncionRecuperacionResultadoCoberturaO405,
		huellaDefinicionFuncionRecuperacionResultadoCoberturaO405,
	).Scan(
		&oidFuncion,
		&usuarioSesion,
		&usuarioEfectivo,
		&tlsActivo,
		&primaria,
		&loginSeguro,
		&grupoSeguro,
		&membresiaDirectaExacta,
		&membresiaTotalExacta,
		&loginSinAutoridad,
		&grupoSinAutoridad,
		&privilegiosEfectivosExactos,
		&funcionExacta,
		&propietarioExacto,
		&seguridadFuncionExacta,
		&configuracionFuncionExacta,
		&firmaRetornoFuncionExactos,
		&lenguajeProbinExactos,
		&prosrcFuncionExacto,
		&definicionFuncionExacta,
	)
	tlsEsperado := modo == modoTLSAcreditacionPoolO405Produccion
	if err != nil || oidFuncion == 0 ||
		(oidEsperado != 0 && oidFuncion != oidEsperado) ||
		usuarioSesion == "" || usuarioSesion != usuarioEfectivo ||
		tlsActivo != tlsEsperado || !primaria || !loginSeguro ||
		!grupoSeguro || !membresiaDirectaExacta ||
		!membresiaTotalExacta || !loginSinAutoridad ||
		!grupoSinAutoridad || !privilegiosEfectivosExactos ||
		!funcionExacta || !propietarioExacto ||
		!seguridadFuncionExacta || !configuracionFuncionExacta ||
		!firmaRetornoFuncionExactos || !lenguajeProbinExactos ||
		!prosrcFuncionExacto || !definicionFuncionExacta {
		return 0, errorAcreditacionPoolO405(ctx)
	}
	return oidFuncion, nil
}

func errorAcreditacionPoolO405(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
}
