package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registradorIntentosCorreosPrueba struct {
	ordenes                   []vecports.OrdenIntentoAuditoria
	errPreflight, errAppend   error
	fallarPrimero, acuseAjeno bool
	antesAppend               func()
}

func (r *registradorIntentosCorreosPrueba) PreflightIntentoAuditoria(context.Context) error {
	return r.errPreflight
}
func (r *registradorIntentosCorreosPrueba) AppendIntentoAuditoria(ctx context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	if r.antesAppend != nil {
		r.antesAppend()
	}
	r.ordenes = append(r.ordenes, o)
	if ctx.Err() != nil {
		return vecports.AcuseIntentoAuditoria{}, ctx.Err()
	}
	if r.errAppend != nil {
		return vecports.AcuseIntentoAuditoria{}, r.errAppend
	}
	if r.fallarPrimero && len(r.ordenes) == 1 {
		return vecports.AcuseIntentoAuditoria{}, errors.New("commit ambiguo")
	}
	datos, err := o.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	correlacion := datos.Datos.CorrelacionRef
	if r.acuseAjeno {
		correlacion = "correlacion_" + strings.Repeat("0", 32)
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_correos_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64),
		CorrelacionRef: correlacion, RegistradaEn: time.Now().UTC()}, nil
}

func contextoIntentosCorreosPrueba(t *testing.T, destino *registradorIntentosCorreosPrueba, instante ...time.Time) (*autoridadPreferenciasUsuariosDesarrollo, context.Context) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if len(instante) != 0 {
		ahora = instante[0]
	}
	f := nuevoEscenarioMaterialRutasDietasPrueba(t, dietasports.AccionConsultarCatalogoRutasDietas, ahora)
	s, err := f.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	registro, err := nuevoRegistroIntentosConsultaCorreos(context.Background(), destino, "vec_usuarios_prueba", s.ReferenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	a := &autoridadPreferenciasUsuariosDesarrollo{ruta: usuarioshttp.RutaMisCorreos, superficie: core.SuperficieAutenticacionInternaCorporativaV1,
		reloj: relojRutasDietas{}, intentosConsultaCorreos: registro}
	a.proveedorCorreos = &proveedorCorreosUsuarios{autoridad: a, motivoConsulta: s.ReferenciaMotivo}
	ctx := context.WithValue(context.Background(), claveContextoPreferenciasUsuarios{}, contextoPreferenciasUsuarios{autoridad: a, vinculo: s.VinculoAutenticacionActor, resultado: f.resultado})
	ctx, err = vecports.ConCorrelacionIncidenciasPeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return a, ctx
}

func TestIntentosConsultaCorreosConservaIdentidadMotivoYCorrelacion(t *testing.T) {
	for _, estado := range []int{401, 403, 409, 422, 429, 503} {
		t.Run(http.StatusText(estado), func(t *testing.T) {
			destino := &registradorIntentosCorreosPrueba{}
			a, ctx := contextoIntentosCorreosPrueba(t, destino)
			original := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
			ctx, err := a.capturarIntentoConsultaCorreos(ctx)
			if err != nil || a.PrepararIntentoConsultaCorreos(ctx) != nil || len(destino.ordenes) != 0 {
				t.Fatal("captura inválida o intento registrado antes de terminar la consulta", err)
			}
			if err := a.AuditarIntentoConsultaCorreos(ctx, estado); err != nil || len(destino.ordenes) != 1 {
				t.Fatal("intento no confirmado", err)
			}
			datos, err := destino.ordenes[0].Datos()
			correlacion, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
			ref, _ := correlacion.ValorCanonico()
			resultado := core.ResultadoIntentoAuditoriaError
			if estado == 401 || estado == 403 {
				resultado = core.ResultadoIntentoAuditoriaDenegado
			}
			if err != nil || datos.Datos.Resultado != resultado || datos.Datos.Motivo != a.proveedorCorreos.motivoConsulta ||
				datos.Datos.CorrelacionRef != ref || datos.Datos.RecursoRef != original.resultado.Contexto.PersonaRef ||
				datos.Datos.Accion != usuariosports.AccionConsultarCorreos || datos.Datos.FinalidadRef != usuariosports.FinalidadCorreosPropios ||
				datos.ResultadoContexto.HuellaSHA256 != original.resultado.HuellaSHA256 || datos.Vinculo.ValidarPara(original.resultado) != nil {
				t.Fatal("material de auditoría divergente", err)
			}
		})
	}
}

