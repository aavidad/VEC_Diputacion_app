package bootstrap

import (
	"context"
	"crypto/sha256"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

var seudonimoAlmacenValido = regexp.MustCompile(`^hmac-sha256:[a-z0-9_]{1,64}:[0-9a-f]{64}$`)

func TestSeudonimizadorAlmacenUsaClavePropiaYEsDeterminista(t *testing.T) {
	var maestra [sha256.Size]byte
	copy(maestra[:], strings.Repeat("m", sha256.Size))
	s := nuevoSeudonimizadorAlmacenDesarrollo(maestra)
	if !s.valido() || s.clave == maestra || s.clave == derivarClaveDesarrollo(maestra, "vec.kms.desarrollo.envoltura.v1") {
		t.Fatal("la clave de seudónimos debe ser propia y distinta de la maestra y de otras derivadas")
	}
	e := &emisorConcesionAlmacenDocumentosDesarrollo{seudonimizador: s}
	a := docports.AutorizacionV3{PrincipalID: "per_1", CorrelacionRef: "correlacion_1", RecursoRef: "ref:doc"}
	uno, err := e.SeudonimosLecturaOriginal(context.Background(), a)
	if err != nil || !seudonimoAlmacenValido.MatchString(uno.SujetoHMAC) || !seudonimoAlmacenValido.MatchString(uno.SolicitudHMAC) {
		t.Fatalf("seudónimos: %+v %v", uno, err)
	}
	dos, _ := e.SeudonimosLecturaOriginal(context.Background(), a)
	if uno != dos {
		t.Fatal("los seudónimos deben ser deterministas")
	}
	otro := a
	otro.RecursoRef = "ref:otro"
	if tres, _ := e.SeudonimosLecturaOriginal(context.Background(), otro); tres.SujetoHMAC != uno.SujetoHMAC || tres.SolicitudHMAC == uno.SolicitudHMAC {
		t.Fatal("el sujeto se conserva y la solicitud cambia con el documento")
	}
	if strings.Contains(uno.SujetoHMAC, "per_1") {
		t.Fatal("el seudónimo no debe contener la referencia de la persona")
	}
	for nombre, e := range map[string]*emisorConcesionAlmacenDocumentosDesarrollo{
		"sin seudonimizador": {}, "clave vacía": {seudonimizador: &seudonimizadorAlmacenDesarrollo{}},
	} {
		if _, err := e.SeudonimosLecturaOriginal(context.Background(), a); err == nil {
			t.Errorf("%s: se aceptó", nombre)
		}
	}
}

type autorizadorAlmacenPrueba struct {
	solicitud core.SolicitudAutorizacionLigadaV3
	resultado core.ResultadoContextoActorRegistradoV2
	concesion pruebas.ConcesionV3Prueba
	err       error
	llamadas  int
}

func (a *autorizadorAlmacenPrueba) ExigirSolicitudLigadaV3(_ context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (
	core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	a.llamadas++
	a.solicitud, a.resultado = s, r
	return a.concesion.Decision, a.concesion.Confirmacion, a.err
}

func contextoPeticionDocumentosPrueba(t *testing.T, a *autoridadDocumentosDesarrollo) (context.Context, core.VinculoAutenticacionActorV2) {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(a.reloj.Ahora(),
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return context.WithValue(context.Background(), claveContextoDocumentos{}, contextoDocumentos{
		autoridad: a, ruta: "/api/vec/documentos/originales/descargas",
		seguridad: contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado},
	}), vinculo
}

type relojLecturaAlmacenPrueba struct{ t time.Time }

func (r relojLecturaAlmacenPrueba) Ahora() time.Time { return r.t }

func TestEmisorConcesionAlmacenUsaElVinculoDeLaMismaPeticion(t *testing.T) {
	incidencias := &incidenciasDocumentosPrueba{}
	autoridad := &autoridadDocumentosDesarrollo{reloj: relojLecturaAlmacenPrueba{time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)}, incidencias: incidencias}
	ctx, vinculo := contextoPeticionDocumentosPrueba(t, autoridad)
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autorizador := &autorizadorAlmacenPrueba{}
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_documentos", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)}
	e := &emisorConcesionAlmacenDocumentosDesarrollo{autoridad: autoridad, autorizador: autorizador, motivo: motivo,
		seudonimizador: nuevoSeudonimizadorAlmacenDesarrollo([sha256.Size]byte{1})}
	recurso := core.RecursoAutorizable{Referencia: "ref:" + strings.Repeat("1", 64), ModuloID: "documentos",
		Tipo: "documento_original", Ambitos: map[string]string{"organizacion_ref": docports.OrganizacionRefV3},
		Atributos: map[string]string{"documento_version": "3"}}
	pedida := docautorizacion.SolicitudConcesionAlmacenV3{PrincipalID: datos.PrincipalID, PerfilActivoRef: datos.PerfilActivoRef,
		CorrelacionRef: "correlacion:descarga", Accion: vecports.AccionNegocioLeerOriginalDocumentoGenerado,
		Finalidad: "descargar_documento_original", Recurso: recurso}
	if _, err := e.EmitirConcesionAlmacenV3(ctx, pedida); err != nil {
		t.Fatalf("emisión: %v", err)
	}
	s, err := autorizador.solicitud.Datos()
	if err != nil || s.Accion != pedida.Accion || s.Finalidad != pedida.Finalidad || s.Recurso.Referencia != recurso.Referencia ||
		!s.VinculoAutenticacionActor.CoincideExactamenteCon(vinculo) || s.ReferenciaMotivo != motivo {
		t.Fatalf("solicitud al PDP inesperada: %+v %v", s, err)
	}
	peticion := ctx.Value(claveContextoDocumentos{}).(contextoDocumentos)
	if autorizador.resultado.HuellaSHA256 != peticion.seguridad.Resultado.HuellaSHA256 ||
		autorizador.resultado.RegistroContextoRef != peticion.seguridad.Resultado.RegistroContextoRef {
		t.Fatal("el PDP debe evaluar con el resultado de contexto de la misma petición")
	}

	// Sin contexto de la petición, de otra autoridad o de otra persona o
	// perfil, ni siquiera se consulta al PDP.
	casos := map[string]func() (context.Context, docautorizacion.SolicitudConcesionAlmacenV3){
		"sin contexto": func() (context.Context, docautorizacion.SolicitudConcesionAlmacenV3) {
			return context.Background(), pedida
		},
		"otra autoridad": func() (context.Context, docautorizacion.SolicitudConcesionAlmacenV3) {
			otro, _ := contextoPeticionDocumentosPrueba(t, &autoridadDocumentosDesarrollo{reloj: autoridad.reloj})
			return otro, pedida
		},
		"otra persona": func() (context.Context, docautorizacion.SolicitudConcesionAlmacenV3) {
			p := pedida
			p.PrincipalID = "per_otra"
			return ctx, p
		},
		"vínculo caducado": func() (context.Context, docautorizacion.SolicitudConcesionAlmacenV3) {
			tarde := &autoridadDocumentosDesarrollo{reloj: relojLecturaAlmacenPrueba{autoridad.reloj.Ahora().Add(time.Hour)}}
			c, _ := contextoPeticionDocumentosPrueba(t, autoridad)
			valor := c.Value(claveContextoDocumentos{}).(contextoDocumentos)
			valor.autoridad = tarde
			e.autoridad = tarde
			return context.WithValue(context.Background(), claveContextoDocumentos{}, valor), pedida
		},
		"contexto cancelado": func() (context.Context, docautorizacion.SolicitudConcesionAlmacenV3) {
			c, cancelar := context.WithCancel(ctx)
			cancelar()
			return c, pedida
		},
		"otro perfil": func() (context.Context, docautorizacion.SolicitudConcesionAlmacenV3) {
			p := pedida
			p.PerfilActivoRef = "prf_otro"
			return ctx, p
		},
	}
	antes := autorizador.llamadas
	for nombre, caso := range casos {
		c, p := caso()
		if _, err := e.EmitirConcesionAlmacenV3(c, p); !errors.Is(err, errLecturaAlmacenDocumentosDenegada) {
			t.Errorf("%s: %v", nombre, err)
		}
		e.autoridad = autoridad
	}
	if autorizador.llamadas != antes {
		t.Fatal("se consultó al PDP sin el vínculo de la misma petición")
	}
	autorizador.err = core.ErrAutorizacionDenegada
	if _, err := e.EmitirConcesionAlmacenV3(ctx, pedida); !errors.Is(err, errLecturaAlmacenDocumentosDenegada) || len(incidencias.emitidas) != 0 {
		t.Fatalf("una denegación del PDP deniega sin incidencia: %v %d", err, len(incidencias.emitidas))
	}
	autorizador.err = errors.New("pdp caído")
	if _, err := e.EmitirConcesionAlmacenV3(ctx, pedida); !errors.Is(err, errLecturaAlmacenDocumentosDenegada) ||
		len(incidencias.emitidas) != 1 || incidencias.emitidas[0].Codigo != core.IncidenciaGobiernoV3NoDisponible {
		t.Fatalf("un fallo del PDP deniega y declara la incidencia: %v %+v", err, incidencias.emitidas)
	}
}

func TestSinSeudonimizadorNoSeComponeLaLectura(t *testing.T) {
	autoridad := &autoridadDocumentosDesarrollo{reloj: relojLecturaAlmacenPrueba{time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)}}
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_documentos", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)}
	if f, err := nuevaFabricaLecturaDocumentosDesarrollo(autoridad, &autorizadorAlmacenPrueba{}, motivo, nil); f != nil || err != nil {
		t.Fatalf("sin seudonimizador la descarga no se publica: %v %v", f, err)
	}
	s := nuevoSeudonimizadorAlmacenDesarrollo([sha256.Size]byte{1})
	if _, err := nuevaFabricaLecturaDocumentosDesarrollo(autoridad, nil, motivo, s); err == nil {
		t.Fatal("sin autorizador no debe componerse")
	}
	if _, err := nuevaFabricaLecturaDocumentosDesarrollo(autoridad, &autorizadorAlmacenPrueba{}, core.ReferenciaEntradaCatalogo{}, s); err == nil {
		t.Fatal("sin motivo no debe componerse")
	}
	if f, err := nuevaFabricaLecturaDocumentosDesarrollo(autoridad, &autorizadorAlmacenPrueba{}, motivo, s); f == nil || err != nil {
		t.Fatalf("con todo compuesto debe haber fábrica: %v", err)
	}
}

func TestSeudonimizadorAlmacenSeBorraAlCerrar(t *testing.T) {
	s := nuevoSeudonimizadorAlmacenDesarrollo([sha256.Size]byte{7})
	s.borrar()
	if s.valido() {
		t.Fatal("la clave derivada debe quedar a cero al cerrar")
	}
}
