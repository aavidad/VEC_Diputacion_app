package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instanteGobiernoConvocatoriaPrueba = time.Date(2026, time.July, 16, 8, 0, 0, 0, time.UTC)

func TestAltaExigeIntencionAutorizacionIdempotenciaYReciboLigados(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	selladoMotivo := atestacionMotivoConvocatoriaPrueba(
		t, AccionCrearBorradorConvocatoria, version.Referencia(), 'a',
	)
	material, err := MaterialAltaBorradorConvocatoria(version, nil, nil, selladoMotivo)
	if err != nil {
		t.Fatal(err)
	}
	autorizacion := autorizacionMutacionConvocatoriaPrueba(t, material, version)
	testimonio := testimonioIdempotenciaConvocatoriaPrueba(t, material, autorizacion)
	preparacion := PreparacionTransaccionGobiernoConvocatoria{
		Material: material, Idempotencia: testimonio, Autorizacion: autorizacion,
		SelladoMotivo: selladoMotivo,
		SolicitadaEn:  instanteGobiernoConvocatoriaPrueba,
	}
	confirmacion := ConfirmacionAltaBorradorConvocatoria{Version: version, Transaccion: preparacion}
	if err := confirmacion.Validar(); err != nil {
		t.Fatalf("confirmacion valida rechazada: %v", err)
	}
	if _, err := json.Marshal(confirmacion); !errors.Is(err, ErrSerializacionGobiernoConvocatoriaProhibida) {
		t.Fatalf("orden interna serializable: %v", err)
	}
	for _, formato := range []string{"%v", "%+v", "%#v"} {
		if salida := fmt.Sprintf(formato, confirmacion); strings.Contains(salida, version.ID) {
			t.Fatalf("orden interna filtrada con %s: %s", formato, salida)
		}
	}

	datosAutorizacion, _ := autorizacion.Datos()
	datosIdempotencia, _ := testimonio.Datos()
	datosSellado, _ := selladoMotivo.DatosParaConsumo()
	huellaIntencion, _ := material.HuellaSHA256()
	recibo := ReciboGobiernoConvocatoria{
		TransaccionRef: "transaccion:convocatoria:001", Accion: material.Accion,
		EstadoPrincipal:                    material.EstadoPrincipalNuevo,
		PrincipalRef:                       datosAutorizacion.Decision.PrincipalID,
		AutorizacionRef:                    datosAutorizacion.Decision.DecisionRef,
		HuellaAutorizacionSHA256:           datosAutorizacion.HuellaDecisionSHA256,
		AtestacionAutorizacionRef:          "atestacion:pdp:001",
		HuellaAtestacionAutorizacionSHA256: huellaConvocatoriaPrueba('3'),
		ConsumoAutorizacionRef:             "consumo:autorizacion:001",
		IndiceIdempotenciaHMACSHA256:       datosIdempotencia.IndiceOperacionHMACSHA256,
		AtestacionIdempotenciaRef:          datosIdempotencia.AtestacionRef,
		HuellaAtestacionIdempotenciaSHA256: datosIdempotencia.HuellaAtestacionSHA256,
		HuellaIntencionSHA256:              huellaIntencion,
		AuditoriaRef:                       "auditoria:convocatoria:001", HuellaAuditoriaSHA256: huellaConvocatoriaPrueba('1'),
		EventoOutboxRef: "outbox:convocatoria:001", HuellaEventoOutboxSHA256: huellaConvocatoriaPrueba('2'),
		ConsumoMotivo: func() *ReciboConsumoVerificacionConvocatoria {
			consumo := reciboConsumoMotivo(datosSellado)
			return &consumo
		}(),
		ConfirmadaEn: instanteGobiernoConvocatoriaPrueba.Add(time.Second),
	}
	if err := recibo.ValidarPara(preparacion, version); err != nil {
		t.Fatalf("recibo valido rechazado: %v", err)
	}
	preparacionNoCanonica := preparacion
	preparacionNoCanonica.SolicitadaEn = preparacion.SolicitadaEn.Add(time.Nanosecond)
	if err := recibo.ValidarPara(preparacionNoCanonica, version); !errors.Is(err, ErrReciboGobiernoConvocatoriaInvalido) {
		t.Fatalf("recibo acepto una preparacion temporal no canonica: %v", err)
	}
	sinAtestacionPDP := recibo
	sinAtestacionPDP.AtestacionAutorizacionRef = ""
	if err := sinAtestacionPDP.ValidarPara(preparacion, version); !errors.Is(err, ErrReciboGobiernoConvocatoriaInvalido) {
		t.Fatalf("recibo sin atestacion PDP aceptado: %v", err)
	}
	pruebasReutilizadas := recibo
	pruebasReutilizadas.ConsumoAutorizacionRef = recibo.AtestacionAutorizacionRef
	if err := pruebasReutilizadas.ValidarPara(preparacion, version); !errors.Is(err, ErrReciboGobiernoConvocatoriaInvalido) {
		t.Fatalf("recibo reutilizo identidad entre atestacion y consumo: %v", err)
	}
	if _, err := json.Marshal(recibo); !errors.Is(err, ErrSerializacionGobiernoConvocatoriaProhibida) {
		t.Fatalf("recibo interno serializable: %v", err)
	}
	for _, formato := range []string{"%v", "%+v", "%#v"} {
		if salida := fmt.Sprintf(formato, recibo); strings.Contains(salida, recibo.PrincipalRef) {
			t.Fatalf("recibo filtro principal con %s: %s", formato, salida)
		}
	}
	recibo.EventoOutboxRef = recibo.AuditoriaRef
	if err := recibo.ValidarPara(preparacion, version); !errors.Is(err, ErrReciboGobiernoConvocatoriaInvalido) {
		t.Fatalf("auditoria y outbox compartieron identidad: %v", err)
	}
}

