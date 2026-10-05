package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// autorizadorFirmaEspiaPrueba cuenta las llamadas al PDP y las deniega: basta
// para saber si la firma V1 llegó a pedir la decisión V3.
type autorizadorFirmaEspiaPrueba struct {
	autorizadorLigadoContratacionTemporalDesarrollo
	llamadas int
	persona  string
}

var errAutorizadorFirmaEspiaPrueba = errors.New("pdp espía: denegado")

func (a *autorizadorFirmaEspiaPrueba) ExigirSolicitudLigadaV3(_ context.Context, _ dominiovec.SolicitudAutorizacionLigadaV3,
	r dominiovec.ResultadoContextoActorRegistradoV2,
) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	a.llamadas++
	a.persona = r.Contexto.PersonaRef
	return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, errAutorizadorFirmaEspiaPrueba
}

// La persona registrada de la sesión no es el identificador del certificado
// mTLS. Un perfil nominal de cargo, vigente y con su paso en el catálogo, debe
// llegar al PDP con la persona registrada; antes se comparaba con el ID del
// certificado y la firma V1 se denegaba siempre.
func TestFirmaDocumentoV1PerfilNominalLlegaAlPDPConLaPersonaRegistrada(t *testing.T) {
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	perfil, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, "cargo_prueba",
		[]string{httpinterno.RutaFirmaDocumento},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			concesiones := []dominiovec.ConcesionRol{{Accion: ports.AccionFirmarDocumento, ModuloID: ports.ModuloContratacion,
				TipoRecurso: ports.TipoRecursoFirmaDocumento, Finalidades: []string{ports.FinalidadFirmaDocumento},
				GarantiaMinima: dominiovec.AuthAssuranceHigh}}
			return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(actor, ref, ahora,
				"cargo_firma_ct_prueba", "Cargo de firma de prueba", "asignacion-cargo-firma-ct-prueba", concesiones,
				[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(perfil) != nil {
		t.Fatal("perfil nominal no compuesto", err)
	}
	perfil.contextoEsperadoRegistrado = perfil.contexto.Resultado
	perfil.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: perfil.contexto}
	registrada := perfil.contexto.Resultado.Contexto.PersonaRef
	if registrada == "" || registrada == principal.ID {
		t.Fatal("el escenario debe distinguir certificado y persona registrada")
	}
	a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	a.asignaciones = map[string]instantaneaPublicadaDesarrollo{perfil.perfilRef(): {
		instantanea:    clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(perfil.plantilla),
		actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}

	// Catálogo de ejemplo con el primer paso asignado al perfil nominal.
	bruto, err := os.ReadFile(rutaCircuitoFirmaCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	const pasoOriginal = `"perfil_ref": "perfil:ct:tecnico_rrhh"`
	if strings.Count(string(bruto), pasoOriginal) != 1 {
		t.Fatal("el catálogo de ejemplo ha cambiado")
	}
	catalogo := filepath.Join(t.TempDir(), "circuito.json")
	if err := os.WriteFile(catalogo, []byte(strings.Replace(string(bruto), pasoOriginal,
		`"perfil_ref": "`+perfil.perfilRef()+`"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionCircuitoFirmaPrueba(catalogo), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := compuestas.circuitoFirmaCT.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	m := materialFirmaDesarrolloPrueba()
	m.CatalogoRef, m.CatalogoHuella = circuito.CatalogoID+":1", circuito.HuellaCatalogo
	m.PasoRef = circuito.Documentos[0].Pasos[0].Referencia
	if m.Validar() != nil || !perfilPasoFirmaDocumentoCTDesarrolloCoincide(m, circuito, perfil.perfilRef()) {
		t.Fatal("el material no corresponde al paso del perfil nominal")
	}

	espia := &autorizadorFirmaEspiaPrueba{autorizadorLigadoContratacionTemporalDesarrollo: base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)}
	f := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s, autorizador: espia,
		postgresql: dependenciasPostgreSQLContratacionTemporalDesarrollo{
			proveedorMaterialFirmaDocumento: new(proveedorMaterialAltaContratacionTemporalDesarrollo)}},
		circuito: compuestas.circuitoFirmaCT, reloj: s.reloj}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaFirmaDocumento)
	if _, err := f.AutorizarFirmaDocumento(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("la denegación del PDP espía no se ha traducido: %v", err)
	}
	if espia.llamadas != 1 || espia.persona != registrada {
		t.Fatalf("la firma V1 no llegó al PDP con la persona registrada: llamadas=%d", espia.llamadas)
	}

	// La sesión registrada válida de otro perfil, aunque coincida con el
	// contexto esperado, se deniega en el cotejo con el perfil antes del PDP.
	otro, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, "otro_cargo_prueba",
		[]string{httpinterno.RutaConsultaFirmaDocumento},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, ref, ahora)
		})
	if err != nil || otro.contexto.Resultado.Validar() != nil {
		t.Fatal("perfil ajeno no compuesto", err)
	}
	perfil.contextoEsperadoRegistrado = otro.contexto.Resultado
	perfil.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: otro.contexto}
	ctx = contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaFirmaDocumento)
	if _, err := f.AutorizarFirmaDocumento(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || espia.llamadas != 1 {
		t.Fatalf("la sesión de otro perfil alcanzó el PDP: %v, llamadas=%d", err, espia.llamadas)
	}
}
