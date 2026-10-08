package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	protectororiginal "vec-diputacion-granada/internal/modules/bolsa/adapters/protectorstagingdesarrollo"
	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

const categoriaCargaPrueba = "categoria:rpt:auxiliar_administrativo"

// autorizadorCargaPrueba reutiliza la decisión del doble de borradores y
// exporta el material con la audiencia propia de la carga.
type autorizadorCargaPrueba struct {
	base                autorizadorBorradorPrueba
	audiencia           string
	solicitudes         []dominiovec.DatosSolicitudAutorizacionLigadaV3
	materialVistaPrevia bool
}

func (a *autorizadorCargaPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	decision, confirmacion, _, err := a.base.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		return decision, confirmacion, nil, err
	}
	datos := datosSolicitudBorradorPrueba(a.base.t, solicitud)
	a.solicitudes = append(a.solicitudes, datos)
	datosMaterial := datos
	if a.materialVistaPrevia {
		datosMaterial.Recurso.Atributos = map[string]string{"fase": "vista_previa"}
	}
	return decision, confirmacion, exportadorBorradorPrueba{material: materialCargaPrueba(a.base.t, decision, resultado, datosMaterial, a.base.instante, a.audiencia)}, nil
}

func materialCargaPrueba(t *testing.T, decision dominiovec.DecisionAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2, datos dominiovec.DatosSolicitudAutorizacionLigadaV3, instante time.Time, audiencia string) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	dh, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	mh, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	rh, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:borrador:prueba", dh, mh, resultado.RegistroContextoRef, resultado.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh, audiencia, instante, instante.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	dc, err := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	mc, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		t.Fatal(err)
	}
	material, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'b'}, puertosvec.TamanoMinimoCapacidadCanonicaV3), resumen, dc, mc, resultado.RepresentacionCanonica, resultado.Contexto.Instantanea.PersonaVersion, resultado.Contexto.Instantanea.PerfilVersion, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return material
}

type originalCargaPrueba struct{ llamadas int }

func (c *originalCargaPrueba) Preparar(_ context.Context, actaRef, _ string, formato string, contenido []byte) (puertosbolsa.OriginalProtegidoCargaConvoca, error) {
	c.llamadas++
	return puertosbolsa.OriginalProtegidoCargaConvoca{
		Referencia: "original:convoca:" + actaRef[len("acta:importacion-convoca:"):],
		Formato:    formato, BytesOriginales: len(contenido), ContenidoCifrado: []byte("solo prueba"),
	}, nil
}

type lectorContadoCarga struct {
	llamadas int
}

type preparadorErrorCarga struct{ err error }

func (p preparadorErrorCarga) PrepararLote(context.Context, importacionapp.SolicitudImportacion) (importacion.LoteValidado, error) {
	return importacion.LoteValidado{}, p.err
}

func (l *lectorContadoCarga) Decodificar(ctx context.Context, r io.ReadSeeker) (importacion.HojaStaging, error) {
	l.llamadas++
	return xlsconvoca.NuevoLector().Decodificar(ctx, r)
}

type constituidorCargaPrueba struct {
	solicitudes     []constitucion.Solicitud
	material        puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	contextoRecurso []byte
	lote            importacion.LoteValidado
	original        puertosbolsa.OriginalProtegidoCargaConvoca
	reutilizada     bool
	err             error
}