func TestClaveIdempotenciaExigeEntropiaYNoSeFiltra(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	material, _ := MaterialAltaBorradorConvocatoria(version, nil, nil, atestacionMotivoConvocatoriaPrueba(
		t, AccionCrearBorradorConvocatoria, version.Referencia(), 'a',
	))
	solicitud := SolicitudProtegerIdempotenciaConvocatoria{
		ClaveIdempotencia: strings.Repeat("k", 31), PrincipalRef: "per_0123456789abcdefghijkl",
		Material: material, SolicitadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	if err := solicitud.Validar(); !errors.Is(err, ErrIdempotenciaConvocatoriaInvalida) {
		t.Fatalf("clave corta aceptada: %v", err)
	}
	solicitud.ClaveIdempotencia = strings.Repeat("k", 32)
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("clave de 32 bytes rechazada: %v", err)
	}
	if _, err := json.Marshal(solicitud); !errors.Is(err, ErrSerializacionIdempotenciaConvocatoria) {
		t.Fatalf("solicitud con clave serializable: %v", err)
	}
	for _, formato := range []string{"%v", "%+v", "%#v"} {
		if salida := fmt.Sprintf(formato, solicitud); strings.Contains(salida, solicitud.ClaveIdempotencia) {
			t.Fatalf("clave filtrada con %s: %s", formato, salida)
		}
	}
}

