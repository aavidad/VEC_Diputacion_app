package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var (
	instanteDerivacionPublicaPrueba = time.Date(2026, 7, 18, 9, 30, 0, 0, time.UTC)
	errFinDerivacionPublicaPrueba   = errors.New("fin deliberado tras consulta de identidades")
	errDependenciaNoInvocable       = errors.New("dependencia posterior no invocable")
)

type relojDerivacionPublicaPrueba struct{}

func (relojDerivacionPublicaPrueba) Ahora() time.Time { return instanteDerivacionPublicaPrueba }

type catalogoDerivacionPublicaPrueba struct {
	plantilla gobiernoconvocatorias.PlantillaBorradorResuelta
}

func (c catalogoDerivacionPublicaPrueba) ResolverPlantillaBorrador(
	_ context.Context,
	selector gobiernoconvocatorias.SelectorPlantillaBorrador,
	_ time.Time,
) (gobiernoconvocatorias.PlantillaBorradorResuelta, error) {
	if selector.ID != c.plantilla.Referencia.ID || selector.Version != c.plantilla.Referencia.Version ||
		selector.HuellaContenidoSHA256 != c.plantilla.Referencia.HuellaContenidoSHA256 {
		return gobiernoconvocatorias.PlantillaBorradorResuelta{}, errDependenciaNoInvocable
	}
	return c.plantilla, nil
}

func (catalogoDerivacionPublicaPrueba) PrepararAltaBorrador(
	context.Context,
	gobiernoconvocatorias.PlantillaBorradorResuelta,
	string,
	string,
	time.Time,
) (gobiernoconvocatorias.PreparacionAltaBorrador, error) {
	return gobiernoconvocatorias.PreparacionAltaBorrador{}, errDependenciaNoInvocable
}

func (catalogoDerivacionPublicaPrueba) ResolverMotivoBorrador(
	_ context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
	_ time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	if referencia.Validar() != nil {
		return dominiovec.ReferenciaEntradaCatalogo{}, errDependenciaNoInvocable
	}
	return referencia, nil
}

type dependenciasPosterioresNoInvocables struct{}

func (dependenciasPosterioresNoInvocables) ObtenerBorradorExacto(
	context.Context,
	puertosbolsa.ReferenciaEstadoVersionConvocatoria,
) (dominiobolsa.VersionConvocatoriaGobernada, error) {
	return dominiobolsa.VersionConvocatoriaGobernada{}, errDependenciaNoInvocable
}

func (dependenciasPosterioresNoInvocables) ComprometerMotivo(
	context.Context,
	puertosbolsa.SolicitudComprometerMotivoGobiernoConvocatoria,
) (puertosbolsa.HMACMotivoGobiernoConvocatoria, error) {
	return puertosbolsa.HMACMotivoGobiernoConvocatoria{}, errDependenciaNoInvocable
}

func (dependenciasPosterioresNoInvocables) EvaluarDecisionBorrador(
	context.Context,
	dominiovec.ContextoActor,
	dominiovec.VinculoAutenticacionActorV1,
	dominiovec.RecursoAutorizable,
	string,
	dominiovec.ReferenciaEntradaCatalogo,
	gobiernoconvocatorias.IntencionBorradorCanonica,
	time.Time,
) (gobiernoconvocatorias.ResultadoEvaluacionPDPBorrador, error) {
	return gobiernoconvocatorias.ResultadoEvaluacionPDPBorrador{}, errDependenciaNoInvocable
}

func (dependenciasPosterioresNoInvocables) VerificarYSellarMotivo(
	context.Context,
	gobiernoconvocatorias.SolicitudSelladoMotivoBorrador,
) (gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador, error) {
	return gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador{}, errDependenciaNoInvocable
}

func (dependenciasPosterioresNoInvocables) CifrarBorrador(
	context.Context,
	gobiernoconvocatorias.SolicitudCifradoBorrador,
) (gobiernoconvocatorias.ResultadoCifradoBorrador, error) {
	return gobiernoconvocatorias.ResultadoCifradoBorrador{}, errDependenciaNoInvocable
}

type autoridadPosteriorNoInvocable struct{ nombre string }

