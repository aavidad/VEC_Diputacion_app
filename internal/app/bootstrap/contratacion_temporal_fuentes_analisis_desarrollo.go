package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	audienciaFuentesAnalisisDesarrollo     = "servicio:ct:desarrollo:analisis"
	raizFuentesAnalisisDesarrolloRef       = "raiz:ct:desarrollo:analisis:v1"
	autoridadFuenteRCAnalisisDesarrollo    = "autoridad:ct:desarrollo:fuente-rc"
	backendFuenteRCAnalisisDesarrollo      = "backend:ct:desarrollo:fuente-rc"
	autoridadCosteAnalisisDesarrollo       = "autoridad:ct:desarrollo:calculo-coste"
	backendCosteAnalisisDesarrollo         = "backend:ct:desarrollo:calculo-coste"
	autoridadVerificadorAnalisisDesarrollo = "autoridad:ct:desarrollo:verificador"
	backendVerificadorAnalisisDesarrollo   = "backend:ct:desarrollo:verificador"
	autoridadPublicadorAnalisisDesarrollo  = "autoridad:ct:desarrollo:publicador"
	backendPublicadorAnalisisDesarrollo    = "backend:ct:desarrollo:publicador"

	numeroRCAnalisisDesarrollo              = "rc:desarrollo:numero:001"
	documentoRCAnalisisDesarrollo           = "documento:rc:desarrollo:001"
	centimosRCAnalisisDesarrollo            = int64(5_000_000)
	centimosCosteAnalisisDesarrollo         = int64(4_000_000)
	dominioSelloPeticionAnalisisDesarrollo  = "hmac-sha256:fuente-analisis-v1:"
	dominioSelloRespuestaAnalisisDesarrollo = "hmac-sha256:fuente-analisis-respuesta/v"
)

type generadorPeticionFuenteAnalisisDesarrollo struct{}