func TestMotivoSoloCruzaSelladorVersionadoYNoSeFiltra(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	motivo := "Motivo administrativo reservado para el expediente."
	solicitud := SolicitudSellarMotivoGobiernoConvocatoria{
		DominioCriptografico: DominioCriptograficoMotivoGobiernoConvocatoriaV1,
		Accion:               AccionCrearBorradorConvocatoria, ConvocatoriaRef: version.Referencia(),
		PrincipalRef: "per_0123456789abcdefghijkl", CorrelacionRef: "correlacion:convocatoria:motivo:001",
		Motivo: motivo, SolicitadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud de sellado valida rechazada: %v", err)
	}
	if _, err := json.Marshal(solicitud); !errors.Is(err, ErrSerializacionMotivoGobiernoConvocatoria) {
		t.Fatalf("motivo serializable: %v", err)
	}
	for _, formato := range []string{"%v", "%+v", "%#v"} {
		if salida := fmt.Sprintf(formato, solicitud); strings.Contains(salida, motivo) {
			t.Fatalf("motivo filtrado con %s: %s", formato, salida)
		}
	}
	hmac := hmacMotivoConvocatoriaPrueba('a')
	if err := hmac.Validar(); err != nil || hmac.GeneracionClave != 3 || hmac.ClaveHMACRef != "motivo-gobierno-v3" {
		t.Fatalf("HMAC tipado sin clave versionada: %+v / %v", hmac, err)
	}
	huellaSolicitud, err := solicitud.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	reintento := solicitud
	reintento.SolicitadaEn = solicitud.SolicitadaEn.Add(time.Minute)
	huellaReintento, err := reintento.HuellaSHA256()
	if err != nil || huellaReintento != huellaSolicitud {
		t.Fatalf("un reintento temporal altero la intencion semantica: %v", err)
	}
	atestacion, err := NuevaAtestacionSelladoMotivoConvocatoria(
		solicitud,
		DatosAtestacionSelladoMotivoConvocatoria{
			HMAC: hmac, Accion: solicitud.Accion, ConvocatoriaRef: solicitud.ConvocatoriaRef,
			PrincipalRef: solicitud.PrincipalRef, CorrelacionRef: solicitud.CorrelacionRef,
			HuellaSolicitudSHA256: huellaSolicitud, SelladorRef: "sellador:motivo:v3",
			AtestacionRef: "atestacion:motivo:001", HuellaAtestacionSHA256: huellaConvocatoriaPrueba('b'),
			TokenConsumoRef:       "consumo:motivo:001",
			AtestacionEmitidaEn:   instanteGobiernoConvocatoriaPrueba.Add(time.Minute),
			AtestacionValidaHasta: instanteGobiernoConvocatoriaPrueba.Add(4 * time.Minute),
		},
	)
	if err != nil {
		t.Fatalf("atestacion de sellado valida rechazada: %v", err)
	}
	solicitudDistinta := solicitud
	solicitudDistinta.Motivo += " Otro contenido."
	if _, err := NuevaAtestacionSelladoMotivoConvocatoria(
		solicitudDistinta, func() DatosAtestacionSelladoMotivoConvocatoria {
			datos, _ := atestacion.DatosParaConsumo()
			return datos
		}(),
	); !errors.Is(err, ErrSelladoMotivoGobiernoConvocatoriaInvalido) {
		t.Fatalf("atestacion se reutilizo para otro motivo: %v", err)
	}
	tipoFabrica := reflect.TypeOf(MaterialAltaBorradorConvocatoria)
	if tipoFabrica.In(3) != reflect.TypeOf(AtestacionSelladoMotivoConvocatoria{}) {
		t.Fatal("la mutacion acepta un HMAC fabricable sin atestacion del sellador")
	}
}