func (a autoridadPosteriorNoInvocable) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		a.nombre,
		"instancia-"+a.nombre,
		"credencial-"+a.nombre,
		"rol-"+a.nombre,
	)
	return identidad
}

func (autoridadPosteriorNoInvocable) VinculoVerificadorReciboBorrador() gobiernoconvocatorias.VinculoVerificadorReciboBorrador {
	identidadPersistencia := autoridadPosteriorNoInvocable{
		nombre: "verificador-derivacion-publica",
	}.IdentidadAutoridadBorrador()
	identidadCriptografica, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		"cripto-derivacion-publica", "instancia-cripto-derivacion-publica",
		"credencial-cripto-derivacion-publica", "rol-cripto-derivacion-publica",
	)
	vinculo, _ := gobiernoconvocatorias.NuevoVinculoVerificadorReciboBorrador(
		identidadPersistencia, identidadCriptografica,
	)
	return vinculo
}

func (autoridadPosteriorNoInvocable) SeleccionarPoliticaCifradoBorrador(
	context.Context,
	gobiernoconvocatorias.SolicitudSeleccionPoliticaCifradoBorrador,
) (gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador, error) {
	return gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador{}, errDependenciaNoInvocable
}

func (autoridadPosteriorNoInvocable) ResolverPerfilCifradoBorrador(
	context.Context,
	gobiernoconvocatorias.SolicitudResolucionPerfilCifradoBorrador,
) (gobiernoconvocatorias.ResolucionPerfilCifradoBorrador, error) {
	return gobiernoconvocatorias.ResolucionPerfilCifradoBorrador{}, errDependenciaNoInvocable
}

func (autoridadPosteriorNoInvocable) ConfirmarBorrador(
	context.Context,
	gobiernoconvocatorias.SolicitudConfirmacionBorrador,
) (gobiernoconvocatorias.ResultadoConfirmacionAtomica, error) {
	return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, errDependenciaNoInvocable
}

func (autoridadPosteriorNoInvocable) VerificarReciboBorrador(
	context.Context,
	gobiernoconvocatorias.ProyeccionReciboBorrador,
) error {
	return errDependenciaNoInvocable
}

type diarioCapturaDerivacionPublica struct {
	mu        sync.Mutex
	consultas []gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador
}

func (d *diarioCapturaDerivacionPublica) ConsultarIdentidades(
	_ context.Context,
	solicitud gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador,
) (gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador, error) {
	copia := solicitud
	copia.Identidades = append(
		[]gobiernoconvocatorias.ProyeccionIdentidadOperacion(nil), solicitud.Identidades...,
	)
	d.mu.Lock()
	d.consultas = append(d.consultas, copia)
	d.mu.Unlock()
	// Corte deliberado: la llegada al diario prueba que Derivar devolvio un
	// ConjuntoIdentidadesOperacion valido. PDP, cifrado y persistencia quedan
	// fuera de T20C y se prueban en sus propias verticales.
	return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, errFinDerivacionPublicaPrueba
}

func (*diarioCapturaDerivacionPublica) ReservarDecision(
	context.Context,
	gobiernoconvocatorias.SolicitudReservaDecisionBorrador,
) (gobiernoconvocatorias.ResultadoReservaDecisionBorrador, error) {
	return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, errDependenciaNoInvocable
}

func (*diarioCapturaDerivacionPublica) Reconciliar(
	context.Context,
	gobiernoconvocatorias.SolicitudReconciliacionBorrador,
) (gobiernoconvocatorias.ResultadoReconciliacionBorrador, error) {
	return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, errDependenciaNoInvocable
}

func (*diarioCapturaDerivacionPublica) ReclamarDecision(
	context.Context,
	gobiernoconvocatorias.SolicitudReclamacionDecisionBorrador,
) (gobiernoconvocatorias.ResultadoOperacionDiario, error) {
	return gobiernoconvocatorias.ResultadoOperacionDiario{}, errDependenciaNoInvocable
}

func (d *diarioCapturaDerivacionPublica) instantanea() []gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador {
	d.mu.Lock()
	defer d.mu.Unlock()
	resultado := make([]gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador, len(d.consultas))
	for indice := range d.consultas {
		resultado[indice] = d.consultas[indice]
		resultado[indice].Identidades = append(
			[]gobiernoconvocatorias.ProyeccionIdentidadOperacion(nil), d.consultas[indice].Identidades...,
		)
	}
	return resultado
}