func (generadorPeticionFuenteAnalisisDesarrollo) NuevaReferenciaPeticionFuenteAnalisis(
	ctx context.Context,
	tipo ports.TipoPeticionFuenteAnalisis,
) (string, error) {
	if contextoInterfazNulo(ctx) ||
		(tipo != ports.TipoPeticionValidacionRC &&
			tipo != ports.TipoPeticionCalculoCoste) {
		return "", ports.ErrInfraestructuraFuenteAnalisisNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(
			ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	material := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, material); err != nil {
		borrarBytes(material)
		return "", ports.ErrInfraestructuraFuenteAnalisisNoDisponible
	}
	defer borrarBytes(material)
	if err := ctx.Err(); err != nil {
		return "", errors.Join(
			ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	return "pet_" + base64.RawURLEncoding.EncodeToString(material), nil
}

type selladorPeticionFuenteAnalisisDesarrollo struct {
	derivador *derivadorIdentidadOperacionDesarrollo
}

func (s *selladorPeticionFuenteAnalisisDesarrollo) SellarPeticionFuenteAnalisis(
	ctx context.Context,
	preimagen ports.PreimagenPeticionFuenteAnalisis,
) (string, error) {
	if s == nil || s.derivador == nil || !s.derivador.valido() ||
		contextoInterfazNulo(ctx) {
		return "", ports.ErrInfraestructuraFuenteAnalisisNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(
			ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	material, err := preimagen.Bytes()
	if err != nil {
		return "", ports.ErrPeticionFuenteAnalisisInvalida
	}
	defer borrarBytes(material)
	resultados, err := s.derivador.calcularHMAC(material, material)
	if err != nil || len(resultados) == 0 {
		borrarResultadosHMACIdempotenciaDesarrollo(resultados)
		return "", ports.ErrInfraestructuraFuenteAnalisisNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	if err := ctx.Err(); err != nil {
		return "", errors.Join(
			ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	return dominioSelloPeticionAnalisisDesarrollo +
		hex.EncodeToString(resultados[0].localizador[:]), nil
}

type preparadorSolicitudesFuentesAnalisisDesarrollo struct {
	generador ports.GeneradorPeticionFuenteAnalisis
	sellador  ports.SelladorPeticionFuenteAnalisis
	reloj     relojContratacionTemporalDesarrollo
}

func (p *preparadorSolicitudesFuentesAnalisisDesarrollo) PrepararSolicitudesFuentesAnalisisO3(
	ctx context.Context,
	solicitud ports.SolicitudPrepararArtefactoAnalisis,
) (ports.SolicitudesFuentesAnalisisO3, error) {
	vacias := ports.SolicitudesFuentesAnalisisO3{}
	if p == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.generador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.sellador) ||
		contextoInterfazNulo(ctx) || solicitud.Validar() != nil ||
		!solicitudAnalisisContratacionTemporalDesarrolloValida(solicitud) {
		return vacias, ports.ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacias, errors.Join(
			ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	solicitudRC, err := application.NuevaSolicitudValidarRC(
		ctx,
		p.generador,
		p.sellador,
		p.reloj,
		ports.PreparacionSolicitudValidarRC{
			OrganizacionRef:   solicitud.OrganizacionRef,
			ExpedienteRef:     solicitud.ExpedienteRef,
			VersionExpediente: solicitud.VersionExpediente,
			Entrada:           solicitud.DatosFuncionales.EntradaRC,
			Declaracion:       declaracionRCAnalisisContratacionTemporalDesarrollo(),
		},
	)
	if err != nil {
		return vacias, err
	}
	solicitudCoste, err := application.NuevaSolicitudCalcularCoste(
		ctx,
		p.generador,
		p.sellador,
		p.reloj,
		ports.PreparacionSolicitudCalcularCoste{
			OrganizacionRef:   solicitud.OrganizacionRef,
			ExpedienteRef:     solicitud.ExpedienteRef,
			VersionExpediente: solicitud.VersionExpediente,
			CategoriaRef:      solicitud.DatosFuncionales.CategoriaRef,
			GrupoSubgrupo:     solicitud.DatosFuncionales.GrupoSubgrupo,
			ModalidadClave:    solicitud.DatosFuncionales.ModalidadClave,
			CausaClave:        solicitud.DatosFuncionales.CausaClave,
			Periodo:           solicitud.DatosFuncionales.Periodo,
			Jornada:           solicitud.DatosFuncionales.PorcentajeJornada,
		},
	)
	if err != nil {
		return vacias, err
	}
	return ports.SolicitudesFuentesAnalisisO3{
		ValidacionRC: solicitudRC,
		CalculoCoste: &solicitudCoste,
	}, nil
}

func declaracionRCAnalisisContratacionTemporalDesarrollo() domain.DeclaracionRC {
	return domain.DeclaracionRC{
		Existe: true,
		Numero: numeroRCAnalisisDesarrollo,
		Fecha:  time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		Importe: domain.Importe{
			Centimos: centimosRCAnalisisDesarrollo,
			Moneda:   "EUR",
		},
		DocumentoRef: documentoRCAnalisisDesarrollo,
	}
}

type presentadorAutoridadAnalisisDesarrollo struct {
	derivador  *derivadorIdentidadOperacionDesarrollo
	etiqueta   string
	generacion uint32
	credencial ports.CredencialAutoridadFuenteAnalisis
}

func (p *presentadorAutoridadAnalisisDesarrollo) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	if p == nil || p.derivador == nil || !p.derivador.valido() ||
		p.generacion == 0 || contextoInterfazNulo(ctx) {
		return ports.PresentacionAutoridadFuenteAnalisis{},
			errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	materialDesafio, err := desafio.Bytes()
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{},
			errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	defer borrarBytes(materialDesafio)
	semilla, generacion, err := derivarSemillaAutoridadAnalisisDesarrollo(
		p.derivador,
		p.etiqueta,
	)
	if err != nil || generacion != p.generacion {
		borrarBytes(semilla[:])
		return ports.PresentacionAutoridadFuenteAnalisis{},
			errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	defer borrarBytes(semilla[:])
	privada := ed25519.NewKeyFromSeed(semilla[:])
	defer borrarBytes(privada)
	firma := ed25519.Sign(privada, materialDesafio)
	defer borrarBytes(firma)
	return ports.NuevaPresentacionAutoridadFuenteAnalisis(
		p.credencial,
		firma,
	)
}

type fuenteRCAnalisisDesarrollo struct {
	*presentadorAutoridadAnalisisDesarrollo
	derivador    *derivadorIdentidadOperacionDesarrollo
	autoridadRef string
	generacion   uint32
	reloj        relojContratacionTemporalDesarrollo
}

func (f *fuenteRCAnalisisDesarrollo) ValidarRC(
	ctx context.Context,
	solicitud ports.SolicitudValidarRC,
) (ports.ResultadoValidacionRC, error) {
	if f == nil || f.derivador == nil || !f.derivador.valido() ||
		f.generacion == 0 || contextoInterfazNulo(ctx) {
		return ports.ResultadoValidacionRC{},
			ports.ErrFuentePresupuestariaNoDisponible
	}
	datos, err := solicitud.Datos()
	declaracion := declaracionRCAnalisisContratacionTemporalDesarrollo()
	if err != nil || datos.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		datos.Entrada.Referencia != entradaRCAnalisisContratacionTemporalDesarrollo ||
		!hmac.Equal(
			[]byte(datos.Entrada.HuellaSHA256),
			[]byte(huellaEntradaRCAnalisisContratacionTemporalDesarrollo),
		) ||
		datos.Declaracion != declaracion {
		return ports.ResultadoValidacionRC{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoValidacionRC{}, err
	}
	ahora := f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.Before(datos.SolicitadaEn) {
		return ports.ResultadoValidacionRC{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	reciboRef, err := referenciaReciboFuenteAnalisisDesarrollo(
		f.derivador,
		"rc",
		datos.PeticionRef,
	)
	if err != nil {
		return ports.ResultadoValidacionRC{},
			ports.ErrFuentePresupuestariaNoDisponible
	}
	fecha := declaracion.Fecha
	importe := declaracion.Importe
	validacion := domain.ValidacionRC{
		Resultado:           domain.RCValidada,
		EntradaRef:          datos.Entrada.Referencia,
		HuellaEntradaSHA256: datos.Entrada.HuellaSHA256,
		FuenteRef:           f.autoridadRef,
		ReciboRef:           reciboRef,
		ValidadaEn:          ahora,
		FechaRC:             &fecha,
		Numero:              declaracion.Numero,
		Importe:             &importe,
		DocumentoRef:        declaracion.DocumentoRef,
	}
	metadatos := ports.MetadatosAtestacionRespuestaFuenteAnalisis{
		AutoridadRef: f.autoridadRef,
		Generacion:   f.generacion,
		ReciboRef:    reciboRef,
		EmitidaEn:    ahora,
		ValidaHasta:  ahora.Add(ports.VigenciaMaximaRespuestaFuenteAnalisis),
	}
	preimagen, err := ports.NuevaPreimagenRespuestaValidacionRC(
		solicitud,
		validacion,
		ports.MotivoFuenteAnalisis{},
		metadatos,
	)
	if err != nil {
		return ports.ResultadoValidacionRC{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	atestacion, err := nuevaAtestacionRespuestaFuenteAnalisisDesarrollo(
		f.derivador,
		preimagen,
		metadatos,
	)
	if err != nil {
		return ports.ResultadoValidacionRC{}, err
	}
	return ports.NuevoResultadoValidacionRC(
		solicitud,
		validacion,
		ports.MotivoFuenteAnalisis{},
		atestacion,
	)
}

type calculadorCosteAnalisisDesarrollo struct {
	*presentadorAutoridadAnalisisDesarrollo
	derivador    *derivadorIdentidadOperacionDesarrollo
	autoridadRef string
	generacion   uint32
	reloj        relojContratacionTemporalDesarrollo
}

func (c *calculadorCosteAnalisisDesarrollo) CalcularCoste(
	ctx context.Context,
	solicitud ports.SolicitudCalcularCoste,
) (ports.ResultadoCalculoCoste, error) {
	if c == nil || c.derivador == nil || !c.derivador.valido() ||
		c.generacion == 0 || contextoInterfazNulo(ctx) {
		return ports.ResultadoCalculoCoste{},
			ports.ErrCalculadorCosteNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil ||
		!datosCalculoCosteAnalisisContratacionTemporalDesarrolloValidos(datos) {
		return ports.ResultadoCalculoCoste{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	ahora := c.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.Before(datos.SolicitadaEn) {
		return ports.ResultadoCalculoCoste{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	reciboRef, err := referenciaReciboFuenteAnalisisDesarrollo(
		c.derivador,
		"coste",
		datos.PeticionRef,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{},
			ports.ErrCalculadorCosteNoDisponible
	}
	importe := domain.Importe{
		Centimos: centimosCosteAnalisisDesarrollo,
		Moneda:   "EUR",
	}
	metadatos := ports.MetadatosAtestacionRespuestaFuenteAnalisis{
		AutoridadRef: c.autoridadRef,
		Generacion:   c.generacion,
		ReciboRef:    reciboRef,
		EmitidaEn:    ahora,
		ValidaHasta:  ahora.Add(ports.VigenciaMaximaRespuestaFuenteAnalisis),
	}
	preimagen, err := ports.NuevaPreimagenRespuestaCalculoCoste(
		solicitud,
		c.autoridadRef,
		reciboRef,
		importe,
		ahora,
		metadatos,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	atestacion, err := nuevaAtestacionRespuestaFuenteAnalisisDesarrollo(
		c.derivador,
		preimagen,
		metadatos,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	return ports.NuevoResultadoCalculoCoste(
		solicitud,
		c.autoridadRef,
		reciboRef,
		importe,
		ahora,
		atestacion,
	)
}

func datosCalculoCosteAnalisisContratacionTemporalDesarrolloValidos(
	datos ports.DatosSolicitudCalcularCoste,
) bool {
	solicitud := ports.SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      artefactoAnalisisContratacionTemporalDesarrollo,
		OrganizacionRef:   datos.OrganizacionRef,
		ExpedienteRef:     datos.ExpedienteRef,
		VersionExpediente: datos.VersionExpediente,
		DatosFuncionales: ports.DatosFuncionalesOperacionAnalisis{
			ModalidadClave:    datos.ModalidadClave,
			CategoriaRef:      datos.CategoriaRef,
			GrupoSubgrupo:     datos.GrupoSubgrupo,
			CausaClave:        datos.CausaClave,
			Periodo:           datos.Periodo,
			PorcentajeJornada: datos.Jornada,
			EntradaRC: domain.VinculoEntradaRC{
				Referencia:   entradaRCAnalisisContratacionTemporalDesarrollo,
				HuellaSHA256: huellaEntradaRCAnalisisContratacionTemporalDesarrollo,
			},
		},
		SolicitadaEn: datos.SolicitadaEn,
	}
	return solicitud.Validar() == nil &&
		solicitudAnalisisContratacionTemporalDesarrolloValida(solicitud)
}

type verificadorRespuestaFuenteAnalisisDesarrollo struct {
	*presentadorAutoridadAnalisisDesarrollo
	derivador    *derivadorIdentidadOperacionDesarrollo
	autoridadRef string
	reloj        relojContratacionTemporalDesarrollo
}

func (v *verificadorRespuestaFuenteAnalisisDesarrollo) VerificarRespuestaFuenteAnalisis(
	ctx context.Context,
	solicitud ports.SolicitudVerificarRespuestaFuenteAnalisis,
) (ports.ConfirmacionRespuestaFuenteAnalisis, error) {
	if v == nil || v.derivador == nil || !v.derivador.valido() ||
		contextoInterfazNulo(ctx) {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			ports.ErrVerificacionFuenteAnalisisNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{}, err
	}
	preimagen, atestacion, err := solicitud.Material()
	if err != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	material, err := preimagen.Bytes()
	if err != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	defer borrarBytes(material)
	clave, err := derivarClaveRespuestaAnalisisDesarrollo(
		v.derivador,
		atestacion.Metadatos.Generacion,
	)
	if err != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			ports.ErrVerificacionFuenteAnalisisNoDisponible
	}
	defer borrarBytes(clave[:])
	mac := hmac.New(sha256.New, clave[:])
	_, _ = mac.Write(material)
	suma := mac.Sum(nil)
	defer borrarBytes(suma)
	esperado := dominioSelloRespuestaAnalisisDesarrollo +
		numeroDecimal(atestacion.Metadatos.Generacion) + ":" +
		hex.EncodeToString(suma)
	if !hmac.Equal([]byte(esperado), []byte(atestacion.SelloHMAC)) {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	ahora := v.reloj.Ahora().UTC().Truncate(time.Microsecond)
	return ports.NuevaConfirmacionRespuestaFuenteAnalisis(
		solicitud,
		v.autoridadRef,
		ahora,
	)
}

type publicadorMotivoFuenteAnalisisDesarrollo struct {
	*presentadorAutoridadAnalisisDesarrollo
}

func (*publicadorMotivoFuenteAnalisisDesarrollo) VerificarPublicacionMotivoFuenteAnalisis(
	ctx context.Context,
	_ ports.SolicitudVerificarPublicacionMotivoFuenteAnalisis,
) (ports.ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
	if contextoInterfazNulo(ctx) {
		return ports.ConfirmacionPublicacionMotivoFuenteAnalisis{},
			ports.ErrVerificacionFuenteAnalisisNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionPublicacionMotivoFuenteAnalisis{}, err
	}
	return ports.ConfirmacionPublicacionMotivoFuenteAnalisis{},
		ports.ErrVerificacionFuenteAnalisisNoDisponible
}

func nuevoPreparadorFuentesAnalisisContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*application.CapacidadPrepararArtefactoAnalisisO3, error) {
	if derivador == nil || !derivador.valido() {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	semillaRaiz, generacion, err := derivarSemillaAutoridadAnalisisDesarrollo(
		derivador,
		"raiz-institucional",
	)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	defer borrarBytes(semillaRaiz[:])
	privadaRaiz := ed25519.NewKeyFromSeed(semillaRaiz[:])
	defer borrarBytes(privadaRaiz)
	publicaRaiz := append(
		ed25519.PublicKey(nil),
		privadaRaiz.Public().(ed25519.PublicKey)...,
	)
	defer borrarBytes(publicaRaiz)
	crearPresentador := func(
		etiqueta string,
		autoridadRef string,
		backendRef string,
		rol ports.RolAutoridadFuenteAnalisis,
		serie uint64,
	) (*presentadorAutoridadAnalisisDesarrollo, error) {
		semilla, generacionPresentador, errSemilla :=
			derivarSemillaAutoridadAnalisisDesarrollo(derivador, etiqueta)
		if errSemilla != nil || generacionPresentador != generacion {
			borrarBytes(semilla[:])
			return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
		}
		defer borrarBytes(semilla[:])
		privada := ed25519.NewKeyFromSeed(semilla[:])
		defer borrarBytes(privada)
		datos := ports.DatosCredencialAutoridadFuenteAnalisis{
			RaizClaveID:        raizFuentesAnalisisDesarrolloRef,
			AutoridadRef:       autoridadRef,
			BackendRef:         backendRef,
			OrganizacionRef:    organizacionAltaContratacionTemporalDesarrollo,
			Audiencia:          audienciaFuentesAnalisisDesarrollo,
			Rol:                rol,
			Serie:              serie,
			Generacion:         generacion,
			ClavePruebaEd25519: privada.Public().(ed25519.PublicKey),
			EmitidaEn:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ValidaHasta:        time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		material, errMaterial :=
			ports.MaterialFirmaCredencialAutoridadFuenteAnalisis(datos)
		if errMaterial != nil {
			return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
		}
		defer borrarBytes(material)
		firma := ed25519.Sign(privadaRaiz, material)
		defer borrarBytes(firma)
		credencial, errCredencial :=
			ports.NuevaCredencialAutoridadFuenteAnalisis(datos, firma)
		if errCredencial != nil {
			return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
		}
		return &presentadorAutoridadAnalisisDesarrollo{
			derivador: derivador, etiqueta: etiqueta,
			generacion: generacion, credencial: credencial,
		}, nil
	}
	presentadorRC, err := crearPresentador(
		"fuente-rc",
		autoridadFuenteRCAnalisisDesarrollo,
		backendFuenteRCAnalisisDesarrollo,
		ports.RolFuentePresupuestaria,
		1,
	)
	if err != nil {
		return nil, err
	}
	presentadorCoste, err := crearPresentador(
		"calculo-coste",
		autoridadCosteAnalisisDesarrollo,
		backendCosteAnalisisDesarrollo,
		ports.RolCalculadorCoste,
		2,
	)
	if err != nil {
		return nil, err
	}
	presentadorVerificador, err := crearPresentador(
		"verificador",
		autoridadVerificadorAnalisisDesarrollo,
		backendVerificadorAnalisisDesarrollo,
		ports.RolVerificadorRespuesta,
		3,
	)
	if err != nil {
		return nil, err
	}
	presentadorPublicador, err := crearPresentador(
		"publicador",
		autoridadPublicadorAnalisisDesarrollo,
		backendPublicadorAnalisisDesarrollo,
		ports.RolPublicadorCatalogo,
		4,
	)
	if err != nil {
		return nil, err
	}
	confianza, err := ports.NuevaConfianzaAutoridadesFuenteAnalisis(
		organizacionAltaContratacionTemporalDesarrollo,
		audienciaFuentesAnalisisDesarrollo,
		[]ports.RaizConfianzaAutoridadFuenteAnalisis{{
			ClaveID:             raizFuentesAnalisisDesarrolloRef,
			ClavePublicaEd25519: publicaRaiz,
			Estado:              ports.RaizAutoridadActiva,
			ValidaDesde:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ValidaHasta:         time.Date(2037, 1, 1, 0, 0, 0, 0, time.UTC),
			UltimaEmisionPermitida: time.Date(
				2036, 1, 1, 0, 0, 0, 0, time.UTC,
			),
		}},
		nil,
	)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	preparador := &preparadorSolicitudesFuentesAnalisisDesarrollo{
		generador: generadorPeticionFuenteAnalisisDesarrollo{},
		sellador:  &selladorPeticionFuenteAnalisisDesarrollo{derivador: derivador},
		reloj:     reloj,
	}
	capacidad, err :=
		application.NuevaCapacidadPrepararArtefactoAnalisisO3ParaComposicionInterna(
			preparador,
			&fuenteRCAnalisisDesarrollo{
				presentadorAutoridadAnalisisDesarrollo: presentadorRC,
				derivador:                              derivador, autoridadRef: autoridadFuenteRCAnalisisDesarrollo,
				generacion: generacion, reloj: reloj,
			},
			&calculadorCosteAnalisisDesarrollo{
				presentadorAutoridadAnalisisDesarrollo: presentadorCoste,
				derivador:                              derivador, autoridadRef: autoridadCosteAnalisisDesarrollo,
				generacion: generacion, reloj: reloj,
			},
			&verificadorRespuestaFuenteAnalisisDesarrollo{
				presentadorAutoridadAnalisisDesarrollo: presentadorVerificador,
				derivador:                              derivador,
				autoridadRef:                           autoridadVerificadorAnalisisDesarrollo,
				reloj:                                  reloj,
			},
			&publicadorMotivoFuenteAnalisisDesarrollo{
				presentadorAutoridadAnalisisDesarrollo: presentadorPublicador,
			},
			confianza,
			reloj,
		)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	return capacidad, nil
}

func derivarSemillaAutoridadAnalisisDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	etiqueta string,
) ([ed25519.SeedSize]byte, uint32, error) {
	var semilla [ed25519.SeedSize]byte
	if derivador == nil || !derivador.valido() || etiqueta == "" {
		return semilla, 0, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	resultados, err := derivador.calcularHMAC(
		[]byte("vec.ct.desarrollo.analisis."+etiqueta+".v1"),
		[]byte("vec.ct.desarrollo.analisis."+etiqueta+".material.v1"),
	)
	if err != nil || len(resultados) == 0 {
		borrarResultadosHMACIdempotenciaDesarrollo(resultados)
		return semilla, 0, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	copy(semilla[:], resultados[0].localizador[:])
	return semilla, resultados[0].generacion, nil
}

func derivarClaveRespuestaAnalisisDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	generacion uint32,
) ([sha256.Size]byte, error) {
	var clave [sha256.Size]byte
	if derivador == nil || !derivador.valido() || generacion == 0 {
		return clave, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	resultados, err := derivador.calcularHMAC(
		[]byte("vec.ct.desarrollo.analisis.hmac-respuesta.v1"),
		[]byte("vec.ct.desarrollo.analisis.hmac-respuesta.material.v1"),
	)
	if err != nil {
		borrarResultadosHMACIdempotenciaDesarrollo(resultados)
		return clave, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	for _, resultado := range resultados {
		if resultado.generacion == generacion {
			copy(clave[:], resultado.huellaSolicitud[:])
			return clave, nil
		}
	}
	return clave, errAnalisisContratacionTemporalDesarrolloNoDisponible
}

func nuevaAtestacionRespuestaFuenteAnalisisDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	preimagen ports.PreimagenRespuestaFuenteAnalisis,
	metadatos ports.MetadatosAtestacionRespuestaFuenteAnalisis,
) (ports.AtestacionRespuestaFuenteAnalisis, error) {
	material, err := preimagen.Bytes()
	if err != nil || metadatos.Validar() != nil {
		return ports.AtestacionRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	defer borrarBytes(material)
	clave, err := derivarClaveRespuestaAnalisisDesarrollo(
		derivador,
		metadatos.Generacion,
	)
	if err != nil {
		return ports.AtestacionRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	defer borrarBytes(clave[:])
	mac := hmac.New(sha256.New, clave[:])
	_, _ = mac.Write(material)
	suma := mac.Sum(nil)
	defer borrarBytes(suma)
	return ports.NuevaAtestacionRespuestaFuenteAnalisis(
		metadatos,
		dominioSelloRespuestaAnalisisDesarrollo+
			numeroDecimal(metadatos.Generacion)+":"+
			hex.EncodeToString(suma),
	)
}

func referenciaReciboFuenteAnalisisDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	tipo string,
	peticionRef string,
) (string, error) {
	if tipo == "" || peticionRef == "" {
		return "", errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	material := []byte("vec.ct.desarrollo.analisis.recibo." + tipo + ".v1\x00" + peticionRef)
	defer borrarBytes(material)
	resultados, err := derivador.calcularHMAC(material, material)
	if err != nil || len(resultados) == 0 {
		borrarResultadosHMACIdempotenciaDesarrollo(resultados)
		return "", errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	return "recibo:ct:desarrollo:" + tipo + ":" +
		hex.EncodeToString(resultados[0].localizador[:16]), nil
}

var (
	_ ports.GeneradorPeticionFuenteAnalisis            = generadorPeticionFuenteAnalisisDesarrollo{}
	_ ports.SelladorPeticionFuenteAnalisis             = (*selladorPeticionFuenteAnalisisDesarrollo)(nil)
	_ ports.PreparadorSolicitudesFuentesAnalisisO3     = (*preparadorSolicitudesFuentesAnalisisDesarrollo)(nil)
	_ ports.FuentePresupuestaria                       = (*fuenteRCAnalisisDesarrollo)(nil)
	_ ports.CalculadorCostePersonal                    = (*calculadorCosteAnalisisDesarrollo)(nil)
	_ ports.VerificadorRespuestaFuenteAnalisis         = (*verificadorRespuestaFuenteAnalisisDesarrollo)(nil)
	_ ports.VerificadorPublicacionMotivoFuenteAnalisis = (*publicadorMotivoFuenteAnalisisDesarrollo)(nil)
)