func TestAutorizacionNoSeReutilizaParaOtraIntencionAccionORecurso(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	selladoMotivo := atestacionMotivoConvocatoriaPrueba(
		t, AccionCrearBorradorConvocatoria, version.Referencia(), 'a',
	)
	material, _ := MaterialAltaBorradorConvocatoria(version, nil, nil, selladoMotivo)
	autorizacion := autorizacionMutacionConvocatoriaPrueba(t, material, version)
	testimonio := testimonioIdempotenciaConvocatoriaPrueba(t, material, autorizacion)

	otroMaterial := material
	otroMaterial.HuellaMotivoHMACSHA256 = hmacConvocatoriaPrueba("motivo", 'b')
	if otroMaterial.Validar() != nil {
		t.Fatal("la segunda intencion de prueba no es valida")
	}
	if err := testimonio.ValidarPara(otroMaterial, principalAutorizacionConvocatoria(t, autorizacion)); !errors.Is(err, ErrIdempotenciaConvocatoriaInvalida) {
		t.Fatalf("testimonio acepto otra intencion: %v", err)
	}
	preparacion := PreparacionTransaccionGobiernoConvocatoria{
		Material: otroMaterial, Idempotencia: testimonio, Autorizacion: autorizacion,
		SelladoMotivo: selladoMotivo,
		SolicitadaEn:  instanteGobiernoConvocatoriaPrueba,
	}
	if err := preparacion.ValidarPara(version); !errors.Is(err, ErrConfirmacionGobiernoConvocatoriaInvalida) {
		t.Fatalf("autorizacion se reutilizo con otra preimagen: %v", err)
	}

	selector := SelectorVersionConvocatoriaExacta{ID: version.ID, Secuencia: version.Secuencia}
	consulta := SolicitudConsultaVersionConvocatoriaAutorizada{
		Selector: selector, Autorizacion: autorizacion, ConsultadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	if err := consulta.Validar(); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("autorizacion de alta habilito consulta: %v", err)
	}
}

func TestConsultaInternaRechazaSuperficieExternaYCamposNoExactos(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	selector := SelectorVersionConvocatoriaExacta{ID: version.ID, Secuencia: version.Secuencia}
	recurso, err := RecursoAutorizableConsultaVersionConvocatoria(selector)
	if err != nil {
		t.Fatal(err)
	}
	vinculoExterno := vinculoExternoPersonalConvocatoriaPrueba(t, instanteGobiernoConvocatoriaPrueba)
	autorizacionExterna := evidenciaAutorizacionConvocatoriaConVinculoPrueba(
		t, AccionConsultarVersionConvocatoria, recurso, instanteGobiernoConvocatoriaPrueba, vinculoExterno,
	)
	solicitud := SolicitudConsultaVersionConvocatoriaAutorizada{
		Selector: selector, Autorizacion: autorizacionExterna,
		ConsultadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	if err := solicitud.Validar(); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("superficie externa accedio a gobierno interno: %v", err)
	}

	vinculoInterno, err := pruebasvec.NuevoVinculoGenerico(instanteGobiernoConvocatoriaPrueba)
	if err != nil {
		t.Fatal(err)
	}
	autorizacionAmpliada := evidenciaAutorizacionConvocatoriaConVinculoYCamposPrueba(
		t, AccionConsultarVersionConvocatoria, recurso, instanteGobiernoConvocatoriaPrueba,
		vinculoInterno, []string{"instancia_flujo", "version_convocatoria"},
	)
	solicitud.Autorizacion = autorizacionAmpliada
	if err := solicitud.Validar(); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("consulta sin flujo acepto campos ampliados: %v", err)
	}
}

func TestConsultaExactaAuditaYNoPublicaBorrador(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	selector := SelectorVersionConvocatoriaExacta{ID: version.ID, Secuencia: 1}
	recurso, err := RecursoAutorizableConsultaVersionConvocatoria(selector)
	if err != nil {
		t.Fatal(err)
	}
	autorizacion := evidenciaAutorizacionConvocatoriaPrueba(
		t, AccionConsultarVersionConvocatoria, recurso, instanteGobiernoConvocatoriaPrueba,
	)
	solicitud := SolicitudConsultaVersionConvocatoriaAutorizada{
		Selector: selector, Autorizacion: autorizacion, ConsultadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	datos, _ := autorizacion.Datos()
	huellaVersion, _ := version.HuellaSHA256()
	resultado := ResultadoConsultaVersionConvocatoria{
		Version:                            version,
		HuellaVersionSHA256:                huellaVersion,
		AutorizacionRef:                    datos.Decision.DecisionRef,
		HuellaAutorizacionSHA256:           datos.HuellaDecisionSHA256,
		AtestacionAutorizacionRef:          "atestacion:pdp:consulta:001",
		HuellaAtestacionAutorizacionSHA256: huellaConvocatoriaPrueba('3'),
		ConsumoAutorizacionRef:             "consumo:autorizacion:consulta:001",
		AuditoriaRef:                       "auditoria:consulta:001", HuellaAuditoriaSHA256: huellaConvocatoriaPrueba('4'),
		ConsultadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	if err := resultado.ValidarPara(solicitud); err != nil {
		t.Fatalf("lectura interna exacta rechazada: %v", err)
	}
	pruebasReutilizadas := resultado
	pruebasReutilizadas.AuditoriaRef = resultado.ConsumoAutorizacionRef
	if err := pruebasReutilizadas.ValidarPara(solicitud); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("consulta reutilizo identidad entre consumo y auditoria: %v", err)
	}
	otra := solicitud
	otra.Selector.Secuencia = 2
	if err := otra.Validar(); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("decision para #1 leyo #2: %v", err)
	}
	instanciaFalsa := instanciaFlujoConvocatoriaPuertosPrueba(version)
	resultado.InstanciaFlujo = &instanciaFalsa
	if err := resultado.ValidarPara(solicitud); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("un borrador acepto una instancia de flujo fabricada: %v", err)
	}
}