func (c *constituidorCargaPrueba) Constituir(_ context.Context, lote importacion.LoteValidado, s constitucion.Solicitud, original puertosbolsa.OriginalProtegidoCargaConvoca, contextoRecurso []byte, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (puertosbolsa.ReciboCargaConvoca, error) {
	c.solicitudes, c.material, c.lote, c.original, c.contextoRecurso = append(c.solicitudes, s), m, lote, original, append([]byte(nil), contextoRecurso...)
	if c.err != nil {
		return puertosbolsa.ReciboCargaConvoca{}, c.err
	}
	var r puertosbolsa.ReciboCargaConvoca
	r.ActaRef, r.BolsaRef, r.DecisionRef, r.AuditoriaRef = importacionapp.ReferenciaActa(s.HuellaFicheroSHA256, s.CategoriaRef), "bolsa:auxiliar_administrativo:2026-10-05", "decision:borrador:prueba", "aud_v3_0123456789abcdef0123456789abcdef"
	r.ActaReutilizada = c.reutilizada
	return r, nil
}

type escenarioCargaConvoca struct {
	servicio     *ServicioCargaConvoca
	solicitud    puertosbolsa.SolicitudConfirmarCargaConvoca
	autorizador  *autorizadorCargaPrueba
	original     *originalCargaPrueba
	lector       *lectorContadoCarga
	constituidor *constituidorCargaPrueba
}

func nuevoEscenarioCargaConvoca(t *testing.T) *escenarioCargaConvoca {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(instanteBorradorLlamamientoPrueba, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	previsualizador, err := NuevoPrevisualizadorCargaConvoca(xlsconvoca.NuevoLector())
	if err != nil {
		t.Fatal(err)
	}
	e := &escenarioCargaConvoca{
		autorizador:  &autorizadorCargaPrueba{base: autorizadorBorradorPrueba{t: t, instante: instanteBorradorLlamamientoPrueba}, audiencia: puertosbolsa.AudienciaConfirmarCargaConvoca},
		original:     &originalCargaPrueba{},
		lector:       &lectorContadoCarga{},
		constituidor: &constituidorCargaPrueba{},
	}
	preparador, err := importacionapp.NuevoPreparador(e.lector, func() time.Time { return instanteBorradorLlamamientoPrueba })
	if err != nil {
		t.Fatal(err)
	}
	contexto := &contextualizadorBorradorPrueba{contexto: puertosbolsa.ContextoBorradorLlamamientoResuelto{UnidadRef: "unidad:seleccion", AmbitoRef: "ambito:bolsa"}}
	e.servicio, err = NuevoServicioCargaConvoca(previsualizador, contexto, e.autorizador, e.original, preparador, e.constituidor, func() time.Time { return instanteBorradorLlamamientoPrueba })
	if err != nil {
		t.Fatal(err)
	}
	e.solicitud = puertosbolsa.SolicitudConfirmarCargaConvoca{Vinculo: vinculo, ResultadoContexto: resultado, Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba(),
		CategoriaRef: categoriaCargaPrueba, NombreFichero: "carga_convoca_ejemplo.xlsx", Contenido: ejemploCargaConvoca(t)}
	return e
}

func TestConfirmarCargaConvocaExigeAceptarLasFilasConErrores(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, false); !errors.Is(err, ErrCargaConvocaConErrores) {
		t.Fatalf("cargó con filas con errores sin aceptarlo: %v", err)
	}
	if len(e.autorizador.solicitudes) != 1 || e.lector.llamadas != 1 || e.original.llamadas != 0 || len(e.constituidor.solicitudes) != 0 {
		t.Fatal("el rechazo de filas debe seguir autorización y parse único, sin efecto")
	}
}

func TestConfirmarCargaConvocaLigaDecisionAlActaEImporta(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	resultado, err := e.servicio.Confirmar(context.Background(), e.solicitud, true)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(e.solicitud.Contenido)
	huella := hex.EncodeToString(suma[:])
	acta := importacionapp.ReferenciaActa(huella, categoriaCargaPrueba)
	if resultado.HuellaSHA256 != huella || resultado.FilasCargadas != 11 || resultado.FilasExcluidas != 1 || resultado.ActaReutilizada || resultado.Recibo.ActaRef != acta {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if len(e.autorizador.solicitudes) != 1 {
		t.Fatalf("decisiones = %d", len(e.autorizador.solicitudes))
	}
	d := e.autorizador.solicitudes[0]
	if d.Accion != puertosbolsa.AccionConfirmarCargaConvoca || d.Finalidad != puertosbolsa.FinalidadConfirmarCargaConvoca || d.Recurso.Referencia != acta ||
		d.Recurso.Tipo != puertosbolsa.TipoRecursoCargaConvoca || d.Recurso.ModuloID != puertosbolsa.ModuloCargaConvoca ||
		d.Recurso.Ambitos["unidad_ref"] != "unidad:seleccion" || d.Recurso.Ambitos["ambito_ref"] != "ambito:bolsa" {
		t.Fatalf("decisión no ligada a la carga: %+v", d)
	}
	if e.original.llamadas != 1 || e.lector.llamadas != 1 || e.constituidor.lote.Acta.ActorRef != actorActaCargaConvoca("per_0123456789abcdefghijkl") ||
		e.constituidor.lote.Acta.BolsaRef != e.solicitud.BolsaRef || e.constituidor.lote.Acta.NombreFichero != "carga_convoca_ejemplo.xlsx" {
		t.Fatalf("preparación inesperada: %+v", e.constituidor.lote.Acta)
	}
	preimagen := `{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{}}`
	huellaContexto := sha256.Sum256(e.constituidor.contextoRecurso)
	if string(e.constituidor.contextoRecurso) != preimagen || e.constituidor.material.ResumenCapacidad().EfectoHuellaSHA256() != hex.EncodeToString(huellaContexto[:]) {
		t.Fatalf("contexto de confirmación divergente: %s", e.constituidor.contextoRecurso)
	}
	c := e.constituidor.solicitudes
	if len(c) != 1 || c[0].ActorRef != "per_0123456789abcdefghijkl" || c[0].HuellaFicheroSHA256 != huella || c[0].CategoriaRef != categoriaCargaPrueba ||
		e.constituidor.material.ValidarEstructura() != nil {
		t.Fatalf("constitución inesperada: %+v", c)
	}
}

func TestConfirmarCargaConvocaRechazaMaterialDeVistaPrevia(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.autorizador.materialVistaPrevia = true
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, puertosbolsa.ErrCargaConvocaNoDisponible) ||
		e.lector.llamadas != 0 || e.original.llamadas != 0 || len(e.constituidor.solicitudes) != 0 {
		t.Fatalf("material de vista previa alcanzó la confirmación: %v", err)
	}
}

func TestConfirmarCargaConvocaReutilizaActaExistente(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.constituidor.reutilizada = true
	resultado, err := e.servicio.Confirmar(context.Background(), e.solicitud, true)
	if err != nil || !resultado.ActaReutilizada || e.lector.llamadas != 1 || e.original.llamadas != 1 || len(e.constituidor.solicitudes) != 1 {
		t.Fatalf("replay debe ser una sola operación atómica: %+v err=%v", resultado, err)
	}
}

func TestConfirmarCargaConvocaDenegadaNoEscribe(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.autorizador.base.err = puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("denegación perdida: %v", err)
	}
	if e.original.llamadas != 0 || e.lector.llamadas != 0 || len(e.constituidor.solicitudes) != 0 {
		t.Fatal("escribió tras una denegación")
	}
	e = nuevoEscenarioCargaConvoca(t)
	e.autorizador.base.err = errors.New("PDP caído: dato personal sintético")
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, puertosbolsa.ErrCargaConvocaNoDisponible) {
		t.Fatalf("una indisponibilidad se convirtió en otra cosa: %v", err)
	} else {
		var causa CausaInternaCargaConvoca
		if !errors.As(err, &causa) || causa.Etapa != "autorizacion" || causa.Codigo != "dependencia_no_disponible" || strings.Contains(err.Error(), "dato personal") {
			t.Fatalf("causa interna insegura: %v", err)
		}
	}
}