func TestIntentosConsultaCorreosNoAtribuyeSinCapturaNiASuperficieAjena(t *testing.T) {
	destino := &registradorIntentosCorreosPrueba{}
	a, ctx := contextoIntentosCorreosPrueba(t, destino)
	if a.PrepararIntentoConsultaCorreos(ctx) == nil || a.AuditarIntentoConsultaCorreos(ctx, 403) == nil ||
		a.AuditarIntentoConsultaCorreos(context.Background(), 401) == nil {
		t.Fatal("fabricó identidad o intento no capturado")
	}
	c := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	sinCorrelacion := context.WithValue(context.Background(), claveContextoPreferenciasUsuarios{}, c)
	if _, err := a.capturarIntentoConsultaCorreos(sinCorrelacion); err == nil {
		t.Fatal("fabricó correlación alternativa")
	}
	ctxCapturado, err := a.capturarIntentoConsultaCorreos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctxReemplazado, _ := vecports.ConCorrelacionIncidenciasPeticion(ctxCapturado)
	if a.AuditarIntentoConsultaCorreos(ctxReemplazado, 403) == nil {
		t.Fatal("aceptó otra correlación tras captura")
	}
	a.superficie = core.SuperficieAutenticacionExternaPersonalV1
	if _, err := a.capturarIntentoConsultaCorreos(ctx); err == nil || a.AuditarIntentoConsultaCorreos(ctxCapturado, 503) == nil {
		t.Fatal("registró contexto exterior")
	}
	if len(destino.ordenes) != 0 {
		t.Fatal("intento ajeno alcanzó registro común")
	}
}

func TestIntentosConsultaCorreosReintentaMismaOrdenTrasCancelar(t *testing.T) {
	destino := &registradorIntentosCorreosPrueba{fallarPrimero: true}
	a, ctx := contextoIntentosCorreosPrueba(t, destino)
	ctx, err := a.capturarIntentoConsultaCorreos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := a.AuditarIntentoConsultaCorreos(ctx, 503); err != nil || len(destino.ordenes) != 2 {
		t.Fatal("cancelación perdió registro o reintento", err)
	}
	primera, _ := destino.ordenes[0].Datos()
	segunda, _ := destino.ordenes[1].Datos()
	if primera.IntentoRef != segunda.IntentoRef || primera.Datos != segunda.Datos {
		t.Fatal("reintento cambió la orden")
	}
	if err := a.AuditarIntentoConsultaCorreos(ctx, 503); err != nil || len(destino.ordenes) != 2 {
		t.Fatal("acuse confirmado se volvió a registrar", err)
	}
	if a.AuditarIntentoConsultaCorreos(ctx, 403) == nil || a.AuditarIntentoConsultaCorreos(ctx, 200) == nil {
		t.Fatal("intento aceptó material cambiado o resultado permitido")
	}
}