func TestAtestacionesRevalidablesNoAceptanReplayDeRevision(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	solicitud := SolicitudVerificarDependenciasConvocatoria{
		Version: version, VerificarEn: instanteGobiernoConvocatoriaPrueba,
	}
	huellaContenido, _ := version.HuellaContenidoSHA256()
	huellaEstado, _ := version.HuellaSHA256()
	evidencia := dominiobolsa.EvidenciaDependenciasConvocatoria{
		Referencia: "dependencias:convocatoria:001", HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('5'),
		ConvocatoriaRef: version.Referencia(), Revision: version.Revision,
		HuellaContenidoSHA256: huellaContenido, HuellaEstadoSHA256: huellaEstado,
		VerificadaEn: instanteGobiernoConvocatoriaPrueba.Add(-time.Minute),
	}
	datos := DatosAtestacionDependenciasConvocatoria{
		Evidencia: evidencia, RevisionVersion: version.Revision, HuellaEstadoVersionSHA256: huellaEstado,
		VerificadorRef: "verificador:dependencias:v1", AtestacionRef: "atestacion:dependencias:001",
		HuellaAtestacionSHA256: huellaConvocatoriaPrueba('6'), TokenConsumoRef: "consumo:dependencias:001",
		AtestacionEmitidaEn:   instanteGobiernoConvocatoriaPrueba.Add(-30 * time.Second),
		AtestacionValidaHasta: instanteGobiernoConvocatoriaPrueba.Add(2 * time.Minute),
	}
	atestacion, err := NuevaAtestacionDependenciasConvocatoria(solicitud, datos)
	if err != nil || atestacion.ValidarPara(solicitud, instanteGobiernoConvocatoriaPrueba) != nil {
		t.Fatalf("atestacion valida rechazada: %v", err)
	}
	if _, err := json.Marshal(atestacion); !errors.Is(err, ErrSerializacionVerificacionConvocatoria) {
		t.Fatalf("atestacion serializable: %v", err)
	}
	for _, formato := range []string{"%v", "%+v", "%#v"} {
		if salida := fmt.Sprintf(formato, atestacion); strings.Contains(salida, evidencia.Referencia) {
			t.Fatalf("atestacion filtrada con %s: %s", formato, salida)
		}
	}

	actualizada := actualizarYRestaurarContenidoConvocatoriaPrueba(t, version)
	peticionReplay := SolicitudVerificarDependenciasConvocatoria{
		Version: actualizada, VerificarEn: actualizada.UltimaModificacionEn.Add(time.Minute),
	}
	if _, err := NuevaAtestacionDependenciasConvocatoria(peticionReplay, datos); !errors.Is(err, ErrComprobacionDependenciasConvocatoriaInvalida) {
		t.Fatalf("atestacion de revision 1 reutilizada en revision %d: %v", actualizada.Revision, err)
	}

	tipo := reflect.TypeOf(ConfirmacionPublicacionConvocatoria{})
	campo, existe := tipo.FieldByName("Dependencias")
	if !existe || campo.Type != reflect.TypeOf(AtestacionDependenciasConvocatoria{}) {
		t.Fatal("la publicacion acepta evidencia estructural sin atestacion revalidable")
	}
}

