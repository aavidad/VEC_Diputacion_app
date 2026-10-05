package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

const categoriaCargaPrueba = "categoria:rpt:auxiliar_administrativo"

// autorizadorCargaPrueba reutiliza la decisión del doble de borradores y
// exporta el material con la audiencia propia de la carga.
type autorizadorCargaPrueba struct {
	base        autorizadorBorradorPrueba
	audiencia   string
	solicitudes []dominiovec.DatosSolicitudAutorizacionLigadaV3
}

func (a *autorizadorCargaPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	decision, confirmacion, _, err := a.base.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		return decision, confirmacion, nil, err
	}
	datos := datosSolicitudBorradorPrueba(a.base.t, solicitud)
	a.solicitudes = append(a.solicitudes, datos)
	return decision, confirmacion, exportadorBorradorPrueba{material: materialCargaPrueba(a.base.t, decision, resultado, datos, a.base.instante, a.audiencia)}, nil
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

type custodioCargaPrueba struct{ llamadas int }

func (c *custodioCargaPrueba) Custodiar(_ context.Context, contenido []byte) (string, error) {
	c.llamadas++
	suma := sha256.Sum256(contenido)
	return "fichero:sha256:" + hex.EncodeToString(suma[:]), nil
}

type importadorCargaPrueba struct {
	existe     bool
	importadas []importacionapp.SolicitudImportacion
}

func (i *importadorCargaPrueba) ActaImportada(context.Context, string, string) (bool, error) {
	return i.existe, nil
}

func (i *importadorCargaPrueba) Importar(_ context.Context, s importacionapp.SolicitudImportacion) (importacionapp.ResultadoImportacion, error) {
	i.importadas = append(i.importadas, s)
	suma := sha256.Sum256(s.Contenido)
	huella := hex.EncodeToString(suma[:])
	var r importacionapp.ResultadoImportacion
	r.Acta.ActaRef, r.Acta.HuellaFicheroSHA256 = importacionapp.ReferenciaActa(huella, s.CategoriaRef), huella
	return r, nil
}

type constituidorCargaPrueba struct {
	solicitudes []constitucion.Solicitud
	material    puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	err         error
}

func (c *constituidorCargaPrueba) Constituir(_ context.Context, s constitucion.Solicitud, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (puertosbolsa.ReciboCargaConvoca, error) {
	c.solicitudes, c.material = append(c.solicitudes, s), m
	if c.err != nil {
		return puertosbolsa.ReciboCargaConvoca{}, c.err
	}
	var r puertosbolsa.ReciboCargaConvoca
	r.ActaRef, r.BolsaRef, r.DecisionRef, r.AuditoriaRef = importacionapp.ReferenciaActa(s.HuellaFicheroSHA256, s.CategoriaRef), "bolsa:auxiliar_administrativo:2026-10-05", "decision:borrador:prueba", "aud_v3_0123456789abcdef0123456789abcdef"
	return r, nil
}

type escenarioCargaConvoca struct {
	servicio     *ServicioCargaConvoca
	solicitud    puertosbolsa.SolicitudConfirmarCargaConvoca
	autorizador  *autorizadorCargaPrueba
	custodio     *custodioCargaPrueba
	importador   *importadorCargaPrueba
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
		custodio:     &custodioCargaPrueba{},
		importador:   &importadorCargaPrueba{},
		constituidor: &constituidorCargaPrueba{},
	}
	contexto := &contextualizadorBorradorPrueba{contexto: puertosbolsa.ContextoBorradorLlamamientoResuelto{UnidadRef: "unidad:seleccion", AmbitoRef: "ambito:bolsa"}}
	e.servicio, err = NuevoServicioCargaConvoca(previsualizador, contexto, e.autorizador, e.custodio, e.importador, e.constituidor, func() time.Time { return instanteBorradorLlamamientoPrueba })
	if err != nil {
		t.Fatal(err)
	}
	e.solicitud = puertosbolsa.SolicitudConfirmarCargaConvoca{Vinculo: vinculo, ResultadoContexto: resultado, Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba(),
		CategoriaRef: categoriaCargaPrueba, BolsaRef: "bolsa:auxiliar_administrativo:2026-10-05", NombreFichero: "carga_convoca_ejemplo.xlsx", Contenido: ejemploCargaConvoca(t)}
	return e
}

func TestConfirmarCargaConvocaExigeAceptarLasFilasConErrores(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, false); !errors.Is(err, ErrCargaConvocaConErrores) {
		t.Fatalf("cargó con filas con errores sin aceptarlo: %v", err)
	}
	// El permiso se comprueba antes de leer el libro; sin aceptar los errores
	// no se custodia, ni se importa, ni se constituye.
	if len(e.autorizador.solicitudes) != 1 || e.custodio.llamadas != 0 || len(e.importador.importadas) != 0 || len(e.constituidor.solicitudes) != 0 {
		t.Fatal("escribió antes de aceptar las filas con errores")
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
	if e.custodio.llamadas != 1 || len(e.importador.importadas) != 1 || e.importador.importadas[0].ActorRef != actorActaCargaConvoca("per_0123456789abcdefghijkl") ||
		e.importador.importadas[0].BolsaRef != e.solicitud.BolsaRef || e.importador.importadas[0].NombreFichero != "carga_convoca_ejemplo.xlsx" {
		t.Fatalf("importación inesperada: %+v", e.importador.importadas)
	}
	c := e.constituidor.solicitudes
	if len(c) != 1 || c[0].ActorRef != "per_0123456789abcdefghijkl" || c[0].HuellaFicheroSHA256 != huella || c[0].CategoriaRef != categoriaCargaPrueba ||
		e.constituidor.material.ValidarEstructura() != nil {
		t.Fatalf("constitución inesperada: %+v", c)
	}
}

func TestConfirmarCargaConvocaReutilizaActaExistente(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.importador.existe = true
	resultado, err := e.servicio.Confirmar(context.Background(), e.solicitud, true)
	if err != nil || !resultado.ActaReutilizada || e.custodio.llamadas != 0 || len(e.importador.importadas) != 0 || len(e.constituidor.solicitudes) != 1 {
		t.Fatalf("acta existente reimportada: %+v err=%v", resultado, err)
	}
}

func TestConfirmarCargaConvocaDenegadaNoLeeElLibro(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.autorizador.base.err = puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3
	e.solicitud.Contenido = []byte("no es un libro")
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("sin permiso debe denegar antes de validar el fichero: %v", err)
	}
}

func TestConfirmarCargaConvocaDenegadaNoEscribe(t *testing.T) {
	e := nuevoEscenarioCargaConvoca(t)
	e.autorizador.base.err = puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("denegación perdida: %v", err)
	}
	if e.custodio.llamadas != 0 || len(e.importador.importadas) != 0 || len(e.constituidor.solicitudes) != 0 {
		t.Fatal("escribió tras una denegación")
	}
	e = nuevoEscenarioCargaConvoca(t)
	e.autorizador.base.err = errors.New("PDP caído")
	if _, err := e.servicio.Confirmar(context.Background(), e.solicitud, true); !errors.Is(err, puertosbolsa.ErrCargaConvocaNoDisponible) {
		t.Fatalf("una indisponibilidad se convirtió en otra cosa: %v", err)
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
		func(s *puertosbolsa.SolicitudConfirmarCargaConvoca) { s.BolsaRef = "" },
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