type derivadorPublicoObservado struct {
	real                      gobiernoconvocatorias.DerivadorIdentidadOperacion
	llamadas                  atomic.Uint64
	materialesEfimerosValidos atomic.Uint64
	conjuntosDerivados        atomic.Uint64
}

func (d *derivadorPublicoObservado) Derivar(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudDerivacionIdempotencia,
) (gobiernoconvocatorias.ConjuntoIdentidadesOperacion, error) {
	d.llamadas.Add(1)
	preimagenL, preimagenF, err := solicitud.MaterialParaConectorConfiable()
	if err != nil || len(preimagenL) == 0 || len(preimagenF) == 0 {
		borrarBytes(preimagenL)
		borrarBytes(preimagenF)
		if err == nil {
			err = gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida
		}
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{}, err
	}
	borrarBytes(preimagenL)
	borrarBytes(preimagenF)
	d.materialesEfimerosValidos.Add(1)
	conjunto, err := d.real.Derivar(ctx, solicitud)
	if err == nil {
		d.conjuntosDerivados.Add(1)
	}
	return conjunto, err
}

func TestDerivarPublicoRecorreSolicitudCanonicaHastaConjuntoG2G1(t *testing.T) {
	servicio, orden, diario, observado := escenarioDerivacionPublicaPrueba(t)
	if _, err := servicio.Crear(context.Background(), orden); !errors.Is(err, errFinDerivacionPublicaPrueba) {
		t.Fatalf("la ejecucion no alcanzo el diario tras Derivar: %v", err)
	}
	if observado.llamadas.Load() != 1 {
		t.Fatalf("llamadas publicas a Derivar = %d", observado.llamadas.Load())
	}
	if observado.materialesEfimerosValidos.Load() != 1 || observado.conjuntosDerivados.Load() != 1 {
		t.Fatalf(
			"recorrido solicitud/material/conjunto incompleto: material=%d conjunto=%d",
			observado.materialesEfimerosValidos.Load(), observado.conjuntosDerivados.Load(),
		)
	}
	consultas := diario.instantanea()
	if len(consultas) != 1 {
		t.Fatalf("consultas capturadas = %d", len(consultas))
	}
	validarConsultaDerivacionPublica(t, consultas[0])
}

func TestDerivarPublicoEsSeguroBajoCargaConcurrenteReal(t *testing.T) {
	servicio, ordenBase, diario, observado := escenarioDerivacionPublicaPrueba(t)
	const total = 32
	inicio := make(chan struct{})
	errores := make(chan error, total)
	var grupo sync.WaitGroup
	grupo.Add(total)
	for indice := 0; indice < total; indice++ {
		indice := indice
		go func() {
			defer grupo.Done()
			<-inicio
			orden := ordenBase
			orden.CorrelacionRef = fmt.Sprintf("correlacion:derivacion-publica:%02d", indice)
			clave, err := gobiernoconvocatorias.NuevaClaveClienteIdempotenciaConvocatoria(
				claveIdempotenciaConcurrentePrueba(indice + 1),
			)
			if err != nil {
				errores <- err
				return
			}
			orden.ClaveCliente = clave
			_, err = servicio.Crear(context.Background(), orden)
			errores <- err
		}()
	}
	close(inicio)
	grupo.Wait()
	close(errores)
	for err := range errores {
		if !errors.Is(err, errFinDerivacionPublicaPrueba) {
			t.Fatalf("ejecucion concurrente: %v", err)
		}
	}
	if observado.llamadas.Load() != total {
		t.Fatalf("llamadas concurrentes a Derivar = %d; esperadas %d", observado.llamadas.Load(), total)
	}
	if observado.materialesEfimerosValidos.Load() != total || observado.conjuntosDerivados.Load() != total {
		t.Fatalf(
			"recorridos concurrentes incompletos: material=%d conjunto=%d",
			observado.materialesEfimerosValidos.Load(), observado.conjuntosDerivados.Load(),
		)
	}
	consultas := diario.instantanea()
	if len(consultas) != total {
		t.Fatalf("consultas concurrentes = %d; esperadas %d", len(consultas), total)
	}
	localizadoresPrimarios := make(map[string]struct{}, total)
	huellaSemantica := ""
	for _, consulta := range consultas {
		validarConsultaDerivacionPublica(t, consulta)
		primaria := consulta.Identidades[0]
		localizadoresPrimarios[primaria.Localizador.ValorHMACSHA256] = struct{}{}
		if huellaSemantica == "" {
			huellaSemantica = primaria.HuellaSolicitud.ValorHMACSHA256
		} else if primaria.HuellaSolicitud.ValorHMACSHA256 != huellaSemantica {
			t.Fatal("la misma intencion canonica produjo huellas F distintas")
		}
	}
	if len(localizadoresPrimarios) != total {
		t.Fatalf("claves cliente distintas produjeron %d localizadores L para %d operaciones", len(localizadoresPrimarios), total)
	}
}