func TestInterfacesDelContratoQuedanSeparadasDeHTTP(t *testing.T) {
	var _ ConsultaGobiernoConvocatorias = consultaGobiernoConvocatoriasContrato{}
	var _ ProtectorIdempotenciaConvocatorias = protectorIdempotenciaConvocatoriasContrato{}
	var _ SelladorMotivoGobiernoConvocatoria = selladorMotivoGobiernoConvocatoriaContrato{}
	var _ VerificadorDependenciasConvocatoria = verificadorDependenciasConvocatoriaContrato{}
	var _ VerificadorAprobacionConvocatoria = verificadorAprobacionConvocatoriaContrato{}
	var _ RepositorioGobiernoConvocatorias = repositorioGobiernoConvocatoriasContrato{}
}

type selladorMotivoGobiernoConvocatoriaContrato struct{}

func (selladorMotivoGobiernoConvocatoriaContrato) SellarMotivo(
	context.Context,
	SolicitudSellarMotivoGobiernoConvocatoria,
) (AtestacionSelladoMotivoConvocatoria, error) {
	return AtestacionSelladoMotivoConvocatoria{}, nil
}

type consultaGobiernoConvocatoriasContrato struct{}

func (consultaGobiernoConvocatoriasContrato) ObtenerVersionExacta(
	context.Context,
	SolicitudConsultaVersionConvocatoriaAutorizada,
) (ResultadoConsultaVersionConvocatoria, error) {
	return ResultadoConsultaVersionConvocatoria{}, nil
}

type protectorIdempotenciaConvocatoriasContrato struct{}

func (protectorIdempotenciaConvocatoriasContrato) Proteger(
	context.Context,
	SolicitudProtegerIdempotenciaConvocatoria,
) (TestimonioIdempotenciaConvocatoria, error) {
	return TestimonioIdempotenciaConvocatoria{}, nil
}

type verificadorDependenciasConvocatoriaContrato struct{}

func (verificadorDependenciasConvocatoriaContrato) VerificarDependencias(
	context.Context,
	SolicitudVerificarDependenciasConvocatoria,
) (AtestacionDependenciasConvocatoria, error) {
	return AtestacionDependenciasConvocatoria{}, nil
}

type verificadorAprobacionConvocatoriaContrato struct{}

func (verificadorAprobacionConvocatoriaContrato) ComprobarAprobacion(
	context.Context,
	SolicitudComprobarAprobacionConvocatoria,
) (AtestacionAprobacionConvocatoria, error) {
	return AtestacionAprobacionConvocatoria{}, nil
}

type repositorioGobiernoConvocatoriasContrato struct{}

func (repositorioGobiernoConvocatoriasContrato) ConfirmarAltaBorrador(
	context.Context, ConfirmacionAltaBorradorConvocatoria,
) (ReciboGobiernoConvocatoria, error) {
	return ReciboGobiernoConvocatoria{}, nil
}
func (repositorioGobiernoConvocatoriasContrato) ConfirmarActualizacionBorrador(
	context.Context, ConfirmacionActualizacionBorradorConvocatoria,
) (ReciboGobiernoConvocatoria, error) {
	return ReciboGobiernoConvocatoria{}, nil
}
func (repositorioGobiernoConvocatoriasContrato) ConfirmarPublicacion(
	context.Context, ConfirmacionPublicacionConvocatoria,
) (ReciboGobiernoConvocatoria, error) {
	return ReciboGobiernoConvocatoria{}, nil
}
func (repositorioGobiernoConvocatoriasContrato) ConfirmarRetirada(
	context.Context, ConfirmacionRetiradaConvocatoria,
) (ReciboGobiernoConvocatoria, error) {
	return ReciboGobiernoConvocatoria{}, nil
}
