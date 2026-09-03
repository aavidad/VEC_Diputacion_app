package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	audienciaFuentesCoberturaDesarrollo     = "servicio:ct:desarrollo:cobertura"
	raizFuentesCoberturaDesarrolloRef       = "raiz:ct:desarrollo:cobertura:v1"
	backendFuenteCoberturaDesarrolloRef     = "fuente:ct:desarrollo:cobertura:v1"
	autoridadFuenteCoberturaDesarrollo      = "autoridad:ct:desarrollo:fuente-cobertura"
	autoridadVerificadorCoberturaDesarrollo = "autoridad:ct:desarrollo:verificador-cobertura"
	autoridadPublicadorCoberturaDesarrollo  = "autoridad:ct:desarrollo:publicador-catalogo"
	backendVerificadorCoberturaDesarrollo   = "backend:ct:desarrollo:verificador-cobertura"
	backendPublicadorCoberturaDesarrollo    = "backend:ct:desarrollo:publicador-catalogo"
)

var errFuentesCoberturaDesarrolloNoDisponibles = errors.New(
	"contratacion temporal: fuentes de cobertura de desarrollo no disponibles",
)

type dependenciasFuentesCoberturaDesarrollo struct {
	fuente       ports.FuenteComprobacionCobertura
	verificador  ports.VerificadorRespuestaCobertura
	publicador   ports.PublicadorCatalogoCobertura
	autenticador ports.VerificadorPresentacionesAutoridadFuenteAnalisis
	referencias  application.GeneradorReferenciasComprobacionCobertura
	cerrar       func()
}

type presentadorAutoridadFuenteAnalisisDesarrollo struct {
	credencial ports.CredencialAutoridadFuenteAnalisis
	privada    ed25519.PrivateKey
}

func (p *presentadorAutoridadFuenteAnalisisDesarrollo) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	if p == nil || len(p.privada) != ed25519.PrivateKeySize ||
		contextoInterfazNulo(ctx) {
		return ports.PresentacionAutoridadFuenteAnalisis{},
			errFuentesCoberturaDesarrolloNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{}, err
	}
	material, err := desafio.Bytes()
	if err != nil {
		return ports.PresentacionAutoridadFuenteAnalisis{},
			errFuentesCoberturaDesarrolloNoDisponibles
	}
	defer borrarBytes(material)
	firma := ed25519.Sign(p.privada, material)
	defer borrarBytes(firma)
	return ports.NuevaPresentacionAutoridadFuenteAnalisis(
		p.credencial,
		firma,
	)
}

type registroCoberturaSinteticaDesarrollo struct {
	categoriaRef string
	periodo      domain.PeriodoPrevisto
	resultado    domain.ResultadoComprobacion
}

type fuenteComprobacionCoberturaDesarrollo struct {
	*presentadorAutoridadFuenteAnalisisDesarrollo
	autoridadRef   string
	backendRef     string
	generacion     uint32
	claveRespuesta [sha256.Size]byte
	claveRecibo    [sha256.Size]byte
	reloj          ports.Reloj
	registros      []registroCoberturaSinteticaDesarrollo
}