func TestIntentosConsultaCorreosRechazaAcuseAjenoYRegistroCaido(t *testing.T) {
	for _, destino := range []*registradorIntentosCorreosPrueba{{acuseAjeno: true}, {errAppend: errors.New("registro caído")}} {
		a, ctx := contextoIntentosCorreosPrueba(t, destino)
		ctx, err := a.capturarIntentoConsultaCorreos(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if a.AuditarIntentoConsultaCorreos(ctx, 403) == nil || len(destino.ordenes) != 2 {
			t.Fatal("fallo o acuse ajeno pasó por confirmado")
		}
	}
}

func TestIntentosConsultaCorreosPreflightYMotivoObligatorios(t *testing.T) {
	var nulo *registradorIntentosCorreosPrueba
	for _, d := range []registradorIntentosConsultaCorreos{nil, nulo, &registradorIntentosCorreosPrueba{errPreflight: errors.New("sin provisión")}} {
		if _, err := nuevoRegistroIntentosConsultaCorreos(context.Background(), d, "vec_usuarios_prueba", core.ReferenciaEntradaCatalogo{}); err == nil {
			t.Fatal("registrador sin provisión/motivo aceptado")
		}
	}
}

func TestIntentosConsultaCorreosFronteraTempranaNoInventaActor(t *testing.T) {
	destino := &registradorIntentosCorreosPrueba{}
	a, _ := contextoIntentosCorreosPrueba(t, destino)
	legado := &registradorDenegacionPreferenciasPrueba{}
	a.registrador, a.base = legado, &autoridadRutasDietasDesarrollo{}
	a.manejador = http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("anónimo llegó al handler") })
	w := httptest.NewRecorder()
	a.proteger(a.manejador).ServeHTTP(w, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisCorreos, nil))
	if w.Code != 401 || len(legado.ordenes) != 1 || legado.ordenes[0].ActorRef != "" || len(destino.ordenes) != 0 {
		t.Fatal("401 temprano perdió frontera o fabricó auditoría nominal")
	}
}

type emisorConsultaCorreosDenegadaPrueba struct {
	solicitud core.SolicitudAutorizacionLigadaV3
	llamadas  int
}

func (e *emisorConsultaCorreosDenegadaPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s core.SolicitudAutorizacionLigadaV3, _ core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.solicitud, e.llamadas = s, e.llamadas+1
	return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, core.ErrAutorizacionDenegada
}

func TestIntentosConsultaCorreosComparteCorrelacionConPDP(t *testing.T) {
	destino := &registradorIntentosCorreosPrueba{}
	a, ctx := contextoIntentosCorreosPrueba(t, destino)
	ctx, err := a.capturarIntentoConsultaCorreos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	emisor := &emisorConsultaCorreosDenegadaPrueba{}
	a.proveedorCorreos.emisores = map[string]emisorPreferenciasUsuarios{usuariosports.AccionConsultarCorreos: emisor}
	m := usuariosports.MaterialCorreos{Superficie: a.superficie, PersonaRef: c.resultado.Contexto.PersonaRef, PerfilRef: c.resultado.Contexto.PerfilActivoRef,
		Accion: usuariosports.AccionConsultarCorreos, FinalidadRef: usuariosports.FinalidadCorreosPropios}
	if _, err := a.proveedorCorreos.ProveerMaterialCorreos(ctx, c.vinculo, m); !errors.Is(err, usuariosports.ErrCorreosProhibido) || emisor.llamadas != 1 || len(destino.ordenes) != 0 {
		t.Fatal("PDP no terminó antes del registro del intento", err)
	}
	if err := a.AuditarIntentoConsultaCorreos(ctx, 403); err != nil {
		t.Fatal(err)
	}
	solicitud, err := emisor.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	refPDP, _ := solicitud.Correlacion.ValorCanonico()
	datos, _ := destino.ordenes[0].Datos()
	if refPDP != datos.Datos.CorrelacionRef || solicitud.ReferenciaMotivo != datos.Datos.Motivo || solicitud.Recurso.Referencia != datos.Datos.RecursoRef {
		t.Fatal("PDP e intento tienen otro motivo, recurso o correlación")
	}
}