func TestConfirmarCargaConvocaLimitaUnMiBAntesDeAutorizarYDecodificar(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.solicitud.Contenido = make([]byte, MaximoBytesCargaConvoca+1)
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, ErrFicheroCargaConvocaExcesivo) {
		t.Fatalf("fichero excesivo admitido: %v", err)
	}
	if len(e.autorizador.solicitudes) != 0 || e.lector.llamadas != 0 || e.original.llamadas != 0 {
		t.Fatal("fichero excesivo alcanzó el PDP, el lector o el cifrador")
	}
}

func TestConfirmarCargaConvocaMinimizaErrorDelPreparador(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.servicio.preparador = preparadorErrorCarga{err: errors.New("fila con nombre personal sintético")}
	_, err := e.servicio.Confirmar(context.Background(), e.solicitud, true)
	var causa CausaInternaCargaConvoca
	if !errors.Is(err, ErrFicheroCargaConvocaInvalido) || !errors.As(err, &causa) || causa.Etapa != "preparar_lote" || strings.Contains(err.Error(), "nombre personal") || e.original.llamadas != 0 {
		t.Fatalf("error del preparador inseguro: %v", err)
	}
}

func TestConfirmarCargaConvocaNoPersisteXLSXRenombrado(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	p, err := protectororiginal.NuevoProtectorOriginal([32]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	e.servicio.original = p
	e.solicitud.NombreFichero = "carga_convoca_ejemplo.xls"
	_, err = e.servicio.Confirmar(context.Background(), e.solicitud, true)
	var causa CausaInternaCargaConvoca
	if !errors.Is(err, puertosbolsa.ErrCargaConvocaNoDisponible) || !errors.As(err, &causa) || causa.Etapa != "cifrar_original" || len(e.constituidor.solicitudes) != 0 {
		t.Fatalf("formato falso alcanzó la transacción: %v", err)
	}
}

func TestConfirmarCargaConvocaRechazaMaterialDeOtraAudiencia(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.autorizador.audiencia = puertosbolsa.AudienciaEmitirLlamamiento
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, puertosbolsa.ErrCargaConvocaNoDisponible) || len(e.constituidor.solicitudes) != 0 {
		t.Fatalf("material de otra audiencia aceptado: %v", err)
	}
}

func TestConfirmarCargaConvocaPropagaDenegacionDelConsumo(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.constituidor.err = dominiovec.ErrAutorizacionDenegada
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("denegación del consumo perdida: %v", err)
	}
	e.constituidor.err = puertosbolsa.ErrConstitucionBolsaEnConflicto
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, puertosbolsa.ErrConstitucionBolsaEnConflicto) {
		t.Fatalf("conflicto perdido: %v", err)
	}
}

func TestConfirmarCargaConvocaRechazaSolicitudIncompleta(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	for _, mutar := range []func(*puertosbolsa.SolicitudConfirmarCargaConvoca){
		func(s *puertosbolsa.SolicitudConfirmarCargaConvoca) { s.CategoriaRef = "" },
		func(s *puertosbolsa.SolicitudConfirmarCargaConvoca) { s.Contenido = nil },
		func(s *puertosbolsa.SolicitudConfirmarCargaConvoca) {
			s.MotivoAutorizacion = dominiovec.ReferenciaEntradaCatalogo{}
		},
	} {
		s := e.solicitud
		mutar(&s)
		if _, err := e.servicio.Confirmar(context.Background(), s, true); !errors.Is(err, puertosbolsa.ErrCargaConvocaInvalida) {
			t.Fatalf("solicitud incompleta admitida: %v", err)
		}
	}
	if len(e.autorizador.solicitudes) != 0 {
		t.Fatal("pidió decisión con una solicitud incompleta")
	}
}