func escenarioDerivacionPublicaPrueba(
	t *testing.T,
) (*gobiernoconvocatorias.ServicioBorradores, gobiernoconvocatorias.OrdenCrearBorrador,
	*diarioCapturaDerivacionPublica, *derivadorPublicoObservado) {
	t.Helper()
	actor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instanteDerivacionPublicaPrueba,
		"per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl",
		dominiovec.AuthMethodCertificate,
		dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, configuracion := datosDerivacionPublicaPrueba(t)
	referenciaPlantilla := dominiobolsa.ReferenciaConfiguracionConvocatoria{
		ID: "plantilla:bolsa:derivacion-publica", Version: 2,
		HuellaContenidoSHA256: strings.Repeat("8", 64),
	}
	configuracion.Plantilla = referenciaPlantilla
	plantilla := gobiernoconvocatorias.PlantillaBorradorResuelta{
		Referencia:    referenciaPlantilla,
		Configuracion: configuracion,
	}
	catalogo := catalogoDerivacionPublicaPrueba{plantilla: plantilla}
	clave, err := gobiernoconvocatorias.NuevaClaveClienteIdempotenciaConvocatoria(
		claveIdempotenciaConcurrentePrueba(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	orden := gobiernoconvocatorias.OrdenCrearBorrador{
		ClaveCliente: clave, Actor: actor, VinculoAutenticacionActor: vinculo,
		Plantilla: gobiernoconvocatorias.SelectorPlantillaBorrador{
			ID: plantilla.Referencia.ID, Version: plantilla.Referencia.Version,
			HuellaContenidoSHA256: plantilla.Referencia.HuellaContenidoSHA256,
		},
		CodigoVersionPublica: "v1", Contenido: contenido,
		ExpedienteRef: "expediente:seleccion:derivacion-publica",
		MotivoCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_rrhh", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("9", 64), EntradaClave: "crear_borrador",
		},
		CorrelacionRef: "correlacion:derivacion-publica:00",
	}
	real := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	observado := &derivadorPublicoObservado{real: real}
	diario := &diarioCapturaDerivacionPublica{}
	posteriores := dependenciasPosterioresNoInvocables{}
	procedencia, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		"desarrollo", gobiernoconvocatorias.AutoridadActoNoAutoritativa,
		referenciaProveedorIdempotenciaDesarrollo, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := gobiernoconvocatorias.NuevoServicioBorradores(
		relojDerivacionPublicaPrueba{}, catalogo, catalogo, posteriores, posteriores,
		observado, posteriores, diario, posteriores,
		autoridadPosteriorNoInvocable{nombre: "politica-derivacion-publica"},
		autoridadPosteriorNoInvocable{nombre: "perfil-derivacion-publica"},
		posteriores,
		autoridadPosteriorNoInvocable{nombre: "confirmador-derivacion-publica"},
		autoridadPosteriorNoInvocable{nombre: "verificador-derivacion-publica"},
		procedencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, orden, diario, observado
}

func validarConsultaDerivacionPublica(
	t *testing.T,
	consulta gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador,
) {
	t.Helper()
	if len(consulta.Identidades) != 2 || !consulta.SolicitadaEn.Equal(instanteDerivacionPublicaPrueba) {
		t.Fatalf("consulta no representa la ventana completa: %+v", consulta)
	}
	for indice, generacion := range []uint32{2, 1} {
		identidad := consulta.Identidades[indice]
		if identidad.Validar() != nil ||
			identidad.Localizador.VersionEsquema != versionHMACIdempotenciaBorrador ||
			identidad.HuellaSolicitud.VersionEsquema != versionHMACIdempotenciaBorrador ||
			identidad.Localizador.GeneracionClave != generacion ||
			identidad.HuellaSolicitud.GeneracionClave != generacion ||
			identidad.Localizador.ClaveRef != referenciaLocalizadorIdempotenciaDesarrollo(generacion) ||
			identidad.HuellaSolicitud.ClaveRef != referenciaHuellaIdempotenciaDesarrollo(generacion) {
			t.Fatalf("identidad %d invalida: %+v", indice, identidad)
		}
		for _, valor := range []string{
			identidad.Localizador.ValorHMACSHA256,
			identidad.HuellaSolicitud.ValorHMACSHA256,
		} {
			if bytes, err := hex.DecodeString(valor); err != nil || len(bytes) != 32 {
				t.Fatalf("HMAC no es SHA-256 hexadecimal canonico: %q", valor)
			}
		}
	}
}

func claveIdempotenciaConcurrentePrueba(numero int) string {
	material := make([]byte, 32)
	binary.BigEndian.PutUint64(material[24:], uint64(numero))
	return base64.RawURLEncoding.EncodeToString(material)
}

func datosDerivacionPublicaPrueba(
	t *testing.T,
) (dominiobolsa.ContenidoPublicableConvocatoria, dominiobolsa.ConfiguracionFijadaConvocatoria) {
	t.Helper()
	contenido := dominiobolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-administrativo-derivacion", Tipo: "bolsa_temporal",
		CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64),
		},
		Categorias: []string{"auxiliar_administrativo"}, Titulo: "Bolsa temporal de auxiliares",
		Resumen: "Convocatoria publica para bolsa temporal.", Descripcion: "Proceso sujeto a bases.",
		Plazos: []dominiobolsa.PlazoConvocatoria{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripcion",
			Descripcion: "Plazo de presentacion.", AbreEn: instanteDerivacionPublicaPrueba.Add(24 * time.Hour),
			CierraEn: instanteDerivacionPublicaPrueba.Add(30 * 24 * time.Hour),
		}},
		Requisitos: []dominiobolsa.RequisitoConvocatoria{{
			Referencia: "requisito:edad", Orden: 1, Titulo: "Edad",
			Descripcion: "Cumplir la edad exigida.", Obligatorio: true,
		}},
		Documentos: []dominiobolsa.DocumentoPublicableConvocatoria{{
			Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
			Descripcion: "Bases firmadas.", Formato: "pdf", URL: "/bolsa/documentos/bases.pdf",
		}},
	}
	referencia := func(id string, marca byte) dominiobolsa.ReferenciaConfiguracionConvocatoria {
		return dominiobolsa.ReferenciaConfiguracionConvocatoria{
			ID: id, Version: 1, HuellaContenidoSHA256: strings.Repeat(string(marca), 64),
		}
	}
	configuracion := dominiobolsa.ConfiguracionFijadaConvocatoria{
		Catalogos: referencia("catalogos:bolsa", '1'), Calendario: referencia("calendario:bolsa", '2'),
		ReglasBaremacion: referencia("baremo:bolsa", '3'),
		FlujoProceso:     referencia("convocatoria-bolsa", '4'),
		FlujoSolicitud:   referencia("solicitud-bolsa", '5'),
		Plantilla:        referencia("plantilla:bolsa:derivacion-publica", '8'),
		Documentos: []dominiobolsa.ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "documento:bases",
			DocumentoRef: "documento:logico:bases:derivacion", VersionDocumento: 1,
			RepresentacionRef:     "representacion:pdf:bases:derivacion",
			HuellaContenidoSHA256: strings.Repeat("b", 64),
			FirmaValidadaRef:      "firma:validada:bases:derivacion",
			ReciboCustodiaRef:     "custodia:bases:derivacion",
		}},
	}
	if configuracion.ValidarPara(contenido) != nil {
		t.Fatal("fixture de configuracion no valida el contenido")
	}
	return contenido, configuracion
}