func TestIntentosConsultaCorreosSnapshotHistoricoNoRenuevaLectura(t *testing.T) {
	destino := &registradorIntentosCorreosPrueba{}
	a, ctx := contextoIntentosCorreosPrueba(t, destino, time.Now().UTC().Truncate(time.Microsecond).Add(-24*time.Hour))
	ctx, err := a.capturarIntentoConsultaCorreos(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.ResolverOrdenCorreos(ctx); !errors.Is(err, usuariosports.ErrCorreosNoAutenticado) {
		t.Fatal("captura renovó la lectura caducada", err)
	}
	if err := a.AuditarIntentoConsultaCorreos(ctx, 401); err != nil || len(destino.ordenes) != 1 {
		t.Fatal("sello histórico perdió su atribución", err)
	}
	c := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	i, _ := a.intentoConsultaCorreos(ctx)
	c.resultado.RepresentacionCanonica[0] ^= 1
	if i.resultado.Validar() != nil {
		t.Fatal("snapshot comparte los bytes del contexto original")
	}
}

func TestIntentosConsultaCorreosCierraPoolPropioTrasFalloDeMontaje(t *testing.T) {
	for _, caso := range []string{"abrir", "preflight", "proceso"} {
		t.Run(caso, func(t *testing.T) {
			d := &registradorIntentosCorreosPrueba{}
			a, _ := contextoIntentosCorreosPrueba(t, d)
			cerrados := 0
			abrir := func() (vecports.RegistradorIntentosAuditoria, string, func(), error) {
				proceso := "vec_usuarios_prueba"
				var err error
				switch caso {
				case "abrir":
					err = errors.New("pool incompleto")
				case "preflight":
					d.errPreflight = errors.New("sin provisión")
				case "proceso":
					proceso = ""
				}
				return d, proceso, func() { cerrados++ }, err
			}
			if r, cerrar, err := abrirRegistroIntentosConsultaCorreos(context.Background(), abrir, a.proveedorCorreos.motivoConsulta); err == nil || r != nil || cerrar != nil || cerrados != 1 {
				t.Fatal("montaje fallido dejó el pool nominal abierto", err)
			}
		})
	}
	d := &registradorIntentosCorreosPrueba{}
	a, _ := contextoIntentosCorreosPrueba(t, d)
	cerrados := 0
	r, cerrar, err := abrirRegistroIntentosConsultaCorreos(context.Background(), func() (vecports.RegistradorIntentosAuditoria, string, func(), error) {
		return d, "vec_usuarios_prueba", func() { cerrados++ }, nil
	}, a.proveedorCorreos.motivoConsulta)
	if err != nil || r == nil || cerrar == nil || cerrados != 0 {
		t.Fatal("pool nominal no entregado a su propietario", err)
	}
	composicion := &composicionPreferenciasUsuarios{interna: &autoridadPreferenciasUsuariosDesarrollo{cerrar: cerrar}}
	composicion.cerrar()
	composicion.cerrar()
	if cerrados != 1 {
		t.Fatal("cierre raíz perdió o repitió el cierre nominal")
	}
}

func TestIntentosConsultaCorreosCaducaEntreCapturaYDispatcher(t *testing.T) {
	for _, fallaAppend := range []bool{false, true} {
		t.Run(map[bool]string{false: "acuse", true: "registro caído"}[fallaAppend], func(t *testing.T) {
			destino := &registradorIntentosCorreosPrueba{}
			if fallaAppend {
				destino.errAppend = errors.New("registro caído")
			}
			a, ctx := contextoIntentosCorreosPrueba(t, destino)
			c := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
			v, err := c.vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			ahora := time.Now().UTC().Truncate(time.Microsecond)
			hasta := ahora.Add(500 * time.Millisecond)
			autenticacion := core.AutenticacionRevalidadaV1{
				AutenticacionRef: v.AutenticacionRef, AutenticacionHuellaSHA256: v.AutenticacionHuellaSHA256,
				AsercionRef: v.AsercionRef, SesionRef: v.SesionRef,
				ControlSesionRef: v.ControlSesionRef, ControlSesionRevision: v.ControlSesionRevision, ControlSesionHuellaSHA256: v.ControlSesionHuellaSHA256,
				CuentaRef: v.CuentaRef, CuentaOrdinariaRef: v.CuentaOrdinariaRef, CuentaPrivilegiada: v.CuentaPrivilegiada,
				Superficie: v.Superficie, MetodoObservado: v.MetodoObservado, GarantiaObservada: v.GarantiaObservada,
				PoliticaGarantiaRef: v.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: v.PoliticaGarantiaHuellaSHA256,
				AutenticacionVerificadaEn: v.AutenticacionVerificadaEn, SesionEmitidaEn: v.SesionEmitidaEn,
				SesionRevalidadaEn: v.SesionRevalidadaEn, SesionValidaHasta: hasta,
			}
			c.vinculo, c.resultado, err = core.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidadorMaterialRutasDietasPrueba{autenticacion},
				core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef},
				resolutorMaterialRutasDietasPrueba{c.resultado}, core.SolicitudContextoActor{
					Cuenta:          core.CuentaAutenticadaContextoActor{CuentaRef: v.CuentaRef, Metodo: v.MetodoObservado, Garantia: v.GarantiaObservada},
					PerfilActivoRef: c.resultado.Contexto.PerfilActivoRef}, &relojMaterialRutasDietasPrueba{ahora: ahora})
			if err != nil {
				t.Fatal(err)
			}
			ctx = context.WithValue(ctx, claveContextoPreferenciasUsuarios{}, c)
			ctx, err = a.capturarIntentoConsultaCorreos(ctx)
			if err != nil || !c.vinculo.VigenteEn(a.reloj.Ahora(), c.resultado) {
				t.Fatal("captura sin vínculo vigente", err)
			}
			ctx, err = vechttp.ConActorVerificadoAuditoriaPreferenciasUsuarios(ctx, c.resultado.Contexto)
			if err != nil {
				t.Fatal(err)
			}
			legado := &registradorDenegacionPreferenciasPrueba{}
			composicion := &composicionPreferenciasUsuarios{interna: &autoridadPreferenciasUsuariosDesarrollo{correos: a}}
			autoridad := autoridadExactasConUsuariosPreferencias{usuarios: composicion}
			llamadas := 0
			despachador, err := vechttp.NewHandlerSoloRutasExactas([]vechttp.RutaExacta{{Ruta: usuarioshttp.RutaMisCorreos,
				Manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { llamadas++ })}}, autoridad,
				registradorFronterasConUsuariosPreferencias{interna: legado})
			if err != nil {
				t.Fatal(err)
			}
			<-time.After(time.Until(hasta) + time.Millisecond)
			if c.vinculo.VigenteEn(a.reloj.Ahora(), c.resultado) {
				t.Fatal("el vínculo no caducó")
			}
			w := httptest.NewRecorder()
			destino.antesAppend = func() {
				if w.Body.Len() != 0 || llamadas != 0 {
					t.Fatal("append posterior a respuesta o acceso pese a caducidad")
				}
			}
			despachador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisCorreos, nil).WithContext(ctx))
			estado, appendEsperados := 403, 1
			if fallaAppend {
				estado, appendEsperados = 503, 2
			}
			if w.Code != estado || len(destino.ordenes) != appendEsperados || len(legado.ordenes) != 0 || llamadas != 0 {
				t.Fatalf("caducidad perdió auditoría o abrió consulta: HTTP%d append%d legado%d handler%d", w.Code, len(destino.ordenes), len(legado.ordenes), llamadas)
			}
			datos, _ := destino.ordenes[0].Datos()
			correlacion, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
			ref, _ := correlacion.ValorCanonico()
			if datos.Datos.Resultado != core.ResultadoIntentoAuditoriaDenegado || datos.Datos.CorrelacionRef != ref || datos.ResultadoContexto.HuellaSHA256 != c.resultado.HuellaSHA256 {
				t.Fatal("dispatcher sustituyó correlación o identidad capturadas")
			}
		})
	}
}