func (f *fuenteComprobacionCoberturaDesarrollo) ConsultarCobertura(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (ports.ResultadoConsultaCobertura, error) {
	if f == nil || dependenciaEsNulaContratacionTemporalDesarrollo(f.reloj) ||
		f.generacion == 0 || contextoInterfazNulo(ctx) ||
		solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.ViaClave != domain.ClaveCatalogo("bolsa_vigente") ||
		solicitud.Comprobacion != comprobacionFuenteCoberturaDesarrollo(f.backendRef) {
		return ports.ResultadoConsultaCobertura{},
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaCobertura{}, err
	}
	resultado, existe := f.resultadoPara(solicitud.CategoriaRef, solicitud.Periodo)
	if !existe {
		return ports.ResultadoConsultaCobertura{},
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	materialPeticion, err := solicitud.MaterialCanonico()
	if err != nil {
		return ports.ResultadoConsultaCobertura{},
			ports.ErrPeticionFuenteCoberturaInvalida
	}
	defer borrarBytes(materialPeticion)
	huellaPeticion := sha256.Sum256(materialPeticion)
	reciboMAC := hmac.New(sha256.New, f.claveRecibo[:])
	_, _ = reciboMAC.Write(materialPeticion)
	recibo := reciboMAC.Sum(nil)
	defer borrarBytes(recibo)
	ahora := f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.Before(solicitud.SolicitadaEn) {
		return ports.ResultadoConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	reciboRef := "recibo:ct:desarrollo:cobertura:" + hex.EncodeToString(recibo)
	datos := ports.DatosResultadoConsultaCobertura{
		PeticionRef:          solicitud.PeticionRef,
		HuellaPeticionSHA256: hex.EncodeToString(huellaPeticion[:]),
		OrganizacionRef:      solicitud.OrganizacionRef,
		ExpedienteRef:        solicitud.ExpedienteRef,
		VersionExpediente:    solicitud.VersionExpediente,
		Catalogo:             solicitud.Catalogo,
		ViaClave:             solicitud.ViaClave,
		ProcedenciaClave:     solicitud.Comprobacion.Procedencia.Clave,
		CategoriaRef:         solicitud.CategoriaRef,
		Periodo:              solicitud.Periodo,
		Comprobacion: domain.ComprobacionCobertura{
			Clave:      solicitud.Comprobacion.Clave,
			Resultado:  resultado,
			FuenteRef:  f.autoridadRef,
			ReciboRef:  reciboRef,
			EvaluadaEn: ahora,
		},
		DefinicionFuenteRef: solicitud.Comprobacion.Procedencia.DefinicionFuenteRef,
	}
	metadatos := ports.MetadatosAtestacionRespuestaCobertura{
		AutoridadRef: f.autoridadRef,
		Generacion:   f.generacion,
		ReciboRef:    reciboRef,
		EmitidaEn:    ahora,
		ValidaHasta:  ahora.Add(ports.VigenciaMaximaRespuestaCobertura),
	}
	preimagen, err := ports.NuevaPreimagenRespuestaCobertura(datos, metadatos)
	if err != nil {
		return ports.ResultadoConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	materialRespuesta, err := preimagen.Bytes()
	if err != nil {
		return ports.ResultadoConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	defer borrarBytes(materialRespuesta)
	mac := hmac.New(sha256.New, f.claveRespuesta[:])
	_, _ = mac.Write(materialRespuesta)
	suma := mac.Sum(nil)
	defer borrarBytes(suma)
	sello := "hmac-sha256:fuente-cobertura-respuesta/v" +
		numeroDecimal(f.generacion) + ":" + hex.EncodeToString(suma)
	atestacion, err := ports.NuevaAtestacionRespuestaCobertura(metadatos, sello)
	if err != nil {
		return ports.ResultadoConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return ports.NuevoResultadoConsultaCobertura(datos, atestacion)
}

func comprobacionFuenteCoberturaDesarrollo(
	backendRef string,
) domain.ComprobacionExigibleCobertura {
	return domain.ComprobacionExigibleCobertura{
		Clave:       "existe_bolsa_vigente",
		Orden:       1,
		Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "bolsa",
			DefinicionFuenteRef: backendRef,
		},
	}
}

func (f *fuenteComprobacionCoberturaDesarrollo) resultadoPara(
	categoriaRef string,
	periodo domain.PeriodoPrevisto,
) (domain.ResultadoComprobacion, bool) {
	for _, registro := range f.registros {
		if registro.categoriaRef == categoriaRef &&
			registro.periodo.Inicio.Equal(periodo.Inicio) &&
			registro.periodo.Fin.Equal(periodo.Fin) {
			return registro.resultado, true
		}
	}
	return "", false
}

type verificadorRespuestaCoberturaDesarrollo struct {
	*presentadorAutoridadFuenteAnalisisDesarrollo
	autoridadRef   string
	generacion     uint32
	claveRespuesta [sha256.Size]byte
	reloj          ports.Reloj
}

func (v *verificadorRespuestaCoberturaDesarrollo) VerificarRespuestaCobertura(
	ctx context.Context,
	solicitud ports.SolicitudVerificarRespuestaCobertura,
) (ports.ConfirmacionRespuestaCobertura, error) {
	if v == nil || dependenciaEsNulaContratacionTemporalDesarrollo(v.reloj) ||
		v.generacion == 0 || contextoInterfazNulo(ctx) {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionRespuestaCobertura{}, err
	}
	preimagen, atestacion, err := solicitud.Material()
	if err != nil || atestacion.Metadatos.Generacion != v.generacion {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	material, err := preimagen.Bytes()
	if err != nil {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	defer borrarBytes(material)
	mac := hmac.New(sha256.New, v.claveRespuesta[:])
	_, _ = mac.Write(material)
	suma := mac.Sum(nil)
	defer borrarBytes(suma)
	esperado := "hmac-sha256:fuente-cobertura-respuesta/v" +
		numeroDecimal(v.generacion) + ":" + hex.EncodeToString(suma)
	if !hmac.Equal([]byte(esperado), []byte(atestacion.SelloHMAC)) {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	ahora := v.reloj.Ahora().UTC().Truncate(time.Microsecond)
	preimagenConfirmacion, err := ports.NuevaPreimagenConfirmacionRespuestaCobertura(
		solicitud,
		v.autoridadRef,
		ahora,
	)
	if err != nil {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	materialConfirmacion, err := preimagenConfirmacion.Bytes()
	if err != nil {
		return ports.ConfirmacionRespuestaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	defer borrarBytes(materialConfirmacion)
	firma := ed25519.Sign(v.privada, materialConfirmacion)
	defer borrarBytes(firma)
	return ports.NuevaConfirmacionRespuestaCobertura(
		solicitud,
		v.autoridadRef,
		ahora,
		firma,
	)
}

type publicadorCatalogoCoberturaDesarrollo struct {
	*presentadorAutoridadFuenteAnalisisDesarrollo
	autoridadRef string
	reloj        cobertura.RelojGobiernoOperacionCobertura
	gobierno     cobertura.ResolutorGobiernoOperacionCobertura
}

func (p *publicadorCatalogoCoberturaDesarrollo) ConsultarPublicacionCobertura(
	ctx context.Context,
	solicitud ports.SolicitudConsultarCobertura,
) (ports.ConfirmacionPublicacionCobertura, error) {
	if p == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.gobierno) ||
		contextoInterfazNulo(ctx) ||
		solicitud.Validar() != nil {
		return ports.ConfirmacionPublicacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	constructores := []func(string, string, uint64) (
		cobertura.SolicitudGobiernoOperacionCobertura,
		error,
	){
		cobertura.NuevaSolicitudGobiernoDecisionCobertura,
		cobertura.NuevaSolicitudGobiernoRectificacionCobertura,
	}
	for _, construir := range constructores {
		peticion, err := construir(
			solicitud.OrganizacionRef,
			solicitud.ExpedienteRef,
			solicitud.VersionExpediente,
		)
		if err != nil {
			continue
		}
		gobierno, err := cobertura.ObtenerGobiernoOperacionCobertura(
			ctx,
			p.reloj,
			p.gobierno,
			peticion,
		)
		if err != nil {
			continue
		}
		datos, err := gobierno.DesplegarPara(ctx, p.reloj, peticion)
		if err != nil ||
			!datos.Catalogo.Identidad().CoincideExactamente(solicitud.Catalogo) {
			continue
		}
		ahora, err := p.reloj.AhoraGobiernoOperacionCobertura(ctx)
		if err != nil {
			break
		}
		return ports.NuevaConfirmacionPublicacionCobertura(
			p.autoridadRef,
			datos.Catalogo.Publicacion(),
			ahora,
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionPublicacionCobertura{}, err
	}
	return ports.ConfirmacionPublicacionCobertura{},
		ports.ErrResultadoFuenteCoberturaNoConfiable
}

type materialFuentesCoberturaDesarrollo struct {
	raiz, fuente, verificador, publicador [ed25519.SeedSize]byte
	respuesta, recibo                     [sha256.Size]byte
	generacion                            uint32
}

func (m *materialFuentesCoberturaDesarrollo) borrar() {
	if m == nil {
		return
	}
	borrarBytes(m.raiz[:])
	borrarBytes(m.fuente[:])
	borrarBytes(m.verificador[:])
	borrarBytes(m.publicador[:])
	borrarBytes(m.respuesta[:])
	borrarBytes(m.recibo[:])
	m.generacion = 0
}

func nuevasDependenciasFuentesCoberturaDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
	gobierno cobertura.ResolutorGobiernoOperacionCobertura,
) (dependenciasFuentesCoberturaDesarrollo, error) {
	var vacias dependenciasFuentesCoberturaDesarrollo
	if derivador == nil || !derivador.valido() ||
		dependenciaEsNulaContratacionTemporalDesarrollo(gobierno) {
		return vacias, errFuentesCoberturaDesarrolloNoDisponibles
	}
	material, err := derivarMaterialFuentesCoberturaDesarrollo(derivador)
	if err != nil {
		return vacias, err
	}
	defer material.borrar()
	claveRaiz := ed25519.NewKeyFromSeed(material.raiz[:])
	defer borrarBytes(claveRaiz)
	claveFuente := ed25519.NewKeyFromSeed(material.fuente[:])
	claveVerificador := ed25519.NewKeyFromSeed(material.verificador[:])
	clavePublicador := ed25519.NewKeyFromSeed(material.publicador[:])
	clavesOperativas := []ed25519.PrivateKey{
		claveFuente,
		claveVerificador,
		clavePublicador,
	}
	presentadores := make([]*presentadorAutoridadFuenteAnalisisDesarrollo, 0, 3)
	completo := false
	defer func() {
		if !completo {
			for _, clave := range clavesOperativas {
				borrarBytes(clave)
			}
		}
	}()
	validaDesde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validaHasta := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	crearPresentador := func(
		autoridadRef string,
		backendRef string,
		rol ports.RolAutoridadFuenteAnalisis,
		serie uint64,
		privada ed25519.PrivateKey,
	) (*presentadorAutoridadFuenteAnalisisDesarrollo, error) {
		datos := ports.DatosCredencialAutoridadFuenteAnalisis{
			RaizClaveID:        raizFuentesCoberturaDesarrolloRef,
			AutoridadRef:       autoridadRef,
			BackendRef:         backendRef,
			OrganizacionRef:    organizacionAltaContratacionTemporalDesarrollo,
			Audiencia:          audienciaFuentesCoberturaDesarrollo,
			Rol:                rol,
			Serie:              serie,
			Generacion:         material.generacion,
			ClavePruebaEd25519: privada.Public().(ed25519.PublicKey),
			EmitidaEn:          validaDesde,
			ValidaHasta:        validaHasta,
		}
		preimagen, err := ports.MaterialFirmaCredencialAutoridadFuenteAnalisis(datos)
		if err != nil {
			return nil, errFuentesCoberturaDesarrolloNoDisponibles
		}
		defer borrarBytes(preimagen)
		firma := ed25519.Sign(claveRaiz, preimagen)
		defer borrarBytes(firma)
		credencial, err := ports.NuevaCredencialAutoridadFuenteAnalisis(datos, firma)
		if err != nil {
			return nil, errFuentesCoberturaDesarrolloNoDisponibles
		}
		presentador := &presentadorAutoridadFuenteAnalisisDesarrollo{
			credencial: credencial,
			privada:    privada,
		}
		presentadores = append(presentadores, presentador)
		return presentador, nil
	}
	presentadorFuente, err := crearPresentador(
		autoridadFuenteCoberturaDesarrollo,
		backendFuenteCoberturaDesarrolloRef,
		ports.RolFuenteCobertura,
		1,
		claveFuente,
	)
	if err != nil {
		return vacias, err
	}
	presentadorVerificador, err := crearPresentador(
		autoridadVerificadorCoberturaDesarrollo,
		backendVerificadorCoberturaDesarrollo,
		ports.RolVerificadorCobertura,
		2,
		claveVerificador,
	)
	if err != nil {
		return vacias, err
	}
	presentadorPublicador, err := crearPresentador(
		autoridadPublicadorCoberturaDesarrollo,
		backendPublicadorCoberturaDesarrollo,
		ports.RolPublicadorCatalogoCobertura,
		3,
		clavePublicador,
	)
	if err != nil {
		return vacias, err
	}
	confianza, err := ports.NuevaConfianzaAutoridadesFuenteAnalisis(
		organizacionAltaContratacionTemporalDesarrollo,
		audienciaFuentesCoberturaDesarrollo,
		[]ports.RaizConfianzaAutoridadFuenteAnalisis{{
			ClaveID:                raizFuentesCoberturaDesarrolloRef,
			ClavePublicaEd25519:    claveRaiz.Public().(ed25519.PublicKey),
			Estado:                 ports.RaizAutoridadActiva,
			ValidaDesde:            validaDesde,
			ValidaHasta:            validaHasta.AddDate(1, 0, 0),
			UltimaEmisionPermitida: validaHasta,
		}},
		nil,
	)
	if err != nil {
		return vacias, errFuentesCoberturaDesarrolloNoDisponibles
	}
	autenticador, err := seguridadcontratacion.NuevoAutenticadorFuentesAnalisisConConfianza(
		confianza,
	)
	if err != nil {
		return vacias, errFuentesCoberturaDesarrolloNoDisponibles
	}
	respuestaFuente := material.respuesta
	respuestaVerificador := material.respuesta
	dependencias := dependenciasFuentesCoberturaDesarrollo{
		fuente: &fuenteComprobacionCoberturaDesarrollo{
			presentadorAutoridadFuenteAnalisisDesarrollo: presentadorFuente,
			autoridadRef:   autoridadFuenteCoberturaDesarrollo,
			backendRef:     backendFuenteCoberturaDesarrolloRef,
			generacion:     material.generacion,
			claveRespuesta: respuestaFuente,
			claveRecibo:    material.recibo,
			reloj:          reloj,
			registros: []registroCoberturaSinteticaDesarrollo{
				{
					categoriaRef: categoriaAltaContratacionTemporalDesarrollo,
					periodo: domain.PeriodoPrevisto{
						Inicio: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
						Fin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
					},
					resultado: domain.ComprobacionAfirmativa,
				},
				{
					categoriaRef: "categoria:desarrollo:sin-cobertura",
					periodo: domain.PeriodoPrevisto{
						Inicio: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
						Fin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
					},
					resultado: domain.ComprobacionNegativa,
				},
			},
		},
		verificador: &verificadorRespuestaCoberturaDesarrollo{
			presentadorAutoridadFuenteAnalisisDesarrollo: presentadorVerificador,
			autoridadRef:   autoridadVerificadorCoberturaDesarrollo,
			generacion:     material.generacion,
			claveRespuesta: respuestaVerificador,
			reloj:          reloj,
		},
		publicador: &publicadorCatalogoCoberturaDesarrollo{
			presentadorAutoridadFuenteAnalisisDesarrollo: presentadorPublicador,
			autoridadRef: autoridadPublicadorCoberturaDesarrollo,
			reloj:        reloj,
			gobierno:     gobierno,
		},
		autenticador: autenticador,
		referencias:  seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptografico(),
	}
	var unaVez sync.Once
	dependencias.cerrar = func() {
		unaVez.Do(func() {
			for _, presentador := range presentadores {
				borrarBytes(presentador.privada)
			}
			fuente := dependencias.fuente.(*fuenteComprobacionCoberturaDesarrollo)
			verificador := dependencias.verificador.(*verificadorRespuestaCoberturaDesarrollo)
			borrarBytes(fuente.claveRespuesta[:])
			borrarBytes(fuente.claveRecibo[:])
			borrarBytes(verificador.claveRespuesta[:])
		})
	}
	completo = true
	return dependencias, nil
}

func derivarMaterialFuentesCoberturaDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
) (materialFuentesCoberturaDesarrollo, error) {
	var material materialFuentesCoberturaDesarrollo
	etiquetas := []string{
		"raiz-institucional",
		"fuente-operativa",
		"verificador-operativo",
		"publicador-operativo",
		"hmac-respuesta",
		"hmac-recibo",
	}
	destinos := []*[sha256.Size]byte{
		&material.raiz,
		&material.fuente,
		&material.verificador,
		&material.publicador,
		&material.respuesta,
		&material.recibo,
	}
	for indice, etiqueta := range etiquetas {
		resultados, err := derivador.calcularHMAC(
			[]byte("vec.ct.desarrollo.cobertura."+etiqueta+".v1"),
			[]byte("vec.ct.desarrollo.cobertura."+etiqueta+".confirmacion.v1"),
		)
		if err != nil || len(resultados) == 0 ||
			(material.generacion != 0 && material.generacion != resultados[0].generacion) {
			borrarResultadosHMACIdempotenciaDesarrollo(resultados)
			material.borrar()
			return materialFuentesCoberturaDesarrollo{},
				errFuentesCoberturaDesarrolloNoDisponibles
		}
		material.generacion = resultados[0].generacion
		*destinos[indice] = resultados[0].localizador
		borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	}
	return material, nil
}

func (relojContratacionTemporalDesarrollo) AhoraGobiernoOperacionCobertura(
	ctx context.Context,
) (time.Time, error) {
	if contextoInterfazNulo(ctx) {
		return time.Time{}, errFuentesCoberturaDesarrolloNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	return time.Now().UTC().Truncate(time.Microsecond), nil
}

var (
	_ ports.FuenteComprobacionCobertura         = (*fuenteComprobacionCoberturaDesarrollo)(nil)
	_ ports.VerificadorRespuestaCobertura       = (*verificadorRespuestaCoberturaDesarrollo)(nil)
	_ ports.PublicadorCatalogoCobertura         = (*publicadorCatalogoCoberturaDesarrollo)(nil)
	_ cobertura.RelojGobiernoOperacionCobertura = relojContratacionTemporalDesarrollo{}
)
